# Claude Code: Session Resilience Analysis

Deep analysis of Claude Code's source at `/Users/pedronauck/dev/knowledge/.resources/claude-code/` covering stop reason taxonomy, session repair on resume, and loop/recursion guards.

---

## 1. Stop Reason Taxonomy

Claude Code uses a two-layer stop reason system: **internal query loop reasons** (why the `query()` generator returned) and **SDK result subtypes** (what the external caller sees).

### 1.1 Internal Query Loop Terminal Reasons

The `query()` function in `query.ts` returns a `Terminal` object with a `reason` string. Every exit path from the query loop is explicitly named:

| Reason | Trigger | Location |
|--------|---------|----------|
| `'completed'` | Model finished naturally (no tool calls, stop hooks pass) | `query.ts:1264, 1357` |
| `'aborted_streaming'` | User cancelled during model streaming (`abortController.signal.aborted`) | `query.ts:1051` |
| `'aborted_tools'` | User cancelled during tool execution | `query.ts:1515` |
| `'max_turns'` | Turn count exceeded `maxTurns` limit | `query.ts:1711` |
| `'hook_stopped'` | A Stop hook set `preventContinuation = true` | `query.ts:1520` |
| `'stop_hook_prevented'` | Stop hook explicitly prevented continuation | `query.ts:1279` |
| `'blocking_limit'` | Token count hit hard limit (auto-compact off, context too large) | `query.ts:646` |
| `'prompt_too_long'` | API returned prompt-too-long error, all recovery failed | `query.ts:1175, 1182` |
| `'image_error'` | Image too large or resize failed | `query.ts:977, 1175` |
| `'model_error'` | Unrecoverable API/model error (catch-all) | `query.ts:996` |

### 1.2 SDK Result Subtypes (External Interface)

The `QueryEngine` (`QueryEngine.ts`) translates internal reasons into SDK-facing result messages:

| Subtype | Meaning | `is_error` |
|---------|---------|------------|
| `'success'` | Turn completed normally | `false` (or `true` if API error occurred) |
| `'error_during_execution'` | Model failed to produce valid response | `true` |
| `'error_max_turns'` | Hit `maxTurns` limit | `true` |
| `'error_max_budget_usd'` | Spend exceeded `maxBudgetUsd` | `true` |
| `'error_max_structured_output_retries'` | Structured output validation failed N times (default 5) | `true` |

### 1.3 API Error Classification

The `SDKAssistantMessageErrorSchema` defines the error types surfaced to callers:
- `'authentication_failed'`
- `'billing_error'`
- `'rate_limit'`
- `'invalid_request'`
- `'server_error'`
- `'unknown'`
- `'max_output_tokens'`

### 1.4 Session End Reasons

The `SessionEnd` hook (`coreSchemas.ts:747-755`) defines why a session ended:
- `'clear'` -- User cleared the session
- `'resume'` -- Session ended because a different session was resumed
- `'logout'` -- User logged out
- `'prompt_input_exit'` -- User exited at the prompt (Ctrl+D, /exit)
- `'other'` -- Catch-all
- `'bypass_permissions_disabled'` -- Bypass permissions mode was disabled

### 1.5 API-Level Stop Reasons

From the Anthropic API, `stop_reason` values include:
- `'end_turn'` -- Normal completion
- `'tool_use'` -- Model wants to use a tool (unreliable per comments in `query.ts:557`)
- `'max_tokens'` -- Output token limit hit (handled via recovery loop)

The SDK captures `stop_reason` from `message_delta` events during streaming (`QueryEngine.ts:806-808`).

---

## 2. Session Repair on Resume

### 2.1 Resume Entry Points

Claude Code supports multiple resume paths:
- `--continue` (most recent session)
- `--resume <session-id>` (specific session)
- `--resume <path.jsonl>` (specific transcript file)

All funnel through `loadConversationForResume()` in `utils/conversationRecovery.ts`.

### 2.2 Transcript Loading and Chain Reconstruction

Sessions are persisted as JSONL files with `parentUuid`-linked messages forming a DAG. Resume must reconstruct the linear conversation:

1. **`loadTranscriptFile()`** -- Parses JSONL, builds `byUuid` map, identifies leaf UUIDs
2. **`buildConversationChain()`** -- Walks `parentUuid` back from the newest non-sidechain leaf to build the linear chain
3. **`recoverOrphanedParallelToolResults()`** -- Finds `tool_result` messages whose `parentUuid` points to the wrong message (due to parallel tool execution race conditions) and inserts them at the correct position
4. **`removeExtraFields()`** -- Strips internal fields before deserialization

### 2.3 Consistency Checks

