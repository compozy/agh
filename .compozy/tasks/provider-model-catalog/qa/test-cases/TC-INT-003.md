# TC-INT-003: HTTP/UDS Canonical JSON Byte Equality + CLI Parity

**Priority:** P0
**Type:** Integration
**Systems:** `internal/api/core` deterministic encoder, `internal/api/testutil/model_catalog_parity_test.go`.
**Requirement:** TechSpec Testing Approach, Task 11.
**Status:** Not Run

## Objective

Verify the native catalog payload bytes match exactly between HTTP and UDS for at least one deterministic catalog state, that CLI structured JSON output covers the same persisted state, and that the Host API projection is structurally equivalent.

## Preconditions

- [ ] Daemon seeded with deterministic catalog state.
- [ ] Bearer auth + UDS socket from bootstrap manifest.

## Test Steps

1. **Capture HTTP `GET /api/providers/models` response body bytes.**
2. **Capture UDS `GET /api/providers/models` response body bytes.**
   - **Expected:** Byte equal after canonical sort.
3. **Capture CLI `agh provider models list -o json` output.**
   - **Expected:** Structurally equivalent (same provider/model rows, sources, availability) after JSON normalization; CLI may add wrapper metadata but core list matches.
4. **Capture Host API `models/list` (extension capability granted) response.**
   - **Expected:** Same provider/model rows; daemon-owned projection (not raw extension payload).
5. **Repeat for status (`GET /api/providers/models/status`) and refresh (one cycle).**
   - **Expected:** Same parity holds.
6. **Modify state via Settings > Providers (TC-UI-001).**
   - **Expected:** Subsequent CLI/HTTP/UDS/Host API calls reflect the change uniformly.

## Audit Coverage

- C5, C8.
- TC-INT-002 covers shape; TC-INT-003 enforces byte/structural identity.

## Pass Criteria

- All four surfaces agree for at least the steady-state list payload.

## Failure Criteria

- Any drift between HTTP and UDS bytes.
- CLI loses fields.
- Host API exposes raw extension payload.
