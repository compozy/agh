# Hermes vs AGH — ACP & Agent Lifecycle

## Executive Summary

- **AGH's subprocess lifecycle is already production-grade and strictly better than Hermes's.** AGH has a 4-state process machine (`internal/subprocess/process.go:39-47`), cooperative shutdown → SIGTERM → SIGKILL escalation (`:342-410`), a bounded JSON-RPC transport with pending-call cancellation (`internal/subprocess/transport.go:224-285`), and `/proc`-walk descendant signalling on Linux (`internal/procutil/process_group_unix.go:111-164`). Hermes uses `subprocess.Popen` + `terminate()/kill()` with no descendant handling (`.resources/hermes/agent/copilot_acp_client.py:283-299`). **Do not borrow Hermes's subprocess model.**
- **AGH lacks the error-classification and retry layer Hermes invested in.** Hermes's `error_classifier.py` is an 829-line priority-ordered pipeline that turns provider errors (auth, 402, 413, 429, 500/503, context-overflow, thinking-signature, long-context tier) into a structured `ClassifiedError` with action hints. AGH treats ACP errors as opaque strings (`internal/acp/client.go:648-657`, `handlers.go:805-818`) and has no retryable vs non-retryable distinction anywhere.
- **AGH has no backoff primitive and no cross-session rate-limit coordination.** Hermes's `retry_utils.jittered_backoff` (`.resources/hermes/agent/retry_utils.py:19-57`) and `nous_rate_guard` atomic shared-file guard (`nous_rate_guard.py:70-159`) are simple, valuable patterns. Nothing equivalent exists in AGH.
- **AGH has stronger permission and resume infrastructure than Hermes — protect this.** Path-escape protection (`internal/acp/permission.go:139-208`), turn-scoped terminal ownership (`internal/acp/handlers.go:395-406`, `849-880`), audit-trailed interactive permissions (`internal/acp/permission.go:301-352`), structured session-liveness classification (`internal/session/liveness.go:24-113`), and resume-fallback on ACP `ResourceNotFound` (`internal/acp/client.go:435-449`) are all absent from Hermes. Hermes auto-denies on timeout with no event emitted (`.resources/hermes/acp_adapter/permissions.py:59-75`).
- **AGH has no prompt deadline.** A hung child agent is only caught at daemon-restart time by the stall classifier (`internal/session/liveness.go:13-15`). Add a per-prompt timeout before first release.

## Capability-by-Capability Gap Analysis

### 1. Subprocess lifecycle

| Aspect | Hermes | AGH | Winner |
|---|---|---|---|
| Spawn | `Popen(text=True, bufsize=1)` | `exec.Cmd` + `execabs.LookPath` + process group (`internal/subprocess/process.go:196-230`) | AGH |
| Stdio reader | Python threads + `queue.Queue` (`copilot_acp_client.py:388-404`) | `bufio.Scanner` with `maxMessageBytes+1` buffer; rejects oversize frames (`transport.go:227-261`) | AGH |
| Shutdown | `terminate(); wait(2s); kill()` — no cooperative RPC | `shutdown` RPC → close stdin → SIGTERM → `PostSignalGrace` → SIGKILL → `WaitForCommandProcessGroupExit` (`process.go:342-410`) | AGH |
| Descendants | None — leaks grandchildren | Linux `/proc` walk + group signal + drain poll (`process_group_unix.go:111-164`) | AGH |
| Stderr capture | 40-line `deque` | 128 KiB tail `boundedBuffer` attached to exit error (`process.go:223`, `539-594`) | AGH |

**Verdict:** AGH is production-grade; Hermes is a prototype.

### 2. JSON-RPC framing

| Aspect | Hermes | AGH |
|---|---|---|
| Framing | Line-delimited JSON; unknown methods → `-32601` | Line-delimited JSON; strict `jsonrpc:"2.0"` check, hard-fail on malformed frame (`transport.go:224-250`) |
| ID correlation | Integer counter; ignores unsolicited responses | Atomic seq with string+numeric IDs; rejects fractional (`transport.go:200-423`) |
| Cancellation | Deadline only; no context plumbing | `ctx.Done()` deletes pending entry, returns ctx error (`transport.go:190-197`) |
| Max frame | Unbounded | 10 MiB cap with `"token too long"` translation |

**Verdict:** AGH wins. Nothing to borrow.

### 3. Permission requests

