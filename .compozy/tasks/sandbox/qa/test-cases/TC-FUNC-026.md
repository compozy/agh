## TC-FUNC-026: environment.prepare deny aborts session creation

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 08

---

### Objective

Verify that when the `environment.prepare` sync hook returns a `ControlPatch.Deny`, session creation is aborted with an error that includes the deny reason.

---

### Preconditions

- [x] Hook registered for `environment.prepare` that returns Deny patch

---

### Test Steps

1. **Create session with hook that denies prepare**
   - Input: Hook returns `{Deny: true, DenyReason: "policy violation"}`
   - **Expected:** Session creation fails, error message includes "policy violation", `Provider.Prepare()` is NOT called

2. **Verify hook payload contains expected fields**
   - **Expected:** Payload includes `workspace_id`, `backend`, `profile`
