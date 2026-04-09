# Hermes: Session Resilience Analysis

## 1. Stop Reason Taxonomy

Hermes does **not** define a formal enum for stop reasons. Instead, stop reasons are tracked across two independent dimensions: the **session-level** `end_reason` (persisted to SQLite) and the **turn-level** `_turn_exit_reason` (diagnostic logging only). Additionally, the LLM API's `finish_reason` is captured per-message.

### 1.1 Session-Level End Reasons (`sessions.end_reason` column)

These are free-form strings passed to `SessionDB.end_session()`. Observed values across the codebase:

| end_reason | Where set | Meaning |
|---|---|---|
| `"cli_close"` | `cli.py:8536` | User exited the CLI (Ctrl-C, `/exit`, EOF) |
| `"user_exit"` | Tests and examples | Explicit user termination |
| `"new_session"` | `cli.py:3443` | User started a new session (`/new`) |
| `"resumed_other"` | `cli.py:3521` | User switched to a different session via `/resume` |
| `"branched"` | `cli.py:3607` | User branched to a new session via `/branch` |
| `"session_reset"` | `gateway/session.py:763,865` | Gateway session reset (idle timeout, manual `/reset`) |
| `"session_switch"` | `gateway/session.py:919` | Gateway session switched to a named session |
| `"compression"` | `run_agent.py:6049` | Context compression triggered a session split (new child session) |
| `"cron_complete"` | `cron/scheduler.py:803` | Scheduled cron job completed |
| `"timeout"` | Test fixtures | Session timed out |

**Key observation:** Hermes has no crash-detection end_reason. If the process dies, the session's `ended_at` remains NULL and `end_reason` remains NULL -- the session is effectively "still running" in the database. This is recovered on resume (see Section 2).

### 1.2 Turn-Level Exit Reasons (`_turn_exit_reason`, diagnostic only)

These are logged at INFO/WARNING level at the end of every `run_conversation()` call. They are NOT persisted. Observed values:

| _turn_exit_reason | Condition |
|---|---|
| `"text_response(finish_reason=stop)"` | Normal completion -- model returned text without tool calls |
| `"text_response(finish_reason=length)"` | Model hit max output tokens |
| `"interrupted_by_user"` | User sent a new message while agent was running (gateway interrupt) |
| `"interrupted_during_api_call"` | Interrupt detected while waiting for API response |
| `"budget_exhausted"` | `IterationBudget.remaining <= 0` |
| `"max_iterations_reached(N/M)"` | `api_call_count >= max_iterations` |
| `"error_near_max_iterations(msg)"` | API error when near iteration limit |
| `"unknown"` | Default -- set at loop start, overwritten if a specific reason applies |

### 1.3 LLM API Finish Reasons (per-message `finish_reason`)

Stored in the `messages.finish_reason` column per assistant message:

| finish_reason | Meaning |
|---|---|
| `"stop"` | Normal completion |
| `"tool_calls"` | Model wants to call tools |
| `"length"` | Response truncated by max output tokens |
| `"incomplete"` | Codex Responses API: partial response |

The `stop_reason_map` in `run_agent.py:7737` normalizes Anthropic's stop reasons:
```python
stop_reason_map = {
    "end_turn": "stop",
    "tool_use": "tool_calls",
    "max_tokens": "length",
    "stop_sequence": "stop",
}
```

### 1.4 Result Dictionary Returned by `run_conversation()`

Every turn returns a structured dict with:
```python
{
    "final_response": str | None,
    "completed": bool,          # True if response exists AND api_calls < max_iterations
    "partial": bool,            # True only when stopped due to invalid tool calls
    "interrupted": bool,        # True if user interrupt triggered
    "interrupt_message": str,   # The message that triggered the interrupt
    "api_calls": int,
    "messages": list,
    "model": str,
    "input_tokens": int,
    "output_tokens": int,
    ...
}
```

**What's missing from Hermes:** There is no single canonical `stop_reason` field in the result. The caller must infer from the combination of `completed`, `partial`, `interrupted`, and `final_response is None`. AGH should define a proper enum.

---

## 2. Session Repair on Resume

### 2.1 Resume Mechanism

Hermes supports session resume through three paths:

1. **CLI `--continue` / `--resume`** (`cli.py:2390-2422`, `cli.py:2583-2640`)
2. **CLI `/resume` slash command** (`cli.py:3488-3560`) for mid-conversation session switching
3. **ACP `_restore()`** (`acp_adapter/session.py:333-405`) for editor reconnections after process restart

