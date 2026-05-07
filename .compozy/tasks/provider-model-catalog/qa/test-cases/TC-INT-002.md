# TC-INT-002: HTTP/UDS Native Catalog Handlers Serve Daemon-Owned Projection

**Priority:** P0
**Type:** Integration
**Systems:** `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/modelcatalog`.
**Requirement:** TechSpec Public Interfaces, ADR-001, Task 07.
**Status:** Not Run

## Objective

Verify the native catalog HTTP and UDS routes (registered via the `/api/providers/*catalog_path` dispatcher) return identical daemon-owned payloads for list / refresh / status across both transports.

## Preconditions

- [ ] Daemon running with seeded catalog state from TC-INT-001 fixture.
- [ ] Bearer token configured for HTTP; UDS connected via local socket from bootstrap manifest.

## Test Steps

1. **HTTP `GET /api/providers/models`.**
   - **Expected:** 200; payload `ProviderModelPayload` shape; deterministic sort order.
2. **HTTP `GET /api/providers/{provider_id}/models`.**
   - **Expected:** Returns subset filtered by provider; same deterministic sort.
3. **HTTP `POST /api/providers/{provider_id}/models/refresh`.**
   - **Expected:** Returns `[]SourceStatus` with `refresh_request_id`; status reflects new `last_refresh_at`.
4. **HTTP `GET /api/providers/models/status` and `/api/providers/{provider_id}/models/status`.**
   - **Expected:** Status payload includes redacted `last_error`; rows match SQLite source rows.
5. **UDS parity.**
   - Repeat each call via UDS client (`internal/cli/client_provider_models.go` exposes the parity surface).
   - **Expected:** Same shape; UDS responses match HTTP byte-equally for steady-state list payloads (TC-INT-003 validates byte equality).
6. **Refresh failure path.**
   - Force a source to fail; refresh again.
   - **Expected:** HTTP and UDS responses both surface failed status with redacted error.

## Audit Coverage

- C5 channel coverage, C8 cross-surface truth.
- SI-4, SI-9, SI-13.

## Pass Criteria

- All routes respond with documented payloads on both transports.
- Refresh failures surface consistently.

## Failure Criteria

- Any route differs in shape between HTTP and UDS.
- Refresh failure exposes raw error.
