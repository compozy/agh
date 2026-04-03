# Query Engine & Context Management Analysis

## How It Works (in Claude Code)

### 1. Generator-Based Query Loop (`query.ts`)

The core query loop is an `AsyncGenerator` function (`async function* queryLoop`) that runs a `while(true)` loop. Each iteration represents one "turn" -- a call to the model followed by tool execution, followed by context management checks.

**Architecture:**

```
query() [outer wrapper]
  --> queryLoop() [the actual while(true) generator]
       --> each iteration:
            1. Context compression pipeline (snip, microcompact, contextCollapse, autocompact)
            2. API call via deps.callModel() (streamed)
            3. Tool execution (streamed or sequential)
            4. Stop hooks evaluation
            5. State transition (continue with next turn or return terminal)
```

**The State Machine:**

The loop carries a mutable `State` struct between iterations:

```typescript
type State = {
  messages: Message[]
  toolUseContext: ToolUseContext
  autoCompactTracking: AutoCompactTrackingState | undefined
  maxOutputTokensRecoveryCount: number
  hasAttemptedReactiveCompact: boolean
  maxOutputTokensOverride: number | undefined
  pendingToolUseSummary: Promise<ToolUseSummaryMessage | null> | undefined
  stopHookActive: boolean | undefined
  turnCount: number
  transition: Continue | undefined  // Why the previous iteration continued
}
```

Each iteration destructures state at the top, runs its pipeline, and at "continue" sites writes a new `state = { ... }` object. The `transition` field records WHY the loop continued (e.g., `'next_turn'`, `'reactive_compact_retry'`, `'max_output_tokens_recovery'`, `'collapse_drain_retry'`, `'stop_hook_blocking'`, `'token_budget_continuation'`, `'max_output_tokens_escalate'`).

**Terminal conditions** (exit the loop via `return`):
- `'completed'` -- model response has no tool_use, stop hooks pass
- `'aborted_streaming'` / `'aborted_tools'` -- user interrupted
- `'max_turns'` -- turn limit reached
- `'blocking_limit'` -- prompt too long with no recovery
- `'model_error'` -- unrecoverable API error
- `'image_error'` -- image size issues
- `'hook_stopped'` / `'stop_hook_prevented'` -- hooks blocked continuation

**Yield/Resume Pattern:**

The generator yields different message types to the consumer:
- `StreamEvent` -- raw SSE chunks from the API
- `RequestStartEvent` -- signals new API request starting
- `Message` (user/assistant/system/attachment/progress) -- conversation messages
- `TombstoneMessage` -- signals message removal (fallback scenario)
- `ToolUseSummaryMessage` -- haiku-generated summaries of tool actions

The outer `query()` wrapper delegates to `queryLoop()` via `yield*` and handles command lifecycle notifications on normal completion.

**Dependency Injection:**

The loop accepts a `QueryDeps` object (defaulting to `productionDeps()`) with four injectable dependencies:
```typescript
type QueryDeps = {
  callModel: typeof queryModelWithStreaming
  microcompact: typeof microcompactMessages
  autocompact: typeof autoCompactIfNeeded
  uuid: () => string
}
```

This enables clean testing without module-level mocks.

### 2. Three-Tier Context Compression

Context management runs at the TOP of each loop iteration, BEFORE the API call. The pipeline has a specific ordering and each tier addresses different context pressure scenarios.

**Order of execution:**
1. Tool result budget enforcement (`applyToolResultBudget`)
2. Snip compact (`snipCompactIfNeeded`) -- feature-gated
3. Microcompact (`deps.microcompact`)
4. Context collapse (`contextCollapse.applyCollapsesIfNeeded`) -- feature-gated
5. Autocompact (`deps.autocompact`)

Plus reactive recovery AFTER a failed API call:
6. Context collapse drain (`contextCollapse.recoverFromOverflow`)
7. Reactive compact (`reactiveCompact.tryReactiveCompact`)

#### Tier 1: Microcompact (`services/compact/microCompact.ts`)

Microcompact operates at the **tool result level**. It removes or clears old tool results to reduce context size without losing the conversation structure.

