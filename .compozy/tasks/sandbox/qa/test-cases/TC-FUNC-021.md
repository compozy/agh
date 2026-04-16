## TC-FUNC-021: Reconciliation reattaches recoverable session

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 07

---

### Objective

Verify that daemon restart reconciliation attempts to reattach to a remote sandbox for sessions in non-terminal states, using persisted `EnvironmentID`, `InstanceID`, and `ProviderState`.

---

### Preconditions

- [x] Session with remote backend in non-terminal state (e.g., `running`) persisted in store
- [x] Mock provider that accepts reattach

---

### Test Steps

1. **Boot daemon with a persisted running remote session**
   - **Expected:** Reconciliation calls `Provider.Prepare()` with `EnvironmentID`, `InstanceID`, and `ProviderState` from persisted metadata

2. **Verify successful reattach**
   - Input: Provider returns success
   - **Expected:** Session environment metadata updated with fresh provider state, session remains in its prior state
