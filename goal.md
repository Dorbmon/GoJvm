# GoJVM: Complete Design Document for a Go-to-JVM Compiler Backend

> Document status: Implementation Baseline  
> Intended audience: automated development agents, compiler engineers, runtime engineers, and test engineers  
> Upstream baseline: Go 1.26.4  
> Minimum JVM: OpenJDK 21  
> Target class-file version: 65 (Java 21)  
> Document date: 2026-06-22

---

## 0. Conventions and Normative Language

The terms **MUST**, **MUST NOT**, **SHOULD**, and **MAY** are normative. An implementation agent MUST NOT silently change a rule marked MUST or MUST NOT without first updating this design document, the corresponding Architecture Decision Record (ADR), and the compatibility tests.

The project is provisionally named **GoJVM**. Its compiler intermediate representation is named **GoJVM-MIR (GoJVM Mid-level Intermediate Representation)**.

The project is governed by these principles:

1. Reuse as much of the official Go compiler as possible, including lexical analysis, parsing, type checking, generic instantiation, inlining, and the front-end Unified IR.
2. Branch away from the native pipeline before target-specific machine stages and construct a JVM-specific mid-level IR.
3. Do not carry register allocation, native stack-frame layout, raw address arithmetic, or Go's own garbage collector into the JVM backend.
4. Represent heap state as JVM-managed objects and use the JVM garbage collector.
5. Use one JVM virtual thread as the execution carrier for each Go goroutine, while a GoJVM runtime preserves Go semantics for scheduling, channels, `select`, `panic`/`defer`/`recover`, and synchronization.
6. Explicitly reject cgo, `unsafe`, assembly, and features that depend on a native ABI.
7. Prioritize semantic correctness, verifiability, and debuggability before performance optimization.

---

## 1. Project Goals, Non-goals, and Compatibility Levels

### 1.1 Goals

The first phase of GoJVM MUST:

- Fork the Go 1.26.4 compiler and toolchain.
- Add the target pair `GOOS=jvm`, `GOARCH=jvm`.
- Compile pure-Go programs into JARs executable by OpenJDK 21 or later.
- Support Go static typing, generics, closures, interfaces, arrays, slices, strings, maps, channels, `defer`, `panic`, `recover`, `select`, synchronization primitives, and the core reflection model.
- Carry goroutines on JVM virtual threads.
- Reuse the existing JVM garbage collector rather than implementing a Go heap, marker, sweeper, compactor, root scanner, or write barrier.
- Preserve the main development experience of `go build`, `go run`, and `go test`.
- Emit standard JVM class files accepted by the JVM bytecode verifier.
- Produce explicit compile-time, link-time, or startup-time diagnostics for unsupported features; it MUST NOT silently degrade to incorrect semantics.
- Maintain a minimally invasive fork that can be rebased onto later upstream Go releases.

### 1.2 Non-goals

The first stable release explicitly does not support:

- cgo.
- JNI, JNA, or any project-supplied native bridge.
- User code importing `unsafe`.
- Go assembly files (`.s`, `.S`).
- C, C++, Objective-C, or Fortran source files.
- Go plugins.
- Native build modes such as `-buildmode=c-archive`, `c-shared`, `shared`, `plugin`, or `pie`.
- The race detector, MemorySanitizer, or AddressSanitizer.
- Native system-call ABIs, raw file-descriptor tricks, or `mmap` pointer semantics.
- Arbitrary interoperability with Java source code. A controlled Java interoperability layer may be designed later, but it MUST NOT contaminate the core Go semantic model.
- Exact reproduction of native Go scheduling timing, stack-growth implementation, GC pause metrics, or memory statistics.
- Native machine-code generation.
- Implementing or embedding Go's own garbage collector.
- Building every official standard-library package unchanged when it depends on `unsafe`, cgo, or assembly.

### 1.3 Compatibility Tiers

Compatibility MUST be reported by tier rather than with the vague statement that the implementation “supports Go”:

- **Language compatibility:** as close as practical to the Go 1.26 language specification, excluding `unsafe` behavior.
- **Runtime compatibility:** semantic compatibility for goroutines, channels, `select`, `panic`/`defer`/`recover`, synchronization, and atomics.
- **Standard-library compatibility:** a documented allowlist and port-status matrix.
- **Toolchain compatibility:** common `go build`, `go run`, `go test`, and `go list` workflows; not every native linker flag is supported.
- **Performance compatibility:** no claim of performance equivalence with native Go.
- **Binary compatibility:** no compatibility with native Go object files, shared libraries, or ABI.

---

## 2. Fixed Technical Baseline

### 2.1 Go Upstream Version

- The initial implementation is pinned to Go 1.26.4.
- The bootstrap Go used to build the Go 1.26 source tree MUST satisfy the upstream requirement: Go 1.24.6 or a later compatible version.
- The fork MUST preserve upstream commit identity and define a read-only `upstream` remote.
- Development MUST NOT span multiple Go minor versions during the first milestone.
- Any upstream upgrade MUST first pass the Frontend Integration Contract Tests.

Recommended Git initialization:

```bash
git remote add upstream https://go.googlesource.com/go
git fetch upstream --tags
git checkout -b feature/gojvm-backend go1.26.4
```

### 2.2 JVM and OpenJDK Baseline

- Emit Java 21 class files with major version 65.
- The minimum runtime is OpenJDK 21.
- CI MUST cover OpenJDK 21 and OpenJDK 25.
- OpenJDK 26 MAY be used for forward-compatibility testing, but no JDK 26-only API may enter target artifacts.
- Java runtime-support code MUST be compiled with `javac --release 21`.
- Preview features MUST NOT be used.
- The runtime MUST NOT depend on a particular GC algorithm. G1, ZGC, Shenandoah, and other compatible JVM collectors should all work.

Java 21 is selected because virtual threads became a final feature in that release. Class-file version 65 is selected so artifacts run on the minimum supported JVM and are not accidentally upgraded when CI uses a newer JDK.

### 2.3 Build Host and Target

- The GoJVM compiler itself continues to be built by a host Go toolchain.
- The target is `GOOS=jvm GOARCH=jvm`.
- Under this target, `CGO_ENABLED` MUST be forced to behave as `0`.
- Language-level `int`, `uint`, and `uintptr` are all defined as 64-bit for `GOARCH=jvm`.
- Target endianness is not a statement about raw memory layout. Internally the target may be described as “managed/no native endian”; standard-library code that requires an explicit byte order MUST perform explicit byte operations.

### 2.4 Go 1.26 Language and Experiment Checklist

GoJVM MUST explicitly account for behaviors introduced or fixed by the pinned baseline:

- `new(expr)`: evaluate the expression in ordinary Go evaluation order, allocate an addressable zero slot, store the initial value, and return a managed pointer. If evaluation panics, the allocated result MUST NOT become observably available first.
- Self-referential generic constraints: rely on the Go 1.26 front-end type checker. MIR type IDs and generic dictionaries MUST represent recursive constraint graphs by node ID rather than recursively expanding them until stack overflow.
- `GOEXPERIMENT=simd` and `simd/archsimd` are architecture-specific experiments and MUST be rejected for the initial JVM target. Future support should introduce a portable vector MIR rather than pretending to be amd64.
- `GOEXPERIMENT=runtimesecret` MUST initially be rejected because its secure-erasure guarantees involve registers, stacks, and native heap state that cannot be guaranteed under a JVM/JIT model.
- Changes to Go 1.26's native GC are irrelevant to GoJVM and MUST NOT be ported.
- The experimental goroutine-leak profile is initially unsupported. A later implementation may use the GoJVM waiter graph, weak owners, and registry, but MUST NOT claim exact equivalence with an upstream algorithm based on Go-GC reachability.

### 2.5 Go Target Registration and Bootstrap Build

Milestone M0 MUST inspect and integrate every relevant target-registration point in the Go source tree, not merely one platform list. At minimum:

- `cmd/dist` known GOOS/GOARCH values, build ordering, and test filters.
- `internal/platform`, `internal/goos`, `internal/goarch`, and generated files.
- File suffixes, build tags, and capability tables in `go/build` and `cmd/go`.
- `runtime.GOOS` and `runtime.GOARCH` constants.
- `_jvm.go` file-selection rules.
- `jvm` MUST NOT be classified as `unix`, `windows`, `plan9`, `js`, or `wasip1`.
- Default executable suffix and `go install` behavior. The recommended default artifact name is `<package>.jar`.
- Package-archive output from `go tool compile` and the JVM payload.
- JVM dispatch in `go tool link`.
- Build-cache keys MUST include the class-file target version and the GoJVM runtime ABI version.

The minimal GoJVM runtime kernel is written in Java and built directly with JDK tools:

```bash
"$JAVA_HOME/bin/javac" --release 21 -d build/rt-classes @runtime-java-sources.txt
"$JAVA_HOME/bin/jar" --create --file build/gojvm-rt.jar -C build/rt-classes .
```

Maven or Gradle MUST NOT become a hard dependency of compiler bootstrapping. Development builds may lazily create `gojvm-rt.jar` on first use of the JVM target. Release distributions should include runtime classes generated by a reproducible process, together with source and rebuild verification. Building only the host native Go toolchain should not require a JDK; building or testing the JVM target MUST check `JAVA_HOME` and the Java version and report a precise error.

### 2.6 `GOOS=jvm` Platform Contract

`GOOS=jvm` is an independent portable platform. It does not pretend at runtime to be `linux`, `windows`, or `darwin`:

- `runtime.GOOS == "jvm"` and `runtime.GOARCH == "jvm"` are process-wide constants.
- Build tags such as `linux`, `windows`, and `unix` are false.
- The same class-file-65 JAR should run on supported Linux, Windows, and macOS OpenJDK installations.
- Host differences may be handled only by runtime dispatch through Java APIs in the GoJVM standard-library adaptation layer. They MUST NOT alter the language ABI or package selection.
- The path contract uses `/` by default. JAR, embed, and resource paths always use `/`; `java.nio.file` performs host conversion on Windows.
- `os.PathSeparator`, `filepath`, and related APIs follow documented JVM-platform rules rather than claiming perfect fidelity to any one host OS.
- Native path-list syntax, drive letters, UNC paths, permission bits, inodes, device files, symlinks, and case sensitivity belong in the platform-compatibility matrix.
- `runtime.NumCPU`, timezone, environment variables, and available memory may query the host at runtime.
- If strict Linux or Windows behavior is later needed, introduce an explicit Host Profile that changes the build ID. A single JAR MUST NOT silently change Go constant semantics according to the host.

GoJVM uses a logical layout of 64-bit words, maximum natural alignment 8, and logical little-endian. This layout is used only for `reflect.Type.Size/Align`, `StructField.Offset`, ABI metadata, and standard-library internal constants; it does not describe physical Java-object layout.

---

## 3. Overall Architecture

### 3.1 Layered Architecture

```text
Go source
   |
   v
Official Go parser / type checker / Unified IR / generics / inliner
   |
   |  GoJVM branch point
   v
GoJVM normalization and closure/value lowering
   |
   v
GoJVM-MIR
   |-- verifier
   |-- conservative optimizer
   |-- JVM representation lowering
   v
JVM bytecode emitter + class-file writer
   |
   v
Package archive with Go export data + GoJVM payload
   |
   v
GoJVM linker
   |-- whole-program reachability
   |-- interface adapter generation
   |-- package init ordering
   |-- class/resource sharding
   v
Executable JAR + gojvm runtime classes
   |
   v
OpenJDK 21+
```

### 3.2 Recommended Repository Layout

```text
src/cmd/compile/internal/gojvm/
    backend.go
    config.go
    diagnostics.go
    normalize/
    mir/
        module.go
        type.go
        value.go
        block.go
        instr.go
        verify.go
        print.go
        serialize.go
    lower/
    opt/
    abi/
    names/
    emit/

src/cmd/compile/internal/jvmclass/
    class.go
    constant_pool.go
    descriptor.go
    code.go
    stackmap.go
    attributes.go
    writer.go
    verify.go

src/cmd/link/internal/jvm/
    linker.go
    archive.go
    reachability.go
    adapters.go
    initgraph.go
    jar.go
    launcher.go

src/cmd/go/internal/gojvm/
    env.go
    build.go
    run.go
    test.go
    java.go

src/gojvm/runtime-java/
    src/main/java/org/gojvm/rt/...
    build.sh

src/runtime/
    *_jvm.go

src/internal/goarch/
src/internal/goos/
src/internal/platform/

test/gojvm/
docs/gojvm/
    design.md
    compatibility.md
    progress.md
    adr/
```

### 3.3 Controlling the Invasive Surface

