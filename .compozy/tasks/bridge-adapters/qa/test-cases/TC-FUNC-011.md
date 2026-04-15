## TC-FUNC-011: Delivery Edit Semantics

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-15

---

### Objective
Validate that when a delivery includes a `ReplaceRemoteMessageID` in the acknowledgment, the daemon tracks the replacement mapping, and subsequent edit deliveries reference the correct prior `RemoteMessageID` through the `DeliveryMessageReference`.

### Preconditions
- [ ] A bridge instance exists in `status=ready`
- [ ] A prior delivery has completed with FINAL event
- [ ] The ack from the prior delivery returned `remote_message_id: "remote-msg-001"`
- [ ] The delivery snapshot retains `remote_message_id` and `replace_remote_message_id`

### Test Steps
1. **Complete initial delivery and capture remote message ID**
   - Input: START->DELTA->FINAL delivery, transport acks with `remote_message_id: "remote-msg-001"`
   - **Expected:** Delivery snapshot stores `remote_message_id: "remote-msg-001"`

2. **Send an edit delivery referencing the original message**
   - Input: New delivery event with:
     ```json
     {
       "delivery_id": "del-002",
       "bridge_instance_id": "inst-001",
       "seq": 0,
       "event_type": "start",
       "final": false,
       "operation": "edit",
       "reference": {
         "delivery_id": "del-001",
         "remote_message_id": "remote-msg-001"
       },
       "content": {"text": "Updated message text"}
     }
     ```
   - **Expected:**
     - `DeliveryEvent.Validate()` passes
     - `operation` = `"edit"`
     - `reference.delivery_id` = `"del-001"`
     - `reference.remote_message_id` = `"remote-msg-001"`
     - The extension receives the edit event and updates the original platform message

3. **Verify edit operation requires a reference**
   - Input: Delivery event with `operation: "edit"` but `reference: null`
   - **Expected:** Validation error: "delivery edit operation requires a reference"

4. **Verify post operation rejects a reference**
   - Input: Delivery event with `operation: "post"` and `reference: {...}`
   - **Expected:** Validation error: "delivery post operation cannot include a reference"

5. **Verify reference requires at least one identifier**
   - Input: `reference: {"delivery_id": "", "remote_message_id": ""}`
   - **Expected:** Validation error: "delivery reference requires delivery id or remote message id"

6. **Verify ReplaceRemoteMessageID tracking in ack**
   - Input: Edit delivery ack returns:
     ```json
     {
       "delivery_id": "del-002",
       "seq": 0,
       "remote_message_id": "remote-msg-002",
       "replace_remote_message_id": "remote-msg-001"
     }
     ```
   - **Expected:**
     - `replace_remote_message_id` = `"remote-msg-001"` (the message that was replaced)
     - `remote_message_id` = `"remote-msg-002"` (the new message ID after edit)
     - Future edits should reference `remote-msg-002`

7. **Verify chained edits track replacement chain**
   - Input: Third delivery with `reference.remote_message_id: "remote-msg-002"`, ack returns `remote_message_id: "remote-msg-003"`
   - **Expected:** The daemon tracks the full chain: `remote-msg-001 -> remote-msg-002 -> remote-msg-003`

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Edit with only delivery_id reference | `reference: {"delivery_id": "del-001"}` | Valid (remote_message_id is optional in reference) |
| Edit with only remote_message_id | `reference: {"remote_message_id": "remote-msg-001"}` | Valid |
| Edit non-existent delivery_id | `reference: {"delivery_id": "nonexistent"}` | Error or the extension handles the missing reference |
| Edit with FINAL event | `operation: "edit", event_type: "final", final: true` | Valid edit with final flag |

### Related Test Cases
- TC-FUNC-009 (initial delivery pipeline)
- TC-FUNC-010 (ack with RemoteMessageID)
- TC-FUNC-012 (delete semantics)
