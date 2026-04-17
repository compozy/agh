## TC-FUNC-028: environment.stop deny prevents destroy

**Priority:** P2 (Medium)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 08

---

### Objective

Verify that when the `environment.stop` sync hook returns Deny, the sandbox is NOT destroyed but the session still stops.

---

### Preconditions

- [x] Hook registered for `environment.stop` that returns Deny
- [x] Session with transient persistence (would normally destroy)

---

### Test Steps

1. **Stop session with hook that denies stop**
   - Input: Hook returns `{Deny: true, DenyReason: "preserve for debugging"}`
   - **Expected:** `Provider.Destroy()` is NOT called, session stops normally, sandbox left alive

2. **Verify session metadata reflects stopped state**
   - **Expected:** Session is stopped, environment state updated accordingly
