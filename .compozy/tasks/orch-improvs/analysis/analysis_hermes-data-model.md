# Analysis: hermes-data-model

Read-only exploration of `.resources/hermes/` (kanban data-model surface) for the AGH task `orch-improvs`. Cross-referenced with AGH `internal/task/` and `internal/store/globaldb/`.

## Scope
- Path explored: `.resources/hermes/hermes_cli/kanban_db.py`, `tests/hermes_cli/test_kanban_db.py`, `tests/hermes_cli/test_kanban_boards.py`, `tests/hermes_cli/test_kanban_core_functionality.py` (sampled by grep), `website/docs/user-guide/features/kanban.md`, `RELEASE_v0.10.0.md`, `RELEASE_v0.11.0.md`, `RELEASE_v0.12.0.md`.
- Topic: kanban persistence — schema, transitions, concurrency, audit, multi-tenancy.
- Files read in full vs. sampled: read in full — `hermes_cli/kanban_db.py` (3326 lines), `tests/hermes_cli/test_kanban_db.py` (730 lines), `website/docs/user-guide/features/kanban.md`. Sampled by targeted grep — `test_kanban_boards.py` (slug + path tests), `test_kanban_core_functionality.py` (legacy backfill, run_id propagation, CAS races). Release notes scanned for `kanban` keywords.
- Total available files: 4 kanban Python sources/tests, ~5 docs/release files referenced.

## Overview

Hermes kanban persists every collaboration primitive in a single SQLite file (`<root>/kanban.db` or `<root>/kanban/boards/<slug>/kanban.db`) with five tables — `tasks`, `task_links`, `task_comments`, `task_events`, `task_runs`, plus the gateway-facing `kanban_notify_subs`. The schema is intentionally narrow: a task is a logical unit, a run is one attempt, a link is a parent→child edge, an event is an immutable audit row, and a subscription is a gateway delivery cursor. Multi-board isolation is achieved entirely through filesystem layout (one SQLite file per board) — there is no `board_id` column anywhere.

Concurrency is handled with WAL mode + `BEGIN IMMEDIATE` write transactions + compare-and-swap (CAS) updates on `tasks.status` and `tasks.claim_lock`. SQLite serializes writers via the WAL lock; the CAS pattern (`UPDATE ... WHERE status='ready' AND claim_lock IS NULL`) means losers get `rowcount=0` and quietly bow out — no distributed-lock machinery, no retry loops. Stale claims expire by TTL (`claim_expires`, 15-minute default) and a separate `release_stale_claims` sweep returns them to `ready`. Crash detection adds host-local PID liveness checks (with Linux `/proc/<pid>/status` zombie filtering) and a per-task `max_runtime_seconds` watchdog with SIGTERM/SIGKILL escalation.

The `tasks`/`task_runs` split is the most load-bearing pattern. `tasks` stores logical state (status, assignee, dependencies, workspace); `task_runs` stores per-attempt facts (claim lock, lease, PID, summary, metadata, error). `tasks.current_run_id` is a denormalized pointer the writers maintain via CAS, with an explicit invariant: `current_run_id IS NULL` ⇔ the pointed-to run row is in a terminal state. Defensive code closes leaked runs as `outcome='reclaimed'` whenever the pointer is reset (re-claim, unblock, archive). When a terminal transition (complete/block) is invoked on a never-claimed task, a synthetic zero-duration run is inserted so handoff fields (`summary`, `metadata`) are never silently dropped.

Migrations are not numbered — `init_db` re-runs `CREATE TABLE IF NOT EXISTS` plus `_migrate_add_optional_columns`, an `ALTER TABLE ADD COLUMN` cascade gated by `PRAGMA table_info(tasks)`. There is also a one-shot in-place data migration: `task_events.kind` rename (`ready→promoted`, `priority→reprioritized`, `spawn_auto_blocked→gave_up`), and a one-shot backfill that synthesises `task_runs` rows for legacy `running` tasks. This is precisely the `EnsureSchema`-style boot reconciliation that AGH `internal/store` forbids for column changes.

## Mechanisms / Patterns