Target-specific logic should be concentrated in `gojvm` subdirectories and `_jvm.go` files. Changes to shared upstream paths should:

- Add only small branches, interfaces, or hooks.
- Avoid rewriting the existing native compilation path.
- Annotate every upstream-file change with the GoJVM rationale and corresponding ADR.
- Use target registration tables rather than scattered `if goos == "jvm"` checks whenever possible.
- Preserve zero regressions in native Go's `all.bash` and existing tests.

---

## 4. Integration with the Official Go Compiler Pipeline

### 4.1 Reused Components

The implementation MUST reuse:

- Lexing and parsing.
- The `go/types`/`types2` type system.
- Unified IR.
- Generic shape and instantiation infrastructure.
- Method-set and interface type checking.
- Front-end closure-analysis information.
- Target-independent parts of devirtualization.
- Target-independent parts of inlining.
- Loop-variable semantic rewriting.
- Export data and package dependency information.

The following MUST NOT be reused as a semantic basis for the JVM backend:

- Stack/heap placement decisions from native escape analysis.
- Native ABI wrappers.
- `walk` lowerings that depend on native runtime memory layout.
- The native SSA backend.
- Register allocation.
- Native stack-frame layout.
- Native object files and relocations.
- `reflectdata` output tied to the native Go runtime type-descriptor layout.

### 4.2 Branch Point

The preferred branch occurs after these front-end steps:

1. Package loading and type checking.
2. Unified IR construction.
3. Required generic-instantiation preparation.
4. Devirtualization.
5. Inlining.
6. Target-independent wrappers from `noder.MakeWrappers`, after they are verified as such.
7. Loop-variable capture rewriting.

For Go 1.26.4's `cmd/compile/internal/gc/main.go`, the preferred insertion point is after loop-variable rewriting and `ir.CurFunc = nil`, but before `pkginit.MakeTask`, `symABIs.GenABIWrappers`, native escape analysis, `reflectdata`, `walk`, native SSA, and native object-file emission. `pkginit.MakeTask` and ABI wrappers belong to the native-backend contract; GoJVM MUST use its own initialization graph and call ABI.

Pseudocode:

```go
// After devirtualization, inlining, noder wrappers, and loopvar capture.
if buildcfg.GOOS == "jvm" && buildcfg.GOARCH == "jvm" {
    gojvm.CompilePackage(typecheck.Target, gojvm.FrontendContext{
        Package: types.LocalPkg,
        Profile: profile,
    })
    base.ExitIfErrors()
    return
}

// Existing native pkginit / ABI / escape / walk / SSA pipeline follows.
```

`cmd/compile/main.go` currently expects each GOARCH to provide `ssagen.ArchInfo`. M0 may add `cmd/compile/internal/jvm.Init`, supplying a frontend-only pseudo architecture with logical pointer/register size 8 but no instruction encoding, registers, or native ABI. The pipeline MUST branch before any `ssagen` compilation function can run. Long term, `gc.Main` should be split into a target-independent front end plus a Backend interface, reducing dependence on the pseudo architecture.

Front-end `types.PtrSize`, width, and alignment calculations may be used only to satisfy upstream type checking and constant rules. GoJVM MUST NOT treat those offsets or widths as physical JVM-object layout. It MUST maintain a separate logical type layout and field IDs.

The actual integration point MUST be confirmed by reviewing the pinned upstream phase ordering and by contract tests; function-name guesses are insufficient. Every rebase MUST revalidate the insertion point.

### 4.3 GoJVM-owned Normalization

After branching to the JVM path, perform:

1. **Desugaring:** expand compound statements into explicit control flow.
2. **Closure conversion:** turn captured variables into fields or Cells.
3. **Address-capture rewriting:** convert address-taken locals into managed location objects.
4. **Value-copy annotation:** make struct and array copy boundaries explicit.
5. **Unwind-region annotation:** mark `defer`/`panic` regions.
6. **Interface-operation expansion:** make interface operations explicit.
7. **Explicit checks:** materialize nil, bounds, divide-by-zero, and slice checks.
8. **Builtin lowering.**
9. Construct GoJVM-MIR.

GoJVM-MIR MUST NOT depend on internal object layouts from the native Go runtime.

### 4.4 Frontend Integration Contract Tests

Keep a suite of small source programs and assert the critical shape of IR immediately before the branch point, covering:

- Generic-function instantiation.
- Generic methods.
- Closure capture.
- Interface calls.
- Method values and method expressions.
- Evaluation order of arguments in `defer f(x)`.
- `range` variable semantics.
- Multiple return values.
- Named return values.
- Type aliases and defined types.
- Initialization dependencies.

When rebasing to a new Go version, run this contract suite before modifying the GoJVM backend.

---

## 5. Toolchain and Artifact Model

### 5.1 User Commands

Target user experience:

```bash
GOOS=jvm GOARCH=jvm CGO_ENABLED=0 go build -o app.jar ./cmd/app
java -jar app.jar

GOOS=jvm GOARCH=jvm go run ./cmd/app
GOOS=jvm GOARCH=jvm go test ./...
```

Even when explicitly set, `CGO_ENABLED=1` MUST produce a clear error:

```text
go: GOOS=jvm does not support cgo; use CGO_ENABLED=0 and remove imports of "C"
```

### 5.2 Package Archive Format

To reuse the Go build cache and dependency graph as much as possible, package compilation still emits a Go package archive, with GoJVM-specific members added:

```text
__.PKGDEF       # official export data / build information
__.GOJVM        # GoJVM manifest, type metadata, reachability summary
classes/...     # generated .class files for this package
resources/...   # string blocks, debug metadata, and similar data
```

`__.GOJVM` requires a versioned header:

```text
magic: GOJVMAR\0
format_version: 1
compiler_version: go1.26.4-gojvm.N
mir_schema_version: 1
classfile_major: 65
package_import_path: ...
build_id: ...
```

The linker MUST NOT depend on an unversioned internal serialization format.

### 5.3 JVM Linker

The JVM link path is responsible for:

- Reading the GoJVM payload from every package.
- Building a whole-program reachability graph.
- Dropping unreachable functions and type metadata.
- Generating interface adapters/itabs.
- Producing an explicit package-initialization order.
- Generating the entry class and `MANIFEST.MF`.
- Merging GoJVM runtime classes.
- Sharding classes, methods, and constant pools.
- Producing a deterministic JAR.
- Detecting duplicate classes and incompatible versions.
- Verifying that every target class file has major version 65.

The first release supports only `-buildmode=exe`. Other build modes MUST be rejected rather than falling through to the native linker.

### 5.4 Deterministic Builds

Identical source, dependencies, compiler version, and flags MUST produce byte-identical JARs. The implementation MUST:

- Sort JAR entries.
- Fix ZIP timestamps to a constant.
- Stably order constant-pool generation inputs.
- Assign stable names to synthetic classes and methods.
- Avoid embedding absolute working directories.
- Distinguish upstream Go version, GoJVM version, MIR version, and JDK target version in the build ID.
- Include a two-build SHA-256 equality test.

### 5.5 `go run` and `go test`

`go run` performs:

1. Build a temporary JAR.
2. Invoke the configurable `JAVA_HOME/bin/java`.
3. Forward program arguments.
4. Map the JVM exit code back to `go run`.
5. Never automatically use newer bytecode features merely because the host JDK is newer.

`go test`:

- Reuses the official generated test-main mechanism.
- Builds the test binary as a JAR.
- Launches it with `java -jar`.
- Supports common flags such as `-run`, `-bench`, `-count`, and `-timeout`.
- Does not initially support native coverage instrumentation; later coverage may be implemented with MIR counters.
- On timeout, emits a GoJVM goroutine dump before terminating the JVM.

---

## 6. GoJVM-MIR Specification

### 6.1 Design Goals

GoJVM-MIR MUST provide:

- Typed basic blocks.
- Explicit locals.
- Explicit control flow.
- Explicit calls.
- Explicit nil checks and bounds checks.
- Explicit object, array, slice, map, interface, and channel operations.
- No machine register allocation.
- No native stack-frame layout.
- No raw address arithmetic in the default model.
- A stable mapping to the JVM verifier type system and `StackMapTable`.
- Structural and type verification without executing JVM bytecode.
- A form suitable for text serialization, binary serialization, differential testing, and fuzzing.

### 6.2 Core Invariants

1. Every local has one unique, immutable MIR type within a function.
2. Every basic-block entry has a defined initialized-local environment.
3. The JVM operand stack is empty at every normal basic-block entry and exit.
4. Complex expressions are decomposed into linear instructions.
5. Every implicit language operation that may panic is represented as an explicit check or an explicit runtime call.
6. Every control transfer explicitly names its target blocks.
7. Every value-type copy is represented by `copy.value` unless the verifier proves it can be eliminated.
8. Every address value is a managed location, never an integerized raw pointer.
9. Every call records call kind, signature, whether it may raise a Go panic, and whether it is a scheduling safe point.
10. MIR verification failure is a compiler error. The compiler MUST NOT emit a “best effort” class file.

### 6.3 Type System

Recommended MIR type families:

```text
Void
Bool
I8 I16 I32 I64
U8 U16 U32 U64
F32 F64
Complex64 Complex128
GoInt GoUint GoUintptr
String
Array<T, N>
Slice<T>
Struct<TypeID>
Pointer<T>
Map<K, V>
Chan<T, Dir>
Func<SigID>
Interface<InterfaceID>
ConcreteRef<TypeID>
TypeRef
ItabRef
PanicRef
RecoverToken
RuntimeRef<ClassID>
```

The type system MUST distinguish signed and unsigned Go integers even when they map to the same JVM primitive category. Failing to preserve this distinction causes incorrect comparisons, division, remainder, and formatting.

`Pointer<T>` is a logical location type, not a raw JVM address.

### 6.4 Modules, Packages, and Functions

Conceptual structure:

```text
module GoJVMModule {
  version
  package
  imports
  type_table
  const_table
  globals
  functions
  init_dependencies
  debug_files
}

function {
  id
  go_symbol
  jvm_owner
  jvm_name
  signature
  locals
  entry_block
  blocks
  unwind_regions
  source_map
  attributes
}
```

### 6.5 Locals

Persistent MIR does not require SSA single assignment. Reasons:

- JVM local-variable slots are naturally mutable.
- Explicit locals make stable `StackMapTable` generation easier.
- Large-method splitting can move locals into a Frame object naturally.
- Named results, `defer`, and addressable locals are easier to model directly.

Optimization passes MAY use temporary SSA or data-flow analysis, but SSA MUST NOT become the persistent MIR format.

Example local declarations:

```text
local %0 : GoInt    # parameter n
local %1 : GoInt    # result
local %2 : Bool
local %3 : PanicRef
```

To simplify JVM verification, every JVM slot corresponding to a MIR local MUST be initialized in the function prologue to the type's zero value or `null`. Go locals already have language-level zero values, so this does not alter observable Go semantics.

### 6.6 Basic Blocks

Each block contains metadata and instructions of this form:

```text
block bb0(
  locals_in: {%0: GoInt initialized, %1: GoInt initialized},
  handler: none
) {
  ...instructions...
  terminator ...
}
```

A block type consists of:

- The initialized-local set.
- The fixed type of every local.
- The current exception/unwind handling context.
- Optional non-null facts.
- Optional range facts used only for verification and optimization, never as a semantic requirement.

### 6.7 Instruction Categories

#### 6.7.1 Constants and Moves

```text
const.bool
const.i64
const.f64
const.string
zero
move
copy.value
```

`move` is valid only for scalars and values with reference semantics. Go assignment of arrays and structs uses `copy.value` by default.

#### 6.7.2 Arithmetic and Logic

```text
add sub mul
sdiv udiv srem urem
neg
and or xor not
shl sshr ushr
cmp.eq cmp.ne
cmp.slt cmp.sle cmp.sgt cmp.sge
cmp.ult cmp.ule cmp.ugt cmp.uge
convert
```

Shift counts MUST first undergo explicit Go-semantic checking or normalization. The JVM masks high bits of the shift count; GoJVM MUST NOT depend on that behavior.

#### 6.7.3 Explicit Checks

```text
check.nil        value, panic_site
check.bounds     index, length, panic_site
check.slice      low, high, max, cap, panic_site
check.divzero    divisor, panic_site
check.shift      count, panic_site_or_normalization
check.make_len   len, element_kind, panic_site
check.typeassert interface, target_type, comma_ok
check.mapkey     key_type_rules
```

An optimizer may remove a check only after proving it redundant while preserving panic order and the order of observable side effects.

#### 6.7.4 Objects, Fields, and Value Types