### 2.2 What Happens on Resume

The resume flow performs these steps:

1. **Look up session in SQLite** (`SessionDB.get_session(session_id)`) -- validates the session exists
2. **Load full message history** (`SessionDB.get_messages_as_conversation(session_id)`) -- restores all role/content/tool_calls/reasoning fields
3. **Filter out metadata entries** -- strips `role="session_meta"` entries
4. **Clear ended_at/end_reason** (`SessionDB.reopen_session()`) -- marks the session as active again
5. **Pass loaded history as `conversation_history`** to the next `run_conversation()` call

### 2.3 Consistency Checks Performed (or Not Performed)

**What Hermes DOES check:**
- Session existence in database
- Empty message history (falls back to "starting fresh")
- System prompt caching: on resume, loads the stored `system_prompt` from the session record instead of rebuilding (preserves Anthropic prefix cache)
- Budget warning cleanup: `_strip_budget_warnings_from_history()` removes turn-scoped budget pressure strings from previous turns that could confuse the model
- Todo store hydration: `_hydrate_todo_store()` recovers in-memory todo state from the most recent todo tool response in conversation history
- Preflight context compression: before entering the main loop, checks if restored history already exceeds the model's context threshold (handles model downgrade between sessions)

**What Hermes does NOT check:**
- **No orphaned tool call detection on resume.** If the session crashed mid-tool-execution (assistant message has tool_calls but some/all tool results are missing), the API call will fail with a mismatched tool_call_id error. The ContextCompressor's `_sanitize_tool_pairs()` only runs after compression, not on resume.
- **No crash marker.** There is no equivalent to `end_reason="crash"`. A crashed session looks identical to an active session (ended_at is NULL).
- **No message integrity validation.** No check for role alternation invariants, no verification that the last message is in a valid state.
- **No checkpoint/rollback.** Although Hermes has a checkpoint system (`checkpoints_enabled`), it is for file-system snapshots (undo tool changes), not for conversation state rollback.

### 2.4 Crash Recovery is Implicit, Not Explicit

Hermes relies on the model to handle inconsistent state gracefully. Key patterns:

1. **`ensure_session()`** (`hermes_state.py:502-521`): If `create_session()` failed at startup (transient SQLite lock), the flush path uses INSERT OR IGNORE to create the session retroactively.

2. **`_get_messages_up_to_last_assistant()`** (`run_agent.py:1955-1984`): Can roll back to the last complete assistant turn, but is only used for trajectory saving, not for resume.

3. **Error result injection** (`run_agent.py:9204-9229`): When a tool execution error occurs, the code walks backward to find the assistant message with pending tool_calls and fills in error results for any unanswered tool_call_ids. This prevents the "orphan tool call" API error.

4. **Context compression summary prefix** (`context_compressor.py:28-35`): Explicitly tells the model that earlier turns were compacted and the session state may reflect prior work:
```
"[CONTEXT COMPACTION] Earlier turns in this conversation were compacted
to save context space. The summary below describes work that was already
completed, and the current session state may still reflect that work..."
```

### 2.5 ACP Session Restore (`acp_adapter/session.py:333-405`)

The ACP path is more robust because it must handle editor reconnections:

1. Query the database for the session record
2. Validate `source == "acp"` (only restore ACP sessions)
3. Extract `cwd` from `model_config` JSON
4. Restore `provider`, `base_url`, `api_mode` from session metadata
5. Recreate a fresh AIAgent with the original configuration
6. Load conversation history from the database
7. Re-register task-specific cwd overrides for tools

---

## 3. Loop/Recursion Guards

Hermes implements multiple layers of guards against infinite loops and resource exhaustion.

### 3.1 Iteration Budget (Primary Guard)

**File:** `run_agent.py:168-211`

```python
class IterationBudget:
    def __init__(self, max_total: int):
        self.max_total = max_total  # Default: 90 for parent, 50 for subagents
        self._used = 0
        self._lock = threading.Lock()

    def consume(self) -> bool:
        """Try to consume one iteration. Returns True if allowed."""

    def refund(self) -> None:
        """Give back one iteration (for execute_code turns)."""
```

- **Parent agent:** `max_iterations=90` (default, configurable)
- **Subagent:** `delegation.max_iterations=50` (configurable via `config.yaml`)
- **Per-turn reset:** Budget resets at the start of each `run_conversation()` call (`run_agent.py:7071`)
- **Refund mechanism:** `execute_code` (programmatic tool calling) iterations are refunded so they don't eat the budget
- **Thread-safe:** Uses `threading.Lock` for concurrent subagent access

