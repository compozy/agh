# Synthesis: hermes orch-improvs vs AGH

## Inputs
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-data-model.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-cli-tools.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-dispatcher.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-dashboard.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-orchestrator-skills.md`
- Synthesis date: 2026-05-04
- AGH branch: `qa-final`
- AGH HEAD: `cdc9a234` (`docs: archive prds`)

## AGH baseline (current state)

AGH already owns a strict three-layer task surface that is in many ways stronger than hermes' kanban. The state machine is `task.Service.ClaimNextRun` (`internal/task/lease_manager.go:13-55`) plus `HeartbeatRunLease` / `ReleaseRunLease` / `CompleteRunLease` / `FailRunLease` / `RecoverExpiredRunLeases` (`internal/task/lease_manager.go:57-320`). The `Manager` interface enumerates 30+ verbs across task and run primitives (`internal/task/interfaces.go:9-50`). Persistence is a single `agh.db` global catalog with a numbered migration registry (`internal/store/globaldb/global_db.go:340-580`); `task_runs` carries `claim_token`, `claim_token_hash`, `lease_until`, `heartbeat_at`, and `coordination_channel_id` (added in numbered migration `7`); `task_events` references `task_runs.id` with `ON DELETE SET NULL` (`global_db.go:404-424`). Dependencies live in `task_dependencies` with `kind='blocks'` and a transitive-path helper (`DependencyStore.HasDependencyPath`, `interfaces.go:88`).

The mechanical scheduler at `internal/scheduler/scheduler.go:117-540` is intentionally narrow: it sweeps expired leases via `task.Service.RecoverExpiredRunLeases`, picks eligible idle sessions, emits wake notifications, and never replicates `ClaimNextRun` (the package's `doc.go` codifies the rule). Triple-surface parity is wired through `internal/api/core/tasks.go` (1858 lines, `BaseHandlers`), `internal/api/httpapi/routes.go:197-229` (operator HTTP), `internal/api/udsapi/routes.go:124-130` (agent autonomy under `/agent/tasks/*`) and `internal/api/udsapi/routes.go:220-257` (operator UDS); CLI is the 18-verb `agh task …` tree at `internal/cli/task.go:46-2052`. SSE streaming is already in place (`tasks.GET("/:id/stream")`).

The agent-callable surface today is 12 tools: 7 task tools (`agh__task_list`, `agh__task_read`, `agh__task_create`, `agh__task_child_create`, `agh__task_update`, `agh__task_cancel`, `agh__task_run_list`) at `internal/tools/builtin/tasks.go:1-230`, plus 5 autonomy tools (`task_run_claim_next`, `task_run_heartbeat`, `task_run_complete`, `task_run_fail`, `task_run_release`) gated under `ToolsetIDAutonomy` at `internal/tools/builtin/autonomy.go:1-132`. The extension model is declarative TOML/JSON manifests with subprocess + bridge + hooks declarations (`internal/extension/manifest.go:42-235`); there is no equivalent for the hermes-style React-frontend extension contract. The web surface (`web/src/systems/tasks/`, `web/src/routes/_app/tasks*.tsx`) renders task list / detail / runs / new / edit but no kanban-board projection. Skills and AGENT.md are RFC-001/002-driven (`docs/rfcs/001_*`, `docs/rfcs/002_*`) with five-layer precedence; `internal/skills/` already has registry, loader, watcher, MCP sidecar, and `VerifyContent` security scan.

## Comparison matrix

| # | Pattern | Hermes | AGH today | Gap | Recommendation | Effort | AGH layer |
|---|---|---|---|---|---|---|---|
| 1 | Two-table task/run split with `current_run_id` denormalized pointer | `tasks.current_run_id` with strict "NULL ⇔ terminal run" invariant (`kanban_db.py:786-806`, `1538-1592`) | `tasks` + `task_runs` exist, but no `current_run_id` pointer; live UI queries `task_runs(status, lease_until)` (`global_db.go:352-388`) | partial | adapt | M | `internal/store/globaldb` + `internal/task` |
| 2 | Synthetic zero-duration run on never-claimed terminal | `_synthesize_ended_run` preserves `summary`/`metadata` even when human force-completes (`kanban_db.py:1602-1650`) | `CompleteRunLease`/`FailRunLease` require an existing claimed run; force-complete from CLI/HTTP would discard handoff fields | missing | steal | S | `internal/task` |
| 3 | Structured-handoff payload `summary` + `metadata` + `result` | `kanban_complete` schema (`tools/kanban_tools.py:447-493`) and `kanban.md:645-696` lock the three-field shape | `LeaseCompletion` carries `result` + `metadata` but no normalized `summary` field on the run; contract fields exist piecewise (`api/contract/tasks.go:565-617`) | partial | steal | M | contract + `internal/task` + `internal/tools` |
| 4 | Worker-scoped mutation enforcement (#19534) | `_enforce_worker_task_ownership` rejects `complete`/`block`/`heartbeat` on any task other than `HERMES_KANBAN_TASK` (`tools/kanban_tools.py:82-111`) | `LookupActiveRunForSession` already binds session→run via `claim_token_hash` (`lease_manager.go:347-364`); autonomy tools accept a free `run_id` arg in their schema (`autonomy.go:93-132`) and rely on store-level token check | partial | adapt | S | `internal/tools/builtin` |
| 5 | Spawn-failure circuit breaker (N strikes → auto-block) | `_record_spawn_failure` + `gave_up` outcome at 5 consecutive failures (`kanban_db.py:2105`, `2371-2428`) | No counter on `task_runs`; expired-lease recovery just re-queues forever | missing | steal | M | `internal/task` + `internal/scheduler` |
| 6 | Per-task `max_runtime_seconds` with SIGTERM→grace→SIGKILL | `enforce_max_runtime` in dispatch tick (`kanban_db.py:2215-2302`) | Lease TTL handles stale leases; no per-task wall-clock budget that escalates signaling | missing | adapt | M | `internal/task` + `internal/scheduler` + `internal/procutil` |
| 7 | Zombie-aware host-local PID liveness | `/proc/<pid>/status` State `Z` filter on top of `os.kill(pid, 0)` (`kanban_db.py:2129-2173`) | `internal/procutil` centralizes signaling but not zombie-state detection | partial | adapt | S | `internal/procutil` |
| 8 | Health-window telemetry (`bad_ticks` counter) | "Ready pending && nothing spawned for N ticks" → rate-limited warning (`gateway/run.py:3860-3974`) | Scheduler logs cycles but no "all spawns failing for the same reason" detector | missing | steal | S | `internal/scheduler` + `internal/observe` |
| 9 | Refuse bulk-mutate when handoff fields present | CLI rejects `complete a b c --summary X` (`hermes_cli/kanban.py:1146-1187`) | AGH bulk verbs/contracts not yet wired; bulk endpoints planned but not present | missing | steal | S | CLI + `internal/api/contract` |
| 10 | Per-id partial-failure bulk endpoint | `POST /tasks/bulk` returns `{results: [{id, ok, error?}]}` (`plugin_api.py:635-705`) | No bulk endpoints in `httpapi/routes.go`/`udsapi/routes.go` for tasks | missing | steal | M | `internal/api/contract` + `internal/api/core` + CLI + tools |
| 11 | "Show-everything" worker context bundle | `kanban_show` returns `{task, parents, children, comments, events[-50:], runs, worker_context}` (`tools/kanban_tools.py:129-198`) | `task_read` returns `View` with structured detail but no pre-baked context blob with prior-run summaries + capped recent events | partial | steal | M | `internal/api/core` + `internal/tools/builtin` |
| 12 | Bounded prior-attempts context builder | `build_worker_context` with byte caps `_CTX_MAX_FIELD_BYTES = 4 KB`, `_CTX_MAX_BODY_BYTES = 8 KB`, `_CTX_MAX_PRIOR_ATTEMPTS = 10` (`kanban_db.py:2756-2937`) | No equivalent assembler for retry context | missing | steal | M | `internal/task` or `internal/situation` |
| 13 | Append-only event stream + WS cursor seeding | `latest_event_id` returned in initial `GET /board`, then 300 ms WS poll `id > cursor LIMIT 200` (`plugin_api.py:1109-1182`) | SSE already streams via `/api/tasks/:id/stream`; no documented `latest_event_id`/`after_seq` cursor seeding pattern in the task contract | partial | adapt | S | `internal/api/core` + `internal/sse` + `web/` |
| 14 | Refetch-on-burst (no per-event reducers) | UI reloads cheap denormalized board endpoint instead of patching local state per event kind (`kanban.md:481-484`) | `web/src/systems/tasks` not kanban-shaped today; no documented refetch-on-burst rule | missing | steal | S | `web/` + docs |
| 15 | Self-healing fresh-install endpoints | Per-call `init_db()` so first-touch dashboard works on empty home (`plugin_api.py:97-113`) | `globaldb` migrations run at boot; per-request init is forbidden (would clash with numbered migration registry) | already-stronger | reject | n/a | n/a |
| 16 | Plugin REST + frontend manifest contract | `manifest.json` + `dist/index.js` + `plugin_api.py` triplet, host-side React SDK on `window.__HERMES_PLUGIN_SDK__` (`registry.ts:1-149`, `usePlugins.ts:1-123`, `slots.ts:60-199`) | `internal/extension/manifest.go` covers backend extensions only; no frontend manifest, no slot system, no React-on-window SDK in `web/` | missing | adapt | L | `internal/extension` + RFC + `web/` + `@agh/extension-sdk` |
| 17 | Slot registry with `KNOWN_SLOT_NAMES` allowlist | Page-scoped slots (`sessions:top`, `chat:bottom`) let extensions inject without overriding (`slots.ts:60-199`) | Web has no slot/extension surface | missing | adapt | M | `web/` + RFC |
| 18 | Run-row invariant guarded at API layer | PATCH `status=running` rejected with 400 because only `claim_task` may write that status (`plugin_api.py:452-456`) | `core.tasks` and `httpapi/routes.go` use distinct verbs; `running` is a run state, not a task patch field — invariant held by the verb shape | already-have | already-have | n/a | n/a |
| 19 | Notifier subscription cursor (`last_event_id` per platform/chat/thread) | `kanban_notify_subs` table + pull-based `unseen_events_for_sub` / `advance_notify_cursor` (`kanban_db.py:3002-3117`) | AGH bridges already broadcast events; no durable per-(channel, peer, thread) cursor for replay-after-reconnect | partial | steal | M | `internal/bridges` + `internal/store/globaldb` |
| 20 | Idempotency on create (per-task `idempotency_key`) | `tasks.idempotency_key` (`kanban_db.py:1180-1188`) | `task_run_idempotency` exists per-run (`global_db.go:430-442`) and `task_create` contract has it for runs but not for tasks — `task_create` tool schema does not list `idempotency_key` (`tools/builtin/tasks.go`) | partial | steal | S | contract + `internal/api/core` + `internal/tools/builtin` |
| 21 | `--json` everywhere on read verbs + tool-error envelope with next-action hints | Every read verb has `--json`; errors say what to do next (`tools/kanban_tools.py:200-238`) | AGH supports `text`/`json`/`toon`; check error envelope from `core.errors` carries reason codes + hints uniformly | partial | adapt | S | `internal/cli` + `internal/api/core/errors.go` |
| 22 | Auto-injected lifecycle guidance on worker spawn | `KANBAN_GUIDANCE` system-prompt block (`agent/prompt_builder.py:185-241`) | No "claim_next → work → heartbeat → complete/fail/release" baseline injected when a session is spawned to drain a run | missing | steal | M | `internal/coordinator` or `internal/skills/bundled` |
| 23 | Two-tier orchestration content (orchestrator playbook + worker playbook) | `kanban-orchestrator` + `kanban-worker` skills as content-only playbooks (`devops-kanban-{worker,orchestrator}.md`) | `internal/skills/bundled/` exists but no `agh-orchestrator-skill` / `agh-worker-skill` shipped | missing | steal | M | `internal/skills/bundled` |
| 24 | Plan-to-setup compiler (typed plan JSON → idempotent setup script) | `bootstrap_pipeline.py` derives task graph + writes profiles/SOULs/setup.sh (`scripts/bootstrap_pipeline.py:1-501`) | No equivalent agent/CLI tool; would need `agh capability scaffold` style verb | missing | adapt | L | RFC + new CLI verb (out of scope for v1) |
| 25 | Director persona-as-policy (no execution, content-level only) | SOUL.md + `kanban-orchestrator` rule "do not execute the work yourself" (`role-archetypes.md:14-29`) | AGH AGENT.md model exists; no orchestrator persona shipped; AGH posture leans toward stronger gates, not policy-only | partial | adapt | S | RFC 001/002 + `internal/skills/bundled` |
| 26 | Best-effort polling monitor (STUCK / OVERTIME / FLAPPING) | `monitor.py` polls `task list --json` every 30 s, classifies, prints to stderr, never auto-recovers (`scripts/monitor.py:82-131`) | No CLI monitor verb; SSE exists but no canned health classification | missing | steal | S | `internal/cli` + `internal/observe` |
| 27 | Argparse-tree reused for `/agh task …` slash from any chat surface | `run_slash()` runs the same parser for CLI, REPL, and 8 messaging platforms (`hermes_cli/kanban.py:1656-1700`) | Cobra command tree is not exposed for re-dispatch from bridges; `/agh` slash command parity not implemented | missing | adapt | M | `internal/cli` + `internal/bridges` |
| 28 | Plugin REST under documented unauth bypass | `/api/plugins/*` carved out of auth middleware behind loopback + host-header DNS-rebind defense (`web_server.py:228`) | All `/api/*` is authed; UDS auth model is stronger | already-stronger | reject | n/a | n/a |
| 29 | Filesystem-only board isolation (one DB per board) | `<root>/kanban/boards/<slug>/kanban.db` (`kanban_db.py:55-138`) | AGH `globaldb` is single-catalog with `workspace_id` foreign keys; per-board DB sharding would conflict with composition-root | rejected | reject | n/a | n/a |
| 30 | `EnsureSchema`-style boot reconciliation for column changes | `_migrate_add_optional_columns` cascades `ALTER TABLE ADD COLUMN` (`kanban_db.py:919-1037`) | AGH numbered migration registry forbids this for column changes (CLAUDE.md, `agh-schema-migration` skill) | rejected | reject | n/a | n/a |
| 31 | Lock format `host:pid`, host-prefix scan for crash detection | Single-host assumption baked in (`kanban_db.py:2129-2173`) | AGH uses `claim_token_hash` + `lease_until` (security invariant in `internal/CLAUDE.md`) | already-stronger | reject | n/a | n/a |
| 32 | Polling-only dispatch model (60 s default tick) | Pure poll; no notify channel | AGH already has hooks-not-bus + scheduler observe/sweep + SSE | already-stronger | reject | n/a | n/a |
| 33 | Frozen brief contract (`brief.md` immutable after setup) | "If the brief changes, re-fire the kanban — don't edit live" (`brief.md.tmpl:78-79`) | AGH has no orchestration-brief primitive; a contract artifact is interesting but conflicts with agent-managed mid-flight scope refinement | partial | adapt | S | docs / RFC |
| 34 | Workspace-kind taxonomy (`scratch` / `dir:<path>` / `worktree`) | First-class enum on the row, dispatcher resolves abs path before spawn (`kanban_db.py:2611-2623`) | `internal/workspace` exists; no `scratch` / `worktree` per-task kind on `task_runs` | partial | adapt | M | `internal/workspace` + `internal/task` |
| 35 | Stale-process detection + kill at upgrade time | `hermes update` reuses `--stop` SIGTERM/grace/SIGKILL kill path (`test_update_stale_dashboard.py:189-291`) | `agh daemon` has restart logic; `--status`/`--stop` precedence routing is worth confirming as a hardening lesson | partial | adapt | S | `internal/cli` (daemon command) |

## Ranked steal-list

The list is ordered by leverage × low-effort, weighted by AGH posture (`agent-manageable by default`, `hooks-not-bus`, `claim_token` redaction, greenfield-no-compat). Twelve entries — every one is achievable inside existing AGH packages without breaking invariants.

### 1. Worker-scoped mutation enforcement on autonomy tools
**What.** Tighten `task_run_complete` / `task_run_fail` / `task_run_release` so the autonomy tools refuse mutations to any run other than the caller session's currently-claimed run. Replace the loose `run_id` arg with a session-bound resolution path.
**Why now.** The hermes #19534 lesson is exactly the missing rail in `internal/tools/builtin/autonomy.go`. AGH already has `LookupActiveRunForSession` (`internal/task/lease_manager.go:347-364`) and `claim_token_hash` semantics — wiring it into the autonomy dispatcher closes the worker-isolation gap without new schema. Also lets AGH delete the `run_id` parameter from the autonomy schemas (greenfield-no-compat).
**Where it lives.** `internal/tools/builtin/autonomy.go` (input schemas + handlers), `internal/api/core/tasks.go` (autonomy handlers), `internal/api/udsapi/routes.go:124-130`. No DB migration.
**Cost / blast radius.** Tool-input contract change (codegen co-ship), CLI `agh task run` autonomy verbs may need to drop `--run-id` for autonomy actor, docs update on `internal/tools` and the `autonomy` toolset description. No web changes.
**Open questions.** Should the operator UDS path (`/api/task-runs/:id/complete`) accept arbitrary run IDs from a privileged operator while the agent UDS path (`/agent/tasks/:run_id/complete`) be restricted to the caller's claim? Pedro's call on dual policy.

### 2. Three-field structured handoff (`summary` + `metadata` + `result`)
**What.** Promote `summary` from optional to a first-class field on `LeaseCompletion` and `FailRunLease`, alongside `result` (typed JSON) and `metadata` (free-form JSON). Persist on `task_runs` and surface in `task_read`/`task_run_read`. Refuse bulk-complete when `summary`/`metadata` are present (per-target only).
**Why now.** The downstream "child reads parent's last completed run summary" pattern is the single biggest unlock for multi-agent pipelines, and AGH already has the `task_runs.metadata_json` / `result_json` columns (`global_db.go:381-382`). Hermes' `_synthesize_ended_run` (item below) hangs off this.
**Where it lives.** `internal/api/contract/tasks.go:565-617` (add `summary` to lease completion/failure payloads), `internal/store/globaldb` numbered migration to add `task_runs.summary TEXT`, `internal/task/lease_manager.go:188-256`, `internal/tools/builtin/autonomy.go:103-122`, `internal/cli/task.go` complete/fail verbs, `internal/api/core/tasks.go` JSON envelopes, `web/src/systems/tasks` run drawer.
**Cost / blast radius.** Numbered migration (`add_task_run_summary`), contract codegen co-ship (`make codegen`), 12-tool descriptor regeneration, web typegen, docs in `packages/site/docs/runtime/tasks.mdx` (verify path).
**Open questions.** Cap `summary` length (hermes ships 4 KB caps in worker context; AGH should pick a hard limit at ingestion to keep events under SSE message budgets).

### 3. Synthetic terminal run on never-claimed force-complete
**What.** When an operator/agent calls `CompleteTask` / `FailTask` on a task with no active run, synthesize a zero-duration `task_runs` row (started_at == ended_at == now, `claimed_by_kind = origin actor`, `result`/`summary`/`metadata` from the call) before transitioning the task to terminal. Mirror hermes' `_synthesize_ended_run`.
**Why now.** AGH today silently drops handoff fields on a force-complete; this is the cheapest win that prevents data loss and keeps `task_runs` as the single attempt history.
**Where it lives.** `internal/task/manager_*.go` (CompleteTask path), `internal/task/lease_manager.go` complete/fail wrappers, no schema change required (uses existing columns once item #2 lands).
**Cost / blast radius.** Pure runtime addition; emits a new `task_event` that observers must accept (`runtime` event coverage matrix test in `internal/observe`).
**Open questions.** Does the `attempt` counter increment for synthetic runs, or stay at the next free attempt? Pedro's call.

### 4. Spawn-failure circuit breaker (N consecutive failures → block)
**What.** Add a per-task counter that increments on every `FailRunLease` outcome whose error class is `spawn_failed`/`session_unreachable`/`provider_auth`. After N (config default 5), `FailRunLease` transitions the task to a "blocked-by-runtime" state with the last error attached, and the scheduler stops re-queueing. Counter is reset on the first successful claim.
**Why now.** AGH's expired-lease recovery (`RecoverExpiredRunLeases`, `lease_manager.go:259-320`) re-queues forever; a stuck provider/profile thrashes the autonomy kernel until an operator intervenes. The greenfield path is to encode the breaker in the lease state machine, not a side effect of a sweep — fits the "hooks dispatch at the call site" rule.
**Where it lives.** Numbered migration (`task_runs.spawn_failures INTEGER NOT NULL DEFAULT 0` or a separate `task_spawn_health` table), `internal/task/lease_manager.go` (`FailRunLease` increments + transitions), `internal/scheduler` for selection filtering, `internal/observe` for the new `task_run_spawn_gave_up` event.
**Cost / blast radius.** Numbered migration, new event type (codegen co-ship), task contract gains a `spawn_failures` summary field, CLI `agh task list -o json` output change.
**Open questions.** Counter time-decay (sliding window vs hard count) — hermes ships hard count, but AGH could ship a sliding window if cheap. Where does the "blocked-by-runtime" state sit in the existing task status enum (`tasks.status`)?

### 5. `current_run_id` denormalized pointer with strict invariant
**What.** Add `tasks.current_run_id TEXT REFERENCES task_runs(id) ON DELETE SET NULL`. Maintain the invariant: `current_run_id IS NULL` ⇔ pointed-to run is terminal. Set it inside the `ClaimNextRun` transaction; clear it inside `CompleteRunLease` / `FailRunLease` / `RecoverExpiredRunLeases`. Defensive cleanup on re-claim closes leaked runs as `outcome='reclaimed'`.
**Why now.** Today the live drawer / kanban-shaped UI needs an "is this task running, and which run?" query that walks `task_runs(status, lease_until)`. Adding the pointer is a single covering-index lookup; downstream item #11 (show-everything bundle) and the future kanban view depend on it.
**Where it lives.** Numbered migration (`add_tasks_current_run_id` with one-shot backfill from existing `task_runs.status='running'`), `internal/task/lease_manager.go` (every transition writes the pointer), `internal/store/globaldb` reads, `internal/api/contract/tasks.go` exposes the value on `TaskSummaryPayload`.
**Cost / blast radius.** Numbered migration with backfill, contract codegen, 5+ touch points in the lease manager (audit each transition for the invariant).
**Open questions.** Do we expose `current_run_id` on the operator HTTP response or only on the UDS / agent context? Pedro's call.

### 6. Health-window telemetry (`bad_ticks` style detector)
**What.** Inside the scheduler cycle, track "queued runs eligible to claim AND no session was woken in N consecutive ticks". Emit a single rate-limited warning event (e.g., once per 5 minutes) and surface it on `/api/observe/events/stream` and a new `agh observe health task-runs` CLI subcommand.
**Why now.** Catches "every spawn fails for the same reason" (broken PATH, provider auth missing) — the per-task circuit breaker (#4) doesn't surface this because each task fails individually. The detector is observe-only; it doesn't claim or own state, so it doesn't violate the `ClaimNextRun` exclusivity rule.
**Where it lives.** `internal/scheduler/scheduler.go` (cycle metric + threshold), `internal/observe` (event recording + classification), `internal/api/contract` for the new event kind, `internal/cli` for the CLI surface.
**Cost / blast radius.** New event type (codegen co-ship), new CLI verb, no DB migration.
**Open questions.** What's the right threshold for AGH (hermes uses 6 consecutive ticks at 60 s)? Pedro's call given AGH's faster cycle.

### 7. Per-task `max_runtime_seconds` budget with managed-stop escalation
**What.** Add `task_runs.max_runtime_seconds INTEGER` (config default optional). If the lease's `started_at + max_runtime_seconds < now`, the recovery sweep transitions the run via `FailRunLease(reason='timed_out')` after issuing a managed-stop request through `SessionExecutor.RequestTaskStop` → `ForceTaskStop` (already in the interface, `task/interfaces.go:152-158`). Honors AGH's existing supervision posture (SD-001: heartbeats, never wall-clock — but a per-task budget is a separate, contractual deadline distinct from the supervision heartbeat).
**Why now.** Today only the lease TTL bounds runaway sessions, and that's about lease freshness, not actual work duration. A per-task budget is what operators expect for "this should never run more than 4 h".
**Where it lives.** Numbered migration, `internal/task/lease_manager.go` (recovery sweep), `internal/scheduler` (cycle stage), `internal/procutil` (existing SIGTERM/grace/SIGKILL helpers), CLI verb on create/update.
**Cost / blast radius.** Numbered migration, contract codegen, web "max runtime" field on task create/edit forms (`web/src/routes/_app/tasks.new.tsx`, `tasks.$id.edit.tsx`), docs.
**Open questions.** Does `max_runtime` belong on `tasks` (logical, applies to every retry) or on `task_runs` (per attempt)? Hermes places it on `tasks`; AGH's per-attempt model would put it on `task_runs`, but a default-from-task is desirable.

### 8. Per-id partial-failure bulk endpoints
**What.** Ship `POST /api/tasks/bulk` (and UDS twin) returning `{results: [{id, ok, error?}]}`. Same shape for `bulk-cancel`, `bulk-archive`, `bulk-dismiss`. Refuse bulk operations when the request carries per-target-only fields (`summary`, `metadata`, `result`) — surface a typed `ErrBulkPerTargetField` from `internal/api/core/errors.go`.
**Why now.** Today operators must loop the CLI; the agent-tool surface has no bulk verb at all. Hermes' invariant ("never broadcast per-target metadata") is a real footgun blocker. The per-id failure shape preserves the agent's ability to retry partials.
**Where it lives.** `internal/api/contract` (new `BulkOperationResultPayload`), `internal/api/core/tasks.go` (handlers reusing existing single-target methods), `internal/api/httpapi/routes.go` + `internal/api/udsapi/routes.go`, new agent tools `agh__task_bulk_cancel` / `agh__task_bulk_archive`, CLI `agh task bulk` subverb.
**Cost / blast radius.** Contract codegen, OpenAPI spec, 2-3 new agent tools (descriptor regeneration), CLI verb, web bulk-action UI hook.
**Open questions.** Atomicity expectations — hermes does best-effort per-id; AGH should match (no transactional bulk) since transactional bulk over `task_runs` would conflict with `ClaimNextRun` exclusivity.

### 9. "Show-everything" task context bundle on `task_read`
**What.** Add an opt-in `?include=context,parents,children,recent_events,prior_runs` projection (or always-on, denormalized payload) that returns: task body, parent run summaries, last 5 closed runs (with summaries), last 50 events, current run pointer. Pre-format a `worker_context` blob with byte caps (`_CTX_MAX_FIELD_BYTES = 4 KB`, `_CTX_MAX_PRIOR_ATTEMPTS = 5`, `_CTX_MAX_COMMENTS` analogue) suitable for direct injection into a re-entrant worker prompt.
**Why now.** Re-entrant workers today must call N tools (`task_read` + `task_run_list` + `task_event_*`) to reconstruct context; the hermes pattern collapses this to one call. Item #2 (summary field) and item #5 (current_run_id) make the bundle trivial to assemble.
**Where it lives.** `internal/api/core/tasks.go` (new projection), `internal/api/contract/tasks.go` (new `TaskContextBundlePayload`), `internal/tools/builtin/tasks.go` (bump `agh__task_read` schema with `include`), `internal/situation` may host the cap-discipline assembler.
**Cost / blast radius.** Contract codegen, tool descriptor change, web detail-page query change, byte-cap constants in code (no migration).
**Open questions.** Should `worker_context` be agent-callable as a separate tool (`agh__task_context_bundle`) or just an `include` projection on `task_read`? Pedro's call.

### 10. Cursor-seeded SSE / event-tail discipline
**What.** Standardize that every event-bearing GET (e.g., `GET /api/tasks/:id`, `GET /api/tasks`) returns a `latest_event_id` (already AGH's `event_seq` from `task_events.event_seq`, `global_db.go:404-424`). Every consuming SSE / WS handshake accepts `?after_seq=<id>` and resumes from there. Document the "refetch-on-burst" rule: clients render the cheap denormalized payload on burst, never reduce per-event.
**Why now.** AGH already has `event_seq` and `after_seq` reconnect semantics noted in `internal/CLAUDE.md`; what's missing is the canonical contract that every list/read call advertises the cursor seed and every stream consumer uses it. This is the single change that makes the future kanban-shaped web view safe to ship.
**Where it lives.** `internal/api/contract/tasks.go` (add `LatestEventSeq` to list/read payloads), `internal/api/core/tasks.go`, `internal/sse` (handshake parser), `web/src/systems/tasks/hooks` (consume the cursor), docs.
**Cost / blast radius.** Contract codegen, web subscription-hook refactor, no schema change.
**Open questions.** Should the rule apply only to task surfaces or be hoisted to a daemon-wide event-stream contract (`event_seq` on every list payload)? Pedro's call.

### 11. Auto-injected lifecycle guidance on task-spawned sessions
**What.** Ship a bundled, always-loaded skill (`agh-task-worker`, embedded in `internal/skills/bundled/`) whose `instructions.md` codifies the AGH autonomy lifecycle: "claim_next → work (heartbeat every N minutes) → complete with `summary` + `metadata` + `result`, or fail with `error` + `metadata`, or release with `reason`". The coordinator/session bootstrap loads it on every session that gets spawned to drain a queued run.
**Why now.** Hermes' `KANBAN_GUIDANCE` injection is the difference between agents that consistently heartbeat / release and agents that wander off. AGH's RFC 002 already supports `always_load`; this is content-only and uses existing primitives.
**Where it lives.** `internal/skills/bundled/agh-task-worker/`, `internal/coordinator` bootstrap (auto-attach when the session is run-bound), docs `packages/site`.
**Cost / blast radius.** New bundled skill (no schema, no codegen), tests under `internal/skills/bundled` + `agh-test-conventions`.
**Open questions.** Does AGH want a separate `agh-task-orchestrator` skill (decompose-don't-execute playbook) shipped alongside? Pedro's call; could be follow-up.

### 12. Notifier subscription cursor for bridges
**What.** New `bridge_event_subscriptions` table (or extend existing `bridge_routes`) keyed `(bridge_instance_id, peer_id, thread_id)` with `last_task_event_seq`. Bridge ingest pulls events with `event_seq > cursor`, advances cursor only on confirmed delivery (separate `AdvanceCursor` call). Replaces any push-only notification with a durable resume-after-reconnect contract.
**Why now.** Hermes' `kanban_notify_subs` is the cleanest "Slack-notify me when task X completes" pattern, and it composes with item #10's cursor-seeding rule. AGH's bridges already have `bridge_routes.last_activity_at` (`global_db.go:490-505`) — this is one numbered migration away.
**Where it lives.** Numbered migration, `internal/bridges`, `internal/store/globaldb`.
**Cost / blast radius.** Schema migration, contract codegen if exposed as agent-tool, docs.
**Open questions.** Belongs in `internal/bridges` or in a new shared `internal/notifications` primitive? Pedro's call.

## Reject list

- **`EnsureSchema`-style boot reconciliation for column changes** (matrix #30). Direct violation of the `agh-schema-migration` rule (numbered registry, transactional wrap). AGH already does this correctly — never copy hermes' additive `_migrate_add_optional_columns` cascade.
- **One-shot data renames inside the migrator** (`task_events.kind` rename `ready→promoted`, `priority→reprioritized`, `spawn_auto_blocked→gave_up`, hermes `kanban_db.py:919-1037`). Greenfield-no-compat: emit the new event kinds from day one and delete the old code in the same change set.
- **Filesystem-only board isolation / one DB per board** (matrix #29). Conflicts with the single composition root and AGH's `globaldb` authority. Use `workspace_id` scoping on `tasks`/`task_runs` instead — already in place.
- **Lock format `host:pid` with host-prefix crash detection** (matrix #31). Regression vs `claim_token_hash` + `lease_until`. AGH's distributed-friendly lease design is strictly stronger; never store raw `host:pid` in `task_events.payload`.
- **Polling-only dispatch model with a 60 s tick floor** (matrix #32). AGH's hooks-not-bus + scheduler observe/sweep + SSE is event-richer; the right transfer is the *reclaim discipline* (TTL + PID liveness + max-runtime), not the polling itself.
- **Plugin REST under documented unauth bypass** (matrix #28). Hermes carves `/api/plugins/*` out of auth middleware behind loopback. AGH binds web to the daemon's authed surface and UDS to the CLI; any extension HTTP route must stay under the existing token middleware. Keep auth uniform.
- **In-process Python plugin loader (`importlib.spec_from_file_location` + `sys.modules` injection)**. AGH's extension model is out-of-process bridges with capability-scoped Host APIs (`internal/extension/host_api*.go`). Don't regress to in-process loading even if a React-frontend extension contract lands.
- **Hand-rolled markdown renderer** (`dist/index.js:122-185`). Use a vetted library (`markdown-it` + DOMPurify) in `web/`; don't ship escape-then-replace.
- **Self-healing per-call `init_db()` on every endpoint** (matrix #15). Conflicts with AGH's numbered migration registry. AGH boots schemas at daemon start; per-request reconciliation is forbidden.
- **Persona-only execution restriction as the *only* gate** (matrix #25). The pattern is fine *as a layer* (AGENT.md + skill content), but AGH should keep stronger primitives (toolset gating, claim_token-bound capabilities) as the actual enforcement boundary.
- **Plan-compiler that ships executable Python inside a skill** (matrix #24). RFC 002 §2.2 mandates `VerifyContent` scanning; shipping a 500-line Python compiler as a skill resource fails the security posture. If we want a pipeline scaffolder, it should be a bundled CLI verb (`agh capability scaffold`) — not skill-internal scripts.
- **Frozen brief contract as a literal "edit the brief and re-fire" workflow** (matrix #33). Conflicts with agent-managed mid-flight scope refinement. The *invariant* is useful (a contract artifact); the *immutability rule* is not.

## Cross-cutting themes

**Theme A — One DB module, three callers, identical codepaths.** Every analysis cites this: hermes' CLI, agent tools, and dashboard plugin all import `kanban_db` and call the same kernel functions, so "two surfaces never drift". AGH already follows this pattern via `internal/api/core/BaseHandlers` (HTTP + UDS) plus `internal/tools/builtin/tasks.go` routing through the same `core.TaskService` interface — but the *enforcement* (a contract test that asserts every CLI verb has an HTTP twin and a tool twin) is not codified. This is the single highest-leverage discipline rule we should keep visible: a feature is incomplete if it lives in only one of the three surfaces.

**Theme B — Structured handoff is the missing message bus.** The data-model, CLI-tools, dispatcher, and orchestrator-skills analyses all converge on `summary + metadata + result` as the typed message between roles. Hermes' bulk-mutate refusal, build_worker_context's prior-attempts replay, and the orchestrator's `task_graph` metadata are all downstream of that one shape. AGH has the columns but not the discipline; promoting `summary` to a first-class field and refusing per-target broadcasts (items #2 + #9) closes a real gap. This is the single cheapest change that unlocks multi-agent pipelines — and it composes naturally with AGH's existing `task_runs.metadata_json`/`result_json` and `claim_token_hash` discipline.

**Theme C — Reclaim discipline > polling cadence.** The dispatcher analysis is explicit: AGH should not import hermes' poll-every-60 s model, but must import the *three independent reclaim paths* (TTL, host-local PID liveness, per-task max-runtime), the *spawn-failure circuit breaker*, and the *health-window telemetry*. AGH's scheduler today owns observe/sweep/recovery cleanly; the work is to add the per-task budget (item #7), the breaker (item #4), and the health detector (item #6) without violating the `ClaimNextRun` exclusivity rule. All three observe state; none claim it.

**Theme D — Plugin SDK on `window` vs out-of-process bridges.** The dashboard analysis is the only one where AGH's posture clearly diverges. Hermes ships React-extension bundles loaded into the host page via `window.__HERMES_PLUGIN_SDK__`, with all plugins sharing one React identity. AGH's extension model is out-of-process bridges with capability-scoped APIs — strictly safer, but harder to grow into a "kanban-shaped extension tab" experience. If AGH ever wants UI extensibility, this needs an RFC, not a port. The slot registry pattern (matrix #17) is portable in principle; the plugin-Python-loaded-in-process pattern is not.

## Open questions for Pedro

- **Vocabulary call.** Hermes uses `kanban / board / lane`. AGH glossary forbids `recipe`/`workflow`/`procedure`/`playbook` for current AGH artifacts. Do we expose a future kanban-shaped web view as "task board" (UI affordance only) or rename to "task pipeline" / "task graph"? Affects items #11 (web view), #14 (refetch rule), and any future extension UI naming.
- **Scope boundary for v1.** Steal-list 1-7 are the load-bearing core (handoff + invariants + supervision); 8-12 are quality-of-life. Does v1 ship the whole core in one TechSpec, or split into "lease state machine v2" (1-7) and "agent-experience v2" (8-12) as two TechSpecs?
- **Operator vs agent permission split on autonomy mutations.** Item #1: should the operator UDS path accept arbitrary `run_id` while the agent UDS path is claim-bound? Or do we make both claim-bound and add an explicit operator override verb?
- **`current_run_id` placement.** Item #5: expose on the operator HTTP surface or only on the UDS / agent context (`/agent/context`)? Affects web visibility and codegen scope.
- **Per-task `max_runtime` placement.** Item #7: column on `tasks` (logical, default-for-retries) or column on `task_runs` (per attempt) with default-from-task? The hermes design folds this into `tasks`; AGH's per-attempt model leans toward `task_runs`.
- **Spawn-failure breaker semantics.** Item #4: hard count (5 strikes), sliding window (5 in 1 h), or exponential backoff? Hermes ships hard count; AGH could be stricter.
- **Bundled `agh-task-worker` skill scope.** Item #11: also ship `agh-task-orchestrator` (decompose-don't-execute persona) in v1, or follow-up?
- **Notifier-cursor home.** Item #12: `internal/bridges` or new `internal/notifications` primitive? Affects whether agent tools (`agh__notify_subscribe`) emerge in v1.
- **Frontend extension contract.** Theme D: is there appetite for a TechSpec on UI extensibility (slot registry + manifest + SDK on window with stable semver), or do we keep web extension-less for the foreseeable future?
- **Slash-command surface from bridges.** Matrix #27: do we want `/agh task …` parity from any bridge chat surface? Worth a separate spec.

## Evidence index

Hermes side (read in the per-competitor analyses):
- `.resources/hermes/hermes_cli/kanban_db.py` (SCHEMA + state machine + dispatch + claim/lease/reclaim + worker context)
- `.resources/hermes/hermes_cli/kanban.py` (CLI + slash command + `_check_dispatcher_presence`)
- `.resources/hermes/tools/kanban_tools.py` (7-tool agent surface, worker-scope enforcement, structured handoff)
- `.resources/hermes/gateway/run.py` (embedded dispatcher watcher, cooperative shutdown, health window)
- `.resources/hermes/agent/prompt_builder.py` (`KANBAN_GUIDANCE` auto-injected into worker prompts)
- `.resources/hermes/cron/scheduler.py` (orthogonal cron with file-locked tick)
- `.resources/hermes/plugins/kanban/dashboard/manifest.json` (plugin manifest)
- `.resources/hermes/plugins/kanban/dashboard/plugin_api.py` (REST + WebSocket tail loop)
- `.resources/hermes/plugins/kanban/dashboard/dist/index.js` (IIFE plugin entry, drag-drop, markdown renderer)
- `.resources/hermes/hermes_cli/web_server.py` (auth middleware carve-out, plugin discovery, static serve)
- `.resources/hermes/web/src/plugins/{registry.ts,usePlugins.ts,slots.ts}` (frontend plugin SDK + slots)
- `.resources/hermes/tests/hermes_cli/test_kanban_db.py` (CAS race, stale reclaim, heartbeat)
- `.resources/hermes/tests/tools/test_kanban_tools.py` (worker-scope regression, lifecycle, prompt-injection)
- `.resources/hermes/tests/plugins/test_kanban_dashboard_plugin.py` (running-PATCH guard, auto-init, WS auth, bulk partial-failure)
- `.resources/hermes/tests/hermes_cli/test_dashboard_browser_safe_imports.py` (root-barrel import lint)
- `.resources/hermes/tests/hermes_cli/test_dashboard_lifecycle_flags.py` + `test_update_stale_dashboard.py` (`--stop`/`--status` precedence + stale-PID kill)
- `.resources/hermes/website/docs/user-guide/features/kanban.md` + `kanban-tutorial.md` (design rationale, surface tables, run/handoff semantics)
- `.resources/hermes/website/docs/user-guide/skills/bundled/devops/devops-kanban-{worker,orchestrator}.md` (worker contract, orchestrator playbook)
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/` (SKILL.md, intake.md, role-archetypes.md, tool-matrix.md, kanban-setup.md, monitoring.md, examples.md, scripts/bootstrap_pipeline.py, scripts/monitor.py, assets/{soul.md.tmpl,brief.md.tmpl,setup.sh.tmpl})
- `.resources/hermes/plugins/kanban/systemd/hermes-kanban-dispatcher.service` (deprecated standalone unit)

AGH side (cross-referenced):
- `internal/CLAUDE.md` (security invariants, authoritative-primitive exclusivity, hooks-not-bus, agent-manageable-by-default)
- `docs/_memory/standing_directives.md` (SD-001 supervision posture, SD-002 greenfield)
- `docs/_memory/glossary.md` (vocabulary)
- `docs/rfcs/001_agent-md-with-skills-memory.md`
- `docs/rfcs/002_skills-system-final.md`
- `internal/task/lease_manager.go` (lines 13-571: claim/heartbeat/release/complete/fail/recover + autonomy lookup)
- `internal/task/interfaces.go` (Manager, Store composition, RunStore, EventStore)
- `internal/store/globaldb/global_db.go` (lines 340-580: tasks/task_runs/task_events/task_dependencies/task_run_idempotency/task_triage_state schema + numbered migrations 1-10)
- `internal/scheduler/scheduler.go` + `doc.go` (mechanical scheduler, `RunOnce`, `sweepExpiredLeases`)
- `internal/cli/task.go` (18-verb tree + `task run` subgroup)
- `internal/api/contract/tasks.go` (operator + agent payload contract)
- `internal/api/core/tasks.go` (`BaseHandlers`)
- `internal/api/httpapi/routes.go:197-229` (operator HTTP)
- `internal/api/udsapi/routes.go:124-130` (agent autonomy `/agent/tasks/*`) and `:220-257` (operator UDS twin)
- `internal/tools/builtin/tasks.go` (7 task tools)
- `internal/tools/builtin/autonomy.go` (5 autonomy tools)
- `internal/tools/builtin_ids.go` (tool ID catalog)
- `internal/extension/manifest.go` (declarative extension manifest)
- `web/src/routes/_app/tasks*.tsx` (task list/new/detail/edit/runs routes)
- `web/src/systems/tasks/` (adapters, components, hooks, lib, types)
