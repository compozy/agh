---
name: hermes-vs-agh-release-analysis
description: Consolidated gap analysis between Hermes (.resources/hermes/) and AGH, prioritized for the first release
type: analysis
date: 2026-04-24
inputs:
  - analysis_state_persistence.md
  - analysis_observability.md
  - analysis_acp_lifecycle.md
  - analysis_gateway_cron.md
  - analysis_tools_security.md
  - analysis_memory_context.md
  - analysis_cli_setup.md
---

# Hermes vs AGH — Release-Readiness Gap Analysis

Seven parallel investigations compared Hermes (Python agent runtime, ~80K LOC) against AGH (Go Agent OS daemon) across state/persistence, observability, ACP lifecycle, gateway/scheduling, tools/security, memory/context, and CLI/setup. This document consolidates the findings into a single, prioritized release plan.

## TL;DR — The One-Paragraph Verdict

AGH's **core runtime is architecturally stronger** than Hermes in the areas where it shipped first: subprocess lifecycle (process groups + SIGTERM/KILL escalation + descendant walk), JSON-RPC framing, session state machine with `canTransition` gate, hot/cold DB split (`agh.db` + per-session `events.db`), corrupt-DB quarantine, gocron v2 automation, memory FTS5 + BM25 catalog + dream consolidation runtime, and resume classification (`ClassifyInactiveMetaForRecovery`). **These must not regress as we add features.** AGH's **release blockers** all cluster in the **guardrail and DX layers** Hermes invested in for years: secret redaction, log rotation, a `doctor` command, dangerous-command approval, SSRF/URL safety, skill content scanning with trust tiers, memory-write injection scanning, and a cross-platform install/uninstall/packaging story. The single largest production risk, by LOC ratio (~65:1), is the tools/skills security surface.

## P0 — Release Blockers (must land before v1.0)

These are correctness, safety, or "will be demoed and embarrass us" issues. No optional.

| # | Gap | Domain | AGH target | Source |
|---|-----|--------|------------|--------|
| 1 | **Secret redaction on every log record** — `sk-…`, JWT, DB URIs, URL userinfo, OAuth codes currently land in `agh.log` verbatim | Observability | new `internal/logger/redact.go` + wrap `slog.Handler` in `New()` | [observability §redaction](analysis_observability.md) |
| 2 | **Log rotation** — `os.OpenFile(O_APPEND)` grows forever | Observability | `lumberjack.v2` with `max_size_mb`/`max_backups`/`max_age_days` in `config.toml` | [observability §rotation](analysis_observability.md) |
| 3 | **`agh doctor` command** — zero preflight; users have no diagnostic | Observability + CLI | new `internal/cli/doctor.go`, ~250 LOC (home layout, config schema, agent `exec.LookPath`, DB health, WAL size, UDS perms) | [observability §doctor](analysis_observability.md), [cli §doctor](analysis_cli_setup.md) |
| 4 | **Auto log correlation** — session/workspace/request IDs only stamped at 3 sites; HTTP has no request_id middleware | Observability | context-aware `slog.Handler` that pulls typed keys; gin middleware + UDS handler wrapper | [observability §correlation](analysis_observability.md) |
| 5 | **SQLITE_BUSY retry with `BEGIN IMMEDIATE`** — pure `busy_timeout(5000)` causes convoy stalls under daemon + CLI + reconciler + automation concurrent writers | State | new `internal/store/sql_helpers.go :: ExecWithRetry(ctx, db, fn)` w/ jittered backoff | [state §1](analysis_state_persistence.md) |
| 6 | **Dangerous-command approval flow** — current `approve-all` bootstrap default + 3-mode static policy means an agent can `rm -rf /` and AGH rubber-stamps it | Tools/Security | expand `internal/acp/permission.go`: per-pattern regexes, per-session/permanent scopes, `once/session/always/deny` persistence | [tools §approval](analysis_tools_security.md) |
| 7 | **SSRF / URL safety** — zero URL guards; any URL-capable MCP or extension can probe private/loopback/metadata IPs | Tools/Security | new `internal/security/urlsafety` (block private/loopback/CGNAT/link-local/metadata, DNS fails-closed, both v4 + v6) | [tools §url](analysis_tools_security.md) |
| 8 | **Skill content scanning + trust-tier install matrix** — `skills/verify.go` runs 11 patterns at list-time but never gates install; a marketplace skill with `rm -rf /` passes as long as its hash matches | Tools/Security | expand `internal/skills/verify.go` with Hermes `skills_guard` patterns (80+ across 10 categories), structural checks (file count, size, binary ext, invisible unicode, symlink escape), then gate `internal/skills/install.go` on trust-tier × verdict matrix | [tools §skill-scan](analysis_tools_security.md) |
| 9 | **Memory-write injection scanner** — memory is the highest-leverage prompt-injection vector in the entire OS; `store.Write` has zero scanning | Memory | port `_scan_memory_content` (tools/memory_tool.py:65-102) into `internal/memory/store.go :: Write` | [memory §privacy](analysis_memory_context.md) |

