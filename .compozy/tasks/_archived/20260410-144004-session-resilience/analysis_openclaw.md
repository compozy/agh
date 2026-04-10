# OpenClaw: Session Resilience Analysis

## 1. Stop Reason Taxonomy

OpenClaw uses a **two-layer** stop reason taxonomy: an internal FailoverReason system for retry/recovery decisions, and an ACP-facing StopReason for protocol-level communication.

### ACP-Level StopReason (Protocol Surface)

Defined via `@agentclientprotocol/sdk`, the StopReason type exposed to ACP clients is intentionally narrow:

| StopReason    | When Emitted | Gateway State |
|---------------|-------------|---------------|
| `end_turn`    | Normal completion, errors (mapped), final state events | `state: "final"` or `state: "error"` |
| `max_tokens`  | Context window exhausted | `stopReason: "max_tokens"` in gateway payload |
| `cancelled`   | User-initiated abort or aborted state | `state: "aborted"` or explicit cancel |

**Key mapping logic** (`src/acp/translator.ts`, lines 948-964):

```typescript
if (state === "final") {
  const rawStopReason = payload.stopReason as string | undefined;
  const stopReason: StopReason = rawStopReason === "max_tokens" ? "max_tokens" : "end_turn";
  await this.finishPrompt(pending.sessionId, pending, stopReason);
}
if (state === "aborted") {
  await this.finishPrompt(pending.sessionId, pending, "cancelled");
}
if (state === "error") {
  // ACP has no explicit "server_error" stop reason. Use "end_turn" so clients
  // do not treat transient backend errors as deliberate refusals.
  void this.finishPrompt(pending.sessionId, pending, "end_turn");
}
```

Design choice: errors are mapped to `end_turn` rather than surfacing a distinct error stop reason. This prevents ACP clients from treating transient backend failures (timeouts, rate limits) as permanent refusals.

### Internal FailoverReason (Retry/Recovery Engine)

Defined in `src/agents/pi-embedded-helpers/types.ts`:

```typescript
export type FailoverReason =
  | "auth"              // 401 - authentication failure
  | "auth_permanent"    // 403 - permanent auth rejection
  | "format"            // 400 - malformed request
  | "rate_limit"        // 429 - provider rate limiting
  | "overloaded"        // 503 - provider overloaded
  | "billing"           // 402 - billing/quota exceeded
  | "timeout"           // 408 - request timeout
  | "model_not_found"   // 404 - model unavailable
  | "session_expired"   // 410 - session no longer exists
  | "unknown";          // Catch-all
```

Each FailoverReason maps to an HTTP status code via `resolveFailoverStatus()` and drives the retry/failover policy engine.

### Assistant-Level stopReason (LLM Response)

The LLM response layer uses its own stop reason set within assistant messages:

| stopReason   | Meaning |
|-------------|---------|
| `stop`       | Normal completion (model chose to stop) |
| `toolUse`    | Model wants to call a tool (agentic loop continues) |
| `error`      | Error during generation |
| `aborted`    | Externally aborted |
| `max_tokens` | Context window hit |

### Session Entry Status

Persisted session status (`src/config/sessions/types.ts`, line 142):

```typescript
status?: "running" | "done" | "failed" | "killed" | "timeout";
```

### Unhandled Stop Reason Recovery

OpenClaw wraps LLM streams to catch unknown/unhandled stop reasons from providers (`src/agents/pi-embedded-runner/run/attempt.stop-reason-recovery.ts`). When a provider returns a stop reason OpenClaw does not recognize, it:
1. Detects the pattern via regex: `/^Unhandled stop reason:\s*(.+)$/i`
2. Patches the assistant message to `stopReason: "error"` with a normalized error message
3. Builds a synthetic error stream to prevent crashes

---

## 2. Session Repair on Resume

OpenClaw performs multiple layers of session repair, from file-level JSONL integrity to transcript-level tool call pairing.

### 2.1 Session File Repair (`session-file-repair.ts`)

On resume, `repairSessionFileIfNeeded()` performs JSONL integrity repair:

