# TC-INT-104: Slack Form Command And Block Action Flow

**Priority:** P1
**Type:** Integration
**Systems:** Slack Bridge, Form Webhook, Bridge SDK Host API
**API Endpoint:** Slack bridge webhook route

## Objective

Verify Slack slash commands and interactive block actions are mapped to command/action envelopes and honor DM policies.

## Preconditions

- Slack provider route is active with signing secret.
- Form content type is allowed.
- Host API ingest fixture captures envelopes.

## Test Steps

1. POST a signed slash command form body.
   **Expected:** Command envelope includes command, text, trigger ID, sender identity, group or peer target, and idempotency key.
2. POST a signed block action payload.
   **Expected:** Action envelope includes action ID, block ID, value, message ID, response URL metadata, and correct thread ID.
3. Repeat the action with same action timestamp.
   **Expected:** Dedup suppresses duplicate ingest.
4. Configure direct-message allowlist and send a disallowed direct form event.
   **Expected:** Webhook returns OK and does not ingest.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Missing payload | form without command or payload | 400 |
| Non-block action payload | `type != block_actions` | OK, no ingest |
| Pairing policy | allowed username/user ID | accepted |

## Related

- TC-SEC-002
- TC-SEC-004
