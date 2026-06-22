# ADR-0013: Pure-Go class-file writer

## Context

Build and bootstrap must not depend on Maven/Gradle. Bytecode emission must remain fully
auditable and deterministic.

## Decision

GoJVM will implement class-file and JAR construction in Go using a pure-Go serializer.

## Alternatives Considered

- Delegating to external Java bytecode libraries.
- Using a JVM-based toolchain to generate bytecode as part of the compiler.

## Consequences

- Removes extra build dependencies from the compiler bootstrap path.
- Enables strict deterministic-jar checks and straightforward versioned format governance.

## Tests

- Class-file golden tests
- Verifier smoke tests (`java -Xverify:all`)

## Migration Plan

- Start with minimal Code + constant pool support for scalar execution; expand with
  StackMapTable and attributes in iterative milestones.
