# Hermes vs AGH — Gateway, Delivery & Scheduling

## Executive Summary

- AGH already has a production-grade **scheduler** (`internal/automation/schedule.go`, gocron v2, cron/interval/one-shot, singleton reschedule, clockwork-mockable). More idiomatic than Hermes' hand-rolled tick loop. Gaps: **misfire grace windows, at-most-once via preemptive next-fire advance, pre-run scripts + wake-gates, per-run output archive, inactivity-based run timeouts**.
- AGH has **no multi-channel delivery layer**. Hermes' `DeliveryTarget.parse` + router (`origin | local | platform | platform:id[:thread]`) is the model to steal for automation `deliver` and any future webhook/network-peer response path.
- AGH restart (`internal/daemon/restart.go`, `session/resume_repair.go`) has solid PID/socket handoff but **lacks Hermes' `resume_pending` vs `suspended` two-tier recovery** (`session.py:849-984`). AGH just stamps `stop_reason` — no "in-flight retry-safe vs stuck-loop" classification.
- Streaming progressive-edit consumer (`stream_consumer.py`, 873 LOC) — only useful with a rate-limited chat adapter. **Skip.**
- Hooks: near-parity. AGH's typed hook system is cleaner than Hermes' YAML+importlib. Only worth stealing: wildcard event selector (`command:*`).
- Pairing codes / device pairing: chat-platform concern. **Skip.**

## Capability-by-Capability Gap Analysis

### Gateway session model
- Hermes (`session.py:332-468, 470-526`): two-level key — deterministic `session_key` (platform+chat+thread+user) → mutable `session_id` (transcript) — lets one chat own successive transcripts when auto-reset fires.
- AGH (`internal/session/session.go`): single session id = transcript, no aggregator key.
- **Verdict**: Skip until AGH grows external ingress (chat/webhook routing).

### Multi-platform delivery
- Hermes (`gateway/delivery.py:28-104`): `DeliveryTarget.parse("origin" | "local" | "<platform>[:chat_id[:thread_id]]")`, resolved against home channels + channel directory. Router fans out to adapters + local archive, truncating oversize with "full output saved to…" footer (`delivery.py:239-247`).
- AGH: No delivery surface. Automation output stays in the session event store.
- **Verdict**: **Adopt the target parser + router pattern** for `Job.Deliver` and future webhook response path. Equivalent grammar: `origin | local | webhook:<id> | session:<id>`.

### Stream consumption & backpressure
- Hermes `stream_consumer.py`: sync→async bridge, progressive edits with flood-control adaptive backoff (strike counter, interval doubling to 10s), think-block filtering, tool-boundary segment breaks.
- AGH `api/core/sse.go`: one-way server→client, HTTP flush only.
- **Verdict**: **Skip** — AGH's consumers are browsers and CLIs, not rate-limited chat APIs.

### Pairing / device pairing
- Hermes `pairing.py`: 8-char codes, 1h TTL, rate limit, lockout, `chmod 0600` atomic writes.
- AGH: token/UID auth via `httpapi/middleware.go`.
- **Verdict**: **Skip** for local-first. Revisit only for remote ingress.

### Restart / session rehydration
- Hermes (`session.py:849-984`): `suspend_recently_active(max_age_seconds=120)` on boot — sessions updated in the last 2 min are marked suspended (force-reset on next access) because they were likely in-flight during crash. Separate `mark_resume_pending` path preserves `session_id` and auto-continues on next turn. Stuck-loop counter escalates persistent `resume_pending` → `suspended`.
- AGH (`daemon/restart.go:49-62`, `session/resume_repair.go`): durable `RestartOperation` handoff; resume validation stamps `stop_reason = crashed_during_start`. No "in-flight retry-safe vs stuck-loop" split.
- **Verdict**: **Adopt the two-tier pattern**. On boot, classify sessions by `updated_at` window: within `restart_drain_timeout` → `stop_reason = restart_drained` (retry-safe); older → `crashed_during_start` (forced wipe). Add a `consecutive_resume_failures` counter to escalate stuck loops. Real reliability gap.