- **Five-status state machine on `tasks`** — `triage | todo | ready | running | blocked | done | archived`. Promotion `todo→ready` happens in a sweep (`recompute_ready`) iff every parent reached `done`. Claim `ready→running` is a single CAS UPDATE. Terminal transitions (`done`, `archived`) clear `claim_lock`/`claim_expires`/`worker_pid` and (for completed) write `result`. `triage` is opt-in: tasks created with `triage=True` skip parent-driven status inference and park for a specifier. See `kanban_db.py:90` (status set), `kanban_db.py:1196-1264` (create), `kanban_db.py:1657-1683` (recompute_ready), `kanban_db.py:1690-1776` (claim_task CAS).

- **Two-table split: `tasks` (logical) + `task_runs` (per-attempt)** — load-bearing for retry history, structured handoff, and the "current_run_id NULL ⇔ terminal" invariant. `task_runs` carries claim lock, lease/expiry, PID, started/ended, outcome (`completed | blocked | crashed | timed_out | spawn_failed | gave_up | reclaimed`), summary, metadata (free-form JSON), error. `tasks.current_run_id` is the active pointer, cleared on `_end_run`. See `kanban_db.py:786-806` (DDL), `kanban_db.py:1538-1592` (`_end_run`), `kanban_db.py:1602-1650` (`_synthesize_ended_run`), `kanban_db.py:1706-1725` (defensive cleanup on re-claim).

- **CAS claim primitive** — `UPDATE tasks SET status='running', claim_lock=?, claim_expires=?, started_at=COALESCE(started_at, ?) WHERE id=? AND status='ready' AND claim_lock IS NULL`. Combined with `BEGIN IMMEDIATE`, only one of N concurrent claimers wins; losers get `rowcount=0` and return `None`. No retry loop. Validated by `test_concurrent_claims_only_one_wins` (8-thread race). See `kanban_db.py:1726-1740` and `test_kanban_db.py:199-213`.

- **Lease + heartbeat + stale reclaim** — claims have a 15-minute TTL stored in `claim_expires`. Workers expecting longer runs call `heartbeat_claim` to extend (CAS-guarded by `claim_lock`). A `release_stale_claims` sweep flips any expired `running` row back to `ready`, closes its run as `outcome='reclaimed'`, and emits a `reclaimed` event. See `kanban_db.py:97` (TTL constant), `kanban_db.py:1779-1807` (heartbeat), `kanban_db.py:1810-1841` (release_stale_claims).

- **Crash + timeout enforcement** — `detect_crashed_workers` uses host-local PID liveness (`os.kill(pid, 0)` plus Linux `/proc/<pid>/status` Z-state zombie filtering); `enforce_max_runtime` walks `running` tasks whose `started_at + max_runtime_seconds < now`, sends SIGTERM, polls 5 s, SIGKILLs, then closes the run with `outcome='timed_out'`. Both filter by `claim_lock` host-prefix (PIDs from other hosts are meaningless). See `kanban_db.py:2129-2173` (zombie-aware `_pid_alive`), `kanban_db.py:2215-2302` (max-runtime), `kanban_db.py:2320-2368` (crash detection).

- **Spawn-failure circuit breaker** — `_record_spawn_failure` increments `tasks.spawn_failures`; on the Nth (default 5) consecutive failure the task auto-transitions to `blocked` (outcome `gave_up`) instead of returning to `ready`. Cleared on successful spawn. Prevents thrashing on unfixable tasks. See `kanban_db.py:2105` (constant), `kanban_db.py:2371-2428`.

- **Append-only events table with `run_id` correlation** — `task_events(id INTEGER PK, task_id, run_id, kind, payload TEXT, created_at)`. Events of kinds `claimed | spawned | heartbeat | reclaimed | crashed | timed_out | spawn_failed | gave_up | completed | blocked` carry `run_id`; lifecycle/edit kinds (`created | promoted | unblocked | assigned | edited | reprioritized | linked | unlinked | commented | archived | status`) set `run_id=NULL`. Index `(task_id, created_at)` and `(run_id, id)` support per-task tail and per-run grouping. Referenced as a tail target by the dashboard WS and gateway notifier. See `kanban_db.py:770-777, 830-831` and the event reference in `kanban.md:697-734`.

