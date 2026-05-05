# TC-FUNC-009: Network Send Request Validation

**Priority:** P0
**Type:** Functional
**Module:** API Core + Network Runtime
**Requirement:** Send requests must validate required fields and preserve wire metadata.

## Objective

Verify `NetworkSendRequestFromPayload` and `POST /api/network/send` reject malformed input and pass valid metadata to the runtime.

## Preconditions

- Network is enabled.
- Runtime send service can capture `network.SendRequest`.
- Test payloads include body, ext, explicit ID, trace, causation, reply, interaction, and expiry metadata.

## Test Steps

1. Submit a valid `say` request with body and extension metadata.
   **Expected:** Runtime receives trimmed session, channel, kind, body clone, ext clone, and optional fields.
2. Submit empty body.
   **Expected:** 400 `body is required`.
3. Submit malformed JSON body.
   **Expected:** 400 `body must be valid JSON`.
4. Runtime returns target peer not found.
   **Expected:** API maps to 404.
5. Runtime returns duplicate or invalid-field error.
   **Expected:** API maps duplicate/invalid to the contract-defined status.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Whitespace optional values | `" trace "` | trimmed in runtime request |
| Explicit message ID | `id="msg-1"` | preserved |
| Nested extension payload | object/number/string values | JSON raw messages preserved |

## Related

- SMOKE-003
- TC-INT-006
