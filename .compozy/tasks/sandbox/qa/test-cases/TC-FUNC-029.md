## TC-FUNC-029: Host API environment/list returns instances

**Priority:** P2 (Medium)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 08

---

### Objective

Verify that the `environment/list` Host API method returns active environment instances visible to the caller.

---

### Preconditions

- [x] Host API method registered
- [x] At least one active session with environment

---

### Test Steps

1. **Call environment/list**
   - Input: `{}`
   - **Expected:** Response includes array of environments with `session_id`, `backend`, `profile`, `instance_id`, `state`

2. **Verify only active environments returned**
   - **Expected:** Stopped/destroyed sessions not included in list

3. **Verify visibility filtering**
   - **Expected:** Workspace-scoped extensions only see environments for sessions in their workspace
