# Test Specification: ACP Mock Diagnostic Attribution

Canonical test contract for BUG-0036. Companion to `_techspec.md`. No separate `_user_stories.md`
exists; the release-operator journey and edge cases derive from the QA bug and the authorized
TechSpec.

## Test Placement

- **Invariant:** every driver-emitted record carries its daemon-owned AGH session identity and the
  central writer cannot retain a caller-supplied owner. **Owning layers:** diagnostics writer unit
  plus real ACP mock driver subprocess boundary. **Canonical suites:**
  `cmd/acpmock-driver/diagnostics_test.go`, the colocated writer owner created because no direct
  writer suite existed, and `internal/testutil/acpmock/driver_test.go`, the existing diagnostics
  capture test.
- **Invariant:** AGH-session selection is exact, ordered, and fail-closed for an empty owner.
  **Owning layer:** diagnostics test helper. **Canonical suite:**
  `internal/testutil/acpmock/fixture_test.go`, existing JSONL/projection owner.
- **Invariant:** concurrent processes using the same fixture-agent registration cannot pollute one
  session's reasoning sequence even when their ACP session IDs collide. **Owning layer:** daemon E2E
  composition. **Canonical suite:**
  `TestDaemonE2EProviderReasoningNegotiatesThroughAdvertisedACPOptions` in
  `internal/daemon/daemon_mock_agents_integration_test.go`.

No duplicate regression is created: the new colocated writer suite proves anti-forgery at the
lowest layer, while the existing subprocess suite distinctly proves environment propagation.

## Strategy

- **Frameworks and harnesses:** Go `testing`, the real `acpmock-driver` subprocess, and the existing
  daemon `RuntimeHarness`; fakes remain at provider I/O boundaries.
- **Execution:** unit/driver cases run with `CGO_ENABLED=1` and `-race`; daemon cases use the existing
  `integration` build tag; final runtime/Web/monorepo gates run serially after source freeze.
- **Conventions:** every case uses `t.Run("Should …")`; independent subtests use `t.Parallel`; no
  `time.Sleep`, `t.Setenv` in parallel tests, discarded errors, snapshots, or implementation-literal
  assertions.

## Coverage Matrix

| Source | Behavior | Unit | Integration | E2E |
| --- | --- | --- | --- | --- |
| BUG-0036 | Stable AGH owner on every diagnostics record | UT-001 | IT-001 | E2E-001 |
| BUG-0036 | Exact session selection before protocol projection | UT-002, UT-003, UT-004 | — | E2E-001 |
| BUG-0036 concurrency | Two same-fixture processes share JSONL without attribution pollution | UT-005 | IT-002 | E2E-002 |
| Diagnostics writer | Caller cannot forge another process owner | UT-006 | IT-001 | — |
| Release gate | Hermes runtime evidence is trustworthy after source freeze | — | — | E2E-003 |

## Unit Tests

### Diagnostics record and selection (`internal/testutil/acpmock`)

- **UT-001** (happy): `ReadDiagnostics` decodes `agh_session_id` from a JSONL record and preserves it
  alongside the existing ACP `session_id`.
- **UT-002** (ordering): `DiagnosticsForAGHSession` given interleaved owners `sess-a`, `sess-b`,
  `sess-a` returns the two `sess-a` records in source order.
- **UT-003** (boundary): `DiagnosticsForAGHSession` given a whitespace-only requested owner returns an
  allocated empty result and never selects records whose owner is empty.
- **UT-004** (error boundary): `DiagnosticsForAGHSession` given an unknown non-empty owner returns no
  records without falling back to ACP session ID, fixture agent name, or timestamp.
- **UT-005** (state): `ProtocolDiagnostics(DiagnosticsForAGHSession(records, "sess-a"))` excludes
  protocol records owned by `sess-b` even when both records carry the same ACP `session_id`.

### Driver writer (`cmd/acpmock-driver`)

- **UT-006** (state): `writeDiagnostics` replaces a caller-supplied `AGHSessionID` with the trimmed
  process environment owner before JSON marshaling.

## Integration Tests

### Real mock-driver process

- **IT-001**: the existing diagnostics capture test launches `acpmock-driver` with
  `AGH_SESSION_ID=sess-driver-owner` through `acp.StartOpts.Env`; the decoded lifecycle record carries
  exactly `sess-driver-owner` while retaining its existing MCP server payload.
- **IT-002**: two daemon-launched processes for one registered fixture agent append decodable records
  to the same diagnostics JSONL; every record carries a non-empty daemon owner, and each user-session
  selection carries exactly its own API-returned owner. Background-session owners remain excluded.

## End-to-End Tests

### Provider reasoning negotiation

- **E2E-001**: every successful case in
  `TestDaemonE2EProviderReasoningNegotiatesThroughAdvertisedACPOptions` filters records by the created
  `SessionPayload.ID` before `ProtocolDiagnostics`, then observes exactly
  `model → reasoning → prompt`.
- **E2E-002** (concurrency): create two sessions from the same reasoning fixture-agent registration,
  prompt them concurrently, confirm their ACP session IDs collide, and prove each AGH session ID
  selects exactly its own three-record reasoning sequence with no cross-owner record.
- **E2E-003**: after source freeze, `make test-e2e-runtime` passes with memory extraction and package
  parallelism enabled; the former BUG-0036 test has no extra model/reasoning records.

## Verification Commands

Focused iteration only:

```bash
CGO_ENABLED=1 go test -race -count=1 ./internal/testutil/acpmock/...
CGO_ENABLED=1 go test -race -tags integration -count=10 \
  -run '^TestDaemonE2EProviderReasoningNegotiatesThroughAdvertisedACPOptions$' \
  ./internal/daemon
```

Source-freeze gates, serial and exactly once where globally scoped:

```bash
make codegen-check
make test-e2e-runtime
make test-e2e-web
make verify
```