```text
new.object
new.value
get.field
set.field
copy.field
zero.value
box.value
unbox.value
```

#### 6.7.5 Arrays and Slices

```text
new.array
array.len
array.load
array.store
array.addr
slice.make
slice.from_array
slice.len
slice.cap
slice.index
slice.addr
slice.reslice
slice.copy
slice.append
```

For value elements, `array.load` MUST define whether it returns a copy. Prefer type rules that insert `copy.value` automatically rather than leaving the decision to the emitter.

#### 6.7.6 Strings

```text
string.len_bytes
string.index_byte
string.slice
string.concat
string.eq
string.compare
string.to_bytes_copy
bytes.to_string_copy
string.range_decode
```

#### 6.7.7 Managed Pointers

```text
ptr.root.local_cell
ptr.root.global_cell
ptr.root.heap
ptr.project.field
ptr.project.array_elem
ptr.project.slice_elem
ptr.load
ptr.store
ptr.eq
ptr.is_nil
```

There is no `ptr.to_uintptr`, `uintptr.to_ptr`, or arbitrary pointer arithmetic.

#### 6.7.8 Maps

```text
map.make
map.len
map.lookup
map.lookup_ok
map.assign
map.delete
map.clear
map.iter.init
map.iter.next
```

#### 6.7.9 Interfaces

```text
iface.make
iface.nil
iface.type
iface.data
iface.itab
iface.call
iface.assert
iface.assert_ok
iface.eq
```

#### 6.7.10 Channels and Concurrency

```text
chan.make
chan.send
chan.recv
chan.recv_ok
chan.close
chan.len
chan.cap
select.begin
select.case.send
select.case.recv
select.case.default
select.commit
sched.go
sched.yield
sched.poll
```

#### 6.7.11 Calls and Returns

```text
call.static
call.method
call.funcvalue
call.interface
call.runtime
return
```

Every call instruction MUST include:

- The complete Go signature.
- The lowered JVM signature.
- Argument-local list.
- Result-local list.
- Whether it may block in the scheduler.
- Whether it may throw an internal Go unwind exception.
- The current recover-token propagation policy.
- Source position.

#### 6.7.12 `defer`, `panic`, and Unwinding

```text
defer.push
panic.raise
panic.resume
recover.try
goexit.raise
unwind.enter
unwind.next
```

#### 6.7.13 Control-flow Terminators

```text
br target
br.if cond, true_target, false_target
switch value, cases..., default
return values...
throw.internal value
unreachable
```

### 6.8 Textual Syntax Example

Go source:

```go
func Sum(a []int, i int) int {
    return a[i] + 1
}
```

MIR:

```text
func @pkg.Sum(%a: Slice<GoInt>, %i: GoInt) -> GoInt {
  local %len: GoInt
  local %v: GoInt
  local %r: GoInt

bb0:
  check.nil %a panic="nil slice descriptor"
  %len = slice.len %a
  check.bounds %i, %len panic="index out of range"
  %v = slice.index %a, %i
  %r = add %v, 1
  return %r
}
```

In actual Go, taking the length of a nil slice is valid. Therefore the correct optimized MIR MUST NOT panic merely when reading a nil slice descriptor. The intentionally incorrect example above is a warning: **checks must be inserted from Go semantics, not from intuition about the Java representation.** A correct form is:

```text
bb0:
  %len = slice.len_or_zero %a
  check.bounds %i, %len panic="index out of range"
  %v = slice.index_checked %a, %i
  %r = add %v, 1
  return %r
```

This case MUST be retained as a regression test so Java `null` behavior never leaks into Go semantics.

### 6.9 MIR Verifier

At minimum, the verifier MUST check:

- Reachability of CFG blocks and completeness of terminators.
- Local definitions, types, and definite initialization.
- Legality of local-environment merges at block entries.
- Call argument and result type matching.
- That explicit nil, bounds, and divide-by-zero checks dominate operations that require them, unless the operation is itself declared checked.
- That unrecoverable JVM exceptions are not treated as Go panics.
- Completeness of value-copy boundaries.
- Proper nesting of unwind regions.
- That recover tokens cannot leak or be stored on the heap.
- Absence of raw address arithmetic.
- Absence of residual cgo, JNI, native methods, or `unsafe` operations.
- An empty JVM stack at block boundaries.
- That projected JVM parameter slots, local slots, method code, and constant-pool sizes do not overflow limits.
- Stable and conflict-free naming of every synthetic symbol.

### 6.10 MIR Optimization

The first release permits only optimizations that are easy to prove and do not alter panic or side-effect order:

- Constant folding.
- Unreachable-block deletion.
- Empty-jump merging.
- Local copy propagation.
- Removal of demonstrably redundant nil or bounds checks.
- Backend supplementary inlining of small pure functions.
- Value-copy elimination.
- Constant-string concatenation.
- Selection of an appropriate switch form.
- Boxing elimination.

The first release MUST NOT implement aggressive exception-path reordering, cross-call check elimination, or optimizations that rely on a particular JVM JIT implementation.

---

## 7. JVM Bytecode Backend

### 7.1 Class-file Writer

Implement a small, auditable class-file writer in Go rather than introducing ASM as a compiler build dependency. It MUST support:

- Constant pools.
- Fields and methods.
- The `Code` attribute.
- Exception tables.
- `StackMapTable`.
- `LineNumberTable`.
- `LocalVariableTable` in debug builds.
- `SourceFile`.
- `InnerClasses`, `NestHost`, and `NestMembers` if used.
- Custom GoJVM metadata attributes.
- Class-file version 65.
- Deterministic serialization.

Even where the JVM specification permits frame omission, GoJVM MUST explicitly emit a correct `StackMapTable`. It MUST NOT depend on permissive verification behavior in any particular JVM release.

The class-file writer requires unit tests for these common traps:

- `CONSTANT_Utf8` uses modified UTF-8, not ordinary UTF-8; its length is a `u2` byte count.
- `long` and `double` constant-pool entries occupy two indices.
- The verifier's uninitialized-object type between `new` and `invokespecial <init>` may not cross illegal control flow.
- `uninitializedThis` at constructor entry.
- `long` and `double` are category-2 values in local slots and on the operand stack.
- An exception-handler entry stack contains exactly one `Throwable`, which should immediately be stored in a local.
- Conditional branches have only short offsets; far targets require condition inversion plus `goto_w`.
- Four-byte alignment and relative offsets for `tableswitch` and `lookupswitch`.
- Selection of `wide`, `ldc`, or `ldc_w`, and correct maximum-stack calculation.
- Offset deltas and append/chop/full-frame encodings in stack-map frames.
- Backpatching final PC ranges for exception, line-number, and local-variable tables after layout.
- Rechecking method code length after branch expansion and switch padding.

### 7.2 Operand-stack Policy

Use an **empty stack at block boundaries** policy:

- Each MIR instruction uses the JVM operand stack only transiently within its block.
- Instruction results are immediately stored into JVM locals.
- The operand stack is empty before every branch.
- Exception-handler entry contains only the JVM-mandated exception object, which is immediately stored in a dedicated local.
- Stack values never flow across basic-block boundaries.

Benefits:

- Simple `StackMapTable` generation.
- Safe CFG rewriting.
- Clear error localization.
- Easier large-method splitting.
- No complex phi merging on the JVM stack.

### 7.3 JVM Local-slot Allocation

This is not machine register allocation. Slot allocation only needs to:

- Assign a JVM local index to each MIR local.
- Reserve two slots for `long` and `double`.
- Preserve one verifier type for each local throughout the function.
- Avoid slot reuse in the first release for verification stability and debugging clarity.
- Later, MAY reuse slots with non-overlapping lifetimes, subject to verifier-type compatibility.
- Trigger method splitting before approaching 65,535 local slots.

### 7.4 Handling JVM Limits

The backend MUST proactively handle:

- The approximately 64 KiB maximum `Code` attribute for one method.
- The 255 parameter-slot limit.
- The 65,535 constant-pool-entry limit.
- The 65,535 field and method count limits.
- Local-index and maximum-stack limits.
- UTF-8 constant-length limits.
- Signed 32-bit JVM array indexing.

Required strategies:

1. **Large-method splitting**
   - Partition the CFG into region methods.
   - Store live locals in a synthetic Frame object.
   - Use a numbered-state dispatcher.
   - Never split `defer`/unwind regions unsafely.
   - Run MIR-equivalence tests before and after splitting.

2. **Argument carrier**
   - Generate a synthetic argument object when JVM parameter slots approach 255.
   - Preserve the user-visible Go signature in metadata.
   - Apply one generation rule to direct calls and reflection.

3. **Class sharding**
   - Distribute large packages across multiple owner classes using a stable hash.
   - Store strings and large constants in resources or constant-block classes.
   - Emit interface adapters in groups.

### 7.5 Bytecode-verification Gate

Every generated JAR MUST at least be tested with:

```bash
java -Xverify:all -jar app.jar
javap -v -classpath app.jar generated.ClassName
```

Running only the entry point does not load every unreachable or cold-path class. A `VerifyJar` tool is therefore required. It enumerates every `.class` in the JAR, defines and explicitly resolves each class through an isolated class loader without executing user `<clinit>`, and also parses every method and `StackMapTable` using GoJVM's own structural scanner. If the Java API cannot guarantee complete linkage verification for a class, the test tool should generate a temporary verifier class that creates controlled symbolic references to all target classes and forces resolution.

CI MUST include a class-file structural scanner that verifies:

- Every class has version 65.
- No method is `native`.
- No JNI symbol is present.
- There is no unintended dependency on `sun.misc.Unsafe` or `jdk.internal.misc.Unsafe`.
- No preview attribute is present.
- `StackMapTable` agrees with the CFG.
- No host absolute path is embedded.

---

## 8. Symbols, ABI, and Calling Conventions

### 8.1 Name Encoding

Java internal names MUST:

- Be derived stably from the Go import path, declaration name, receiver type, and signature.
- Escape `/`, `.`, Unicode, reserved words, and characters illegal in JVM identifiers.
- Use a readable prefix plus a stable hash for excessively long names.
- Never use process-unstable values such as `hashCode()`.
- Preserve a reverse mapping table for stack-trace beautification.

Example:

```text
Go: example.com/acme/math.(*T).Add
JVM owner: gojvm/p/8f31acme_math/FnShard0
JVM method: m$T$Add$7d29c1
```

### 8.2 Functions and Methods

- Package-level functions are emitted as `static` methods.
- Go methods should also be emitted as `static` methods, with the receiver as the first explicit parameter.
- Java virtual-method rules MUST NOT substitute for Go method-set rules.
- Java visibility exists mainly for verifier and runtime access and does not define a public Java API.
- Multiple return values use a synthetic strongly typed result carrier.
- No result maps to `void`.
- One scalar or reference result is returned directly.
- A large value result MAY later use a caller-allocated carrier as an optimization.

### 8.3 Function Values and Closures

Generate a signature-specific JVM interface for each Go function signature:

```java
interface Fn$abc123 {
    long invoke(Object env, long x, RecoverToken token);
}
```

The exact interface shape is specialized by signature, avoiding the boxing and runtime type hazards of a universal `Object[]` convention.

Closure rules:

- A function value with no captures may use a singleton.
- By-value captures become fields.
- Captures that are mutated or address-taken use Cells.
- Closure field order is stable.
- Function values may be compared only with nil; Java object equality MUST NOT become Go function equality.

### 8.4 Recover Token

To implement precisely the rule that only a direct call from a deferred function may recover, the internal ABI appends a hidden `RecoverToken` parameter to every callable Go function:

- Ordinary calls pass `null`.
- When the defer dispatcher directly invokes a deferred function during a panic, it passes that panic's token.
- Any further call made by the deferred function MUST pass `null`.
- `recover()` succeeds only when the token is non-null, valid, unconsumed, and matches the active panic.
- A token MUST NOT be stored on the heap, returned, captured, or propagated across goroutines.
- The upstream or GoJVM inliner MUST NOT inline a function containing `recover` into a position that changes the direct-call test. This property MUST be verified after the branch point.
- Wrappers, generic-dictionary calls, interface adapters, and function-value trampolines MUST explicitly choose between `null` and the active defer token; token forwarding MUST NOT be an implicit default.
- The MIR verifier MUST validate all token flows.

---

## 9. Representation of Go Values on the JVM

### 9.1 Scalars

