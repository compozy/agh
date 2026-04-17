## TC-FUNC-019: SSH keepalive 30s interval

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 06

---

### Objective

Verify SSH client-level keepalive is configured with a 30-second interval to prevent connection drops during idle periods.

---

### Preconditions

- [x] SSH client configuration accessible

---

### Test Steps

1. **Inspect SSH client config**
   - **Expected:** `ClientConfig` includes keepalive or `ServerAliveInterval` equivalent set to 30s

2. **Verify connection stays alive during idle**
   - Input: Establish SSH session, wait 60+ seconds without activity
   - **Expected:** Connection remains open, no timeout error
