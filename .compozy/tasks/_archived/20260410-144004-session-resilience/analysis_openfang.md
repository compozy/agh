# OpenFang: Session Resilience Analysis

## 1. Stop Reason Taxonomy

### LLM-Level Stop Reasons

OpenFang defines a minimal 4-variant `StopReason` enum at the LLM protocol level (`crates/openfang-types/src/message.rs:207-216`):

```rust
pub enum StopReason {
    EndTurn,       // Model finished its turn naturally
    ToolUse,       // Model wants to call a tool
    MaxTokens,     // Model hit the output token limit
    StopSequence,  // Model hit a configured stop sequence
}
```

This is a **wire-level enum** that maps 1:1 to LLM API responses. It intentionally does NOT encode application-level stop reasons like "budget exceeded" or "user cancelled" -- those are expressed at higher layers.

### Application-Level Stop Reasons (Implicit Taxonomy)

OpenFang does NOT have an explicit `SessionStopReason` enum. Instead, stop reasons are distributed across error types, hook metadata, and loop exit paths. Reconstructing the full taxonomy from code:

| Stop Reason | How It's Expressed | Source File |
|---|---|---|
| **Completed naturally** | `StopReason::EndTurn` with non-empty text; `AgentLoopResult` returned with `silent: false` | `agent_loop.rs:464` |
| **Silent completion** | `NO_REPLY` / `[SILENT]` token detected; `AgentLoopResult.silent = true` | `agent_loop.rs:475-496` |
| **Max iterations exceeded** | `OpenFangError::MaxIterationsExceeded(u32)` error returned | `agent_loop.rs:968`, `error.rs:76` |
| **Max continuations (token limit)** | After `MAX_CONTINUATIONS` (5) consecutive `MaxTokens` responses, returns partial | `agent_loop.rs:898-937` |
| **Circuit breaker (loop guard)** | `OpenFangError::Internal(msg)` with "Circuit breaker" reason | `agent_loop.rs:685-704` |
| **Tool call blocked** | `LoopGuardVerdict::Block` -- individual tool skipped, loop continues | `agent_loop.rs:706-714` |
| **Rate limited** | `OpenFangError::LlmDriver("Rate limited after N retries")` | `agent_loop.rs:1019-1022` |
| **Overloaded** | `OpenFangError::LlmDriver("Model overloaded after N retries")` | `agent_loop.rs:1038-1041` |
| **Auth/billing failure** | `OpenFangError::LlmDriver(sanitized_msg)` -- non-retryable | `llm_errors.rs` |
| **Context overflow** | `LlmErrorCategory::ContextOverflow` -- triggers recovery pipeline | `context_overflow.rs`, `llm_errors.rs:32` |
| **Model not found** | `LlmErrorCategory::ModelNotFound` -- triggers fallback chain | `agent_loop.rs:978-984` |
| **Quota exceeded** | `OpenFangError::QuotaExceeded(msg)` -- hourly/daily/monthly | `metering.rs:27-60` |
| **Shutdown in progress** | `OpenFangError::ShuttingDown` | `error.rs:79` |
| **Agent crashed** | `AgentState::Crashed` -- tracked by supervisor | `agent.rs:185` |
| **Tool timeout** | Individual tool returns error after `TOOL_TIMEOUT_SECS` (120s) or `AGENT_TOOL_TIMEOUT_SECS` (600s) | `agent_loop.rs:44-58` |
| **Hook blocked** | Hook `fire()` returns `Err(reason)` -- tool call skipped | `agent_loop.rs:745-757` |
| **Approval denied** | Tool requires human approval, was denied -- tool skipped with guidance | `agent_loop.rs:850-863` |
| **Max restarts exceeded** | `Supervisor.record_agent_restart()` returns `Err(count)` | `supervisor.rs:79-95` |
| **Unresponsive (heartbeat)** | Agent inactive > 2x heartbeat interval -- flagged for recovery | `heartbeat.rs:186` |

### Hook Metadata for Stop Reasons