**`checkResumeConsistency()`** (`sessionStorage.ts:2224`) -- Finds the latest `turn_duration` checkpoint message in the chain and compares its recorded `messageCount` against the chain's reconstructed position. Emits `tengu_resume_consistency_delta` to BigQuery:
- `delta > 0`: resume loaded MORE messages than were in-session (the common failure mode)
- `delta < 0`: resume loaded FEWER (chain truncation)
- `delta = 0`: round-trip consistent

### 2.4 Message Deserialization Filters

`deserializeMessagesWithInterruptDetection()` applies five repair passes in order:

1. **`migrateLegacyAttachmentTypes()`** -- Transforms `new_file` to `file`, `new_directory` to `directory`, backfills `displayPath`
2. **Strip invalid permission modes** -- Removes `permissionMode` values from deserialized user messages that don't match the current build's valid modes
3. **`filterUnresolvedToolUses()`** -- Removes assistant messages containing `tool_use` blocks that have no matching `tool_result`. This is the primary crash recovery mechanism: if the process crashed between emitting a `tool_use` and receiving the `tool_result`, the orphaned assistant message is dropped
4. **`filterOrphanedThinkingOnlyMessages()`** -- Removes assistant messages that contain ONLY thinking/redacted_thinking blocks without text or tool_use. These arise from streaming yielding separate messages per content block; interleaved user messages prevent merging by `message.id`
5. **`filterWhitespaceOnlyAssistantMessages()`** -- Removes assistant messages with only whitespace text (happens when model outputs `\n\n` before thinking and user cancels mid-stream)

### 2.5 Turn Interruption Detection

`detectTurnInterruption()` classifies the conversation state after filtering:

| Last Message Type | Condition | Classification |
|-------------------|-----------|----------------|
| `assistant` | Any | `'none'` (turn completed -- `stop_reason` is always null on persisted messages in the streaming path) |
| `user` (meta/compact) | `isMeta` or `isCompactSummary` | `'none'` |
| `user` (tool result) | Terminal tool (Brief/SendUserFile) | `'none'` |
| `user` (tool result) | Non-terminal | `'interrupted_turn'` -- tool was executing when crash occurred |
| `user` (plain text) | Not meta | `'interrupted_prompt'` -- user submitted prompt, model never responded |
| `attachment` | Any | `'interrupted_turn'` |

When an interruption is detected:
- `interrupted_turn`: A synthetic "Continue from where you left off." user message is appended
- `interrupted_prompt`: The original user message is preserved for auto-continuation
- A synthetic assistant sentinel (`NO_RESPONSE_REQUESTED`) is appended after the last user message to make the conversation API-valid

### 2.6 State Restoration

Beyond messages, resume restores:
- **Skill state** (`restoreSkillStateFromMessages()`) -- Scans for `invoked_skills` attachments and re-registers skills to survive across compaction cycles
- **Skill listing suppression** -- Prevents re-announcing available skills
- **File history** (`copyFileHistoryForResume()`) -- Copies file state snapshots
- **Plans** (`copyPlanForResume()`) -- Associates plans with the resumed session
- **Agent context** -- Restores `agentName`, `agentColor`, `agentSetting`, `customTitle`, `tag`, `mode` (coordinator/normal)
- **Worktree session** -- Restores worktree path if present
- **PR context** -- Restores `prNumber`, `prUrl`, `prRepository`
- **Coordinator mode** (`matchSessionMode()`) -- Flips `CLAUDE_CODE_COORDINATOR_MODE` env var to match the resumed session's mode
- **Session start hooks** -- Fires `processSessionStartHooks('resume', { sessionId })`

### 2.7 Transcript Pre-flush

The `QueryEngine` writes the user's message to the transcript BEFORE entering the query loop (`QueryEngine.ts:440-463`). This ensures:
- If the process is killed before the API responds, the transcript is resumable from the point the user message was accepted
- `--resume` finds the session even if no API response ever arrived

### 2.8 Compact Boundary Handling

Before writing a `compact_boundary` message, a flush of all in-memory messages up through the preserved segment tail is forced. Without this, if the subprocess restarts between turns, `tailUuid` points to a never-written message, and `applyPreservedSegmentRelinks` fails.

---

## 3. Loop/Recursion Guards

### 3.1 Max Turns Limit

**The primary loop guard.** The `query()` function tracks `turnCount` and checks against `maxTurns` before each recursive iteration (`query.ts:1704-1711`):

```typescript
if (maxTurns && nextTurnCount > maxTurns) {
  yield createAttachmentMessage({
    type: 'max_turns_reached',
    maxTurns,
    turnCount: nextTurnCount,
  })
  return { reason: 'max_turns', turnCount: nextTurnCount }
}
```

`maxTurns` can be set via:
- `QueryEngine.config.maxTurns` -- SDK/headless callers
- Agent frontmatter `maxTurns` field -- Per-agent limits
- `AgentDefinitionSchema` validates it as a positive integer

