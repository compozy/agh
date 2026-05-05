---
status: completed
title: Memory Visibility and Future Interfaces
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 07: Memory Visibility and Future Interfaces

## Overview

Expose memory health and operation history now while preparing narrow interfaces for later context reference and provider-hook integration. This task focuses on observable memory behavior and future-safe seams; it explicitly avoids injecting memory context into runtime prompts in this slice.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-005, and task_02 health outputs before changing memory surfaces
- DO NOT integrate memory context refs into runtime prompts in this task
- DO NOT add provider-specific behavior behind generic memory APIs
- Operation history must redact secret content and bound result size
- CLI and API output must be consistent and typed
</critical>

<requirements>
- MUST add `agh memory health` with useful global and workspace memory health state
- MUST add `agh memory history` backed by a durable or queryable operation log
- MUST expose typed API support for memory health and operation history where needed
- MUST prepare interfaces for future context refs and provider hooks without runtime prompt integration
- MUST add tests for health, history filtering, redaction, and interface boundaries
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this task
</requirements>

## Subtasks
- [x] 7.1 Add memory health read model and CLI/API output wired to observe health where appropriate
- [x] 7.2 Add memory operation history persistence or query support with filters and bounded output
- [x] 7.3 Implement `agh memory health` and `agh memory history` command behavior
- [x] 7.4 Define future context-ref and provider-hook interfaces without wiring prompt injection
- [x] 7.5 Add tests for health output, history filtering, redaction, and interface-only future seams
- [x] 7.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

This task should give operators visibility into memory state and recent operations, not change how agents receive memory. Keep the future-facing interfaces small, documented, and uncalled by runtime prompt assembly until a later task explicitly owns that integration.

### Relevant Files
- `internal/cli/memory.go` - memory command expansion
- `internal/cli/client.go` - CLI client methods for memory API calls
- `internal/api/core/memory.go` - memory handlers and conversions
- `internal/api/contract/contract.go` - memory health and history DTOs
- `internal/memory/` - memory service, store, and operation model
- `internal/store/globaldb/global_db_observe.go` - operation history or health persistence if reused
- `web/src/routes/_app/settings/memory.tsx` - memory settings or health display if impacted
- `packages/site/` - memory CLI and operator docs

### Dependent Files
- `internal/memory/*_test.go` - health, history, and interface tests
- `internal/cli/*memory*_test.go` - CLI command output tests
- `internal/api/core/*memory*_test.go` - memory API contract tests
- `web/src/` - typed client or settings tests for memory health/history if surfaced
- `packages/site/` - docs updates for memory health/history commands
- `.compozy/tasks/hermes/task_10.md` - QA plan must include memory visibility coverage

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - includes memory visibility in selected hardening work
- [ADR-005: Memory Health and History Before Runtime Context References](adrs/adr-005-memory-health-history-before-runtime-contextrefs.md) - defers prompt integration while preparing future seams

## Deliverables
- `agh memory health` command and supporting typed data path
- `agh memory history` command with bounded, redacted operation history
- API support for memory health/history where needed by CLI or web
- Future context-ref/provider-hook interfaces without runtime prompt integration
- Tests covering redaction, filters, and health state
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] Memory health reports configured, degraded, and unavailable states accurately
  - [x] Memory history filters by workspace, scope, operation, and time where supported
  - [x] History output redacts secret or sensitive payload fields
  - [x] Future interfaces compile and remain unused by runtime prompt assembly
- Integration tests:
  - [x] CLI health/history commands return the same typed data as API handlers
  - [x] Operation history survives process restart if stored durably
  - [x] Web or docs updates reflect the new memory visibility behavior when surfaced
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- Operators can inspect memory health and recent memory operations
- Runtime prompt assembly remains unchanged by context refs in this task
- Future integration seams are explicit and testable
- Affected backend, CLI, web, and docs tests pass

## Implementation Notes

- Added typed memory health/history API support through direct HTTP and UDS routes.
- Added `agh memory health` and `agh memory history` with typed CLI client calls and history filters for workspace, scope, operation, time, and bounded limit.
- Extended the durable memory catalog operation log with scope, workspace, filename, operation, redacted bounded summary, and timestamp fields.
- Added future context-ref/provider-hook interfaces in `internal/memory` without runtime prompt integration.
- Regenerated OpenAPI and web generated types; no settings UI behavior was required for this slice.
- Updated site memory CLI/API/operator docs and Task 10 QA coverage notes.

## Verification Evidence

- `go test ./internal/memory ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli`
- `make codegen-check`
- `bun run --cwd web typecheck:raw`
- `bun run --cwd packages/site typecheck`
- `bun run --cwd packages/site build`
- `make verify`
