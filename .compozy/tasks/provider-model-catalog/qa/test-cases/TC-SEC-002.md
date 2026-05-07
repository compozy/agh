# TC-SEC-002: `/api/openai/v1/models` Auth + OpenAI-Shaped Errors

**Priority:** P0
**Type:** Security
**OWASP Category:** A01 (broken access control)
**Risk Level:** High
**Requirement:** TechSpec OpenAI-Compatible Projection.
**Status:** Not Run

## Objective

Verify the OpenAI projection enforces bearer auth like every `/api/*` route, returns OpenAI-shaped error envelope on auth failure, and remains absent from UDS where authentication semantics differ.

## Preconditions

- [ ] Daemon running with bearer auth enforced.
- [ ] Test client without token.

## Test Steps

1. **Unauthenticated HTTP request.**
   - Command: `GET /api/openai/v1/models` without `Authorization`.
   - **Expected:** 401 / 403 with OpenAI-shaped error envelope: `{"error":{"message":"...","type":"...","code":"..."}}`; AGH HTTP status code matches `/api/*` semantics; no catalog data leaked.
2. **Bad bearer token.**
   - **Expected:** Same shape; rate limiting and CORS middleware applied if enabled.
3. **CORS preflight.**
   - Send OPTIONS with allowed origin.
   - **Expected:** CORS responds per `/api/*` policy.
4. **Authenticated `provider_id` filter for unknown provider.**
   - **Expected:** 200 with empty `data`; no error.
5. **Method not supported for refresh.**
   - Command: `POST /api/openai/v1/models`.
   - **Expected:** 404/405 with OpenAI-shaped error if applicable.
6. **UDS does not expose the route.**
   - Command: hit UDS path.
   - **Expected:** 404; auth boundary respected.

## Audit Coverage

- C5, C9 boundary, C11.

## Pass Criteria

- All steps return documented behavior.

## Failure Criteria

- Unauthenticated call returns data.
- Error envelope diverges from OpenAI shape.
- UDS exposes the route.