There is **no hardcoded default** for the main session -- the limit is only enforced when explicitly set. Subagents inherit their definition's `maxTurns` or use the caller-provided value.

### 3.2 Budget/Cost Guards

**USD Budget** (`QueryEngine.ts:972-1002`):
```typescript
if (maxBudgetUsd !== undefined && getTotalCost() >= maxBudgetUsd) {
  // yields error_max_budget_usd result
}
```
Checked after every message in the query loop. Yields `error_max_budget_usd`.

**Token Budget** (`query/tokenBudget.ts`):
- Tracks `continuationCount`, `lastDeltaTokens`, `lastGlobalTurnTokens`
- `COMPLETION_THRESHOLD = 0.9` -- Continues auto-nudging until 90% of budget consumed
- `DIMINISHING_THRESHOLD = 500` -- Early stop if less than 500 new tokens per check for 3+ consecutive checks (diminishing returns detection)
- When continuing, injects a nudge message with percentage and token usage

### 3.3 Max Output Tokens Recovery

When the model hits `max_output_tokens`, Claude Code has a multi-stage recovery (`query.ts:1188-1256`):

1. **Escalation** (first attempt): If using the default 8k cap and no env override, retry at `ESCALATED_MAX_TOKENS` (64k) -- single shot, same request
2. **Multi-turn recovery** (up to `MAX_OUTPUT_TOKENS_RECOVERY_LIMIT = 3`): Injects a meta message "Output token limit hit. Resume directly -- no apology, no recap..." and continues the loop
3. **Exhaustion**: After 3 recovery attempts, surfaces the withheld error

### 3.4 Prompt-Too-Long Recovery Chain

When the context exceeds the model's limit, a cascading recovery chain fires:

1. **Context Collapse Drain** -- Commits all staged collapses (cheap, keeps granular context)
2. **Reactive Compact** -- Full conversation summarization as fallback
3. **Surface Error** -- If both fail, yields the error and exits

Guard against infinite loops: `hasAttemptedReactiveCompact` flag prevents retry spirals. If reactive compact already ran and prompt is still too long, the error surfaces immediately.

### 3.5 Stop Hook Loop Prevention

Stop hooks can return `blockingErrors` that cause the query loop to continue (model retries with the error feedback). Key guards:

- `stopHookActive` flag tracks whether we're already in a stop-hook-retry loop
- `hasAttemptedReactiveCompact` is **preserved** across stop-hook retries to prevent: `compact -> still too long -> error -> stop hook blocking -> compact -> ...` infinite loop (`query.ts:1295-1298`)
- API error messages bypass stop hooks entirely: "hooks evaluating [an error response] create a death spiral: error -> hook blocking -> retry -> error -> ..." (`query.ts:1259-1264`)

### 3.6 API Retry Guards

`withRetry.ts` enforces:
- `DEFAULT_MAX_RETRIES = 10` (configurable via `CLAUDE_CODE_MAX_RETRIES` env var)
- `MAX_529_RETRIES = 3` (server overload errors)
- `BASE_DELAY_MS = 500` with exponential backoff
- Foreground-only 529 retry: Background queries (summaries, titles, suggestions) bail immediately on 529 to avoid "3-10x gateway amplification" during capacity cascades
- Model fallback: On `FallbackTriggeredError`, switches to `fallbackModel` and retries

### 3.7 Structured Output Retry Guard

`QueryEngine.ts:1004-1048`:
- `MAX_STRUCTURED_OUTPUT_RETRIES` defaults to 5 (configurable via env)
- Counts `SyntheticOutputTool` calls per query; exits with `error_max_structured_output_retries` when exceeded

### 3.8 Subagent Depth/Recursion Guards

- Subagents run `query()` independently but with their own `maxTurns` (from agent definition or caller)
- Async agents get a new unlinked `AbortController` (run independently of parent)
- Sync agents share parent's `AbortController` (parent cancel kills child)
- Agent definitions support a `querySource` tag (e.g., `'agent:builtin:fork'`) used for recursive-fork guards at the `AgentTool.tsx` call site
- `filterIncompleteToolCalls()` in `runAgent.ts` strips tool calls from parent context that lack results, preventing API errors in forked conversations

### 3.9 Auto-Compact as Implicit Guard

Auto-compaction fires when context tokens approach the model's limit, summarizing history. This acts as an implicit loop guard by preventing context exhaustion, but it is NOT itself a turn limiter. It works with:
- `snipCompact` -- Removes old messages exceeding a threshold
- `microcompact` -- Per-message budget on tool result size
- `contextCollapse` -- Hierarchical context management (staged collapses)
- Consecutive failure tracking with circuit breaker

### 3.10 AbortController Chain