Three sub-strategies:
- **Time-based microcompact**: When gap since last assistant message exceeds threshold (cache is cold anyway), content-clear all but the N most recent compactable tool results. Mutates message content directly.
- **Cached microcompact** (ant-only): Uses the API's cache editing feature to delete tool results server-side WITHOUT invalidating the cached prefix. Tracks tool IDs in a module-level state, queues `cache_edits` blocks for the API layer.
- **API-based microcompact** (`apiMicrocompact.ts`): Server-side context management using `clear_tool_uses_20250919` and `clear_thinking_20251015` strategies.

Only specific tools are compactable: Read, Bash/shell, Grep, Glob, WebSearch, WebFetch, Edit, Write.

#### Tier 2: Snip Compact (feature-gated `HISTORY_SNIP`)

Snip operates at the **message level**, removing entire older messages from the conversation. It runs BEFORE microcompact and reports `tokensFreed` so autocompact's threshold check reflects what snip already removed. Returns a boundary message to notify the UI.

#### Tier 3: Context Collapse (feature-gated `CONTEXT_COLLAPSE`)

Context collapse is a **read-time projection** system. It replaces groups of messages with summary messages from a collapse store. Key insight: "Nothing is yielded -- the collapsed view is a read-time projection over the REPL's full history."

Runs BEFORE autocompact so that if collapse gets context under the autocompact threshold, autocompact is a no-op and granular context is preserved.

Has a reactive recovery path: `recoverFromOverflow` drains staged collapses when a real API 413 occurs.

#### Tier 4: Autocompact (`services/compact/autoCompact.ts`)

The heaviest compression -- summarizes the entire conversation via a separate API call. Triggered when token count exceeds `getAutoCompactThreshold(model)` which is `effectiveContextWindow - 13,000` tokens.

Key features:
- **Circuit breaker**: Stops retrying after 3 consecutive failures to avoid wasting API calls
- **Session memory compaction** tried first (experimental)
- **Tracking state**: Records `compacted`, `turnCounter`, `turnId`, `consecutiveFailures` across iterations
- **PTL retry loop**: If the compact request itself hits prompt-too-long, it truncates the oldest API-round groups and retries (up to 3 times)

The compact process (`compact.ts`):
1. Strips images from messages (not needed for summary)
2. Strips re-injectable attachments (skill_discovery, skill_listing)
3. Sends the conversation to the model with a summary prompt
4. Creates post-compact file attachments (re-reads the 5 most recently accessed files)
5. Re-announces deferred tools, agent listings, MCP instructions
6. Runs SessionStart hooks

#### Reactive Compact (prompt-too-long recovery)

When the API returns a 413 prompt-too-long error, the streaming loop WITHHOLDS the error message. Then after streaming completes:
1. First tries context collapse drain (cheap, keeps granular context)
2. If that fails, tries reactive compact (full summary)
3. Single-shot on each -- if retry still 413's, error surfaces

### 3. Streaming and Retries (`services/api/withRetry.ts`)

**Streaming architecture:**

The API call returns an async iterable of messages. During streaming:
- Assistant messages are accumulated into `assistantMessages[]`
- Tool use blocks are detected and accumulated
- The `StreamingToolExecutor` can begin executing tools WHILE streaming continues
- Recoverable errors (prompt-too-long, max-output-tokens, media-size) are WITHHELD from yield to allow recovery

**Retry mechanism (`withRetry`):**

An async generator that wraps API operations with sophisticated retry logic:

- **Default max retries**: 10 (configurable via env)
- **Exponential backoff**: `min(500ms * 2^(attempt-1), 32s)` + 25% jitter
- **Retry-After header**: Honored when present
- **529 (overloaded) handling**: Up to 3 consecutive 529s, then triggers model fallback (e.g., Opus -> Sonnet)
- **401/403 handling**: Refreshes OAuth tokens, clears credential caches
- **Context overflow (400)**: Parses error to extract token counts, adjusts max_tokens for retry
- **Persistent retry mode** (unattended sessions): Infinite retries with chunked sleeps that yield heartbeat messages every 30s to keep connections alive. Respects rate-limit-reset timestamps.
- **Fast mode fallback**: On 429/529, short delays retry with fast mode, long delays enter cooldown (standard speed)
- **Foreground vs background**: Only foreground queries retry 529s; background tasks (summaries, suggestions) bail immediately to reduce amplification during capacity cascades

**Fallback model pattern:**

