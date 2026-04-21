## TC-INT-008: Capability catalogs survive session-to-network join plumbing

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/session`, `internal/network`
**Traceability:** Task 02; TechSpec data flow; shared-memory note that `session.NetworkPeerJoin` is the runtime/network join contract.
**Execution Surfaces:** Session activation, join lifecycle, network manager.
**Durable Regression Anchors:** `TestJoinNetworkPeerHandlesNoOpConditionsAndCapabilityProjection`, `TestManagerIntegrationCapabilityAwareJoinCarriesCatalogAcrossCreateResumeAndStop`, `TestPrepareJoinLocalPeerUsesCapabilityAwareRuntimeInput`

### Objective

Verify loaded capabilities travel through the session-owned join contract into local peer registration on create and resume, while preserving deterministic empty slices for no-catalog peers.

### Preconditions

- [ ] Temporary agent directory exists with a valid capability catalog.
- [ ] A channel-enabled session can be created and resumed through the normal manager path.
- [ ] The executor can inspect the join payload delivered to the network lifecycle.

### Test Steps

1. Start a session for an agent with a valid capability catalog on a non-empty channel.
   - **Expected:** Session activation reaches the join path exactly once and supplies the expected `session_id`, `peer_id`, `channel`, and capability payload.
2. Resume or restart the same session through the supported manager path.
   - **Expected:** Resume preserves the same capability-aware join payload semantics and does not regress to a default/no-capability local peer.
3. Repeat with a no-catalog agent as a control.
   - **Expected:** Join still occurs successfully, but the capability slice is deterministic and empty rather than `nil`.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Channel | any valid non-empty channel | Required to hit join logic |
| Catalog IDs | `review-pr`, `draft-spec` | Baseline join payload |
| Control fixture | no capability catalog | Empty-slice behavior |

### Post-conditions

- Sessions may be stopped and cleaned up.
- Evidence includes create and resume join payloads plus the no-catalog control.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Nil or blank join inputs | Missing session or channel | No-op behavior remains intact |
| Reconnect cycle | Join, leave, rejoin | Capability payload remains stable and does not duplicate |

### Related Test Cases

- `TC-INT-009`
- `TC-INT-011`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- This case is a gate for every downstream discovery claim because discovery begins from the join payload.
