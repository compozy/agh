## TC-FUNC-014: Structured Degradation Reporting

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective
Validate that a provider can report structured degradation on a bridge instance via the Host API `instances/report_state`, that the instance transitions to `auth_required` or `degraded` with the correct `BridgeDegradation` reason and message, and that the degradation payload persists correctly.

### Preconditions
- [ ] Daemon is running with a bridge instance in `status=ready`, `enabled=true`
- [ ] Bridge instance ID is known (e.g., `inst-001`)
- [ ] Host API `instances/report_state` method is available

### Test Steps
1. **Report auth_failed degradation**
   - Input: `instances/report_state` with:
     ```json
     {
       "bridge_instance_id": "inst-001",
       "status": "auth_required",
       "degradation": {
         "reason": "auth_failed",
         "message": "OAuth token expired at 2026-04-15T09:30:00Z"
       }
     }
     ```
   - **Expected:**
     - Instance transitions from `ready` to `auth_required` (valid transition per lifecycle.go)
     - `degradation.reason` = `"auth_failed"`
     - `degradation.message` = `"OAuth token expired at 2026-04-15T09:30:00Z"`
     - `updated_at` timestamp advances

2. **Report rate_limited degradation**
   - Input: First reset instance to `ready`, then report:
     ```json
     {
       "bridge_instance_id": "inst-001",
       "status": "degraded",
       "degradation": {
         "reason": "rate_limited",
         "message": "Slack API rate limit exceeded, retry after 30s"
       }
     }
     ```
   - **Expected:**
     - Instance transitions to `degraded`
     - `degradation.reason` = `"rate_limited"`
     - `degradation.message` contains the descriptive text

3. **Report webhook_invalid degradation**
   - Input:
     ```json
     {
       "status": "degraded",
       "degradation": {"reason": "webhook_invalid", "message": "Webhook URL returned 404"}
     }
     ```
   - **Expected:** `degradation.reason` = `"webhook_invalid"`

4. **Report provider_timeout degradation**
   - Input:
     ```json
     {
       "status": "degraded",
       "degradation": {"reason": "provider_timeout", "message": "Telegram API timed out after 10s"}
     }
     ```
   - **Expected:** `degradation.reason` = `"provider_timeout"`

5. **Report tenant_config_invalid degradation**
   - Input:
     ```json
     {
       "status": "degraded",
       "degradation": {"reason": "tenant_config_invalid", "message": "Missing required enterprise_url in provider_config"}
     }
     ```
   - **Expected:** `degradation.reason` = `"tenant_config_invalid"`

6. **Verify degradation is only allowed with degraded/auth_required/error status**
   - Input: Report degradation with `status: "ready"`
   - **Expected:** Validation error: "bridge degradation requires degraded, auth_required, or error status"

7. **Verify degradation reason is required when payload is present**
   - Input: `degradation: {"reason": "", "message": "some issue"}`
   - **Expected:** Validation error: "bridge degradation reason is required"

8. **Verify unsupported degradation reason is rejected**
   - Input: `degradation: {"reason": "network_partition"}`
   - **Expected:** Validation error: "unsupported bridge degradation reason"

9. **Retrieve instance and verify degradation persists**
   - Input: `instances/get` with the instance ID
   - **Expected:** `degradation` object is present with the reported reason and message

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Degradation with empty message | `degradation: {"reason": "auth_failed"}` | Valid; message is optional |
| Degradation with whitespace-only message | `degradation: {"reason": "auth_failed", "message": "   "}` | Normalized to empty message |
| Null degradation on degraded status | `status: "degraded", degradation: null` | Depends on implementation; may require degradation for degraded status |
| Report state on non-existent instance | Unknown bridge_instance_id | Error: ErrBridgeInstanceNotFound |
| Report state on disabled instance | Instance with `enabled=false` | Invalid transition attempt |

### Related Test Cases
- TC-FUNC-005 (lifecycle transitions)
- TC-FUNC-013 (error classification triggers degradation)
- TC-FUNC-015 (recovery from degradation)
- TC-FUNC-020 (health metrics reflect degradation)