When a `FallbackTriggeredError` is thrown, the outer streaming loop:
1. Yields tombstones for orphaned assistant messages
2. Clears accumulated state
3. Strips thinking signatures (model-bound)
4. Switches to fallback model
5. Yields a system warning message
6. Retries the entire request

**Max output tokens recovery:**

When model hits output token limit:
1. First tries escalating from default 8k to 64k (single retry)
2. Then injects recovery messages asking model to resume mid-thought (up to 3 attempts)
3. If all recovery exhausted, surfaces the withheld error

### 4. Cost Tracking (`cost-tracker.ts`)

**Architecture:**

Cost tracking is centralized in bootstrap state (global singletons). The `cost-tracker.ts` module provides the public API while state lives in `bootstrap/state.ts`.

**Per-request tracking:**
```typescript
function addToTotalSessionCost(cost: number, usage: Usage, model: string): number
```
- Accumulates per-model usage (input, output, cache read, cache creation, web search)
- Records context window and max output tokens per model
- Feeds OpenTelemetry counters (cost counter, token counter by type)
- Recursively processes advisor model usage (model-within-model for advisor tools)

**Session persistence:**
- `saveCurrentSessionCosts()`: Saves to project config (`.claude/config.json`) on process exit
- `restoreCostStateForSession()`: Restores on `--resume` if session ID matches
- `getStoredSessionCosts()`: Reads without restoring (used for display)

**Budget enforcement (in QueryEngine):**
```typescript
if (maxBudgetUsd !== undefined && getTotalCost() >= maxBudgetUsd) {
  yield { type: 'result', subtype: 'error_max_budget_usd', ... }
  return
}
```
Checked after every yielded message in the QueryEngine's `submitMessage` loop.

**Token budget tracking (`query/tokenBudget.ts`):**

Separate from USD budget -- tracks output tokens against a per-turn budget:
```typescript
type BudgetTracker = {
  continuationCount: number
  lastDeltaTokens: number
  lastGlobalTurnTokens: number
  startedAt: number
}
```
- Continues if under 90% of budget and not showing diminishing returns
- Diminishing returns detection: 3+ continuations with <500 tokens each
- Injects nudge messages to tell the model to keep going

**Display:**
- `formatTotalCost()`: Shows cost, duration (API + wall), lines changed, per-model breakdown
- `useCostSummary()`: React hook that prints cost summary and saves on process exit

### 5. Conversation History Management

#### QueryEngine (`QueryEngine.ts`)

`QueryEngine` is the class that owns the query lifecycle for a conversation. One instance per conversation. Key state:

```typescript
class QueryEngine {
  private mutableMessages: Message[]
  private abortController: AbortController
  private permissionDenials: SDKPermissionDenial[]
  private totalUsage: NonNullableUsage
  private readFileState: FileStateCache
  private discoveredSkillNames: Set<string>
  private loadedNestedMemoryPaths: Set<string>
}
```

**`submitMessage()` flow:**
1. Process user input (slash commands, attachments, tools)
2. Push new messages to `mutableMessages`
3. Persist user messages to transcript BEFORE entering query loop
4. Build system init message (tools, model, permissions, etc.)
5. Enter `query()` generator loop
6. For each yielded message: record to transcript, accumulate usage, track turns
7. Check budget limits after each message
8. On completion: extract text result, check success, yield result message

**Message persistence:**
- `recordTranscript(messages)`: Writes to session storage (JSONL file)
- Fire-and-forget for assistant messages (mutation-based, lazy serialize)
- Awaited for user messages (must be resumable if process killed)
- Compact boundaries trigger flush of in-memory messages

**Memory management after compaction:**
```typescript
// Release pre-compaction messages for GC
const mutableBoundaryIdx = this.mutableMessages.length - 1
if (mutableBoundaryIdx > 0) {
  this.mutableMessages.splice(0, mutableBoundaryIdx)
}
```

#### Prompt History (`history.ts`)

Separate from conversation messages -- this is the user's command history (like shell history).

**Storage**: Append-only JSONL file at `~/.claude/history.jsonl`
- File-level locking for concurrent session safety
- Max 100 entries per project
- Session-scoped ordering (current session entries first in up-arrow)

**Paste handling**: Large pasted content stored in a separate paste store:
- Small pastes (<1024 chars): Inline in history entry
- Large pastes: Content-addressed hash, stored separately
- Lazy resolution on read (`resolve()`)

