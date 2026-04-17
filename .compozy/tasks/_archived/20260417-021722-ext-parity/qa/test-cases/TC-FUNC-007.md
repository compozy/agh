# TC-FUNC-007: Codec decode failure rejects invalid payload

**Priority:** P1
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 02

## Objective

Validate that the typed codec layer rejects malformed or structurally invalid payloads before they reach the persistence layer. When a codec is registered for a resource kind and a `PutRaw` (or equivalent) is called with a payload that fails decoding, the operation must be rejected with a clear validation error. No invalid data should be persisted to SQLite.

## Preconditions

- A fresh resource store is initialized with schema applied.
- A codec for `tool` kind is registered that expects a JSON object with required fields (e.g., `{"name": string, "description": string}`).
- A valid `MutationActor` is configured.

## Test Steps

1. Call `PutRaw` with `Kind="tool"`, `ID="good-tool"`, `ExpectedVersion=0`, and payload `{"name": "grep", "description": "search files"}`.
   **Expected:** The call succeeds. The codec decodes the payload without error. The record is persisted with `Version=1`.

2. Call `PutRaw` with `Kind="tool"`, `ID="bad-tool"`, `ExpectedVersion=0`, and payload `{invalid json`.
   **Expected:** The call returns an error indicating JSON syntax failure. No record is created for `bad-tool`. A subsequent `Get` for `bad-tool` returns not-found.

3. Call `PutRaw` with `Kind="tool"`, `ID="missing-fields"`, `ExpectedVersion=0`, and payload `{"name": "grep"}` (missing required `description`).
   **Expected:** The call returns a validation error from the codec indicating the missing required field. No record is persisted.

4. Call `PutRaw` with `Kind="tool"`, `ID="wrong-types"`, `ExpectedVersion=0`, and payload `{"name": 42, "description": true}`.
   **Expected:** The call returns a type mismatch error from the codec. No record is persisted.

5. Call `PutRaw` with `Kind="tool"`, `ID="extra-fields"`, `ExpectedVersion=0`, and payload `{"name": "grep", "description": "search", "unknown_field": "value"}`.
   **Expected:** Behavior depends on codec strictness policy -- either the extra field is silently ignored (lenient) or rejected (strict). Either way, no partial or corrupted record is stored.

## Edge Cases

- Empty payload (`{}`) is rejected if required fields are missing, not silently stored as an empty record.
- Null payload (`null`) is rejected at the raw input layer before reaching the codec.
- Payload that is valid JSON but exceeds a size limit (if configured) is rejected with a clear size error.
- Codec decode errors include sufficient context (field name, expected type, actual value) for debugging.
- A payload that passes JSON decoding but fails domain validation (e.g., `name` is an empty string when non-empty is required) is caught by the codec's validation phase.
- Update operations (`ExpectedVersion > 0`) with invalid payloads are also rejected, and the existing valid record remains untouched.
