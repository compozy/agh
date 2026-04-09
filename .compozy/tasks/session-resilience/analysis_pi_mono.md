# Pi-Mono: Session Resilience Analysis

Pi-Mono (authored by Mario Zechner, aka "badlogic") is a TypeScript monorepo containing `pi-agent-core` (generic agent framework), `pi-ai` (LLM streaming library), and `pi-coding-agent` (the coding agent CLI). This analysis covers session resilience across all three layers.

---

## 1. Stop Reason Taxonomy

### Core StopReason Type (pi-ai layer)

Defined in `packages/ai/src/types.ts:182`:

```typescript
export type StopReason = "stop" | "length" | "toolUse" | "error" | "aborted";
```

| StopReason | Meaning | When Produced |
|---|---|---|
| `"stop"` | Natural completion -- the model finished its response | LLM returns a normal stop signal |
| `"length"` | Max tokens reached -- output was truncated | Model hit `maxTokens` limit |
| `"toolUse"` | Tool call requested -- assistant wants to invoke a tool | Model emits one or more `toolCall` content blocks |
| `"error"` | Runtime/API error -- request failed | Network errors, rate limits, overloaded servers, context overflow, auth failures |
| `"aborted"` | User/system cancellation -- the stream was aborted | `AbortSignal` triggered (Ctrl+C, abort button, programmatic cancel) |

### Stream Event Protocol (pi-ai layer)

The stream event discriminator in `packages/ai/src/types.ts:237-249` splits terminal events into two categories:

```typescript
| { type: "done"; reason: Extract<StopReason, "stop" | "length" | "toolUse">; message: AssistantMessage }
| { type: "error"; reason: Extract<StopReason, "aborted" | "error">; error: AssistantMessage }
```

This means the stream protocol already separates "success with variants" from "failure" at the type level. A `done` event with `reason: "length"` is still considered a success (truncated but usable). An `error` event always carries an `AssistantMessage` with `errorMessage` populated.

### Agent-Level Stop Semantics (pi-agent-core layer)

In `packages/agent/src/agent-loop.ts:194`, the agent loop checks stop reasons to decide whether to continue:

```typescript
if (message.stopReason === "error" || message.stopReason === "aborted") {
    await emit({ type: "turn_end", message, toolResults: [] });
    await emit({ type: "agent_end", messages: newMessages });
    return; // Terminate the loop
}
```

The loop only continues if `stopReason` is `"stop"`, `"length"`, or `"toolUse"`. For `"toolUse"`, it enters tool execution. For `"stop"` or `"length"` without tool calls, it checks for steering/follow-up messages before exiting.

When the agent loop throws an unhandled error (not from the stream), the `Agent` class synthesizes a failure message in `packages/agent/src/agent.ts:459-474`:

```typescript
const failureMessage = {
    role: "assistant",
    stopReason: aborted ? "aborted" : "error",
    errorMessage: error instanceof Error ? error.message : String(error),
    // ...
};
```

### Session-Level Error Classification (pi-coding-agent layer)

`AgentSession` in `packages/coding-agent/src/core/agent-session.ts` adds a higher-level classification on top of stop reasons:

1. **Retryable errors** (`_isRetryableError`, line 2381): Errors matching patterns like `overloaded`, `rate_limit`, `429`, `500-504`, `timeout`, `connection_error`, etc. are automatically retried.

2. **Context overflow errors** (`isContextOverflow` in `packages/ai/src/utils/overflow.ts`): A dedicated subsystem detects context window exceeded errors across 18+ provider-specific patterns. These are NOT retried -- instead they trigger automatic compaction.

3. **Non-retryable errors**: All other errors (auth failures, malformed requests, etc.) are surfaced to the user.

4. **User cancellation** (`"aborted"`): Skipped by both retry and compaction logic.

### Print Mode Exit Codes

In `packages/coding-agent/src/modes/print-mode.ts:111-123`, the exit code is derived from the stop reason:

