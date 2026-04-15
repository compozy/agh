## TC-INT-004: Delivery End-to-End Through Provider

**Priority:** P1
**Type:** Integration
**Systems:** bridges.Broker, bridges.DeliveryTransport, bridgesdk.Runtime (bridges/deliver handler), bridges.DeliveryRequest, bridges.DeliveryAck, extension.Manager (DeliverBridge), bridges.DeliveryProjectionEvent
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective
Validate the full outbound delivery path: the daemon projects session agent output into a `DeliveryEvent` stream (START, DELTA, FINAL), the broker enqueues events on the per-route worker, sends `bridges/deliver` JSON-RPC requests to the provider extension, the provider invokes its platform API (mocked), and returns a `DeliveryAck` with `RemoteMessageID`. Confirms progressive delivery sequencing and the ack validation contract.

### Preconditions
- [ ] Provider runtime is initialized with 1 bridge instance (`brg-del-1`, scope=global, platform=telegram, routing_policy={IncludePeer:true})
- [ ] A route exists for `brg-del-1` + `peer_id=peer-A` -> `session_id=sess-1`
- [ ] Broker is constructed with a `DeliveryTransport` wired to the extension manager's `DeliverBridge` method
- [ ] Provider's `DeliveryHandler` calls a mock platform API and returns acks with `RemoteMessageID`
- [ ] Mock platform API server is running at a known test URL

### Test Steps
1. **Register a prompt delivery for the session turn**
   - Input: `PromptDeliveryRegistration{SessionID: "sess-1", TurnID: "turn-1", ExtensionName: "telegram-adapter", RoutingKey: {scope: global, bridge_instance_id: "brg-del-1", peer_id: "peer-A"}, DeliveryTarget: {bridge_instance_id: "brg-del-1", peer_id: "peer-A", mode: "direct-send"}}`
   - **Expected:** Returns a `DeliverySnapshot` with `delivery_id` set, `latest_seq=0`, `final=false`

2. **Project a START event from agent output**
   - Input: `DeliveryProjectionEvent{Type: "agent_message", TurnID: "turn-1", Text: "Hello "}`
   - **Expected:** Broker projects a `DeliveryEvent` with `event_type=start`, `seq=1`, `content.text="Hello "`, `final=false`

3. **Project DELTA events from streaming agent output**
   - Input: `DeliveryProjectionEvent{Type: "agent_message", TurnID: "turn-1", Text: "world! "}` then `DeliveryProjectionEvent{Type: "agent_message", TurnID: "turn-1", Text: "How are you?"}`
   - **Expected:** Broker projects `event_type=delta` events with cumulative `content.text`: `"Hello world! "` then `"Hello world! How are you?"`; `seq` increments

4. **Project a FINAL event on agent completion**
   - Input: `DeliveryProjectionEvent{Type: "done", TurnID: "turn-1"}`
   - **Expected:** Broker projects a `DeliveryEvent` with `event_type=final`, `final=true`, `content.text="Hello world! How are you?"`

5. **Verify the broker sends bridges/deliver requests to the provider**
   - Input: Wait for delivery transport calls (allow up to 5s for background worker)
   - **Expected:** At least 2 `DeliveryRequest` calls received: one with `event_type=start`, one with `event_type=final`; intermediate deltas may be coalesced

6. **Verify provider receives correct DeliveryRequest structure**
   - Input: Inspect captured `DeliveryRequest` at the provider's `DeliveryHandler`
   - **Expected:** `event.delivery_id` matches the registered delivery; `event.bridge_instance_id` = `brg-del-1`; `event.routing_key` matches the registration; `event.delivery_target.peer_id` = `peer-A`; `event.delivery_target.mode` = `direct-send`

7. **Verify provider returns valid DeliveryAck**
   - Input: Inspect acks returned to the broker
   - **Expected:** Each ack has `delivery_id` matching the request (or empty); `seq` matches the request event seq (or zero); START ack includes `remote_message_id` = the mock platform's message ID; FINAL ack may include `replace_remote_message_id`

8. **Verify broker snapshot after FINAL delivery**
   - Input: `broker.Snapshot(ctx, deliveryID)`
   - **Expected:** `latest_event_type=final`, `final=true`, `last_acked_seq >= latest_seq`, `remote_message_id` is non-empty

### Data Validation
| Field | Source Value | Transformed Value | Status |
|-------|------------|-------------------|--------|
| DeliveryProjectionEvent.Type=agent_message | Session agent_message event | DeliveryEvent.EventType=start (first) or delta | |
| DeliveryProjectionEvent.Type=done | Session done event | DeliveryEvent.EventType=final, Final=true | |
| Cumulative content | "Hello " + "world! " + "How are you?" | DeliveryEvent.Content.Text="Hello world! How are you?" | |
| DeliveryEvent.Seq | Auto-incremented by broker | Monotonically increasing per delivery | |
| DeliveryAck.RemoteMessageID | Mock platform response | Stored in broker delivery state | |
| DeliveryEvent.Operation | Default | DeliveryOperationPost | |

### Error Scenarios
- [ ] Provider DeliveryHandler returns an error: broker retries with RESUME after `retryDelay`
- [ ] DeliveryAck.DeliveryID mismatches the event: `ValidateFor` returns error, broker treats as send failure
- [ ] DeliveryAck.Seq mismatches the event: `ValidateFor` returns error
- [ ] DeliveryTransport is nil (no extension connected): broker returns `ErrDeliveryTransportUnavailable`, retries with RESUME
- [ ] Broker lifecycle context cancelled: all route workers drain and stop
- [ ] Queue saturated (> queueCapacity items): broker returns `ErrDeliveryQueueSaturated` or coalesces deltas

### Related Test Cases
- TC-INT-001 (provider must be launched before delivery)
- TC-INT-005 (delivery recovery after provider restart uses RESUME)
- TC-INT-011 (delivery coalescing under pressure)