**Write batching**: Entries accumulate in `pendingEntries[]`, flushed with retry logic (max 5 retries, 500ms between). Cleanup registered for process exit.

## Key Patterns Worth Adopting

### 1. Generator-Based Agentic Loop with Explicit State Transitions

The `while(true)` + `state = next` pattern makes the loop a state machine with named transitions. Each "continue" site constructs a complete new state, and the `transition` field records the reason.

```typescript
// From query.ts -- state transition example
const next: State = {
  messages: [...messagesForQuery, ...assistantMessages, recoveryMessage],
  toolUseContext,
  autoCompactTracking: tracking,
  maxOutputTokensRecoveryCount: maxOutputTokensRecoveryCount + 1,
  hasAttemptedReactiveCompact,
  maxOutputTokensOverride: undefined,
  pendingToolUseSummary: undefined,
  stopHookActive: undefined,
  turnCount,
  transition: { reason: 'max_output_tokens_recovery', attempt: maxOutputTokensRecoveryCount + 1 },
}
state = next
continue
```

**Why this matters**: Tests can assert on `transition.reason` without inspecting message contents. The state is always consistent (no partial updates).

### 2. Dependency Injection for the Query Loop

```typescript
// From query/deps.ts
export type QueryDeps = {
  callModel: typeof queryModelWithStreaming
  microcompact: typeof microcompactMessages
  autocompact: typeof autoCompactIfNeeded
  uuid: () => string
}
```

Production code uses `productionDeps()`. Tests inject fakes directly. Eliminates 6-8 files of `spyOn` boilerplate per test.

### 3. Withhold-Then-Recover Error Handling

Recoverable errors (413, max_output_tokens, media-size) are detected during streaming but NOT yielded to consumers. They're pushed to `assistantMessages` for the recovery checks to find. If recovery succeeds, the consumer never sees the error. If recovery fails, the error surfaces.

```typescript
let withheld = false
if (reactiveCompact?.isWithheldPromptTooLong(message)) withheld = true
if (isWithheldMaxOutputTokens(message)) withheld = true
if (!withheld) yield yieldMessage
```

### 4. Multi-Layer Context Compression Pipeline

Each layer addresses a different granularity and cost profile:
1. **Tool result budget** -- per-message, zero API cost
2. **Snip** -- message-level, zero API cost
3. **Microcompact** -- tool-result-level, cache-aware, zero/low API cost
4. **Context collapse** -- projection-based, zero API cost
5. **Autocompact** -- full summary, one API call

They compose cleanly because each runs before the next and reports what it freed.

### 5. Streaming Tool Execution with Concurrency Control

```typescript
class StreamingToolExecutor {
  // Concurrent-safe tools run in parallel
  // Non-concurrent tools run exclusively
  // Results buffered and emitted in order
  // Sibling abort: Bash error kills sibling subprocesses
}
```

Tools start executing AS the model streams, not after. This overlaps model inference latency with tool execution latency.

### 6. Circuit Breaker for Autocompact

```typescript
const MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES = 3

if (tracking?.consecutiveFailures >= MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES) {
  return { wasCompacted: false }  // Stop trying
}
```

Without this, sessions with irrecoverably long context hammer the API with doomed compaction attempts on every turn.

### 7. Config Snapshot at Entry

```typescript
// From query/config.ts
export type QueryConfig = {
  sessionId: SessionId
  gates: {
    streamingToolExecution: boolean
    emitToolUseSummaries: boolean
    isAnt: boolean
    fastModeEnabled: boolean
  }
}
```

Snapshotted once at `query()` entry. Feature gates that are tree-shaking boundaries stay inline (bun:bundle dead code elimination). Runtime gates that merely need consistency within a turn go in the config.

### 8. Prefetch-During-Streaming Pattern

Memory prefetch and skill discovery prefetch start at the top of each iteration, before the API call. They resolve during model streaming (which takes 5-30s). They're consumed after tool execution -- by which point they've almost certainly resolved.

```typescript
using pendingMemoryPrefetch = startRelevantMemoryPrefetch(messages, toolUseContext)
// ... model streams for 5-30s ...
// ... tools execute ...
if (pendingMemoryPrefetch.settledAt !== null) {
  const memoryAttachments = await pendingMemoryPrefetch.promise
  // inject results
}
```

### 9. Tool Use Summary Generation (Fire-and-Forget)

