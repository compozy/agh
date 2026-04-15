## TC-FUNC-008: Inbound Typed Interaction Events

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Validate that command, action, and reaction inbound events are ingested through the bridge protocol as typed events in the `InboundMessageEnvelope`, preserving their structured payloads rather than hiding them behind opaque metadata blobs.

### Preconditions
- [ ] Daemon is running with a bridge instance in `status=ready` (e.g., Slack which supports all three interaction types)
- [ ] Bridge instance ID is known (e.g., `inst-slack-001`)
- [ ] Instance has `scope=global`

### Test Steps
1. **Ingest an inbound command event**
   - Input:
     ```json
     {
       "bridge_instance_id": "inst-slack-001",
       "scope": "global",
       "peer_id": "user-10",
       "received_at": "2026-04-15T10:05:00Z",
       "sender": {"id": "user-10", "username": "bob"},
       "event_family": "command",
       "command": {
         "command": "/deploy",
         "text": "production v2.1.0",
         "trigger_id": "trigger-abc"
       },
       "idempotency_key": "slack-cmd-001"
     }
     ```
   - **Expected:**
     - `InboundMessageEnvelope.Validate()` passes
     - `event_family` = `"command"`
     - `command.command` = `"/deploy"`
     - `command.text` = `"production v2.1.0"`
     - `command.trigger_id` = `"trigger-abc"`
     - `content.text` must be empty (command family excludes message payload fields)
     - `platform_message_id` must be empty
     - `action` and `reaction` must be `null`
     - `attachments` must be empty

2. **Ingest an inbound action event**
   - Input:
     ```json
     {
       "bridge_instance_id": "inst-slack-001",
       "scope": "global",
       "peer_id": "user-10",
       "received_at": "2026-04-15T10:06:00Z",
       "sender": {"id": "user-10", "username": "bob"},
       "event_family": "action",
       "action": {
         "action_id": "approve_deploy",
         "message_id": "msg-999",
         "value": "approved",
         "trigger_id": "trigger-def"
       },
       "idempotency_key": "slack-action-001"
     }
     ```
   - **Expected:**
     - `event_family` = `"action"`
     - `action.action_id` = `"approve_deploy"`
     - `action.message_id` = `"msg-999"`
     - `action.value` = `"approved"`
     - `command` and `reaction` must be `null`
     - `content.text`, `platform_message_id`, `attachments` must be empty

3. **Ingest an inbound reaction event**
   - Input:
     ```json
     {
       "bridge_instance_id": "inst-slack-001",
       "scope": "global",
       "peer_id": "user-10",
       "received_at": "2026-04-15T10:07:00Z",
       "sender": {"id": "user-10", "username": "bob"},
       "event_family": "reaction",
       "reaction": {
         "message_id": "msg-500",
         "emoji": "thumbsup",
         "raw_emoji": "\ud83d\udc4d",
         "added": true
       },
       "idempotency_key": "slack-reaction-001"
     }
     ```
   - **Expected:**
     - `event_family` = `"reaction"`
     - `reaction.message_id` = `"msg-500"`
     - `reaction.emoji` = `"thumbsup"`
     - `reaction.added` = `true`
     - `command` and `action` must be `null`

4. **Verify reaction removal event**
   - Input: Same as step 3 but with `"added": false`
   - **Expected:** `reaction.added` = `false` (reaction removed)

5. **Verify typed events are not demoted to provider_metadata**
   - Input: Inspect the routed event downstream
   - **Expected:** The `command`, `action`, or `reaction` fields are first-class typed fields in the envelope, not hidden inside `provider_metadata`

### Edge Cases & Variations
| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Command family missing command payload | `event_family: "command"`, `command: null` | Validation error: "inbound command family requires command payload" |
| Command family with action payload | `event_family: "command"`, `action: {...}` | Validation error: "inbound command family cannot include action or reaction payloads" |
| Action family missing action payload | `event_family: "action"`, `action: null` | Validation error: "inbound action family requires action payload" |
| Reaction family missing reaction payload | `event_family: "reaction"`, `reaction: null` | Validation error: "inbound reaction family requires reaction payload" |
| Command with empty command string | `command: {"command": ""}` | Validation error: "inbound command is required" |
| Action with empty action_id | `action: {"action_id": ""}` | Validation error: "inbound action id is required" |
| Reaction with empty message_id | `reaction: {"message_id": "", "emoji": "ok"}` | Validation error: "inbound reaction message id is required" |
| Reaction with empty emoji | `reaction: {"message_id": "m1", "emoji": ""}` | Validation error: "inbound reaction emoji is required" |
| Unsupported event_family | `event_family: "modal"` | Validation error: unsupported inbound event family |
| Command family with content.text | `event_family: "command"`, `content: {"text": "hello"}` | Validation error: "inbound command family cannot include message payload fields" |

### Related Test Cases
- TC-FUNC-007 (standard message ingestion)
- TC-FUNC-016 (routing key construction applies to all families)
