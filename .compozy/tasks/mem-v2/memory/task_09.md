# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Build the Slice 1 memory observability layer so global and per-workspace `memory_events` authorities feed operator-facing observe/health surfaces, while SSE/log-facing payloads cannot leak prompt-only `<memory-context>` sections.

## Important Decisions

- Added `Store.ListMemoryEventSummaries` and a memory observability aggregator over the existing global/shared catalog plus existing per-workspace `<workspace>/.agh/agh.db` authorities.
- Workspace observability aggregation only opens a per-workspace DB when the DB file already exists. Health/observe reads must not create empty workspace DB authorities or double-count the shared global catalog.
- `Observer.QueryEvents` now accepts an optional `MemoryEventSource`, merges canonical memory summaries with registry summaries for non-session-scoped queries, sorts/clamps once, and filters registry `memory.*` rows when the memory source is active to avoid double-counting.
- Session-scoped observe queries intentionally remain registry-only for this slice because the current workspace resolver input is session-derived; later public route work can add explicit workspace-aware session fan-in.
- Added `internal/sse` memory-context scrubbers that handle literal `<memory-context>`/`<memory_context>` and JSON-escaped `\u003c...` forms, including unclosed prompt fences.
- SSE raw writes, observe payload conversion, memory operation payloads, and prompt-stream redaction now route through the shared scrubber before log/SSE-facing exposure.
- Decision event writes now carry the durable `workspace_id` into `memory_events`; recall events already had this path.

## Learnings

- `encoding/json` escapes `<memory-context>` as `\u003cmemory-context\u003e`; SSE hygiene tests must cover escaped and literal forms.
- The Memory v2 `memory_events` schema has `actor_kind` but no `actor_id`. Summary projections should emit an empty actor ID rather than querying a nonexistent column.
- SQL subqueries with `LIMIT` need aliases for computed `COALESCE(...)` columns so the outer select remains portable across SQLite execution paths.
- Adding workspace DB fan-in to health can accidentally create and reindex empty workspace DBs. Treat per-workspace observability DBs as authorities only when `agh.db` already exists.
- `Store.HealthStats` is lint-sensitive after fan-in; source accumulation now lives in small observability helpers rather than one high-complexity method.

## Files / Surfaces

- `internal/memory/observability.go`
- `internal/memory/observability_test.go`
- `internal/memory/store.go`
- `internal/memory/decision.go`
- `internal/observe/observer.go`
- `internal/observe/query.go`
- `internal/observe/observer_test.go`
- `internal/daemon/daemon.go`
- `internal/sse/scrub.go`
- `internal/sse/decode_test.go`
- `internal/api/core/sse.go`
- `internal/api/core/conversions.go`
- `internal/api/core/memory.go`
- `internal/api/core/prompt_stream.go`
- `internal/api/core/sse_hygiene_test.go`

## Errors / Corrections

- Initial observability tests created a workspace identity without first creating the workspace root; fixtures now create the root before `EnsureIdentity`.
- The first aggregator query used a nonexistent `actor_id` column; summary projection now returns an empty actor ID for Memory v2 events.
- The limited-event query initially failed because computed `session_id`/`agent_name` columns lacked aliases in the subquery; aliases are now explicit.
- `make lint` flagged `Store.HealthStats` gocyclo after source fan-in; extraction into a health accumulator fixed the lint issue without changing behavior.

## Ready for Next Run

- Task 09 passed focused aggregation/observe/SSE tests, race tests for touched packages, coverage checks (`internal/memory` 80.4%, `internal/sse` 86.4%), `git diff --check`, `make lint`, and full `make verify` before tracking updates.
- Next loop iteration should execute `task_10` (Extractor Hook, Inbox, and Runtime Queue).