**Why these nine:** 1–4 mean support tickets are either unsolvable or unsafe to attach logs to. 5 means concurrent-writer deadlocks as soon as real traffic hits the daemon. 6–9 mean a compromised or sloppy agent has direct RCE/exfiltration paths.

## P1 — Important Hardening (land in v1.1 or early post-release)

Sequentially numbered continuing from P0. Domain tag in the `[State]`, `[Observability]` etc. prefix.

| # | Gap | Notes | Source |
|---|-----|-------|--------|
| 10 | **[State] `schema_migrations` table** | linear runner in `internal/store/schema.go`; keep column-probe helpers as safety nets | [state §2](analysis_state_persistence.md) |
| 11 | **[State] Wire `observability.retention_days`** | config knob currently does nothing; add daily sweep in `internal/observe/` | [state §11](analysis_state_persistence.md) |
| 12 | **[State] `agh backup` / `agh restore`** | use `VACUUM INTO` for WAL-safe DB snapshot; zip home minus socket/pid | [state §6](analysis_state_persistence.md), [cli §backup](analysis_cli_setup.md) |
| 13 | **[State] Subprocess HOME isolation** | opt-in `agents.isolate_home`; inject `HOME` + `XDG_*` per session to stop agents sharing `~/.config`, `~/.claude`, `~/.npm`. Port Hermes' test matrix | [state §12](analysis_state_persistence.md) |
| 14 | **[Observability] ACP error classifier** | small `FailureKind` enum (`AuthInvalid`, `RateLimited`, `ContextOverflow`, `SubprocessCrashed`, `ProtocolError`, `Transport`, `Unknown`) stored on `StopReason` and surfaced in `observe/health.go` + SSE | [observability §classifier](analysis_observability.md), [acp §7](analysis_acp_lifecycle.md) |
| 15 | **[Observability] Health probes for downstream agents** | `exec.LookPath` + optional ACP handshake every 60s; surface as `AgentProbes` in `Health` | [observability §health](analysis_observability.md) |
| 16 | **[Observability] Crash bundle / panic handler** | `panichandler.Recover` at every goroutine entry in `internal/session/` + `internal/acp/`, writing `~/.agh/crashes/YYYYMMDD-HHMMSS-<pid>.log` | [observability §crash](analysis_observability.md) |
| 17 | **[ACP] Jittered exponential backoff primitive** | port `retry_utils.jittered_backoff` to `internal/retry` or inline in `internal/subprocess` | [acp §6](analysis_acp_lifecycle.md) |
| 18 | **[ACP] `WithPromptTimeout(d)` option** | today the only hung-prompt bound is daemon-restart-time stall classification | [acp §10](analysis_acp_lifecycle.md) |
| 19 | **[ACP] Benign-probe log filter** | frozen set of ACP `ping`/`health` method names suppressed from `NewMethodNotFound` stderr noise | [acp §patterns](analysis_acp_lifecycle.md) |
| 20 | **[Gateway] Two-tier restart recovery** | classify sessions by `updated_at` window: within `restart_drain_timeout` → `restart_drained` (retry-safe), older → `crashed_during_start` (forced wipe). Add `consecutive_resume_failures` counter to escalate stuck sessions | [gateway §restart](analysis_gateway_cron.md) |
| 21 | **[Gateway] Misfire grace window** | on `ScheduleSpec` — per-job `CatchUpPolicy` (none/grace/all) with half-period clamp (120s..2h) | [gateway §cron](analysis_gateway_cron.md) |
| 22 | **[Gateway] At-most-once via preemptive `advance_next_run`** | flip `next_run_at` *before* dispatching so crashes can't re-fire | [gateway §cron](analysis_gateway_cron.md) |
| 23 | **[Gateway] Per-run artifact archive** | under `~/.agh/automation/runs/<job_id>/<run_id>.{md,json}` for UI + `agh automation logs` | [gateway §cron](analysis_gateway_cron.md) |
| 24 | **[Gateway] DeliveryTarget grammar** | (`origin \| local \| webhook:<id> \| session:<id>`) + router for `Job.Deliver` and future webhook response paths | [gateway §delivery](analysis_gateway_cron.md) |
| 25 | **[Gateway] Split `last_error` from `last_delivery_error`** | so delivery retries don't re-run the agent | [gateway §delivery](analysis_gateway_cron.md) |
| 26 | **[Tools] OSV check before spawning npx/uvx MCP servers** | query `api.osv.dev` for `MAL-*` advisories; fail-open on network error | [tools §osv](analysis_tools_security.md) |
| 27 | **[Tools] MCP OAuth 2.1 + PKCE** | `HermesTokenStorage`-equivalent with `0o600` atomic writes, absolute `expires_at`, ephemeral localhost callback, per-server token files. Blocks Linear/Notion/Drive integrations | [tools §mcp-oauth](analysis_tools_security.md) |
| 28 | **[Tools] Symlink-escape guard on skills** | `internal/skills/loader.go` and `provenance.go` must verify symlink targets stay inside the skill dir | [tools §path-sandbox](analysis_tools_security.md) |
| 29 | **[Tools] Process registry with checkpoint-on-write + PID probe on boot** | `$AGH_HOME/processes.json`; recover across restart via `procutil.Alive` | [tools §process-registry](analysis_tools_security.md) |
| 30 | **[Tools] Per-thread tool interrupt** | per-session `atomic.Bool` pollable by long-running tools without threading `ctx` through every callback | [tools §interrupt](analysis_tools_security.md) |
| 31 | **[Tools] Tool-call budget** | per-tool `max_result_size_chars`, watch-pattern rate limit, session-scoped invocation count | [tools §budget](analysis_tools_security.md) |
| 32 | **[Tools] Credential isolation + `0o600` token files** | sanitize subprocess env; enforce file perms on token caches | [tools §creds](analysis_tools_security.md) |
| 33 | **[Memory] `@file:` / `@folder:` / `@git:` / `@url:` context references** | with 50%/25% token budget + sensitive-path blocklist (`.ssh`, `.aws`, `.gnupg`, `.env`, `.netrc`, `.pgpass`, …) | [memory §references](analysis_memory_context.md) |
| 34 | **[Memory] `agh memory health` CLI** | `HealthStats` already computed in `store.go:404-458`; just unexposed | [memory §health](analysis_memory_context.md) |
| 35 | **[Memory] Memory-provider hooks** | `on_turn_start`, `on_session_end`, `on_pre_compress` — even without a full plugin system, so observe/transcript can participate | [memory §plugin-arch](analysis_memory_context.md) |
| 36 | **[CLI] `agh config show\|edit\|set\|get\|path\|check`** | users hand-edit TOML today | [cli §config](analysis_cli_setup.md) |
| 37 | **[CLI] `agh uninstall`** | stops daemon, removes launchd/systemd unit, strips PATH edits, optionally purges `~/.agh` | [cli §uninstall](analysis_cli_setup.md) |
| 38 | **[CLI] `agh auth login\|logout\|status`** | with file-locked `~/.agh/auth.json` at mode 0600 | [cli §auth](analysis_cli_setup.md) |
| 39 | **[CLI] `agh completion {bash,zsh,fish,powershell}`** | Cobra built-in; ~30 LOC | [cli §completion](analysis_cli_setup.md) |
| 40 | **[CLI] Install script** | (`curl \| sh`) with arch detection, cosign verification, shell-rc edit, `~/.local/bin` symlink | [cli §install](analysis_cli_setup.md) |
| 41 | **[CLI] Packaging stanzas in `.goreleaser.yml`** | `brews:`, `nfpms:`, `scoops:`, `dockers:` (goreleaser-pro already paid for) | [cli §packaging](analysis_cli_setup.md) |
| 42 | **[CLI] `AGH_MANAGED` convention** | bake env var into package-manager installs; `config set` / `update` defer to `brew upgrade` etc. | [cli §managed](analysis_cli_setup.md) |
| 43 | **[CLI] `.env` credential ASCII-sanitization + multi-key repair** | 10-line port of Hermes' empirical bug fixes | [cli §env-loader](analysis_cli_setup.md) |