The main loop guard (`run_agent.py:7303`):
```python
while api_call_count < self.max_iterations and self.iteration_budget.remaining > 0:
```

### 3.2 Budget Pressure Warnings (Soft Guard)

**File:** `run_agent.py:674-679, 6777-6798`

Two tiered thresholds injected into tool result content (not as separate messages):

| Threshold | Level | Message |
|---|---|---|
| 70% of max_iterations | CAUTION | "Iteration N/M. You're approaching the iteration limit. Start wrapping up..." |
| 90% of max_iterations | WARNING | "Iteration N/M. You are almost out of iterations. Respond NOW..." |

Previous turns' budget warnings are stripped on resume (`_strip_budget_warnings_from_history()`) to prevent models from refusing to make tool calls.

### 3.3 Max Iterations Handler (Graceful Degradation)

**File:** `run_agent.py:6840-6990`

When max iterations are reached, Hermes doesn't just stop -- it asks the model for a summary:
```python
summary_request = (
    "You've reached the maximum number of tool-calling iterations allowed. "
    "Please provide a final response summarizing what you've found..."
)
```

This ensures the user always gets a response, even when the agent runs out of budget.

### 3.4 Subagent Delegation Depth Limit

**File:** `tools/delegate_tool.py:37`

```python
MAX_DEPTH = 2  # parent (0) -> child (1) -> grandchild rejected (2)
MAX_CONCURRENT_CHILDREN = 3
```

- `delegate_task` is in `DELEGATE_BLOCKED_TOOLS`, so children cannot recursively delegate
- But even if they could, `MAX_DEPTH=2` would prevent it
- Maximum 3 concurrent child agents per parent (`MAX_CONCURRENT_CHILDREN`)
- Tasks beyond the limit are silently truncated (`run_agent.py:2903-2929`)

### 3.5 Tool Call Deduplication (Per-Turn)

**File:** `run_agent.py:2931-2947`

```python
@staticmethod
def _deduplicate_tool_calls(tool_calls: list) -> list:
    """Remove duplicate (tool_name, arguments) pairs within a single turn."""
    seen: set = set()
    unique: list = []
    for tc in tool_calls:
        key = (tc.function.name, tc.function.arguments)
        if key not in seen:
            seen.add(key)
            unique.append(tc)
```

This catches the common case where models emit the same tool call multiple times in one response. Applied at `run_agent.py:8892`.

### 3.6 Invalid Tool Call Retry Limits

Multiple retry counters with hard limits of 3:

| Counter | Max Retries | What it guards |
|---|---|---|
| `_invalid_tool_retries` | 3 | Model hallucinating tool names that don't exist |
| `_invalid_json_retries` | 3 | Model producing malformed JSON in tool arguments |
| `_empty_content_retries` | 3 | Model returning empty/null responses |
| `_incomplete_scratchpad_retries` | 2 | Unclosed reasoning scratchpad tags |
| `_codex_incomplete_retries` | variable | Codex Responses API returning `finish_reason=incomplete` |
| `_thinking_prefill_retries` | variable | Thinking block signature failures |

All counters reset at the start of each turn (`run_agent.py:7045-7050`).

### 3.7 Tool Name Repair (Fuzzy Matching)

**File:** `run_agent.py:2949-2975`

Before declaring a tool call invalid, Hermes attempts repair:
1. Try lowercase
2. Try normalized (hyphens/spaces to underscores)
3. Try fuzzy match (difflib, cutoff=0.7)

### 3.8 Tool Result Size Budget (Context Overflow Prevention)

**File:** `tools/tool_result_storage.py`, `tools/budget_config.py`

Three-layer defense against context window overflow from tool outputs:

| Layer | Threshold | Action |
|---|---|---|
| Per-tool output cap | Tool-specific | Tools pre-truncate their own output |
| Per-result persistence | 100K chars default | Large outputs written to disk, replaced with preview + file path |
| Per-turn aggregate budget | 200K chars | If all tool results in a turn exceed 200K, largest are spilled to disk |

Special case: `read_file` has `threshold=inf` to prevent infinite persist-read-persist loops.

### 3.9 Context Compression (Automatic)

**File:** `agent/context_compressor.py`

When the conversation approaches the model's context limit (default: 50% threshold), Hermes:
1. Prunes old tool results (cheap, no LLM call)
2. Protects head messages (system prompt + first exchange)
3. Protects tail messages by token budget (~20K tokens)
4. Summarizes middle turns with a structured LLM summary
5. Sanitizes orphaned tool_call/tool_result pairs after compression
6. On subsequent compressions, iteratively updates the previous summary

