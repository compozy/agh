# TC-SEC-010: HTTP Mutation Routes Disabled Without Auth Middleware

**Priority:** P1
**Type:** Security
**Package:** internal/api/httpapi
**Related Tasks:** 06

## Objective

Validate that HTTP mutation routes (PUT, DELETE) on `/api/resources` are not accessible without the operator auth middleware. These routes must either not be registered in the HTTP router or return 404/405 when the auth middleware is not configured, preventing unauthenticated remote writes.

## Preconditions

- The daemon is running with the HTTP/SSE API enabled (for web UI).
- The operator auth middleware is either not configured or explicitly disabled (simulating a deployment without authentication).
- The resource store contains at least one record.
- An HTTP client can reach the daemon's HTTP port.

## Test Steps

1. Send an HTTP `PUT` request to `/api/resources` with a valid resource record payload.
   **Expected:** Response is 404 Not Found or 405 Method Not Allowed. The route is not registered when auth middleware is absent.

2. Send an HTTP `DELETE` request to `/api/resources/{kind}/{id}` targeting an existing record.
   **Expected:** Response is 404 Not Found or 405 Method Not Allowed. Same behavior as PUT -- mutation routes are entirely absent.

3. Send an HTTP `GET` request to `/api/resources` (read path).
   **Expected:** Response is 200 OK with the resource list. Read-only routes remain available for the web UI regardless of auth middleware configuration.

4. Enable the operator auth middleware, then retry the PUT request from step 1 without authentication credentials.
   **Expected:** Response is 401 Unauthorized. With auth middleware enabled, the route exists but requires valid credentials.

5. With auth middleware enabled, send the PUT request with valid operator credentials.
   **Expected:** Response is 200/201 and the record is created. The authenticated path works correctly.

## Edge Cases

- HTTP `PATCH` method on resource routes -- verify it is also blocked or not registered.
- HTTP `POST` to resource routes -- verify consistent treatment with PUT/DELETE.
- Attempting to reach mutation routes via path traversal (e.g., `/api/../api/resources` or `/api/resources/..%2f..%2f`).
- Sending mutation requests with `X-Forwarded-For` or other proxy headers to simulate internal origin.
- OPTIONS preflight request to mutation routes -- verify CORS preflight does not inadvertently confirm route existence when routes should be hidden.
- HTTP method override via `X-HTTP-Method-Override` header (e.g., GET request with `X-HTTP-Method-Override: PUT`).

## Threat Model

This test prevents **unauthenticated remote resource mutation via the HTTP API**. The HTTP/SSE interface is designed primarily for the web UI's read-only consumption of resource state. Unlike the UDS API (which is restricted to local processes via Unix domain socket permissions), the HTTP API may be exposed on a network interface. If mutation routes were registered without authentication, any network-adjacent attacker could create, modify, or delete resource records remotely -- injecting malicious tool definitions, removing hook bindings, or corrupting the resource store. The defense-in-depth approach of not registering mutation routes at all (rather than just requiring auth) ensures that a misconfigured auth middleware cannot silently expose write access.
