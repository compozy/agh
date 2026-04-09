# GoClaw: Session Resilience Analysis

## 1. Stop Reason Taxonomy

GoClaw has an **implicit** stop reason taxonomy -- there is no single `StopReason` enum. Instead, termination reasons are encoded across multiple layers: the LLM provider's `FinishReason`, the agent loop's internal exit paths, and the scheduler/event system's lifecycle events.

### 1.1 Provider-Level FinishReason

The `ChatResponse.FinishReason` field (`internal/providers/types.go:77`) carries the raw LLM stop signal:

| Value | Meaning |
|-------|---------|
| `"stop"` | Model completed naturally (no more tool calls, produced text) |
| `"tool_calls"` | Model wants to call tools (loop continues) |
| `"length"` | Output truncated -- hit `max_tokens` ceiling |

GoClaw does **not** passthrough the provider FinishReason to the caller. It is consumed internally by the loop to decide next steps.

### 1.2 Agent Loop Exit Paths

The `runLoop()` function in `internal/agent/loop.go` has these distinct exit paths:

| Exit Path | Trigger | Result |
|-----------|---------|--------|
| **Natural completion** | `len(resp.ToolCalls) == 0` -- model produces text without requesting tools | Normal `RunResult` with content |
| **Max iterations** | `rs.iteration >= maxIter` (default 30, configurable per-agent and per-request) | Loop ends; last iteration strips all tools and forces text-only response |
| **Budget exceeded (monthly)** | `spentCents >= l.budgetMonthlyCents` (pre-loop check) | Returns error: `"monthly budget exceeded ($X / $Y)"` |
| **Tool budget exceeded** | `rs.totalToolCalls > l.maxToolCalls` | Injects system message forcing summarization, then one more iteration |
| **LLM call error** | Provider returns error (API failure, rate limit, auth) | Returns error: `"LLM call failed (iteration N): <err>"` |
| **Loop detector kill** | Same-args loop, same-result loop, or read-only streak hits critical threshold | `RunResult.LoopKilled = true`, content set to explanation |
| **Truncation retry limit** | `maxTruncationRetries = 3` consecutive truncated outputs | Loop breaks, returns truncation fallback message |
| **Context cancellation** | `ctx.Done()` fires (user `/stop` command or parent cancellation) | Returns `ctx.Err()` -- typically `context.Canceled` |
| **Panic recovery** | `recover()` in deferred handler catches panics | Returns error: `"agent loop panic: <value>"` |

### 1.3 Run-Level Event Classification

The `Run()` method in `internal/agent/loop_run.go` maps loop outcomes to four event types:

| Event | Condition | Protocol Constant |
|-------|-----------|-------------------|
| `run.completed` | `err == nil` | `AgentEventRunCompleted` |
| `run.failed` | `err != nil && ctx.Err() == nil` | `AgentEventRunFailed` |
| `run.cancelled` | `err != nil && ctx.Err() != nil` (user cancel) | `AgentEventRunCancelled` |
| `run.retrying` | LLM call being retried (transient provider error) | `AgentEventRunRetrying` |

### 1.4 Trace Status Taxonomy

Traces (observability layer in `internal/store/tracing_store.go`) use four terminal states:

```go
TraceStatusRunning   = "running"
TraceStatusCompleted = "completed"
TraceStatusError     = "error"
TraceStatusCancelled = "cancelled"
```

The mapping: `run.completed` -> `completed`, `run.failed` -> `error`, `run.cancelled` -> `cancelled`.

### 1.5 Team Task Outcome Mapping

In `cmd/gateway_consumer_post_turn.go`, the `resolveTeamTaskOutcome` function maps run results to task lifecycle states:

| Condition | Task Action |
|-----------|-------------|
| `outcome.Err != nil` | `FailTask` -- agent errored |
| `flags.Completed \|\| flags.Escalated` | Skip -- tool already handled lifecycle |
| `flags.Reviewed` | Renew lock -- task under review |
| `outcome.Result.LoopKilled` | `FailTask` with reason `"loop_detector_kill"` |
| Default (normal completion) | `CompleteTask` with deliverables |


## 2. Session Repair on Resume

