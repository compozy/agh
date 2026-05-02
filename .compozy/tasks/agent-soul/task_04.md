---
status: completed
title: Integrate Soul With Sessions, Prompt Context, and Task Provenance
type: backend
complexity: critical
dependencies:
  - task_01
  - task_02
  - task_03
---

# Task 04: Integrate Soul With Sessions, Prompt Context, and Task Provenance

## Overview

Wire resolved Soul profiles into runtime context, session lifecycle, prompt assembly, explicit refresh, spawn semantics, and task claim provenance. This task is the behavioral integration point for `SOUL.md`, while preserving existing task-run lease and orchestration authorities.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, ADR-001 through ADR-006, and current session/task code before editing.
- REFERENCE TECHSPEC for prompt ordering, compact context, full read model, snapshot lifecycle, and failure matrix.
- FOCUS ON WHAT must hold: deterministic inclusion, session snapshots, explicit refresh, claim-time provenance, and no file I/O in `ClaimNextRun`.
- MINIMIZE CODE in notes; implement with existing session and task abstractions.
- TESTS REQUIRED for prompt inclusion, refresh locking, spawn behavior, task metadata provenance, and invalid-soul failures.
- NO WORKAROUNDS: do not make `SOUL.md` a task queue, heartbeat, liveness, network presence, or capability authority.
</critical>

<requirements>
- MUST activate `agh-code-guidelines` and `golang-pro` before editing production Go.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST include resolved Soul in prompt/context exactly where `_techspec_soul.md` specifies.
- MUST snapshot `soul_digest` and profile projection at session start and on explicit refresh.
- MUST add compact Soul projection to `/agent/context` data production and prepare full read-model data for later routes.
- MUST record task claim Soul provenance without performing file I/O inside `ClaimNextRun`.
- MUST preserve spawn semantics: spawned sessions use the target agent's own `SOUL.md` plus explicit spawn overlays, with no implicit parent Soul inheritance.
- MUST ensure invalid existing `SOUL.md` fails closed consistently across session start, refresh, and task claim.
</requirements>

## Subtasks
- [x] 4.1 Wire resolved Soul snapshots into session start and prompt-context assembly.
- [x] 4.2 Add explicit session Soul refresh with bounded detached lifetime and session-scoped locking.
- [x] 4.3 Add compact Soul projection to agent context and prepare full read-model sources.
- [x] 4.4 Persist task-run claim provenance from pre-resolved snapshots without claim-time file I/O.
- [x] 4.5 Implement spawn/subagent lineage metadata without implicit parent Soul inheritance.
- [x] 4.6 Add behavioral tests for prompt, session, claim, refresh, spawn, and failure-matrix scenarios.

## Implementation Details

Treat the resolver and persisted snapshots from tasks 01-03 as the only Soul authority. The task claim path must use already-resolved snapshot data and must not read `SOUL.md` while claiming work.

### Relevant Files
- `internal/session/manager_helpers.go` - session setup and resolved agent state wiring.
- `internal/session/spawn.go` - spawn/subagent semantics and lineage metadata.
- `internal/session/manager_prompt.go` - prompt turn integration and refresh locking.
- `internal/daemon/prompt_input_composite.go` - prompt input assembly.
- `internal/daemon/prompt_sections.go` - prompt section ordering and rendering.
- `internal/situation/render.go` - compact `/agent/context` projection assembly.
- `internal/task/lease_manager.go` - task claim integration boundary.
- `internal/store/globaldb/global_db_task_claim.go` - claim metadata persistence.
- `internal/api/core/agent_channels.go` - existing agent context route sources.
- `internal/soul/` - resolver, authoring, and snapshot types.

### Dependent Files
- `internal/session/*_test.go` - session start, refresh, prompt, spawn, and locking tests.
- `internal/daemon/*_test.go` - prompt section and context projection tests.
- `internal/task/*_test.go` - claim provenance and no file-I/O regression tests.
- `internal/store/globaldb/*_test.go` - task metadata persistence tests.
- `.compozy/tasks/agent-soul/task_10.md` - exposes the contract after runtime data exists.
- `.compozy/tasks/agent-soul/task_11.md` - exposes route parity after core behavior exists.

### Related ADRs
- [ADR-001: Optional Scoped SOUL.md Persona Artifact](adrs/adr-001.md) - defines strict runtime boundaries.
- [ADR-002: Soul Prompt and Read Model Exposure](adrs/adr-002.md) - defines direct prompt inclusion and read models.
- [ADR-003: Soul Snapshot Lifecycle](adrs/adr-003.md) - defines session and task provenance.
- [ADR-004: No Implicit Parent Soul Inheritance](adrs/adr-004.md) - defines spawn behavior.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: emit stable snapshot/provenance data for hooks, Host API reads, tools/resources, SDKs, and future bundles in later tasks.
- Agent manageability: prepares data for `agh session soul refresh`, `/agent/context`, and full read-model endpoints; route exposure deferred to tasks 10-12.
- Config lifecycle: consumes `[agents.soul]` resolver behavior; no additional config keys.

### Web/Docs Impact
- Web impact: generated types and frontend consumers must be updated after contract exposure in tasks 10 and 14.
- Docs impact: task_15 must document session snapshot, explicit refresh, task provenance, and spawn inheritance rules.

## Deliverables
- Soul prompt/context integration with deterministic ordering and redaction.
- Session snapshot and explicit refresh behavior.
- Compact `/agent/context` Soul projection and full read-model backing data.
- Task claim provenance using no claim-time file I/O.
- Spawn semantics and lineage metadata consistent with ADR-004.
- Unit and integration tests for all runtime behavior above.

## Tests
- Unit tests:
  - [x] Prompt assembly includes Soul projection in the approved order and respects truncation.
  - [x] Invalid `SOUL.md` fails closed at session start with a redacted diagnostic.
  - [x] Explicit refresh updates the session snapshot under a session-scoped lock.
  - [x] Spawned sessions do not inherit parent `SOUL.md` implicitly.
  - [x] Task claim metadata includes `soul_digest` and provenance from existing snapshots.
  - [x] `ClaimNextRun` path does not read the filesystem for `SOUL.md`.
- Integration tests:
  - [x] Session start, prompt, refresh, and task claim flow works across daemon/store boundaries.
  - [x] `/agent/context` internal core data includes compact Soul projection without exposing full body where forbidden.
  - [x] Restart/reopen preserves session and task provenance.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec_soul.md` - session, prompt, task, and spawn behavior.
- `.compozy/tasks/agent-soul/analysis/analysis_hermes.md` - prompt snapshot and run provenance findings.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw.md` - prompt-context injection findings.
- `.resources/hermes/agent/prompt_builder.py:1028-1054` - Soul-like prompt insertion precedent.
- `.resources/hermes/run_agent.py:4810-4844` - session prompt snapshot precedent.
- `.resources/openclaw/src/agents/system-prompt.ts:950-1006` - composed context precedent.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- `SOUL.md` affects authored identity context only through validated, snapshotted runtime paths.
- Task ownership, scheduler authority, session liveness, and network presence remain governed by existing runtime primitives.
