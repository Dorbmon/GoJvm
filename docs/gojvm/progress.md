# GoJVM Progress

- Last updated: 2026-06-22
- Branch: feature/gojvm-backend
- Last pushed commit: 04aafbe
- Last known-good commit: 04aafbe
- Current milestone: M0 (Fork, Target Skeleton, and CI)

## Completed

- Repository baseline documentation added (`goal.md`).
- Progress tracking structure and baseline implementation scripts added.
- Added milestone-oriented documentation and ADR index.
- Added unsupported-feature/build-mode policy package shared at `src/gojvm/target`.
- Added compile-side checker package (`src/cmd/compile/internal/gojvm/checks`) with CLI (`gojvmcheck`) and tests.
- Wired policy/validation through `CompilePackage` and `cmd/go` scaffolding helpers.
- Added runtime bootstrap build path (`scripts/build-gojvm-runtime.sh`) and Makefile/CI hooks to build and verify `gojvm-rt.jar`.

## In progress

- Creating the `GOOS=jvm` target registration path in a minimal fork.
- Wiring explicit rejection for unsupported inputs (`cgo`, `unsafe`, assembly files) end-to-end.
- Integrating target registration and branch-point integration into an actual Go 1.26.4 checkout.

## Next three tasks

1. Materialize the Go 1.26.4 fork and branch policy with reproducible setup script.
2. Materialize the Go 1.26.4 fork plus target registration edits.
3. Replace placeholder back-end/compiler packages with functional classfile and linker flow.

## Tests run

- `./scripts/milestone-check.sh` (initial scaffold validation)
- `bash -n` on all added shell scripts
- `go test ./...` (validation tests now cover unsupported feature checks)
- `bash scripts/check-unsupported-features.sh .` (go-based enforcement path)
- `./scripts/build-gojvm-runtime.sh` (runtime bootstrap build validation)

## Known failures

- No Go/JVM compiler or runtime integration exists yet.
- No bytecode emitter or linker is implemented yet.

## Blockers / decisions needed

- Upstream Go toolchain snapshot availability in the working environment.
- Storage budget and timeline for complete source-tree fork.

## Compatibility changes

- No compatibility decisions have been activated in code yet.
- The target remains design/documentation-only until M0 milestone target registration is added.