The `AgentLoopEnd` hook fires with structured JSON `data` containing a `reason` field. Known values:
- `"circuit_break"` -- loop guard circuit breaker fired
- `"max_continuations"` -- hit MAX_CONTINUATIONS limit
- `"max_iterations_exceeded"` -- hit max_iterations limit
- (absent/normal) -- completed successfully

### Agent Lifecycle States

The `AgentState` enum (`agent.rs:173-186`) tracks macro-level agent health:

```rust
pub enum AgentState {
    Created,     // Not yet started
    Running,     // Active
    Suspended,   // Paused
    Terminated,  // Permanently stopped
    Crashed,     // Awaiting recovery
}
```

### LLM Error Classification

OpenFang has a sophisticated 8-category error classifier (`llm_errors.rs`) that parses raw LLM API errors using pattern matching against 19+ provider error formats:

```rust
pub enum LlmErrorCategory {
    RateLimit,       // 429, quota exceeded
    Overloaded,      // 503, high demand
    Timeout,         // Network failures
    Billing,         // 402, insufficient credits
    Auth,            // 401/403, invalid key
    ContextOverflow, // Context window exceeded
    Format,          // Malformed request
    ModelNotFound,   // Unknown model
}
```

Each category has `is_retryable` and `is_billing` flags. RateLimit, Overloaded, and Timeout are retryable; the rest are not.

---

## 2. Session Repair on Resume

OpenFang has a **dedicated session repair module** (`crates/openfang-runtime/src/session_repair.rs`) that validates and fixes message history before every LLM call -- not just on resume. This is the primary consistency mechanism.

### When Repair Runs

Session repair runs at **three distinct points** in every agent loop iteration:

1. **Before the initial LLM call** (`agent_loop.rs:319`): `validate_and_repair(&llm_messages)` -- cleans the full message history
2. **After context overflow recovery** (`agent_loop.rs:389`): Re-validates after draining old messages, which may have split ToolUse/ToolResult pairs
3. **After silent failure retry** (`agent_loop.rs:519`): Re-validates if the LLM returned 0 input tokens (indicating broken tool pairing)

### Repair Phases (Ordered Pipeline)

The repair pipeline runs 5 phases in strict order (`session_repair.rs:52-197`):

**Phase 1 -- Collect ToolUse IDs**: Builds a `HashSet` of all `tool_use_id` values from assistant messages.

**Phase 2 -- Filter orphans and empties**:
- Drops `ToolResult` blocks whose `tool_use_id` has no matching `ToolUse` anywhere in history
- Drops empty messages (empty text or all blocks filtered out)
- Tracks stats: `orphaned_results_removed`, `empty_messages_removed`

**Phase 2b -- Reorder misplaced ToolResults** (`session_repair.rs:204-339`):
- Builds a `tool_use_id -> assistant_msg_index` map
- For each user message containing ToolResults, checks if it immediately follows the correct assistant message
- If misplaced, moves the ToolResult to the correct position (insert after the assistant message containing the matching ToolUse)
- Handles edge cases: appending to existing user messages, creating new user messages

