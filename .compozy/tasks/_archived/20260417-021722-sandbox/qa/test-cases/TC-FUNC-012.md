## TC-FUNC-012: Session start uses RuntimeRootDir in StartOpts

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that `startSession()` uses `Prepared.RuntimeRootDir` (not the local workspace path) in `acp.StartOpts.Cwd`, and `Prepared.RuntimeAdditionalDirs` in `StartOpts.AdditionalDirs`.

---

### Preconditions

- [x] Mock provider returns custom RuntimeRootDir in Prepared
- [x] ACP driver captures StartOpts

---

### Test Steps

1. **Create session with remote provider returning custom runtime path**
   - Input: Provider returns `RuntimeRootDir = "/home/daytona/workspace"`, `RuntimeAdditionalDirs = ["/home/daytona/extra"]`
   - **Expected:** `StartOpts.Cwd == "/home/daytona/workspace"`, `StartOpts.AdditionalDirs` includes `"/home/daytona/extra"`

2. **Create session with local provider**
   - Input: Local provider returns `RuntimeRootDir == LocalRootDir`
   - **Expected:** `StartOpts.Cwd` matches original workspace `RootDir` (unchanged behavior)