- **Idempotency on creation** — `tasks.idempotency_key` (indexed). `create_task` does a `SELECT ... ORDER BY created_at DESC LIMIT 1` *before* the write transaction, returning the existing id on hit. Race tolerated: two concurrent creators can both insert; subsequent lookups stabilize. See `kanban_db.py:1180-1188, 826`.

- **Multi-tenancy via free-text `tenant` column** — soft filter for shared-fleet specialists (`biz-a`, `biz-b`); tasks without a tenant are global. Indexed; propagated into `created` event payload and into worker env (`HERMES_TENANT`). The hard isolation boundary is *boards*, not tenants. See `kanban_db.py:823, 825` and `test_tenant_column_filters_listings`.

- **Boards = filesystem isolation, not row-level** — slug-validated directory namespace under `<root>/kanban/boards/<slug>/`, each carrying its own `kanban.db`, `workspaces/`, `logs/`, `board.json` (display metadata). `default` keeps the legacy `<root>/kanban.db` path for back-compat. Resolution chain: `HERMES_KANBAN_DB` env (highest) → `HERMES_KANBAN_BOARD` env → `<root>/kanban/current` text file → `default`. The dispatcher injects all three env vars into spawned workers as defense-in-depth so symlink/Docker layouts can't desynchronise dispatcher↔worker. See `kanban_db.py:55-138, 277-547, 2596-2628`.

- **Dependency graph with cycle detection** — `task_links(parent_id, child_id PRIMARY KEY)`. `link_tasks` walks descendants of `child_id` via DFS to reject parent insertions that would close a cycle. Linking a not-yet-`done` parent demotes a `ready` child back to `todo`. See `kanban_db.py:1349-1399`.

- **Bounded worker context builder** — `build_worker_context` produces the worker prompt with hard caps: 10 most-recent prior attempts, 30 most-recent comments, 4 KB per summary/error/metadata field, 8 KB per body, 2 KB per comment. Older items collapse to a one-line marker. Includes parent results, prior attempt history, and a "recent work by @assignee" cross-task slice for role continuity. See `kanban_db.py:104-109, 2756-2937`.

- **GC retention policy** — `gc_events` deletes `task_events` older than 30 days *only for tasks in terminal state* (`done`/`archived`); `gc_worker_logs` deletes log files by mtime, scoped per-board. Running/ready/blocked tasks keep full event history. See `kanban_db.py:3124-3162`.

- **Notifier subscription pump** — `kanban_notify_subs(task_id, platform, chat_id, thread_id, user_id, last_event_id)` with composite PK. `unseen_events_for_sub` returns events with `id > last_event_id`; `advance_notify_cursor` is a separate call so the gateway can advance only after successful delivery. Pull-based, idempotent. See `kanban_db.py:812-821, 3002-3117`.

- **Schema evolution via `ALTER TABLE ADD COLUMN` cascade + in-place data renames** — `_migrate_add_optional_columns` introspects `PRAGMA table_info(tasks)` and conditionally adds tenant/result/idempotency_key/spawn_failures/worker_pid/last_spawn_error/max_runtime_seconds/last_heartbeat_at/current_run_id/workflow_template_id/current_step_key/skills. Plus a one-shot synthesised-runs backfill for legacy `running` tasks (uses CAS `current_run_id IS NULL` as the guard) and an event-kind rename pass. Runs every `init_db` call; cached per-path so `connect()` only triggers it once per process. See `kanban_db.py:919-1037`.