**Phase 2c -- Deduplicate ToolResults** (`session_repair.rs:449-476`):
- Keeps only the first `ToolResult` for each `tool_use_id`
- Critical ordering: dedup runs BEFORE synthetic insertion (regression fix for issue #1013 -- Moonshot provider reuses `tool_use_id` values like `memory_store:0` across turns)

**Phase 2d -- Synthetic error results** (`session_repair.rs:352-438`):
- Counts ToolUse vs ToolResult occurrences per ID (not just presence -- handles providers that reuse IDs)
- For any orphaned ToolUse (no matching ToolResult), inserts a synthetic error result: `"[Tool execution was interrupted or lost]"` with `is_error: true`
- Inserts immediately after the assistant message containing the orphaned ToolUse

**Phase 2e -- Remove aborted assistant messages** (`session_repair.rs:483-519`):
- Detects assistant messages with empty content (blank text or no blocks) that indicate interrupted tool-use
- Removes these to prevent broken state from propagating

**Phase 3 -- Merge consecutive same-role messages** (`session_repair.rs:164-176`):
- The Anthropic API requires strict user/assistant alternation
- Merges consecutive messages with the same role by appending content blocks

### Repair Statistics

The repair returns a `RepairStats` struct tracking every fix applied:

```rust
pub struct RepairStats {
    pub orphaned_results_removed: usize,
    pub empty_messages_removed: usize,
    pub messages_merged: usize,
    pub results_reordered: usize,
    pub synthetic_results_inserted: usize,
    pub duplicates_removed: usize,
}
```

This is logged as a structured warning whenever any repair was needed.

### Additional Repair: Tool Result Sanitization

`strip_tool_result_details()` (`session_repair.rs:542-561`) sanitizes tool output before feeding it back to the LLM:
- Truncates to 10K chars max
- Strips base64 blobs >1000 chars (replaces with placeholder)
- Removes prompt injection markers (`<|im_start|>`, `<<SYS>>`, `IGNORE PREVIOUS INSTRUCTIONS`, etc.)

### Heartbeat Pruning

`prune_heartbeat_turns()` (`session_repair.rs:650-696`) removes `NO_REPLY` / `[no reply needed]` heartbeat turns from session history to save context budget. Keeps the last `keep_recent` messages intact.

### Context Overflow Recovery Pipeline

A separate 4-stage recovery pipeline (`context_overflow.rs`) handles sessions that grow too large:

| Stage | Trigger | Action |
|---|---|---|
| 1 | 70-90% of context window | Moderate trim: keep last 10 messages |
| 2 | >90% of context window | Aggressive trim: keep last 4 messages + summary marker |
| 3 | Still over after stage 2 | Truncate all historical tool results to 2K chars |
| 4 | Still over after stage 3 | Return `FinalError` -- suggest `/reset` or `/compact` |

The `safe_drain_boundary()` function ensures draining doesn't split ToolUse/ToolResult pairs across the boundary.

### Interim Saves (Crash Protection)

The agent loop performs **interim saves after every tool execution round** (`agent_loop.rs:893`):
```rust
// Interim save after tool execution to prevent data loss on crash
if let Err(e) = memory.save_session_async(session).await {
    warn!("Failed to interim-save session: {e}");
}
```

It also saves before returning on max iterations exceeded (`agent_loop.rs:950`), max continuations (`agent_loop.rs:908`), and circuit breaker (`agent_loop.rs:688`).

### Graceful Shutdown Sequence

The `ShutdownCoordinator` (`graceful_shutdown.rs`) enforces an ordered 10-phase shutdown:

1. Running -> Draining (stop new requests)
2. Broadcasting shutdown to WebSocket clients
3. Waiting for in-flight agent loops (with `agent_timeout`: 60s default)
4. Closing browser sessions
5. Closing MCP connections
6. Stopping background tasks
7. Flushing audit log
8. Closing database connections
9. Complete

Each phase has configurable timeouts: `drain_timeout` (30s), `agent_timeout` (60s), `total_timeout` (120s).

### Crash Recovery via Heartbeat

The heartbeat monitor (`heartbeat.rs`) detects crashed/unresponsive agents:
- Checks every 30s (configurable)
- Agent considered unresponsive after 2x its heartbeat interval (default: 180s timeout)
- Crashed agents get auto-recovery attempts up to `max_recovery_attempts` (default: 3)
- Recovery has a cooldown between attempts (default: 60s)
- After exhausting recovery attempts, agent is marked `Terminated`
- Idle agents (never processed a message) are skipped to prevent false crash-recover loops

The `Supervisor` (`supervisor.rs`) tracks per-agent restart counts and enforces `max_restarts` limits (default: 10 from `AutonomousConfig`).

---

## 3. Loop/Recursion Guards

### LoopGuard (Primary Loop Detection)

The `LoopGuard` (`crates/openfang-runtime/src/loop_guard.rs`) is the most sophisticated loop detection system I've found in any agent harness. It tracks tool calls within a single agent loop execution using SHA-256 hashes.

**Configuration defaults** (`LoopGuardConfig`, line 56-68):

| Parameter | Default | Purpose |
|---|---|---|
| `warn_threshold` | 3 | Identical calls before warning appended to result |
| `block_threshold` | 5 | Identical calls before call is blocked (skipped) |
| `global_circuit_breaker` | 30 | Total tool calls before entire loop is killed |
| `poll_multiplier` | 3 | Multiplier for poll tool thresholds (e.g., effective block = 15) |
| `outcome_warn_threshold` | 2 | Identical call+result pairs before warning |
| `outcome_block_threshold` | 3 | Identical call+result pairs before auto-block |
| `ping_pong_min_repeats` | 3 | Pattern repeats before ping-pong blocking |
| `max_warnings_per_call` | 3 | Warnings per call hash before upgrading to Block |

**Four verdict levels** (`LoopGuardVerdict`):

```rust
pub enum LoopGuardVerdict {
    Allow,              // Proceed normally
    Warn(String),       // Proceed, but append warning to tool result
    Block(String),      // Skip this tool call
    CircuitBreak(String), // Kill the entire agent loop
}
```

### Detection Strategies

**1. Simple repetition detection** (lines 146-218):
- SHA-256 hash of `(tool_name, serialized_params)` -- deterministic because serde_json sorts object keys
- Per-hash count tracked in `HashMap<String, u32>`
- Graduated response: Allow -> Warn (at threshold 3) -> Block (at threshold 5)

**2. Outcome-aware detection** (lines 251-281):
- After tool execution, hashes `(tool_name | params_json | result_truncated_1000)` -- the result is truncated to 1000 chars
- If the same call produces the same result 2 times: warning
- If 3 times: the call hash is added to a `blocked_outcomes` set, auto-blocking the NEXT `check()` call
- This catches loops where the agent retries the same failing operation

**3. Ping-pong detection** (lines 362-498):
- Maintains a ring buffer of last 30 call hashes
- Detects A-B-A-B alternating patterns (checks last 6 entries for 3 repeats of length 2)
- Detects A-B-C-A-B-C cycling patterns (checks last 9 entries for 3 repeats of length 3)
- Below `ping_pong_min_repeats`: warns. At or above: blocks
- Uses separate warning bucket key (`pingpong_{hash}`) to track ping-pong warnings independently

**4. Warning bucket / escalation** (lines 206-214):
- Tracks how many warnings have been emitted per call hash
- After `max_warnings_per_call` (3) warnings for the same call, upgrades to Block
- Prevents the agent from ignoring repeated warnings

**5. Poll tool handling** (lines 334-360):
- `POLL_TOOLS` list: `["shell_exec"]`
- A call is considered "polling" if the tool is in POLL_TOOLS AND the params contain status/poll/wait/watch/tail/ps/docker/kubectl keywords
- Generic poll detection: params JSON containing "status", "poll", or "wait"
- Poll calls get relaxed thresholds: effective_warn = 9, effective_block = 15

**6. Backoff suggestions** (lines 287-304):
- For poll calls, suggests increasing delays: 5s, 10s, 30s, 60s (capped at 60s)
- Returns `Option<u64>` in milliseconds; no backoff on first call

### Max Iterations Guard

Defined at the agent loop level (`agent_loop.rs:35`):
- `MAX_ITERATIONS = 50` (constant default)
- Overridable per-agent via `AutonomousConfig.max_iterations` (default: 50, checked at line 355)
- The loop guard's `global_circuit_breaker` is scaled up to `max_iterations * 3` for autonomous agents (line 363-367)
- When exceeded: session is saved, `AgentLoopEnd` hook fires with `reason: "max_iterations_exceeded"`, returns `OpenFangError::MaxIterationsExceeded`

### Max Continuations Guard

For `StopReason::MaxTokens` responses (`agent_loop.rs:898-945`):
- `MAX_CONTINUATIONS = 5`
- Consecutive MaxTokens responses are counted
- Under the limit: partial response added, "Please continue." appended, loop continues
- At the limit: returns partial response, fires hook with `reason: "max_continuations"`
- Counter resets on any ToolUse response (line 658)

### Context Window Guard

- `MAX_HISTORY_MESSAGES = 20` (`agent_loop.rs:66`) -- hard safety valve for message count
- Context budget system (`context_budget.rs`) for dynamic tool result truncation
- 4-stage context overflow recovery pipeline (see Section 2)

### Phantom Action Detection

`phantom_action_detected()` (`agent_loop.rs:71-87`) catches when the LLM claims to have performed an action (sent, posted, emailed) without actually calling any tools. This prevents hallucinated completions where the model fabricates task completion.

Detection: text contains action verbs ("sent", "posted", "emailed") AND channel references ("telegram", "slack", "discord"). If detected on iteration 0 with no tools executed, the agent is re-prompted:

```
[System: You claimed to perform an action but did not call any tools.
You must use the appropriate tool to actually perform the action.]
```

### Tool Error Fabrication Prevention

After tool errors, two guidance injections prevent the agent from fabricating results:

1. **TOOL_ERROR_GUIDANCE** (`agent_loop.rs:97-98`): Injected when any tool returns `is_error: true`. Tells the agent NOT to invent missing results or pretend failed tools succeeded.

2. **Non-denial error guidance** (`agent_loop.rs:872-882`): Separate guidance for non-approval-related errors, instructing the agent to report errors honestly.

### Approval Denial Loop Prevention

When tools are denied by approval policy (`agent_loop.rs:850-863`), the agent receives guidance to NOT retry denied tools, preventing an infinite retry loop where the agent keeps asking to execute denied operations.

### Inter-Agent Recursion Limits

- `AGENT_TOOL_TIMEOUT_SECS = 600` (10 minutes) for `agent_send` / `agent_spawn` tool calls
- Each agent has its own `max_iterations` limit
- The supervisor enforces `max_restarts` per agent (default: 10)
- No explicit recursion depth counter across nested agent calls

### Provider-Level Circuit Breaker

`ProviderCooldown` (`auth_cooldown.rs`) prevents request storms to failing providers:
- Three verdicts: `Allow`, `AllowProbe`, `Reject { reason, retry_after_secs }`
- Records successes and failures per provider
- After repeated failures, rejects requests with a cooldown period
- Periodically allows probe requests to test recovery

---

## 4. Key Code References

| Component | File | Key Lines |
|---|---|---|
| StopReason enum | `crates/openfang-types/src/message.rs` | 207-216 |
| AgentState enum | `crates/openfang-types/src/agent.rs` | 173-186 |
| Error taxonomy | `crates/openfang-types/src/error.rs` | 7-101 |
| LLM error classifier | `crates/openfang-runtime/src/llm_errors.rs` | 19-37 (categories), 241-392 (classifier) |
| Agent loop | `crates/openfang-runtime/src/agent_loop.rs` | 173-968 (main loop) |
| Loop guard | `crates/openfang-runtime/src/loop_guard.rs` | 1-949 (entire file) |
| Session repair | `crates/openfang-runtime/src/session_repair.rs` | 1-1409 (entire file) |
| Context overflow recovery | `crates/openfang-runtime/src/context_overflow.rs` | 117-222 (pipeline) |
| Graceful shutdown | `crates/openfang-runtime/src/graceful_shutdown.rs` | 1-443 (entire file) |
| Supervisor | `crates/openfang-kernel/src/supervisor.rs` | 1-228 |
| Heartbeat monitor | `crates/openfang-kernel/src/heartbeat.rs` | 1-546 |
| Metering/quotas | `crates/openfang-kernel/src/metering.rs` | 1-810 |
| Resource quotas | `crates/openfang-types/src/agent.rs` | 248-282 |
| Autonomous config | `crates/openfang-types/src/agent.rs` | 70-95 |

---

## 5. Patterns Worth Adopting

### High Priority

**1. Session repair as a pre-flight check, not just a recovery mechanism**
OpenFang runs `validate_and_repair()` before EVERY LLM call, not just on resume. This is a defensive programming pattern that catches corruption from any source (compaction bugs, provider quirks, crash recovery). AGH should do the same -- validate event replay output before sending to the ACP agent.

**2. Multi-strategy loop guard with graduated response**
The 4-verdict system (Allow/Warn/Block/CircuitBreak) is more nuanced than a simple iteration counter. Key innovations:
- **Outcome-aware detection**: tracking that identical calls produce identical results is far more useful than just counting call repetitions
- **Ping-pong detection**: A-B-A-B and A-B-C-A-B-C patterns evade simple per-call counting
- **Warning escalation**: warnings upgrade to blocks after `max_warnings_per_call`, preventing agents from ignoring warnings indefinitely
- **Poll tool exemptions**: status-checking tools get relaxed thresholds with backoff suggestions

**3. Interim saves after every tool execution round**
This is critical for crash resilience. If the process dies mid-loop, the session history up to the last completed tool round is preserved. AGH should persist events after each tool execution, not just at loop end.

**4. Synthetic error results for interrupted tool calls**
When a ToolUse has no matching ToolResult (crash/interrupt), OpenFang inserts `[Tool execution was interrupted or lost]` with `is_error: true`. This prevents LLM API validation errors and gives the model a signal that something went wrong. AGH needs this for ACP event replay.

### Medium Priority

**5. 8-category LLM error classification**
The pattern-matching classifier across 19+ providers is reusable. AGH should classify ACP agent errors into retryable vs. non-retryable categories, with provider-specific pattern tables for Claude Code, Codex, Gemini CLI, etc.

**6. Ordered shutdown phases with observability**
The 10-phase `ShutdownCoordinator` with timing logs and WS broadcast is a clean pattern. AGH's daemon shutdown should follow a similar sequence: stop accepting -> drain in-flight sessions -> save state -> close stores -> exit.

**7. Phantom action detection**
Detecting when the LLM claims to have performed an action without tool calls is a clever anti-hallucination guard. AGH could track "claimed completions" vs "actual tool invocations" and flag discrepancies.

**8. Tool result sanitization (injection prevention)**
Stripping prompt injection markers from tool output before feeding back to the LLM is important for security. AGH should sanitize ACP event content, especially from agents that execute shell commands.

### Lower Priority (But Worth Noting)

**9. Context overflow recovery pipeline**
The 4-stage progressive recovery (moderate trim -> aggressive trim -> truncate tool results -> error) is better than a single emergency action. AGH should implement similar staged recovery for sessions approaching context limits.

**10. Provider circuit breaker**
The `ProviderCooldown` with probe requests prevents request storms to failing LLM providers. AGH could use this pattern for ACP agent subprocess management -- if an agent keeps crashing, back off before respawning.

**11. Heartbeat-driven crash recovery with idle agent detection**
The `never_active` grace period (agents that were spawned but never received a message are NOT flagged as unresponsive) prevents false crash-recovery loops. AGH should implement similar logic for session health monitoring.

### What's Missing in OpenFang (Gaps AGH Can Fill)

1. **No explicit `SessionStopReason` enum** -- stop reasons are scattered across error types, hook metadata, and code paths. AGH should have a single, canonical enum.

2. **No cross-agent recursion depth tracking** -- if Agent A calls Agent B calls Agent A, only timeouts prevent infinite recursion. AGH should track delegation depth.

3. **No session-level budget tracking** -- metering is per-agent-per-time-window. There's no "this session has spent $X" limit. AGH should support per-session cost caps.

4. **No structured resume protocol** -- session repair is purely message-level. There's no "last known good state" checkpoint or resumption protocol. AGH can design a proper checkpoint system with the event store.

5. **No user cancellation handling** -- there's no explicit `UserCancelled` stop reason or graceful cancellation of an in-flight agent loop (only SIGTERM-level shutdown). AGH should support per-session cancellation via context.
