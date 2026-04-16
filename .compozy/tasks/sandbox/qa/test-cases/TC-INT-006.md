## TC-INT-006: Daytona provider E2E lifecycle

**Priority:** P2 (Medium)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-16
**Task:** 06
**Gated by:** DAYTONA_API_KEY environment variable

---

### Objective

Verify the complete Daytona provider lifecycle: create workspace with Daytona profile -> create session -> SSH transport works -> tar sync files -> stop session -> verify sync-back -> cleanup sandbox.

---

### Preconditions

- [x] `DAYTONA_API_KEY` set
- [x] Network access to Daytona API
- [x] Valid snapshot or image configured

---

### Test Steps

1. **Create sandbox via Daytona SDK**
   - **Expected:** Sandbox created with AGH labels, SSH token obtained

2. **SSH connect without PTY**
   - **Expected:** Clean stdio connection, no terminal escape sequences

3. **Sync workspace files via tar**
   - **Expected:** Files appear in sandbox at `RuntimeRootDir`

4. **Launch ACP agent via SSH**
   - **Expected:** Agent process starts, stdin/stdout pipes work for JSON-RPC

5. **Stop session and sync back**
   - **Expected:** Modified files synced back to local, last-write-wins applied

6. **Cleanup sandbox**
   - **Expected:** Sandbox destroyed (transient) or archived per persistence setting

---

### Error Scenarios

- [x] Network timeout: SDK calls have configured timeout
- [x] SSH token expiry: Refresh at 50% expiry
- [x] Sandbox creation failure: Error returned with context