### Hooks / plugin extension points
- Hermes `hooks.py`: YAML+importlib discovery, wildcard matching (`command:*`), sync/async handlers, errors never block pipeline.
- AGH (`internal/hooks/`, `daemon/hooks_bridge.go`, `daemon/hook_agent_events.go`): typed bindings, `Event` + `SessionLifecycle`, bridge protocol, resource reconciliation.
- **Verdict**: Near-parity. Only worth borrowing: the **wildcard selector** (`command:*`, `session:*`) if AGH doesn't already support it.

### Cron / scheduler
Hermes (`cron/jobs.py`, `cron/scheduler.py`):
- Schedule DSL `30m | 2h | every 30m | <cron> | <ISO ts>` via `parse_schedule` (`jobs.py:123-209`).
- **Grace window** (`_compute_grace_seconds`, `jobs.py:258-287`): missed runs within half-period (clamped 120s–2h) catch up; else fast-forward. Prevents thundering-herd after downtime.
- **At-most-once** (`advance_next_run` before execution, `jobs.py:643+`): flips `next_run_at` pre-run so crashes can't re-fire.
- **Pre-run script + wake-gate** (`scheduler.py:486-590`): sandboxed script under `HERMES_HOME/scripts/`, stdout injected into prompt, last JSON line `{"wakeAgent": false}` short-circuits the agent.
- **Inactivity timeout** (`scheduler.py:900-967`): kills when `agent.get_activity_summary().seconds_since_activity` exceeds limit; NOT wall-clock.
- **File-based `.tick.lock`** prevents overlapping ticks (`scheduler.py:1046+`).
- **Per-run archive** `cron/output/{job_id}/{ts}.md` full audit trail.
- **`[SILENT]` marker** (`scheduler.py:84`) suppresses delivery but still archives.

AGH (`automation/schedule.go`):
- gocron v2: cron / `every` / one-shot (`schedule.go:431-475`), singleton via `gocron.LimitModeReschedule`, clockwork clock, dispatcher interface.

**Verdict**: AGH scheduler is structurally better. Port:
1. **Misfire grace window** per-job `CatchUpPolicy` (none/grace/all) on `ScheduleSpec` — gocron just skips past times.
2. **Pre-run command + wake-gate** JSON contract; sandbox under `~/.agh/scripts/`.
3. **Per-run artifact archive** `~/.agh/automation/runs/<job_id>/<run_id>.{md,json}` for UI + `agh automation logs`.
4. **Inactivity-based run timeout** from dispatched session's activity tracker.
5. **`[SILENT]` sentinel** for "monitor" agents that only speak on change.

### Background jobs beyond cron
- Hermes: pairing expiry, `prune_old_entries`, directory refresh every 5 min, memory flush watcher.
- AGH: `store/session_liveness.go`, `observe/reconcile.go`. Scattered.
- **Verdict**: Consider a single **periodic-tasks runner** wired through `daemon/boot.go` that owns liveness, resume-repair sweeps, artifact GC, bundle re-index.

### Status / health surface
- Hermes `status.py`: JSON PID file, `/proc/<pid>/stat` validation, XDG scope locks, `--replace` takeover marker.
- AGH `daemon/lock.go`, `restart.go`: PID, socket, restart op ledger — similar rigor.
- **Verdict**: Near-parity. Worth stealing the **per-integration health map** (`status.py:232-272`) so operators see webhook trigger reachability, skill catalog state, etc.

### Mirror pattern
- Hermes `mirror.py`: cross-session transcript writes for multi-agent context.
- AGH: event store + notifier cover in-process fan-out; `session/network_peer.go` handles peer routing without cross-writes.
- **Verdict**: **Skip** — cross-transcript writes would violate transcript-as-source-of-truth.

