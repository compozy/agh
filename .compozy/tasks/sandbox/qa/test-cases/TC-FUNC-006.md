## TC-FUNC-006: Defaults.Environment cascade resolves profile

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify the environment resolution cascade: `Workspace.EnvironmentRef` -> `Config.Defaults.Environment` -> implicit `local`.

---

### Preconditions

- [x] Workspace resolution calls `buildResolvedWorkspace`
- [x] Config has `Defaults.Environment` field

---

### Test Steps

1. **Workspace with explicit EnvironmentRef**
   - Input: `Workspace.EnvironmentRef = "daytona-dev"`, `Config.Defaults.Environment = "staging"`
   - **Expected:** Resolved environment uses `daytona-dev` profile (workspace wins)

2. **Workspace without EnvironmentRef, config has default**
   - Input: `Workspace.EnvironmentRef = ""`, `Config.Defaults.Environment = "staging"`
   - **Expected:** Resolved environment uses `staging` profile

3. **No workspace ref, no config default**
   - Input: Both empty
   - **Expected:** Resolved environment is `local` (implicit default)
