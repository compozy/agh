# Analysis: hermes-cli-tools

Read-only exploration of `.resources/hermes/` (kanban CLI + agent tools) for AGH task `orch-improvs`. Cross-referenced with AGH `internal/cli/`, `internal/api/core/`.

## Scope
- Path explored: `.resources/hermes/hermes_cli/kanban.py`, `.resources/hermes/tools/kanban_tools.py`, `.resources/hermes/tests/hermes_cli/test_kanban_cli.py`, `.resources/hermes/tests/tools/test_kanban_tools.py`, `.resources/hermes/website/docs/user-guide/features/kanban.md`, `.resources/hermes/website/docs/user-guide/features/kanban-tutorial.md`, `.resources/hermes/AGENTS.md`, `.resources/hermes/cli-config.yaml.example`. AGH cross-reference: `internal/cli/task.go`, `internal/cli/root.go`, `internal/api/core/tasks.go`, `internal/api/httpapi/routes.go`, `internal/api/udsapi/routes.go`, `internal/tools/builtin/tasks.go`, `internal/tools/builtin/autonomy.go`, `internal/tools/builtin_ids.go`.
- Topic: kanban CLI + agent tool surface — symmetry, permission, JSON contract.
- Files read in full: `kanban_tools.py` (785 lines), `test_kanban_cli.py` (211 lines), `test_kanban_tools.py` (613 lines). Files read by chunks: `hermes_cli/kanban.py` (1701 lines, read 1-1200 + 1200-1700), `website/docs/.../kanban.md` (742 lines, read 1-742). Sampled (grep + line counts only): `AGENTS.md`, `cli-config.yaml.example`, `kanban-tutorial.md`. AGENTS.md and cli-config.yaml.example contain no `kanban` references.
- Total available files matching `kanban` in hermes resources: 30+ (Python sources, tests, docs, plugins). Three are load-bearing: `hermes_cli/kanban.py`, `tools/kanban_tools.py`, and `kanban_db.py` (delegated DB layer, not read in full but referenced).

## Overview

