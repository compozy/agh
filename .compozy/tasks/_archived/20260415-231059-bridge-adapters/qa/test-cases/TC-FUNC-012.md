## TC-FUNC-012: Delivery Delete Semantics

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-15

---

### Objective

Validate that a DELETE delivery event correctly references a previously delivered message via `RemoteMessageID`, uses the `delete` operation, carries no content text, and that the extension receives the delete instruction.

### Preconditions

- [ ] A bridge instance exists in `status=ready`
- [ ] A prior delivery has completed with a FINAL ack containing `remote_message_id: "remote-msg-001"`
- [ ] The delivery broker supports DELETE event type

### Test Steps

1. **Send a DELETE delivery event referencing a prior message**
   - Input:
     ```json
     {
       "delivery_id": "del-003",
       "bridge_instance_id": "inst-001",
       "routing_key": { "scope": "global", "bridge_instance_id": "inst-001", "peer_id": "user-42" },
       "delivery_target": {
         "bridge_instance_id": "inst-001",
         "peer_id": "user-42",
         "mode": "direct-send"
       },
       "seq": 0,
       "event_type": "delete",
       "final": true,
       "operation": "delete",
       "reference": {
         "delivery_id": "del-001",
         "remote_message_id": "remote-msg-001"
       },
       "content": { "text": "" }
     }
     ```
   - **Expected:**
     - `DeliveryEvent.Validate()` passes
     - `event_type` = `"delete"`
     - `final` = `true` (delete events must be final)
     - `operation` = `"delete"`
     - `reference.remote_message_id` = `"remote-msg-001"`
     - `content.text` is empty

2. **Verify delete event must be final**
   - Input: `event_type: "delete", final: false`
   - **Expected:** Validation error: "delivery delete event must set final=true"

3. **Verify delete event must use delete operation**
   - Input: `event_type: "delete", operation: "post"`
   - **Expected:** Validation error: "delete delivery events must use delete operation"

4. **Verify delete operation requires delete event type**
   - Input: `operation: "delete", event_type: "start"`
   - **Expected:** Validation error: "delete operation requires delete event type"

5. **Verify delete event rejects content text**
   - Input: `event_type: "delete"` with `content: {"text": "some text"}`
   - **Expected:** Validation error: "delivery delete events cannot include message content"

6. **Verify delete event rejects error payload**
   - Input: `event_type: "delete"` with `error: {"message": "oops"}`
   - **Expected:** Validation error: "delivery delete events cannot include error or resume payloads"

7. **Verify delete event rejects resume payload**
   - Input: `event_type: "delete"` with `resume: {"latest_event_type": "delta"}`
   - **Expected:** Validation error: "delivery delete events cannot include error or resume payloads"

8. **Verify delete operation requires a reference**
   - Input: `operation: "delete", reference: null`
   - **Expected:** Validation error: "delivery delete operation requires a reference"

### Edge Cases & Variations

| Variation                                 | Input                                                     | Expected Result                                      |
| ----------------------------------------- | --------------------------------------------------------- | ---------------------------------------------------- |
| Delete with only delivery_id in reference | `reference: {"delivery_id": "del-001"}`                   | Valid (remote_message_id is optional in reference)   |
| Delete with only remote_message_id        | `reference: {"remote_message_id": "remote-msg-001"}`      | Valid                                                |
| Delete with empty reference               | `reference: {"delivery_id": "", "remote_message_id": ""}` | Validation error: reference requires at least one ID |
| Delete a message that was already deleted | Double delete                                             | Extension handles gracefully (platform-dependent)    |
| Delete with provider_metadata             | `provider_metadata: {"reason": "moderation"}`             | Valid; metadata preserved                            |

### Related Test Cases

- TC-FUNC-009 (delivery event ordering)
- TC-FUNC-010 (delivery ack returns RemoteMessageID)
- TC-FUNC-011 (edit semantics — edit vs delete operations)
