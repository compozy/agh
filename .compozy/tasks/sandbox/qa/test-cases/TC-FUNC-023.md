## TC-FUNC-023: Reconciliation finds sandbox by agh_environment_id

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 07

---

### Objective

Verify that when local metadata has `EnvironmentID` but no `InstanceID` (partial create), reconciliation uses the `Finder` interface to discover remote sandboxes by `agh_environment_id` label.

---

### Preconditions

- [x] Session with `EnvironmentID` persisted but `InstanceID` empty (simulating timeout after remote create)
- [x] Mock provider implements `Finder` interface

---

### Test Steps

1. **Boot with partial-create metadata**
   - Input: `EnvironmentID = "env-123"`, `InstanceID = ""`, remote sandbox exists with `agh_environment_id = "env-123"` label
   - **Expected:** Provider.FindEnvironment called with `EnvironmentID`, returns sandbox info

2. **Verify reattach for recoverable case**
   - Input: Session is recoverable
   - **Expected:** Sandbox attached, `InstanceID` and `ProviderState` persisted

3. **Verify destroy for unrecoverable case**
   - Input: Session is unrecoverable
   - **Expected:** Sandbox destroyed, cleanup logged
