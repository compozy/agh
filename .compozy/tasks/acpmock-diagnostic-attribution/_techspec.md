# TechSpec — ACP Mock Diagnostic Attribution

- **Status:** Approved for immediate implementation by the user on 2026-07-13
- **Date:** 2026-07-13
- **Source:** `docs/qa/bugs/BUG-0036.md`; no separate PRD or user-story catalog exists

## Executive Summary

The full runtime E2E gate cannot reliably prove provider reasoning negotiation because multiple ACP
processes can write to one fixture-agent diagnostics JSONL while reusing process-local ACP session
IDs. Production already supplies the correct owner, `AGH_SESSION_ID`, to every ACP subprocess. This
change makes that existing identity explicit on every mock diagnostic record and requires
session-specific assertions to filter by the API-returned AGH session ID before semantic projection.

The implementation is test-harness-only. It does not add production correlation fields, change ACP
wire contracts, disable background memory work, or split the shared diagnostics file. The shared
file remains the faithful concurrency boundary; attribution makes it trustworthy.

**MVP boundary:** one implementation slice adds the diagnostics owner, the exact filter, canonical
unit/driver/E2E regressions, BUG-0036 verification evidence, and the final Hermes bridge source-freeze
gates. There is no post-MVP phase in this TechSpec. New production observability, public diagnostics
APIs, per-process files, and memory-runtime changes are explicitly out of scope.

## Forensic Frame

- **Observed:** 2026-07-12 during `make test-e2e-runtime` with race-enabled package parallelism.
- **Failing owner:**
  `TestDaemonE2EProviderReasoningNegotiatesThroughAdvertisedACPOptions/Should resolve the AGENT reasoning default before the first prompt`.
- **Evidence:** the shared diagnostics file contained
  `model → effort → prompt → model → effort`; daemon logs identified two distinct AGH session IDs
  whose mock processes both emitted the same process-local ACP session ID.
- **Control:** the exact reasoning E2E passed 80/80 subtests across ten isolated runs, demonstrating
  that the failure depends on a concurrent daemon-owned session rather than reasoning negotiation.
- **Root cause:** `DiagnosticsRecord` persists only the process-local ACP session ID. The mock driver
  drops the daemon-owned `AGH_SESSION_ID` already present in its environment.

## Goals and Non-Goals

### Goals

- Attribute every driver-emitted diagnostics record to its owning AGH session.
- Select records by the session ID returned by the public API before protocol/prompt projections.
- Preserve shared-file concurrency and prove two same-fixture processes cannot pollute each other's
  assertions even when their ACP session IDs collide.
- Restore trustworthy runtime E2E and final verification evidence for the Hermes bridge program.

### Non-Goals

- No ACP protocol, sandbox, session-launch, HTTP, UDS, CLI, OpenAPI, or Web contract changes.
- No new daemon logs, metrics, persisted database state, or operator-facing diagnostics surface.
- No suppression or disabling of the memory extractor or other background sessions.
- No per-process diagnostics file allocation or compatibility path for unscoped assertions.

## System Architecture

### Component Overview

1. `internal/session` remains the production owner of `AGH_SESSION_ID` subprocess injection.
2. `internal/testutil/acpmock/cmd/acpmock-driver` reads that immutable process environment value and
   stamps it at the single JSONL writer boundary.
3. `internal/testutil/acpmock` decodes records and owns the exact AGH-session filter.
4. `internal/daemon` E2E tests use `SessionPayload.ID` returned by the operator API to select records
   before calling `ProtocolDiagnostics`.

### Data Flow

```text
SessionPayload.ID
  -> sessionStartEnvForProvider("AGH_SESSION_ID")
  -> acpmock-driver process environment
  -> writeDiagnostics(record.AGHSessionID)
  -> shared per-agent diagnostics JSONL
  -> ReadDiagnostics
  -> DiagnosticsForAGHSession(SessionPayload.ID)
  -> ProtocolDiagnostics / PromptDiagnostics
  -> assertion
```

## Architectural Boundaries

- `internal/session` remains unchanged and owns production session identity propagation.
- `internal/testutil/acpmock` may define test-evidence DTOs and pure selection helpers; it must not
  import `internal/session`, `internal/daemon`, API transports, or stores.
- `cmd/acpmock-driver` receives ownership only through the existing process environment and stamps it
  centrally; record-producing call sites must not accept or invent an AGH session ID.
- `internal/daemon` is the E2E composition root and may combine the API-returned session payload with
  diagnostics helpers.
- No new internal package, package-boundary exception, reverse dependency, or production sidecar is
  introduced.

## Implementation Design

### Core Interfaces

```go
type DiagnosticsRecord struct {
    AGHSessionID string `json:"agh_session_id,omitempty"`
    AgentName    string `json:"agent_name"`
    SessionID    string `json:"session_id"`
    // Existing protocol, lifecycle, prompt, match, and step fields remain unchanged.
}

func DiagnosticsForAGHSession(
    records []DiagnosticsRecord,
    aghSessionID string,
) []DiagnosticsRecord
```