Haiku generates a human-readable summary of what tools did. The promise starts after tool batch completes, resolves during the NEXT model streaming call, and is yielded at the start of the NEXT iteration.

```typescript
nextPendingToolUseSummary = generateToolUseSummary({ tools: toolInfoForSummary, ... })
  .then(summary => summary ? createToolUseSummaryMessage(summary, toolUseIds) : null)
  .catch(() => null)
```

### 10. Exponential Backoff with Multiple Strategies

```typescript
// Base: 500ms * 2^(attempt-1), capped at 32s, +25% jitter
// Persistent mode: up to 5min backoff, chunked 30s heartbeats
// Fast mode: short delays retry same model, long delays switch to standard
// Rate limit reset: reads anthropic-ratelimit-unified-reset header
```

## Ideas for Our System

### 1. Go Channel-Based Query Loop

Translate the generator pattern to Go using channels. The `while(true)` loop becomes a goroutine that sends events on a channel:

```go
type QueryEvent struct {
    Type       EventType
    Message    *Message
    Stream     *StreamChunk
    Transition *Transition
}

type QueryLoop struct {
    state  *LoopState
    deps   QueryDeps
    config QueryConfig
    events chan<- QueryEvent
}

func (q *QueryLoop) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        transition := q.runIteration(ctx)
        if transition.IsTerminal() {
            return nil
        }
        q.state = transition.NextState()
    }
}
```

### 2. Pluggable Context Compression Pipeline

Define a `ContextCompressor` interface and compose them:

```go
type ContextCompressor interface {
    Name() string
    Compress(ctx context.Context, messages []Message, opts CompressOpts) (*CompressResult, error)
}

type CompressResult struct {
    Messages    []Message
    TokensFreed int
    Boundary    *BoundaryMessage // optional
}

// Pipeline runs compressors in order, short-circuiting if under threshold
type CompressionPipeline struct {
    compressors []ContextCompressor
    threshold   int
}
```

This maps to their snip -> microcompact -> collapse -> autocompact pipeline but is extensible.

### 3. Withhold-Recover Error Pattern

Implement a "withholdable" error wrapper:

```go
type WithheldError struct {
    Original    error
    RecoveryFn  func(ctx context.Context, state *LoopState) (*LoopState, error)
}

// During streaming, collect withheld errors instead of returning
// After streaming, attempt recovery in priority order
```

### 4. Circuit Breaker for Compaction

```go
type CircuitBreaker struct {
    maxFailures int
    failures    int
    mu          sync.Mutex
}

func (cb *CircuitBreaker) Allow() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    return cb.failures < cb.maxFailures
}

func (cb *CircuitBreaker) RecordSuccess() { cb.failures = 0 }
func (cb *CircuitBreaker) RecordFailure() { cb.failures++ }
```

### 5. Streaming Tool Executor with Concurrency Control

```go
type StreamingToolExecutor struct {
    tools       []TrackedTool
    results     chan ToolResult  // ordered output channel
    concSafe    sync.WaitGroup  // tracks concurrent-safe tools
    exclusive   sync.Mutex      // exclusive access for non-concurrent tools
    siblingCtx  context.Context
    siblingStop context.CancelFunc
}
```

Use Go's native concurrency primitives instead of Promise.race patterns.

### 6. Cost Tracker as First-Class Component

```go
type CostTracker struct {
    mu           sync.RWMutex
    totalCostUSD float64
    modelUsage   map[string]*ModelUsage
    budgetUSD    *float64  // nil = unlimited
    tokenBudget  *TokenBudget
}

func (ct *CostTracker) Add(usage Usage, model string) error {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    cost := calculateCost(model, usage)
    ct.totalCostUSD += cost
    if ct.budgetUSD != nil && ct.totalCostUSD >= *ct.budgetUSD {
        return ErrBudgetExceeded
    }
    return nil
}
```

### 7. Turn-Scoped State with Explicit Transitions

Define the state machine explicitly in Go:

```go
type Transition struct {
    Reason string      // "next_turn", "reactive_compact_retry", etc.
    State  *LoopState
}

type LoopState struct {
    Messages                []Message
    AutoCompactTracking     *AutoCompactTracking
    MaxOutputRecoveryCount  int
    HasAttemptedCompact     bool
    TurnCount               int
    PendingToolSummary      <-chan *ToolSummary
}
```