```typescript
if (assistantMsg.stopReason === "error" || assistantMsg.stopReason === "aborted") {
    exitCode = 1;
} else {
    // Success: exitCode = 0
}
```

### What Pi-Mono Does NOT Have

- **No "budget exceeded" stop reason**: There is no cost/budget limit enforcement. The system tracks `Usage.cost` per message but never enforces a cap.
- **No "max iterations" stop reason**: The agent loop has no iteration counter or max-turns limit in production code (only in tests).
- **No "loop detected" stop reason**: There is no cycle detection or loop guard at the framework level.
- **No "completed" vs "paused" distinction**: A `"stop"` reason means the model finished, but there is no semantic encoding of "task complete" vs "gave up" vs "waiting for input".

---

## 2. Session Repair on Resume

### Session Storage Format

Sessions are stored as JSONL files (one JSON object per line). Each file begins with a `SessionHeader`:

```typescript
interface SessionHeader {
    type: "session";
    version?: number;  // Currently version 3
    id: string;
    timestamp: string;
    cwd: string;
    parentSession?: string;
}
```

Every subsequent line is a `SessionEntry` with `id` and `parentId` forming a tree structure (not a flat list). The `leafId` pointer tracks the current position in the tree.

### Resume Flow: `setSessionFile()`

When resuming a session (`SessionManager.open()` or `SessionManager.continueRecent()`), the key repair logic is in `packages/coding-agent/src/core/session-manager.ts:695-723`:

```typescript
setSessionFile(sessionFile: string): void {
    this.sessionFile = resolve(sessionFile);
    if (existsSync(this.sessionFile)) {
        this.fileEntries = loadEntriesFromFile(this.sessionFile);

        // REPAIR: If file was empty or corrupted (no valid header),
        // truncate and start fresh
        if (this.fileEntries.length === 0) {
            const explicitPath = this.sessionFile;
            this.newSession();
            this.sessionFile = explicitPath;
            this._rewriteFile();
            this.flushed = true;
            return;
        }

        // Extract header, apply migrations
        const header = this.fileEntries.find(e => e.type === "session");
        this.sessionId = header?.id ?? randomUUID();

        if (migrateToCurrentVersion(this.fileEntries)) {
            this._rewriteFile();  // Persist migration results
        }

        this._buildIndex();
        this.flushed = true;
    } else {
        // File doesn't exist -- create new session
        const explicitPath = this.sessionFile;
        this.newSession();
        this.sessionFile = explicitPath;
    }
}
```

### Repair Checks Performed

1. **Malformed line recovery** (`loadEntriesFromFile`, line 433-458):
   - Parses each line independently with `JSON.parse()` inside a try/catch
   - Malformed lines are silently skipped (partial writes from crashes survive)
   - Validates the first entry is a `SessionHeader` with a valid `id` field
   - Returns empty array if header is missing/invalid

