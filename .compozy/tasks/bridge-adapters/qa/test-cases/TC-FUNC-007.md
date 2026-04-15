## TC-FUNC-007: Inbound Message Ingestion

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Validate that an inbound text message submitted through the Host API `messages/ingest` method with a valid `bridge_instance_id` and routing dimensions is correctly validated, deduplicated, and dispatched to the daemon's routing layer.

### Preconditions
- [ ] Daemon is running with a bridge instance in `status=ready`
- [ ] Bridge instance ID is known (e.g., `inst-001`)
- [ ] Instance has `scope=global`, `routing_policy: {include_peer: true, include_thread: true, include_group: false}`
- [ ] No prior inbound messages with the same idempotency key exist

### Test Steps
1. **Submit a valid inbound message envelope**
   - Input (Host API `messages/ingest`):
     ```json
     {
       "bridge_instance_id": "inst-001",
       "scope": "global",
       "peer_id": "user-42",
       "thread_id": "thread-abc",
       "platform_message_id": "plat-msg-001",
       "received_at": "2026-04-15T10:00:00Z",
       "sender": {
         "id": "user-42",
         "username": "alice",
         "display_name": "Alice Smith"
       },
       "content": {"text": "Hello from Telegram"},
       "event_family": "message",
       "idempotency_key": "tg-msg-001-inst-001"
     }
     ```
   - **Expected:**
     - Ingest call succeeds (no error returned)
     - `InboundMessageEnvelope.Validate()` passes
     - Routing key is built from `scope=global`, `bridge_instance_id=inst-001`, `peer_id=user-42`, `thread_id=thread-abc`
     - The daemon dispatches the message to the appropriate ACP session

2. **Verify sender metadata is preserved**
   - Input: Same as step 1
   - **Expected:** The routed event retains `sender.id`, `sender.username`, `sender.display_name` without mutation

3. **Verify message with attachments**
   - Input: Same base message plus:
     ```json
     "attachments": [
       {"id": "att-1", "name": "photo.jpg", "mime_type": "image/jpeg", "url": "https://cdn.example.com/photo.jpg"}
     ]
     ```
   - **Expected:** Attachment is preserved in the routed event with all fields intact

4. **Verify idempotency dedup**
   - Input: Submit the same message with `idempotency_key: "tg-msg-001-inst-001"` a second time
   - **Expected:** Second submission is deduplicated (no duplicate routing); either silently accepted or returns a dedup indicator

5. **Verify message with provider_metadata**
   - Input: Same base message plus `"provider_metadata": {"telegram_chat_id": 12345}`
   - **Expected:** `provider_metadata` is preserved as valid JSON in the envelope

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Missing bridge_instance_id | `bridge_instance_id: ""` | Validation error: "inbound message bridge instance id is required" |
| Missing scope | `scope: ""` | Validation error: "scope is required" |
| Missing received_at | `received_at: zero` | Validation error: "inbound message received at is required" |
| Missing idempotency_key | `idempotency_key: ""` | Validation error: "inbound message idempotency key is required" |
| Missing event_family (defaults to message) | `event_family: ""`, no command/action/reaction | Normalizes to `"message"` |
| Instance is disabled | Bridge instance in `status=disabled` | Rejected with `ErrBridgeInstanceUnavailable` |
| Instance does not exist | `bridge_instance_id: "nonexistent"` | Error: `ErrBridgeInstanceNotFound` |
| Invalid provider_metadata JSON | `provider_metadata: "not-json{"` | Validation error: must be valid JSON |
| Message family with command payload | `event_family: "message"` + `command: {...}` | Validation error: "inbound message family cannot include command, action, or reaction payloads" |
| Workspace-scoped message without workspace_id | `scope: "workspace", workspace_id: ""` | Validation error: "workspace scope requires workspace id" |
| Empty content text | `content: {"text": ""}` | Accepted (text may be empty if attachments present) |

### Related Test Cases
- TC-FUNC-008 (typed interaction events)
- TC-FUNC-009 (delivery ordering after ingestion)
- TC-FUNC-016 (routing key construction)