### Delivery reliability
- Hermes: truncate-oversize → local archive fallback → live adapter first, standalone HTTP fallback → `last_delivery_error` tracked separately from `last_error`.
- AGH: no delivery surface yet.
- **Verdict**: When AGH adds delivery, **split `run.status` from `run.delivery_status`** (`jobs.py:592-640`) so delivery retries don't re-run the agent.

### Config hot-reload
- Hermes re-reads `.env` + `config.yaml` every cron run (`scheduler.py:763-768`).
- AGH: load-once at boot, resource reconcile for runtime changes.
- **Verdict**: **Skip**. Resource reconcile is the right granularity.

## Patterns worth stealing

1. **DeliveryTarget grammar** (`origin | local | <platform>[:id[:thread]]`) — drop into `automation.Job.Deliver` plus a `core.DeliveryRouter` used by automation runs and any future webhook trigger response path. `gateway/delivery.py:28-104`.
2. **Misfire grace window** with half-period clamp (120s..2h) and catch-up vs fast-forward classification. Attach to `ScheduleSpec` in `internal/automation/model/`. `cron/jobs.py:258-287`.
3. **At-most-once via preemptive `advance_next_run`** — call the scheduler's internal update-next-fire *before* dispatching the run, so a daemon crash mid-run won't re-fire on restart. `cron/jobs.py:643+`.
4. **Pre-run script + wake-gate JSON contract** — optional `script` field on an automation Job, output prepended to the prompt, last stdout line parsed as `{"wakeAgent": false}` to short-circuit. Tightly sandboxed under `~/.agh/scripts/`. `cron/scheduler.py:486-590`.
5. **Inactivity-based run timeout** surfaced from the dispatched session's last-activity tracker, not wall-clock. `cron/scheduler.py:900-967`.
6. **Per-run artifact archive** under `~/.agh/automation/runs/<job_id>/<run_id>.{md,json}` with prompt, delivery outcome, duration. Feeds UI + `agh automation logs`.
7. **Resume-pending vs suspend-recently-active** on daemon boot — two-tier recovery for in-flight sessions; classify by `updated_at` window and escalate stuck sessions via consecutive-failure counter. `gateway/session.py:849-984`.
8. **`[SILENT]` sentinel** for scheduled agent outputs that should archive but not notify.
9. **Split `last_error` from `last_delivery_error`** so delivery retries don't require re-running the agent. `cron/jobs.py:600-612`.
10. **Wildcard hook event matching** (`command:*`) if `internal/hooks/` doesn't already support it.

## Explicitly skip

- Multi-chat-platform adapters (Telegram, Discord, Slack, WhatsApp, Signal, Matrix, Mattermost, DingTalk, Feishu, WeCom, Weixin, SMS, Email, WeChat, QQBot, BlueBubbles, HomeAssistant) — out of scope for AGH's local-first agent OS. `gateway/platforms/` 27 files, 2375 LOC in `base.py` alone.
- Progressive-edit stream consumer with flood-control backoff — SSE is not rate-limited, no chat platform to edit. `gateway/stream_consumer.py`.
- Pairing codes / DM allowlists — not needed for UDS/local HTTP. `gateway/pairing.py`.
- PII redaction helpers tied to chat-platform IDs. `gateway/session.py:176-184`.
- Channel directory refresh (platform-specific chat enumeration). `gateway/channel_directory.py`.
- Per-run `.env` / `config.yaml` hot-reload — AGH's resource reconcile is the right granularity.
- `--replace` takeover marker and SIGTERM gymnastics specific to systemd unit flap-loop scenarios; AGH's `RestartOperation` handoff is cleaner. `gateway/status.py:428-552`.
- Cross-session mirror writes — AGH's notifier + event store already fan out correctly inside the process; cross-transcript writes would violate the transcript-as-source-of-truth invariant.

Key AGH files referenced:
- `/Users/pedronauck/Dev/compozy/agh/internal/automation/schedule.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/automation/manager.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/automation/types.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/daemon/restart.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/session/resume_repair.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/api/core/sse.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/hooks/`
- `/Users/pedronauck/Dev/compozy/agh/internal/daemon/hooks_bridge.go`
