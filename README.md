# GoJVM

This repository tracks the Go-to-JVM backend implementation effort defined in
[`goal.md`](goal.md). It starts as a Go 1.26.4 fork target for `GOOS=jvm`, `GOARCH=jvm`
with explicit milestones, compatibility reporting, and a documented acceptance path.

## Current state

- Source bootstrap and target registration scaffolding are not complete yet.
- The repository currently contains the implementation plan and milestone progress
  infrastructure.
- Follow-up work is focused on M0 (target skeleton, diagnostics, and build path)
  before full compiler backend work.

## Milestones

1. M0: Fork, target skeleton, and CI.
2. M1: Class-file writer and scalar hello-world execution path.
3. M2: Core MIR and control-flow lowering.
4. M3+: Runtime, concurrency, channels, interfaces, maps, reflection, and tests.

See:

- [`docs/gojvm/progress.md`](docs/gojvm/progress.md) for status.
- [`docs/gojvm/compatibility.md`](docs/gojvm/compatibility.md) for tiered compatibility.
- [`docs/gojvm/adr`](docs/gojvm/adr) for architecture decisions.

## Development workflow

Use the helper scripts:

- `scripts/init-gojvm-workspace.sh` to initialize a local Go 1.26.4 fork and branch.
- `scripts/milestone-check.sh` to validate basic repository invariants.

