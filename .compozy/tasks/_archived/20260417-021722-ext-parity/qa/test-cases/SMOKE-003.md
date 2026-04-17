# SMOKE-003: Reconcile Driver Boot Ordering

**Priority:** P0
**Type:** Smoke
**Package:** internal/resources
**Related Tasks:** 03

## Objective

Validate that the reconcile driver executes registered projectors in correct topological order based on their DependsOn() edges during RunBoot(). This ensures that projectors which depend on the output of other projectors always see a consistent, already-reconciled state when they run.

## Preconditions

- A reconcile driver instance with no prior state
- Three or more test projectors registered with explicit dependency edges:
  - Projector A: no dependencies
  - Projector B: depends on A
  - Projector C: depends on A and B
- Each projector records its execution timestamp or sequence index for verification

## Test Steps

1. **Register Projector A** with kind="test.base" and no DependsOn() edges.
   **Expected:** Registration succeeds with no error.

2. **Register Projector B** with kind="test.middle" and DependsOn()=["test.base"].
   **Expected:** Registration succeeds with no error.

3. **Register Projector C** with kind="test.leaf" and DependsOn()=["test.base", "test.middle"].
   **Expected:** Registration succeeds with no error.

4. **Call RunBoot()** on the reconcile driver.
   **Expected:** All three projectors execute successfully with no errors.

5. **Verify execution order** from the recorded sequence.
   **Expected:** A executed before B, and B executed before C. The order satisfies the topological constraint: A < B < C.

6. **Register a fourth Projector D** with kind="test.sibling" and DependsOn()=["test.base"] (same dependency as B, but independent of B and C).
   **Expected:** Registration succeeds.

7. **Call RunBoot() again** (or a fresh driver with all four).
   **Expected:** A runs first. B and D may run in any order relative to each other (both only depend on A). C runs after both A and B. No projector runs before its dependencies complete.

## Edge Cases

- Registering a projector that depends on a non-existent kind returns an error at registration or at RunBoot() time
- A circular dependency (A depends on B, B depends on A) is detected and returns an error, not an infinite loop
- A projector that returns an error during boot halts the boot sequence and propagates the error
- RunBoot() with zero registered projectors completes successfully (no-op)
- RunBoot() with a single projector (no dependencies) executes it exactly once
- A projector registered twice for the same kind returns an error or replaces the previous registration
- Projectors at the same topological level execute concurrently if the driver supports parallel boot (verify via timing or explicit concurrency test)
