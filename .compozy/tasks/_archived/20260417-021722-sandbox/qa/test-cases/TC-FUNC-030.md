## TC-FUNC-030: Host API sandbox/exec requires capability

**Priority:** P2 (Medium)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 08

---

### Objective

Verify that the `sandbox/exec` Host API method requires the `sandbox.exec` security capability, and callers without it receive an authorization error.

---

### Preconditions

- [x] Host API method registered with capability requirement
- [x] Capability mapping in `internal/extension/capability.go`

---

### Test Steps

1. **Call sandbox/exec without capability**
   - Input: Extension without `sandbox.exec` grant calls `{session_id: "...", command: "echo hello"}`
   - **Expected:** Authorization error returned

2. **Call sandbox/exec with capability**
   - Input: Extension with `sandbox.exec` grant calls same request
   - **Expected:** Command executed, response includes `exit_code`, `stdout`, `stderr`

3. **Verify timeout handling**
   - Input: `{session_id: "...", command: "sleep 60", timeout: 5}`
   - **Expected:** Command killed after timeout, error or non-zero exit code returned
