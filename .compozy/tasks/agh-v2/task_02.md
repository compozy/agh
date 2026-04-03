---
status: pending
domain: Database
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
  - task_00
---

# Task 02: Store Package

## Overview

Implement the `internal/store` package that provides SQLite storage for both per-session event databases and the global index database. This package owns schema definitions, migrations, and query methods. Uses WAL mode and single-writer pattern for crash safety.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST use `modernc.org/sqlite` (pure Go, no CGo)
- MUST support two DB types: per-session `events.db` and global `agh.db`
- MUST implement schemas as defined in TechSpec (events, token_usage, sessions, event_summaries, token_stats, permission_log tables)
- MUST use WAL mode and NORMAL synchronous for all databases
- MUST implement single-writer pattern for per-session DBs
- MUST support atomic `meta.json` writes (temp file + rename)
- MUST implement event recording with monotonic sequence numbers and turn IDs
- MUST implement event querying with filters (type, since, limit, agent, follow-compatible)
- MUST implement global DB session index operations (register, update state, list)
- MUST implement token stats aggregation in global DB
- MUST implement permission log writes in global DB
</requirements>

## Subtasks

- [x] 2.1 Define store types (SessionEvent, TokenUsage, SessionInfo, EventQuery, etc.)
- [x] 2.2 Implement per-session DB: open, schema creation, close
- [x] 2.3 Implement per-session event recording with sequence numbers
- [x] 2.4 Implement per-session token usage recording
- [x] 2.5 Implement per-session event querying with filters
- [x] 2.6 Implement global DB: open, schema creation, close
- [x] 2.7 Implement global DB session index (register, update, list, reconcile)
- [x] 2.8 Implement global DB event summaries, token stats, permission log
- [x] 2.9 Implement atomic meta.json read/write
- [x] 2.10 Implement turn-structured history query (group events by turn_id for conversation view)
- [x] 2.11 Handle SQLite open failures gracefully (corruption detection, rename + recreate)

## Implementation Details

Create the following files:

- `internal/store/store.go` — Types, interfaces
- `internal/store/session_db.go` — Per-session SQLite operations
- `internal/store/global_db.go` — Global SQLite operations
- `internal/store/schema.go` — DDL statements, migration logic
- `internal/store/meta.go` — Atomic meta.json read/write

### Relevant Files

- `.compozy/tasks/agh-v2/_techspec.md` — SQLite schemas section

### Old Project Reference

- `.old_project/internal/state/db.go` — SQLite wrapper, WAL mode, schema initialization
- `.old_project/internal/state/queries.go` — SQL query patterns for events and sessions
- `.old_project/internal/state/writer.go` — Single-writer pattern and transaction handling
- `.old_project/internal/state/snapshot.go` — Per-session snapshot patterns

### Related ADRs

- [ADR-006: Dual SQLite Storage](../adrs/adr-006.md) — Per-session + global DB design

## Deliverables

- `internal/store/` package with per-session and global DB operations
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests with real SQLite temp DBs **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Open per-session DB creates schema correctly
  - [x] Record event with auto-incrementing sequence
  - [x] Record token usage with nullable fields (nil values stored as NULL)
  - [x] Query events by type filter
  - [x] Query events by time range (--since)
  - [x] Query events with limit
  - [x] Query events ordered by sequence
  - [x] Global DB: register session, update state, list sessions
  - [x] Global DB: write event summary
  - [x] Global DB: update token stats aggregation
  - [x] Global DB: write permission log entry
  - [x] Atomic meta.json: write and read back
  - [x] Atomic meta.json: concurrent writes don't corrupt
  - [x] WAL mode enabled on open
  - [x] Turn-structured history: events grouped by turn_id with correct ordering
  - [x] SQLite corruption: open failure triggers rename + recreate
  - [x] Disk-full: write failure returns error, does not panic
- Integration tests:
  - [x] Full lifecycle: open DB → write events → query → close → reopen → query still works
  - [x] Multiple concurrent readers with single writer
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes
- Per-session DB creates all tables and indexes from TechSpec schema
- Global DB creates all tables and indexes from TechSpec schema
- Event queries return correct results with all filter combinations