## P2 — Nice-to-Have (post-v1.1)

| # | Gap | Source |
|---|-----|--------|
| 44 | **[State] Periodic `wal_checkpoint(PASSIVE)`** every ~5 min to bound WAL growth | [state §5](analysis_state_persistence.md) |
| 45 | **[State] Mode-preserving `AtomicWriteFile`** | [state §4](analysis_state_persistence.md) |
| 46 | **[State] Zip-traversal guard on `agh restore`** | [state §6](analysis_state_persistence.md) |
| 47 | **[State] Periodic `PRAGMA integrity_check`** in observer loop | [state §3](analysis_state_persistence.md) |
| 48 | **[Observability] `agh dump`** (support bundle, local-only; defer `debug share` upload until redaction battle-tested) | [observability §dump](analysis_observability.md) |
| 49 | **[Observability] `agh observe logs --session S --level warn --since 30m --follow`** | [observability §logs-cli](analysis_observability.md) |
| 50 | **[Gateway] Inactivity-based run timeout** (from dispatched session's activity tracker, not wall-clock) | [gateway §cron](analysis_gateway_cron.md) |
| 51 | **[Gateway] `[SILENT]` sentinel** for monitor-only scheduled agents | [gateway §cron](analysis_gateway_cron.md) |
| 52 | **[Gateway] Pre-run script + wake-gate JSON contract** for automation jobs | [gateway §cron](analysis_gateway_cron.md) |
| 53 | **[Gateway] Wildcard hook event selectors** (`command:*`, `session:*`) | [gateway §hooks](analysis_gateway_cron.md) |
| 54 | **[Gateway] Per-integration health map** (webhook reachability, skill catalog state, etc.) | [gateway §status](analysis_gateway_cron.md) |
| 55 | **[CLI] Profiles** (`~/.agh/profiles/<name>/` + `~/.local/bin/<profile>` wrapper) | [cli §profiles](analysis_cli_setup.md) |
| 56 | **[CLI] Banner + tips + version check** | [cli §banner](analysis_cli_setup.md) |
| 57 | **[CLI] `agh update` self-update** (deferring to package manager when `AGH_MANAGED` is set) | [cli §update](analysis_cli_setup.md) |
| 58 | **[CLI] TTY guards on every interactive command** | [cli §tty](analysis_cli_setup.md) |
| 59 | **[CLI] `agh extension enable\|disable` + `requires_env` handshake on install** | [cli §plugins](analysis_cli_setup.md) |
| 60 | **[Memory] `agh memory history`** exposing the existing `memory_operation_log` | [memory §op-log](analysis_memory_context.md) |
| 61 | **[Memory] Per-scope soft memory cap** that triggers forced dream consolidation | [memory §size](analysis_memory_context.md) |
| 62 | **[ACP] Usage-header bucket model** (if AGH ever proxies HTTP for bridges) | [acp §rate-limit](analysis_acp_lifecycle.md) |
| 63 | **[ACP] Cross-session rate-limit guard file** (`~/.agh/rate_limits/<agent>.json`) if shared-credential scenarios arise | [acp §rate-limit](analysis_acp_lifecycle.md) |

## Cross-Cutting Themes

Five patterns recur across multiple sub-analyses — they're architectural, not feature-level, and worth treating as first-class:

1. **Secret/sensitive-data boundary.** Log redaction (observability), memory-write scanning (memory), `@ reference` sensitive-path blocklist (memory), env-var ASCII sanitization (cli), credential file `0o600` perms (tools), env scrubbing on subprocess (tools). **Action**: a shared `internal/security/redact` package that exposes `(a) regex library + matcher`, `(b) `io.Writer` wrapper for log handler`, `(c) sensitive-path predicate`. Everything else consumes it.

2. **Structured classification over free-form strings.** ACP error classifier (observability + acp), dangerous-command patterns (tools), trust tiers × scan verdicts (tools), stop reasons (state), delivery outcomes (gateway). **Action**: when in doubt, ship an enum, not a `string`. Store the enum next to, not instead of, the human message.

3. **Auditability.** Memory operation log (already in AGH), per-run automation archive (P1), permission pre/post events (already in AGH), crash log files (P1), memory history CLI (P2). **Action**: every mutating subsystem emits a structured event that survives restart. SSE consumers are secondary — the primary is disk.

4. **Session isolation.** HOME isolation (state), per-thread tool interrupt (tools), credential isolation (tools), path sandboxing (already in AGH for ACP, missing for skills). **Action**: make isolation the default, not an opt-in knob, for any new feature. Today AGH runs subprocess-per-session for ACP (good); everything else should match.

5. **Fail-safely preflight.** Doctor (observability + cli), OSV check (tools), DNS-fails-closed (tools), managed-mode blocking (cli), config schema check (cli). **Action**: add a `Preflight()` hook to every long-running subsystem, and surface the aggregated result as `agh doctor`.

## What AGH Already Beats Hermes On (Do Not Regress)

Protect these in code review; they're hard-won wins:

- **Subprocess lifecycle**: cooperative shutdown → SIGTERM → `PostSignalGrace` → SIGKILL; Linux `/proc`-walk descendant signalling; 128 KiB bounded stderr tail; 10 MiB JSON-RPC frame cap (`internal/subprocess/`, `internal/procutil/`)
- **JSON-RPC framing**: strict `jsonrpc:"2.0"` check, atomic seq IDs, context-aware cancellation with pending-entry cleanup (`internal/subprocess/transport.go`)
- **Permission infrastructure**: stable request IDs, pre+post decision events, path-escape resolver walking existing ancestors, typed `permissionDecision` with preference ordering (`internal/acp/permission.go`)
- **Session state machine**: real `canTransition` gate (`internal/session/session.go:22-29, 599-698`)
- **Hot/cold DB split**: global index `agh.db` + per-session `events.db`; deleting a session = `rm -rf <dir>`
- **Corrupt DB quarantine**: rename to `.corrupt.<ts>` alongside `-wal`/`-shm` before reopening (`internal/store/sqlite.go`)
- **Resume classification**: `ClassifyInactiveMetaForRecovery` with `procutil.Alive(pid)` + 2-min stall heuristic; `IsLoadSessionResourceMissing` for ACP-side fallback
- **Memory substrate**: dual-scope Markdown + SQLite FTS5 catalog + frontmatter taxonomy + PID-based cross-process lock + `memory_operation_log` + dream consolidation runtime
- **Automation scheduler**: gocron v2 with singleton reschedule, clockwork clock — structurally better than Hermes' hand-rolled tick loop
- **Subprocess-per-session model** — no shared mutable agent object, no save/restore dance across concurrent prompts

## Explicitly NOT Adopt From Hermes

Tempting, but wrong fit or out of scope:

- **`copilot_acp_client.py` architecture** — string-stuffs ACP prompt from chat transcript, regex-scrapes `<tool_call>{...}</tool_call>`, auto-allows every permission. AGH's direct JSON-RPC is strictly correct.
- **Single-process multi-session with `ThreadPoolExecutor` + shared `AIAgent`** ([acp §12](analysis_acp_lifecycle.md)).
- **`asyncio ↔ thread` 5s log-and-drop bridge** — "dropped on timeout" is a bug, not a feature. AGH's blocking producer is correct.
- **V4A patch / unified-diff parsing in the runtime** — agents own their rendering; AGH forwards.
- **Slash-command interception inside the prompt handler** — already exposed via CLI/HTTP in AGH.
- **Full 1240-LOC `doctor.py` scope** — target ~250 LOC that covers home layout, config, agent `exec.LookPath`, DB, UDS, marketplace reachability.
- **`debug share` paste-URL upload** — redaction must be battle-tested first; keep `dump` local-only.
- **paste-URL sleeping-subprocess pending-deletion** — take the lesson (JSON journal); skip the mechanism.
- **`insights.py` engine port** (~930 LOC) — AGH's `observe/query.go` + `observe/tasks.go` already cover sessions + tokens.
- **Thread-local session context (`threading.local()`)** — Go's `context.Context` is the right carrier.
- **Multi-file logging** (`agent.log`/`errors.log`/`gateway.log`) — single `agh.log` + level filter is enough.
- **Chat-platform adapters** (Telegram/Discord/Slack/WhatsApp/Signal/Matrix/DingTalk/Feishu/WeCom/WeChat/QQBot/BlueBubbles/HomeAssistant, 27 files, 2375 LOC in `base.py` alone) — out of scope.
- **Progressive-edit stream consumer with flood-control backoff** — SSE is not rate-limited.
- **Pairing codes / DM allowlists** — not needed for UDS/local HTTP.
- **Cross-session mirror writes** — would violate transcript-as-source-of-truth.
- **`--replace` SIGTERM takeover gymnastics** — AGH's `RestartOperation` handoff is cleaner.
- **Nous subscription / credential-pool rotation** — Hermes-unique commercial integration.
- **Python venv / uv bootstrapping** — AGH is a Go binary.
- **Monolithic `cli.py` (10832 LOC) REPL** — belongs in the web UI / future TUI, not the daemon CLI.
- **Honcho / SOUL / personalities** — AGH-unique marketplace covers the use case.
- **Nix / container-mode dispatch, Termux/Android** — not current platforms.
- **HRR / holographic memory plugin** — solves retrieval at the embedding layer; FTS5 + BM25 is right for AGH's scope.
- **Honcho per-turn prefetch thread orchestration** — network-bound backend optimization; N/A for local FTS5.
- **Tirith remote binary download** — prefer bundling a Go-native scanner over downloading from GitHub Releases at first run (though the cosign+SHA-256 pattern IS worth stealing for any future binary deps).
- **`INSTALL_POLICY["agent-created"] = ("allow","allow","ask")`** — AGH should not let an agent self-install a skill without a human gate. Start stricter, relax later.
- **FTS5 over messages, trajectory JSONL dumps, shadow-git checkpoint manager** — duplicate existing AGH systems or irrelevant domain.

## Recommended Implementation Sequence

Phased so each phase is independently shippable:

### Phase 0 — Pre-release blockers (1-2 weeks)
Parallel tracks:
1. **Track A — Secret/log hygiene**: P0 #1 (redaction) → P0 #2 (rotation) → P0 #4 (correlation middleware). Same engineer. Unblocks everything else that logs.
2. **Track B — Data durability**: P0 #5 (`ExecWithRetry` + `BEGIN IMMEDIATE`). Independent; different engineer.
3. **Track C — Security gates**: P0 #6 (approval flow) → P0 #7 (URL safety) → P0 #8 (skill scanner + trust tiers) → P0 #9 (memory-write scanner). Same engineer; they share the `internal/security/redact` + regex library substrate.
4. **Track D — DX safety net**: P0 #3 (`agh doctor`). Can land with or after Track A.

### Phase 1 — Release hardening (2-3 weeks post-v1.0)
1. **State**: #10 schema migrations, #11 retention sweep, #12 backup/restore, #13 HOME isolation.
2. **Observability**: #14 ACP error classifier, #15 downstream health probes, #16 panic handler.
3. **ACP**: #17 jittered backoff, #18 `WithPromptTimeout`, #19 benign-probe filter.
4. **Gateway**: #20 two-tier restart recovery, #21 misfire grace, #22 at-most-once, #23 per-run archive, #24 DeliveryTarget grammar, #25 split delivery error.
5. **Tools**: #26 OSV check, #27 MCP OAuth, #28 symlink guard, #29 process registry, #30 tool interrupt, #31 budgets, #32 credential isolation.
6. **Memory**: #33 `@` references, #34 `memory health`, #35 provider hooks.
7. **CLI**: #36 `config`, #37 `uninstall`, #38 `auth`, #39 `completion`, #40 install script, #41 goreleaser stanzas, #42 `AGH_MANAGED`, #43 `.env` sanitization.

### Phase 2 — Post-release polish
Items #44–#63 as operational need emerges.

## Index of Sub-analyses

- [State, Persistence & Recovery](analysis_state_persistence.md) — SQLite fundamentals, migrations, backup, HOME isolation
- [Observability & Diagnostics](analysis_observability.md) — redaction, rotation, doctor, correlation, error classification
- [ACP & Agent Lifecycle](analysis_acp_lifecycle.md) — subprocess management, JSON-RPC, retry, permissions, prompt timeout
- [Gateway, Delivery & Scheduling](analysis_gateway_cron.md) — restart recovery, misfire, DeliveryTarget, artifact archive
- [Tools, Skills & Security](analysis_tools_security.md) — approval flow, SSRF, skill scanner, trust tiers, MCP OAuth, process registry
- [Memory & Context](analysis_memory_context.md) — injection scanner, `@` references, provider hooks, health CLI
- [CLI UX, Setup, Config & Packaging](analysis_cli_setup.md) — doctor, config tree, uninstall, auth, profiles, packaging, install script
