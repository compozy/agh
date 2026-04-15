## TC-FUNC-002: Bridge Instance Enable/Start

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Validate that enabling a disabled bridge instance triggers the correct lifecycle transition from disabled to starting to ready when the provider runtime initializes successfully.

### Preconditions
- [ ] Daemon is running with the Telegram provider extension registered
- [ ] A bridge instance exists with `enabled=false`, `status=disabled`
- [ ] Provider runtime (subprocess) is healthy and can accept instance initialization
- [ ] SQLite store is available via `t.TempDir()` isolation

### Test Steps
1. **Create a disabled bridge instance**
   - Input:
     ```json
     {
       "scope": "global",
       "platform": "telegram",
       "extension_name": "bridges/telegram",
       "display_name": "Test Telegram",
       "enabled": false,
       "status": "disabled"
     }
     ```
   - **Expected:** Instance created with `enabled=false`, `status=disabled`

2. **Enable the instance by transitioning to starting**
   - Input: Update instance with `enabled=true`, `status=starting`
   - **Expected:**
     - `ValidateInstanceStateTransition(current, true, "starting")` returns nil
     - Instance persists with `enabled=true`, `status=starting`
     - `updated_at` timestamp advances

3. **Verify the provider runtime receives the instance snapshot**
   - Input: Observe the provider-scoped runtime's internal instance cache
   - **Expected:** The runtime's `InstanceCache` contains the newly enabled instance with its resolved secret bindings and provider_config

4. **Provider reports ready status via instances/report_state**
   - Input: Provider calls Host API `instances/report_state` with `status=ready` for the instance
   - **Expected:**
     - `ValidateInstanceStateTransition(current, true, "ready")` returns nil (starting -> ready is valid)
     - Instance persists with `enabled=true`, `status=ready`
     - `degradation` remains `null`
     - `updated_at` timestamp advances

5. **Verify the instance is now routable**
   - Input: Send an inbound message targeting this bridge_instance_id
   - **Expected:** Message is accepted and routed (not rejected with `ErrBridgeInstanceUnavailable`)

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Direct disabled -> ready (skip starting) | `enabled=true, status=ready` | Rejected with `ErrInvalidBridgeStateTransition` |
| Enable without changing status | `enabled=true, status=disabled` | Rejected: enabled instance cannot report disabled status |
| Starting -> degraded (partial init) | Provider reports `status=degraded` during init | Valid transition; instance persists as degraded with degradation reason |
| Starting -> error (init failure) | Provider reports `status=error` during init | Valid transition; instance persists as error |
| Starting -> auth_required | Provider reports `status=auth_required` | Valid transition; instance persists as auth_required |
| Re-enable after error -> starting | Instance in error state, update to `enabled=true, status=starting` | Valid transition (error -> starting is allowed) |
| Double-enable (starting -> starting) | Already starting, report starting again | Valid no-op transition |

### Related Test Cases
- TC-FUNC-001 (creation)
- TC-FUNC-005 (full state machine)
- TC-FUNC-014 (degradation reporting)
