## TC-FUNC-005: Lifecycle State Machine Transitions

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-15

---

### Objective
Verify all valid lifecycle transitions succeed and all invalid transitions are rejected with `ErrInvalidBridgeStateTransition`. The state machine enforces that bridge instances follow the allowed transition graph defined in `lifecycle.go`.

### Preconditions
- [ ] `internal/bridges` package is compiled and testable
- [ ] `ValidateInstanceStateTransition` function is available
- [ ] Understanding of the six statuses: `disabled`, `starting`, `ready`, `degraded`, `auth_required`, `error`

### Test Steps
1. **Validate all VALID transitions (should return nil error)**
   - Input/Expected table:

   | From (enabled, status) | To (enabled, status) | Expected |
   |------------------------|----------------------|----------|
   | `false, disabled` | `true, starting` | Valid |
   | `true, starting` | `true, ready` | Valid |
   | `true, starting` | `true, degraded` | Valid |
   | `true, starting` | `true, auth_required` | Valid |
   | `true, starting` | `true, error` | Valid |
   | `true, starting` | `false, disabled` | Valid |
   | `true, starting` | `true, starting` | Valid (no-op) |
   | `true, ready` | `true, degraded` | Valid |
   | `true, ready` | `true, auth_required` | Valid |
   | `true, ready` | `true, error` | Valid |
   | `true, ready` | `true, starting` | Valid |
   | `true, ready` | `false, disabled` | Valid |
   | `true, ready` | `true, ready` | Valid (no-op) |
   | `true, degraded` | `true, ready` | Valid (recovery) |
   | `true, degraded` | `true, starting` | Valid (restart) |
   | `true, degraded` | `true, auth_required` | Valid |
   | `true, degraded` | `true, error` | Valid |
   | `true, degraded` | `false, disabled` | Valid |
   | `true, degraded` | `true, degraded` | Valid (no-op) |
   | `true, auth_required` | `true, starting` | Valid (re-auth restart) |
   | `true, auth_required` | `true, error` | Valid |
   | `true, auth_required` | `false, disabled` | Valid |
   | `true, auth_required` | `true, auth_required` | Valid (no-op) |
   | `true, error` | `true, starting` | Valid (retry) |
   | `true, error` | `false, disabled` | Valid |
   | `true, error` | `true, error` | Valid (no-op) |

2. **Validate all INVALID transitions (should return ErrInvalidBridgeStateTransition)**
   - Input/Expected table:

   | From (enabled, status) | To (enabled, status) | Expected |
   |------------------------|----------------------|----------|
   | `false, disabled` | `true, ready` | Invalid (must go through starting) |
   | `false, disabled` | `true, degraded` | Invalid |
   | `false, disabled` | `true, error` | Invalid |
   | `false, disabled` | `true, auth_required` | Invalid |
   | `true, auth_required` | `true, ready` | Invalid (must go through starting) |
   | `true, auth_required` | `true, degraded` | Invalid |
   | `true, error` | `true, ready` | Invalid (must go through starting) |
   | `true, error` | `true, degraded` | Invalid |
   | `true, error` | `true, auth_required` | Invalid |

3. **Verify enabled/status invariant enforcement**
   - Input: `enabled=true, status=disabled`
   - **Expected:** Validation error: "enabled bridge instance cannot report status disabled"

   - Input: `enabled=false, status=ready`
   - **Expected:** Validation error: "disabled bridge instance must report status disabled"

4. **Verify same-state transitions are no-ops**
   - Input: Instance with `enabled=true, status=ready`, transition to `enabled=true, status=ready`
   - **Expected:** Returns nil (valid no-change)

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Transition from invalid current state | Instance with `enabled=false, status=ready` (should never exist) | Validation error on current state |
| Unnormalized status string | `status=" Ready "` | Normalized to `"ready"` before checking |
| Unknown status value | `status="paused"` | Validation error: unsupported bridge status |

### Related Test Cases
- TC-FUNC-002 (enable/start flow)
- TC-FUNC-014 (degradation triggers status change)
- TC-FUNC-015 (recovery clears degradation)
