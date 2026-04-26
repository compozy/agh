# Observability & Self-Correction — Slice Analysis

> Slice: telemetry agents and humans need to *see* what an autonomous run is doing, *detect* when it goes wrong, and *self-correct*. Covers the feedback loop layer: live event surfaces, replay, watchdogs, eval harnesses.

---

## 1. TL;DR

- AGH already records a strong **append-only event log per session** (`internal/store/sessiondb/session_db.go:514-542`), a **cross-session event summary table + token stats + permission log** in `globaldb/` (`internal/observe/observer.go:616-698`), and a **canonical replay assembler** that rebuilds chat history from the raw events (`internal/transcript/transcript.go:113-130`). Foundations are healthy.
- The runtime has a **per-prompt activity supervisor** that emits `runtime_progress`/`runtime_warning` events, marks the session `stalled`, force-cancels the prompt, and stops the session after a hard timeout (`internal/session/prompt_activity.go:170-277`). This is the closest thing AGH has to a watchdog.
- **Hook telemetry is rich**: every hook execution is recorded with `outcome`, `duration_ns`, `dispatch_depth`, optional patch JSON, error string, and a counters/latency histogram in-memory (`internal/hooks/telemetry.go:31-110`, persisted via `sessiondb.RecordHookRun`).
- **Network telemetry is shallow**: only counters per `Kind` and a JSONL audit log (`internal/network/stats.go:74-105`, `internal/network/audit.go:107-129`). No causal trace by `trace_id`/`causation_id`, no SLO dashboard, no per-peer latency.
- **What's missing for autonomy**: (1) no agent-callable read API for "show me my own session, my peers' state, channel activity" — the SSE streams exist but no in-tool helpers; (2) no global watchdog beyond the per-turn activity timeout (no max-iterations, no cost-budget circuit breaker, no recovery prompt injection); (3) no eval/replay harness that can re-run a recorded session deterministically; (4) no workflow-correlation across sessions; (5) no operator-facing alerting (only health endpoint poll).
- The biggest single gap is the **closed-loop "agent self-monitoring" surface**: agents have no first-class way to ask "is my last `direct` message acknowledged? have I exceeded N tool calls? what is my peer's current liveness?" — every signal exists in the daemon DBs but is not callable from inside an agent's tool list.

---

## 2. Current observability surface (exists)

### 2.1 Per-session event store (`sessiondb`)

- **Schema** at `internal/store/sessiondb/session_db.go:23-67`: `events` table is append-only (`INSERT`, no `UPDATE`), keyed by `id`, indexed by `(sequence, type, timestamp, turn_id)`. Sister tables `token_usage` (per-turn upsert) and `hook_runs` (full audit row).
- **Writer model**: dedicated goroutine writer with bounded channel (`writeCh`, cap `defaultWriteBufferSize=256`), drain-on-close (`writerLoop` at `:464-476`, `drainWrites` at `:478-495`). Sequence numbers are monotonic per-session and assigned at write time (`writeEvent` `:514-542`).
- **Query surface**: `Query(ctx, EventQuery)` filtered by `type/agent_name/turn_id/since/after_sequence` (`:324-376`), `History(ctx, EventQuery)` groups events by `turn_id` (`:379-401`).

### 2.2 Cross-session global registry (`globaldb` via `observe.Observer`)

- The Observer implements `session.Notifier` and `session.AgentEventNotifier`:
  - `OnSessionCreated` → `RegisterSession` row (`internal/observe/observer.go:421-461`)
  - `OnSessionStopped` → `UpdateSessionState` with stop reason, failure, liveness, environment (`:464-496`)
  - `OnAgentEventForSession` → updates indexed liveness on every event tick *and* writes summary/token/permission rows (`:505-529`).
- **Three derived tables** in the global DB:
  - `event_summaries` — one row per agent event with `summary` + `timestamp`, queried by `ListEventSummaries` (`:616-630`).
  - `token_stats` — per-session aggregate of input/output/total/cost/turns (`:632-659`).
  - `permission_log` — every permission decision with action/resource/decision/policy_used (`:661-698`).
