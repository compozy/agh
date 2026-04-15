## TC-FUNC-004: Bridge Instance List and Get

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective

Validate that bridge instances can be listed with filters for scope, platform, and status, and that individual instances can be retrieved by ID with the correct response shape matching the API contract.

### Preconditions

- [ ] Daemon is running with at least two bridge provider extensions registered (e.g., `telegram`, `slack`)
- [ ] Multiple bridge instances exist with different scopes, platforms, and statuses:
  - Instance A: `scope=global, platform=telegram, status=disabled`
  - Instance B: `scope=global, platform=slack, status=ready, enabled=true`
  - Instance C: `scope=workspace, workspace_id=ws-001, platform=telegram, status=degraded, enabled=true`
  - Instance D: `scope=workspace, workspace_id=ws-001, platform=slack, status=disabled`
  - Instance E: `scope=workspace, workspace_id=ws-002, platform=telegram, status=ready, enabled=true`

### Test Steps

1. **List all instances (unfiltered)**
   - Input: `instances/list` with no filters
   - **Expected:**
     - Response is an array of `BridgeInstance` objects
     - Contains all 5 instances (A through E)
     - Each instance has all required fields: `id`, `scope`, `platform`, `extension_name`, `display_name`, `source`, `enabled`, `status`, `dm_policy`, `routing_policy`, `created_at`, `updated_at`

2. **List filtered by scope=global**
   - Input: `instances/list` with filter `scope=global`
   - **Expected:** Returns exactly instances A and B

3. **List filtered by platform=telegram**
   - Input: `instances/list` with filter `platform=telegram`
   - **Expected:** Returns exactly instances A, C, and E

4. **List filtered by status=ready**
   - Input: `instances/list` with filter `status=ready`
   - **Expected:** Returns exactly instances B and E

5. **List filtered by scope=workspace, workspace_id=ws-001**
   - Input: `instances/list` with filter `scope=workspace, workspace_id=ws-001`
   - **Expected:** Returns exactly instances C and D

6. **List with combined filters: scope=global, platform=slack**
   - Input: `instances/list` with `scope=global, platform=slack`
   - **Expected:** Returns exactly instance B

7. **List with filters that match nothing**
   - Input: `instances/list` with `platform=discord`
   - **Expected:** Returns empty array `[]`, not null

8. **Get a single instance by ID**
   - Input: `instances/get` with Instance B's ID
   - **Expected:**
     - Response is a single `BridgeInstance` object (not an array)
     - All fields match the persisted values for Instance B
     - `provider_config` and `delivery_defaults` are valid JSON or null
     - `degradation` is null (Instance B is ready)

9. **Get instance C (degraded) — verify degradation payload**
   - Input: `instances/get` with Instance C's ID
   - **Expected:**
     - `status` = `"degraded"`
     - `degradation` is an object with `reason` and optional `message`
     - `degradation.reason` is one of the valid `BridgeDegradationReason` values

10. **Get non-existent instance**
    - Input: `instances/get` with `id=nonexistent-uuid`
    - **Expected:** Error response with `ErrBridgeInstanceNotFound`

### Edge Cases & Variations

| Variation                        | Input                          | Expected Result                                     |
| -------------------------------- | ------------------------------ | --------------------------------------------------- |
| Empty instance store             | List with no instances created | Returns empty array `[]`                            |
| Filter by invalid scope          | `scope=tenant`                 | Validation error or empty result                    |
| Filter by invalid status         | `status=paused`                | Validation error or empty result                    |
| Case-insensitive platform filter | `platform=Telegram`            | Normalizes to lowercase; returns matching instances |
| Get with whitespace-padded ID    | `id=" inst-id "`               | Normalizes; returns matching instance               |

### Related Test Cases

- TC-FUNC-001 (creation — sets up instances for listing)
- TC-FUNC-003 (update — changes fields returned in list/get)
- TC-FUNC-005 (lifecycle — affects status filter results)