GoClaw's session model is **stateless-resume** -- there is no explicit "resume after crash" mechanism with consistency checks. Instead, it relies on layered persistence and startup recovery.

### 2.1 Session Storage Architecture

Sessions are persisted in two ways:

1. **JSON files on disk** (`internal/sessions/manager.go`): Each session is atomically written via temp-file-then-rename (`tmpFile -> Sync -> Rename`). On startup, `loadAll()` reads all `.json` files from the storage directory and populates the in-memory map. There is no schema validation, version check, or integrity verification -- JSON unmarshal errors are silently skipped.

2. **SQLite database** (`internal/store/sqlitestore/`): For the gateway, sessions are persisted in SQLite with a full schema including `spawn_depth`, `agent_id`, `user_id`, `metadata`, `team_id`, and `tenant_id`.

### 2.2 Crash Safety: Periodic Checkpoint

The loop implements periodic checkpointing to limit data loss on crash (`loop.go:750-762`):

```go
const checkpointInterval = 5
if rs.iteration > 0 && rs.iteration%checkpointInterval == 0 && len(rs.pendingMsgs) > 0 {
    for _, msg := range rs.pendingMsgs {
        l.sessions.AddMessage(ctx, req.SessionKey, msg)
    }
    rs.checkpointFlushedMsgs += len(rs.pendingMsgs)
    rs.pendingMsgs = rs.pendingMsgs[:0]
    l.sessions.Save(ctx, req.SessionKey) // best-effort persistence
}
```

This means: on a crash between checkpoints, up to 5 iterations of messages are lost. The comment explicitly says: "Trade-off: partial visibility to concurrent reads vs full data loss on crash."

### 2.3 Stale Trace Recovery on Startup

The tracing collector (`internal/tracing/collector.go:228-246`) performs orphan trace cleanup on startup:

```go
func (c *Collector) recoverStaleTraces() {
    const staleThreshold = 30 * time.Minute
    cutoff := time.Now().UTC().Add(-staleThreshold)
    recovered, err := c.store.RecoverStaleRunningTraces(ctx, cutoff)
}
```

Any trace stuck in `"running"` status from before the crash (older than 30 minutes) is marked as `"error"`. This prevents the UI from showing perpetually-running ghosts.

### 2.4 Safety Net Trace Finalization

In `loop_run.go:122-138`, a deferred function ensures root traces are always finalized, even on panic or goroutine leak:

```go
defer func() {
    if traceFinalized { return }
    slog.Warn("tracing: safety-net finalizing orphan trace", ...)
    l.traceCollector.FinishTrace(safeCtx, traceID, store.TraceStatusError,
        "trace finalized by safety net (likely panic or goroutine leak)", "")
}()
```

### 2.5 Session Resume Behavior

When a session is "resumed" (user sends a new message to an existing session key), GoClaw simply:

1. Loads the full message history from the session store (`GetHistory`)
2. Loads the summary if one exists (`GetSummary`)
3. Rebuilds the LLM messages from scratch via `buildMessages()`
4. Runs a new loop iteration

There are **no consistency checks** such as:
- Verifying the last message is properly terminated (no half-written assistant turns)
- Detecting orphaned tool calls (tool_use without matching tool_result)
- Repairing incomplete tool execution sequences
- Validating message role alternation

The system relies on the LLM being robust enough to handle inconsistent history. If a crash left a session with an assistant message containing tool_calls but no corresponding tool results, the next run would simply append the new user message and let the LLM sort it out.

### 2.6 History Compaction Safety Net

The `channels/history_compaction.go` notes a "safety net for post-restart scenarios" -- after restart, the in-memory count may be stale, so it re-checks the DB for the real count before deciding whether compaction is needed.

### 2.7 What GoClaw Does NOT Do on Resume

