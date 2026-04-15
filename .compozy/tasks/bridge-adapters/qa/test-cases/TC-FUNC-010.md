## TC-FUNC-010: Delivery Acknowledgment

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective
Validate that after a FINAL delivery event, the extension sends a `DeliveryAck` back to the daemon containing the `DeliveryID`, `Seq`, and `RemoteMessageID`, and that the ack is validated against the event it acknowledges.

### Preconditions
- [ ] A delivery pipeline is active with a START->DELTA->FINAL sequence
- [ ] The mock `DeliveryTransport` is configured to return `DeliveryAck` values
- [ ] The delivery broker is tracking the active delivery

### Test Steps
1. **Complete a delivery with FINAL event and receive ack**
   - Input: Send FINAL event:
     ```json
     {
       "delivery_id": "del-001",
       "bridge_instance_id": "inst-001",
       "seq": 3,
       "event_type": "final",
       "final": true,
       "content": {"text": "Complete response text"}
     }
     ```
   - Transport returns ack:
     ```json
     {
       "delivery_id": "del-001",
       "seq": 3,
       "remote_message_id": "slack-msg-ABC123"
     }
     ```
   - **Expected:**
     - `DeliveryAck.ValidateFor(event)` returns nil
     - `ack.DeliveryID` = `"del-001"` (matches event)
     - `ack.Seq` = `3` (matches event)
     - `ack.RemoteMessageID` = `"slack-msg-ABC123"` (platform-assigned ID)

2. **Verify ack for intermediate DELTA event**
   - Input: Send DELTA event with `seq=1`, transport returns ack with `seq=1`
   - **Expected:** `DeliveryAck.ValidateFor(event)` returns nil

3. **Verify ack with delivery_id mismatch is rejected**
   - Input: Event has `delivery_id: "del-001"`, ack has `delivery_id: "del-999"`
   - **Expected:** `ValidateFor` returns error: "delivery ack delivery id does not match event"

4. **Verify ack with seq mismatch is rejected**
   - Input: Event has `seq: 3`, ack has `seq: 2`
   - **Expected:** `ValidateFor` returns error: "delivery ack sequence does not match event"

5. **Verify ack with ReplaceRemoteMessageID for streaming updates**
   - Input: Transport returns ack with:
     ```json
     {
       "delivery_id": "del-001",
       "seq": 1,
       "remote_message_id": "slack-msg-DELTA1",
       "replace_remote_message_id": "slack-msg-START"
     }
     ```
   - **Expected:**
     - `ack.RemoteMessageID` = `"slack-msg-DELTA1"` (new message ID)
     - `ack.ReplaceRemoteMessageID` = `"slack-msg-START"` (previous message ID that was replaced)
     - Both IDs are preserved in the delivery snapshot

6. **Verify ack with empty optional fields is valid**
   - Input: Transport returns ack with only `delivery_id` and `seq` (no remote_message_id)
   - **Expected:** `ValidateFor` returns nil (RemoteMessageID is optional)

7. **Verify ack is normalized (whitespace trimmed)**
   - Input: Ack with `delivery_id: "  del-001  "`, `remote_message_id: " msg-1 "`
   - **Expected:** Normalized to `"del-001"` and `"msg-1"` before validation

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Ack with zero seq (matches start event) | `seq: 0` for START event with `seq: 0` | Valid |
| Ack with all empty fields | `{}` | Valid (empty ack, no fields to mismatch) |
| Ack delivery_id empty, event has one | `ack.delivery_id: ""` | Valid (empty ack field is not checked) |
| Ack seq=0 when event seq=5 | `ack.seq: 0`, `event.seq: 5` | Valid (zero seq is not checked — treated as unset) |

### Related Test Cases
- TC-FUNC-009 (delivery event ordering)
- TC-FUNC-011 (edit uses RemoteMessageID from ack)
- TC-FUNC-012 (delete uses RemoteMessageID from ack)
