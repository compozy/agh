# Hermes vs AGH — Observability & Diagnostics

## Executive Summary
- AGH ships a very thin logger (`internal/logger/logger.go`, 107 LOC — JSON slog, one file, one level, no rotation, no redaction); Hermes ships a 390-LOC logging subsystem with rotation, component routing, verbose mode, redacting formatter and thread-local session context.
- AGH has **zero secret redaction** anywhere on the log path — every API key, JWT, DB password or OAuth code that appears in an ACP stderr line, event payload, or error will land in `agh.log` verbatim. Hermes' `agent/redact.py` is ~190 LOC of production-grade regexes and is applied inside the `RedactingFormatter`.
- AGH has no `doctor` / `dump` / `debug share` CLI. `agh observe health` only returns an in-memory snapshot (`internal/observe/health.go:18-68`). Hermes' `hermes doctor`, `hermes dump`, and `hermes debug share` are the difference between "user pastes a GitHub issue" and "user pastes a support URL".
- AGH has **no ACP error classifier**: every agent subprocess failure collapses into opaque `stopCause` / `waitErr` strings (`internal/session/manager_lifecycle.go:240`). Hermes ships a 12-reason taxonomy (`agent/error_classifier.py`) with `retryable`, `should_rotate_credential`, `should_compress`, `should_fallback` hints.
- Log rotation, log rate-limit / spam prevention, crash bundle capture, and any form of metrics counters/histograms are **entirely missing** from AGH — token aggregation (`observer.go:595`) is the only quantitative series that exists.

## Capability-by-Capability Gap Analysis

### Structured logging + correlation IDs
- **Hermes**: `hermes_logging.py:90-119` installs a `LogRecordFactory` that injects `session_tag` onto every record, per-thread via `threading.local()`; enter/exit at conversation start (`set_session_context`). Format includes `[session_id]` transparently.
- **AGH today**: `slog.JSONHandler` with no record factory (`logger.go:85`). Correlation happens only when callers manually `.With("session_id", …)` — done in three spots (`session/manager_helpers.go:202`, `manager_start.go:347`, `resume_repair.go:358`). `acp/`, `observe/`, `api/httpapi/`, `network/` handlers log without session/workspace/request IDs. HTTP requests get no request_id middleware (checked `api/httpapi/middleware.go` + `server.go:385` — only `gin.Recovery()` is wired).
- **Gap**: AGH lacks both automatic record enrichment and an HTTP/UDS request_id middleware. Cross-reference log lines with web UI / CLI calls is impossible today.
- **Priority**: P0
- **Recommended adoption**: Add a `slog.Handler` wrapper in `internal/logger/` that pulls `session_id`, `workspace_id`, `agent_name`, `request_id` from `ctx` (via typed keys), plus a `gin` middleware + UDS handler wrapper that stamps `request_id` onto the context. Every handler already has `ctx` — change is mechanical.

### Secret redaction
- **Hermes**: `agent/redact.py` (~190 LOC) — known-prefix vendor regexes (`sk-`, `ghp_`, `AKIA`, `eyJ…`, Telegram bot tokens, Slack `xoxb`, GitHub PAT, JWTs, Discord snowflakes, DB connstrings, URL userinfo, form bodies, query strings). `RedactingFormatter` (line 332) runs on every log record. Env-snapshotted flag so an LLM can't disable mid-session (line 60).
- **AGH today**: No redaction. `grep -r "redact" internal/` returns only unrelated `_test.go` strings and Daytona shell escaping. `os.Environ()` is read raw (`internal/acp/` spawn paths).
- **Gap**: Total. `OPENAI_API_KEY=sk-…` will land in `agh.log` any time an ACP subprocess stderrs its env, any time an error message wraps the env, or any time an agent event payload echoes back an argv.
- **Priority**: P0 (blocker for first release — pushing logs to support, pastebin, or GitHub is unsafe).
- **Recommended adoption**: Port `_PREFIX_PATTERNS`, `_JSON_FIELD_RE`, `_AUTH_HEADER_RE`, `_JWT_RE`, `_DB_CONNSTR_RE`, `_URL_USERINFO_RE` to a `internal/logger/redact.go`. Wrap the handler inside `New()` (`logger.go:85-87`) with a `redactingHandler` that rewrites string `slog.Value`s before writing. Env-snapshot `AGH_REDACT_SECRETS` at package init.