Every long-running operation checks `toolUseContext.abortController.signal`:
- Model streaming loop checks after every chunk
- Tool execution checks before and after each tool
- Stop hooks check after each hook result
- User interrupts (Ctrl+C) set `signal.reason = 'interrupt'` for submit-interrupts (queued message follows)

---

## 4. Key Code References

| File | Key Content |
|------|-------------|
| `query.ts` | Main query loop with all terminal reasons, recovery chains, and guards |
| `QueryEngine.ts` | SDK-facing orchestrator, budget checks, result subtype translation |
| `utils/conversationRecovery.ts` | `loadConversationForResume()`, `deserializeMessagesWithInterruptDetection()`, `detectTurnInterruption()` |
| `utils/sessionStorage.ts:2224` | `checkResumeConsistency()` -- delta monitoring for write/load drift |
| `utils/messages.ts:2795` | `filterUnresolvedToolUses()` -- primary crash recovery filter |
| `utils/messages.ts:4991` | `filterOrphanedThinkingOnlyMessages()` |
| `query/stopHooks.ts` | Stop hook execution, blocking error handling, loop prevention |
| `query/tokenBudget.ts` | Token budget tracking with diminishing returns detection |
| `services/api/withRetry.ts` | API retry logic with exponential backoff, 529-specific limits |
| `entrypoints/sdk/coreSchemas.ts:747` | `EXIT_REASONS` enum, `HOOK_EVENTS` list |
| `entrypoints/sdk/coreSchemas.ts:1407-1455` | SDK result schemas (success, error subtypes) |
| `tools/AgentTool/runAgent.ts` | Subagent lifecycle, abort controller isolation, `maxTurns` inheritance |
| `coordinator/coordinatorMode.ts` | Coordinator mode matching on resume |
| `bootstrap/state.ts` | Global state including `strictToolResultPairing` flag |

---

## 5. Patterns Worth Adopting

### 5.1 Multi-Pass Message Sanitization on Resume

Claude Code's `deserializeMessagesWithInterruptDetection()` runs 5 ordered filters that progressively clean the message history. AGH should adopt this pattern:
- Filter orphaned tool calls (no result)
- Filter orphaned thinking blocks
- Filter whitespace-only messages
- Detect and classify turn interruptions
- Append synthetic continuation messages

This is more robust than trying to validate everything in a single pass.

### 5.2 Explicit Terminal Reason Taxonomy

Every exit path from the query loop returns a `{ reason: string }` object. This makes telemetry, debugging, and downstream behavior trivial. AGH should define a Go enum of terminal reasons rather than relying on error types alone.

### 5.3 Cascading Recovery with One-Shot Guards

The pattern of `hasAttemptedReactiveCompact` (boolean latch) preventing infinite recovery loops is elegant. For AGH:
- Each recovery mechanism gets a one-shot flag
- Recovery chains cascade (cheap first, expensive last)
- Once exhausted, error surfaces immediately
- Flags are explicitly preserved across stop-hook retries

### 5.4 Budget Checks in the Hot Loop

Cost and turn limits are checked inline after every message yield, not after the query completes. This enables fine-grained control and immediate termination. AGH should check budgets at the same granularity (per-event, not per-turn).

### 5.5 Transcript-Before-Query Pattern

Writing the user message to the transcript BEFORE entering the query loop ensures sessions are always resumable, even if the API call never completes. This is a crash recovery best practice AGH must adopt.

### 5.6 Turn Interruption Classification

The 3-way classification (`none`, `interrupted_prompt`, `interrupted_turn`) with synthetic continuation messages is clean. AGH should detect:
- Clean completion (assistant message is last)
- Mid-tool interruption (tool result is last, non-terminal)
- Pre-response interruption (user message is last, unanswered)

### 5.7 Consistency Delta Monitoring

`checkResumeConsistency()` compares checkpointed message counts against reconstructed chain lengths. This detects silent data corruption (messages lost or duplicated during write/load round-trips). AGH should implement similar checksums or sequence counters in its SQLite event store.

### 5.8 Hook-Aware Loop Guards

Stop hooks that return blocking errors cause the model to retry, but API error responses bypass stop hooks entirely. Without this, errors create death spirals: `error -> hook blocking -> retry -> error -> ...`. AGH's hook system must have the same bypass for error states.

### 5.9 Subagent Abort Controller Isolation

Async subagents get unlinked abort controllers so they run independently. Sync subagents share the parent's controller so parent cancel kills child. AGH should implement the same dual strategy for its session spawning.

### 5.10 Diminishing Returns Detection

The token budget system's `DIMINISHING_THRESHOLD = 500` check (if the model produces fewer than 500 new tokens for 3+ consecutive continuation nudges, stop early) prevents wasted compute. AGH should adopt similar heuristics for its agentic loop termination.
