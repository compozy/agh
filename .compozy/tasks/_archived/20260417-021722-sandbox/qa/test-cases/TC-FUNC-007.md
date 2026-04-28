## TC-FUNC-007: Missing SandboxRef resolves to local

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that when neither workspace nor config defaults specify an environment, the resolved environment defaults to local backend with no-op sync and no-op lifecycle.

---

### Preconditions

- [x] Workspace has no `SandboxRef`
- [x] Config has no `Defaults.Sandbox`

---

### Test Steps

1. **Resolve workspace with no sandbox reference**
   - Input: Workspace with `SandboxRef = ""`, config with no default
   - **Expected:** `ResolvedWorkspace.Sandbox.Backend == BackendLocal`, `SyncMode == SyncModeNone`, `Persistence == PersistenceTransient`

2. **Verify local provider is used for session**
   - **Expected:** Session manager selects local provider from registry, Prepare is a no-op