2. **Empty/corrupted file recovery** (`setSessionFile`, line 699-706):
   - If `loadEntriesFromFile` returns no entries, the file is treated as corrupted
   - A fresh session is created and the file is overwritten
   - The original file path is preserved (user's `--session` flag is honored)

3. **Version migration** (`migrateToCurrentVersion`, line 261-271):
   - v1 -> v2: Adds `id`/`parentId` tree structure to flat entries
   - v2 -> v3: Renames `hookMessage` role to `custom`
   - Migrations mutate entries in-place, then the file is rewritten

4. **Orphan handling** (`getTree`, line 1070-1108):
   - When building the tree, entries with broken `parentId` chains are treated as roots
   - This handles partial writes where a child was written but the parent was not

5. **Model/state restoration** (`createAgentSession` in `packages/coding-agent/src/core/sdk.ts:190-342`):
   - Reads existing session context via `sessionManager.buildSessionContext()`
   - Attempts to restore the model from the session's `model_change` entries
   - If the model is no longer available (removed provider, expired auth), falls back to `findInitialModel()`
   - Thinking level is restored from `thinking_level_change` entries, or defaults
   - Messages are replayed into the agent via `agent.state.messages = existingSession.messages`

6. **Lazy file creation** (`_persist`, line 796-814):
   - Session files are not written until the first assistant message arrives
   - This prevents clutter from sessions that never got a response (crash during first prompt)
   - On resume after crash, this means partially-initialized sessions leave no trace

### What Pi-Mono Does NOT Do on Resume

- **No WAL or journaling**: The JSONL is append-only but has no write-ahead log. A crash mid-write can leave a partially-written last line, which is handled by the per-line try/catch but means the last entry may be lost.
- **No lock file for concurrent access**: No check for whether another process has the session open.
- **No integrity checksums**: No CRC or hash verification of entries.
- **No "dirty state" detection**: No mechanism to detect if tools were mid-execution when the crash occurred. Tool results that were never written are simply missing from the resumed session.
- **No pending-tool-call recovery**: If the agent crashed while executing a tool call, the resumed session will have the assistant message with `toolCall` blocks but no corresponding `toolResult` entries. The LLM simply sees missing tool results and works around it.

---

## 3. Loop/Recursion Guards

### The Notable Absence

**Pi-Mono has no built-in loop detection, iteration limits, or recursion depth guards in production code.** The agent loop (`packages/agent/src/agent-loop.ts`) is an unbounded `while(true)` loop:

```typescript
// Outer loop: continues when queued follow-up messages arrive
while (true) {
    let hasMoreToolCalls = true;

    // Inner loop: process tool calls and steering messages
    while (hasMoreToolCalls || pendingMessages.length > 0) {
        // ... stream assistant, execute tools, check steering
    }

    // Check for follow-up messages
    const followUpMessages = (await config.getFollowUpMessages?.()) || [];
    if (followUpMessages.length > 0) {
        pendingMessages = followUpMessages;
        continue;
    }
    break;
}
```

The only exits from this loop are:
1. **Error/abort**: `stopReason === "error" || "aborted"` terminates immediately
2. **No tool calls + no steering + no follow-ups**: Natural exit when the model says "stop" and no queued work remains
3. **AbortSignal**: User cancellation via Ctrl+C

### Guards That DO Exist (Indirect)

1. **Context overflow as implicit iteration limit** (`_checkCompaction` in agent-session.ts:1739-1817):
   - When context usage exceeds `contextWindow - reserveTokens` (default: context window minus 16,384 tokens), auto-compaction is triggered
   - If compaction + retry fails once, `_overflowRecoveryAttempted` is set to `true` and a second overflow terminates the loop
   - This effectively caps session length but not iteration count

2. **Auto-retry cap** (agent-session.ts:2396-2472):
   - Retryable errors have exponential backoff: `baseDelayMs * 2^(attempt-1)` (default: 2s, 4s, 8s)
   - Max retries: 3 (configurable via `settings.retry.maxRetries`)
   - Max delay cap: 60,000ms per retry
   - After max retries, the error is surfaced to the user

3. **`beforeToolCall` hook** (agent/src/types.ts:42-49):
   ```typescript
   interface BeforeToolCallResult {
       block?: boolean;
       reason?: string;
   }
   ```
   Extensions can block individual tool executions. This is the ONLY hook point for implementing custom loop guards -- an extension could count tool calls per session and block after a threshold.

4. **Tool validation** (agent-loop.ts:479-522):
   - Unknown tools produce an immediate error result (not a loop terminator)
   - Schema validation failures produce error results
   - These feed back into the LLM as error tool results, which may cause the model to retry the same tool -- potentially creating a loop

5. **Abort mechanism** (agent.ts:285-287):
   - `Agent.abort()` triggers the abort controller
   - The signal is threaded through to stream functions and tool execution
   - This is the user's manual circuit breaker

### What Is NOT Guarded

- **No max-turns/max-iterations limit**: An agent can loop indefinitely through tool calls as long as the context window holds
- **No tool-call cycle detection**: If the model calls `read -> edit -> read -> edit` in a cycle, nothing detects or breaks it
- **No cost budget enforcement**: Token costs accumulate without any cap
- **No wall-clock timeout**: No maximum runtime for a session or prompt
- **No recursion depth tracking**: Subagent spawning (via extensions) has no depth limit
- **No repeated-failure detection**: If the same tool fails 100 times with the same error, the loop continues (the model will eventually run into context overflow)

### Test-Only Guards

In test files, manual limits are used to prevent infinite loops:

```typescript
// packages/ai/test/stream.test.ts:286
const maxTurns = 5; // Prevent infinite loops

// packages/coding-agent/test/sdk-codex-cache-probe-tool-loop.ts:44
const MAX_TURNS = 50;
```

These are NOT present in production code.

---

## 4. Key Code References

### Stop Reason / Error Handling

| File | Lines | What |
|---|---|---|
| `packages/ai/src/types.ts` | 182 | `StopReason` type definition: `"stop" \| "length" \| "toolUse" \| "error" \| "aborted"` |
| `packages/ai/src/types.ts` | 237-249 | `AssistantMessageEvent` stream protocol with `done`/`error` discriminator |
| `packages/ai/src/utils/overflow.ts` | 28-131 | `isContextOverflow()` with 18 provider-specific regex patterns |
| `packages/agent/src/agent-loop.ts` | 155-232 | `runLoop()` -- the unbounded while(true) agent loop |
| `packages/agent/src/agent-loop.ts` | 194-198 | Error/abort termination check |
| `packages/agent/src/agent.ts` | 459-474 | Synthetic failure message creation for unhandled errors |
| `packages/coding-agent/src/core/agent-session.ts` | 112-129 | `AgentSessionEvent` extensions (compaction, retry events) |
| `packages/coding-agent/src/core/agent-session.ts` | 2381-2393 | `_isRetryableError()` with regex pattern matching |
| `packages/coding-agent/src/core/agent-session.ts` | 2396-2472 | `_handleRetryableError()` with exponential backoff |
| `packages/coding-agent/src/modes/print-mode.ts` | 111-123 | Exit code derivation from stop reason |

### Session Persistence / Resume

| File | Lines | What |
|---|---|---|
| `packages/coding-agent/src/core/session-manager.ts` | 29 | `CURRENT_SESSION_VERSION = 3` |
| `packages/coding-agent/src/core/session-manager.ts` | 433-458 | `loadEntriesFromFile()` with malformed-line recovery |
| `packages/coding-agent/src/core/session-manager.ts` | 695-723 | `setSessionFile()` -- repair logic for corrupted/empty files |
| `packages/coding-agent/src/core/session-manager.ts` | 210-271 | Migration pipeline (v1->v2->v3) |
| `packages/coding-agent/src/core/session-manager.ts` | 308-417 | `buildSessionContext()` -- tree traversal for context reconstruction |
| `packages/coding-agent/src/core/session-manager.ts` | 796-814 | `_persist()` -- lazy write with deferred-until-assistant-message guard |
| `packages/coding-agent/src/core/sdk.ts` | 169-364 | `createAgentSession()` factory with model/state restoration |

### Auto-Compaction (Implicit Loop Guard)

| File | Lines | What |
|---|---|---|
| `packages/coding-agent/src/core/agent-session.ts` | 1739-1817 | `_checkCompaction()` -- overflow and threshold detection |
| `packages/coding-agent/src/core/agent-session.ts` | 1822-1900 | `_runAutoCompaction()` -- compaction execution with extension hooks |
| `packages/coding-agent/src/core/compaction/compaction.ts` | 219-222 | `shouldCompact()` -- threshold calculation |
| `packages/coding-agent/src/core/settings-manager.ts` | 7-11 | `CompactionSettings` interface (enabled, reserveTokens, keepRecentTokens) |
| `packages/coding-agent/src/core/settings-manager.ts` | 18-23 | `RetrySettings` interface (enabled, maxRetries, baseDelayMs, maxDelayMs) |

### Tool Blocking (Extension Hook Point)

| File | Lines | What |
|---|---|---|
| `packages/agent/src/types.ts` | 42-49 | `BeforeToolCallResult` with `block` flag |
| `packages/agent/src/agent-loop.ts` | 491-508 | `beforeToolCall` hook invocation in `prepareToolCall()` |

---

## 5. Patterns Worth Adopting

### Adopt: Stream-Level Error Encoding

Pi-Mono's principle that **failures must be encoded in the stream, not thrown** (`packages/ai/src/types.ts:120-128`) is excellent. This ensures that every agent loop iteration produces a well-formed `AssistantMessage` with a classifiable `stopReason`, regardless of whether the LLM call succeeded. AGH should ensure that ACP driver failures always produce a structured event rather than raw errors.

### Adopt: Provider-Agnostic Overflow Detection

The `isContextOverflow()` function with 18 provider-specific regex patterns is battle-tested infrastructure. AGH can port this pattern to detect context overflow across different ACP-compatible agents, translating provider-specific error strings into a canonical `stop_reason_overflow` enum value.

### Adopt: Retryable Error Classification via Regex

The single regex in `_isRetryableError()` that matches `overloaded|rate_limit|429|500|502|503|504|timeout|connection_error|...` is pragmatic and effective. AGH should have a similar classifier in its session state machine, but should make the patterns configurable per agent driver.

### Adopt: JSONL Append-Only with Malformed-Line Recovery

The JSONL format with per-line try/catch parsing is crash-resilient by design. A partial write corrupts at most one entry. AGH's SQLite approach is more robust overall, but AGH should ensure its event store handles partial-write corruption gracefully (SQLite's WAL already provides this, but the application should verify it).

