# TC-FUNC-016: UDS DELETE rejects stale version

**Priority:** P1
**Type:** Functional
**Package:** internal/api/udsapi
**Related Tasks:** 06

## Objective

Validate that the UDS API enforces optimistic concurrency control on DELETE operations. When a client sends DELETE /api/resources/tool/my-tool with an expected_version that does not match the current stored version, the server must return HTTP 409 Conflict. This prevents stale clients from deleting resources that have been modified since they last read them.

## Preconditions

- UDS server is running and accepting connections on its Unix domain socket.
- A tool resource `my-tool` exists in the resource store with a known version (e.g., version 3).
- The caller has appropriate permissions to delete tool resources.

## Test Steps

1. Create a tool resource `my-tool` via PUT /api/resources/tool/my-tool with valid spec payload.
   **Expected:** 200 OK returned. Resource stored at version 1.

2. Update the resource via PUT /api/resources/tool/my-tool with modified spec and expected_version=1.
   **Expected:** 200 OK returned. Resource now at version 2.

3. Send DELETE /api/resources/tool/my-tool with expected_version=1 (stale).
   **Expected:** 409 Conflict returned. Response body contains an error indicating version mismatch. The resource remains in the store at version 2.

4. Confirm the resource still exists by sending GET /api/resources/tool/my-tool.
   **Expected:** 200 OK returned. Resource spec matches the version 2 payload. Version field is 2.

5. Send DELETE /api/resources/tool/my-tool with expected_version=2 (current).
   **Expected:** 200 OK (or 204 No Content) returned. Resource is removed from the store.

6. Confirm deletion by sending GET /api/resources/tool/my-tool.
   **Expected:** 404 Not Found returned.

## Edge Cases

- DELETE with expected_version=0 (never valid) returns 409, not 400.
- DELETE without expected_version header/param follows the API contract (either unconditional delete or rejection depending on design — verify which behavior is specified).
- DELETE on a non-existent resource returns 404 regardless of expected_version value.
- Concurrent DELETE requests with the same correct expected_version: exactly one succeeds, the other gets 409 or 404.
- DELETE with expected_version matching a previously deleted resource's last version returns 404 (resource already gone).