| Go type | JVM representation | Notes |
|---|---|---|
| `bool` | `int`/`boolean` | Internally normalized to 0 or 1 |
| `int8/uint8` | `int` | Truncate on write; unsigned semantics are explicit |
| `int16/uint16` | `int` | Same rule |
| `int32/uint32` | `int` | Dedicated unsigned compare, divide, and remainder |
| `int64/uint64` | `long` | Explicit unsigned operations |
| `int/uint/uintptr` | `long` | Fixed at 64 bits for `GOARCH=jvm` |
| `float32` | `float` | Preserve Go rounding boundaries |
| `float64` | `double` | Java 17+ strict floating-point semantics may be relied upon |
| `complex64/128` | Two floating-point locals; carrier object when escaping | Do not box by default |

`uintptr` is only an unsigned integer. Because `unsafe` is unsupported, no conversion between it and managed references is provided.

#### 9.1.1 Integer, Shift, and Conversion Semantics

JVM `int` and `long` operations are carriers, not a complete implementation of Go integer semantics. The backend MUST obey these rules:

- Fixed-width addition, subtraction, multiplication, and negation truncate to the corresponding width. Observable write boundaries for `int8`, `uint8`, `int16`, and `uint16` require explicit narrowing.
- A narrow integer read is sign-extended or zero-extended according to its Go type. Java's signed `byte` and `short` types MUST NOT determine Go unsigned behavior.
- `uint32` comparison, division, and remainder use unsigned 32-bit operations. `uint64` uses `Long.compareUnsigned`, `Long.divideUnsigned`, `Long.remainderUnsigned`, or a verified equivalent.
- Signed minimum divided by `-1` retains the minimum value with remainder zero, as required by Go; it is not reported as overflow.
- Integer division by zero raises a Go runtime panic at MIR `check.divzero`, rather than relying on JVM `ArithmeticException`.
- A runtime shift count that is negative raises a Go panic. When the count is at least the operand width, the result follows Go rules—zero or a sign-extended result—not the JVM's masked-count result.
- Constant expressions are evaluated at arbitrary precision by the Go front end, and representability is checked when assigning to a concrete type. The compiler MUST NOT first place them in JVM constants and rely on overflow.
- Integer-to-integer conversion first truncates to the target width and then interprets the bits with the target signedness.
- `uintptr` and `uint64` share a width but retain distinct type identity, and there is no managed-reference conversion instruction.
- The 64-bit choice for `int` and `uint` is part of the public `GOARCH=jvm` platform ABI. Changing it requires an ABI-version increase and a new ADR.

#### 9.1.2 Floating-point and Complex Semantics

- Every observable `float32` result MUST be rounded at 32-bit precision. Intermediate values MUST NOT remain as `double` until only the final store.
- Java 17 and later always-strict floating-point evaluation may be relied upon, but Go conversions, complex arithmetic, and library functions still require dedicated tests.
- NaN is unequal to itself; `+0` and `-0` are equal. Map hashing MUST place both zeros in one equivalence class.
- Conversion of out-of-range floating values, NaN, and infinities to integers lies in an implementation-defined area permitted by Go. GoJVM MUST define one deterministic rule in `docs/gojvm/platform.md` and preserve it. A recommended rule is to use Java 21's conversion result and then truncate to the target width, but behavior MUST NOT drift across supported JDKs.
- Integer-to-float conversion rounds to the nearest representable value. `uint64` values above `MaxInt64` require an explicit unsigned path.
- `complex64` rounds real and imaginary components at `float32` boundaries; `complex128` uses two `double` values. Complex division MUST use a scaling algorithm validated against Go tests rather than a naïve formula that can overflow spuriously.
- Compatibility of `math` functions for NaN payloads, signed zero, infinity, and boundary rounding requires differential testing. Functions where Java `Math` differs from Go use GoJVM-specific implementations.

### 9.2 Go Strings

Go strings MUST NOT be represented as `java.lang.String`. A Go string is an arbitrary byte sequence and may contain invalid UTF-8; a Java String is a UTF-16 text abstraction.

Recommended representation:

```java
final class GoString {
    final byte[] data;
    final int off;
    final int len;
}
```

Rules:

- A string is never nil. Its zero value is a shared zero-length instance.
- String slices may share an immutable backing array.
- `[]byte(s)` MUST copy.
- `string(b)` MUST copy unless the compiler proves the backing can never again be modified; do not perform this optimization in the first release.
- `len` returns the byte length.
- Indexing returns a byte.
- `range` follows Go UTF-8 decoding. Invalid encodings produce `utf8.RuneError` and advance one byte.
- String comparison uses unsigned-byte lexicographic order and MUST NOT call `java.lang.String.compareTo`.
- `string(rune)` emits U+FFFD for an invalid Unicode code point.
- `[]rune(s)` decodes using Go UTF-8 rules and allocates an independent array. `string([]rune)` UTF-8-encodes and copies.
- Runtime boundaries between Go strings and Java Strings MUST explicitly specify UTF-8 and are allowed only for host APIs that are semantically textual. Arbitrary-byte paths MUST NOT use `Charset.defaultCharset()`.
- When filenames, environment variables, or other host text cannot losslessly represent arbitrary Go bytes, the platform-compatibility document MUST specify an error or escape policy rather than silently replacing bytes.
- String literals are stored as raw bytes and are not round-tripped through Java source-string encoding.
- Large literals are stored as JAR resources or chunked static byte arrays to avoid constant-pool UTF-8 limits.
- The compiler may coalesce immutable backing arrays, but MUST NOT expose a writable alias through `[]byte(s)`, `string([]byte)`, or reflection.

### 9.3 Structs and Arrays

Go structs and arrays have value semantics; Java objects have reference semantics. Aliasing MUST therefore be handled explicitly.

Initial strategy:

- Generate a managed value object for each Go struct.
- Represent each Go array with a generated wrapper or element-kind-specific backing.
- Insert `copy.value` at Go assignment, parameter passing, return, and interface-boxing boundaries.
- Recursively copy value fields inside structs.
- Shallow-copy descriptors or references for slices, maps, channels, functions, pointers, and interfaces.
- Copy elimination is allowed only after proving no observable alias.
- Zero values MUST correctly handle nested structs and arrays.

To avoid enormous zero-value object trees, the runtime may use a lazy zero value. An unmaterialized nested value field can use `null` to mean the logical zero value; reads return a zero-value view, while writes and address-taking materialize storage. This optimization MUST be invisible to Go code.

MIR MUST explicitly represent value-copy boundaries:

| Go operation | Struct/array behavior | Reference-semantic fields |
|---|---|---|
| Ordinary assignment | Destination receives an independent value copy | Slice/map/channel/function/pointer/interface descriptors are shallow-copied |
| Function argument | Copy before the call | Same |
| Function return | Copy at the return boundary; may be proven removable | Same |
| Interface boxing | Copy dynamic value into the box | Reference fields inside the box are shallow-copied |
| Value-producing type assertion | Return a copy of the dynamic value | Same |
| Map key/value insertion | Copy both key and value | Same |
| Map index read | Return a copy of the value | Same |
| Channel send | Form a copy during send-expression evaluation | Same |
| Channel receive | Copy into the receive target | Same |
| `range` iteration variable | Assign/copy each iteration according to the language version | Same |
| Array/slice element read as a value expression | Copy according to context; address-taking produces a location | Same |

Copy elision MUST prove that source and destination cannot later be observed simultaneously through any managed pointer, interface box, closure, map, channel, or reflection. Merely observing that a Java object currently has one reference is insufficient across a call.

### 9.4 Slices

Recommended representation:

```java
final class SliceRef {
    Object backing; // primitive array or Object[]
    int off;
    int len;
    int cap;
    ElementOps ops;
}
```

Element-kind-specific slice classes MAY be generated to reduce boxing.

The implementation MUST distinguish:

- Nil slice: descriptor reference is `null`.
- Non-nil empty slice: descriptor is non-null with `len=0`.
- `len(nil)==0` and `cap(nil)==0`.
- `append` may reuse the backing array or allocate a new one.
- Go `int` is 64-bit, while JVM array length is an `int`. Any array or backing allocation larger than `math.MaxInt32` MUST explicitly panic, and the limitation MUST be documented.
- Load, store, and copy for value elements follow Go value-copy semantics.
- Overlapping `copy` follows Go rules.

### 9.5 Managed Pointers

Pointers do not contain raw addresses. A pointer denotes a **stable root location plus a typed projection path**, not a Java value object that happened to be read at one moment:

```text
RootLocation:
  LocalCell<T>
  GlobalCell<T>
  HeapObjectRoot<T>
  ArrayBackingRoot<T>
  SliceBackingRoot<T>

ProjectionStep:
  Field<FieldID>
  ArrayIndex<ConstOrRuntimeIndex>
  DerefCell

ManagedPtr<T> { root, canonicalProjection }
```

This model is required by code such as:

```go
s := T{X: 1}
p := &s.X
s = T{X: 2}
println(*p) // must print 2
```

If `p` retains only a reference to the field in the first value object, replacing `s` would incorrectly print 1. The correct implementation promotes address-taken `s` to a stable Cell, and `p` follows the `Cell -> field X` projection to the current storage on every access.

Rules:

- Every location kind provides typed `get` and `set` operations.
- Whole-value assignment to an address-observable struct or array MUST either overwrite stable storage in place or allow projection pointers to resolve through a root Cell to the new value. Existing derived pointers MUST NOT become stale.
- `*p = aggregate` on a heap root with derived pointers MUST copy fields or elements into the existing root rather than replacing root identity.
- Nil equals nil.
- Pointer equality is canonical root identity plus canonical projection.
- A pointer to a slice element identifies the backing root plus a fixed index. A later `append` that replaces the slice backing does not change the old pointer.
- `&*p` normalizes to `p`.
- Nested field and array projections MUST be canonicalized; ad hoc string paths are not acceptable for equality.
- There is no arbitrary byte offset within an object.
- A pointer cannot be converted to `uintptr` and back.
- Map elements are not addressable; the MIR verifier MUST reject forming such an address.
- For pointers to zero-sized objects, choose one stable behavior within the latitude permitted by the Go specification and lock it down with tests.
- The optimizer may replace or rebind a stable-root object only after proving the absence of address observation and derived pointers.

Before MIR construction, addressability analysis MUST convert a local variable—or any subobject of it—that is address-taken into a Cell when necessary. This is different from native escape analysis; the Cell itself is managed by JVM GC. A pointer MUST NOT refer to a JVM local slot.

### 9.6 Maps

`java.util.HashMap` MUST NOT be used directly as the semantic implementation because Go key equality and hashing differ from Java, especially for floating-point NaN, signed zero, interface keys, and struct keys.

Implement a GoJVM-managed hash table whose memory is ordinary JVM-managed memory:

- Open addressing or bucket-based organization is acceptable.
- Generate `TypeOps` for each key type: `hash`, `equal`, and `copy`.
- Float keys follow Go equality rules.
- Struct and array keys recurse by component.
- Interface keys combine dynamic type and dynamic value.
- Using an incomparable dynamic value as an interface key panics.
- A nil map is represented by `null`. Read, `len`, `delete`, and `clear` are valid on nil; assignment to a nil map panics.
- A map index result is not addressable. Reading a value type returns a copy; insertion copies both key and value under Go rules.
- A NaN key can be inserted and observed during `range`, but a later lookup using even the same NaN value may fail because NaN is unequal to itself.
- Positive and negative zero compare equal and therefore MUST hash into one equivalence class while retaining any representation permitted for the stored key.
- Iteration order is unstable and may vary between iterations. The visibility of insertions and deletions during `range` MUST remain within the set permitted by the Go specification.
- Exact reproduction of native Go's concurrent-read/write diagnostic is not required initially, but concurrent misuse MUST NOT corrupt JVM memory.

### 9.7 Interfaces and Typed Nil

Recommended interface value:

```java
final class GoIface {
    GoType type;     // dynamic concrete type; null for a nil interface
    Object data;     // boxed data; may be null for a typed nil
    Object itab;     // interface-specific dispatch adapter
}
```

The implementation MUST guarantee:

- Nil interface: `type == null`.
- Typed nil: `type != null` and `data == null`; therefore the interface itself is non-nil.
- Assertions from empty and non-empty interfaces follow Go rules.
- Interface equality panics when the dynamic value is incomparable.
- Interface calls use Go interface method sets rather than guessed Java inheritance.
- Identity of an unexported method includes its declaring package path. Identically named and typed unexported methods from different packages are not the same interface method.
- Defined types, type aliases, instantiated generic types, and pointer/non-pointer method sets use canonical type identity from the Go front end rather than being merged only because JVM descriptors match.
- Embedded-interface method sets are normalized, deduplicated, and stably ordered before linking, but method ordinals are not a public promise across ABI versions.
- Itab adapters preserve value copies for value-receiver wrappers and preserve nil-receiver semantics for pointer receivers.

