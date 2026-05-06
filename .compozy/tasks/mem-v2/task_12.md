---
status: completed
title: Session Lineage and Ledger Materialization
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
---

# Task 12: Session Lineage and Ledger Materialization

## Overview

Add the durable session-lineage and forensic-ledger slice required by Memory v2. This task makes `parent_session_id` durable where needed and materializes `ledger.jsonl` from existing durable session evidence without turning the ledger into a second live authority.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Session ledger`, `System Architecture`, and `Development Sequencing` steps 17-18.
- ACTIVATE `agh-code-guidelines`, `agh-cleanup-failure-paths`, and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside session lineage and ledger-materialization seams.
- TESTS REQUIRED: lineage persistence, session-end ledger write, restart safety, and forensic-only semantics must ship here.
- NO WORKAROUNDS: `ledger.jsonl` is a projection/materialization, not a second live source of truth.
</critical>

<requirements>
- MUST persist the session-lineage fields needed by the TechSpec, including `parent_session_id` where required.
- MUST implement `ledger.jsonl` materialization on session end from existing durable session evidence.
- MUST keep the live session DB/event store authoritative and the ledger forensic-only.
- MUST place unbound sessions and workspace-bound sessions in the correct storage layout.
- MUST leave the daemon/session lifecycle code with a thin hook into the ledger materializer.
</requirements>

## Subtasks
- [x] 12.1 Add or extend durable lineage fields in session storage.
- [x] 12.2 Implement the ledger materializer that emits `ledger.jsonl` at session end.
- [x] 12.3 Handle workspace-bound and unbound session layouts correctly.
- [x] 12.4 Add focused tests for lineage persistence, ledger output, restart safety, and forensic-only semantics.

## Implementation Details

See TechSpec `Session ledger`, `Filesystem layout`, and `Development Sequencing` steps 17-18. The ledger should be implemented as a pure reader/materializer invoked from lifecycle code, not as part of the live session DB writer path.

### Relevant Files
- `internal/store/sessiondb/session_db.go` — durable session-event store and a source of live truth.
- `internal/store/session_lineage.go` — existing lineage domain helpers.
- `internal/session/manager_start.go` — existing lineage/session-start behavior.
- `internal/session/session.go` — durable session model and metadata fields.
- `internal/session/manager_lifecycle.go` — likely session-end hook point for ledger materialization.
- `.compozy/tasks/mem-v2/analysis/analysis_session-ledger-retention.md` — forensic retention and competitor evidence.

### Dependent Files
- `internal/api/contract/contract.go` — later public contract work may expose ledger or lineage metadata.
- `internal/daemon/boot.go` — later daemon wiring will connect the ledger materializer.
- `web/src/systems/session/components/session-inspector.tsx` — later UI task may surface lineage/ledger metadata.
- `.compozy/tasks/mem-v2/task_14.md` — public contract task depends on final lineage/ledger fields.
- `.compozy/tasks/mem-v2/task_19.md` — daemon wiring task depends on this materializer.

### Related ADRs
- [ADR-006: Session Ledger Hybrid (events.db Live + ledger.jsonl Forensic)](adrs/adr-006.md) — normative ledger behavior.
- [ADR-004: Stable Workspace ID via .agh/workspace.toml](adrs/adr-004.md) — constrains ledger path layout.
- [ADR-012: Slice 1 Fat Scope — Single TechSpec with Four Eixos](adrs/adr-012.md) — explains why lineage/ledger ships in Slice 1.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none yet — checked surfaces are provider hooks, extension host routes, and SDKs; they remain unchanged in this task.
- Agent manageability: no public route/CLI change lands here yet, but later inspect/status surfaces must reflect lineage and ledger semantics faithfully.
- Config lifecycle: none — checked surfaces are memory/session settings, config keys, and docs; no new config key lands in this task.

### Web/Docs Impact

- `web/`: none yet — checked surfaces are generated types and session inspector; UI work is deferred to later tasks.
- `packages/site`: none yet — checked surfaces are runtime session/workspace/memory docs and references; docs update after public surfaces stabilize.

## Deliverables

- Durable session-lineage fields extended as required by the TechSpec.
- Session-end `ledger.jsonl` materializer implemented as a forensic projection.
- Focused lineage/ledger tests covering workspace-bound and unbound layouts.

## Tests

- Unit tests:
  - [ ] Session lineage fields persist and round-trip correctly through the store.
  - [ ] Ledger materialization writes the expected JSONL shape from durable event/session inputs.
  - [ ] Unbound and workspace-bound ledger layouts resolve to the correct target paths.
- Integration tests:
  - [ ] Session end materializes `ledger.jsonl` once without changing live authority or replay semantics.
  - [ ] Restart or interrupted shutdown does not corrupt lineage or ledger outputs.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/codex/codex-rs/rollout/src/recorder.rs`
- `.resources/claude-code/memdir/memdir.ts`
- `.resources/hermes/hermes_state.py`
- `.resources/goclaw/internal/sessions/key.go`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Session lineage is durable and `ledger.jsonl` is materialized as a forensic projection only.
- Later inspect/manageability surfaces can expose ledger metadata without inventing a second live authority.
