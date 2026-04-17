## TC-INT-004: Session list API includes environment field

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that session list and session get API responses include the `environment` field with correct values, end-to-end from session creation through API response.

---

### Test Steps

1. **Create session via API**
   - **Expected:** Session created with environment metadata

2. **List sessions via API**
   - **Expected:** Response includes `environment` object: `{environment_id: "...", backend: "local", state: "..."}`

3. **Get specific session via API**
   - **Expected:** Full environment payload: `environment_id`, `backend`, `profile`, `state`, `instance_id`, `sync_state`, `last_sync_error`

4. **Verify JSON serialization**
   - **Expected:** `SessionEnvironmentPayload` correctly serialized in JSON response
