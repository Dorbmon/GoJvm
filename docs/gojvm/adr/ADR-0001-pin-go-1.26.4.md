# ADR-0001: Pin fork target to Go 1.26.4

## Context

A stable baseline is required to avoid semantic drift during backend development and to
compare against a fixed Go language/runtime contract.

## Decision

GoJVM will fork from upstream Go tag `go1.26.4` and keep this pin during M0–M8.

## Alternatives Considered

- Tracking `master` directly to reduce rebasing effort.
- Pinning to an earlier version to lower upstream churn.

## Consequences

- Rebase work is bounded and deterministic.
- Required bootstrap and compiler behavior are tied to 1.26.4 language contracts.

## Tests

- Rebase tests must rerun frontend contract checks before code changes.
- Upstream identity checks in `scripts/init-gojvm-workspace.sh`.

## Migration Plan

- Future upgrades require explicit release planning and a full frontend integration
  retest suite before any backend changes.
