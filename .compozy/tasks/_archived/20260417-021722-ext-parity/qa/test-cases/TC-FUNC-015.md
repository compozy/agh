# TC-FUNC-015: UDS PUT creates record with 201

**Priority:** P0
**Type:** Functional
**Package:** internal/api/udsapi
**Related Tasks:** 06

## Objective

Validate that the UDS (Unix Domain Socket) API correctly handles resource creation via `PUT /api/resources/{kind}/{id}`. A PUT request with `expected_version=0` must create a new record and return HTTP 201 with the full record shape, including `version=1`. This verifies the end-to-end path from the HTTP transport layer through to the resource store and back.

## Preconditions

- The AGH daemon is running with the UDS API server listening on its configured socket path.
- The resource store is initialized with schema applied.
- The `tool` resource kind is registered with a valid codec.
- The UDS socket is accessible and the test client can connect to it.

## Test Steps

1. Send `PUT /api/resources/tool/my-tool` with request body:
   ```json
   {
     "expected_version": 0,
     "scope": "workspace",
     "scope_id": "ws-1",
     "spec": {
       "name": "my-tool",
       "description": "A test tool"
     }
   }
   ```
   **Expected:** Response status is `201 Created`. Response body contains the full record:
   ```json
   {
     "kind": "tool",
     "id": "my-tool",
     "version": 1,
     "scope": "workspace",
     "scope_id": "ws-1",
     "owner_kind": "...",
     "owner_id": "...",
     "source_kind": "...",
     "source_id": "...",
     "spec": {
       "name": "my-tool",
       "description": "A test tool"
     },
     "created_at": "...",
     "updated_at": "..."
   }
   ```
   The `version` field is exactly `1`. The `created_at` and `updated_at` timestamps are valid ISO 8601 and close to the current time. The `owner_*` and `source_*` fields are populated from the authenticated actor (derived from the UDS connection).

2. Send the same `PUT /api/resources/tool/my-tool` request again with `expected_version=0`.
   **Expected:** Response status is `409 Conflict`. The response body includes an error message indicating version conflict. The `Content-Type` header is `application/json`.

3. Send `PUT /api/resources/tool/my-tool` with `expected_version=1` and an updated spec.
   **Expected:** Response status is `200 OK` (update, not creation). Response body contains the updated record with `version=2`.

4. Send `GET /api/resources/tool/my-tool`.
   **Expected:** Response status is `200 OK`. The returned record matches the state from step 3 with `version=2`.

5. Send `PUT /api/resources/tool/another-tool` with `expected_version=0` and a valid spec.
   **Expected:** Response status is `201 Created` with `version=1`. Independent of the first tool.

6. Send `PUT /api/resources/invalid-kind/test` with `expected_version=0`.
   **Expected:** Response status is `400 Bad Request` or `422 Unprocessable Entity`. The error indicates the kind is not recognized.

## Edge Cases

- PUT with missing `expected_version` field: returns `400 Bad Request` with a clear validation error, not a silent default to 0.
- PUT with `expected_version` as a string instead of integer: returns `400` for type mismatch.
- PUT with an empty request body: returns `400` with a validation error.
- PUT with a valid kind but malformed `spec` that fails codec validation: returns `422` with codec error details, and no record is created.
- The `id` in the URL path and any `id` in the request body must match, or the body `id` is ignored in favor of the URL path.
- Very long `id` values (e.g., 1000 characters) are rejected with a validation error.
- Concurrent PUT requests from two UDS clients for the same `(kind, id)` with `expected_version=0`: exactly one gets `201`, the other gets `409`.
- Response headers include appropriate `Content-Type: application/json` and no caching headers.
