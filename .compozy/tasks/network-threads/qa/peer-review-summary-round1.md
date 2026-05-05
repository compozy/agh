# Peer Review Summary Round 1

## Verdict

`NEEDS_REWORK`

Claude/Opus found the product direction coherent with the greenfield hard-cut posture, but not execution-ready. The main problem is not the conceptual model; it is that several implementation-critical details are still underspecified enough that implementation agents would need to invent behavior.

## Counts

- Blockers: 8
- Nits: 10

## Blockers

- `B-001` — `Integration Points / Impact Analysis (SD-011 extensibility)`: missing explicit extensibility, agent-manageability, hooks, tools/resources, bundles, registries, bridge SDK, MCP, and config lifecycle analysis.
- `B-002` — `Data Models — AGH Runtime store shape`: missing concrete numbered SQLite migration, DDL, transaction shape, migration tests, and explicit deletion of the old `EnsureSchema`-style table creation path.
- `B-003` — `Implementation Design — Core Interfaces`: missing concrete Go definitions for summaries, queries, messages, `Work`, `WorkState`, reason codes, validators, and hard-cut symbol renames.
- `B-004` — `Safety Invariants #8 / API Endpoints — POST /api/network/channels/{channel}/directs/resolve`: missing deterministic direct-room resolution algorithm, unique constraints, and concurrent resolve semantics.
- `B-005` — `Safety Invariants #5 / #6 — Cross-container work_id enforcement`: missing durable `network_work`-style binding table or equivalent authoritative store enforcement.
- `B-006` — `Implementation Design — relationship of work_id to task_runs`: missing explicit statement that `work_id` is network-only in v0 or, alternatively, exactly how it relates to `task_runs` without introducing a second queue.
- `B-007` — `Safety Invariants #9 — same-transaction summary updates`: missing concrete same-transaction write/UPSERT strategy for timeline rows, conversation summaries, and work activity.
- `B-008` — `Data Models / API Endpoints — RFC 004 trust integration`: missing v1 trust signed-field set for `surface`, `thread_id`, `direct_id`, and `work_id`.

## Nits

- `N-001` — add an explicit web route map for the new TanStack Router shape.
- `N-002` — enumerate CLI commands, flags, and structured output shapes.
- `N-003` — declare that `ConversationStore` is consumed in `internal/network` and implemented in `internal/store/globaldb`.
- `N-004` — add an explicit old-to-new Go symbol rename list for interaction/work lifecycle names.
- `N-005` — align MVP tasks 01-08 with the build-order numbering.
- `N-006` — state that archived historical `.compozy/tasks/_archived/*` artifacts may keep old terminology.
- `N-007` — note that expanding `direct_room` beyond two peers is a future schema migration trigger.
- `N-008` — define metric names, units, and cardinality keys.
- `N-009` — add the 80% touched-package coverage floor and Linux-Race parity expectations.
- `N-010` — outline the rewritten bundled network skill sections and example envelopes.

## Affected Sections

- `_techspec.md`: MVP boundary, architectural boundaries, implementation design, data models, schema migration, API endpoints, integration points, impact analysis, testing approach, monitoring/observability, safety invariants.
- `adr-001.md`: direct-room participant cap and future schema implications.
- `adr-002.md`: hard-cut Go symbol rename list.
- `adr-003.md`: likely no structural change, but examples may need to reference the finalized signed-field and surface model.

## Artifacts

- Raw event stream: `.compozy/tasks/network-threads/qa/peer-review-result-round1.json`
- Clean final JSON: `.compozy/tasks/network-threads/qa/peer-review-final-round1.json`
- Stderr: `.compozy/tasks/network-threads/qa/peer-review-result-round1.err`

## Notes

`peer-review-result-round1.err` contains a non-fatal workspace extension discovery warning for `.compozy/extensions/cy-qa-workflow` using an unknown hook event `plan.pre_resolve_task_runtime`. The `compozy exec` run completed successfully.
