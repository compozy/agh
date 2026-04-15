## TC-INT-003: Multi-Instance Routing Isolation

**Priority:** P0
**Type:** Integration
**Systems:** bridges.RoutingKey, bridges.BridgeRoute, bridges.BuildRoutingKey, bridges.RoutingPolicy, extension.HostAPI (bridges/messages/ingest), bridgesdk.HostAPIClient, store/globaldb
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-15

---

### Objective

Validate that two bridge instances under the same provider with different routing policies produce distinct routing keys, route inbound events to separate sessions, and prevent cross-instance message leakage. Confirms that the `RoutingPolicy` dimensions (IncludePeer, IncludeThread, IncludeGroup) participate correctly in `BuildRoutingKey` and that the SHA-256 routing key hash differentiates routes.

### Preconditions

- [ ] Provider runtime initialized with 2 bridge instances:
  - `brg-iso-1`: scope=global, platform=slack, routing_policy={IncludePeer:true, IncludeThread:false, IncludeGroup:false}
  - `brg-iso-2`: scope=global, platform=slack, routing_policy={IncludePeer:true, IncludeThread:true, IncludeGroup:false}
- [ ] globaldb bridge_routes table is empty
- [ ] Session manager is configured to create new sessions on route miss

### Test Steps

1. **Send inbound message for instance 1 from peer-A**
   - Input: InboundMessageEnvelope{bridge_instance_id: `brg-iso-1`, peer_id: `peer-A`, thread_id: `thread-1`, event_family: message, content.text: `hello from iso-1`, idempotency_key: `iso1-msg1`, platform_message_id: `pmsg-1`}
   - **Expected:** Ingest result returns `route_created=true`, `session_id=sess-1`, routing_key contains `peer_id=peer-A` but no `thread_id` (excluded by policy)

2. **Send inbound message for instance 2 from peer-A in thread-1**
   - Input: InboundMessageEnvelope{bridge_instance_id: `brg-iso-2`, peer_id: `peer-A`, thread_id: `thread-1`, event_family: message, content.text: `hello from iso-2`, idempotency_key: `iso2-msg1`, platform_message_id: `pmsg-2`}
   - **Expected:** Ingest result returns `route_created=true`, `session_id=sess-2` (different session), routing_key contains both `peer_id=peer-A` AND `thread_id=thread-1`

3. **Verify routing key hashes are distinct**
   - Input: Compute `RoutingKey.Hash()` for both returned routing keys
   - **Expected:** Hash values are different because instance 2 includes thread_id while instance 1 does not

4. **Send a second message for instance 2 from peer-A in thread-1**
   - Input: InboundMessageEnvelope{bridge_instance_id: `brg-iso-2`, peer_id: `peer-A`, thread_id: `thread-1`, event_family: message, content.text: `followup iso-2`, idempotency_key: `iso2-msg2`, platform_message_id: `pmsg-3`}
   - **Expected:** `route_created=false`, `session_id=sess-2` (reuses existing route)

5. **Send a message for instance 2 from peer-A in thread-2 (different thread)**
   - Input: InboundMessageEnvelope{bridge_instance_id: `brg-iso-2`, peer_id: `peer-A`, thread_id: `thread-2`, event_family: message, content.text: `new thread`, idempotency_key: `iso2-msg3`, platform_message_id: `pmsg-4`}
   - **Expected:** `route_created=true`, `session_id=sess-3` (different thread creates a new route since instance 2 includes thread in routing)

6. **Verify no cross-instance leakage in persisted routes**
   - Input: Query globaldb bridge_routes for `bridge_instance_id=brg-iso-1`
   - **Expected:** Exactly 1 route row with `peer_id=peer-A`, `thread_id=""`, `session_id=sess-1`
   - Input: Query globaldb bridge_routes for `bridge_instance_id=brg-iso-2`
   - **Expected:** Exactly 2 route rows: one for thread-1/sess-2, one for thread-2/sess-3

7. **Verify CanonicalizeRoutingKey strips excluded dimensions**
   - Input: Call `CanonicalizeRoutingKey(brg-iso-1, RoutingKey{..., peer_id: "peer-A", thread_id: "thread-1"})`
   - **Expected:** Returned key has `thread_id=""` because instance 1 policy excludes thread

### Data Validation

| Field                           | Source Value                        | Transformed Value                      | Status |
| ------------------------------- | ----------------------------------- | -------------------------------------- | ------ |
| brg-iso-1 RoutingKey.PeerID     | `peer-A`                            | included (IncludePeer=true)            |        |
| brg-iso-1 RoutingKey.ThreadID   | `thread-1`                          | stripped to `""` (IncludeThread=false) |        |
| brg-iso-2 RoutingKey.PeerID     | `peer-A`                            | included (IncludePeer=true)            |        |
| brg-iso-2 RoutingKey.ThreadID   | `thread-1`                          | included (IncludeThread=true)          |        |
| brg-iso-1 route hash            | SHA-256(scope+instance+peer)        | distinct from brg-iso-2 hash           |        |
| brg-iso-2 route hash (thread-1) | SHA-256(scope+instance+peer+thread) | distinct from thread-2 hash            |        |

### Error Scenarios

- [ ] RoutingPolicy with IncludeThread=true but IncludePeer=false and IncludeGroup=false: BuildRoutingKey returns validation error ("routing policy cannot include thread without peer or group")
- [ ] Inbound message with empty peer_id when IncludePeer=true: routing still works (peer dimension is empty string in key), but results in a single-instance shared route
- [ ] Instance not found in registry during ingest: Host API returns `ErrBridgeInstanceNotFound`
- [ ] Instance exists but status is `auth_required` or `disabled`: Host API returns `ErrBridgeInstanceUnavailable`

### Related Test Cases

- TC-INT-001 (instances must be launched before routing can occur)
- TC-INT-002 (webhook ingress feeds into the same ingest path)
- TC-INT-004 (delivery broker uses the same routing keys for outbound delivery)
