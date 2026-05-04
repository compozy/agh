# TC-INT-006: CLI And UDS Network Contract Parity

**Priority:** P1
**Type:** Integration
**Systems:** CLI, UDS API, API Core
**API Endpoint:** `/api/network/*`

## Objective

Verify CLI network commands use the same shared contracts as UDS/HTTP handlers and preserve metadata in both directions.

## Preconditions

- CLI can be exercised with a stub UDS client or local daemon.
- UDS handlers are registered for network endpoints.
- Output formats human, JSON, and TOON are available.

## Test Steps

1. Run `agh network status -o json`.
   **Expected:** JSON decodes to the shared status payload and includes kind metrics.
2. Run `agh network peers builders -o json`.
   **Expected:** Query includes `channel=builders` and response decodes to peer payloads.
3. Run `agh network send` with body, ext, expiry, trace, causation, interaction, and reply metadata.
   **Expected:** UDS request body preserves metadata and response renders it.
4. Run `agh network inbox --session sess-a -o human`.
   **Expected:** Output contains channel, workflow, and handoff metadata.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Invalid body JSON | `--body not-json` | CLI rejects before API call |
| Non-object ext | `--ext []` | CLI rejects |
| RFC3339 expiry | timestamp string | converted to unix seconds |

## Related

- SMOKE-001
- TC-FUNC-009
- TC-FUNC-010
