# TC-NET-001: Network Routing and Audit

**Priority:** P0
**Type:** Integration / Regression
**Status:** Pass
**Created:** 2026-04-24

## Objective

Verify that local and remote network envelopes are validated, routed to the correct AGH sessions, and recorded in both audit and timeline persistence.

## Preconditions

- AGH network is enabled in the test configuration.
- At least two sessions have joined the same channel.
- Global DB and audit sink are available.

## Test Steps

1. Send a valid directed `direct` envelope to a local peer.
   **Expected:** exactly one delivery is produced for the target session.

2. Send a valid broadcast `say` envelope on the channel.
   **Expected:** all eligible local peers receive it; sender metadata is preserved.

3. Send `whois` and capability messages.
   **Expected:** peer registry and capability catalog are updated/responded to according to protocol.

4. Query audit and timeline records.
   **Expected:** accepted messages have `received`, completed prompts have `delivered`, outbound generated messages have `sent`.

5. Send duplicate, expired, unsupported, and invalid target variants.
   **Expected:** no local prompt delivery; rejection reason is auditable.

## Execution History

| Date       | Tester | Build | Result | Notes                                                                                                                                                                                                                                  |
| ---------- | ------ | ----- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-24 | Codex  | local | Pass   | Covered by full `make test-integration`, daemon network collaboration integration, network manager/router/audit tests, full runtime E2E, full web E2E Network route, and real AGH Codex ACP network smoke with `messages_delivered=1`. |
