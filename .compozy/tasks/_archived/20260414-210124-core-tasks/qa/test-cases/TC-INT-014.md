## TC-INT-014: Network peer creates task with channel binding and ActorKindNetworkPeer

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-14

---

### Objective
Validate that an authenticated network peer creates a task with a channel binding, the persisted task carries `created_by.kind = "network_peer"` and `origin.kind = "network"`, the channel is validated, and the task is correctly bound to the specified network channel.

---

### Preconditions
- [ ] AGH daemon running with all subsystems including network subsystem
- [ ] At least one network peer connected and authenticated
- [ ] A valid network channel exists (e.g., "ch-collab-001")
- [ ] The peer has task.write capability
- [ ] Task manager accessible to network ingress handlers
- [ ] Clean task store (or known baseline count)

---

### Test Steps

1. **Network peer creates a task with channel binding**
   - The network peer sends a task creation request through the network ingress layer
   - Actor context is derived via `DeriveNetworkPeerActorContext(peerRef, originRef)`
   - Input (effective request):
     ```json
     {
       "scope": "global",
       "title": "Network Peer Task",
       "network_channel": "ch-collab-001"
     }
     ```
   - **Expected:** Task created successfully

2. **Verify ActorKindNetworkPeer on the created task**
   - Input: `GET http://localhost:2123/api/tasks/<network-task-id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.created_by.kind` equals `"network_peer"`
   - **Expected:** `task.task.created_by.ref` is non-empty and identifies the peer (e.g., peer ID or name)

3. **Verify OriginKindNetwork on the created task**
   - **Expected:** `task.task.origin.kind` equals `"network"`
   - **Expected:** `task.task.origin.ref` is non-empty (peer ref or peer/channel context)

4. **Verify the actor-origin pair is valid**
   - **Expected:** The combination `actor.kind = "network_peer"` with `origin.kind = "network"` passes `validateActorOriginPair`

5. **Verify channel binding**
   - **Expected:** `task.task.network_channel` equals `"ch-collab-001"`
   - **Expected:** The channel was validated against `network.ValidateChannel` before creation

6. **List tasks filtered by network_channel**
   - Input: `GET http://localhost:2123/api/tasks?network_channel=ch-collab-001`
   - **Expected:** HTTP 200
   - **Expected:** The network peer's task appears in the filtered results

7. **Network peer creates task with invalid channel**
   - Input: Task creation with `network_channel: "invalid!!channel"`
   - **Expected:** Rejected with ErrValidation (channel format invalid)
   - **Expected:** No task persisted

8. **Network peer creates task without channel (allowed if policy permits)**
   - Input: Task creation without `network_channel` field
   - **Expected:** Either succeeds with empty channel or rejected based on network ingress policy
   - **Expected:** If allowed, `network_channel` is empty on the persisted task

9. **Enqueue run with network channel override**
   - Input: Enqueue a run for the network task with a different channel
     ```json
     {
       "network_channel": "ch-collab-002"
     }
     ```
   - **Expected:** Run created with `network_channel = "ch-collab-002"` (run-level override)

10. **Verify audit events carry network peer actor**
    - Input: Check events on the created task
    - **Expected:** task_created event has `actor.kind = "network_peer"` and `origin.kind = "network"`

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| task.created_by.kind | (server-derived) | "network_peer" | [ ] |
| task.created_by.ref | (server-derived) | Peer ID/name | [ ] |
| task.origin.kind | (server-derived) | "network" | [ ] |
| task.origin.ref | (server-derived) | Peer ref or context | [ ] |
| task.network_channel | "ch-collab-001" | "ch-collab-001" | [ ] |
| Actor-origin pair | validateActorOriginPair | No error | [ ] |
| Channel filter result | List with channel filter | Contains task | [ ] |
| Invalid channel | creation attempt | Rejected (400) | [ ] |
| Run channel override | "ch-collab-002" | "ch-collab-002" | [ ] |
| Event actor.kind | task_created event | "network_peer" | [ ] |
| Event origin.kind | task_created event | "network" | [ ] |

---

### Error Scenarios
- [ ] Network peer with `origin.kind = "http"` is rejected by `validateActorOriginPair` (network_peer requires network origin)
- [ ] Network peer with empty actor ref is rejected by validation
- [ ] Network peer without task.write capability is denied (ErrPermissionDenied, 403)
- [ ] Task creation with channel that does not exist in the network layer may be rejected depending on validation policy

---

### Related Test Cases
- TC-INT-011: Automation creates task (different actor kind)
- TC-INT-013: Extension creates task (different actor kind)
- TC-INT-015: Network peer writes to task with stale channel
