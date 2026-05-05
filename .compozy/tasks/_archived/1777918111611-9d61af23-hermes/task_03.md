---
status: completed
title: ACP and Session Lifecycle Hardening
type: backend
complexity: critical
dependencies:
  - task_01
  - task_02
---

# Task 03: ACP and Session Lifecycle Hardening

## Overview

Make ACP and session lifecycle failures explicit, persisted, observable, and recoverable. This task introduces structured failure classification, exposes failure state consistently through API/SSE/CLI surfaces, adds downstream agent probes, and emits crash bundles that operators can use without scraping logs.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, and task_02 health outputs before changing lifecycle state
- DO NOT compare error strings; use structured failure kinds and wrapped errors
- DO NOT hide subprocess crashes behind generic session failure text
- EVERY goroutine or subprocess watcher must have explicit ownership and shutdown
- Crash evidence must redact secrets before it is persisted or exposed
</critical>

<requirements>
- MUST introduce a typed `FailureKind` model for ACP and session lifecycle failures
- MUST persist failure kind and diagnostic summary where session state is stored
- MUST expose failure information through contract DTOs, SSE events, and CLI read paths
- MUST add downstream agent health probes for ACP-compatible providers
- MUST generate redacted crash bundles for subprocess exits, protocol errors, and startup failures
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this task
</requirements>

## Subtasks
- [x] 3.1 Define `FailureKind` values and map ACP/session errors to them at source
- [x] 3.2 Persist failure classification and redacted diagnostic summaries in session storage
- [x] 3.3 Update API, SSE, and CLI conversions so operators see consistent failure state
- [x] 3.4 Add downstream agent probe logic with timeout, cancellation, and structured health output
- [x] 3.5 Add crash bundle generation with redaction and tests for subprocess/protocol/startup failures
- [x] 3.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

Classification should happen near the failure source and then flow through existing session and observe models. Keep crash bundle contents bounded and redacted. Web updates may include typed client changes, session detail failure badges, or test fixtures if the contract shape changes.

### Relevant Files
- `internal/store/types.go` - session state and failure model definitions
- `internal/store/meta.go` - persisted metadata helpers
- `internal/session/` - lifecycle manager, state transitions, and notifier calls
- `internal/acp/client.go` - ACP protocol and subprocess error sources
- `internal/acp/launcher.go` - provider startup failure handling
- `internal/observe/` - event recording for failure state and crash evidence
- `internal/api/contract/contract.go` - session DTOs and SSE payloads
- `internal/api/core/conversions.go` - conversion between internal and API failure models

### Dependent Files
- `internal/session/*_test.go` - lifecycle classification and notifier coverage
- `internal/acp/*_test.go` - ACP error classification, probe, and crash bundle tests
- `internal/api/core/*_test.go` - API/SSE conversion coverage
- `web/src/systems/session/` - typed session failure rendering if contract changes
- `packages/site/` - troubleshooting docs for failure kinds and crash bundles
- `.compozy/tasks/hermes/task_10.md` - QA plan must cover failure-kind traceability

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - defines lifecycle failure visibility as a selected hardening track

## Deliverables
- Typed failure-kind model for ACP and session lifecycle failures
- Persisted and API-visible failure diagnostics
- Downstream agent probe implementation and tests
- Redacted crash bundle support for actionable operator debugging
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] ACP startup, protocol, cancellation, and subprocess failures map to expected `FailureKind` values
  - [x] Session state transitions persist failure kind and redacted summary
  - [x] API/SSE conversions expose failure state without leaking secrets
  - [x] Probe timeout and cancellation paths return structured health output
- Integration tests:
  - [x] A failing mock ACP provider produces an observable failure event and persisted failure state
  - [x] Crash bundle generation captures bounded evidence and redacts configured secret patterns
  - [x] CLI or API session read path reports the same failure kind as the persisted record
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- Operators can distinguish provider crash, protocol failure, startup failure, and cancellation
- Session failure state is durable and visible across CLI/API/SSE surfaces
- Crash diagnostics are useful and redacted
- Affected backend and web tests pass
