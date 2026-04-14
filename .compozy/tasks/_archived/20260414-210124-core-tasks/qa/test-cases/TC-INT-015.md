## TC-INT-015: Network peer writes to task with stale channel returns ErrStaleNetworkChannel; task still readable

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that when a network peer attempts a write operation (update, cancel, enqueue run, etc.) on a task whose stored `network_channel` no longer passes the active channel validator, the operation is rejected with `ErrStaleNetworkChannel` (HTTP 409), but the task remains fully readable via GET endpoints.

---

### Preconditions
- [ ] AGH daemon running with all subsystems including network subsystem
- [ ] A network peer connected and authenticated with task.write capability
- [ ] A valid network channel "ch-stale-test" initially exists and passes validation
- [ ] One task T created by the network peer with `network_channel = "ch-stale-test"`
- [ ] A run R enqueued for task T (to test run-level stale channel rejection)
- [ ] Mechanism to invalidate/remove the channel "ch-stale-test" from the active validator (e.g., disconnect the channel, remove the peer, or reconfigure the network layer so the channel is no longer valid)

---

### Test Steps

1. **Verify the task is writable while channel is valid**
   - Input: Update task T via network peer:
     ```json
     PATCH /api/tasks/<T.id>
     {"title": "Updated While Valid"}
     ```
   - **Expected:** HTTP 200
   - **Expected:** Task title updated to "Updated While Valid"

2. **Invalidate the network channel**
   - Action: Remove or invalidate "ch-stale-test" from the active network channel validator
   - This could be done by:
     - Disconnecting the network peer that owns the channel
     - Removing the channel from the network configuration
     - Simulating a channel expiry event
   - **Expected:** `network.ValidateChannel("ch-stale-test")` now returns an error

3. **Attempt to update the task with the stale channel**
   - Input: Network peer attempts:
     ```json
     PATCH /api/tasks/<T.id>
     {"title": "Should Fail - Stale Channel"}
     ```
   - **Expected:** HTTP 409 Conflict
   - **Expected:** Error message references ErrStaleNetworkChannel or "stale network channel"
   - **Expected:** Task title remains "Updated While Valid" (unchanged)

4. **Attempt to cancel the task with the stale channel**
   - Input:
     ```json
     POST /api/tasks/<T.id>/cancel
     {"reason": "Stale channel cancel attempt"}
     ```
   - **Expected:** HTTP 409 Conflict (ErrStaleNetworkChannel)
   - **Expected:** Task status remains unchanged (not cancelled)

5. **Attempt to enqueue a run with the stale channel**
   - Input:
     ```json
     POST /api/tasks/<T.id>/runs
     {}
     ```
   - **Expected:** HTTP 409 Conflict (ErrStaleNetworkChannel)
   - **Expected:** No new run created

6. **Attempt to claim an existing run on the stale-channel task**
   - Input:
     ```json
     POST /api/task-runs/<R.id>/claim
     {}
     ```
   - **Expected:** HTTP 409 Conflict (ErrStaleNetworkChannel) if channel validation applies to run operations
   - **Expected:** Or: claim succeeds if channel validation only applies to task-level writes

7. **Verify the task is still fully readable**
   - Input: `GET http://localhost:2123/api/tasks/<T.id>`
   - **Expected:** HTTP 200
   - **Expected:** Full TaskDetailPayload returned with all fields intact
   - **Expected:** `task.task.network_channel` still equals "ch-stale-test" (the stored value is preserved)
   - **Expected:** Children, dependencies, runs, events all accessible

8. **List tasks includes the stale-channel task**
   - Input: `GET http://localhost:2123/api/tasks`
   - **Expected:** HTTP 200
   - **Expected:** Task T appears in the list

9. **Filter by the stale channel still works for reads**
   - Input: `GET http://localhost:2123/api/tasks?network_channel=ch-stale-test`
   - **Expected:** HTTP 200
   - **Expected:** Task T appears in the filtered results (read operations do not validate channel liveness)

10. **Restore the channel and verify writes resume**
    - Action: Re-validate "ch-stale-test" by re-adding it to the active network validator
    - Input: Network peer updates the task:
      ```json
      PATCH /api/tasks/<T.id>
      {"title": "Updated After Channel Restored"}
      ```
    - **Expected:** HTTP 200
    - **Expected:** Task title updated to "Updated After Channel Restored"

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| Write while valid | PATCH title | 200, title updated | [ ] |
| Write after invalidation | PATCH title | 409, ErrStaleNetworkChannel | [ ] |
| Cancel after invalidation | POST cancel | 409, ErrStaleNetworkChannel | [ ] |
| Enqueue after invalidation | POST runs | 409, ErrStaleNetworkChannel | [ ] |
| GET after invalidation | GET task | 200, full payload | [ ] |
| List after invalidation | GET tasks | 200, task included | [ ] |
| Filter by stale channel | GET tasks?channel | 200, task included | [ ] |
| task.network_channel (stored) | "ch-stale-test" | "ch-stale-test" (preserved) | [ ] |
| Write after restoration | PATCH title | 200, title updated | [ ] |

---

### Error Scenarios
- [ ] Multiple consecutive write attempts all return 409 (no partial state corruption)
- [ ] Stale channel does not cause 500 (the error is a clean 409, not an internal server error)
- [ ] Other tasks without channel binding remain fully writable during channel invalidity
- [ ] Tasks on different valid channels remain fully writable

---

### Related Test Cases
- TC-INT-014: Network peer creates task with channel binding (prerequisite setup)
- TC-INT-004: PATCH behavior for mutable fields (same endpoint, different error path)
- TC-INT-005: UDS parity for stale-channel rejection
- TC-INT-002: List filtering behavior when channel tasks are present