- **Retention sweep loop**: configurable retention with daily ticker, bounded sweep, status surfaced in health (`internal/observe/retention.go:180-216`).

### 2.3 Health snapshot

`internal/observe/health.go:99-160` returns a single `Health` JSON aggregating:

- `ActiveSessions` / `ActiveAgents` / `Uptime`
- `Persistence` (DB sizes) + `Retention` (sweep status, last cutoff, deletion counts)
- `Failures` — total + `by_kind` + `recent` redacted summaries from any `Failure` row in `globaldb`
- `AgentProbes` — invocation results from configured ACP commands (`:187-199`)
- `Bridges` — instance counts by status, delivery backlog, dropped, auth failures (`bridges.go:159-226`)
- `Tasks` — per-status totals and stuck-run thresholds (`tasks.go:19-27`)
- `Activities` — **the most autonomy-relevant slice** at `health.go:78-96, 295-347`: per-active-session `turn_id`, `last_activity_kind`, `current_tool`, `tool_call_id`, `iteration_current/max`, `idle_seconds`, `elapsed_seconds`, `status` (`active|warning|stalled`), `stall_state`, `stall_reason`.

### 2.4 Hook telemetry

- Every hook dispatch produces a `HookRunRecord{Outcome, Duration, Required, DispatchDepth, PatchApplied, Error}` (`internal/hooks/types.go:259-272`).
- Persisted to `sessiondb.hook_runs`; in-memory counters at `internal/hooks/telemetry.go:31-46` track dispatch counts/latency by `(Event,Source,Mode,Outcome)`, async drop count, queue depth high-water, depth-violation count, registry reload latency, permission-escalation blocks.
- Operator query path: `Observer.QueryHookRuns(ctx, store.HookRunQuery{SessionID,Event,Outcome,Since,Limit})` opens the session DB on demand (`internal/observe/query.go:52-87`).

### 2.5 Network audit + stats

- `FileAuditWriter` writes JSONL to disk **and** mirrors normalized rows to a `globaldb.network_audit` table for `sent|received|rejected|delivered` directions (`internal/network/audit.go:107-213`). Task-ingress decisions also flow through this writer (`:131-169`).
- `runtimeStats` keeps in-memory totals per `Kind` and tagged-event counters (`workflowTaggedEvents`, `handoffTaggedEvents`) — exposed via `NetworkStatus` HTTP route (`stats.go:74-124`).
- **Gaps right here**: no cross-correlation by `trace_id`/`causation_id`, no per-peer latency histogram, no "delivery time" metric, no view of currently-open `interaction_id`s.

### 2.6 Diagnostics redaction

`internal/diagnostics/redact.go:22-45` redacts bearer tokens and quoted/unquoted secret patterns, and bounds the result to a byte budget. Used by failure-health summaries (`observe/health.go:228-229`) before exposing crash bundles.

### 2.7 SSE streams (consumers)

`internal/api/httpapi/routes.go:41,78,91,162,263` plus `internal/api/core/handlers.go:389-427, 614+` and `session_stream.go`:

| Route                              | Stream payload                                                | Polling cadence                                                                |
| ---------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `GET /api/sessions/:id/stream`     | session events with `Last-Event-ID`/sequence resume           | `pollAndStreamSessionEvents` ticker, default `100ms` (`server.go:24`)          |
| `GET /api/observe/events/stream`   | global event summaries (cross-session)                        | same ticker                                                                    |
| `GET /api/tasks/:id/stream`        | task-native live events                                       | same                                                                           |
| `GET /api/bridges/health/stream`   | bridge health snapshots                                       | same                                                                           |
| `GET /api/settings/observability/log-tail` | log file tail (privileged)                            | file tail                                                                      |

**Today every consumer is the web UI.** No first-class "agent SSE client" lives in the agent prompt context.

