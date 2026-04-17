---
status: completed
title: "Validate Daytona SSH non-PTY transport"
type: infra
complexity: low
dependencies: []
---

# Task 05: Validate Daytona SSH non-PTY transport

## Overview

Validate that Daytona's SSH gateway supports non-PTY sessions with clean stdio byte streams suitable for ACP JSON-RPC. This is a blocking gate for the Daytona provider implementation. If validation fails, document the failure and the WebSocket sidecar fallback path.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create a Daytona sandbox via API or SDK
- MUST generate an SSH access token via `POST /api/sandbox/{id}/ssh-access`
- MUST connect via SSH without PTY allocation (no `-t` flag, no `RequestPty()`)
- MUST verify clean stdio by sending a JSON payload through stdin and checking stdout matches byte-for-byte
- MUST test with payloads of varying sizes (small: 100B, medium: 10KB, large: 100KB)
- MUST test with newline-delimited JSON (multiple messages in sequence)
- MUST document results: pass/fail, any observed artifacts, latency measurements
- MUST document fallback plan if validation fails (WebSocket sidecar per ADR-002)
- MUST clean up sandbox after validation
</requirements>

## Subtasks

- [ ] 5.1 Create Daytona sandbox and generate SSH token
- [ ] 5.2 Run SSH non-PTY validation tests (echo, cat, multi-message)
- [ ] 5.3 Measure round-trip latency for JSON-RPC-sized payloads
- [ ] 5.4 Document results and gate decision in validation report
- [ ] 5.5 Clean up sandbox

## Implementation Details

See TechSpec section: build order step 9, ADR-002 "Implementation Notes".

Write a small Go test file (`internal/environment/daytona/ssh_validation_test.go`) with build tag `integration` that performs the validation. The test requires `DAYTONA_API_KEY` environment variable.

### Relevant Files

- `internal/environment/daytona/ssh_validation_test.go` — New validation test file (to create)

### Dependent Files

- `internal/environment/daytona/transport.go` — SSH transport implementation depends on validation result (task 06)

### Related ADRs

- [ADR-002: SSH as Primary Transport](adrs/adr-002.md) — This task validates the core assumption

## Deliverables

- `internal/environment/daytona/ssh_validation_test.go` — Integration test with `//go:build integration`
- Validation report documenting: pass/fail, payload sizes tested, latency, any artifacts observed
- Gate decision: proceed with SSH or switch to WebSocket sidecar

## Tests

- Integration tests (tagged, requires `DAYTONA_API_KEY`):
  - [ ] SSH connect without PTY to Daytona sandbox succeeds
  - [ ] Small JSON payload (100B) echoed through `cat` matches byte-for-byte
  - [ ] Medium JSON payload (10KB) echoed correctly
  - [ ] Large JSON payload (100KB) echoed correctly
  - [ ] Newline-delimited multi-message sequence arrives intact
  - [ ] No terminal escape sequences or echo artifacts in output
  - [ ] Round-trip latency under 100ms for 1KB payload
- Test coverage target: N/A (spike, not library code)
- All tests must pass when `DAYTONA_API_KEY` is available

## Success Criteria

- Validation test passes with clean stdio confirmed
- OR: validation test fails and fallback plan documented
- Gate decision documented for task 06
- Sandbox cleaned up after validation