Hermes splits the kanban surface into **two front doors against one DB layer (`kanban_db`)**: a 30-verb argparse-driven `hermes kanban …` CLI for humans/scripts/cron, and a 7-tool `kanban_*` JSON-schema toolset injected into agent worker model schemas only when the dispatcher set `HERMES_KANBAN_TASK` (or the profile's toolsets explicitly include `kanban`). The slash command `/kanban …` shares the *same* argparse builder via `run_slash()` so CLI, interactive `/kanban`, and gateway-platform `/kanban` (Telegram/Discord/Slack/etc.) all parse identical syntax and emit identical formatting (`hermes_cli/kanban.py:1656-1700`). A dashboard plugin (`plugins/kanban/dashboard/plugin_api.py`) exposes the same operations as `/api/plugins/kanban/*` REST routes.

The tool surface is a **deliberately narrow projection of the CLI**, not full parity. The CLI exposes 30 verbs (init, boards CRUD, create/list/show/assign/link/unlink/claim/comment/complete/block/unblock/archive/tail/dispatch/daemon/watch/stats/log/runs/heartbeat/assignees/notify-{subscribe,list,unsubscribe}/context/gc, plus `/kanban` slash-only); the agent-callable tool set has 7 (`kanban_show`, `kanban_complete`, `kanban_block`, `kanban_heartbeat`, `kanban_comment`, `kanban_create`, `kanban_link`). Asymmetry is intentional and rationalized in the file header (`tools/kanban_tools.py:1-25`): tools cover everything an in-flight worker needs to drive its own task and fan out work; humans (and scripts) reach for the CLI for board management, observation, GC, and notifications. The omitted verbs (`assign`, `archive`, `dispatch`, `gc`, `watch`, `tail`, `claim`, `unblock`, `unlink`, boards-management, notify-subscribe) are operator concerns the worker should not touch.

Three design choices matter for AGH: (1) **structured tool errors** — every handler returns a JSON envelope (`{"ok": true, …}` on success, `{"error": "<msg>"}` via `tool_error()` on failure) so the calling LLM can reason about failure rather than parse stderr (`tools/kanban_tools.py:121-122`, `200-238`); (2) **scope-narrowed worker tokens** — `_enforce_worker_task_ownership` rejects mutating calls (`complete`, `block`, `heartbeat`) on any task id other than the one in the spawning env var, with a regression suite tied to issue #19534 verifying the lockdown (`tools/kanban_tools.py:82-111`, `test_kanban_tools.py:497-612`); (3) **discoverability via env defaults** — `task_id` is implicitly `HERMES_KANBAN_TASK`, so the model's first call (`kanban_show()` with no args) just works and returns a pre-baked `worker_context` string the model can paste into reasoning (`tools/kanban_tools.py:129-198`).

Compared to AGH, hermes's CLI does the **same operations the model does**, but agents reach the board through a **different code path** (`kanban_db` Python module called in-process) rather than through the CLI subprocess. AGH's task surface is also triple-layered (CLI in `internal/cli/task.go`, HTTP routes in `internal/api/httpapi/routes.go:197-229`, UDS routes in `internal/api/udsapi/routes.go:219-257`), with a parallel agent-tool projection under `internal/tools/builtin/tasks.go` and `internal/tools/builtin/autonomy.go`. AGH already has more triple-surface parity than hermes — but hermes's *worker-scoped permission model*, *structured-handoff payload* (summary + metadata + result), *bulk-verb refusal when handoff fields are present*, and *JSON contract on errors* are concrete deltas worth porting.

## Mechanisms / Patterns

- **One DB module, three callers, identical codepaths.** `hermes_cli/kanban.py` (CLI), `tools/kanban_tools.py` (agent tools), `plugins/kanban/dashboard/plugin_api.py` (REST) all import `hermes_cli.kanban_db` and call the same kernel functions (`create_task`, `complete_task`, `block_task`, `heartbeat_worker`, …). The kanban.md docs (`kanban.md:21`, `kanban.md:497`) literally promise "the three surfaces can never drift" because there's only one place to mutate state. AGH already follows this with `BaseHandlers` in `internal/api/core/tasks.go` for HTTP/UDS, and the agent tools in `internal/tools/builtin/tasks.go` route through the same `core.TaskService` interface (`internal/api/udsapi/server.go:65,110,815`).

- **Argparse → slash-command reuse.** `run_slash(rest)` (`hermes_cli/kanban.py:1656-1700`) wraps the entire `build_parser` argparse tree and dispatches via `kanban_command`, letting `/kanban` (in interactive CLI and gateway adapters) inherit every flag, validation, and exit-code semantic for free. Both interactive REPL and 8 messaging platforms speak the *same* argparse grammar.

- **Tool gating via env var + toolset config.** `_check_kanban_mode` (`tools/kanban_tools.py:42-67`) gates the tools on either `HERMES_KANBAN_TASK` being set (worker spawn injects this) or the active profile's `toolsets` including `kanban` (orchestrator profiles). Normal `hermes chat` sees zero kanban tools in its schema — no tool bloat. Test `test_kanban_tools_hidden_without_env_var` (`test_kanban_tools.py:21-38`) and `test_kanban_tools_visible_with_env_var` (`test_kanban_tools.py:41-59`) lock this down.

- **Worker-scoped task token (#19534 hardening).** Once the dispatcher spawns a worker process with `HERMES_KANBAN_TASK=t_X`, that process can only `complete`, `block`, or `heartbeat` *task t_X*. Explicit `task_id` arguments to those tools are checked against the env var; mismatched ids return `{"error": "worker is scoped to task <env>; refusing to mutate <other>. Use kanban_comment to hand off … or kanban_create to spawn follow-up."}` (`tools/kanban_tools.py:82-111`). Read tools (`show`, `comment`, `create`, `link`) are unrestricted — workers can comment on peers and create children. Orchestrator profiles (no env var) are exempt because their job is routing.

- **Structured JSON tool envelope.** Every handler returns `json.dumps(...)` from a single `_ok(**fields)` helper (`tools/kanban_tools.py:121-122`) on success and `tool_error(msg)` from `tools/registry` on failure. Errors are designed to be reasonable-by-LLM strings: not just "invalid", but "provide at least one of: summary (preferred), result" (`tools/kanban_tools.py:213-216`) and "metadata must be an object/dict, got list" (`tools/kanban_tools.py:217-220`). They include next-action hints.

- **Structured handoff: `summary` + `metadata` + `result`.** `kanban_complete` accepts three fields with explicit roles in the schema (`tools/kanban_tools.py:447-493`): `summary` (1-3 sentence human handoff stored on the run), `metadata` (free-form dict — `changed_files`, `tests_run`, `findings`), and `result` (legacy short log line on the task row). At least one is required. Bulk-close on the CLI **refuses** when `--summary` / `--metadata` are present, because those are per-run and copy-pasting them across N tasks is "almost always a footgun" (`hermes_cli/kanban.py:1146-1187`, `kanban.md:685`).

- **Idempotency via per-task `idempotency_key`.** Both CLI (`--idempotency-key` on `create`) and tool (`idempotency_key` field on `kanban_create`) accept a dedup key; if a non-archived task with that key already exists, the kernel returns its id instead of creating a duplicate (`tools/kanban_tools.py:668-674`, `hermes_cli/kanban.py:272-274`). Designed for retry-safe webhooks/cron.

- **Health probe before warning.** `_check_dispatcher_presence` (`hermes_cli/kanban.py:98-149`) inspects gateway PID + `dispatch_in_gateway` config, and `hermes kanban create` warns to stderr if a freshly created `ready` task would sit forever because no dispatcher is running. Probe is defensive — silent on probe-failure to avoid crying wolf.

- **Auto-init on every CLI invocation.** `kanban_command` calls `kb.init_db()` before dispatching subcommands (`hermes_cli/kanban.py:566-570`). Idempotent SQL means a fresh `HERMES_HOME` works on first `hermes kanban list` instead of erroring "no such table".

- **Multi-board isolation.** `--board <slug>` flag pins a request to one board (`hermes_cli/kanban.py:177-187`); workers spawned from that board inherit `HERMES_KANBAN_BOARD` and *cannot see other boards' tasks*. Slug validation rejects path-traversal (`hermes_cli/kanban.py:540-551`). Workspaces, logs, and DB are partitioned per board.

- **Bulk verbs + per-id partial-failure reporting.** `complete`, `block`, `unblock`, `archive` accept multiple ids; failures are listed but don't abort siblings (`hermes_cli/kanban.py:1146-1236`).

- **Subscription-based notifier model.** `notify-subscribe`/`notify-list`/`notify-unsubscribe` lets gateway adapters pin a `(platform, chat_id, thread_id)` to a task's terminal events; the gateway notifier polls `task_events` and emits one message per `completed`/`blocked`/`gave_up`/`crashed`/`timed_out` (`hermes_cli/kanban.py:1506-1549`, `kanban.md:572-587`). Auto-subscribed when `/kanban create` runs from inside a gateway chat.

- **Output mode duality.** Almost every read-verb supports `--json` (`hermes_cli/kanban.py:211, 287, 304, 376, 418, 437, 481, 684, 921, 955, 1265, 1525, 1568`) for machine consumers; default is human-formatted text with status icons and column-aligned tables. `tail` and `watch` stream continuously (poll-based, configurable interval).

- **Discoverability via `context` verb + `worker_context` tool field.** `hermes kanban context <id>` and `kanban_show()`'s response both surface a pre-baked `build_worker_context` string ("title + body + parent results + comments") that the worker model can include verbatim in reasoning (`tools/kanban_tools.py:188-192`).

- **Per-task force-loaded skills.** A task can pin extra skills (`--skill translation` repeatable on CLI, `skills: [...]` on the tool) that the dispatcher loads into the worker on top of the built-in `kanban-worker` skill (`tools/kanban_tools.py:684-696`, `kanban.md:312-348`).

## Relevant Code Paths

- `.resources/hermes/hermes_cli/kanban.py:156-501` — `build_parser` constructs the full 30-verb argparse tree (boards, init, create, list, show, assign, link, unlink, claim, comment, complete, block, unblock, archive, tail, dispatch, daemon, watch, stats, notify-{subscribe,list,unsubscribe}, log, runs, heartbeat, assignees, context, gc).
- `.resources/hermes/hermes_cli/kanban.py:508-611` — `kanban_command` dispatcher: `--board` env-var pinning (532-551), auto-init on every call (566-570), handler map (572-601), and `(ValueError, RuntimeError) → exit 1` envelope.
- `.resources/hermes/hermes_cli/kanban.py:898-938` — `_cmd_create` with workspace-flag parsing, duration parsing (`30s`/`5m`/`2h`/`1d`), and the no-dispatcher-running stderr warning.
- `.resources/hermes/hermes_cli/kanban.py:1146-1187` — `_cmd_complete` enforces "no `--summary` / `--metadata` with multiple ids" (the per-run-handoff-isn't-broadcast invariant).
- `.resources/hermes/hermes_cli/kanban.py:1656-1701` — `run_slash()` reuses the argparse tree for `/kanban` slash commands in CLI and gateway, returns captured stdout+stderr; the same code path 8 messaging platforms hit.
- `.resources/hermes/tools/kanban_tools.py:42-67` — `_check_kanban_mode` schema gating: `HERMES_KANBAN_TASK` env var OR `kanban` toolset enabled. Without either, zero tools in schema.
- `.resources/hermes/tools/kanban_tools.py:74-111` — `_default_task_id` env-var fallback + `_enforce_worker_task_ownership` (#19534 fix): mutating calls on non-self tasks are rejected with explicit hint to use `kanban_comment` or `kanban_create` instead.
- `.resources/hermes/tools/kanban_tools.py:200-238` — `_handle_complete` validates `summary or result`, `metadata is dict`, returns `{"ok": true, "task_id", "run_id"}`.
- `.resources/hermes/tools/kanban_tools.py:323-392` — `_handle_create` accepts `parents` as string OR list, `skills` as string OR list, validates types, returns `{"ok": true, "task_id", "status"}`.
- `.resources/hermes/tools/kanban_tools.py:425-718` — full JSON-schema definitions for all 7 tools with descriptions explicitly written for LLMs (when to use, how to use, anti-patterns).
- `.resources/hermes/tools/kanban_tools.py:724-785` — `registry.register(...)` invocations with toolset (`kanban`), `check_fn` gate, and discoverability emoji.
- `.resources/hermes/tests/tools/test_kanban_tools.py:497-612` — regression suite for #19534 (worker can't mutate sibling tasks) including positive case for orchestrator profiles.
- `.resources/hermes/tests/tools/test_kanban_tools.py:365-421` — `test_worker_lifecycle_through_tools` end-to-end: show → heartbeat → comment → create child → complete with metadata, all through the tools, then DB-state verification.
- `.resources/hermes/tests/hermes_cli/test_kanban_cli.py:181-210` — verifies `/kanban` is in `COMMAND_REGISTRY`, in autocomplete, and bypasses the active-session guard.

AGH cross-reference (current state):
- `internal/cli/task.go:46-82` — root `task` command with 18 direct subcommands (list/create/get/update/delete/publish/start/approve/reject/cancel/next/heartbeat/complete/fail/release/child/dependency/run) — already richer than hermes.
- `internal/cli/task.go:888-905` — `task run` group with sub-verbs `list/enqueue/claim/start/attach-session/complete/fail/cancel`.
- `internal/api/httpapi/routes.go:197-229` — full HTTP route surface mapping every CLI verb to a REST endpoint (`POST /api/tasks`, `POST /api/tasks/:id/publish`, `POST /api/task-runs/:id/complete`, etc.).
- `internal/api/udsapi/routes.go:111-133` — agent-kernel-only routes under `/agent/tasks/*` for `claim-next`, `heartbeat`, `complete`, `fail`, `release` (the autonomy primitive).
- `internal/api/udsapi/routes.go:219-257` — full operator/UDS task surface mirroring HTTP — triple-surface parity for tasks already present.
- `internal/tools/builtin/tasks.go:1-231` — 7 agent-callable tools (`task_list`, `task_read`, `task_create`, `task_child_create`, `task_update`, `task_cancel`, `task_run_list`) with JSON-schema inputs and explicit `RiskRead`/`RiskMutating`/`RiskDestructive` annotations.
- `internal/tools/builtin/autonomy.go:1-132` — 5 agent-callable autonomy tools (`task_run_claim_next`, `task_run_heartbeat`, `task_run_complete`, `task_run_fail`, `task_run_release`) under the `autonomy` toolset.
- `internal/tools/builtin_ids.go:71-94` — `agh__task_*` and `agh__task_run_*` tool IDs catalogued.

## Transferable Patterns

- **Worker-scope token (#19534) for AGH.** Today AGH's autonomy tools (`task_run_complete`, `task_run_fail`, `task_run_release` in `internal/tools/builtin/autonomy.go`) implicitly trust the caller session id from the IPC connection. Adopt hermes's pattern: when an ACP worker is spawned to execute a specific run, inject the run id (or claim-token hash) into env, and have the autonomy tools refuse mutations to *other* runs even if the caller passes an explicit `run_id`. Maps cleanly to AGH's `claim_token_hash` correlation key (security invariant in `internal/CLAUDE.md`).

- **Structured-handoff payload (`summary` + `metadata` + `result`).** AGH's `task_run_complete` currently has its own contract; consider augmenting with hermes's three-field structure: short summary (downstream-context), free-form metadata dict (machine facts: changed files, tests run, findings), legacy result (single-line log). Each lives on the run, not the task. This is the missing piece for AGH's "downstream child reads parent's last completed run summary" pattern.

- **Refuse bulk-mutate when handoff fields present.** `hermes kanban complete a b c --summary X` returns exit 2 with explanation. Same heuristic applies to any AGH bulk verb: if a flag is per-target, refuse to broadcast it. Eliminates a whole class of "I copy-pasted the same metadata to 12 tasks" bugs.

- **`--json` everywhere + `human` everywhere** with no third format. AGH already uses `outputBundle` and supports `text`/`json`/`toon` (3 formats). Hermes only has 2, but **every read verb has `--json`**. AGH's `task list` does (via global `-o`); confirm parity for every read verb.

- **Tool-error envelope with next-action hints.** Hermes errors say *what to do next*: "use kanban_comment to hand off …" or "metadata must be an object/dict, got list". AGH's `internal/tools/builtin/tasks.go` schemas don't carry handler-side hint strings; confirm the runtime error envelope (likely in `internal/api/core/errors.go:268-308 StatusForTaskError`) includes machine-readable reason codes + human hints. The hermes pattern is "every error a model sees should suggest a next call".

- **`worker_context` pre-bake on `task_show` / `task_read`.** `kanban_show()` returns `worker_context: build_worker_context(...)` — a pre-formatted "title + body + parent handoffs + prior attempts + comments" block. AGH's `task_read` returns a structured `TaskDetailRecord` (`internal/cli/task.go:1675-1681`). Consider adding a `context` field that pre-formats parent run summaries + task body + last N comments for direct paste into worker reasoning, especially on retries.

- **Auto-init on first CLI use.** `hermes kanban …` calls `init_db()` every invocation. AGH already does this for `agh.db`; confirm that `task_runs` table reconciliation is also idempotent on cold call and doesn't require a separate `agh task init`.

- **Multi-board isolation as a possible AGH "workspace partition".** Hermes boards are filesystem-isolated SQLite DBs with their own dispatcher loop and per-board env injection (`HERMES_KANBAN_BOARD`). AGH already has workspaces; the relevant pattern here is *workspace-scoped agent toolsets*: a worker spawned for workspace W must not be able to enumerate or mutate workspace V's tasks even if it has the right tool. Map to AGH's `WorkspaceID` correlation key + tool-time scope filter.

- **Tool-schema "show me everything" verb.** `kanban_show` is the worker's first call and returns a 7-field bundle (task, parents, children, comments, events[-50:], runs, worker_context). Consider strengthening AGH's `task_read` to return everything a re-entrant worker needs in one shot (parents' last run summaries, retry attempts, recent events) instead of forcing N tool calls.

- **Subscription-based gateway notification.** `notify-subscribe(task_id, platform, chat_id, thread_id)` decouples task lifecycle from message delivery. AGH bridges (`internal/bridges`) already proxy chat platforms; a per-task subscription table backed by terminal events is a clean pattern to port for "Slack-notify me when task X completes" without polling.

- **Slash-command argparse reuse.** Hermes's `run_slash()` replays the argparse tree on `/kanban` from any chat surface; argument parsing, validation, output formatting are bit-identical with `hermes kanban`. AGH could expose `/agh task …` from any agent chat surface (interactive `agh chat`, web UI command bar, bridge `/agh` slash) reusing the cobra command tree to guarantee identical parsing.

## Risks / Mismatches

- **Hermes asymmetry is intentional.** The 7-tool surface excludes `archive`, `assign`, `dispatch`, `gc`, `watch`, boards-management, notify-subscribe, claim, unlink. The rationale is "workers should not do operator work; orchestrators should not be operators". Naively porting this to AGH and limiting agents from touching `task_assign` / `task_archive` would clash with AGH's "Agent-manageable by default" principle — agents in AGH must be able to do everything a human can. The AGH instinct is *full parity*, not narrow projection. The hermes-style narrowing should be expressed as **per-toolset gating**, not absent tools: mark `task_archive`/`task_dispatch` as the `operator` toolset, expose `kanban_orchestrator` toolset for fan-out work, and only enable both for orchestrator profiles, not workers.

- **Hermes's `_enforce_worker_task_ownership` ties to a single mutable env var.** Env-var-based scope is fragile in long-running processes that reuse a python interpreter or in tests. AGH should use the existing claim_token / claim_token_hash correlation primitive (defined in `internal/CLAUDE.md`) — a worker is *bearer of a specific claim*, not just a process with an env var. This is stronger than what hermes ships.

- **Hermes returns JSON-stringified tool output, not native dicts.** `_ok` returns `json.dumps({...})` (`tools/kanban_tools.py:121-122`); the model receives a JSON string it has to parse mentally. AGH's tool runtime should return native objects so model schema serialization handles this — confirm `internal/tools` does this (likely yes, but worth checking).

- **Hermes's `tail`/`watch` are polling-based** (`time.sleep(args.interval)`). AGH already has SSE (`/api/tasks/:id/stream`, `internal/api/httpapi/routes.go:211`) — the right primitive. Don't port the poll loop.

- **No HTTP/REST tool counterpart documented in hermes.** The kanban dashboard plugin's `/api/plugins/kanban/*` is a **dashboard-only** REST surface gated by an ephemeral session token, **not** a daemon-wide REST contract. Agents can't reach it. Hermes lacks the AGH-style HTTP/UDS dual-transport — AGH is already ahead here.

- **Kanban's `/kanban` slash is auth-less by design** (localhost dashboard). AGH's analogous `/agh task …` from a bridge would cross trust boundaries; need authn parity with the existing UDS auth model (`internal/api/udsapi/server.go:34, 65, 110`).

- **Hermes prompts/skills carry tool guidance.** `KANBAN_GUIDANCE` (in `agent/prompt_builder.py`, sized 1.5-4 KB) is injected into worker system prompts when `HERMES_KANBAN_TASK` is set (`test_kanban_tools.py:451-494`). AGH's autonomy worker spawn should mirror this — a small system-prompt block teaching the model the lifecycle (`claim_next` → work → `heartbeat` → `complete`/`fail`/`release`). Without it, the model often forgets to heartbeat or release.

## Open Questions

- Does AGH inject a `claim_token_hash`-bound capability into a worker's tool schema so `task_run_complete` rejects mutations to runs the worker doesn't own? Forensic: compare hermes's `_enforce_worker_task_ownership` (env-var) against AGH's claim-token model — does the autonomy tool dispatcher in `internal/tools/builtin/autonomy.go` consult the active session's claimed run before allowing mutation? If not, this is a worker-isolation gap.
- Does AGH have an `idempotency_key`-on-create equivalent for `task_create`? Schema in `internal/tools/builtin/tasks.go:137-156` does not list one.
- Does AGH's `task_create` tool support `parents` (multi-parent dependencies) directly, or must the agent call `task_create` then `task_dependency add`? Hermes's `kanban_create(parents=[...])` is one tool call; AGH appears to require two (`task_create` + `task_dependency_add` is missing from the tool list — only `task_child_create` exists, which is single-parent).
- Are AGH's `task_run_complete` / `task_run_fail` schemas carrying `summary` + `metadata` + `result` (or just `result`)? File `internal/tools/builtin/autonomy.go` was only sampled for the descriptor block; the input-schema constants live further in the file. Confirm hermes's three-field handoff is matched.
- Where is `/agh task …` slash-command parity? Hermes ships `/kanban` in interactive REPL + 8 gateway platforms reusing the argparse tree. AGH bridges in `internal/bridges` could host `/agh task …` against the cobra tree — does `internal/cli/task.go` expose its tree in a form suitable for re-dispatch from a non-cobra source?
- Does AGH's dashboard expose triple-surface task management or is it HTTP-API-only? Hermes's dashboard plugin is a thin REST→`kanban_db` shim; AGH's web UI should similarly route through `BaseHandlers` (per `internal/CLAUDE.md` "core is the canonical handler home").
- Hermes's `notify-subscribe` ties terminal task events to gateway chats. AGH bridges + automation could implement this — is there an existing automation trigger kind that subscribes to task lifecycle events, or does this need a new trigger type?

## Evidence

- `.resources/hermes/hermes_cli/kanban.py:1-1701` — full CLI surface; key sections cited above (`build_parser` at L156, `kanban_command` at L508, `run_slash` at L1656).
- `.resources/hermes/tools/kanban_tools.py:1-785` — full tool surface; gating L42-67, ownership-enforcement L82-111, structured handoff L200-238, schemas L425-717, registration L724-785.
- `.resources/hermes/tests/hermes_cli/test_kanban_cli.py:1-211` — slash-command + CLI tests, JSON output verification, COMMAND_REGISTRY parity check.
- `.resources/hermes/tests/tools/test_kanban_tools.py:1-613` — schema-visibility gate L21-59, happy-paths L66-359, end-to-end lifecycle L365-421, prompt-injection contract L425-494, worker-scope regression suite L497-612.
- `.resources/hermes/website/docs/user-guide/features/kanban.md:1-742` — design rationale; "Two surfaces" L13-21, tool/CLI symmetry table L236-244, "Why tools instead of shelling" L283-293, REST surface L437-454, security model L471-479, run/handoff semantics L645-696, event reference L697-734.
- `.resources/hermes/website/docs/user-guide/features/kanban-tutorial.md` — narrative walkthrough (309 lines, sampled by line count only).
- `.resources/hermes/AGENTS.md` (773 lines) — no `kanban` section; sampling confirms kanban guidance lives in skills (`kanban-worker`, `kanban-orchestrator`), not the AGENTS.md.
- `.resources/hermes/cli-config.yaml.example` (1039 lines) — no `kanban` keys; configuration lives in `~/.hermes/config.yaml` `kanban:` block per `kanban.md:188-193` (`dispatch_in_gateway`, `dispatch_interval_seconds`, `dashboard.kanban.*`).
- `internal/cli/task.go:46-1180` — AGH task CLI: 18 root verbs + `child` + `dependency` + `run` subgroups.
- `internal/api/httpapi/routes.go:197-229` — HTTP task routes (full parity with CLI verbs).
- `internal/api/udsapi/routes.go:111-257` — UDS routes split across agent-kernel autonomy (`/agent/tasks/*`) and operator (`/tasks/*`, `/task-runs/*`).
- `internal/tools/builtin/tasks.go:1-231` — agent-callable task tools (7 descriptors) with risk classes and JSON-schema inputs.
- `internal/tools/builtin/autonomy.go:1-132` — agent-callable autonomy tools (5 descriptors: claim_next, heartbeat, complete, fail, release).
- `internal/tools/builtin_ids.go:71-94` — `agh__task_*` and `agh__task_run_*` tool ID catalog confirming the 12-tool surface AGH already exposes.