Generate a signature-specific JVM itab interface for each Go interface. The linker creates adapters for reachable concrete-type/interface pairs. Reflection and dynamic assertions use a whole-program type registry as a fallback.

An interface-call adapter MUST NOT panic merely because `data == null`. When the dynamic type is `*T` and the dynamic value is nil, a pointer-receiver method receives a nil receiver and decides its own behavior. A panic occurs only at a Go-defined implicit dereference, value-receiver wrapper, or an actual dereference in the method body.

### 9.8 Comparability of Functions, Maps, and Slices

- Slices, maps, and functions may be compared only with nil.
- Their comparison MUST NOT degrade to Java reference equality.
- Comparing an interface containing one of these dynamic values against a non-nil interface panics according to Go rules.
- Channels and pointers are comparable and use Go identity semantics.

### 9.9 Logical Object Identity and Program Counters

Several Go APIs expose pointer-like IDs or program counters as `uintptr`, but GoJVM MUST NOT leak JVM addresses:

- Assign a collision-resistant 64-bit logical ID—monotonic or randomized—to every owner, Cell, channel, map, function, and code location whose identity is observable.
- `fmt` `%p`, permitted reflection cases, and debugging output use logical IDs rather than Java `identityHashCode`.
- `runtime.Caller` returns an encoded logical program counter.
- `runtime.FuncForPC`, `CallersFrames`, and symbolization accept only entries from that logical-PC registry.
- Logical IDs are stable within one process but need not match across processes.
- Because `unsafe` is forbidden, a logical `uintptr` cannot be converted back into a managed reference.
- JVM `identityHashCode` is only 32-bit and may collide; it MUST NOT implement Go pointer equality or weak-pointer identity.

### 9.10 Nil-behavior Matrix

Java `null` is only an internal representation choice. Every operation must be classified by Go semantics first; an NPE MUST NOT substitute for a language rule:

| Go value/operation | Required behavior |
|---|---|
| `*p` where `p == nil` | Explicit Go nil-dereference panic |
| Pointer-receiver method on a nil pointer | Method is entered; it may handle nil itself |
| Value-receiver wrapper on a nil pointer | Go panic at implicit dereference/copy |
| `len/cap(nilSlice)` | 0 |
| `append(nilSlice, ...)` | Allocate/append normally |
| `range nilSlice` | Zero iterations |
| `len(nilMap)`, lookup, `delete`, `clear` | 0, zero value, or no-op as appropriate |
| Assignment to a nil map | Go panic |
| `len/cap(nilChan)` | 0 |
| Send, receive, or range on a nil channel | Block forever |
| Close a nil channel | Go panic |
| Call a nil function | Go panic; timing in `defer` and `go` follows Go rules |
| Nil interface | No dynamic type; equals nil |
| Typed-nil interface | Has a dynamic type; does not equal nil |
| `len(nil string)` | Not applicable; strings are non-nil and zero to empty |

Every path requires tests asserting that `recover` observes a Go runtime-error value rather than `NullPointerException`, `ClassCastException`, or `ArrayIndexOutOfBoundsException`.

---

## 10. Goroutines and JVM Virtual Threads

### 10.1 One-to-one Carrier Mapping

Each `go f()` creates a JVM virtual thread:

```java
Thread.startVirtualThread(() -> GoRuntime.runG(entry));
```

A platform thread MUST NOT be created per goroutine.

The function value and all arguments of a `go` statement MUST first be evaluated in the parent goroutine, in Go evaluation order, with all required value copies completed before the child virtual thread is created and started. A Java lambda MUST NOT defer reads of parent-stack locals into the child thread; doing so would change evaluation timing, which goroutine owns a panic, and data-race behavior. Startup failure or resource exhaustion is handled as a Go runtime error in the parent goroutine.

### 10.2 `main` Lifecycle

JVM virtual threads are daemon threads. If the platform main thread exits, the JVM may terminate virtual threads that are still running. The entry sequence MUST therefore:

1. Use the JVM platform `main` thread only to bootstrap the kernel without executing user Go code: parse arguments, establish the exit coordinator, and load metadata.
2. Create the initial virtual thread and `GState` that carry Go `runtime.main`.
3. On that initial Go virtual thread, sequentially perform Go runtime initialization, all package initialization, and `main.main`.
4. Have the platform main thread join the initial Go virtual thread while responding to the global exit coordinator.
5. When Go `main` returns, apply Go process-exit semantics without waiting for ordinary remaining goroutines.
6. On an unrecovered panic from init or main, print a Go-style stack and exit nonzero.

Package initialization may start goroutines, block, panic, defer, or call runtime APIs. It MUST never run directly on the platform launcher thread.

### 10.3 `GState`

A virtual thread's `ThreadLocal` stores only a compact `GState`:

- GoJVM goroutine ID.
- State: runnable, running, waiting, syscall, or dead.
- Current panic chain.
- Defer-dispatch context.
- Current scheduling permit.
- Wait reason.
- Go frame-metadata chain or sampling information.
- Future race/trace extension bits.
- Internal cancellation/termination flags.

Large user objects MUST NOT be retained in `ThreadLocal`, because that would extend their lifetime. GoJVM allocates goroutine IDs; JVM Thread IDs are not used. When a virtual thread terminates, it MUST be removed from the registry, and its `ThreadLocal` and waiter references MUST be cleared.

Java thread interruption is not Go context cancellation. Runtime park/unpark uses explicit waiter state. Any unexpected interrupt MUST be consumed, restored, or converted into the corresponding I/O error under controlled rules; `InterruptedException` MUST NOT randomly become a Go panic. Initially forbidding user Java interoperation reduces this semantic entry point.

### 10.4 `GOMAXPROCS`

JVM virtual-thread scheduler parallelism is not equivalent to Go `GOMAXPROCS`. To preserve observable semantics, GoJVM uses **execution permits**:

- At most `GOMAXPROCS` virtual threads may hold permits while executing Go user code.
- A goroutine releases its permit before entering a GoJVM runtime operation that may block for a long time.
- It reacquires a permit before returning to user code after wakeup.
- The compiler inserts `sched.poll` in function prologues, loop back edges, and suitably long straight-line regions.
- `runtime.GOMAXPROCS(n)` dynamically adjusts the permit count.
- Adjustment does not revoke permits already held; it affects future acquisitions.
- Scheduler fairness should approximate Go where practical, but execution order is not promised to match native Go.

This is safer than merely changing global JVM virtual-thread scheduler parallelism, which is process-wide, may affect non-Go code, and cannot fully express Go blocking states.

### 10.5 Blocking and Pinning

On the minimum JDK 21, a virtual thread can pin its carrier while blocked in some `synchronized` regions. JDK 24 substantially improves this, but GoJVM MUST remain correct on JDK 21. Therefore:

- Runtime blocking paths avoid `synchronized` as their primary lock mechanism.
- Use `ReentrantLock`, `LockSupport`, AQS, and atomic variables.
- Never execute user code while holding a runtime lock.
- CI on JDK 21 enables virtual-thread pinning diagnostics during stress tests.
- JDK 21-safe paths MUST NOT be removed merely because JDK 24 or later performs better.

### 10.6 `runtime.LockOSThread`

Virtual threads do not provide a stable one-to-one platform-thread identity. The first release marks `runtime.LockOSThread` unsupported:

- A statically recognized direct call is a compile-time error.
- A reflective or indirect call panics at runtime with a precise diagnostic.
- Ported standard-library code MUST NOT depend on it.

### 10.7 Deadlock Detection

The JVM does not report Go's “all goroutines are asleep” condition. The GoJVM runtime maintains a registry containing:

- Active goroutine count.
- Runnable/running/waiting states.
- Main-goroutine state.
- Timers and external I/O that could produce future wakeups.
- Channel/select wait graph.

When there is no running or runnable goroutine, no future timer or external event, and main has not returned normally, the runtime prints a Go-style deadlock diagnostic. It MUST avoid treating legitimate network-I/O waits as deadlock.

### 10.8 Stacks, Recursion, and Preemption Boundaries

JVM virtual threads use growable stack chunks, but they are not identical to native Go's copyable stacks:

- GoJVM does not implement a Go stack allocator, stack copying, or stack-pointer repair.
- Generated code MUST NOT retain addresses of JVM stack slots. Every location reachable across a call must be a JVM-GC-tracked object.
- `sched.poll` is inserted at least at function entry, loop back edges, and long straight-line regions so CPU-bound goroutines respond to `GOMAXPROCS` changes, global exit, and diagnostic requests.
- The first release does not promise native Go's asynchronous-preemption points; it promises progress at defined polls.
- `StackOverflowError`, excessive recursion, and JVM resource errors are GoJVM fatal conditions by default and MUST NOT be disguised as recoverable Go panics.
- Recursion, large frames, deep defer chains, and large local aggregates require stress tests. The compiler should materialize very large aggregate locals as heap objects to avoid exploding local-slot counts.
- If a goroutine dump cannot safely stop every virtual thread, frames MUST be labeled approximate samples rather than claimed as an exact stop-the-world snapshot.

---

## 11. Channels and `select`

### 11.1 Channel Representation

A JVM-managed `GoChan<T>` contains:

- A unique monotonically increasing channel ID for lock ordering.
- Capacity, current length, and closed state.
- A ring buffer.
- Send-wait queue.
- Receive-wait queue.
- A `ReentrantLock`.
- Optional debugging information.

An element is copied according to Go value semantics when placed in the buffer or handed to a receiver.

### 11.2 Send/receive Semantics

The implementation MUST cover:

- Send and receive on a nil channel block forever.
- Sending on a closed channel panics.
- Closing a nil channel panics.
- Repeated close panics.
- Once a closed channel's buffer is exhausted, receive returns the zero value and `false`.
- Unbuffered send and receive pair directly.
- A blocking operation releases the Go execution permit and reacquires it after wakeup.
- Cancellation or panic unwinding cannot leave ghost waiter nodes in queues.

### 11.3 `select` Algorithm

For each `select`:

1. Evaluate all channel operands and send values exactly once in Go evaluation order, forming required value copies. A receive case's assignment-target expressions are evaluated only after that case wins.
2. Mark nil-channel cases permanently unavailable. `select {}` and an all-nil select without a default block forever.
3. Produce a per-goroutine pseudorandom permutation of case order.
4. Collect non-nil channels, sort by channel ID, and deduplicate.
5. Lock them in stable order to prevent multi-channel deadlock.
6. Probe immediately executable cases.
7. If several are executable, select one according to randomized order.
8. If none is executable and a default exists, select the default.
9. Otherwise create a shared `SelectToken` and register waiters on the participating channels.
10. Release all channel locks and the Go execution permit, then park the virtual thread.
11. Waking contenders use CAS so exactly one case wins.
12. Reacquire relevant locks and remove all losing waiters.
13. Reacquire the Go execution permit and execute the winning case.
14. If the winner is a send to a channel that closed during the race, panic according to Go semantics.

High-concurrency stress tests MUST verify:

- No lost wakeup.
- No double commit.
- No waiter leak.
- No cycle in multi-channel locking.
- Correct close-versus-select races.
- Default never blocks incorrectly.
- Nil cases never enter the lock set.
- `reflect.Select` uses the same commit engine as language `select`.

---

## 12. `defer`, `panic`, `recover`, and `Goexit`

### 12.1 Internal Unwind Mechanism

Private JVM exceptions implement non-local unwinding:

- `GoPanic`: carries the panic chain, original Go interface payload, panic kind, source location, and an optional logical-stack snapshot.
- `GoExit`: implements `runtime.Goexit`.
- `GoFatal`: unrecoverable runtime termination and not an ordinary panic.

These classes are not exposed as user-catchable Java APIs. Nil dereference, bounds failure, divide by zero, send on closed channel, and similar runtime panics construct Go values implementing Go `runtime.Error`. `recover()` returns that Go value or the user's original panic value, never a Java `Throwable`. Panic text is produced by a GoJVM formatter rather than copied from JVM exception messages.

The top-level `runG` for every virtual thread MUST catch internal unwinds. An unrecovered `GoPanic` in a child goroutine does not merely end that thread; it activates a process-wide panic coordinator, prints Go stacks, and terminates the program. `GoExit` ends only the current goroutine. Concurrent unrecovered panics are coordinated through a one-shot fatal state that preserves the first primary diagnostic and necessary supplementary information about simultaneous panics.

### 12.2 `defer`

At execution of `defer f(args...)`, the implementation MUST:

- Immediately evaluate the function value.
- Immediately evaluate arguments in Go order.
- Perform necessary value-argument copies.
- Record the function value, arguments, and source location in a `DeferRecord`.
- Execute records in last-in-first-out order on return.
- Execute them during panic unwinding.
- If the deferred function value is nil, panic when the defer is invoked, not when it is registered.
- Allow a deferred closure to observe and modify named results.

