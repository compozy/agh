## TC-INT-011: Delivery Coalescing for Slow Adapters

**Priority:** P2
**Type:** Integration
**Systems:** bridges.Broker, bridges.routeWorker, bridges.activeDelivery, bridges.deliveryQueueItem, bridges.DeliveryTransport, bridges.DeliveryProjectionEvent, bridges.DeliveryEvent
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective

Validate the delivery broker's coalescing behavior when a provider processes deliveries slowly. When rapid DELTA events arrive while a previous delivery call is in-flight, the broker should coalesce intermediate DELTAs into the latest content, drop stale pending DELTAs when the queue is saturated, and still deliver the FINAL event correctly with the full accumulated content. Confirms bounded queue behavior, the `dropQueuedDeltaLocked` eviction, and delivery metrics tracking.

### Preconditions

- [ ] Broker constructed with `WithDeliveryBrokerQueueCapacity(3)` (tight bound for testing coalescing)
- [ ] Broker constructed with `WithDeliveryBrokerRetryDelay(10ms)` for fast iteration
- [ ] A mock `DeliveryTransport` that adds an artificial 200ms delay to `DeliverBridge` calls (simulating a slow adapter)
- [ ] 1 bridge instance registered (`brg-coal-1`, scope=global, platform=telegram)
- [ ] A delivery registered for `brg-coal-1`, session `sess-1`, turn `turn-1`

### Test Steps

1. **Project a rapid burst of agent_message events**
   - Input: Project 10 `DeliveryProjectionEvent{Type: "agent_message", ...}` events in rapid succession with cumulative text chunks: "A", "B", "C", "D", "E", "F", "G", "H", "I", "J"
   - **Expected:** Broker creates START for the first event, then DELTA events for subsequent ones; due to queue capacity=3, some DELTAs are coalesced (pendingDelta updated in-place) or older DELTAs are evicted via `dropQueuedDeltaLocked`

2. **Verify the slow transport receives START**
   - Input: Wait for the first `DeliverBridge` call (200ms delay)
   - **Expected:** Transport receives `DeliveryRequest` with `event_type=start`, `content.text` starting with "A" (initial content)

3. **Verify intermediate DELTAs are coalesced**
   - Input: After START completes, observe the next `DeliverBridge` call
   - **Expected:** The DELTA event carries the latest cumulative content (e.g., "ABCDEFGHIJ"), not an intermediate partial. Multiple intermediate DELTAs were coalesced into one

4. **Verify delivery metrics track drops**
   - Input: Call `broker.DeliveryMetrics()`
   - **Expected:** Metrics for `brg-coal-1` show `delivery_dropped_total > 0` with `delivery_dropped_by_reason["coalesced"] > 0`

5. **Project a FINAL event while DELTAs are still queued**
   - Input: Project `DeliveryProjectionEvent{Type: "done", TurnID: "turn-1"}`
   - **Expected:** Broker removes any queued DELTA items from the route queue (via `removeQueuedSlotLocked`), queues a TERMINAL event

6. **Verify the FINAL delivery contains complete content**
   - Input: Wait for the FINAL `DeliverBridge` call
   - **Expected:** `event_type=final`, `final=true`, `content.text="ABCDEFGHIJ"` (full accumulated content, nothing lost despite coalescing)

7. **Verify the delivery is cleaned up after FINAL ack**
   - Input: Attempt `broker.Snapshot(ctx, deliveryID)`
   - **Expected:** Returns `ErrDeliveryNotFound` (delivery removed after final ack with no queued items)

8. **Verify no content gaps in the delivered sequence**
   - Input: Record all `DeliveryRequest` payloads received by the transport in order
   - **Expected:** START content is a prefix of FINAL content; any DELTA content is a prefix of FINAL content; content is monotonically growing (no reversed or missing characters)

### Data Validation

| Field                   | Source Value                      | Transformed Value                          | Status |
| ----------------------- | --------------------------------- | ------------------------------------------ | ------ |
| 10 rapid agent_messages | "A" through "J"                   | Cumulative: "ABCDEFGHIJ"                   |        |
| Queue capacity          | 3                                 | At most 3 items in route.queue at any time |        |
| Coalesced DELTA         | Multiple pendingDelta updates     | Single DeliveryEvent with latest content   |        |
| Dropped DELTAs          | dropQueuedDeltaLocked eviction    | metrics.droppedByReason["coalesced"] > 0   |        |
| FINAL content           | Full accumulated text             | "ABCDEFGHIJ" (complete, no gaps)           |        |
| DeliveryBacklog metric  | len(route.queue) at snapshot time | >= 0, bounded by capacity                  |        |

### Error Scenarios

- [ ] Queue saturated and no DELTA to evict (only START + TERMINAL): returns `ErrDeliveryQueueSaturated`
- [ ] Transport returns error for every call: broker retries with RESUME, delivery metrics record failures
- [ ] Provider crashes during a slow DELTA delivery: broker schedules RESUME after transport error
- [ ] FINAL arrives before START is sent (extremely fast agent, extremely slow transport): START is coalesced into the content, FINAL carries full text
- [ ] Multiple concurrent deliveries on the same route worker: queue interleaves events from different delivery IDs
- [ ] metrics.DeliveryFailuresTotal incremented on each delivery error event projected

### Related Test Cases

- TC-INT-004 (normal delivery without coalescing)
- TC-INT-005 (delivery recovery uses the same broker queue infrastructure)