### 2.8 Logger

`internal/logger/logger.go:44-89` produces a structured `slog` JSON logger with optional file mirror. Consumed everywhere (`o.logger.Warn(...)`). No per-session log file; everything is funneled to the daemon log.

### 2.9 Self-monitoring already wired

- `prompt_activity.go` is the **closest thing AGH has to a watchdog today**:
  - Every prompt opens a `promptActivitySupervisor` running its own goroutine with `ActivityHeartbeatInterval` ticker.
  - `evaluate(now)` (`:170-183`) emits **progress** at `ProgressNotifyInterval`, **warning** when idle > `InactivityWarningAfter`, **timeout** when idle > `InactivityTimeout`.
  - On timeout: marks `Liveness.StallState = "detected"`, persists meta, emits `runtime_warning`, calls `manager.CancelPrompt`, then `manager.StopWithCause(...CauseTimeout, SessionStallReasonActivityTimeout)` (`:239-277`).
- `Session.markRuntimeStalled` and `observeRuntimeActivity` (`internal/session/session.go:382-454`) write the `Liveness.Activity` block that drives the health-API `Activities` array.
- **Iteration counters exist in the schema** (`IterationCurrent`, `IterationMax` in `acp.RuntimeActivity` and `store.SessionActivityMeta`) but **the runtime never increments them** — confirmed by grep: only test/demo writes any non-zero value. That field is dead weight today.

---

## 3. What an agent needs to observe (mostly missing)

For an agent to act autonomously it must answer five questions in-loop. AGH today exposes these only to the web UI, never to agents themselves.

| Question                                                              | Who can answer today           | Gap for autonomy                                                                                                                           |
| --------------------------------------------------------------------- | ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| "What did I just do? Replay my last N turns."                         | UI via `GET /sessions/:id/transcript` (`transcript.Assemble`) | No `tool` form. Agent cannot self-introspect prior tool calls or thoughts.                                                                 |
| "Am I making progress, or am I in a loop?"                            | Daemon side via `Activities` stall detector (`health.go:78-96`) | Agent has no callable signal. `IterationCurrent/Max` exist in struct but are unwritten — there is no max-iterations guard.                  |
| "What is peer X doing right now?"                                     | UI via `/network/peers/:id` and `/network/peers/:id/messages`   | Agent has no `peer.status(peer_id)` tool. The `whois` envelope kind reports identity but **not liveness/activity** (compare with `analysis_inter_agent_comm_patterns.md`). |
| "Is the channel I subscribed to active? Has anyone replied to my last `direct`?" | UI via `/network/channels/:channel/messages`              | No agent-callable inbox poll, no `interaction.status(interaction_id)`. The lifecycle state machine in `network/lifecycle.go` knows but doesn't expose. |
| "What error or timeout fired in the last minute?"                     | Daemon via `event_summaries` + `sessiondb` + hook runs           | No filter for `type IN (failure, timeout, stall, hook_error, permission.denied)` callable from prompt context.                              |

### 3.1 Concrete examples of today's blind spots

- An agent in a loop calling the same tool 50 times has no idea — the Observer counts events but never tells the agent. `prompt_activity.go` only times out on **silence**, not on **repetitive activity**.
- An agent waiting on a peer reply has no built-in poll. It must speculatively prompt itself, hoping new messages have arrived — there is no `await_reply(interaction_id, timeout)` primitive.
- A coordinator agent that spawned 5 worker sessions cannot ask "which worker is stalled?" — there is no `workflow_id` linking the sessions in the first place (acknowledged gap in `docs/ideas/orchestration/multi-agent-patterns-analysis.md:300-320`).

---

## 4. What a human operator needs

### 4.1 Exists

