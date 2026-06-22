# ADR-0002: Branch before native escape/walk/SSA

## Context

GoJVM requires semantic lowering that depends on Unified IR and generic/type-check
information, but not on native stack layout, ABI wrapper generation, escape-analysis assumptions,
walk, or SSA native instructions.

## Decision

The compiler path will branch to GoJVM frontend-only compilation right after loop-variable
capture and wrapper verification, before `pkginit`, escape analysis, `reflectdata`,
`walk`, and SSA lowering.

## Alternatives Considered

- Extending native compile pipeline and translating after SSA.
- Branching earlier and reproducing additional front-end work by hand.

## Consequences

- Preserves front-end richness while avoiding native backend coupling.
- Requires explicit frontend contract tests to guard branch-point drift on rebase.

## Tests

- Small contract fixtures for generics, closures, interfaces, defer evaluation order,
  range semantics, multiple return values, and init order.

## Migration Plan

- Introduce a dedicated `gojvm` branch path with compile-time assertion tests and
  a regression fixture to confirm expected control flow.