## Relevant Code Paths
- `.resources/hermes/hermes_cli/kanban_db.py:55-69` — concurrency strategy preamble (WAL + BEGIN IMMEDIATE + CAS).
- `.resources/hermes/hermes_cli/kanban_db.py:90-91` — `VALID_STATUSES`, `VALID_WORKSPACE_KINDS`.
- `.resources/hermes/hermes_cli/kanban_db.py:718-835` — full `SCHEMA_SQL` (tables + indexes).
- `.resources/hermes/hermes_cli/kanban_db.py:842-887` — `connect()` with WAL/sync/foreign-keys pragmas + per-path init cache.
- `.resources/hermes/hermes_cli/kanban_db.py:919-1037` — `_migrate_add_optional_columns` (additive `ALTER TABLE` + run-row backfill + event-kind renames).
- `.resources/hermes/hermes_cli/kanban_db.py:1040-1056` — `write_txn` IMMEDIATE-transaction context manager.
- `.resources/hermes/hermes_cli/kanban_db.py:1098-1270` — `create_task` (idempotency lookup → CAS-safe insert → link rows → `created` event).
- `.resources/hermes/hermes_cli/kanban_db.py:1349-1399` — `link_tasks` + cycle detection.
- `.resources/hermes/hermes_cli/kanban_db.py:1538-1592` — `_end_run` (run closure invariant).
- `.resources/hermes/hermes_cli/kanban_db.py:1602-1650` — `_synthesize_ended_run` (zero-duration run for never-claimed terminals).
- `.resources/hermes/hermes_cli/kanban_db.py:1657-1683` — `recompute_ready` (todo→ready promotion sweep).
- `.resources/hermes/hermes_cli/kanban_db.py:1690-1776` — `claim_task` (CAS + run insertion + pointer install).
- `.resources/hermes/hermes_cli/kanban_db.py:1779-1841` — `heartbeat_claim` and `release_stale_claims`.
- `.resources/hermes/hermes_cli/kanban_db.py:1844-1995` — `complete_task` / `block_task` / `unblock_task` with synthetic-run + invariant-recovery paths.
- `.resources/hermes/hermes_cli/kanban_db.py:2105-2302` — circuit breaker constants, `_pid_alive` (zombie-aware), `enforce_max_runtime`.
- `.resources/hermes/hermes_cli/kanban_db.py:2320-2428` — `detect_crashed_workers`, `_record_spawn_failure`.
- `.resources/hermes/hermes_cli/kanban_db.py:2462-2553` — `dispatch_once` orchestration order.
- `.resources/hermes/hermes_cli/kanban_db.py:2756-2937` — `build_worker_context` bounded context builder.
- `.resources/hermes/hermes_cli/kanban_db.py:3002-3117` — notifier subscription cursor (`unseen_events_for_sub` / `advance_notify_cursor`).
- `.resources/hermes/hermes_cli/kanban_db.py:3124-3162` — `gc_events` / `gc_worker_logs` retention.
- `.resources/hermes/tests/hermes_cli/test_kanban_db.py:152-213` — claim CAS + concurrent-claim race test.
- `.resources/hermes/tests/hermes_cli/test_kanban_db.py:171-196` — stale reclaim, heartbeat extension.
- `.resources/hermes/tests/hermes_cli/test_kanban_db.py:303-323` — events lifecycle + worker context content assertions.
- `.resources/hermes/tests/hermes_cli/test_kanban_boards.py:36-187` — slug validation, multi-board path resolution, env-var precedence.
- `.resources/hermes/website/docs/user-guide/features/kanban.md:643-695` — runs-vs-tasks doc, synthetic-run rationale, current_run_id invariant.

## Transferable Patterns

- **Two-table task/run split** → AGH already has it (`tasks` + `task_runs` in `internal/store/globaldb/global_db.go:352-388`). Hermes' `current_run_id` denormalized pointer with a strict "NULL ⇔ terminal" invariant is the missing piece in AGH today; AGH currently relies on `lease_until`/`heartbeat_at` covering indexes. Adding an explicit `tasks.current_run_id` (with the invariant-recovery write at every transition) would let `internal/task` cheaply answer "is this task running and which run is it?" without querying `task_runs(status='claimed' OR 'running')`.

- **Bounded prior-attempts context builder** → applies to `internal/task` and `internal/situation` because the autonomy kernel will need the equivalent of `build_worker_context` (recent failed runs + parent handoff + comment thread) for any reentry. Hermes' field-byte caps (`_CTX_MAX_FIELD_BYTES = 4 KB`, `_CTX_MAX_BODY_BYTES = 8 KB`, `_CTX_MAX_COMMENT_BYTES = 2 KB`, `_CTX_MAX_PRIOR_ATTEMPTS = 10`, `_CTX_MAX_COMMENTS = 30`) are good defaults to study.

- **Synthetic zero-duration run for never-claimed terminals** → applies to `internal/task` because AGH's `CompleteRun`/`FailRun` paths today require an existing claimed run. If a human/agent forces a task to `done` from CLI without a run, AGH would silently discard the summary. Hermes' `_synthesize_ended_run` (started_at == ended_at == now) preserves attempt history without skewing elapsed metrics.

