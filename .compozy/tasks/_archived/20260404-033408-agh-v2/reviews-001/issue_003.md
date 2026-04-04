---
status: resolved
file: internal/store/global_db.go
line: 376
severity: high
author: claude-code
provider_ref:
---

# Issue 003: UpdateTokenStats uses session_id as primary key, collapsing per-agent stats

## Review Comment

The INSERT at line 376 uses `update.SessionID` for both the `id` (primary key) and `session_id` columns:

```go
update.SessionID,   // id
update.SessionID,   // session_id
update.AgentName,   // agent_name
```

The `ON CONFLICT(id)` clause then merges all updates for the same session into one row, regardless of `agent_name`. If a session uses multiple agents (e.g., "coder" then "reviewer"), the second agent's stats overwrite the first agent's `agent_name` (line 352: `agent_name = excluded.agent_name`) while accumulating their token counts together.

This causes: (1) per-agent token stats within a session are lost, (2) `ListTokenStats` filtering by `agent_name` misses data, (3) the `agent_name` column becomes unreliable.

The schema also lacks a `UNIQUE(session_id, agent_name)` constraint (`internal/store/schema.go:70-81`).

**Suggested fix:** Generate a proper unique `id` (e.g., `newID("ts")`) and add a `UNIQUE(session_id, agent_name)` constraint to the schema, then change the upsert to `ON CONFLICT(session_id, agent_name)`.

## Triage

- Decision: `valid`
- Notes: `UpdateTokenStats` inserts `update.SessionID` into both `id` and `session_id`, then upserts on `id`. That collapses all token usage for a session into one row and overwrites `agent_name`, so per-agent token stats are lost. The schema also lacks the composite uniqueness needed for correct upserts.