1. **Read the session JSONL file** line by line
2. **Parse each line** as JSON; lines that fail parsing are counted as `droppedLines`
3. **Validate session header** -- first entry must have `type: "session"` with a valid `id`
4. If malformed lines exist:
   - **Create a backup** at `{sessionFile}.bak-{pid}-{timestamp}`
   - **Write cleaned file** via atomic rename (write to `.tmp`, then `fs.rename`)
   - Preserve file permissions from original
5. Return a `RepairReport` with `repaired`, `droppedLines`, `backupPath`

This runs before each embedded agent attempt via `repairSessionFileIfNeeded` called from the attempt runner.

### 2.2 Transcript Repair (`session-transcript-repair.ts`)

The transcript repair system handles structural inconsistencies in the message history:

**Tool Call Input Repair** (`repairToolCallInputs`):
- Drops tool call blocks missing `input`, `id`, or valid `name`
- Validates tool names against a max length (64 chars) and regex `/^[A-Za-z0-9_:.-]+$/`
- Optionally filters against an `allowedToolNames` set
- Redacts `sessions_spawn` attachment content to prevent transcript bloat
- If all tool calls in an assistant message are dropped, the entire message is removed

**Tool Use/Result Pairing Repair** (`repairToolUseResultPairing`):
- **Moves displaced toolResult** messages directly after their matching assistant toolCall turn
- **Inserts synthetic error toolResults** for missing IDs (with text: `[openclaw] missing tool result in session history; inserted synthetic error result for transcript repair.`)
- **Drops duplicate toolResults** for the same ID anywhere in the transcript
- **Drops orphan toolResults** that appear outside assistant context
- **Skips synthesis for aborted/errored assistant turns** -- when `stopReason === "error" || "aborted"`, incomplete tool_use blocks are left alone to avoid API 400 errors

### 2.3 Session Tool Result Guard (`session-tool-result-guard.ts`)

A live guard installed on `sessionManager.appendMessage` that:
- **Tracks pending tool calls** -- when an assistant message with tool calls is written, their IDs are registered
- **Matches incoming toolResults** to pending IDs and normalizes tool names
- **Caps tool result size** via `truncateToolResultMessage` with `DEFAULT_MAX_LIVE_TOOL_RESULT_CHARS`
- **Flushes synthetic results** for orphaned pending tool calls when new non-tool-result messages arrive
- **Sanitizes tool call inputs** against an allowlist before persistence
- Supports a `beforeMessageWriteHook` that can block or modify messages before persistence

### 2.4 Session Write Lock (`session-write-lock.ts`)

Prevents concurrent writes to the same session file:
- **File-based locking** via `fs.open(lockPath, "wx")` (exclusive create)
- **Lock payload** includes `pid`, `createdAt`, and process `starttime` (from `/proc/pid/stat`)
- **Stale lock detection**:
  - Dead PID (process no longer alive)
  - Recycled PID (start time mismatch -- detects OS PID reuse)
  - Age exceeds `DEFAULT_STALE_MS` (30 minutes)
  - Orphan self-lock (same PID but no in-memory record)
- **Watchdog timer** runs every 60s, forcibly releasing locks held beyond `maxHoldMs` (default 5 minutes)
- **Signal handlers** (SIGINT, SIGTERM, etc.) release all locks synchronously on process exit
- **Reentrant locking** supported -- same session can re-acquire without deadlock

### 2.5 Gateway Disconnect Recovery

The ACP translator handles gateway disconnects with a grace window (`ACP_GATEWAY_DISCONNECT_GRACE_MS = 5000ms`):

1. On disconnect: start a 5-second grace timer for each pending prompt
2. On reconnect within grace period:
   - Call `agent.wait({ runId, timeoutMs: 0 })` to check if the run completed during disconnect
   - If `status: "ok"` -- resolve with `end_turn`
   - If `status: "timeout"` -- keep pending, schedule another check at the deadline
3. On grace period expiry without reconnect: reject with disconnect error
4. Prompts started during disconnect are queued and reconciled on reconnect

### 2.6 Subagent Orphan Recovery (`subagent-orphan-recovery.ts`)