A function with no defer MUST NOT unconditionally allocate a defer-stack object.

### 12.3 Panic Chains

When a deferred function panics again:

- The new panic becomes current.
- The original panic remains available for diagnostics.
- `recover` recovers only the matching current panic.
- After successful recovery, the containing function returns from its normal defer epilogue rather than resuming at the panic point.

### 12.4 Java Exception Boundary

- Generated GoJVM code treats only `GoPanic` and `GoExit` as language unwinds.
- An ordinary `RuntimeException` MUST NOT silently become a Go panic.
- The runtime may convert expected JVM exceptions, such as controlled I/O exceptions, into Go errors.
- An unexpected JVM exception is wrapped as an internal compiler/runtime error and retains its Java cause.
- `OutOfMemoryError`, `StackOverflowError`, and `VirtualMachineError` are unrecoverable by default.
- Fatal JVM control exceptions such as `ThreadDeath` are not caught and disguised as Go panics.

### 12.5 `runtime.Goexit`

- Run all defers of the current goroutine.
- `recover()` returns nil.
- Do not represent it as an ordinary panic.
- Terminate the current virtual thread and update the goroutine registry.
- Preserve behavior relied upon by the Go test framework.

### 12.6 `panic(nil)`, `os.Exit`, and Fatal Exit

- Implement Go 1.26 `panic(nil)` behavior, including the version's `PanicNilError` and `GODEBUG=panicnil` compatibility switch. Java `throw null` and its resulting NPE MUST NOT substitute for this behavior.
- The panic value itself may be a typed nil; interface boxing must retain the dynamic type.
- `os.Exit(code)` terminates the process immediately without running any defer.
- `runtime.throw`, fatal concurrent-map errors, and similar fatal paths are not recoverable.
- The exit coordinator selects a final exit code exactly once and prevents concurrent virtual threads from racing `System.exit` in a way that truncates logs.
- Test mode may convert exit into controlled launcher state, but the production path MUST preserve the no-defer semantics.

---

## 13. Memory Model, Synchronization, and Atomics

### 13.1 Go Memory Model

The goal is to preserve Go's required sequentially consistent observations for data-race-free programs. The implementation MUST NOT assume that ordinary JVM field access is sufficient for the happens-before relationships established by channels, mutexes, `Once`, and atomics.

### 13.2 Atomic Operations

Use `VarHandle` or an equivalent JDK 21 API:

- Loads and stores use volatile/SC semantics required by the Go API.
- CAS uses `compareAndSet`.
- Swap and add use atomic read-modify-write operations.
- Provide separate 32-bit, 64-bit, and pointer/reference atomic paths.
- Preserve unsigned bit patterns.
- JVM field/array representation removes native alignment hazards, but API compatibility tests remain required.
- Do not use `sun.misc.Unsafe`.

### 13.3 Mutex

Go `sync.Mutex` MUST NOT simply expose Java `ReentrantLock` semantics because a Go mutex is non-reentrant. Implement it using AQS or atomic state plus a waiter queue:

- Repeated `Lock` by the same goroutine deadlocks rather than succeeding recursively.
- Unlocking an unlocked mutex produces the compatible fatal/panic diagnostic.
- A mutex is not owner-bound, so one goroutine may unlock a mutex locked by another, consistent with Go.
- Include starvation control and fairness stress tests.
- Release the execution permit while blocked.

### 13.4 Other Synchronization Primitives

Implement directly or wrap under controlled semantics:

- `RWMutex`.
- `Cond`.
- `Once` and `OnceFunc`.
- `WaitGroup`.
- `Pool`.
- Runtime semaphores.
- Timers and tickers.

Every implementation requires happens-before tests and misuse-diagnostic tests.

---

## 14. JVM GC Integration

### 14.1 Governing Rule

GoJVM **fully reuses the JVM garbage collector**. Every Go heap object MUST appear as an ordinary object or array in the JVM-traceable reference graph.

The implementation MUST NOT create:

- Go-owned heap pages or arenas.
- A custom mark/sweep/compact collector.
- Go GC root maps.
- Go write barriers.
- Pointer bitmaps.
- Manual object relocation or address repair.
- A shadow heap.
- Integer-stored object addresses.
- JNI global-reference management.

### 14.2 Allocation

- Structs, closures, maps, channels, slice descriptors, pointer Cells, and similar values use JVM `new` or array allocation.
- The JVM JIT controls small-object escape analysis.
- The GoJVM compiler may perform scalar replacement but MUST NOT rely on object-address stability.
- `new(T)` returns a managed location/object according to T's representation.
- `make` uses type-specific runtime helpers.

### 14.3 `runtime.GC`

Map `runtime.GC` to a request through `System.gc()`, with explicit limitations:

- The JVM may ignore the request.
- Return does not guarantee a native-Go-equivalent full collection cycle.
- Tests MUST NOT depend on an exact pause or sweep point.
- Test mode may use observable reference queues to help test reachability, but MUST NOT implement a second collector.

### 14.4 `runtime.KeepAlive`

Map to `java.lang.ref.Reference.reachabilityFence(obj)` or a strictly equivalent reachability barrier so the JVM/JIT cannot consider the object unreachable too early.

### 14.5 Cleanups and Finalizers

Prefer modern cleanup semantics:

- Map `runtime.AddCleanup` to a controlled `PhantomReference`/`ReferenceQueue` mechanism. `Cleaner` may be used for reachability notification, but user Go functions MUST NOT execute directly on a Cleaner platform thread.
- The Java reference-processing thread only enqueues a `CleanupRecord` into a GoJVM queue. According to load, the runtime starts one or more internal Go virtual threads to execute `cleanup(arg)` concurrently.
- Every cleanup goroutine has a normal `GState`, panic isolation, and execution permit.
- A cleanup callback MUST NOT strongly reference the target. Diagnose `arg == ptr` and obvious closure capture of the target according to upstream restrictions.
- Support multiple cleanups on one allocation and on different projected locations within it.
- Resolve races between `Cleanup.Stop` and enqueue using an atomic state machine.
- Cleanup order is unspecified; cleanups may run concurrently and are not guaranteed before process exit.
- Cleanup-panic handling MUST match the upstream runtime policy under tests and MUST NOT kill a Java Cleaner thread and silently stop all later cleanup work.
- Export queued/executed cleanup counts through GoJVM runtime metrics.

`runtime.SetFinalizer` is difficult to map precisely because of resurrection, dependency ordering, timing, and deprecation of JVM finalization. Therefore the first stable release:

- Diagnoses reachable `runtime.SetFinalizer` calls as an unsupported compatibility feature by default.
- Ports GoJVM standard-library code to `AddCleanup` or explicit resource management.
- MAY offer approximate compatibility through legacy JVM finalization behind an experimental flag, but it is off by default and MUST NOT claim exact equivalence.
- MUST NOT create a custom GC to support `SetFinalizer`.

### 14.6 Memory Statistics

`runtime.MemStats` can only approximate:

- JVM heap used, committed, and maximum.
- GC count and accumulated time.
- Optional class-loading and thread metrics.

For native-Go-specific fields that cannot be represented precisely:

- Return a documented approximation or zero.
- Expose GoJVM-specific metrics under `runtime/metrics`.
- Do not fabricate precise span, mcache, or heap-object statistics.

### 14.7 The `weak` Package

Go 1.26 includes weak-pointer capabilities, including pointers to object fields and array elements. A simple `WeakReference<Object>` is insufficient to preserve Go location identity. Recommended representation:

```text
WeakLocation<T> {
    WeakReference<RootOwner> root
    uint64 rootIdentity
    ProjectionKey canonicalProjection
    TypeID targetType
}
```

Rules:

- When creating a weak pointer, copy the managed pointer's root identity and canonical projection path.
- `Value()` reads the weak root and reconstructs the managed pointer if the root is still alive; otherwise it returns nil.
- Two weak pointers continue to compare by their saved location keys even after the root has been collected.
- Different fields, elements, or nested paths are unequal.
- A strong-reference global table MUST NOT preserve root identity.
- Store a unique ID in the root/wrapper and copy its numeric value into `WeakLocation`.
- A nil weak pointer has stable zero-value semantics.
- Interactions with Cleaner and finalization require targeted ordering tests.
- Packages such as `unique` that depend on weak semantics cannot be marked stable until these rules pass.

---

## 15. Package Initialization and Globals

### 15.1 Do Not Run User Initialization Through Java `<clinit>`

Java class initialization is lazy and has its own locking and deadlock rules, while Go package initialization has an explicit dependency topology. Therefore:

- `<clinit>` may initialize only side-effect-free constants, static tables, and runtime-internal singletons.
- Non-constant user global initialization goes into an explicit `pkg$init` method.
- The linker generates an init-call table from the Go package dependency topology.
- Initialization order among files in one package follows information from the Go front end.
- `init` function order is stable.
- An initialization panic uses the Go panic mechanism and terminates the program.
- First Java class access MUST NOT implicitly reorder Go package initialization.

### 15.2 Entry Sequence

```text
JVM main platform thread
  -> GoRuntime.bootstrapKernel (no user Go code)
  -> register immutable type/class metadata
  -> start initial Go virtual thread for runtime.main
       -> initialize Go runtime state
       -> initialize imported packages topologically
       -> call main.main in the same initial Goroutine
       -> report normal return / panic / Goexit
  -> platform thread joins initial Go virtual thread
  -> exit with Go semantics
```

All package initialization runs serially in the same initial goroutine. Child goroutines started by init may run concurrently, but the linker MUST NOT run different package initializers in parallel. Type tables and pure runtime constants may be preregistered by the launcher; any initialization that can invoke user code or raise a Go panic remains on the initial Go virtual thread.

---

## 16. Reflection and Type Metadata

### 16.1 Type Identity

Every Go type has a stable `TypeID` derived by canonicalizing and hashing:

- Defining package import path.
- Type name.
- Type parameters and instantiation arguments.
- Underlying type structure.
- Method set.
- GoJVM type-format version.

Java `Class` MUST NOT be the sole identity of a Go type because:

- Multiple defined Go types may share one JVM representation.
- Generics and interfaces require additional information.
- Typed nil cannot be classified only by Java class.
- Unnamed composite types also require identity and structure.

### 16.2 Metadata Contents

Generate metadata only for reachable types, including:

- Kind, logical size-like information, and logical alignment-like information.
- Fields, tags, and methods.
- Element type, key type, and length.
- Comparability.
- Hash, equality, copy, and zero operations.
- Interface-implementation relationships.
- Constructors and boxers.
- Debug display name.

APIs such as `Size` and `Align` that ordinarily describe native layout exist only for internal or reflection compatibility while `unsafe` remains unsupported. They MUST NOT imply physical JVM-object size to user code.

### 16.3 Generics

Initial strategy:

- Reuse the Go front end's instantiation and shape decisions wherever practical.
- Apply limited JVM-side specialization for scalar and reference representations.
- Do not use Java generic erasure as the representation of Go type parameters.
- Carry `TypeOps`, methods, and constraint information in generic dictionaries.
- Trim unreachable instantiations at link time.
- Test generic interfaces, nested instantiations, recursive types, method values, and reflection.

---

## 17. Standard-library Porting Strategy

### 17.1 Tiered Plan

**Tier 0: bootstrap core**

- A GoJVM version of `runtime`.
- JVM variants of `internal/goarch`, `internal/goos`, and `internal/cpu`.
- `sync` and `sync/atomic`.
- `errors`, `math`, and `unicode/utf8`.
- Minimal `os` and syscall-replacement layers.

**Tier 1: core development experience**

- `bytes`, `strings`, and `strconv`.
- `fmt`.
- `io` and `bufio`.
- `reflect`.
- `encoding/json`.
- `testing`.
- `time`.

**Tier 2: platform services**

- The commonly used subset of `os`.
- `path/filepath`.
- `net`, `net/http`, and `net/url`.
- Pure-Go portions of `crypto`, plus portions safely supported through Java APIs.
- `os/exec` through `ProcessBuilder`.

### 17.2 `embed` and Resources

`//go:embed` is a first-release core toolchain feature:

- `cmd/go` continues to parse and validate embed patterns.
- Original file bytes enter the package archive under `resources/`.
- The linker creates collision-free JAR resource names from import path and content hash.
- `string` and `[]byte` targets follow copy and immutability rules.
- `embed.FS` uses a GoJVM resource index and never depends on the Java default character encoding.
- JAR compression choices MUST NOT alter Go-observable modtimes or directory traversal behavior.
- Repeated builds use a stable resource order and fixed ZIP metadata.