Post-compression: `_sanitize_tool_pairs()` fixes orphaned tool results and inserts stub results for orphaned tool calls.

### 3.10 Gateway Inactivity Timeout

**File:** `gateway/run.py:7143-7242`

- Default: 1800 seconds (30 minutes) of inactivity
- Configurable via `HERMES_AGENT_TIMEOUT` env var or `agent.gateway_timeout` config
- Warning at 50% of timeout (default 900s)
- Uses activity tracker (`_touch_activity()`) rather than wall clock
- On timeout: interrupts the agent, sends diagnostic summary to user

### 3.11 Stale Connection Eviction (Gateway)

**File:** `gateway/run.py:1870-1918`

Detects leaked locks from hung/crashed handlers:
- Checks both idle time AND wall-clock age
- Wall-clock TTL: max(10x timeout, 2 hours)
- Logs diagnostic info: last activity, iteration count, current tool

### 3.12 API Error Classification for Recovery

**File:** `agent/error_classifier.py`

Structured error taxonomy (`FailoverReason` enum) with recovery hints:

| Reason | Recovery |
|---|---|
| `auth` | Refresh/rotate credential |
| `billing` | Rotate credential, then fallback |
| `rate_limit` | Backoff, rotate credential, fallback |
| `overloaded` | Backoff |
| `server_error` | Retry |
| `timeout` | Rebuild client, retry |
| `context_overflow` | Compress context |
| `payload_too_large` | Compress payload |
| `model_not_found` | Fallback to different model |
| `format_error` | Abort or strip + retry |
| `thinking_signature` | Retry (Anthropic-specific) |
| `long_context_tier` | Compress (Anthropic tier gate) |
| `unknown` | Retry with backoff |

Heuristic for server disconnect + large session: reclassified as `context_overflow` (not `timeout`) when `approx_tokens > context_length * 0.6` or `num_messages > 200`.

---

## 4. Key Code References

| File | Lines | What |
|---|---|---|
| `run_agent.py` | 168-211 | `IterationBudget` class -- thread-safe iteration counter |
| `run_agent.py` | 473, 559-562 | `max_iterations` default (90) and budget initialization |
| `run_agent.py` | 674-679 | Budget pressure thresholds (70%, 90%) |
| `run_agent.py` | 6777-6798 | `_get_budget_warning()` -- tiered budget pressure messages |
| `run_agent.py` | 6840-6990 | `_handle_max_iterations()` -- graceful degradation with summary |
| `run_agent.py` | 7043-7071 | Per-turn retry counter reset and budget reset |
| `run_agent.py` | 7303 | Main loop guard: `while api_call_count < max_iterations and budget.remaining > 0` |
| `run_agent.py` | 9246-9256 | Turn exit: max iterations reached, completion determination |
| `run_agent.py` | 9267-9309 | Turn-exit diagnostic logging |
| `run_agent.py` | 9338-9362 | Result dictionary construction |
| `run_agent.py` | 2931-2947 | `_deduplicate_tool_calls()` -- per-turn dedup |
| `run_agent.py` | 2949-2975 | `_repair_tool_call()` -- fuzzy tool name repair |
| `run_agent.py` | 8770-8860 | Invalid tool call / JSON retry logic with limits |
| `run_agent.py` | 1894-1953 | `_persist_session()` and `_flush_messages_to_session_db()` |
| `run_agent.py` | 9204-9229 | Error result injection for orphaned tool calls |
| `run_agent.py` | 382-407 | `_strip_budget_warnings_from_history()` |
| `hermes_state.py` | 385-399 | `end_session()` and `reopen_session()` |
| `hermes_state.py` | 502-521 | `ensure_session()` -- crash-recovery INSERT OR IGNORE |
| `hermes_state.py` | 951-993 | `get_messages_as_conversation()` -- session restore |
| `cli.py` | 2390-2422 | CLI session resume with validation |
| `cli.py` | 2583-2640 | `_preload_resumed_session()` -- early history load |
| `cli.py` | 3488-3560 | `/resume` slash command -- mid-conversation session switch |
| `acp_adapter/session.py` | 333-405 | `_restore()` -- ACP session restore from database |
| `agent/context_compressor.py` | 452-510 | `_sanitize_tool_pairs()` -- fix orphans after compression |
| `agent/context_compressor.py` | 612-745 | `compress()` -- main compression algorithm |
| `agent/error_classifier.py` | 25-58 | `FailoverReason` enum |
| `agent/error_classifier.py` | 231-404 | `classify_api_error()` -- structured error classification |
| `tools/delegate_tool.py` | 36-38 | `MAX_CONCURRENT_CHILDREN=3`, `MAX_DEPTH=2`, `DEFAULT_MAX_ITERATIONS=50` |
| `tools/delegate_tool.py` | 532-540 | Delegation depth limit check |
| `tools/tool_result_storage.py` | 0-36 | Three-layer tool result budget system |
| `tools/budget_config.py` | 1-52 | Budget constants and config |

