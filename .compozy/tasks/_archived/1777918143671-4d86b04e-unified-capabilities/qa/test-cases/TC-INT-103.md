# TC-INT-103: Slack JSON Webhook To Ingest Flow

**Priority:** P0
**Type:** Integration
**Systems:** Slack Bridge, Webhook Guard, Bridge SDK, Host API
**API Endpoint:** Slack bridge webhook route

## Objective

Verify Slack Events API JSON payloads are signed, mapped, deduplicated, and ingested through the bridge host API.

## Preconditions

- Slack provider test runtime is initialized.
- Local Slack API fake responds to `auth.test`.
- Bridge instance has bot token, signing secret, and webhook route.

## Test Steps

1. Send signed `url_verification` JSON payload.
   **Expected:** Webhook returns 200 with JSON challenge.
2. Send signed message event payload.
   **Expected:** Message maps to inbound envelope with sender, group or peer, thread ID, content, metadata, and idempotency key.
3. Send the same event ID again.
   **Expected:** Dedup cache accepts only the first ingest; duplicate returns OK without host ingest.
4. Send reaction event payload.
   **Expected:** Reaction maps to inbound reaction envelope and is ingested.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Bot message | `bot_id` present | ignored with OK |
| Missing channel or timestamp | malformed event | 400 |
| Unsupported event type | unknown Slack event | OK and no ingest |

## Related

- TC-SEC-001
- TC-SEC-002