### 17.3 Unsupported Packages and Capabilities

Expected unsupported or replaced areas include:

- `unsafe`.
- `runtime/cgo`.
- `plugin`.
- Packages such as `debug/elf` may read such files but MUST NOT assume that the running program itself has that format.
- Files that depend on native syscall, epoll, or kqueue behavior.
- Native assembly optimization paths.
- `runtime.LockOSThread`.
- cgo traceback support.
- Native profilers and perf-event interfaces.
- Race, msan, and asan support.

Each standard-library package is labeled in `docs/gojvm/compatibility.md` with one of:

```text
unsupported | planned | partial | passes-unit | passes-differential | stable
```

### 17.4 Files, Networking, and Processes

- Filesystem support uses `java.nio.file`.
- Sockets use `java.net` and NIO.
- Blocking I/O may run directly on a virtual thread, but the runtime wrapper updates `GState` and releases the Go execution permit.
- Deadlines and context cancellation require interruptible or closable underlying resources.
- `os/exec` uses `ProcessBuilder`.
- POSIX permissions, signals, file descriptors, and inode behavior are documented as platform differences.
- If `File.Fd` or `NewFile` is offered, it uses a logical token from a GoJVM handle table. Such a token cannot be passed to a host syscall.
- `SyscallConn`, arbitrary file-descriptor ioctls, and native calls from `golang.org/x/sys` are unsupported initially.
- The runtime MUST NOT pretend that a logical descriptor is usable with arbitrary native syscalls.

### 17.5 Time, Timers, and Signals

- Wall-clock time uses `Instant` or `System.currentTimeMillis`; monotonic time uses `System.nanoTime`.
- The monotonic component of `time.Time` MUST NOT be derived from wall-clock time.
- Timers and tickers use one unified GoJVM runtime timer queue rather than permanently consuming one platform thread per timer.
- A timer wakeup is a possible external event and must be considered by deadlock detection.
- `Sleep` and timer-channel blocking release the execution permit.
- System timezone data primarily uses JDK `ZoneId`, but historical transitions and local-time behavior require validation against Go standard-library tests.
- `os/signal` is initially platform-limited. Non-standard `sun.misc.Signal` MUST NOT become a stable dependency.
- No false compatibility is offered for `SIGPIPE`, Unix job control, or signal masks across fork/exec.

### 17.6 Runtime Observability APIs

- `runtime.NumGoroutine` reads the GoJVM registry.
- `runtime.Gosched` yields the execution permit and the virtual thread.
- `runtime.Stack` and `Callers` use the logical-frame registry, source line tables, and, where necessary, JVM `StackWalker`.
- Java runtime/helper frames are hidden from Go users by default.
- pprof and trace are only partially supported initially; documentation distinguishes sampled metrics from exact events.
- JFR may serve as a diagnostic backend but MUST NOT become an execution-semantic dependency.

### 17.7 GC and `runtime/debug` Compatibility Differences

JVM heap parameters are normally fixed at process startup. These APIs can only be approximated or rejected:

- `debug.SetGCPercent`: a GoJVM hint for GC-request frequency, not control over the JVM's actual trigger percentage.
- `debug.SetMemoryLimit`: records a logical soft limit and may trigger active GC or allocation checks, but cannot raise JVM `-Xmx` and cannot guarantee collection at the exact value.
- `debug.FreeOSMemory`: requests `System.gc()` and does not guarantee that committed heap is returned to the OS.
- Heap dump: if implemented, emits JVM HPROF or a GoJVM logical-heap summary, never a file falsely presented as a native Go heap profile.
- `GOGC` and `GOMEMLIMIT`: parsed as hints or soft limits; documentation states that `-Xms` and `-Xmx` remain the JVM hard heap boundaries.
- Visibility of stack objects and JIT scalar-replaced objects is determined by the JVM.

---

## 18. Rejection of cgo, `unsafe`, Assembly, and Native Capabilities

### 18.1 cgo

Reject cgo at multiple layers so it cannot slip through:

- `cmd/go`: force `CGO_ENABLED=0` for `GOOS=jvm`.
- Package loading: reject `import "C"` immediately.
- Build constraints: exclude C, C++, Objective-C, and Fortran files from the target.
- Compiler: reject cgo-generated markers or ABI symbols.
- Linker: reject native-method or JNI dependencies.
- Class-file audit: reject `ACC_NATIVE`.

### 18.2 `unsafe`

- Importing `unsafe` from a user package is a compile-time error.
- The official front end may internally use unsafe-related type nodes; the GoJVM branch must distinguish compiler-internal concepts from reachable unsafe operations in the target program.
- `//go:linkname` is rejected by default.
- Only an explicit, maintained allowlist for GoJVM standard-library internals is permitted, and it must be audited regularly.
- `unsafe.Pointer`, `StringData`, `SliceData`, `Add`, and related operations are unsupported.
- Reflection or linker tricks MUST NOT bypass the prohibition.

### 18.3 Assembly and Native Files

- If a package contains `.s` or `.S` files and no JVM replacement file, report the package and filename.
- Native-ABI directives such as `//go:noescape` and `//go:uintptrescapes` MUST NOT silently take effect.
- `//go:nosplit` has no native-stack meaning on the JVM; reject user use, or ignore it only as an explicitly warned internal marker.
- Do not produce or consume ELF, Mach-O, or PE object code.

---

## 19. Errors, Stacks, and Debug Information

### 19.1 Go-style Stacks

Emit source line tables and maintain a JVM-symbol-to-Go-symbol mapping. Panic output should:

- Show the goroutine ID and state.
- Show Go package path, function, file, and line.
- Hide GoJVM runtime trampolines unless verbose mode is enabled.
- Optionally show a Java cause as a “runtime internal cause.”
- Support all-goroutine dumps.

### 19.2 Debug Builds

Provide flags of this form:

```text
-gcflags=all=-gojvm-mir-dump=...
-gcflags=all=-gojvm-verify
-gcflags=all=-gojvm-no-opt
-ldflags=-gojvm-keep-classes
```

Final flag names should follow Go toolchain conventions and be registered centrally.

### 19.3 Stable MIR Dumps

- Text output uses stable ordering.
- Nondeterministic addresses are hidden.
- Every pass can be dumped.
- Golden files remain readable.
- Diagnostics include function, block, and instruction IDs.
- Failing cases can be minimized.

---

## 20. Test Strategy

### 20.1 Test Pyramid

1. **Unit tests**
   - MIR construction and verification.
   - Type representation.
   - Name encoding.
   - Class-file writing.
   - `StackMapTable`.
   - Map hash/equality.
   - Channel state machines.

2. **Compiler integration tests**
   - Go source to MIR.
   - MIR to class file.
   - Package archive.
   - Linker and JAR.
   - `go build`, `go run`, and `go test`.

3. **Differential tests**
   - Run the same pure-Go, platform-independent program under native Go and GoJVM.
   - Compare stdout, stderr, exit code, and structured results.
   - For concurrency tests, compare the allowed result set rather than exact timing.

4. **Stress tests**
   - 100,000 to 1,000,000 short-lived goroutines.
   - Channels and `select`.
   - `sync` and atomics.
   - GC pressure.
   - Heavy interface and reflection usage.
   - Large methods and constants.

5. **Fuzz tests**
   - MIR parser and verifier.
   - Class-file writer.
   - Small-expression differential behavior.
   - Map keys.
   - UTF-8 and strings.
   - `select` state machines.

### 20.2 Required Semantic Tests

- Evaluation order.
- Go 1.26 `new(expr)` evaluation, initialization, and address behavior.
- Self-referential generic constraints do not recurse metadata generation to stack overflow.
- Nil slice versus empty slice.
- Nil map.
- Typed-nil interface.
- Calling a pointer-receiver method through an interface containing a typed-nil pointer.
- Pointer- and value-receiver method values and method expressions on nil pointers.
- Struct/array copies do not alias.
- Closure capture by reference.
- Lifetime of address-taken locals.
- After `&s.field`, whole-value replacement of `s` still observes the same storage.
- Pointer behavior after `&array[i]`, whole-array replacement, and slice `append` changing backing storage.
- Unsigned comparison, division, and remainder.
- Large shift counts.
- Floating NaN and positive/negative zero as map keys.
- Arbitrary bytes in strings.
- Rune iteration.
- `append` backing reuse.
- Overlapping `copy`.
- Immediate argument evaluation for `defer`.
- Named results plus `defer`.
- Panic during panic.
- Directness of `recover`.
- `Goexit`.
- Send/close/select races.
- Permanent blocking on nil channels.
- Dynamic `GOMAXPROCS` adjustment.
- Termination of other goroutines when main returns.
- `KeepAlive` under JVM GC.
- Cleaner does not strongly retain the target.
- Package initialization order.
- Java class initialization does not change Go initialization order.

### 20.3 Negative Tests

Diagnostics MUST be asserted for:

- `import "C"`.
- Importing `unsafe`.
- Assembly files.
- `CGO_ENABLED=1`.
- Unsupported build modes.
- `runtime.LockOSThread`.
- `SetFinalizer` in default mode.
- Non-allowlisted `//go:linkname`.
- Array lengths beyond JVM limits.
- An oversized method that cannot safely be split.
- Native or JNI class dependencies.

### 20.4 JDK Matrix

| JDK | Role | Required coverage |
|---|---|---|
| 21 | Minimum runtime | Full correctness, verification, and baseline stress |
| 25 | Current long-term support line | Full correctness and performance regression |
| 26 | Forward compatibility | Smoke tests and verifier tests |

All JDKs run the same target-version-65 JAR.

### 20.5 Performance Baseline

The first release is not expected to outperform native Go, but it MUST track:

- Startup time.
- JAR size.
- Compilation time.
- Class count.
- Virtual-thread creation and switching.
- Channel throughput and latency.
- Map operations.
- Interface dispatch.
- Allocation rate.
- GC count and peak heap.
- Large-method splitting overhead.

Performance optimization MUST NOT bypass semantic tests.

---

## 21. CI and Quality Gates

Every pull request runs at least:

1. Build of the GoJVM compiler itself.
2. A native-Go regression subset.
3. MIR unit and golden tests.
4. Class-file unit tests.
5. End-to-end Hello World.
6. `java -Xverify:all`.
7. JDK 21/25 matrix.
8. cgo and `unsafe` negative tests.
9. Deterministic-build test.
10. Static scan for `Unsafe`, JNI, and native methods.
11. Check that `docs/gojvm/progress.md` is updated.
12. Formatting and license checks.

The main branch MUST always build. Incomplete backend capabilities are isolated by feature flags, explicit diagnostics, or a documented test-skip list; invalid partially generated bytecode MUST NOT be committed.

---

## 22. Git, Commit, Push, and Agent Work Protocol

This section is mandatory for the implementation agent.

### 22.1 Branches

- Primary development branch: `feature/gojvm-backend`.
- Large milestones may use short-lived branches named `feature/gojvm-mN-topic`.
- Published branches MUST NOT be force-pushed.
- Pushed history MUST NOT be rewritten.
- Upstream synchronization uses a separate pull request so it is not mixed with feature work.

### 22.2 Regular Commit and Push Requirements

The agent MUST:

- Commit immediately after completing each independently explainable and testable subtask.
- Even during a long task, create a recoverable checkpoint commit at least every 20–30 minutes.
- Push after every one or two commits.
- Push at every milestone, at the end of every work session, and before and after risky refactoring.
- Never leave a day's work only in an uncommitted working tree.
- Never wait until “everything is complete” before the first push.
- Before pushing, run the smallest test set appropriate to the change.
- Record the commit hash after each commit and confirm after each push that the remote contains it.

Recommended commit format:

```text
gojvm(<area>): <imperative summary>

<why and design notes>

Tests: <exact commands and results>
Refs: <ADR/design section>
```

Example:

```text
gojvm(mir): add typed locals and CFG verifier

Rejects type-changing local merges and enforces empty operand-stack
boundaries required by the JVM emitter.

Tests: go test ./src/cmd/compile/internal/gojvm/mir
Refs: docs/gojvm/design.md §6
```

### 22.3 Progress File

Before every push, update `docs/gojvm/progress.md`:

```markdown
# GoJVM Progress

- Last updated:
- Branch:
- Last pushed commit:
- Last known-good commit:
- Current milestone:

## Completed
## In progress
## Next three tasks
## Tests run
## Known failures
## Blockers / decisions needed
## Compatibility changes
```

The file MUST let another agent continue without access to chat history.

### 22.4 Work in Progress and Failure Recovery

If a session ends while changes do not meet the quality bar for the main feature branch:

