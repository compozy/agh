## TC-FUNC-009: Delivery Event Ordering

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Validate that a delivery pipeline emits START, DELTA, DELTA, FINAL events in the correct order with monotonically increasing sequence numbers, correct `event_type` values, and proper `final` flag semantics.

### Preconditions
- [ ] A bridge instance exists in `status=ready` with `enabled=true`
- [ ] A prompt delivery registration has been created binding a session turn to the bridge instance
- [ ] The delivery broker is wired with a mock `DeliveryTransport` for capturing events
- [ ] Routing key and delivery target are resolved for the instance

### Test Steps
1. **Trigger a delivery with progressive streaming**
   - Input: Project four `DeliveryProjectionEvent`s in sequence:
     - Event 1: `type=start, text="Hello"`
     - Event 2: `type=delta, text="Hello, world"`
     - Event 3: `type=delta, text="Hello, world! How are you?"`
     - Event 4: `type=final, text="Hello, world! How are you? I'm a bridge bot."`
   - **Expected:** The mock transport receives exactly 4 `DeliveryEvent`s

2. **Verify START event (seq=0)**
   - **Expected:**
     - `event_type` = `"start"`
     - `seq` = `0`
     - `final` = `false`
     - `content.text` = `"Hello"`
     - `delivery_id` is non-empty and consistent across all events
     - `bridge_instance_id` matches the instance
     - `routing_key` matches the registered routing key
     - `operation` = `"post"` (default)
     - `reference` is `null` (post operation)
     - `error` is `null`
     - `resume` is `null`

3. **Verify first DELTA event (seq=1)**
   - **Expected:**
     - `event_type` = `"delta"`
     - `seq` = `1`
     - `final` = `false`
     - `content.text` = `"Hello, world"`
     - `delivery_id` same as START event

4. **Verify second DELTA event (seq=2)**
   - **Expected:**
     - `event_type` = `"delta"`
     - `seq` = `2`
     - `final` = `false`
     - `content.text` = `"Hello, world! How are you?"`

5. **Verify FINAL event (seq=3)**
   - **Expected:**
     - `event_type` = `"final"`
     - `seq` = `3`
     - `final` = `true`
     - `content.text` = `"Hello, world! How are you? I'm a bridge bot."`

6. **Verify monotonic sequence invariant**
   - **Expected:** For all events `e[i]` and `e[i+1]`: `e[i+1].seq > e[i].seq`

7. **Verify delivery_id consistency**
   - **Expected:** All 4 events share the same `delivery_id`

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| START with final=true | `event_type: "start", final: true` | Validation error: "delivery start event cannot be final" |
| DELTA with final=true | `event_type: "delta", final: true` | Validation error: "delivery delta event cannot be final" |
| FINAL with final=false | `event_type: "final", final: false` | Validation error: "delivery final event must set final=true" |
| Negative sequence number | `seq: -1` | Validation error: "invalid delivery event sequence" |
| Single-event delivery (START+FINAL) | Only one event with `event_type: "final"` | Valid if seq=0 and final=true |
| Missing event_type | `event_type: ""` | Validation error: "delivery event type is required" |
| Unknown event_type | `event_type: "append"` | Validation error: unsupported delivery event type |
| Missing delivery_id | `delivery_id: ""` | Validation error: "delivery event id is required" |
| Mismatched routing key instance | `routing_key.bridge_instance_id != bridge_instance_id` | Validation error: "delivery event bridge instance id must match routing key" |

### Related Test Cases
- TC-FUNC-010 (delivery acknowledgment after FINAL)
- TC-FUNC-011 (edit semantics)
- TC-FUNC-012 (delete semantics)