`DiagnosticsForAGHSession` trims the requested owner once, returns an allocated empty result for an
empty owner, performs exact comparisons against trimmed record owners, and preserves source order.
It does not mutate records or apply protocol/prompt semantics.

### Data Models and Field Rationale

| Field | Shape | Scope | Rationale |
| --- | --- | --- | --- |
| `DiagnosticsRecord.AGHSessionID` | Go `string`, JSON `agh_session_id,omitempty` | Test evidence, AGH-session scoped | Stable daemon-owned process identity used to distinguish records whose ACP session IDs collide |
| Existing `DiagnosticsRecord.SessionID` | Go `string`, JSON `session_id` | ACP-process local | Preserved as protocol evidence; explicitly not an AGH ownership key |

No SQLite column, config key, public payload, frontmatter field, or generated contract changes.

### Side-Table vs JSON Decision

No domain entity or persistent operational state is introduced, so no side table is appropriate.
The field belongs in the existing diagnostics JSON object because the JSONL file is an ephemeral
test-evidence stream and the value describes each emitted record. Storing it in a database or a
parallel metadata file would create a second attribution source.

### Diagnostics Writer

`mockAgent.writeDiagnostics` overwrites `record.AGHSessionID` with the trimmed
`os.Getenv("AGH_SESSION_ID")` immediately before `json.Marshal`. This single point covers session
lifecycle, protocol configuration, prompt, and control diagnostics. Direct driver launches without
the environment variable remain readable but unowned; an exact non-empty filter never selects them.

### Session-Specific Projection

Successful reasoning assertions receive `SessionPayload.ID` from `RuntimeHarness.CreateSession`.
They call `DiagnosticsForAGHSession(records, sessionPayload.ID)` before
`ProtocolDiagnostics`. Failure-path assertions that never receive a session payload keep their
existing agent-scoped evidence and are outside the confirmed collision path.

The canonical concurrent regression creates two sessions from the same registered fixture agent,
prompts both concurrently, confirms their process-local ACP session IDs collide, and proves each
API-returned AGH session ID selects exactly its own `model → effort → prompt` sequence.

## Public Interfaces / Types

- **Added internal test type field:** `DiagnosticsRecord.AGHSessionID`.
- **Added internal test helper:** `DiagnosticsForAGHSession`.
- **Unchanged:** HTTP endpoints, UDS routes, CLI verbs, native tools, OpenAPI, generated TypeScript,
  bridge SDK contracts, provider manifests, and `config.toml`.

## Safety Invariants

1. The driver writer, not individual record producers, assigns `AGHSessionID`.
2. ACP `SessionID` is never accepted as an AGH-session ownership key.
3. An empty requested AGH session ID matches no records.
4. Selection uses an exact trimmed AGH session ID comparison and preserves append order.
5. Records owned by another AGH session never enter a session-specific semantic projection.
6. Lifecycle, protocol, and prompt records all pass through the same stamping boundary.
7. Concurrent processes retain the current single `O_APPEND` write per JSONL object; attribution does
   not add a second file or an in-process lock that cannot coordinate subprocesses.
8. The E2E assertion obtains its owner from the public API response, not from fixture naming, process
   order, timestamps, or daemon logs.
9. Background memory work remains enabled so the gate continues to exercise real daemon concurrency.
10. No production interface is widened to repair test evidence.

## Integration Points

- Existing `sessionStartEnvForProvider` supplies `AGH_SESSION_ID`; no change is required.
- Existing `ReadDiagnostics` remains the only JSONL decoder.
- Existing `ProtocolDiagnostics` and `PromptDiagnostics` remain semantic projections and do not gain
  implicit ownership behavior.
- Existing runtime harness `CreateSession` provides the authoritative public session ID.

## Extensibility Integration Plan

No impact after checking extension manifests, hooks, skills/capabilities, tools/resources, bundles,
registries, bridge SDKs, MCP sidecars, provider subprocess contracts, and protocol documentation.
The change is confined to the in-repo ACP mock executable and test helpers; extension and bridge
processes do not consume `DiagnosticsRecord`.

## Agent Manageability Plan

No public capability is added or changed. CLI verbs, HTTP endpoints, UDS routes, structured outputs,
status/config discovery, deterministic public errors, and native tools were checked and remain
unchanged. Release agents benefit only through deterministic internal gate evidence; they do not
operate this test-only field at runtime.

## Config Lifecycle

No `config.toml` key, default, merge/overlay rule, validator, example, generated CLI reference, or
site configuration documentation changes. `AGH_SESSION_ID` is an existing per-process runtime value,
not operator configuration.

## Web / Docs Impact

- `web/`: no impact; no public payload or behavior changes.
- `packages/site`: no impact; operator bridge behavior is unchanged.
- Internal QA docs: update BUG-0036 and the Hermes bridge QA report/remediation evidence after the
  focused and final gates pass.
- Official AGH skill: no impact; no public command, tool ID, capability, or runtime semantic changes.