1. Create `checkpoint/YYYYMMDD-topic`.
2. Commit with a subject beginning `WIP checkpoint:`.
3. List failing tests and remaining work in the commit body.
4. Push the checkpoint branch.
5. Keep `feature/gojvm-backend` at the last known-good commit.
6. Record the checkpoint branch and hash in `progress.md`.

If a push fails:

- Do not delete or reset the local commit.
- Record the local commit hash.
- Report the remote, branch, and original error.
- Retry after repairing authentication or network access.
- Do not use copied working-tree files as a substitute for Git preservation.

### 22.5 Commit Granularity

Good commits include:

- Target registration.
- MIR type system.
- MIR verifier.
- Class-file constant pool.
- `StackMapTable` support.
- One well-defined runtime primitive.
- One coherent differential-test group.

Bad commits include:

- “Implement backend” while changing hundreds of files.
- Simultaneously refactoring upstream code, adding the runtime, and changing tests.
- A huge untested mechanical generation.
- Mixing formatting, renaming, and behavior changes.
- Committing generated artifacts, JDK distributions, or large binaries.

### 22.6 Architecture Decision Records

A new ADR is mandatory when deciding or changing:

- Value-type representation.
- Interface-dispatch design.
- Goroutine-to-virtual-thread mapping.
- Class-file generation library or approach.
- Finalizer compatibility policy.
- Large-method splitting scheme.
- Java interoperability.
- Minimum JDK.
- Upstream compiler branch point.

ADR template: context, decision, alternatives, consequences, tests, and migration plan.

---

## 23. Milestones and Acceptance Criteria

### M0: Fork, Target Skeleton, and CI

Deliverables:

- Go 1.26.4 fork.
- Registration of `GOOS=jvm GOARCH=jvm`.
- Rejection of cgo, `unsafe`, and assembly.
- JDK 21/25 CI.
- Design document, ADRs, and progress file.
- No regression in native Go paths.

Acceptance: the target is recognized by `go env` and `go list`; unsupported inputs produce clear errors; the first commit/tag is pushed.

### M1: Class-file Writer and Scalar Hello World

Deliverables:

- Class-file v65 writer.
- Constant pool, `Code`, and `StackMapTable`.
- JAR linker.
- Runtime bootstrap.
- Minimal `println` path.

Acceptance: `GOOS=jvm go build` emits a JAR that passes `java -Xverify:all -jar` and runs on both JDK 21 and 25.

### M2: Complete Basic MIR and Control Flow

Deliverables:

- Typed blocks and locals.
- Arithmetic, comparisons, branches, and switch.
- Nil, bounds, divide, and shift checks.
- Multiple return values.
- Large-method preflight checks.

Acceptance: scalar and control-flow differential suites pass; MIR fuzzing does not crash.

### M3: Aggregates, Pointers, Strings, Slices, and Closures

Deliverables:

- Struct/array value copying.
- Managed pointers.
- Byte-backed strings.
- Slices.
- Function values and closures.

Acceptance: differential tests for aliasing, arbitrary string bytes, `append`/`copy`, and address capture pass.

### M4: Virtual Threads, Scheduling, `defer`/`panic`/`recover`

Deliverables:

- Goroutine-to-virtual-thread mapping.
- `GOMAXPROCS` execution permits.
- Scheduler polls.
- `defer`, `panic`, `recover`, and `Goexit`.
- Go stack formatting.

Acceptance: 100,000-goroutine stress; recover-directness tests; main-lifecycle tests; JDK 21 pinning tests.

### M5: Channels, `select`, `sync`, and Atomics

Deliverables:

- Buffered and unbuffered channels.
- `select`.
- `Mutex`, `RWMutex`, `Cond`, `Once`, and `WaitGroup`.
- VarHandle atomics.

Acceptance: high-concurrency stress shows no deadlock or lost wakeup; memory-model litmus tests pass.

### M6: Maps, Interfaces, Generics, and Reflection

Deliverables:

- Go map implementation.
- Typed nil.
- Itab adapters.
- Type registry.
- Generic dictionaries/specialization.
- Core reflection.

Acceptance: differential tests pass for float and struct keys, interface equality, generic interfaces, and reflection.

### M7: Standard Library and `go test`

Deliverables:

- Tier 0 and Tier 1 standard library.
- `testing`.
- Common file, time, and networking capabilities.
- `go test ./...` support.
- Compatibility dashboard.

Acceptance: allowlisted standard-library package tests pass; an example HTTP service runs.

### M8: Hardening, Performance, and Release

Deliverables:

- Large-method and class sharding.
- Deterministic builds.
- Performance baseline.
- Error diagnostics.
- Documentation and migration guide.
- Release candidate.

Acceptance: full JDK 21/25 matrix; no class-verifier error; repeated builds have identical hashes; known-difference list is complete.

---

## 24. High-risk Pitfall Checklist

Each item MUST have a regression test:

1. **Representing a Go string as Java String:** breaks arbitrary bytes and indexing semantics.
2. **Treating Go structs/arrays as ordinary Java references:** introduces accidental aliasing after assignment.
3. **Comparing slices/maps/functions with Java `equals`:** violates Go comparability.
4. **Losing typed nil:** breaks interface nil tests.
5. **Using `HashMap` directly:** mishandles NaN, signed zero, interface keys, and composite keys.
6. **Using JVM shift-count masking as Go semantics.**
7. **Using signed JVM operations for Go unsigned division or comparison.**
8. **Using JVM `NullPointerException` as a Go nil panic:** wrong location and ordering.
9. **Using `ArrayIndexOutOfBoundsException` as a Go bounds panic:** wrong text, ordering, and nil-slice behavior.
10. **Using Java object identity for all Go pointer equality:** wrong for field and element locations.
11. **Forwarding a recover token through ordinary calls:** allows invalid recovery.
12. **Letting the platform main thread exit early because virtual threads are daemon threads.**
13. **Equating JVM scheduler parallelism with `GOMAXPROCS`.**
14. **JDK 21 `synchronized` pinning.**
15. **Lost wakeup in races between select registration and close.**
16. **Deadlock from unstable multi-channel lock ordering.**
17. **Putting user init in `<clinit>`:** changes ordering and introduces Java initialization deadlocks.
18. **Relying on the JVM to infer `StackMapTable`.**
19. **Discovering the 64 KiB method or constant-pool limit only at final link.**
20. **Forgetting that JVM arrays use `int` lengths while Go `int` is 64-bit.**
21. **Allowing Java exceptions to be recovered as Go panics.**
22. **A Cleaner callback strongly retaining its target and never running.**
23. **Building a custom GC for `SetFinalizer`:** violates project scope and introduces extreme risk.
24. **Retaining user objects in `ThreadLocal`:** causes leaks.
25. **Using only Java `Class` for reflection:** loses defined Go type identity.
26. **Importing native Go `walk` or SSA assumptions about addresses and frames.**
27. **Allowing internal standard-library `linkname` to bypass the `unsafe` prohibition.**
28. **Nondeterministic synthetic names:** breaks caching and reproducibility.
29. **Splitting a method across a `defer`/unwind region:** breaks semantics.
30. **An optimization changing panic order or side-effect order.**

---

## 25. Recommended Agent Implementation Order

Proceed in vertical slices rather than implementing the entire MIR before the runtime:

1. Establish the fork, CI, branch, and progress file; commit and push.
2. Register the JVM target and reject cgo/`unsafe`; commit and push.
3. Implement the minimal class-file writer and hand-generate a `main`; commit and push.
4. Lower one scalar Go function through MIR to bytecode; commit and push.
5. Complete package archives, linker, and JAR path; commit and push.
6. For each new MIR instruction category, add verifier rules, emitter support, and end-to-end tests at the same time.
7. For each new Go value category, add tests for copy, zero value, equality, and reflection metadata at the same time.
8. Implement goroutine lifecycle before channels.
9. Test `defer`/`panic`/`recover` first in a small reference interpreter, then emit bytecode.
10. Port the standard library package by package according to tiers; do not conceal errors behind broad temporary stubs.
11. Create and push an annotated tag at each milestone: `gojvm-m0`, `gojvm-m1`, and so on.

If evidence at any stage requires a major representation change, write an ADR and a minimal prototype before allowing the change to spread through the main implementation.

---

## 26. Definition of Done

A feature is complete only when all applicable conditions hold:

- Design and compatibility are documented.
- MIR or runtime semantics have unit tests.
- At least one end-to-end test exists.
- A native-Go differential test exists where applicable.
- JDK 21 and 25 pass where applicable.
- `java -Xverify:all` passes.
- There is no cgo, `unsafe`, JNI, or native-method dependency.
- Error paths have explicit diagnostics.
- `docs/gojvm/progress.md` is updated.
- An atomic commit exists.
- The commit is pushed and its remote hash is confirmed.
- The main feature branch remains buildable.

---

## 27. Recommended Initial Architecture Decision Records

- ADR-0001: Pin the fork to Go 1.26.4.
- ADR-0002: Branch before native escape analysis, `walk`, and SSA.
- ADR-0003: GoJVM-MIR uses typed blocks and explicit locals.
- ADR-0004: Empty JVM operand stack at block boundaries.
- ADR-0005: Go strings use a byte-backed representation.
- ADR-0006: Structs and arrays use explicit value copies.
- ADR-0007: Pointers use managed locations.
- ADR-0008: One JVM virtual thread per goroutine.
- ADR-0009: `GOMAXPROCS` uses execution permits.
- ADR-0010: Channels and `select` use a GoJVM runtime implementation.
- ADR-0011: JVM GC only; no Go GC implementation.
- ADR-0012: `SetFinalizer` unsupported by default; `AddCleanup` mapped through Cleaner/reference queues.
- ADR-0013: Pure-Go class-file writer.
- ADR-0014: Interfaces use `GoType + data + typed itab adapter`.
- ADR-0015: Explicit package initialization rather than Java `<clinit>`.
- ADR-0016: JAR is the initial executable artifact; class-file target is 65.

---

## 28. References

Before modifying a related subsystem, the implementation agent should read the corresponding official material:

- Go 1.26 Release Notes: <https://go.dev/doc/go1.26>
- Go Release History: <https://go.dev/doc/devel/release>
- Go Language Specification: <https://go.dev/ref/spec>
- Go Memory Model: <https://go.dev/ref/mem>
- Go compiler source (`cmd/compile/internal/gc/main.go`): <https://go.dev/src/cmd/compile/internal/gc/main.go>
- Go compiler IR: <https://go.dev/src/cmd/compile/internal/ir/>
- Go `runtime.KeepAlive` / `SetFinalizer` documentation: <https://pkg.go.dev/runtime>
- Go `weak` package: <https://pkg.go.dev/weak>
- JEP 444, Virtual Threads: <https://openjdk.org/jeps/444>
- JEP 491, Synchronize Virtual Threads without Pinning: <https://openjdk.org/jeps/491>
- Java 21 Thread API: <https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/lang/Thread.html>
- Java 21 `Reference.reachabilityFence`: <https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/lang/ref/Reference.html>
- Java 21 VarHandle API: <https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/lang/invoke/VarHandle.html>
- JVM Specification, Class File Format: <https://docs.oracle.com/javase/specs/jvms/se21/html/jvms-4.html>
- JVM Specification, Instruction Set: <https://docs.oracle.com/javase/specs/jvms/se21/html/jvms-6.html>

---

## 29. Final Implementation Command Summary for the Agent

> These constraints are mandatory, not optional recommendations.

```text
1. Fork and pin upstream Go 1.26.4.
2. Work on feature/gojvm-backend.
3. Implement GOOS=jvm / GOARCH=jvm with CGO_ENABLED=0 only.
4. Reject import "C", unsafe, assembly, JNI/native methods, and unsupported build modes.
5. Branch after target-independent front-end transformations and before native escape/walk/SSA.
6. Implement GoJVM-MIR exactly around typed blocks, explicit locals/control/calls/checks/object operations.
7. Do not implement register allocation, native frame layout, raw address arithmetic, or a Go GC.
8. Allocate all heap state as ordinary JVM objects/arrays and rely on JVM GC.
9. Map each goroutine to a JVM virtual thread; implement Go scheduling semantics around it.
10. Target classfile version 65 and test on OpenJDK 21 and 25.
11. Run bytecode verification for every end-to-end artifact.
12. Commit after every coherent subtask and at least every 20–30 minutes.
13. Push every 1–2 commits, at every milestone, and before ending a work session.
14. Update docs/gojvm/progress.md before every push.
15. Never force-push a published branch; preserve WIP on a pushed checkpoint branch.
16. Keep the main feature branch green and record exact tests in every commit.
```
