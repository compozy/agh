## TC-FUNC-023: Reconciliation finds sandbox by agh_sandbox_id

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 07

---

### Objective

Verify that when local metadata has `SandboxID` but no `InstanceID` (partial create), reconciliation uses the `Finder` interface to discover remote sandboxes by `agh_sandbox_id` label.

---

### Preconditions

- [x] Session with `SandboxID` persisted but `InstanceID` empty (simulating timeout after remote create)
- [x] Mock provider implements `Finder` interface

---

### Test Steps

1. **Boot with partial-create metadata**
   - Input: `SandboxID = "env-123"`, `InstanceID = ""`, remote sandbox exists with `agh_sandbox_id = "env-123"` label
   - **Expected:** Provider.FindSandbox called with `SandboxID`, returns sandbox info

2. **Verify reattach for recoverable case**
   - Input: Session is recoverable
   - **Expected:** Sandbox attached, `InstanceID` and `ProviderState` persisted

3. **Verify destroy for unrecoverable case**
   - Input: Session is unrecoverable
   - **Expected:** Sandbox destroyed, cleanup logged
