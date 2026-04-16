# SMOKE-005: UDS Resource PUT/GET/DELETE

**Priority:** P0
**Type:** Smoke
**Package:** internal/api/udsapi
**Related Tasks:** 06

## Objective

Validate that the UDS (Unix Domain Socket) API correctly handles the full resource lifecycle via HTTP-style verbs: PUT to create, GET to retrieve, and DELETE to remove. Verify correct HTTP status codes (201, 200, 204) and that optimistic concurrency via expected_version works end-to-end through the API layer.

## Preconditions

- UDS server running on a temporary socket path (`t.TempDir()` based)
- Resource store initialized and wired into the UDS handler
- A test UDS client configured to connect to the socket

## Test Steps

1. **PUT a new resource** via `PUT /resources/{kind}/{id}` with body containing scope="workspace", owner_kind="session", owner_id="sess-001", expected_version=0, and a JSON spec payload.
   **Expected:** Response status is 201 Created. Response body contains the resource record with version=1, matching kind, scope, owner fields, and the spec payload.

2. **GET the resource** via `GET /resources/{kind}/{id}`.
   **Expected:** Response status is 200 OK. Response body matches the record from step 1 exactly, including version=1 and all metadata.

3. **PUT an update** to the same resource via `PUT /resources/{kind}/{id}` with expected_version=1 and a modified spec payload.
   **Expected:** Response status is 200 OK. Response body contains version=2, the updated spec, and updated_at greater than the original.

4. **GET the updated resource** via `GET /resources/{kind}/{id}`.
   **Expected:** Response status is 200 OK. Response body reflects version=2 and the updated spec payload.

5. **DELETE the resource** via `DELETE /resources/{kind}/{id}?expected_version=2`.
   **Expected:** Response status is 204 No Content. Response body is empty.

6. **GET the deleted resource** via `GET /resources/{kind}/{id}`.
   **Expected:** Response status is 404 Not Found. Response body contains an error message indicating the resource does not exist.

## Edge Cases

- PUT with expected_version=0 on an existing resource returns 409 Conflict
- PUT with a stale expected_version (e.g., 1 when current is 2) returns 409 Conflict
- DELETE with a stale expected_version returns 409 Conflict
- DELETE on a non-existent resource returns 404 Not Found
- PUT with missing required fields (no kind, no scope) returns 400 Bad Request with a validation error
- PUT with an invalid JSON spec payload returns 400 Bad Request
- GET on a non-existent resource returns 404 Not Found
- Concurrent PUT requests to the same resource from different clients: one succeeds, the other gets 409 Conflict
- PUT with a kind containing special characters (dots, hyphens) is handled correctly
- Response Content-Type is application/json for all non-204 responses
