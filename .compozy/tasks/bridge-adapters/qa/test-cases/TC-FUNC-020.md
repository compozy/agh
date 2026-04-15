## TC-FUNC-020: Bridge Health Metrics

**Priority:** P2
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective
Verify that the observer (`internal/observe`) correctly reports bridge health metrics including status counts (ready/degraded/error), per-instance route counts, delivery backlog, delivery failure counts, and auth failure tracking through the health query surface.

### Preconditions
- [ ] `internal/observe` package is compiled and testable
- [ ] An `Observer` is constructed with a mock `BridgeSource` that implements `ListInstances`, `ListRoutes`, and `DeliveryMetrics`
- [ ] Multiple bridge instances exist in various statuses

### Test Steps
1. **Query aggregate bridge health with mixed statuses**
   - Input: Mock `BridgeSource` returns 5 instances:
     - Instance A: `status=ready`, 3 routes
     - Instance B: `status=ready`, 1 route
     - Instance C: `status=degraded`, 2 routes
     - Instance D: `status=disabled`, 0 routes
     - Instance E: `status=error`, 0 routes
   - **Expected:** `BridgeAggregateHealth`:
     - `total_instances` = `5`
     - `status_counts.ready` = `2`
     - `status_counts.degraded` = `1`
     - `status_counts.disabled` = `1`
     - `status_counts.error` = `1`
     - `status_counts.starting` = `0`
     - `status_counts.auth_required` = `0`
     - `route_count` = `6` (sum: 3+1+2+0+0)

2. **Query per-instance health with delivery metrics**
   - Input: Mock `DeliveryMetrics` returns for Instance A:
     ```json
     {
       "delivery_backlog": 3,
       "delivery_dropped_total": 1,
       "delivery_dropped_by_reason": {"queue_saturated": 1},
       "delivery_failures_total": 2,
       "last_success_at": "2026-04-15T09:55:00Z",
       "last_error": "timeout sending to Slack",
       "last_error_at": "2026-04-15T09:50:00Z"
     }
     ```
   - **Expected:** `BridgeInstanceHealth` for Instance A:
     - `bridge_instance_id` = Instance A's ID
     - `status` = `"ready"`
     - `route_count` = `3`
     - `delivery_backlog` = `3`
     - `delivery_dropped_total` = `1`
     - `delivery_dropped_by_reason["queue_saturated"]` = `1`
     - `delivery_failures_total` = `2`
     - `last_success_at` is set
     - `last_error` = `"timeout sending to Slack"`
     - `last_error_at` is set

3. **Record auth failure and verify counter**
   - Input: Call `observer.RecordBridgeAuthFailure("inst-A")` three times
   - **Expected:**
     - `BridgeInstanceHealth` for Instance A has `auth_failures_total` = `3`
     - Aggregate `auth_failures_total` includes the 3 failures

4. **Record runtime issue and verify effective status override**
   - Input: Call `observer.RecordBridgeRuntimeIssue("inst-A", BridgeStatusDegraded, "provider slow")`
   - **Expected:**
     - Instance A persisted status is still `ready`
     - But `effectiveBridgeStatus` returns `degraded` (runtime override)
     - `BridgeInstanceHealth.Status` = `"degraded"`
     - `status_counts.degraded` incremented, `status_counts.ready` decremented

5. **Clear runtime issue and verify status restored**
   - Input: Call `observer.ClearBridgeRuntimeIssue("inst-A")`
   - **Expected:**
     - `effectiveBridgeStatus` returns persisted status `ready`
     - `BridgeInstanceHealth.Status` = `"ready"`
     - `status_counts.ready` restored

6. **Verify aggregate delivery backlog**
   - Input: Mock returns delivery_backlog for A=3, B=0, C=5
   - **Expected:** `BridgeAggregateHealth.delivery_backlog` = `8`

7. **Verify aggregate delivery totals**
   - Input: Mock returns `delivery_dropped_total` A=1, C=2; `delivery_failures_total` A=2, C=3
   - **Expected:**
     - `delivery_dropped_total` = `3`
     - `delivery_failures_total` = `5`

8. **Verify nil BridgeSource returns empty**
   - Input: Observer with `bridgeSource = nil`
   - **Expected:** `QueryBridgeHealth` returns empty slice, zero aggregate

9. **Verify health output is sorted by instance ID**
   - Input: Multiple instances returned in arbitrary order
   - **Expected:** `QueryBridgeHealth` returns slice sorted by `bridge_instance_id` (string comparison)

10. **Verify runtime error overrides persisted status**
    - Input: Instance B is `ready`, `RecordBridgeRuntimeIssue("inst-B", BridgeStatusError, "crash")`
    - **Expected:** `effectiveBridgeStatus` returns `error` (runtime error takes priority over persisted ready)

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Disabled instance with runtime override | Instance D is `disabled`; RecordBridgeRuntimeIssue called | Effective status remains `disabled` (disabled takes absolute priority) |
| Auth_required persisted + degraded runtime | Instance with `status=auth_required`; runtime reports degraded | Effective status is `auth_required` (persisted auth_required takes priority) |
| RecordBridgeAuthFailure with empty ID | `RecordBridgeAuthFailure("")` | No-op; no crash |
| RecordBridgeRuntimeIssue with ready status | `RecordBridgeRuntimeIssue("id", BridgeStatusReady, "ok")` | No-op; only degraded/error statuses are recorded |
| Concurrent health queries | Multiple goroutines calling QueryBridgeHealth | Thread-safe via RWMutex; no data races |
| ListRoutes returns error | BridgeSource.ListRoutes fails | QueryBridgeHealth returns wrapped error |

### Related Test Cases
- TC-FUNC-014 (degradation reporting — feeds into health metrics)
- TC-FUNC-015 (recovery — clears health metrics)
- TC-FUNC-005 (lifecycle — status changes reflected in counts)