### Log rotation + size limits
- **Hermes**: `RotatingFileHandler` with configurable `max_size_mb` and `backup_count` (`hermes_logging.py:156-261`), separate errors.log at 2 MB x 2 backups, gateway.log at 5 MB x 3. `_ManagedRotatingFileHandler` subclass chmods 0660 after rotation (multi-user systemd service case).
- **AGH today**: Plain `os.OpenFile(…O_APPEND|O_CREATE|O_WRONLY…)` (`logger.go:68`). File grows forever. No `lumberjack.v2` import in `go.mod`.
- **Gap**: A long-running daemon will fill disk. Operator has no knob to cap log volume.
- **Priority**: P0
- **Recommended adoption**: `go get gopkg.in/natefinch/lumberjack.v2`; replace `os.OpenFile` with `&lumberjack.Logger{Filename: path, MaxSize: 5, MaxBackups: 3, MaxAge: 28, Compress: true}`. Expose `logging.max_size_mb`, `logging.max_backups`, `logging.max_age_days` in `internal/config/config.go`.

### Error classification
- **Hermes**: `agent/error_classifier.py` — 12-reason enum (`FailoverReason`), `ClassifiedError` dataclass with `retryable / should_compress / should_rotate_credential / should_fallback` hints, HTTP-status + message-pattern + error-code pipeline (lines 242-415), provider-specific quirks (Anthropic thinking sig, OpenRouter metadata.raw unwrap, 402 vs 429 disambiguation, large-session disconnect → context overflow).
- **AGH today**: Only coarse stop-reason classification in `internal/session/stop_reason.go:14` (`classifyStopReason`) + `liveness.go:58-155` (session-level crash/recovery states). Nothing inspects the ACP subprocess stderr, nothing distinguishes auth/rate-limit/context-overflow/transport errors. `TaskEvent.Error` is a free-form string (`tasks_test.go:540 "rate limit"` is just a literal).
- **Gap**: Operators can't tell if a session failed because of API key, rate limit, crash, or network. No data drives retry / failover. User-facing errors are opaque.
- **Priority**: P1 (agents still retry at their own layer, so AGH isn't on fire — but insight gap is large).
- **Recommended adoption**: New `internal/acp/errclass.go` that wraps ACP RPC errors and subprocess exit/stderr into a `FailureKind` enum (`AuthInvalid`, `RateLimited`, `ContextOverflow`, `SubprocessCrashed`, `ProtocolError`, `Transport`, `Unknown`). Store the kind on `store.StopReason` next to `stop_detail`. Surface in `observe/health.go` and SSE.

### `doctor` command
- **Hermes**: `hermes_cli/doctor.py` — 1240 LOC that verifies Python version, required packages, config files, auth providers, directory layout, SQLite WAL size, systemd linger, npm audit, API connectivity per provider (OpenRouter, Anthropic, Z.AI, Kimi, DeepSeek, Bedrock via boto3, …), tool availability, skill hub, memory provider, profiles. `--fix` mode auto-heals missing dirs, stale config keys, broken symlinks.
- **AGH today**: None. `grep "doctor\|diagnose" internal/cli/` returns nothing. `agh observe health` (`internal/cli/observe.go:76`) only prints the Observer snapshot (active sessions, DB sizes, version).
- **Gap**: Users with a misconfigured `~/.agh/agh.toml` or a missing `claude` binary have no one-shot diagnostic. Support burden will be high from day one.
- **Priority**: P0
- **Recommended adoption**: Add `internal/cli/doctor.go` with checks: (1) home layout (`aghconfig.EnsureHomeLayout`), (2) `agh.toml` schema validation, (3) each configured agent command resolves via `exec.LookPath`, (4) global DB + per-session DB open OK, (5) WAL size < 50 MB (Hermes pattern — `doctor.py:590-613`), (6) UDS socket permissions, (7) extension marketplace reachability (optional). Add `--fix` for mkdir + WAL checkpoint.

### Debug dump / bundle
- **Hermes**: `hermes_cli/dump.py` (325 LOC) — compact copy-pasteable text: version, git short-SHA, OS, Python, profile, model/provider, API-key presence matrix, toolsets, MCP server count, memory provider, gateway status, cron summary, skill count, non-default config overrides. `hermes_cli/debug.py` (`run_debug_share`) collects `dump` + redacted log tails (agent/errors/gateway) + full logs, uploads to paste.rs, auto-schedules DELETE after 6 h via `~/.hermes/pastes/pending.json` (avoids the leaked sleeping-subprocess bug — see comment at `debug.py:208-221`).
- **AGH today**: None. No `agh dump`, no `agh debug share`. Version/commit is visible only via `agh version` (present in `internal/cli/root.go`).
- **Gap**: Users opening issues will attach ad-hoc `ls ~/.agh`, partial configs, or nothing. Support will guess.
- **Priority**: P1 (big DX win — not strictly release-blocking if redaction lands first).
- **Recommended adoption**: `internal/cli/dump.go` reading `aghconfig.Config`, `version.Current()`, env-key presence, list of registered agents, MCP servers, extension count, session count from `globaldb`. Defer `debug share` to post-1.0 — it requires a paste target policy and redaction to be battle-tested.

### Log filtering CLI (`hermes logs`)
- **Hermes**: `hermes_cli/logs.py` — tail, follow, `--level`, `--session`, `--component`, `--since 1h`, supports all three log files. `_read_last_n_lines` (line 278) does proper chunked reverse reads for files >1 MB.
- **AGH today**: None. Users run `tail -f ~/.agh/logs/agh.log` manually; no component filter because there are no components; no session filter because session_id is not on every line (see correlation gap).
- **Gap**: Medium — tail/grep is workable, but once logs are JSON-structured, ad-hoc `jq` is awkward.
- **Priority**: P2
- **Recommended adoption**: `agh observe logs --session S --level warn --since 30m --follow` over the existing JSON file. Post-MVP.

### Health check semantics
- **Hermes**: doctor does live HTTP probes to every configured provider (`doctor.py:840-996`). Gateway status via `get_gateway_runtime_snapshot()`.
- **AGH today**: `Observer.Health` is in-memory only — active session count, agents, DB sizes, uptime, version (`internal/observe/health.go:18-68`). Bridge health (`BridgeAggregateHealth`) and task health (`TaskHealth`) are derived from recorded events. **No probe of downstream ACP agent availability** (no `exec.LookPath` on agent commands, no dry-run ACP handshake).
- **Gap**: A health check passes even if the configured `claude` binary is deleted.
- **Priority**: P1
- **Recommended adoption**: Extend `Health` with `AgentProbes []AgentProbe{Name, CommandResolved bool, LastHandshakeAt time.Time, LastError string}` populated once per minute by a background goroutine.

### Metrics (counters, gauges, histograms)
- **Hermes**: No true metrics. `agent/insights.py` aggregates sessions / tokens / tool calls / skills / activity patterns from SQLite on demand. Similar in shape to AGH's `observe/query.go` + `observe/tasks.go`.
- **AGH today**: Same pattern — on-demand SQL aggregation in `internal/observe/query.go` (117 LOC) and `tasks.go`. No Prometheus, no OpenTelemetry.
- **Gap**: None — both systems deliberately skip. For a local-first daemon, aggregated-on-read is fine. If/when AGH grows a Grafana story, revisit.
- **Priority**: P2
- **Recommended adoption**: Keep the status quo. If adding anything, expose `/observe/metrics` in Prometheus exposition format derived from existing aggregates; do not pull in an `otel` runtime dependency for a single-binary daemon.

### User-facing error messages
- **Hermes**: `agent/display.py` (skimmed) formats classifier output into "Your API key for OpenRouter looks invalid — run `hermes setup`" style. `doctor.py:860-870` maps OpenRouter 402 to a concrete remediation: `hermes config set model.provider <provider>` + funding link.
- **AGH today**: CLI errors bubble up raw `fmt.Errorf` chains (`internal/cli/*.go`). Session stop reasons are opaque short strings.
- **Gap**: A fix-forward hint layer is missing.
- **Priority**: P2
- **Recommended adoption**: Once the error classifier above lands, map each `FailureKind` to a single-line remediation string surfaced in the CLI and web UI.

### Rate-limited log spam prevention
- **Hermes**: No explicit rate limiter — relies on `_NOISY_LOGGERS` allow-list dropped to WARNING (`hermes_logging.py:50-65`).
- **AGH today**: No per-component overrides. `WithLevel` applies globally.
- **Gap**: A misbehaving ACP subprocess that re-emits stderr on a hot loop will saturate the log.
- **Priority**: P2
- **Recommended adoption**: When redaction/log rotation lands, also add a per-logger-key debounce (`slog.Handler` middleware that drops identical messages emitted >N times in T seconds). Defer until real-world noise is observed.

### Crash/trace capture
- **Hermes**: `gin` has no equivalent; Python crashes go to stderr. `hermes debug share` captures tail to a paste URL after the fact.
- **AGH today**: `gin.Recovery()` (`server.go:385`) catches HTTP handler panics with a default stack trace. No global `runtime/debug.SetPanicOnFault` or crash-file writer. UDS server handler panic behavior unchecked.
- **Gap**: A goroutine panic in `session/` or `acp/` will kill the daemon with only a stderr trace.
- **Priority**: P1
- **Recommended adoption**: A `panichandler` package used at every `go func` launch site that writes `~/.agh/crashes/YYYYMMDD-HHMMSS-<pid>.log` with redacted stack + recent log tail, then rethrows. Add `defer panichandler.Recover(logger, "session.start")` to the ~dozen goroutine entry points in `internal/session/` and `internal/acp/`.

## Patterns worth stealing

1. **Record factory for context enrichment** (`hermes_logging.py:90-119`) — port to Go as a `slog.Handler` that pulls from `ctx`. Avoids forgetting `.With(…)` at 40+ log sites.
2. **Component prefix map** (`hermes_logging.py:143-149`) — AGH's package names already serve as prefixes (`session.*`, `acp.*`, `observe.*`). Publish them as a constant and let `agh observe logs --component acp` filter.
3. **WAL-size doctor check** (`doctor.py:590-613`) — 4 lines of code, catches stuck SQLite checkpoints that are otherwise invisible.
4. **`dump` as primitive, `debug share` as composition** (`debug.py:389-442` reuses `run_dump`) — keeps redaction policy in one place.
5. **Snapshotted "can LLM disable this?" flag** (`redact.py:60`) — in an agent-first system, an LLM *will* try to `unset AGH_REDACT_SECRETS`. Freeze at process start.
6. **Per-provider live probe in doctor** (`doctor.py:840-996`) — map AGH agents to `exec.LookPath` + optional `--version` dry-run + optional ACP handshake dry-run.
7. **Classifier that emits recovery hints, not just reasons** (`error_classifier.py:62-82`) — `retryable/should_rotate/should_fallback` are what the retry loop actually needs; the enum alone is half the value.
8. **Pending-deletion JSON journal** (`debug.py:41-147`) — the anti-pattern avoided (per-paste sleeping subprocess) is worth a code comment regardless of whether AGH adopts `debug share`.

## Explicitly skip
- **Full `hermes doctor` scope (1240 LOC)** — AGH has no Python packages, no Node submodules, no provider SDKs to probe, no cron/gateway systemd story yet. Target ~250 LOC that covers home layout, config schema, agent command resolution, DB health, extension marketplace reachability.
- **paste.rs / dpaste.com upload from `debug share`** — for a release-day local-first daemon the surface area isn't worth it. Keep `dump` local-only; let users copy-paste.
- **Insights engine port (`agent/insights.py`, 930 LOC)** — AGH's `observe/query.go` + `observe/tasks.go` already cover sessions + token stats. Don't duplicate cost-estimation until an LLM-billing-reconciliation feature is actually scoped.
- **Thread-local session context (`_session_context = threading.local()`)** — Python idiom; Go's `context.Context` is the right carrier. Pull from `ctx` instead.
- **Python `_NOISY_LOGGERS` list** — AGH has no third-party logging library to silence; this is a Python-ecosystem problem.
- **Multi-file logging (agent.log / errors.log / gateway.log)** — AGH has one surface (the daemon). Single `agh.log` + JSON level filter is enough; don't fragment.
