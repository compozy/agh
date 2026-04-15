## SMOKE-007: Lifecycle State Machine Rejects Invalid Transition

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-15

---

### Objective

Verify the lifecycle state machine rejects at least one known invalid transition (e.g., error→ready).

### Preconditions

- [ ] `internal/bridges` lifecycle package available

### Test Steps

1. **Set instance status to `error`**
   - **Expected:** Status set successfully

2. **Attempt transition from `error` to `ready`**
   - **Expected:** Transition rejected with validation error

3. **Attempt valid transition from `error` to `starting`**
   - **Expected:** Transition accepted, status updated to `starting`

### Related Test Cases

- TC-FUNC-005
