## TC-FUNC-015: Rate Limit Recovery

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-15

---

### Objective

Validate that after a rate_limited degradation is reported on a bridge instance, the provider can subsequently report recovery, transitioning the instance from `degraded` back to `ready` with the degradation payload cleared.

### Preconditions

- [ ] Daemon is running with a bridge instance in `status=ready`, `enabled=true`
- [ ] Bridge instance ID is known (e.g., `inst-001`)
- [ ] Host API `instances/report_state` method is available

### Test Steps

1. **Report rate_limited degradation**
   - Input: `instances/report_state` with:
     ```json
     {
       "bridge_instance_id": "inst-001",
       "status": "degraded",
       "degradation": {
         "reason": "rate_limited",
         "message": "Discord API rate limit: retry after 60s"
       }
     }
     ```
   - **Expected:**
     - Instance transitions `ready -> degraded`
     - `status` = `"degraded"`
     - `degradation.reason` = `"rate_limited"`

2. **Verify instance is degraded via get**
   - Input: `instances/get` with instance ID
   - **Expected:**
     - `status` = `"degraded"`
     - `degradation` is not null
     - `degradation.reason` = `"rate_limited"`

3. **Report recovery to ready status**
   - Input: `instances/report_state` with:
     ```json
     {
       "bridge_instance_id": "inst-001",
       "status": "ready",
       "degradation": null
     }
     ```
   - **Expected:**
     - `ValidateInstanceStateTransition` allows `degraded -> ready` (valid transition)
     - Instance transitions to `status=ready`
     - `degradation` is cleared (null)
     - `updated_at` timestamp advances

4. **Verify instance is fully recovered via get**
   - Input: `instances/get` with instance ID
   - **Expected:**
     - `status` = `"ready"`
     - `degradation` is `null`
     - `enabled` = `true`

5. **Verify observer clears runtime issue on recovery**
   - Input: Check `Observer.ClearBridgeRuntimeIssue(instanceID)` is invoked during recovery
   - **Expected:**
     - The observed bridge state for this instance has `runtimeStatus=""`, `runtimeMessage=""`, `runtimeUpdatedAt=zero`
     - Health endpoint no longer reports this instance as degraded

6. **Verify recovery cycle can repeat**
   - Input: Report degradation again -> verify degraded -> report recovery -> verify ready
   - **Expected:** Full degradation -> recovery cycle works multiple times without state leaks

### Edge Cases & Variations

| Variation                               | Input                                               | Expected Result                                    |
| --------------------------------------- | --------------------------------------------------- | -------------------------------------------------- |
| Recovery from provider_timeout          | degraded(provider_timeout) -> ready                 | Valid; degradation cleared                         |
| Recovery from webhook_invalid           | degraded(webhook_invalid) -> ready                  | Valid; degradation cleared                         |
| Recovery from auth_required             | auth_required -> starting -> ready                  | Valid via starting intermediate state              |
| Direct auth_required -> ready           | `status: "ready"` from auth_required                | Invalid transition (must go through starting)      |
| Recovery from error -> starting         | error -> starting                                   | Valid; then starting -> ready                      |
| Direct error -> ready                   | `status: "ready"` from error                        | Invalid transition (must go through starting)      |
| Degraded -> degraded (different reason) | Change reason from rate_limited to provider_timeout | Valid no-op transition; degradation reason updates |

### Related Test Cases

- TC-FUNC-005 (lifecycle state machine)
- TC-FUNC-013 (error classification triggers rate_limit)
- TC-FUNC-014 (degradation reporting)
- TC-FUNC-020 (health metrics reflect recovery)