### Adopt: Deferred File Creation Until First Assistant Message

Pi-Mono's pattern of not writing the session file until the first assistant message arrives prevents file clutter from failed/cancelled sessions. AGH should adopt this for session DB files -- don't create the per-session SQLite database until the first ACP event is received.

### Adopt with Enhancement: Auto-Compaction Overflow Guard

The `_overflowRecoveryAttempted` boolean that prevents infinite compact-and-retry loops is good but primitive. AGH should adopt the concept (compaction as implicit loop guard) but make it more explicit with a configurable max-compaction-retries setting.

### DO NOT Adopt: Unbounded Agent Loop

Pi-Mono's unbounded `while(true)` agent loop with no iteration limit is its most significant resilience gap. AGH MUST add:

1. **Max turns per prompt** -- a hard cap on the number of LLM calls per `session.prompt()` invocation (e.g., 200 turns). Configurable per-session.
2. **Max tool calls per turn** -- a cap on tool calls within a single assistant response (most models already limit this, but a harness-side guard adds defense-in-depth).
3. **Wall-clock timeout per prompt** -- a maximum runtime for a single prompt execution (e.g., 30 minutes).
4. **Cost budget per session** -- a dollar-amount cap that terminates the session when exceeded.
5. **Repeated-failure circuit breaker** -- if the same tool fails N times consecutively with the same error pattern, break the loop.

### DO NOT Adopt: Missing Tool Result on Resume

Pi-Mono's resume behavior silently drops tool results that were mid-execution at crash time. The resumed session has assistant messages with `toolCall` blocks but no `toolResult` entries, leaving the LLM to infer what happened. AGH should instead inject synthetic "tool execution was interrupted by session crash" tool results on resume.

### Consider: Tree-Based Session History

Pi-Mono's tree structure (id/parentId on every entry) enables branching without duplicating files. This is elegant for interactive use but may be overengineered for AGH's daemon model. However, the concept of branching from a past point in the conversation (branch summarization, `branchWithSummary()`) is worth considering for AGH's session fork feature.
