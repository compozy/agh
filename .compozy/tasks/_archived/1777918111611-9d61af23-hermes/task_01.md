---
status: completed
title: Persistence and Retry Foundations
type: backend
complexity: critical
dependencies: []
---

# Task 01: Persistence and Retry Foundations

## Overview

Establish the durable persistence and retry primitives needed by every Hermes hardening track. This task introduces a first-class migration runner for global and session databases, records schema state, and adds a shared jittered retry package that later tasks can use without inventing local backoff loops.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 before changing persistence foundations
- DO NOT hand-edit `go.mod`; use `go get` if a dependency is truly required
- PRESERVE SQLite determinism and test isolation with `t.TempDir()`
- DO NOT add compatibility branches for old alpha database state; greenfield cleanup is allowed
- EVERY retry helper must accept `context.Context` and stop immediately on cancellation
</critical>

<requirements>
- MUST add a migration runner with deterministic ordering and idempotent application
- MUST create and maintain a `schema_migrations` record for global and session database schemas
- MUST wire migration execution into global DB and session DB initialization
- MUST add focused tests for fresh DB creation, repeated boot, and migration failure handling
- MUST add a shared jittered backoff helper for retrying transient operations without `time.Sleep()` in orchestration paths
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this foundation work
</requirements>

## Subtasks
- [x] 1.1 Design the migration runner API and schema record format for global and session SQLite stores
- [x] 1.2 Replace inline schema setup with ordered migrations in `internal/store`, `globaldb`, and `sessiondb`
- [x] 1.3 Add test coverage for fresh initialization, repeated initialization, partial failure, and migration ordering
- [x] 1.4 Add a shared retry/backoff package with context cancellation, jitter bounds, and deterministic test hooks
- [x] 1.5 Update call sites that currently need retry semantics but can be safely migrated in this foundation task
- [x] 1.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

Use this task to create the durable substrate for later work. Keep migration definitions close to their owning store packages, but keep migration execution reusable enough for both global and per-session databases. The retry helper should be small, context-aware, and usable by automation, MCP auth, and process recovery tasks without depending on those packages.

### Relevant Files
- `internal/store/schema.go` - shared schema helpers and migration entry points
- `internal/store/sqlite.go` - SQLite connection setup and migration invocation
- `internal/store/globaldb/global_db.go` - global database initialization
- `internal/store/sessiondb/session_db.go` - per-session database initialization
- `internal/store/globaldb/migrate_workspace.go` - existing migration precedent to fold into the new model
- `internal/procutil/` - process helpers that may consume retry primitives later
- `internal/retry/` - new shared retry/backoff package destination

### Dependent Files
- `internal/store/globaldb/*_test.go` - global DB migration and boot coverage
- `internal/store/sessiondb/*_test.go` - session DB migration and boot coverage
- `internal/retry/*_test.go` - jitter, cancellation, and max-attempt coverage
- `.compozy/tasks/hermes/task_02.md` - depends on durable persistence
- `.compozy/tasks/hermes/task_04.md` - depends on migration and retry primitives
- `.compozy/tasks/hermes/task_05.md` - depends on migration and retry primitives
- `.compozy/tasks/hermes/task_06.md` - depends on migration and retry primitives

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - defines persistence and retry as shared foundations

## Deliverables
- Durable migration runner used by global and session databases
- `schema_migrations` persistence for both schema families
- Shared context-aware retry/backoff helper with jitter and deterministic tests
- Updated store initialization tests proving idempotent boot and failed migration handling
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] Migration runner applies ordered migrations once and records their version
  - [x] Re-running database initialization does not reapply completed migrations
  - [x] Failed migrations return wrapped errors and do not mark success
  - [x] Retry helper respects context cancellation, max attempts, and jitter bounds
- Integration tests:
  - [x] Global DB opens successfully from an empty directory and from a previously initialized directory
  - [x] Session DB opens successfully from an empty session directory and from a previously initialized directory
  - [x] Existing store tests continue to pass under the migration runner
- Test coverage target: >=80%
- All tests must pass

## Completion Notes

- Added `internal/store` migration primitives with ordered/idempotent application, per-migration transactions, checksum/name integrity checks, and durable `schema_migrations` rows.
- Wired global and session DB initialization through the migration runner while preserving the existing global pre-run schema normalizers before recording canonical v1 schema state.
- Added `internal/retry` with context-aware retry, jittered delay calculation, deterministic hooks, and cancellation-aware waiting; migrated `internal/bridgesdk.RetryDo` delay/wait behavior onto the shared primitive.
- `web/` and `packages/site` impact assessment: no required code, docs, typed-client, settings page, example, story, or test update. Task 01 changes are internal Go persistence/retry foundations and do not alter public contracts.
- Verification evidence:
  - `go test -cover ./internal/store ./internal/store/globaldb ./internal/store/sessiondb ./internal/retry ./internal/bridgesdk`
  - `make verify`

## Success Criteria
- Store initialization no longer relies on ad hoc one-shot schema setup
- Global and session databases can prove their schema version durably
- Retry behavior is centralized, context-aware, and ready for later Hermes tracks
- `make test` passes for affected backend packages