- No WAL-style intent logging before operations
- No session state machine (sessions have no status field -- they're just message arrays)
- No "last run outcome" tracking on the session
- No explicit dirty/clean session markers
- No process lock per session to detect unclean shutdown
- No message sequence numbers or gap detection


## 3. Loop/Recursion Guards

GoClaw has the most sophisticated loop detection system of any agent harness I've analyzed. It operates at three layers.

### 3.1 Layer 1: Same-Args Loop Detection (toolLoopState)

**File**: `internal/agent/toolloop.go`

Tracks the last 30 tool calls (`toolLoopHistorySize = 30`) in a sliding window. Each entry records:
- Tool name
- SHA-256 hash of `toolName + stableJSON(args)` (deterministic key ordering)
- SHA-256 hash of the tool result content

Detection logic (`detect()`): counts records where both argsHash AND resultHash match. Only flags **true no-progress loops** -- same input producing same output.

| Threshold | Action |
|-----------|--------|
| 3 identical calls (`toolLoopWarningThreshold`) | Inject warning message into conversation: "Try a completely different approach..." |
| 5 identical calls (`toolLoopCriticalThreshold`) | Force-stop loop, set `rs.loopKilled = true`, return explanation to user |

### 3.2 Layer 2: Same-Result Detection

**File**: `internal/agent/toolloop.go:246-269`

Catches a more subtle loop: the agent varies arguments slightly but gets identical results back each time.

| Threshold | Action |
|-----------|--------|
| 4 same-result calls (`sameResultWarning`) | Warning: "The information is already in your context. Stop re-reading..." |
| 6 same-result calls (`sameResultCritical`) | Force-stop with `loopKilled = true` |

### 3.3 Layer 3: Read-Only Streak Detection (Uniqueness-Aware)

**File**: `internal/agent/toolloop.go:199-241`

Detects agents stuck in read-only mode (reading files without ever writing). Uses a **uniqueness ratio** to distinguish legitimate exploration from stuck loops:

**Tool classification:**
- **Mutating** (resets streak): `write_file`, `edit`, `edit_file`, `spawn`, `message`, `create_image/video/audio`, `tts`, `cron`, `publish_skill`, `sessions_send`
- **Neutral** (no effect on streak): `exec`, `bash`, `mcp_*` prefixed tools
- **Read-only** (increments streak): everything else (`read_file`, `list_files`, etc.)
- **team_tasks**: classified by action -- `list/get/search` are read-only, `progress` is neutral, `create/complete/cancel/comment` are mutating

**Uniqueness ratio** = `readOnlyUnique / readOnlyStreak`

| Mode | Ratio | Warning | Critical |
|------|-------|---------|----------|
| Stuck (re-reading same files) | <= 0.6 | 8 consecutive reads | 12 consecutive reads |
| Exploration (unique files) | > 0.6 | 24 consecutive reads | 36 consecutive reads |

This was specifically designed to fix issue #506 where an agent exploring a monorepo with 11+ unique file reads was falsely killed.

### 3.4 Iteration Budget Guards

**File**: `internal/agent/loop.go`, `internal/config/defaults.go`

| Guard | Default | Configuration |
|-------|---------|---------------|
| `maxIterations` | 30 | Per-agent via DB, per-request via `RunRequest.MaxIterations` (must be lower than agent default) |
| `maxToolCalls` | 0 (unlimited) | Per-agent via `Loop.maxToolCalls` |
| Final iteration tool strip | At `iteration == maxIter` | Removes all tool definitions, injects "[System] Final iteration reached" |
| 75% budget nudge | At `iteration == maxIter*3/4` when no text response yet | Warns: "Start summarizing your findings" |
| Skill evolution nudges | At 70% and 90% of iteration budget | Budget pressure reminders |

### 3.5 Truncation Retry Guard

**File**: `internal/agent/loop.go:417-449`

When the model's output is truncated (`FinishReason == "length"`) or tool call arguments are malformed:

```go
const maxTruncationRetries = 3
```

After 3 consecutive truncation retries, the loop gives up rather than burning all iterations. Sets a fallback content message.

### 3.6 Subagent Spawn Depth Limit

**File**: `internal/tools/subagent_config.go`, `internal/tools/subagent_spawn.go`

| Guard | Default | Max |
|-------|---------|-----|
| `MaxSpawnDepth` | 1 | 5 (configurable, capped by edition) |
| `MaxConcurrent` subagents | 8 | Configurable |
| `MaxChildrenPerAgent` | 5 | Configurable |
| `MaxRetries` per subagent | 2 | Configurable |

At max depth, leaf agents have tools removed (`SubagentDenyLeaf`): they cannot spawn further subagents. The system prompt explicitly tells them: "You are a leaf worker and CANNOT spawn further sub-agents."

### 3.7 Team Task Circuit Breaker

From `docs/11-agent-teams.md`:
- Tasks auto-fail after 3 dispatch attempts (`maxTaskDispatches`) -- prevents infinite loops when agents can't complete a task
- Lead self-dispatch guard: tasks assigned to the lead agent are auto-failed (prevents dual-session loop)
- Loop detector kills propagate to task failure with reason `"loop_detector_kill"`

### 3.8 Input Guards

**File**: `internal/agent/input_guard.go`

Prevents injection attacks that could cause runaway behavior:
- Scans for: `ignore_instructions`, `role_override`, `system_tags`, `instruction_injection`, `null_bytes`, `delimiter_escape`
- Actions: `"log"`, `"warn"` (default), `"block"` (rejects message), `"off"`
- Message size limit: `DefaultMaxMessageChars = 32000` -- oversized messages are truncated with a system notice

### 3.9 Context Window Management

**File**: `internal/agent/pruning.go`, `internal/agent/loop_compact.go`

Two-phase approach when context exceeds budget:
1. **Phase 1: Prune old tool results** at 70% of history budget -- soft trim (keep head+tail), then hard clear (replace with placeholder)
2. **Phase 2: Mid-loop compaction** -- LLM-based summarization of first ~70% of messages, keeping last ~30% intact

Per-result guard: any single tool result exceeding 30% of context window is force-trimmed regardless of overall ratio.

### 3.10 Panic Recovery

**Files**: `internal/agent/loop.go:36-44`, `internal/safego/recover.go`

The main `runLoop()` has a top-level `defer recover()` that catches panics and converts them to errors. Parallel tool execution goroutines each have their own `defer safego.Recover()` that converts panics to error results rather than crashing the loop.


## 4. Key Code References

| File | Lines | What |
|------|-------|------|
| `internal/agent/loop.go` | 35-45 | Panic recovery in runLoop |
| `internal/agent/loop.go` | 138-141 | maxIterations override logic |
| `internal/agent/loop.go` | 144-153 | Monthly budget pre-check |
| `internal/agent/loop.go` | 156-764 | Main iteration loop with all exit paths |
| `internal/agent/loop.go` | 417-449 | Truncation retry guard |
| `internal/agent/loop.go` | 513-524 | Tool budget exceeded handler |
| `internal/agent/loop.go` | 750-762 | Periodic checkpoint flush (every 5 iterations) |
| `internal/agent/loop_run.go` | 18-245 | Run() with event emission and trace lifecycle |
| `internal/agent/loop_run.go` | 122-138 | Safety-net trace finalization deferred |
| `internal/agent/loop_run.go` | 185-214 | Error vs cancel classification for events/traces |
| `internal/agent/toolloop.go` | 12-33 | All detection threshold constants |
| `internal/agent/toolloop.go` | 59-142 | toolLoopState: record + detect (same-args) |
| `internal/agent/toolloop.go` | 148-197 | recordMutation + read-only streak tracking |
| `internal/agent/toolloop.go` | 205-241 | detectReadOnlyStreak with uniqueness ratio |
| `internal/agent/toolloop.go` | 246-269 | detectSameResult (cross-args same output) |
| `internal/agent/loop_tools.go` | 15-147 | processToolResult with loop detection integration |
| `internal/agent/loop_tools.go` | 149-169 | checkReadOnlyStreak with kill flag |
| `internal/agent/loop_types.go` | 488-498 | RunResult including LoopKilled flag |
| `internal/agent/loop_types.go` | 510-557 | runState with all mutable loop state |
| `internal/agent/loop_finalize.go` | 36-209 | finalizeRun: sanitize, flush, build result |
| `internal/agent/loop_tool_filter.go` | 86-93 | Final iteration: strip tools, force text |
| `internal/agent/pruning.go` | 101-269 | Two-pass context pruning (soft trim + hard clear) |
| `internal/agent/loop_compact.go` | 44-118 | Mid-loop LLM compaction |
| `internal/agent/input_guard.go` | 1-99 | Prompt injection detection |
| `internal/config/defaults.go` | 1-13 | All default constants (30 iterations, 200K context, 8192 max tokens) |
| `internal/tools/subagent_config.go` | 7-15 | Subagent defaults (depth 1, max 8 concurrent, 5 children) |
| `internal/tools/subagent_spawn.go` | 38-41 | Spawn depth enforcement |
| `internal/providers/types.go` | 77 | FinishReason field on ChatResponse |
| `internal/store/tracing_store.go` | 13-16 | Trace status constants |
| `internal/tracing/collector.go` | 104 | Startup stale trace recovery |
| `internal/tracing/collector.go` | 228-246 | recoverStaleTraces implementation |
| `internal/sessions/manager.go` | 396-477 | Atomic session persistence (temp+rename) |
| `internal/safego/recover.go` | 1-28 | Panic recovery helper for goroutines |
| `cmd/gateway_consumer_post_turn.go` | 59-211 | Team task outcome mapping (including LoopKilled -> auto-fail) |
| `pkg/protocol/events.go` | 112-121 | Agent event type constants |


## 5. Patterns Worth Adopting

### 5.1 Must Adopt

1. **Multi-layer loop detection**: GoClaw's three-layer approach (same-args, same-result, read-only streak) catches loops that single-metric detectors miss. The uniqueness ratio for distinguishing exploration from stuck loops is particularly clever. AGH should implement all three layers.

2. **LoopKilled propagation**: The `RunResult.LoopKilled` flag flows from detector to consumer to task lifecycle. This clear signal path lets higher-level orchestrators make correct decisions (auto-fail tasks, don't announce results). AGH needs this for hook/session state machines.

3. **Periodic checkpoint flush**: Every 5 iterations, flush pending messages to durable storage. Simple, effective crash safety without full WAL complexity. The explicit trade-off comment is a good pattern: acknowledge what you lose, document why it's acceptable.

4. **Safety-net trace/span finalization**: Deferred cleanup functions that catch orphaned running traces after panics/goroutine leaks. AGH's observe system needs the same for recording events.

5. **Stale state recovery on startup**: On boot, scan for "running" records older than a threshold and mark them as errors. Essential for any system that persists in-flight state.

6. **Truncation retry cap**: Limiting retries when the LLM can't fit output into max_tokens prevents burning the entire iteration budget on a hopeless situation.

### 5.2 Should Adopt

7. **Budget pressure nudges**: At 70% and 90% of iteration budget, inject "start wrapping up" messages. At 75%, warn if no text response yet. These prevent the common failure mode of agents spending all iterations on tools without producing a response.

8. **Final iteration tool stripping**: On the last iteration, remove all tool definitions and inject "[System] Final iteration reached. Summarize and respond." This guarantees the model produces a text response instead of requesting more tools.

9. **Tool classification for streak detection**: Categorizing tools as mutating/neutral/read-only enables nuanced detection. AGH should maintain a similar classification, especially distinguishing ambiguous tools like `exec`.

10. **Per-result context guard**: Force-trim any single tool result exceeding 30% of context window, independently of overall context pressure. Catches outlier outputs that would otherwise crowd out everything else.

### 5.3 Consider Adopting

11. **Spawn depth enforcement via tool stripping**: At max depth, remove spawn-related tools entirely rather than relying on the LLM to obey instructions. Belt-and-suspenders approach.

12. **Adaptive tool slow-timer**: Track tool execution times, compute adaptive thresholds (2x historical max), emit "tool_slow" events when exceeded. Useful for observability.

13. **Input guard for injection detection**: Regex-based scanning for common prompt injection patterns. Low cost, catches obvious attacks. Configurable action levels (log/warn/block/off).

### 5.4 Should NOT Adopt

14. **No session state machine**: GoClaw sessions have no status field -- they're just message arrays. This works for GoClaw's gateway model but AGH explicitly needs session lifecycle states for the ACP protocol. AGH should have a proper state machine.

15. **Silent JSON unmarshal skip on load**: GoClaw silently skips corrupt session files. AGH should at minimum log warnings and consider quarantining corrupt data.

16. **No message sequence validation on resume**: GoClaw trusts the LLM to handle inconsistent history. AGH should validate message integrity (proper role alternation, no orphaned tool calls) because ACP agents are more sensitive to protocol violations than web LLM APIs.
