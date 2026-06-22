# GoJVM Progress

- Last updated: 2026-06-22
- Branch: feature/gojvm-backend
- Last pushed commit: not yet pushed
- Last known-good commit: not yet recorded
- Current milestone: M0 (Fork, Target Skeleton, and CI)

## Completed

- Repository baseline documentation added (`goal.md`).
- Progress tracking structure and baseline implementation scripts added.
- Added milestone-oriented documentation and ADR index.

## In progress

- Creating the `GOOS=jvm` target registration path in a minimal fork.
- Wiring explicit rejection for unsupported inputs (`cgo`, `unsafe`, assembly files).
- Bootstrapping runtime build and JAR packaging checks.

## Next three tasks

1. Materialize the Go 1.26.4 fork and branch policy with reproducible setup script.
2. Add explicit diagnostics for unsupported features and build modes.
3. Add CI checks for docs, scripts, and helper command outputs.

## Tests run

- `./scripts/milestone-check.sh` (initial scaffold validation)
- `bash -n` on all added shell scripts
- `go test ./...` (scaffold package compile check)

## Known failures

- No Go/JVM compiler or runtime integration exists yet.
- No bytecode emitter or linker is implemented yet.

## Blockers / decisions needed

- Upstream Go toolchain snapshot availability in the working environment.
- Storage budget and timeline for complete source-tree fork.

## Compatibility changes

- No compatibility decisions have been activated in code yet.
- The target remains design/documentation-only until M0 milestone target registration is added.
