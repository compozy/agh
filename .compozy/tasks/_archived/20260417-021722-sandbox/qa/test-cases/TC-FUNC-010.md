## TC-FUNC-010: Session start calls Provider.Prepare with correct fields

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that `startSession()` calls `Provider.Prepare()` with a correctly populated `PrepareRequest` containing session ID, workspace info, environment ID, and resolved profile.

---

### Preconditions

- [x] Mock provider captures PrepareRequest

---

### Test Steps

1. **Create session and capture PrepareRequest**
   - **Expected:** `PrepareRequest` contains:
     - `EnvironmentID` matching allocated ID
     - `SessionID` matching session
     - `WorkspaceID` matching workspace
     - `LocalRootDir` matching workspace root
     - `LocalAdditionalDirs` matching workspace additional dirs
     - `Profile` matching resolved environment profile
     - `Env` with session-specific vars (`AGH_SESSION_ID`, etc.)

2. **Verify resume case includes prior state**
   - Input: Resume session with existing environment metadata
   - **Expected:** `PrepareRequest.InstanceID` and `PrepareRequest.ProviderState` populated from persisted metadata