- **Spawn-failure circuit breaker (per-task counter + auto-block at N)** → applies to `internal/scheduler` and `internal/session` because session spawn failures (CLI not on PATH, workspace unmountable, provider auth gone) currently retry indefinitely. A `tasks.spawn_failures` counter cleared on successful spawn and bounded at N (Hermes uses 5) maps cleanly onto the autonomy kernel's reclaim path.

- **Host-local + zombie-aware PID liveness check** → applies to `internal/procutil` because AGH's process-group supervision parity already exists (Unix vs Windows). Hermes' Linux `/proc/<pid>/status` zombie filter (`State: Z` ⇒ dead) is a refinement worth porting before relying on `os.kill(pid, 0)` for crash detection. Useful when validating session termination during reclaim.

- **Append-only events table as the single broadcast source** → AGH's `task_events` already covers this (`global_db.go:404-424`). Hermes' contribution is the **`run_id` correlation column** with NULL semantics for non-run-scoped kinds + per-attempt grouping in UI. AGH already has `run_id TEXT REFERENCES task_runs(id) ON DELETE SET NULL` — the pattern of grouping events by attempt id in the UI/SSE stream is worth borrowing for `internal/observe` and the `events` API.

- **Pull-based notifier cursor** (`last_event_id` per subscription, separate `advance_cursor` call after delivery) → applies to `internal/bridges` and `internal/network` because the existing event broadcast is push-only. Adding a durable `last_event_id` row per (channel, peer, thread) lets bridges replay gracefully after reconnect — already aligned with AGH's `after_seq` reconnect contract noted in `internal/CLAUDE.md`.

- **GC scoped to terminal-state tasks only** → applies to `internal/store/globaldb` retention. Hermes deletes `task_events` older than 30 days only when the parent task is `done`/`archived`. A simple, agent-friendly retention policy that preserves full history on live work.

## Risks / Mismatches

- **`_migrate_add_optional_columns` is `EnsureSchema`-style boot reconciliation** — would violate AGH's `agh-schema-migration` rule (numbered registry, transactional wrap, no `EnsureSchema` for column changes). AGH already has the right shape: numbered migrations in `globalSchemaMigrations` (versions 1–7 in `global_db.go:515-579`). Do not copy Hermes' additive-ALTER pattern; copy only the *intent* (new columns are nullable + back-fill via a one-shot data migration) into a numbered migration entry.

- **Filesystem-only board isolation** — hermes treats boards as one-DB-per-directory. AGH's `globaldb` is a single shared catalog (`agh.db`); workspace isolation is done with `workspace_id` foreign keys (e.g. `bridge_instances.workspace_id`). Importing per-board separate-DB isolation would conflict with AGH's "single composition root" architecture. The AGH-shaped equivalent is `workspace_id` scoping on `tasks` + index — which AGH already has the building blocks for.

- **One-shot data renames in the migrator** (`task_events.kind` rename: `ready→promoted`, `priority→reprioritized`, `spawn_auto_blocked→gave_up`) — would violate AGH's "Greenfield Alpha — Zero Legacy Tolerance" rule (no migration/compat code for old state; delete obsolete code instead). For AGH: emit the new event kinds directly from day one and delete obsolete kinds in the same change set.

- **Lock format `host:pid` is single-host** — `_claimer_id()` returns `socket.gethostname() + ":" + os.getpid()`; crash detection scans only locks with this host's prefix. AGH's distributed-friendly lease design (`claim_token` + `claim_token_hash` + `lease_until`) is stronger; do not regress to host-prefix filtering. AGH's "non-negotiable claim_token redaction" invariant (`internal/CLAUDE.md`, Security Invariants) explicitly forbids exposing raw `claim_token` like Hermes' `claim_lock` is exposed in event payloads (`kanban_db.py:1772-1775` writes `{"lock": lock, ...}` into `task_events.payload`). AGH already uses `ClaimTokenHash` in `runClaimedPayload` (`internal/task/lease_manager.go:45`) — keep it that way.