---

## 5. Patterns Worth Adopting

### 5.1 Definitely Adopt

1. **Iteration Budget with Refund** -- The `IterationBudget` pattern (thread-safe counter with `consume()/refund()`) is clean and simple. Refunding cheap RPC-style calls (like `execute_code`) prevents budget exhaustion from non-LLM operations. AGH should implement this with a similar thread-safe counter per session.

2. **Tiered Budget Pressure Warnings** -- Injecting budget warnings into tool results at 70% and 90% thresholds is elegant. It uses the model's own reasoning to decide when to wrap up rather than hard-cutting. AGH should adopt this pattern, injecting budget pressure into the context at configurable thresholds.

3. **Graceful Max-Iterations Handler** -- Instead of just stopping, requesting a summary from the model ensures the user always gets a response. AGH should always attempt a summary turn before terminating for budget exhaustion.

4. **Structured Error Classification** -- The `FailoverReason` enum with recovery action hints (`retryable`, `should_compress`, `should_rotate_credential`, `should_fallback`) is much better than scattered string matching. AGH should define a comparable Go enum with similar recovery hints.

5. **Tool Call Deduplication** -- Per-turn deduplication of identical `(name, arguments)` pairs catches a common model failure mode. Simple and effective.

6. **Tool Name Repair with Fuzzy Matching** -- Auto-correcting hallucinated tool names (lowercase, normalize, difflib) before declaring failure is a practical resilience pattern.

7. **Turn-Exit Diagnostic Logging** -- The structured log at the end of every turn (`reason, model, api_calls, budget, last_msg_role, last_tool`) is invaluable for debugging "the agent just stopped" issues. AGH should emit a structured log event at every session turn boundary.

### 5.2 Adopt with Improvements

8. **Stop Reason Taxonomy** -- Hermes's approach is too informal. `_turn_exit_reason` is a free-form string only used for logging. `end_reason` is another free-form string. AGH should define a canonical `StopReason` enum that covers: `completed`, `max_iterations`, `budget_exhausted`, `interrupted`, `error`, `timeout`, `crash`, `session_reset`, `session_switch`, `compression`. This enum should be persisted, returned in APIs, and used in metrics.

9. **Session Repair on Resume** -- Hermes does almost no repair. AGH should explicitly:
   - Detect orphaned tool calls (assistant has tool_calls but missing tool results) and inject error stubs
   - Validate role alternation invariants
   - Set `end_reason="crash"` for sessions with NULL `ended_at` that are being resumed
   - Log a structured "session recovered" event with diagnostics

10. **Context Compression Tool Pair Sanitization** -- Hermes only sanitizes after compression. AGH should also sanitize on session load/resume and after any message list mutation.

### 5.3 Do Differently

11. **Budget Reset Per Turn** -- Hermes resets the iteration budget every turn (`run_conversation()` call). For a daemon like AGH that manages long-lived sessions, a per-session cumulative budget with per-turn sub-budgets may be more appropriate. The gateway's inactivity timeout is a better overall session guard.

12. **No Formal Loop Detection** -- Hermes has NO actual loop detection (detecting the agent making the same sequence of tool calls repeatedly). It relies entirely on iteration limits to bound loops. AGH should implement a sliding window check: if the last N tool call sequences are identical, inject a "you appear to be stuck" warning before the budget runs out.

13. **Crash Recovery** -- Hermes's crash recovery is purely implicit (session with NULL ended_at). AGH, as a daemon, should implement heartbeat-based liveness detection and explicit crash marking. When the daemon restarts, all sessions with NULL ended_at should be inspected and either resumed or marked as `end_reason="crash"`.

14. **No Formal State Machine** -- Hermes uses boolean flags (`interrupted`, `completed`, `partial`) to represent session state. AGH should use an explicit state machine: `created -> running -> paused -> completed | error | timeout | crashed`.
