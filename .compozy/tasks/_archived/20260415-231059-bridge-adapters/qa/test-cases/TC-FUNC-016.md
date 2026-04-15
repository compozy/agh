## TC-FUNC-016: Routing Key Construction

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective

Validate that `BuildRoutingKey` constructs canonical routing keys from various combinations of scope, workspace_id, bridge_instance_id, peer_id, thread_id, and group_id, respecting the instance's `RoutingPolicy` to include or exclude dimensions. Verify the key serializes to a stable JSON representation and hashes to a deterministic SHA-256 value.

### Preconditions

- [ ] `internal/bridges` package is compiled and testable
- [ ] `BuildRoutingKey`, `RoutingKey.Serialize()`, and `RoutingKey.Hash()` functions are available
- [ ] Test instances with various `RoutingPolicy` configurations are constructable

### Test Steps

1. **Build routing key with peer-only policy (global scope)**
   - Input:
     - Instance: `id="inst-001", scope=global, routing_policy={include_peer: true, include_thread: false, include_group: false}`
     - Dimensions: `peer_id="user-42", thread_id="thread-1", group_id="group-a"`
   - **Expected:**
     - `routing_key.Scope` = `"global"`
     - `routing_key.WorkspaceID` = `""`
     - `routing_key.BridgeInstanceID` = `"inst-001"`
     - `routing_key.PeerID` = `"user-42"` (included)
     - `routing_key.ThreadID` = `""` (excluded by policy)
     - `routing_key.GroupID` = `""` (excluded by policy)

2. **Build routing key with peer+thread policy**
   - Input:
     - Instance: `routing_policy={include_peer: true, include_thread: true, include_group: false}`
     - Dimensions: `peer_id="user-42", thread_id="thread-abc"`
   - **Expected:**
     - `routing_key.PeerID` = `"user-42"`
     - `routing_key.ThreadID` = `"thread-abc"`
     - `routing_key.GroupID` = `""`

3. **Build routing key with group+thread policy**
   - Input:
     - Instance: `routing_policy={include_peer: false, include_thread: true, include_group: true}`
     - Dimensions: `group_id="channel-xyz", thread_id="thread-1"`
   - **Expected:**
     - `routing_key.PeerID` = `""`
     - `routing_key.ThreadID` = `"thread-1"`
     - `routing_key.GroupID` = `"channel-xyz"`

4. **Build routing key with all dimensions**
   - Input:
     - Instance: `routing_policy={include_peer: true, include_thread: true, include_group: true}`
     - Dimensions: `peer_id="user-1", thread_id="thread-1", group_id="group-1"`
   - **Expected:** All three routing dimensions populated in the key

5. **Build routing key with no optional dimensions**
   - Input:
     - Instance: `routing_policy={include_peer: false, include_thread: false, include_group: false}`
     - Dimensions: `peer_id="user-1"` (ignored)
   - **Expected:**
     - Key contains only `scope`, `workspace_id`, `bridge_instance_id`
     - All routing dimensions are empty

6. **Build routing key with workspace scope**
   - Input:
     - Instance: `scope=workspace, workspace_id="ws-001", routing_policy={include_peer: true}`
     - Dimensions: `peer_id="user-42"`
   - **Expected:**
     - `routing_key.Scope` = `"workspace"`
     - `routing_key.WorkspaceID` = `"ws-001"`
     - `routing_key.BridgeInstanceID` = instance ID
     - `routing_key.PeerID` = `"user-42"`

7. **Verify Serialize() produces stable JSON**
   - Input: Same routing key built twice with identical inputs
   - **Expected:** Both `Serialize()` calls return identical JSON strings

8. **Verify Hash() produces deterministic SHA-256**
   - Input: Same routing key
   - **Expected:** `Hash()` returns a 64-character hex string, identical on repeated calls

9. **Verify different routing keys produce different hashes**
   - Input: Two keys differing only in `peer_id`
   - **Expected:** Different hash values

10. **Verify routing key validation**
    - Input: Routing key with `bridge_instance_id: ""`
    - **Expected:** `Validate()` returns error: "routing key bridge instance id is required"

### Edge Cases & Variations

| Variation                                          | Input                                                                              | Expected Result                                                                |
| -------------------------------------------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Thread without peer or group (policy violation)    | `routing_policy={include_thread: true, include_peer: false, include_group: false}` | Validation error: "routing policy cannot include thread without peer or group" |
| Missing required peer_id when policy includes peer | Policy requires peer, dimensions have `peer_id=""`                                 | Validation error from `validateRoutingDimensions`                              |
| Whitespace in dimensions                           | `peer_id=" user-42 "`                                                              | Normalized to `"user-42"`                                                      |
| Unicode in peer_id                                 | `peer_id: "\u00e9mile"`                                                            | Preserved after normalization                                                  |
| Global scope with workspace_id in key              | Key with `scope=global, workspace_id="ws-1"`                                       | Validation error: "global scope cannot include workspace id"                   |

### Related Test Cases

- TC-FUNC-007 (inbound message uses routing key)
- TC-FUNC-009 (delivery events carry routing key)
- TC-FUNC-017 (delivery target resolution)