After a gateway restart (SIGUSR1), OpenClaw recovers orphaned subagent sessions:

1. **Detection**: Scans the subagent run registry for active runs where the session store has `abortedLastRun: true`
2. **Resume message construction**: Builds a synthetic system message:
   ```
   [System] Your previous turn was interrupted by a gateway reload.
   Your original task was: {task}
   The last message from the user before the interruption was: {lastHumanMessage}
   Please continue where you left off.
   ```
3. **Config change detection**: Scans transcript for config-related mentions to add a hint preventing duplicate config modifications
4. **Idempotent recovery**: Tracks `resumedSessionKeys` to prevent duplicate resumptions
5. **Retry with exponential backoff**: Up to 3 retries with 2x backoff (starting at 5s delay)
6. **Flag persistence**: `abortedLastRun` flag is only cleared after confirmed successful resume

---

## 3. Loop/Recursion Guards

### 3.1 Tool Loop Detection (`tool-loop-detection.ts`)

A sophisticated multi-detector system with configurable thresholds:

**Configuration** (disabled by default):
```json
{
  "enabled": false,
  "historySize": 30,
  "warningThreshold": 10,
  "criticalThreshold": 20,
  "globalCircuitBreakerThreshold": 30,
  "detectors": {
    "genericRepeat": true,
    "knownPollNoProgress": true,
    "pingPong": true
  }
}
```

**Four Detector Types** (`LoopDetectorKind`):

| Detector | What It Detects | Warning At | Critical At |
|----------|----------------|------------|-------------|
| `generic_repeat` | Same tool + same args repeated | 10 calls | Never (warn only) |
| `known_poll_no_progress` | Polling tools (`command_status`, `process:poll/log`) with identical results | 10 calls | 20 calls |
| `global_circuit_breaker` | Any tool with identical no-progress outcomes | N/A | 30 calls |
| `ping_pong` | Alternating A-B-A-B tool call patterns | 10 alternations | 20 alternations (with no-progress evidence) |

**How it works**:

1. **Recording**: Each tool call is hashed (`toolName:sha256(stableStringify(params))`) and stored in a sliding window of the last 30 calls
2. **Outcome tracking**: After each tool call completes, the result is hashed and stored alongside the call record for no-progress detection
3. **No-progress detection**: A "no-progress streak" counts consecutive identical outcomes for the same tool+args combination
4. **Ping-pong detection**: Checks if the tail of the history alternates between exactly two distinct tool signatures, with optional no-progress evidence on both sides
5. **Two severity levels**:
   - `warning`: Injected as a system message telling the agent to stop retrying
   - `critical`: Blocks session execution entirely

**Warning key deduplication**: Each detection result includes a `warningKey` to prevent duplicate warnings for the same pattern.

**Known poll tool identification**: `command_status` and `process:poll/log` are recognized as polling tools with specialized no-progress detection that considers structural result fields (`status`, `exitCode`, `totalLines`, etc.) rather than raw text.

### 3.2 Run Loop Iteration Guard

The main agent run loop (`run.ts`) has a hard iteration cap:

```typescript
const BASE_RUN_RETRY_ITERATIONS = 24;
const RUN_RETRY_ITERATIONS_PER_PROFILE = 8;
const MIN_RUN_RETRY_ITERATIONS = 32;
const MAX_RUN_RETRY_ITERATIONS = 160;

function resolveMaxRunRetryIterations(profileCandidateCount: number): number {
  const scaled = BASE_RUN_RETRY_ITERATIONS +
    Math.max(1, profileCandidateCount) * RUN_RETRY_ITERATIONS_PER_PROFILE;
  return Math.min(MAX_RUN_RETRY_ITERATIONS, Math.max(MIN_RUN_RETRY_ITERATIONS, scaled));
}
```

When exceeded:
- Logs `[run-retry-limit]` with session key, provider, and attempt count
- Evaluates failover policy: either escalates to fallback model or returns error payload
- Error message: "Request failed after repeated internal retries. Please try again, or use /new to start a fresh session."

### 3.3 Subagent Announce Loop Guard