- `GET /api/observe/health` aggregating retention, persistence, failures, agent probes, bridges, tasks, activities (`internal/observe/health.go:99-160`).
- `GET /api/observe/tasks/dashboard` and `/inbox` (`routes.go:96-97`).
- SSE log tail `GET /api/settings/observability/log-tail` (`routes.go:263`).
- Per-session live transcript stream and per-session events stream.
- Bridge health stream + stats.
- Hook catalog, hook runs, hook events introspection (`/api/hooks/catalog|runs|events`).

### 4.2 Missing for autonomy debugging

| Need                                                                      | Why                                                                                  | Closest existing piece                                                                                                                                |
| ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Workflow-level dashboard** spanning all sessions tied to one root prompt | Coordinator + N workers must be visible as one unit                                  | `workflow_id` is proposed in `multi-agent-patterns-analysis.md:308-320` but **not implemented**. Today the only correlation is `interaction_id` on the network. |
| **Stall/loop alerting** (push, not poll)                                  | Operators do not poll `/observe/health` every 5s                                     | Health exposes `Activities[].Status="stalled"` but no webhook/notifier emits when a session crosses the threshold.                                    |
| **Replay tool with diff vs prior run**                                    | "What changed between green run #41 and red run #42?"                                | `transcript.Assemble` rebuilds messages but no diff harness, no fixture replay loop.                                                                  |
| **Eval harness**                                                          | Without one, regressions in agent behavior are invisible                             | None internally. `paperclip/evals/promptfoo` (`/.resources/paperclip/evals/README.md`) shows the pattern but AGH has nothing equivalent.              |
| **Cost/budget breaker per session**                                       | A runaway agent burns money silently                                                 | `token_stats` is aggregated *post-hoc*. There is no `MaxBudgetUSD` check inside the prompt loop. Compare to claude-code `query.ts:1031,1107,1300` which `return { reason: 'max_turns_reached' }` mid-loop. |
| **Per-tool latency p50/p95**                                              | Identify slow tools that block the loop                                              | Hook telemetry has `dispatch_latency` per `(Event,Source,Mode,Outcome)` but it's a sum, not a histogram. No tool-level latency at all.               |
| **Network message lineage view**                                          | "Which `direct` failed and what was its `causation_id` chain?"                       | `network_audit` table stores rows but there is no lineage walker. The `multica` `mention.go` precedent (cited in `analysis_inter_agent_comm_patterns.md:11`) shows what a thread view could look like. |
| **Pre-built Grafana/Prometheus exporters**                                | Today there is no metrics endpoint, no Prom format                                   | Logger is JSON only; no `/metrics` route.                                                                                                            |

---

## 5. Self-correction mechanisms

### 5.1 Exists

| Mechanism                          | Where                                                                                          | Trigger                                                          | Effect                                                                                                                       |
| ---------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Activity heartbeat                 | `internal/session/prompt_activity.go:84-122` (`touch`/`report`/`observeEvent`)                 | every ACP event + every `PromptActivityReport`                   | Resets `LastActivityAt`, persists `Liveness.Activity`, fans out via notifier.                                                |
| Inactivity warning                 | `:205-220, 411-413`                                                                            | idle ≥ `InactivityWarningAfter`                                  | Emits `runtime_warning` event, marks `LastActivityKind="warning"`. **Does not inject a recovery prompt.**                    |
| Inactivity timeout                 | `:222-277`                                                                                     | idle ≥ `InactivityTimeout`                                       | Marks `Liveness.StallState="detected"`, emits `runtime_warning`, calls `CancelPrompt` then `StopWithCause(CauseTimeout)`.    |
| Crash bundle                       | `internal/session/crash_bundle.go` + redaction in `health.go:228`                              | session lifecycle failure                                        | Persisted `CrashBundlePath` + `Failure.Summary` redacted in failure-health view.                                             |
| Hook depth violation               | `internal/hooks/telemetry.go:152-159, 301-313`                                                 | hook chain exceeds depth budget                                  | Counter increments, structured warn log with `event_chain`. Hook is dropped.                                                 |
| Hook async drop                    | `internal/hooks/telemetry.go:120-130`                                                          | async queue overflows                                            | Counter increments + queue-depth high-water tracked.                                                                         |
| Bridge auth-failure / runtime-issue| `internal/observe/bridges.go:91-156`                                                            | adapter reports degraded/error                                   | Effective status overrides persisted status; surfaced in bridge health.                                                      |
| Retention sweep error              | `internal/observe/retention.go:198-216`                                                        | sweep fails                                                      | Persistence degraded, surfaced in `Health.Persistence.Status`.                                                               |

