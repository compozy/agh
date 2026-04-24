# TC-SEC-004: Slack Secret Handling And DM Authorization

**Priority:** P1
**Type:** Security
**Module:** Slack Bridge
**Requirement:** Secrets must stay out of responses/log artifacts and direct-message policy must be enforced.

## Objective

Verify Slack bot token and signing secret are required, are not leaked through state markers or API errors, and DM allowlist/pairing policies are honored.

## Preconditions

- Slack provider initialized with fake secret bindings.
- State, ownership, delivery, and ingest marker files can be inspected.
- DM policy variants are available.

## Test Steps

1. Initialize provider without `bot_token`.
   **Expected:** Instance reports auth required with a non-secret error.
2. Initialize provider without `signing_secret`.
   **Expected:** Instance reports auth required and webhook route is not usable without signature verification.
3. Inspect marker files after initialize, ingest, and delivery.
   **Expected:** Marker files do not include token or signing secret values.
4. Configure DM allowlist and send direct event from disallowed user.
   **Expected:** Webhook returns OK and skips ingest.
5. Configure paired user and send direct event from allowed user.
   **Expected:** Event is ingested.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Username case | mixed-case username | normalized matching |
| User ID case | lower-case user ID | normalized Slack user ID |
| Non-direct event | channel event under strict DM policy | allowed |

## Related

- TC-INT-104
- TC-INT-005