Prevents infinite retry loops when announcing subagent completion (issue #18264):

- **Announce retry count**: Each `SubagentRunRecord` tracks `announceRetryCount` and `lastAnnounceRetryAt`
- **Max retry budget**: Entries over the retry budget are marked completed without announcing
- **Expiry check**: Entries that ended more than 5 minutes ago with high retry counts are skipped
- **Rejection handling**: When `runSubagentAnnounceFlow` rejects, `cleanupHandled` is reset to allow future retries, but the retry counter increments

### 3.4 Context Overflow / Compaction Guards

Multiple compaction guards prevent infinite compaction loops:

```typescript
const MAX_TIMEOUT_COMPACTION_ATTEMPTS = 2;
const MAX_OVERFLOW_COMPACTION_ATTEMPTS = 3;
```

- **Timeout compaction**: Max 2 attempts to compact after timeout errors
- **Overflow compaction**: Max 3 attempts to compact after context overflow
- **Preemptive compaction**: Before prompting, checks if context is near overflow and compacts proactively
- **Tool result truncation**: Before retrying after overflow, truncates oversized tool results in the session

### 3.5 Auth Profile Rotation Guards

Rate limit and overload profile rotation have caps:

- `overloadProfileRotationLimit`: Configurable cap on profile rotations for overloaded providers
- `rateLimitProfileRotationLimit`: Cap before escalating to model fallback
- Exponential backoff before overload failover via `overloadFailoverBackoffMs`

### 3.6 Planning-Only Retry Guard

Detects when the LLM only describes a plan without executing (`incomplete-turn.ts`):

- Regex-based detection: `/\b(?:i(?:'ll| will)|let me|going to|...)\b/i`
- Completion detection to avoid false positives: `/\b(?:done|finished|implemented|...)\b/i`
- Max 700 chars, no code blocks, specific to OpenAI GPT-5 models
- Injects instruction: "Do not restate the plan. Act now: take the first concrete tool action you can."

### 3.7 Fetch Recursion Guard

`src/infra/net/fetch-guard.ts` prevents recursive fetch loops in network operations with depth tracking.

---

## 4. Key Code References

| Component | File | Key Lines/Exports |
|-----------|------|-------------------|
| ACP StopReason mapping | `src/acp/translator.ts` | Lines 948-964 (state-to-stopReason mapping) |
| ACP disconnect grace | `src/acp/translator.ts` | `ACP_GATEWAY_DISCONNECT_GRACE_MS = 5_000` |
| FailoverReason enum | `src/agents/pi-embedded-helpers/types.ts` | Full type definition |
| FailoverError class | `src/agents/failover-error.ts` | `FailoverError`, `resolveFailoverStatus()` |
| Tool loop detection | `src/agents/tool-loop-detection.ts` | `detectToolCallLoop()`, `recordToolCall()`, `recordToolCallOutcome()` |
| Loop detection config | `src/config/types.tools.ts` | `ToolLoopDetectionConfig` type |
| Loop detection docs | `docs/tools/loop-detection.md` | User-facing configuration guide |
| Session file repair | `src/agents/session-file-repair.ts` | `repairSessionFileIfNeeded()` |
| Transcript repair | `src/agents/session-transcript-repair.ts` | `repairToolCallInputs()`, `repairToolUseResultPairing()` |
| Tool result guard | `src/agents/session-tool-result-guard.ts` | `installSessionToolResultGuard()` |
| Session write lock | `src/agents/session-write-lock.ts` | `acquireSessionWriteLock()`, stale lock detection |
| Orphan recovery | `src/agents/subagent-orphan-recovery.ts` | `recoverOrphanedSubagentSessions()`, `scheduleOrphanRecovery()` |
| Run loop guard | `src/agents/pi-embedded-runner/run/helpers.ts` | `resolveMaxRunRetryIterations()`, `MAX_RUN_RETRY_ITERATIONS = 160` |
| Retry limit handler | `src/agents/pi-embedded-runner/run/retry-limit.ts` | `handleRetryLimitExhaustion()` |
| Failover policy | `src/agents/pi-embedded-runner/run/failover-policy.ts` | `resolveRunFailoverDecision()` |
| Stop reason recovery | `src/agents/pi-embedded-runner/run/attempt.stop-reason-recovery.ts` | `wrapStreamFnHandleSensitiveStopReason()` |
| Incomplete turn guard | `src/agents/pi-embedded-runner/run/incomplete-turn.ts` | `resolvePlanningOnlyRetryInstruction()` |
| Announce loop guard | `src/agents/subagent-registry.announce-loop-guard.test.ts` | Regression test for issue #18264 |
| Session entry type | `src/config/sessions/types.ts` | `SessionEntry` with `abortedLastRun`, `status` fields |
| Before-tool-call hook | `src/agents/pi-tools.before-tool-call.ts` | `runBeforeToolCallHook()`, `wrapToolWithBeforeToolCallHook()` |

---

## 5. Patterns Worth Adopting

### 5.1 Two-Layer Stop Reason Architecture
OpenClaw separates **protocol-facing** stop reasons (simple: `end_turn`, `max_tokens`, `cancelled`) from **internal** failover reasons (rich: `auth`, `rate_limit`, `overloaded`, `billing`, `timeout`, etc.). AGH should adopt this: keep the ACP/external surface simple while using a richer internal taxonomy for retry/recovery decisions. The mapping from internal to external is where policy lives.

### 5.2 Tool Loop Detection as a Configurable Subsystem
The multi-detector approach with configurable thresholds per agent is excellent. Key ideas for AGH:
- **Sliding window history** (30 calls) with content hashing for pattern detection
- **No-progress detection** via result outcome hashing, not just call counting
- **Ping-pong detection** for alternating A-B-A-B patterns
- **Two severity levels** (warning = inject hint, critical = block execution)
- **Disabled by default** with per-agent opt-in

### 5.3 Atomic Session File Repair on Resume
The backup-then-atomic-rename pattern for JSONL repair is safe and production-proven:
1. Read the file, drop unparseable lines
2. Write backup with original content
3. Write cleaned content to `.tmp` file
4. Atomic `rename()` to replace original
AGH should adopt this for SQLite WAL recovery and session event store integrity.

### 5.4 Synthetic Tool Result Injection
When tool calls are found without matching results (crash mid-execution), OpenClaw synthesizes error results. This prevents strict providers from rejecting the entire transcript. AGH's event store should support synthetic event injection for the same reason.

### 5.5 Session Write Lock with PID Recycling Detection
The lock system's use of `/proc/pid/stat` start time to detect PID recycling is clever and prevents stale lock false-positives. AGH should use a similar approach for its SQLite-based session locking, especially since the daemon runs as a long-lived process.

### 5.6 Subagent Orphan Recovery with Idempotent Resume
The `abortedLastRun` flag pattern is elegant:
- Flag is set when a run is interrupted
- Flag is only cleared after confirmed successful resume
- If resume fails, the flag persists for the next restart attempt
- `resumedSessionKeys` set prevents duplicate resumptions within a single recovery cycle
AGH should adopt this for its subprocess agent sessions.

### 5.7 Run Loop Hard Cap with Failover Escalation
The escalating retry strategy (rotate auth profile -> fallback model -> error payload) with a hard iteration cap prevents infinite retry loops while maximizing recovery chances. AGH should implement a similar escalation ladder for its session retry logic.

### 5.8 Grace Window for Transient Disconnects
The 5-second grace window for gateway disconnects, with `agent.wait` reconciliation on reconnect, prevents unnecessary session failures during transient network issues. AGH should adopt a similar pattern for its HTTP/SSE and UDS connections.

### 5.9 Before-Tool-Call Hook for Loop Detection Integration
OpenClaw integrates loop detection into the tool execution pipeline via a `before_tool_call` hook that wraps each tool. This is cleaner than checking at the orchestration level because it catches all tool calls regardless of how they were initiated. AGH's hooks system could provide a similar injection point.

### 5.10 Unhandled Stop Reason Recovery
Wrapping LLM streams to catch and normalize unknown stop reasons prevents crashes from provider-specific behaviors. AGH's ACP client should implement similar defensive normalization when parsing agent subprocess responses.
