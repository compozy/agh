# Analysis: hermes-dispatcher

Read-only exploration of `.resources/hermes/` (kanban dispatcher + runner runtime) for AGH task `orch-improvs`. Cross-referenced with AGH `internal/autonomy/`.

## Scope

- Path explored: `.resources/hermes/`
  - `plugins/kanban/systemd/hermes-kanban-dispatcher.service` (full)
  - `website/docs/user-guide/skills/bundled/devops/devops-kanban-worker.md` (full)
  - `website/docs/user-guide/skills/bundled/devops/devops-kanban-orchestrator.md` (full)
  - `hermes_cli/kanban_db.py` (3,326 lines — read claim/heartbeat/dispatch_once/run_daemon/_default_spawn/_record_spawn_failure/release_stale_claims/detect_crashed_workers/enforce_max_runtime/recompute_ready/build_worker_context)
  - `hermes_cli/kanban.py` (1,701 lines — read `_check_dispatcher_presence`, `_cmd_dispatch`, `_cmd_daemon`, parser, health telemetry)
  - `gateway/run.py` (sampled lines 3,795–3,990: `_kanban_dispatcher_watcher`, `_tick_once_for_board`, `_ready_nonempty`)
  - `agent/prompt_builder.py` (KANBAN_GUIDANCE block, lines 185–241)
  - `cron/scheduler.py` (read in full — orthogonal cron tick, file-lock pattern)
  - `RELEASE_v0.11.0.md`, `RELEASE_v0.12.0.md` (sampled — kanban first landed/reverted in v0.12 #16081→#16098, then re-landed)
- Topic: dispatcher loop, claim/lease, worker pool, lifecycle, agent hand-off, scheduling, multi-board isolation
- Files read in full: 4 (the three required files plus `cron/scheduler.py`)
- Files sampled: 5 (`hermes_cli/kanban_db.py`, `hermes_cli/kanban.py`, `gateway/run.py`, `agent/prompt_builder.py`, RELEASE notes)
- Total available files: ~700 Python files in the repo; the kanban dispatcher surface concentrates in `hermes_cli/kanban*.py` (5,027 lines combined) plus the gateway watcher (~200 LoC) and the systemd unit.
- PDF spec: `.resources/hermes/docs/hermes-kanban-v1-spec.pdf` is a binary PDF (1466 LoC raw bytes). Not readable from this read-only shell — recorded as Open Question.

## Overview

Hermes' kanban subsystem is a SQLite-backed multi-profile collaboration board with a polling dispatcher. The dispatcher's job is narrow and well-bounded: every tick, scan the `tasks` table for `ready` rows, atomically transition them to `running` via a CAS update on `claim_lock`, persist a new row in `task_runs` (one row per attempt), and `subprocess.Popen` a `hermes -p <profile> chat -q "work kanban task <id>"` child with an inherited log fd. The child is the "worker"; the dispatcher never streams or supervises its agent loop, only its OS-level liveness.

The architecture went through a clear evolution: a standalone `hermes kanban daemon` systemd unit (now `DEPRECATED` per the service file's first paragraph) → an embedded gateway watcher (`_kanban_dispatcher_watcher`) gated by `kanban.dispatch_in_gateway` (default true). Running both produces a "double-dispatcher race for the same kanban.db". The standalone path is hidden behind `--force` (argparse.SUPPRESS) so users don't rediscover it. Per-board dispatch enumerates `list_boards()` on every tick so new boards are picked up without a gateway restart.

The claim primitive is the load-bearing concurrency invariant: `claim_task()` is a single SQL `UPDATE tasks SET status='running', claim_lock=?, claim_expires=? WHERE id=? AND status='ready' AND claim_lock IS NULL`. The CAS works because SQLite WAL serializes writers — at most one claimer wins, losers see `cur.rowcount == 0`. Three independent recovery paths protect against worker death:
1. **TTL/stale-claim** (`release_stale_claims`) — 15-minute default lease, renewed by worker calling `kanban_heartbeat`.
2. **PID-liveness** (`detect_crashed_workers`) — host-local zombie-aware `os.kill(pid, 0)` + `/proc/<pid>/status` check.
3. **Per-task max-runtime** (`enforce_max_runtime`) — SIGTERM → 5 s grace → SIGKILL, then re-queue.

A spawn-failure circuit breaker (`_record_spawn_failure`) auto-blocks a task after 5 consecutive spawn failures so the dispatcher doesn't thrash on a missing profile, broken venv, or unmountable workspace. The orchestrator skill is purely a prompt-level convention — it's not a different runtime; it's a profile (`planner`) whose worker reads `KANBAN_GUIDANCE` (auto-injected into every kanban worker's system prompt) and reacts by calling `kanban_create(parents=[...])` to fan out work instead of executing it.

## Mechanisms / Patterns

### Dispatch loop

- **Polling model, no notify**. Pure poll: `dispatch_once` runs every `dispatch_interval_seconds` (default 60s, gateway floor 1.0s; standalone configurable). No SQLite triggers, no `inotify`, no NATS — just ORDER BY priority. Trade-off: up to interval seconds latency between create and pickup. This is acknowledged in `_check_dispatcher_presence` warnings.
- **Per-tick stages, in order** (`kanban_db.py:2491–2553`):
  1. `release_stale_claims(conn)` — TTL reclaim (`status='running' AND claim_expires < now`)
  2. `detect_crashed_workers(conn)` — PID liveness, host-local only (claim_lock prefix matches `host:`)
  3. `enforce_max_runtime(conn)` — kill-and-requeue overdue workers
  4. `recompute_ready(conn)` — `todo → ready` for tasks whose every parent is `done`
  5. `for row in ready_rows: claim_task(...) → resolve_workspace(...) → spawn_fn(...)` ordered by `priority DESC, created_at ASC`.
- **Spawn `subprocess.Popen` (not asyncio)**. SQLite operations wrapped in `asyncio.to_thread` from inside the gateway so the WAL lock never blocks the event loop (`run.py:3941`).
- **Cooperative shutdown**. The gateway loop sleeps in 1 s slices (`run.py:3984–3987`) so `self._running = False` aborts within ~1 s rather than waiting up to `interval` seconds.

### Claim / lease

- **Atomic `ready → running`** (`kanban_db.py:1726–1738`). Single CAS UPDATE; `cur.rowcount != 1` means another claimer won. No retries on the same row in the same tick.
- **Claimer identity**: `_claimer_id()` returns `"<host>:<pid>"`. The host prefix is critical — only the host that owns a PID checks its liveness, allowing a multi-host theoretical deployment without false reclaims.
- **Default TTL**: 15 minutes (`DEFAULT_CLAIM_TTL_SECONDS`). Workers extend via `heartbeat_claim` (`UPDATE … SET claim_expires = ? WHERE … AND claim_lock = ?`).
- **Run row pairing**. Every claim INSERTs a `task_runs` row (status `running`, profile, step_key, claim_lock, claim_expires, max_runtime_seconds). `tasks.current_run_id` points at the active run. Multiple historical runs per task = retry history (`outcome` field: `completed | blocked | crashed | timed_out | spawn_failed | gave_up | reclaimed`).
- **Defensive reclaim on re-claim** (`kanban_db.py:1710–1725`). If a `ready` row somehow has a non-null `current_run_id` (invariant violation), the leaked run is closed `outcome='reclaimed'` before installing the fresh pointer — never strands the prior row.

### Status transitions

```
triage → todo → ready → running → done
                 ↑       ↓     ↘
                 └── reclaim    ↘ blocked → ready (unblock)
                        ↑           ↓
                        └─ TTL / crashed / timed_out / spawn_failed (retry)
                                                                 └ gave_up → blocked (auto, after 5 fails)
                 archived (terminal)
```

`VALID_STATUSES = {"triage","todo","ready","running","blocked","done","archived"}` (`kanban_db.py:90`). `recompute_ready` is the only path from `todo → ready` and runs every tick.

### Failure handling

- **Crash mid-card**: PID-liveness detector reclaims, emits `crashed` event, drops back to `ready`. No state lost — worker context is rebuilt by `build_worker_context` on the next claim from durable run history.
- **Idempotency**: not enforced — the next worker reads the prior runs' `summary`/`error`/`metadata` via `build_worker_context` and the system prompt instructs it to "don't repeat that path" (`devops-kanban-worker.md:120–127`). It's contractual idempotency, not transactional.
- **Quarantine**: `_record_spawn_failure` increments `spawn_failures`; at `failure_limit=5` (configurable), the task is auto-blocked with the last error in `last_spawn_error` and an event `gave_up` is emitted. Successful spawn resets the counter via `_clear_spawn_failures`.
- **Health telemetry**: `bad_ticks` counter — when `ready_pending && !any_spawned` for 6 consecutive ticks, log a single warning every 5 minutes. Catches "every spawn fails silently because PATH is wrong" scenarios that the per-task circuit breaker can't surface.

### Scheduling

- **Priority + FIFO tiebreak**: `ORDER BY priority DESC, created_at ASC` (`kanban_db.py:2500`). No cron-style time windows in kanban.
- **Cron is a separate subsystem** (`cron/scheduler.py`). Different code path, different file (`jobs.json`, `OUTPUT_DIR`). File-locked tick (`fcntl.flock(_LOCK_FILE, LOCK_EX | LOCK_NB)`) to prevent overlapping ticks if a daemon and a systemd timer both fire. `advance_next_run` runs **before** any execution begins to preserve at-most-once semantics on a missed tick.
- **Cron parallelism gate**: `HERMES_CRON_MAX_PARALLEL` env / `cron.max_parallel_jobs` config; jobs with a `workdir` mutate `os.environ["TERMINAL_CWD"]` and are forced sequential.
- **No cron→kanban bridge**. They're orthogonal. A cron job running `hermes kanban create` would be the user-space pattern.

### Worker pool model

- **One process per task**. Not a pool — every claim spawns a fresh `hermes` subprocess. No upper bound enforced by the dispatcher itself; `max_spawn` per tick caps spawns/tick (default unlimited).
- **`subprocess.Popen` with `start_new_session=True`** (`kanban_db.py:2670–2678`). Detaches from the dispatcher's controlling TTY so SIGTERM to the gateway doesn't propagate; init reaps the worker.
- **Log fd inheritance trick** (`kanban_db.py:2685–2690`): parent does NOT close `log_f` after Popen; the kernel keeps the FD open in the child. Parent's reference is GC'd, no leak. Per-task log file rotated when >2 MiB (`_rotate_worker_log`, single-generation).
- **Argv shape**: `["hermes", "-p", <profile>, "--skills", "kanban-worker", "--skills", <extra>, ..., "chat", "-q", "work kanban task <id>"]`. The `kanban-worker` skill is force-loaded for every dispatched worker; per-task skills append. Note the per-name `--skills X` pairs (not comma-joined) — easier to read in `ps` output.
- **Workspace kinds** (`devops-kanban-worker.md:36–42`): `scratch` (tmp dir, GC'd on archive), `dir:<absolute_path>` (shared persistent), `worktree` (git worktree at the resolved path). The dispatcher resolves and persists the path BEFORE spawn (`set_workspace_path`), so the worker just `cd $HERMES_KANBAN_WORKSPACE`.

### Lifecycle / supervision

- **Dispatcher start/stop**:
  - **Embedded (default)**: gateway boot creates `asyncio.create_task(self._kanban_dispatcher_watcher())` (`run.py:3361`). Stops with the gateway. 5 s initial delay so adapters wire first. Failure on tick logged + skipped, never crashes the watcher.
  - **Standalone (`--force`)**: `run_daemon` installs SIGINT/SIGTERM handlers, runs `dispatch_once` until `stop_event.is_set()`. PIDfile written/unlinked. Health-telemetry `_on_tick` callback.
- **Health-checks**: gateway uses python `logger`, standalone uses `print(file=sys.stderr, flush=True)`. No external healthcheck endpoint for dispatcher state.
- **Observability**: every transition writes a row to `task_events` (`claimed`, `spawned`, `heartbeat`, `crashed`, `reclaimed`, `timed_out`, `spawn_failed`, `gave_up`, `promoted`, `completed`, `blocked`). `hermes kanban watch` and `tail` consume this — basically an SSE-equivalent over polling.

### Hand-off into agent execution

- **Selection logic = `task.assignee`**. The task row carries the profile name (`researcher`, `writer`, `backend-eng`, etc.). The dispatcher just runs `hermes -p <assignee>`. There's no scheduler-side capability matching beyond "is assignee non-null" (`row['assignee']` empty → `result.skipped_unassigned.append(...)`).
- **Prompt construction = `KANBAN_GUIDANCE` system prompt block + `build_worker_context()`**. The latter packs: title, body, prior closed runs (capped to 5 most recent, omitted ones rolled into a one-line marker, per-field cap of `_CTX_MAX_FIELD_BYTES`), parent-task handoff summaries+metadata, cross-task role history (5 most recent completed runs by the same assignee on other tasks — implicit identity continuity), and capped comment thread.
- **Orchestrator vs worker = same runtime, different prompt**. The "orchestrator" is just an assignee like `planner` whose skill (`kanban-orchestrator`) is a decomposition playbook. Section "anti-temptation rules" and "decompose, route, and summarize — that's the whole job" enforce it via prompt, not via tool restrictions, although the playbook notes "Your restricted toolset usually doesn't even include terminal/file/code/web". Skill registers the convention; it doesn't enforce it at runtime.

### Multi-board / multi-goal coordination

- Per-board DB at `<HERMES_HOME>/kanban/boards/<slug>/kanban.db`, per-board workspaces and logs roots, per-board metadata. The gateway dispatcher iterates `list_boards(include_archived=False)` on every tick, opening one connection per board, no shared connection across boards.
- "Boards let you separate unrelated streams of work … each board has its own DB, workspaces directory, and dispatcher loop — tasks on one board cannot collide with tasks on another." (`kanban.py:198–203`).
- Worker env pinned to the board the dispatcher claimed from: `HERMES_KANBAN_DB`, `HERMES_KANBAN_WORKSPACES_ROOT`, `HERMES_KANBAN_BOARD` — defense-in-depth so even a profile that rewrites `HERMES_HOME` cannot accidentally see another board (`kanban_db.py:2611–2623`).

## Relevant Code Paths

### Hermes (read in this analysis)

- `.resources/hermes/hermes_cli/kanban_db.py:1690–1776` — `claim_task` (atomic CAS + run-row insert + defensive reclaim).
- `.resources/hermes/hermes_cli/kanban_db.py:1779–1808` — `heartbeat_claim` (claim TTL extension).
- `.resources/hermes/hermes_cli/kanban_db.py:1810–1841` — `release_stale_claims` (TTL reclaim).
- `.resources/hermes/hermes_cli/kanban_db.py:2129–2173` — `_pid_alive` (zombie-aware liveness).
- `.resources/hermes/hermes_cli/kanban_db.py:2215–2302` — `enforce_max_runtime` (SIGTERM/SIGKILL + requeue).
- `.resources/hermes/hermes_cli/kanban_db.py:2320–2368` — `detect_crashed_workers` (host-local PID reclaim).
- `.resources/hermes/hermes_cli/kanban_db.py:2371–2428` — `_record_spawn_failure` (5-strike circuit breaker).
- `.resources/hermes/hermes_cli/kanban_db.py:2462–2553` — `dispatch_once` (the tick).
- `.resources/hermes/hermes_cli/kanban_db.py:2579–2690` — `_default_spawn` (Popen with start_new_session, log inheritance, env pinning).
- `.resources/hermes/hermes_cli/kanban_db.py:2697–2749` — `run_daemon` (legacy standalone loop).
- `.resources/hermes/hermes_cli/kanban_db.py:2756–2937` — `build_worker_context` (capped prompt assembly).
- `.resources/hermes/hermes_cli/kanban_db.py:1657–1683` — `recompute_ready` (`todo → ready` promotion).
- `.resources/hermes/hermes_cli/kanban.py:98–149` — `_check_dispatcher_presence` (warns when `create` would land in a queue with no dispatcher).
- `.resources/hermes/hermes_cli/kanban.py:1257–1296` — `_cmd_dispatch` (one-shot dispatch CLI).
- `.resources/hermes/hermes_cli/kanban.py:1299–1434` — `_cmd_daemon` (deprecated; gated behind `--force`; health-window telemetry).
- `.resources/hermes/gateway/run.py:3795–3987` — `_kanban_dispatcher_watcher` (per-board, asyncio-friendly).
- `.resources/hermes/agent/prompt_builder.py:185–241` — `KANBAN_GUIDANCE` (system-prompt protocol auto-injected into every worker).
- `.resources/hermes/cron/scheduler.py` (in full) — orthogonal cron `tick`, file-lock pattern, `advance_next_run` before execute, parallel/sequential partitioning by `workdir`.

### AGH (cross-reference)

- `internal/task/lease_manager.go:13–55` — `Service.ClaimNextRun` (typed actor, hooks pre-claim, post-claim reconciliation, lease event recording).
- `internal/task/lease_manager.go:57–219` — `HeartbeatRunLease`, `ReleaseRunLease`, `ReleaseSessionRunLeases`, `CompleteRunLease`, `FailRunLease`.
- `internal/task/lease_manager.go:259–320` — `RecoverExpiredRunLeases` (analogue of `release_stale_claims`).
- `internal/task/interfaces.go:28, 99` — `ClaimNextRun` interface (Service + Store).
- `internal/scheduler/scheduler_integration_test.go:89, 127, 189, 260` — uses `ClaimNextRun` from a mechanical scheduler.
- `internal/CLAUDE.md` "Authoritative primitives are exclusive" — `task.Service.ClaimNextRun` is exclusive. The mechanical scheduler does NOT call it; wake/observe/sweep yes, claim/own no.
- `internal/CLAUDE.md` "Hooks are typed dispatch, not an event bus" — dispatch at the call site that owns the state transition.

## Transferable Patterns

1. **CAS-on-status as the single concurrency invariant**. AGH's `ClaimNextRun` already follows this. Hermes adds the `task_runs` row INSERT in the same write transaction (`write_txn(conn):` block bundling UPDATE + INSERT + event append) — single SQLite transaction from `ready → running + run row + claimed event`. AGH's `lease_manager.go:31` does this in one store call too; the layered post-claim reconciliation/event/hook is where AGH adds value beyond Hermes. **Pattern AGH should keep**: never split claim-update from run-row creation across transactions.
2. **Three independent reclaim paths**. TTL, host-local PID liveness, per-task max-runtime — Hermes runs all three at the start of every dispatch tick, before claiming new work. AGH has TTL via `RecoverExpiredRunLeases`; the per-task `max_runtime_seconds` cap with SIGTERM→grace→SIGKILL, and the host-local zombie-aware PID check, are obvious additions for `internal/scheduler` sweeps.
3. **Spawn-failure circuit breaker**. After N consecutive spawn errors on the same task, auto-block it with the last error so the dispatcher doesn't thrash. Maps cleanly to AGH's `task_runs.error` + a `gave_up` lease-failure outcome triggered after a count threshold, owned by `task.Service.FailRunLease` rather than the scheduler.
4. **Health-window telemetry** (`bad_ticks` counter, 6 ticks, rate-limited 5-min warning). Catches "every spawn fails for the same reason" — broken PATH, missing creds. Scheduler-level not transition-level. Implementable in `internal/scheduler` without violating "ClaimNextRun is exclusive" because it only observes spawn results.
5. **Cooperative shutdown via 1 s sleep slices**. Gateway watcher sleeps `min(1.0, interval - slept)` in a loop checking `self._running`. Cleaner than `await asyncio.sleep(interval)` because shutdown latency = 1 s instead of up to `interval`. AGH's scheduler should follow this pattern when adding configurable poll intervals.
6. **Per-task `max_runtime_seconds` with SIGTERM→grace→SIGKILL**. 5 s grace window, then re-queue back to `ready` with `outcome='timed_out'`. Hermes' implementation respects the spawn-failure breaker (`detect_crashed_workers` only requeues when not already given-up). For AGH, `procutil` already centralizes signaling; ties cleanly into existing `internal/subprocess` helpers.
7. **One process per claim, log-fd inheritance**. Detached subprocess with `start_new_session=True` and abandoned-but-inherited log fd is the simplest possible runner model. AGH's session manager already detaches via `context.WithoutCancel`; the equivalent OS-level discipline (Unix process group, Windows forced-exit fallback) is in `internal/procutil`.
8. **Worker context as durable, capped reconstruction**. `build_worker_context` packs prior closed runs + parent handoff summaries + role history + capped comment thread, with per-field char caps and oldest-N collapsed into a one-line marker. The "retry worker reads what didn't work last time" pattern is content-level, but the cap discipline (no single 1 MB summary dominates) is universally useful for AGH's `transcript` / replay paths.
9. **Workspace kind taxonomy as first-class**. `scratch | dir:<path> | worktree` settled into a small enum; the dispatcher resolves to an absolute path before spawn and persists it on the row. Workers `cd $HERMES_KANBAN_WORKSPACE`, no path negotiation. Maps to AGH's `internal/workspace` resolver.
10. **Skill auto-injection on spawn**. The dispatcher always prepends `--skills kanban-worker` to the worker argv plus the system-prompt `KANBAN_GUIDANCE` block. AGH's coordinator-agent bootstrap could follow the same pattern: any "task-runner" capability auto-loads its protocol skill at spawn time.
11. **Multi-board iteration on every tick**. `list_boards()` on every tick = "no restart required when a new board is created". For AGH, equivalent would be "scheduler enumerates active workspaces every tick" — already partially in place via `task_runs` queries.
12. **Embedded-in-existing-daemon vs standalone-binary evolution**. Hermes shipped a systemd unit, then absorbed the dispatcher into the gateway when "two dispatchers racing" became a real footgun. AGH starts in the right place — dispatcher is part of the daemon — but the lesson is: never offer both modes simultaneously without an explicit "you can't run both" gate.

## Risks / Mismatches

- **Polling-only is a deliberate floor.** AGH's autonomy kernel is event-richer (typed hooks, append-only ledger, SSE broadcasters). Adopting Hermes' pure-poll model would regress observability latency. The right transfer is the *reclaim discipline*, not the *polling* itself. Hermes' 60 s-floor latency is unacceptable for AGH's coordinator/peer flows.
- **No event bus, but also no hooks at the dispatcher boundary.** Hermes writes events as side effects of state mutations. AGH's CLAUDE.md says "Hooks are typed dispatch, not an event bus. Dispatch at the call site that owns the state transition." Hermes' approach satisfies the "no event bus" half but lacks the typed-hook layer AGH has at `lease_manager.go:553` (`DispatchTaskRunPreClaim` with deny/narrow). **Transfer pattern, not architecture**.
- **Worker context is rebuilt from durable history every spawn — there's no live agent reattachment.** A retried worker is a fresh process reading SQLite; no streaming reconnect. AGH's session manager intentionally supports detached lifetime + reattachment. Hermes's approach is simpler but loses real-time context. Don't import the "fresh process per retry" model into AGH — AGH already supports better.
- **Single-host assumption baked into PID checks.** `detect_crashed_workers` only inspects PIDs whose `claim_lock` matches `<thishost>:`. Multi-host AGH (if ever) would need a different liveness signal (heartbeat freshness only). Hermes acknowledges: "the whole design is single-host" (`kanban_db.py:2329`).
- **"Boards" map awkwardly to AGH workspaces.** Hermes boards = isolated DB+workspaces. AGH workspaces are tighter than that (single `runtime.db` for events, per-session `events.db`). Don't import per-board DB sharding — AGH's append-only ledger is the canonical authority and shouldn't be partitioned by surface.
- **Orchestrator-vs-worker is purely a prompt convention in Hermes.** The "orchestrator" runtime is identical to a "worker"; it's a profile + a skill + an enforcement-by-prompt. AGH's coordinator-agent has a distinct lifecycle. Don't take Hermes' "anti-temptation rules" as a runtime guard — they're a prompt-engineering pattern, not a permission boundary.
- **Spawn-failure circuit breaker uses a per-task counter without time decay.** Five permanent failures and the task is blocked forever. AGH might want a sliding-window or exponential-backoff variant.
- **Argument-passing inconsistency around `parents=[...]` order**. The orchestrator skill explicitly warns: "argument order for links: `kanban_link(parent_id=..., child_id=...)` — parent first. Mixing them up demotes the wrong task to `todo`." Documentation-as-guardrail. AGH should prefer named structs or strongly-typed parameters at the boundary.
- **"Don't shell out to `hermes kanban <verb>`"** (`devops-kanban-worker.md:141`). The workers' own protocol forbids using the CLI even though the CLI wraps the same database. Reason: terminal backends (Docker, Modal, SSH) don't have the binary. Lesson for AGH: any tool aimed at agents must work uniformly across backends — don't make CLI-vs-tool a distinction the agent has to remember.
- **Heartbeat is purely advisory in Hermes.** A worker that forgets to heartbeat for 15 minutes gets reclaimed. AGH already has an explicit `HeartbeatRunLease` with token verification — strictly stronger.

## Open Questions

- The v1 spec (`.resources/hermes/docs/hermes-kanban-v1-spec.pdf`) is a binary PDF (zip-deflate-encoded, 1466 LoC of binary). The available read-only Bash toolset cannot render it; `Read` requires explicit page ranges and may render the visual content but I did not extract its prose. Worth a separate pass with PDF rendering enabled — the spec likely covers design rationale not visible in the implementation.
- Was there a discrete "v1 → v2" rewrite when kanban was reverted in v0.12 (#16098) and re-landed? `kanban.py:194` mentions "boards (new in v2: multi-project support)". If the spec PDF is v1, the live code is v2 — drift between spec and implementation is implied. (Confirmed at the design level in the orchestrator/worker skills, which version themselves as `2.0.0`.)
- Per-task `step_key` and `current_step_key` columns suggest a multi-step state machine inside a task that's not exposed in the dispatcher I read. Worth investigating if AGH wants step-level checkpointing within a single run — the schema is there but the read paths are not in the files I sampled.
- `tools/skills_tool.py` and `tools/skill_usage.py` (referenced from `cron/scheduler.py:802–818`) drive the skill-loading side that the dispatcher relies on (`--skills kanban-worker` in argv). Not read in this analysis but adjacent to the dispatch surface.
- `kanban_notify_subs` table — referenced from `gateway/run.py:3538–3796` (`_kanban_notifier_watcher`). This is a separate poll loop in the same gateway that delivers terminal events (completed/blocked) to platform subscribers. Worth a dedicated scan if AGH wants to mirror "agent finishes → user gets a Slack/Telegram ping" without polluting the dispatcher.
- Whether the `kanban_*` tools (the ones the worker actually calls — `kanban_show`, `kanban_complete`, `kanban_block`, `kanban_create`) bypass the CLI entirely or wrap the same `kanban_db.py` API. Inferred from the worker doc to be direct DB access, but the source for those tools is in `tools/` and was not opened.
- How `hermes_constants.get_hermes_home()` interacts with `HERMES_HOME` overrides for parallel/isolated runs (worktree-isolation analogue in AGH). Touched lightly in board path resolution but not deeply explored.

## Evidence

- Polling vs notify, interval default 60 s: `.resources/hermes/gateway/run.py:3848` (`interval = float(kanban_cfg.get("dispatch_interval_seconds", 60) or 60)`); standalone `--interval` default `60.0` at `.resources/hermes/hermes_cli/kanban.py:383–384`.
- Atomic claim CAS: `.resources/hermes/hermes_cli/kanban_db.py:1726–1738` (`UPDATE … SET status='running', claim_lock=?, claim_expires=? … WHERE id=? AND status='ready' AND claim_lock IS NULL`); CAS guard at `kanban_db.py:1739–1740` (`if cur.rowcount != 1: return None`).
- Lease TTL 15 min: `.resources/hermes/hermes_cli/kanban_db.py:93–96` (`DEFAULT_CLAIM_TTL_SECONDS`).
- Per-tick stages explicit ordering: `.resources/hermes/hermes_cli/kanban_db.py:2492–2495` (`release_stale_claims → detect_crashed_workers → enforce_max_runtime → recompute_ready`), then `kanban_db.py:2497–2501` (ready-row scan, ORDER BY priority DESC, created_at ASC).
- Spawn-failure circuit breaker default 5: `.resources/hermes/hermes_cli/kanban_db.py:2105` (`DEFAULT_SPAWN_FAILURE_LIMIT = 5`); auto-block path at `kanban_db.py:2389–2407`.
- Process-per-task with detached log fd: `.resources/hermes/hermes_cli/kanban_db.py:2670–2690` (`subprocess.Popen(..., start_new_session=True)`; comment at 2685–2689 explaining intentional non-close).
- Default skill load: `.resources/hermes/hermes_cli/kanban_db.py:2641` (`"--skills", "kanban-worker"`).
- KANBAN_GUIDANCE auto-injection: `.resources/hermes/agent/prompt_builder.py:185–241`.
- Embedded gateway dispatcher start: `.resources/hermes/gateway/run.py:3361` (`asyncio.create_task(self._kanban_dispatcher_watcher())`); per-board iteration `run.py:3891–3906`; cooperative 1 s shutdown `run.py:3984–3987`.
- Standalone dispatcher deprecation: `.resources/hermes/plugins/kanban/systemd/hermes-kanban-dispatcher.service:1–14`; gating `--force` flag at `.resources/hermes/hermes_cli/kanban.py:393–397`.
- Health-window telemetry: standalone at `.resources/hermes/hermes_cli/kanban.py:1363–1389` (HEALTH_WINDOW=6, rate-limit 300 s); gateway at `.resources/hermes/gateway/run.py:3860–3974`.
- Multi-board isolation: `.resources/hermes/hermes_cli/kanban.py:194–203` (boards subcommand description); per-board env pinning at `kanban_db.py:2611–2623`.
- `task_runs` table & outcome enum: `.resources/hermes/hermes_cli/kanban_db.py:779–810` (DDL + comment listing `completed | blocked | crashed | timed_out | spawn_failed | gave_up | reclaimed`).
- Worker-context cap discipline: `.resources/hermes/hermes_cli/kanban_db.py:2756–2937` (`build_worker_context`), with `_CTX_MAX_FIELD_BYTES`, `_CTX_MAX_PRIOR_ATTEMPTS`, `_CTX_MAX_COMMENTS`.
- Cron is orthogonal: `.resources/hermes/cron/scheduler.py` whole file; file-lock at lines 1444–1453; `advance_next_run` before execute at lines 1467–1468.
- Kanban revert/relanding history: `.resources/hermes/RELEASE_v0.12.0.md:438` ("Kanban multi-profile collaboration board — landed in #16081, reverted in #16098 while the design is reworked").
- AGH `ClaimNextRun` exclusivity rule: `internal/CLAUDE.md` — "Authoritative primitives are exclusive. … `task.Service.ClaimNextRun` … no peer package may replicate it. Wake/observe/sweep are allowed; claim/own is not."
- AGH lease state machine: `internal/task/lease_manager.go:14–55` (claim), 57–91 (heartbeat), 93–131 (release), 187–219 (complete), 222–256 (fail), 259–320 (recover-expired).
- AGH hooks-not-bus discipline: `internal/CLAUDE.md` — "Hooks are typed dispatch, not an event bus. Dispatch at the call site that owns the state transition. Never tail event/log tables to fire hooks."
- v1 spec PDF unreadable from this shell: `file` output `PDF document, version 1.5 (zip deflate encoded)`; `wc -l` `1466` lines of binary.
