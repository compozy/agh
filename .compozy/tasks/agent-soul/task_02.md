---
status: completed
title: Persist Soul Snapshots and Authoring Revisions
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Persist Soul Snapshots and Authoring Revisions

## Overview

Add durable storage for resolved Soul snapshots and managed authoring history. This task creates the schema foundation needed by sessions, task claim provenance, rollback, history, and audit surfaces without changing prompt assembly yet.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, and every ADR before changing persistence.
- REFERENCE TECHSPEC for exact table names, DDL, columns, constraints, retention, and delete targets.
- FOCUS ON WHAT must be stored: snapshots, revisions, digests, diagnostics, provenance, and audit metadata.
- MINIMIZE CODE in task execution notes; implement migrations in the existing store style.
- TESTS REQUIRED for fresh DB, reopen, rollback data, constraints, cascade behavior, and no legacy bridge.
- NO WORKAROUNDS: greenfield alpha means hard cut, numbered migration, and no compatibility fallback.
</critical>

<requirements>
- MUST activate `agh-schema-migration`, `agh-code-guidelines`, and `golang-pro`.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST add the numbered global DB migration specified by `_techspec_soul.md` for `agent_soul_snapshots` and `agent_soul_revisions`.
- MUST store `soul_digest`, config digest/provenance, source path, validation status, compact projection, full read model fields, and redacted diagnostics as specified.
- MUST keep mutable authoring history separate from immutable runtime snapshot provenance.
- MUST add store methods used by later authoring, session, and task claim tasks.
- MUST avoid JSON metadata as the authority for fields that need deterministic querying or retention.
</requirements>

## Subtasks
- [x] 2.1 Add the Soul schema migration and register it in the global DB migration registry.
- [x] 2.2 Add store methods for inserting, reading, and listing soul snapshots.
- [x] 2.3 Add store methods for append-only authoring revisions and rollback lookup.
- [x] 2.4 Add constraints, indexes, and retention behavior required by the TechSpec.
- [x] 2.5 Add fresh-DB, reopen, failed-migration, and constraint tests.
- [x] 2.6 Confirm Heartbeat migration numbering remains reserved for task_06 and does not conflict.

## Implementation Details

Follow the existing SQLite migration conventions and keep schema ownership inside the global DB package unless the current store architecture indicates a more specific owner. The schema must make future session and task provenance deterministic without making arbitrary metadata JSON the source of truth.

### Relevant Files
- `internal/store/globaldb/global_db.go` - global DB boot and migration registry.
- `internal/store/globaldb/migrate_workspace.go` - existing migration precedent to preserve or extend.
- `internal/store/globaldb/` - destination for soul snapshot and revision store methods.
- `internal/soul/` - source types for persisted snapshots and revision payloads.
- `internal/store/schema.go` - only if the migration registry requires shared changes.

### Dependent Files
- `internal/store/globaldb/*_test.go` - migration and store method coverage.
- `internal/soul/*_test.go` - persisted snapshot shape expectations if shared test fixtures are useful.
- `.compozy/tasks/agent-soul/task_03.md` - consumes revision storage for managed authoring.
- `.compozy/tasks/agent-soul/task_04.md` - consumes snapshots for sessions and task claim provenance.
- `.compozy/tasks/agent-soul/task_06.md` - adds Heartbeat storage after this migration foundation.

### Related ADRs
- [ADR-001: Optional Scoped SOUL.md Persona Artifact](adrs/adr-001.md) - requires scoped source identity.
- [ADR-003: Soul Snapshot Lifecycle](adrs/adr-003.md) - requires session-start and claim-time provenance.
- [ADR-006: Managed Soul Authoring in v1](adrs/adr-006.md) - requires revision history and rollback.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: store data must support later Host API, hooks, tools/resources, SDKs, bundles, and registry audit surfaces without schema ambiguity.
- Agent manageability: no external route in this task, but revision/snapshot APIs must be ready for CLI/HTTP/UDS and Host API consumers.
- Config lifecycle: persist config digest/provenance used by `[agents.soul]`; do not add new config keys beyond task_01.

### Web/Docs Impact
- Web impact: generated types and UI consumers are not changed in this task.
- Docs impact: schema details are internal; task_15 must document operator-visible history/rollback behavior, not raw table internals.

## Deliverables
- Numbered global DB migration for Soul snapshots and authoring revisions.
- Store methods for snapshot insert/read/list and revision append/history/rollback lookup.
- Constraints and indexes for deterministic lookup by workspace, agent, session, task run, digest, and revision.
- Tests for migration ordering, fresh DB, reopen, constraints, and store behavior.
- Completion evidence proving no compatibility bridge or unnumbered schema mutation was added.

## Tests
- Unit tests:
  - [x] Migration creates all Soul tables, constraints, and indexes on a fresh DB.
  - [x] Reopening an initialized DB does not reapply migrations or lose rows.
  - [x] Store rejects duplicate or malformed snapshot/revision records according to constraints.
  - [x] Revision history is append-only and rollback lookup returns the intended prior version.
  - [x] Redacted diagnostics and config provenance persist without leaking raw prompt-only data.
- Integration tests:
  - [x] Resolver output from task_01 can be persisted and read back with the same digest.
  - [x] Failed migration leaves schema state consistent and returns a wrapped error.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec.md` - aggregate side-table and sequencing decisions.
- `_techspec_soul.md` - normative Soul storage requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_hermes.md` - Hermes session snapshot and durable run metadata findings.
- `.resources/hermes/run_agent.py:4810-4844` - prompt/session snapshot precedent.
- `.resources/hermes/agent/prompt_builder.py:1144-1180` - prompt construction and snapshot relevance.
- `.resources/openclaw/src/agents/system-prompt.ts:950-1006` - composed context provenance precedent.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Soul snapshot and revision data can support session provenance, task provenance, and managed rollback without prompt-time file I/O.
- Schema follows greenfield hard-cut and numbered migration rules.
