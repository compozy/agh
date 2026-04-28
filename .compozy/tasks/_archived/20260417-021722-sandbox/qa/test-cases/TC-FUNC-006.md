## TC-FUNC-006: Defaults.Sandbox cascade resolves profile

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify the environment resolution cascade: `Workspace.SandboxRef` -> `Config.Defaults.Sandbox` -> implicit `local`.

---

### Preconditions

- [x] Workspace resolution calls `buildResolvedWorkspace`
- [x] Config has `Defaults.Sandbox` field

---

### Test Steps

1. **Workspace with explicit SandboxRef**
   - Input: `Workspace.SandboxRef = "daytona-dev"`, `Config.Defaults.Sandbox = "staging"`
   - **Expected:** Resolved environment uses `daytona-dev` profile (workspace wins)

2. **Workspace without SandboxRef, config has default**
   - Input: `Workspace.SandboxRef = ""`, `Config.Defaults.Sandbox = "staging"`
   - **Expected:** Resolved environment uses `staging` profile

3. **No workspace ref, no config default**
   - Input: Both empty
   - **Expected:** Resolved environment is `local` (implicit default)
