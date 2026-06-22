# GoJVM Compatibility Matrix

Compatibility is reported by tiers, not binary statements.

## Language

- Goal: Go 1.26 static semantics and typing, excluding `unsafe` and cgo behavior.
- Planned support (phase-aligned):
  - Generics, interfaces, closures, slices, maps, channels, defer/panic/recover: targeted in M3+.
  - `unsafe`, assembly (`.s`/`.S`), and cgo imports: rejected with explicit diagnostics.

## Runtime

- Goroutines and scheduling are mapped to JVM virtual threads.
- `Goexit`, `Goexit`/`panic`/`recover`, channel blocking semantics, and select behavior
  require M4/M5 features not yet implemented.

## Standard Library

- Tiering is required to keep behavior explicit. A package allowlist is required before
  enabling broad `go test ./...` coverage.

## Toolchain

- Primary goals: preserve Go frontend behavior and maintain `go build`, `go run`, `go test`
  parity for supported package subsets on the target JAR model.

## Binary

- Output artifact is a JAR containing JVM bytecode and runtime classes.
- No native Go object or ELF/Mach-O/PE compatibility expectations.

## Known gaps

- Java/JVM interop and JNI are intentionally out of scope in the first release.
- No stable performance equivalence claims against native Go.