| Aspect | Hermes | AGH |
|---|---|---|
| Bridge | Worker-thread → `asyncio.run_coroutine_threadsafe`, 60s hardcoded, auto-deny with no audit event (`permissions.py:26-75`) | Stable request IDs, pre+post decision events with full `options`/`tool_input`/`tool_call` payload (`permission.go:301-352`, `handlers.go:244-297`) |
| Timeout | 60s hardcoded → auto-deny | `defaultPermissionWait = 5m`, configurable via `WithPermissionTimeout`; timeout emits `decisionRejectOnce` event |
| Decision fallback | String mapping | Typed `permissionDecision` with `selectPermissionOutcome` preference ordering (`permission.go:245-299`) |
| Interactive detection | No | `(decision, interactive)` return; auto emits one event, interactive emits two |
| Path safety | Copilot client only does local cwd check | `resolveExistingAwarePath` walks existing ancestors, handles symlinks, rejects escapes (`permission.go:139-225`) |

**Verdict:** AGH far ahead. Do not regress.

### 4. Tool invocation & result shapes

Hermes builds rich ACP tool-call content (V4A patch → diff blocks, unified-diff split, terminal `$ cmd` previews, 5000-char caps; `tools.py:100-339`). AGH is the *client* and forwards whatever the agent sends — this content construction lives on the agent side (e.g. Claude Code) where it belongs. AGH does add a 64 KiB UTF-8-safe terminal output ring (`handlers.go:940-989`). No gap for v1.

### 5. Event ordering & delivery

| Aspect | Hermes | AGH |
|---|---|---|
| Stream | Per-callback `run_coroutine_threadsafe().result(timeout=5)`, log-and-drop on failure (`events.py:27-40`) | Prompt-scoped buffered channel (cap configurable), `waitForPromptQuiescence` drains trailing updates before `EventTypeDone` (`client.go:940-965`) |
| Resume | ACP `session/load` + SQLite history (`session.py:366-424`) | ACP `session/load` + per-session event store + resume-repair with infra validation (`manager_lifecycle.go:38-101`) |
| Backpressure | Silent drop after 5s | Producer blocks; prompt-scoped lifetime |
| Mid-prompt usage | Captured only at end | `usage_update` notifications merged into running total (`handlers.go:299-327`, `788-803`) |

**Gain for AGH:** mid-prompt usage merging is a real win.

### 6. Retry & backoff

Hermes: `jittered_backoff(attempt, base=5, cap=120, jitter=0.5)` with counter-seeded PRNG for decorrelation (`retry_utils.py:19-57`). AGH: **none.** No retry anywhere in `internal/acp/`, `internal/session/`, or `internal/subprocess/`. **Real gap.** Port a small Go equivalent and use it in shutdown waits and future reconnect paths.

### 7. Error classification

Hermes: 829-line pipeline in `error_classifier.py` with `FailoverReason` enum and action hints (`retryable`, `should_compress`, `should_rotate_credential`, `should_fallback`). AGH: surfaces errors as `EventTypeError{Error: err.Error()}`; only one specific classifier exists — `IsLoadSessionResourceMissing` (`client.go:435-449`). **Medium-priority gap:** AGH's `handleProcessExit` only branches on `waitErr == nil` (`manager_lifecycle.go:122-142`) — a small classifier recognising transport disconnects, rate-limit signals, and stall-detected failures would unlock auto-recovery strategies.

### 8. Rate-limit handling

Hermes: `nous_rate_guard.py` atomic `mkstemp`+`os.replace` writes reset-time to `~/.hermes/rate_limits/<provider>.json` so all sessions self-throttle (`:70-159`). Also parses 12 `x-ratelimit-*` headers into time-decaying buckets (`rate_limit_tracker.py:92-130`). AGH: **none** — each child agent burns its own quota independently. **Low priority for v1** since AGH doesn't speak to providers directly; the child agent owns this. Reconsider if users with shared accounts report quota thrash.

### 9. Cancellation

| Aspect | Hermes | AGH |
|---|---|---|
| Client cancel | `state.cancel_event.set()` + `agent.interrupt()` (`server.py:407-416`) | `session/cancel` notification (`client.go:476-492`); prompt ctx-cancel auto-sends via `context.WithoutCancel(ctx)` with 1s deadline (`client.go:600-631`) |
| Mid-tool-drop | Polls event between steps | Cancels at prompt boundary; terminals killed on process ctx-done (`handlers.go:1014-1033`) |
| Stop couples to cancel | No | `Driver.Stop` sends `session/cancel` before `handle.Stop` (`client.go:541-552`) |

**Verdict:** AGH ahead. `context.WithoutCancel` usage is good — ensures notification sends even when caller ctx is dead.

### 10. Subprocess resource limits

Neither project sets rlimits, memory caps, or wall-clock timeouts. **AGH gap:** no per-prompt deadline. Add `WithPromptTimeout(time.Duration)` and wire through `runPrompt` alongside `ctx`.

### 11. Capability negotiation