Each continue site constructs a complete `Transition`. Tests assert on `Reason` field.

### 8. Retry with Exponential Backoff and Heartbeats

```go
func WithRetry[T any](ctx context.Context, opts RetryOpts, fn func(attempt int) (T, error)) (T, error) {
    for attempt := 1; attempt <= opts.MaxRetries+1; attempt++ {
        result, err := fn(attempt)
        if err == nil {
            return result, nil
        }
        if !isRetryable(err) {
            return zero, err
        }
        delay := calculateDelay(attempt, err, opts)
        // Chunk long sleeps for heartbeats
        if err := sleepWithHeartbeats(ctx, delay, opts.HeartbeatInterval, opts.OnHeartbeat); err != nil {
            return zero, err
        }
    }
}
```

### 9. Prefetch Pattern Using Go Channels

```go
// Start prefetch before API call
prefetchCh := make(chan []Attachment, 1)
go func() {
    result, _ := prefetchMemory(ctx, messages)
    prefetchCh <- result
}()

// ... model streams for seconds ...
// ... tools execute ...

// Consume with non-blocking check
select {
case attachments := <-prefetchCh:
    messages = append(messages, attachments...)
default:
    // Not ready yet, skip this iteration
}
```

### 10. Immutable Config + Mutable State Separation

Mirror the Claude Code pattern of separating immutable per-query config from mutable per-iteration state:

```go
// Immutable -- snapshotted once at query entry
type QueryConfig struct {
    SessionID              string
    StreamingToolExecution bool
    EmitToolUseSummaries   bool
}

// Mutable -- reconstructed at each continue site
type IterationState struct {
    Messages              []Message
    CompactTracking       *CompactTracking
    RecoveryCount         int
    TurnCount             int
    Transition            *Transition
}
```

## Key Files Reference

| File | Description |
|------|-------------|
| `query.ts` | Core `while(true)` query loop -- the heart of the agent. ~1730 lines. Handles streaming, tool execution, error recovery, context compression orchestration. |
| `QueryEngine.ts` | Class wrapping query lifecycle for SDK/headless use. Manages `mutableMessages`, usage accumulation, transcript persistence, permission tracking. |
| `query/config.ts` | Immutable `QueryConfig` snapshotted once per query entry. Separates runtime gates from tree-shaking feature() gates. |
| `query/deps.ts` | Dependency injection for query loop (callModel, microcompact, autocompact, uuid). |
| `query/tokenBudget.ts` | Token budget tracker -- continuation vs stop decisions based on output token consumption. |
| `query/stopHooks.ts` | Stop hook execution after model response. Handles Stop, TeammateIdle, TaskCompleted hooks. |
| `cost-tracker.ts` | Cost tracking API -- per-model usage, session persistence, formatting. State in bootstrap. |
| `costHook.ts` | React hook for displaying cost summary on exit. |
| `history.ts` | Prompt history (shell-like up-arrow). JSONL with file locking, paste store for large content. |
| `services/compact/compact.ts` | Full conversation compaction. Streams a summary via a separate API call, rebuilds post-compact context (files, tools, plans, skills, hooks). |
| `services/compact/autoCompact.ts` | Auto-compaction trigger logic. Threshold calculation, circuit breaker, session memory first. |
| `services/compact/microCompact.ts` | Lightweight tool-result-level compaction. Time-based, cached (cache-editing API), and legacy paths. |
| `services/compact/apiMicrocompact.ts` | Server-side context management using API strategies (clear_tool_uses, clear_thinking). |
| `services/compact/compactWarningState.ts` | Zustand-like store for suppressing compact warnings after successful compaction. |
| `services/api/withRetry.ts` | Retry logic with exponential backoff, 529 handling, model fallback, persistent mode, fast mode cooldown. |
| `services/tools/StreamingToolExecutor.ts` | Concurrent tool execution during streaming. Concurrency control, sibling abort, ordered result emission. |
| `utils/queryHelpers.ts` | Helper functions for result success checks, message normalization, file state extraction. |
| `utils/queryContext.ts` | System prompt assembly -- fetches prompt parts, builds side-question fallback params. |
| `utils/queryProfiler.ts` | Performance profiling with named checkpoints for identifying TTFT bottlenecks. |
| `_prompts/service-compact.md` | Compaction prompt templates -- analysis + summary structure for the summarizer model. |