- **`COALESCE(started_at, ?)` on first-claim** is a nice trick for "started" semantics but assumes there's only one logical "start". AGH's `task_runs` already has per-run `started_at`; the task-level `started_at` (oldest claim) is non-essential and could mislead UIs.

- **`HERMES_KANBAN_*` env-var injection as defense-in-depth** is a worker-process pattern; AGH's equivalent (`AGH_SESSION_ID`, `AGH_AGENT`) is for the actor-identity layer, not for board pinning. The pattern is portable but the failure mode it defends against (symlinked profile homes desynchronising kanban paths) doesn't exist in AGH's globaldb.

## Open Questions

- Should AGH adopt an explicit `tasks.current_run_id` denormalized pointer with the strict "NULL ⇔ terminal run" invariant, or continue using `task_runs(status, lease_until)` covering-index queries? Adding the pointer simplifies "live drawer refresh" UI logic but introduces a new write-time invariant to maintain across `ClaimNextRun`, `CompleteRun`, `FailRun`, `CancelRun`, and the lease-recovery sweep.
- Hermes' `_synthesize_ended_run` is a UX safety net for never-claimed terminals. Does AGH's task surface (`internal/task`) already preserve summary/metadata when a human force-completes a task that was never claimed? If not, this is a small but valuable patch.
- Hermes scopes `gc_events` to terminal tasks only (running/ready/blocked keep full history). Does AGH's retention policy (if any) match? `internal/store/globaldb/global_db.go` doesn't appear to define an event-retention policy.
- Hermes wraps `_migrate_add_optional_columns` outside its IMMEDIATE-transaction guard (the `_INITIALIZED_PATHS` cache short-circuits re-entry). AGH already enforces `BEGIN IMMEDIATE` on numbered migrations — this answer is "AGH already does it right". Worth confirming the AGH `migrate_workspace.go` path runs under `BEGIN IMMEDIATE`.
- Hermes' `kanban_notify_subs` has `(task_id, platform, chat_id, thread_id)` as composite PK with empty-string thread_id sentinel. Should AGH's network/bridge cursors adopt the same shape, or keep the existing AGH "after_seq per channel" contract?
- The `complete_task` allows transitions from `running | ready | blocked → done`, but `block_task` allows only `running | ready → blocked` (not `todo`). Does AGH's task state machine match this asymmetry? Worth a single-page state-machine diagram before importing.

## Evidence
- `.resources/hermes/hermes_cli/kanban_db.py` (3326 lines) — read in full.
- `.resources/hermes/tests/hermes_cli/test_kanban_db.py` (730 lines) — read in full; covers schema init, status inference, link/cycle, claim CAS, concurrent-claim race, stale reclaim, heartbeat, complete/block/unblock/archive, comments, events, worker context, dispatcher, workspace resolution, tenancy, shared-board path resolution.
- `.resources/hermes/tests/hermes_cli/test_kanban_boards.py:1-200` — slug validation, path resolution, current-board precedence, board CRUD.
- `.resources/hermes/tests/hermes_cli/test_kanban_core_functionality.py` — sampled by grep for `task_runs | claim_lock | task_events`; confirms run-row CAS guards, legacy backfill, run_id propagation.
- `.resources/hermes/website/docs/user-guide/features/kanban.md` (743 lines) — read in full; section "Runs — one row per attempt" (lines 643-695) is the authoritative narrative for the tasks/runs split, current_run_id invariant, and synthetic runs.
- `.resources/hermes/RELEASE_v0.12.0.md:438` — kanban v1 was reverted in #16098 (the design is being reworked).
- `internal/store/globaldb/global_db.go:340-580` — AGH's existing tasks/task_runs/task_events/task_dependencies/task_run_idempotency/task_triage_state schema and migration registry.
- `internal/task/interfaces.go:9-115` — AGH's Manager / Store / RunStore interfaces (ClaimNextRun, lease heartbeat/release/complete/fail, expired-lease recovery).
- `internal/task/lease_manager.go:13-55` — AGH's `ClaimNextRun` with claim-token redaction + post-claim event recording.
- `internal/CLAUDE.md` — Security Invariants (claim_token redaction, host-prefix is *not* the AGH model), Authoritative-primitive exclusivity, agent-manageable surfaces.