Hermes advertises richer capabilities server-side (`load_session`, `fork`, `list`, `resume`, `session_capabilities`) (`server.py:339-351`) but is loose on protocol version (`server.py:318-320`). AGH reads `SupportsLoadSession`, mode list, model list (`client.go:266-269`, `916-931`) but ignores fork/list caps. Minor — fine for v1.

### 12. Multi-session isolation

Hermes: one process, shared `AIAgent`, mutates `agent.tool_progress_callback` per prompt with save/restore dance (`server.py:514-545`) — concurrency footgun. AGH: subprocess-per-session, no shared mutable state. **Do not regress.**

### 13. Upgrade / restart semantics

| Aspect | Hermes | AGH |
|---|---|---|
| Survives restart | SQLite restore on `load_session`/`resume_session` (`session.py:426-498`) | Per-session event store + `SessionMeta` JSON + `Resume` with validation (`manager_lifecycle.go:38-101`) |
| Liveness classification | None — unconditional restore | `ClassifyInactiveMetaForRecovery` rewrites Active/Stopping/Starting to Stopped with structured reasons via `procutil.Alive(pid)` + 2-min stall heuristic (`liveness.go:24-113`) |
| ACP session-gone fallback | None | `IsLoadSessionResourceMissing` detects `-32002` "resource not found" → fresh session start (`client.go:435-449`, `manager_lifecycle.go:74-101`) |
| Orphan subprocess detection | None | `syscall.Kill(pid, 0)` probe (`procutil.go:12-20`) |

**Verdict:** AGH significantly ahead. Do not regress.

## Patterns worth stealing

1. **Jittered exponential backoff primitive.** Port `retry_utils.jittered_backoff` (`retry_utils.py:19-57`) into `internal/retry` or inline in `internal/subprocess`. Signature: `JitteredBackoff(attempt int, base, cap time.Duration, jitter float64) time.Duration`. Use in shutdown escalation and any future reconnect loops.
2. **Benign-probe log filter.** `_BenignProbeMethodFilter` (`entry.py:31-60`) suppresses stderr tracebacks for ACP `ping`/`health` method-not-found responses that clients use for liveness. AGH currently surfaces every unknown method via SDK logs. A frozen set of benign probe names before calling `NewMethodNotFound` cuts production log noise.
3. **Small structured error classifier.** Not the 829-line Hermes monster — just `RetryAfter(err) (time.Duration, bool)`, `IsResourceMissing(err) bool`, `IsProtocolError(err) bool`, `IsRateLimited(err) bool`. Feed `handleProcessExit` and `runPrompt`-error paths. Priority-ordered pipeline in `error_classifier.py:242-415` is the template for ordering logic; keep AGH's version ACP-level only.
4. **Cross-session rate-limit guard file.** `nous_rate_guard.py`'s atomic `mkstemp`+`os.replace` pattern is clean and zero-deps. If AGH later needs to rate-limit shared-credential scenarios, `~/.agh/rate_limits/<agent-name>.json` is the template.
5. **`WithPromptTimeout` option.** Simple addition. Today the only upper-bound on a hung prompt is daemon-restart-time stall classification.
6. **Usage-header bucket model.** If AGH ever proxies HTTP for bridges or network peers, `parse_rate_limit_headers` (`rate_limit_tracker.py:92-130`) with `captured_at` + time-decayed `remaining_seconds_now` is a tidy model.

## Explicitly skip

- **`copilot_acp_client.py` architecture.** String-stuffs a chat transcript into the ACP prompt, regex-scrapes `<tool_call>{...}</tool_call>` from the response (`:27-28`, `156-226`), auto-allows every permission (`:535-544`). A hack to glue Copilot ACP into an OpenAI-SDK-shaped consumer. AGH's direct JSON-RPC is strictly correct — port none of this.
- **ThreadPoolExecutor + shared mutable agent** (`server.py:72`, `514-545`). AGH's subprocess-per-session is the right answer.
- **Python asyncio ↔ thread bridge with 5s log-and-drop** (`events.py:27-40`). Not applicable in Go; dropped-on-timeout is a bug, not a feature.
- **Auto-deny on timeout with no event.** Hermes does this (`permissions.py:59-75`); AGH emits events on timeout-reject, which is the right model.
- **Slash-command interception inside the prompt handler** (`server.py:98-138`, `636-779`). AGH already exposes these through its CLI/HTTP API — keep them there.
- **V4A patch / unified-diff parsing** (`tools.py:100-229`). Useful only because Hermes *is* the agent rendering diffs. AGH forwards agent-emitted content.
- **Single SQLite table for ACP+CLI sessions with `source="acp"` discriminator** (`session.py:392-399`). AGH's `globaldb` / `sessiondb` separation is cleaner — do not collapse.