## Delete Targets

- Delete unscoped `ProtocolDiagnostics(records)` use from successful session-specific reasoning
  assertions; replace it with explicit AGH-session selection.
- Do not add aliases, dual owner fields, fallback matching by ACP session ID, or fixture-name
  heuristics.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
| --- | --- | --- | --- |
| `internal/testutil/acpmock/diagnostics.go` | Modified | Adds the owner field and pure filter; low risk | Unit cases for exact, empty, mismatch, and order behavior |
| `cmd/acpmock-driver/diagnostics.go` | Modified | Central stamp from process environment; medium evidence risk | Driver integration proves the emitted field |
| `internal/daemon/daemon_mock_agents_integration_test.go` | Modified | Hard-cuts successful reasoning assertions to API identity; medium concurrency risk | Same-fixture two-session regression plus focused stress |
| QA/review artifacts | Modified | Converts BUG-0036 from open to verified only after fresh evidence | Record commands and truthful verdicts |

## Testing Approach

- Go tests follow AGH conventions: every case is a `t.Run("Should …")`; independent cases use
  `t.Parallel`; no test mutates global environment concurrently.
- Unit ownership remains in `internal/testutil/acpmock/fixture_test.go`, which already owns JSONL
  decoding and diagnostic projections.
- Driver environment propagation remains in `internal/testutil/acpmock/driver_test.go`, the existing
  real subprocess diagnostics suite. It passes `AGH_SESSION_ID` through `StartOpts.Env` rather than
  `t.Setenv`.
- Runtime attribution remains in the existing reasoning E2E suite under the `integration` tag.
- Focused iteration commands target only `internal/testutil/acpmock/...` and the exact daemon
  reasoning E2E. The full runtime E2E, full Web E2E, and one `make verify` run only after source freeze.
- Concrete cases and commands are defined in `_tests.md`.

## Development Sequencing

### Build Order

1. Add red owner/filter cases to the existing `acpmock` suites.
2. Add the red same-fixture concurrent reasoning regression.
3. Add `AGHSessionID`, central writer stamping, and exact selection.
4. Hard-cut successful reasoning assertions to the API-returned owner.
5. Run focused race tests and coverage; update BUG-0036 and review memory.
6. Run deslop and source-freeze checks.
7. Run full runtime E2E, full Web E2E, and one `make verify` serially.
8. Resume Hermes bridge Phase D review rounds until `SHIP`.

### Technical Dependencies

- Existing `AGH_SESSION_ID` propagation and `SessionPayload.ID` are required and already shipped.
- No external service, live bridge account, new dependency, schema migration, or code generation is
  required for implementation.

## Monitoring and Observability

No production metric, log, or alert changes. The observable output is test evidence:

- decoded records carry `agh_session_id`;
- each session-specific projection contains only its exact owner;
- the runtime E2E gate no longer fails from background-session pollution.

## Technical Considerations

### Key Decisions

- Reuse the daemon-owned AGH session identity instead of inventing a process invocation identity.
- Keep ownership selection separate from semantic projections so callers state their scope.
- Preserve the shared diagnostics file to exercise the actual concurrency boundary.
- Fail closed for an empty owner rather than treating unowned records as a match.

### Known Risks

- Other tests may make agent-scoped assertions over intentionally aggregated diagnostics. They remain
  unchanged; only assertions claiming one API session must apply the owner filter.
- Full-lane failures unrelated to BUG-0036 may surface after this fix. They are production/test bugs
  to investigate, not reasons to weaken this contract.

## AGH Impact Audit

- **Native tools:** no impact; checked tool IDs, descriptors, schemas, digests, risk flags, and
  availability gates. The change is test-harness-only.
- **Extensibility and hooks:** no impact; checked extensions, hooks, skills/capabilities,
  tools/resources, bundles, registries, bridge SDKs, MCP sidecars, and config lifecycle. None consume
  the diagnostics DTO.
- **Workspace data isolation:** the new datum is session-scoped test evidence. Exact AGH-session
  selection prevents records from another session or workspace from entering the assertion; no
  runtime store, cache, SSE, event, HTTP, UDS, CLI, or Web path changes.
- **Official AGH skill:** no impact; checked `skills/agh/` public operations and no behavior, tool ID,
  CLI path, hook, capability, bundle/resource, memory, network, or task semantic changes.

## Architecture Decision Records

- [ADR-001: Attribute ACP Mock Diagnostics with the Existing AGH Session ID](adrs/adr-001.md)

## Assumptions / Defaults

- The user-approved shaping decision is final: this TechSpec is harness-only and executes
  immediately without a separate draft-approval interruption.
- `sessionStartEnvForProvider` continues to set a non-empty `AGH_SESSION_ID` for daemon-launched ACP
  processes.
- Direct mock-driver tests may omit the value; such records remain valid aggregate evidence but never
  match a non-empty session-specific filter.
- The unrelated Daytona sidecar asset changes remain excluded from every checkpoint.
- Global gates remain deferred until all focused source work is frozen.
