## TC-FUNC-018: SSH auth failure triggers refresh and retry

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 06

---

### Objective

Verify that on SSH authentication failure, the system refreshes the token once and retries the connection. If the retry also fails, surface the error.

---

### Preconditions

- [x] SSH transport with mock auth
- [x] Token refresh mechanism available

---

### Test Steps

1. **Simulate first auth failure**
   - Input: SSH connect with expired token
   - **Expected:** Token refreshed, connection retried

2. **Simulate successful retry**
   - Input: Fresh token works
   - **Expected:** Connection established, no error surfaced

3. **Simulate both auth failures**
   - Input: Even fresh token fails (e.g., sandbox stopped)
   - **Expected:** Error surfaced to caller, no infinite retry loop
