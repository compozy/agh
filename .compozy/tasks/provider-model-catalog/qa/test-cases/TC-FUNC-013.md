# TC-FUNC-013: Source Error Redaction at Persistence + Projection

**Priority:** P0
**Type:** Functional
**Module:** `internal/modelcatalog/redact.go`, projection helpers.
**Requirement:** TechSpec SI-9.
**Status:** Not Run

## Objective

Verify source errors are redacted at both persistence time and at every public projection boundary so that secrets cannot leak through alternate surfaces.

## Preconditions

- [ ] Stub source whose error message contains an API key (`sk-test-1234567890abcdef`), an OAuth token (`Bearer secret.token`), and an env-shaped secret (`OPENAI_API_KEY=secret-xyzzy`).
- [ ] Daemon log capture available.

## Test Steps

1. **Trigger refresh failure with the seeded error string.**
   - **Expected:** SQLite `model_catalog_sources.last_error` contains a redacted summary; raw secret strings absent.
2. **List status via HTTP / UDS / CLI / Host API.**
   - **Expected:** `last_error` field redacted in every surface; payload byte-equal between HTTP and UDS for the same status row.
3. **List status via web app.**
   - **Expected:** Web component renders the redacted string only; no secret visible in DOM, network response, or React Query cache (TC-UI-001 covers UI rendering).
4. **Daemon log capture.**
   - **Expected:** Structured log entry omits secret strings; correlation keys (`refresh_request_id`, `provider_id`, `source_id`, `source_kind`) present.
5. **Inject error at projection time only (bypassing persistence redaction).**
   - **Expected:** Projection helper still redacts before serialization (defense in depth at HTTP/UDS/Host API/SSE boundary).

## Audit Coverage

- C6 task tree (Task 11), C11 disruption probe.
- SI-9.

## Pass Criteria

- No surface emits raw secret material.
- Both persistence and projection redaction functions invoked.

## Failure Criteria

- Any surface (logs, status, API, web, Host API, SSE) reveals a secret.
- Projection skips redaction when persistence layer is bypassed.
