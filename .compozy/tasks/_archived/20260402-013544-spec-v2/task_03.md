---
status: completed
domain: State
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_01
---

# Task 3: SQLite State Layer

## Overview
Implement the SQLite-based state persistence layer with WAL mode, a single-writer goroutine pattern for serialized writes via a buffered channel, concurrent read support, and all 5 database tables (agents, workgroups, blackboard, status, events) with indexes.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST use modernc.org/sqlite (pure Go, no CGO) per docs/spec-v2/00-executive-summary.md
- MUST initialize SQLite in WAL mode with PRAGMA settings per docs/spec-v2/02-kernel.md
- MUST create all 5 tables with indexes per docs/spec-v2/08-data-models.md schema
- MUST implement single-writer goroutine consuming from a buffered channel (256 capacity)
- MUST support concurrent reads from any goroutine (WAL mode permits this)
- MUST implement graceful drain on shutdown (flush pending writes with timeout)
- MUST implement read helpers for all tables (agents, workgroups, blackboard, status, events)
- MUST implement snapshot-on-close logic for workgroup state compaction
- MUST store one SQLite file per session at sessions/{xid}/session.db
- MUST handle WAL checkpoint on close: PRAGMA wal_checkpoint(TRUNCATE)
</requirements>

## Subtasks
- [x] 3.1 Implement SQLite initialization with WAL mode and PRAGMA settings
- [x] 3.2 Create all 5 tables with indexes via schema DDL
- [x] 3.3 Implement single-writer goroutine with buffered channel pattern
- [x] 3.4 Implement read helpers for all tables (query by scope, agent, type, time range)
- [x] 3.5 Implement snapshot-on-close (query workgroup state, generate summary, insert snapshot row)
- [x] 3.6 Implement graceful shutdown (drain channel, WAL checkpoint, close)

## Implementation Details
Refer to docs/spec-v2/02-kernel.md for SQLite setup and write serialization pattern. Refer to docs/spec-v2/08-data-models.md for complete SQL schema.

### Relevant Files
- `docs/spec-v2/02-kernel.md` — SQLite setup, WAL mode, writer goroutine
- `docs/spec-v2/08-data-models.md` — complete SQL schema for all 5 tables
- `docs/spec-v2/04-workgroups.md` — snapshot-on-close behavior

### Dependent Files
- `internal/kernel/types.go` — WriteOp struct, config types
- `internal/config/` — session directory paths

## Deliverables
- internal/state/db.go — SQLite initialization and WAL setup
- internal/state/writer.go — single-writer goroutine
- internal/state/queries.go — read helpers for all tables
- internal/state/snapshot.go — snapshot-on-close logic
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for concurrent read/write **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Schema creation succeeds with all 5 tables and indexes
  - [x] WAL mode is enabled after initialization
  - [x] Single writer serializes concurrent channel sends into sequential inserts
  - [x] Blackboard append persists with correct scope and author
  - [x] Status update creates row with correct state and task
  - [x] Event log persists with correct type and JSON data
  - [x] Read helpers filter by scope, agent, type, and time range
  - [x] Graceful drain flushes all pending writes before close
  - [x] WAL checkpoint runs on close
- Integration tests:
  - [x] 1000 concurrent writes via channel produce exactly 1000 rows
  - [x] Concurrent reads during writes return consistent data (no corruption)
  - [x] Snapshot-on-close generates summary row with correct content
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Race detector passes with concurrent read/write workload
- Snapshot-on-close preserves workgroup outcome data