### 5.2 Missing

- **Max-iterations / max-turns guard**: claude-code uses `state.turnCount > maxTurns` to terminate (`/.resources/claude-code/query.ts:1705-1711`); AGH has no per-prompt iteration counter. The `IterationCurrent/Max` fields in `acp.RuntimeActivity` are dead weight today.
- **Repetition / loop detection**: no "same tool name + same input args ≥ N times" detector. Could leverage existing `events` table (`SELECT type='tool_call', content, count(*) FROM events WHERE turn_id=?`).
- **Cost circuit breaker**: claude-code `addToTotalSessionCost → checkBudget → return error_max_budget_usd` (`/.resources/claude-code/cost-tracker.ts`) executes inside the streaming loop. AGH only updates `token_stats` after the turn ends.
- **Recovery prompt injection**: no equivalent to claude-code's max-output-tokens recovery (`query.ts:1107`) which injects "please continue" with attempt counter. AGH cancels the prompt cold, no retry-with-hint.
- **Heartbeat from agent → daemon**: today only the daemon polls the agent's stdout. There is no agent-initiated `i'm-still-thinking` channel — the `PromptActivityReport` exists in `internal/acp/types.go:135` but is driver-side, not agent-side.
- **Permission-denial backpressure**: `permission.denied` is recorded but does not auto-pause the agent, so an agent can hammer the same denied tool indefinitely.
- **Stop hook surface**: claude-code has `Stop`, `TeammateIdle`, `TaskCompleted` stop-hook families (`/.resources/claude-code/query/stopHooks.ts:23-30, 65-100`) that can `preventContinuation` or `blockingErrors`. AGH `internal/hooks/events.go:80-97` has `turn.start`/`turn.end` but **no stop-hook semantics that can veto continuation**.
- **Watchdog isolation**: today the watchdog goroutine lives inside the same `Manager` as the prompt — if the prompt path deadlocks, no separate process can yank it. The crash bundle handles the post-mortem but not the live rescue.
- **Workflow-level halt**: cancelling a coordinator session does not propagate `cancel` to spawned worker sessions because there is no link.

---

## 6. Reference comparisons

### 6.1 Claude Code

- **`while(true)` query loop with explicit `transition.reason`** (`docs/ideas/from-claude-code/analysis_query_engine.md:39-56`). Each terminal state (`max_turns`, `blocking_limit`, `model_error`, `hook_stopped`, `stop_hook_prevented`) is enumerated and observable. AGH's prompt loop has no equivalent named-transition vocabulary — failures bubble up as opaque errors.
- **Withhold-then-recover** pattern (`/.resources/claude-code/query.ts:1031, 1107, 1267-1302`): recoverable errors (413, max-output-tokens) are detected during streaming but withheld from the consumer; a recovery branch attempts to continue. AGH's pipeline cancels on first hard error.
- **Stop hooks** (`/.resources/claude-code/query/stopHooks.ts:23-30, 65-100`): `Stop`, `TeammateIdle`, `TaskCompleted` can veto the loop's natural end with `blockingErrors` that get re-injected as messages. AGH `hooks.HookTurnEnd` is fire-and-forget.
- **Cost tracker as first-class component** (`docs/ideas/from-claude-code/analysis_query_engine.md:194-221`): `getTotalCost() >= maxBudgetUsd → return error_max_budget_usd`. AGH's `token_stats` is post-hoc reporting, not in-loop guard.
- **VCR fixture pattern** (`docs/ideas/from-claude-code/analysis_services_infra.md:18`): hashes API inputs, caches responses to disk, dehydrates env-specific values. This is the seed of an eval harness AGH lacks.
- **Circuit breaker on autocompact** (`docs/ideas/from-claude-code/analysis_query_engine.md:387-396`): 3 consecutive failures → stop trying. AGH has no equivalent for any sub-system.

### 6.2 Hermes

- **Trajectory saver** (`/.resources/hermes/agent/trajectory.py`): every conversation written to `trajectory_samples.jsonl` (completed) or `failed_trajectories.jsonl` with model + completed flag. This is the closest analog to "deterministic replay corpus" — AGH has the data (sessiondb), but no exporter.
- **Insights module** (`/.resources/hermes/agent/insights.py`, 39 KB): produces post-run analytics from trajectories. AGH has none.
- **Two output buckets** (success vs failure trajectories) — useful pattern for an eval harness (positive + adversarial fixtures).

### 6.3 Paperclip

- **Promptfoo eval harness** (`/.resources/paperclip/evals/README.md`): a flat YAML-per-case format under `evals/promptfoo/cases/`, with categories (`core`, `governance`) and deterministic assertions ("agent picks up todo/in_progress correctly", "agent stops on 409"). Phase progression: bootstrap → TS harness with seeded scenarios → pairwise/rubric → efficiency metrics → production-case ingestion. AGH should adopt this structure under `internal/eval/` or a top-level `evals/` directory.

### 6.4 Multica

- **Mention parsing** in body text (`/.resources/multica/server/internal/util/mention.go:13`): a precedent for routing decisions baked into message text. While the linked analysis (`analysis_inter_agent_comm_patterns.md:11`) calls this out for *addressing*, the same pattern matters for **observability**: a mention creates a tracked notification side-effect that observers can monitor.

---

## 7. Concrete proposals

### 7.1 Watchdog hardening (smallest, highest value)

- **Wire iteration counters**: increment `acp.RuntimeActivity.IterationCurrent` on every `tool_call` event in `prompt_activity.go:124-133` (`activityFromEvent`). Persist to `Liveness.Activity` so the existing `health.go:78-96` view shows it. Add `MaxIterations` to `aghconfig.SessionSupervisionConfig` and treat overflow as `CauseTimeout`-equivalent (new `CauseMaxIterations`).
- **Repetition detector**: in the supervisor, keep a small bounded LRU of `(tool_name, sha256(tool_input))`; on >= N matches in M iterations, mark `Liveness.StallState="loop_detected"` and emit a `runtime_warning` with `kind="loop"`. No new schema needed — reuse existing `LastActivityKind`.
- **Recovery prompt before timeout**: between `InactivityWarningAfter` and `InactivityTimeout`, optionally inject a synthetic system-role prompt (already supported via `internal/session/synthetic_prompt.go`) that asks the agent to summarise progress or stop. Today the warning is only an event; nothing nudges the agent.
- **Cost circuit breaker**: add `MaxBudgetUSD`, `MaxTokens` to supervision config; check inside `OnAgentEventForSession` after `UpdateTokenStats`, emit `runtime_warning` and `CancelPrompt` with new `CauseBudgetExceeded` cause. The path is wired — only the check is missing.

### 7.2 Agent-callable telemetry tools

Expose a small read-only tool surface to agents (gated by skill/permission) so an autonomous agent can self-monitor. All endpoints exist server-side; this is just a tool wrapper.

| Tool name                  | Reads from                                               | Returns                                                                  |
| -------------------------- | -------------------------------------------------------- | ------------------------------------------------------------------------ |
| `agh.session.recent_events`| `sessiondb.Query(EventQuery{SessionID=self,Limit=N})`    | last N events, optionally filtered by type                               |
| `agh.session.stats`        | `Health().Activities[me]` + `token_stats[me]`            | iteration count, idle seconds, tool currently running, tokens used so far|
| `agh.peer.status`          | `network.peer.GetPeer(peer_id)` + `Health().Activities`  | peer last-seen, current activity, channels joined                        |
| `agh.channel.recent`       | `network/messages` table                                 | last N messages on a channel with sender/intent                          |
| `agh.interaction.status`   | `network/lifecycle` state machine                        | open/closed/timed-out for a given `interaction_id`                       |
| `agh.failures.recent`      | `Health().Failures.Recent`                               | redacted failure summaries the agent itself can react to                 |
| `agh.hook.runs`            | `Observer.QueryHookRuns`                                 | recent hook results in this session (great for self-debug after denial)   |

These should be **opt-in skills**, not always-on tools, to avoid leaking observability data across workspaces.

### 7.3 New observe/ package surface

- **`observe.WorkflowID`**: add an optional column to `event_summaries` (and `sessions`) — propagate from coordinator session into spawned worker sessions via the existing `internal/api/contract` payloads. Once present, add `Observer.QueryWorkflow(ctx, id)` returning all events across all linked sessions in causal order. Aligned with the recommendation in `docs/ideas/orchestration/multi-agent-patterns-analysis.md:308-320`.
- **`observe.AlertNotifier` interface**: a new minimal notifier the daemon owns:
  ```
  type AlertNotifier interface {
      OnSessionStalled(ctx, sessionID, reason string, idleSeconds int64)
      OnLoopDetected(ctx, sessionID, toolName string, count int)
      OnBudgetExceeded(ctx, sessionID, kind string, current, limit float64)
      OnFailure(ctx, sessionID string, failure store.SessionFailure)
  }
  ```
  Implementations: log-only (default), webhook (config-driven), in-memory ring for HTTP polling. Wired from `prompt_activity.go` and `observer.go:OnSessionStopped`.
- **`observe.LoopMetrics`**: histograms (HDR or simple bucketed) for tool call latency keyed by `tool_name`, hook duration keyed by `(event, source)`, network round-trip keyed by `(channel, kind)`. Today everything is sum/count.
- **Prometheus `/metrics` endpoint**: gated by config, reuses the histograms above. Avoid OpenTelemetry coupling.

### 7.4 SSE / WebSocket extensions

- **Add `GET /api/sessions/:id/activity/stream`**: dedicated SSE that pushes only `Liveness.Activity` deltas. Today the full event stream is the only source — too noisy for a watchdog UI tile.
- **Add `GET /api/observe/alerts/stream`**: pushes the new `AlertNotifier` events. Replaces operator polling.
- **Allow agent-side SSE clients** by giving each agent a per-workspace token (skill-gated) so a long-lived "watcher" sub-agent can subscribe to peer events without re-prompting.

### 7.5 Hook events for telemetry

Add to `internal/hooks/events.go:46-97`:

- `runtime.progress` (family `runtime`) — fired by the supervisor whenever it emits `runtime_progress`. Already happens internally; just expose as a hook event so user-defined hooks can react.
- `runtime.warning` and `runtime.timeout` — same.
- `runtime.loop_detected` — new, fired by the proposed repetition detector.
- `session.budget_exceeded` — fired by the cost breaker.

These let users wire e.g. a Slack webhook hook to `runtime.timeout` without touching daemon code.

### 7.6 Replay & eval harness

- **Replay binary**: `agh replay --session <id>` reads `sessiondb`, deterministically replays events through `transcript.Assemble`, optionally diffing against a fixture file. Foundation already exists in `internal/transcript/`.
- **Eval harness skeleton** under `internal/eval/`:
  - YAML cases (`evals/cases/*.yml`) following paperclip's pattern (`/.resources/paperclip/evals/promptfoo/cases/`).
  - A runner that boots a daemon with isolated `AGH_HOME`, seeds a fixture session, prompts an agent, then runs deterministic assertions on `Observer.QueryEvents` output (e.g. "tool `bash` was called exactly once with arg containing `make verify`").
  - Two trajectories per run: success / failure, mirroring hermes (`/.resources/hermes/agent/trajectory.py`).
- **Promote `docs/ideas/qa-e2e/README.md` Phase 1 backlog** (the matrix at lines 144-156, "Suggested First Automation Backlog" at 860-873) into executable harness — start with `E2E-NET-001`, `E2E-AUTO-001`, `E2E-SES-001`, `E2E-SES-004`, `E2E-WS-001` since they map cleanly to deterministic asserts.

---

## 8. Open questions

1. **Where does eval state live?** Should fixture sessions go in `internal/eval/fixtures/` (compiled in) or under `evals/` at repo root (paperclip-style)? Greenfield, no constraint — pick the layout that lets `make test-integration` consume fixtures cleanly.
2. **Halting policy default**: when `MaxIterations` or `MaxBudgetUSD` trips, do we cancel the whole session (current `CauseTimeout` model), pause for operator approval, or inject a recovery prompt asking the agent to wrap up? Claude Code chooses option 3 for max-output-tokens but option 1 for max-turns; AGH should pick consciously.
3. **Escalation policy**: who receives `AlertNotifier.OnSessionStalled`? Need to distinguish "the human owns this workspace" from "the coordinator agent owns this child session" — feeds back into the workflow-correlation question.
4. **Eval determinism**: AGH agents are non-deterministic by nature (LLM sampling). Should the eval harness pin to recorded ACP responses (VCR pattern, `analysis_services_infra.md:18`), or score with a rubric model (paperclip Phase 2)? VCR is faster/cheaper but tests less; rubric is the inverse.
5. **Prom vs OTel**: external dependency choice. Plain `expvar` on a `/metrics` route is the lightest path, Prom format is the de-facto standard, OTel is heavier but plays with traces. Greenfield freedom — but pick before counters proliferate.
6. **Per-session log file**: today everything goes to one daemon log. Should we mirror per-session `events.log` next to `events.db`? It would simplify offline inspection but doubles I/O.
7. **Loop-detection threshold**: how many repeats before we flag? Hermes' batch runner does not flag at all; claude-code uses 3 attempts for compaction failures. Probably belongs in supervision config as `RepeatedToolCallThreshold` defaulting to e.g. 5.
8. **Agent-callable observability vs prompt bloat**: tool definitions cost tokens. Should the watcher tools be opt-in via skills (per `internal/skills/` registry) or always-on for agent-typed sessions? Skills are the cleaner answer but require a one-time activation.

---

## 9. Files referenced

Most-load-bearing for the slice:

- `/Users/pedronauck/Dev/compozy/agh/internal/observe/observer.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/observe/health.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/observe/query.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/observe/bridges.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/observe/retention.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/observe/tasks.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/observe/reconcile.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/store/sessiondb/session_db.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/transcript/transcript.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/session/prompt_activity.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/session/session.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/session/liveness.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/session/notifier.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/session/interfaces.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/hooks/events.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/hooks/telemetry.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/hooks/types.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/audit.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/stats.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/diagnostics/redact.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/logger/logger.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/api/core/handlers.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/api/core/session_stream.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/acp/types.go`
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/from-claude-code/analysis_query_engine.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/from-claude-code/analysis_services_infra.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/orchestration/multi-agent-patterns-analysis.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/qa-e2e/README.md`
- `/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/query.ts` (max-turns guard, withhold-recover, stop hooks)
- `/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/query/stopHooks.ts` (Stop/TeammateIdle/TaskCompleted hook families)
- `/Users/pedronauck/Dev/compozy/agh/.resources/hermes/agent/trajectory.py` (success/failure trajectory dump)
- `/Users/pedronauck/Dev/compozy/agh/.resources/paperclip/evals/README.md` (eval harness phasing)
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/autonomous/analysis/analysis_inter_agent_comm_patterns.md` (cross-slice context)
