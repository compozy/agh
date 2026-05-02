# GoClaw Agent Loop Architecture Analysis

**Comprehensive Deep-Dive into AGH Reference Implementation**

This document analyzes the core agent execution loop in GoClaw (internal/agent/), the reference implementation for the AGH (Artificial General Hivemind) project. The goal is to extract architectural patterns, execution flow, and implementation techniques that AGH can benefit from.

---

## 1. CORE ARCHITECTURE: Think → Act → Observe Loop

### 1.1 Entry Point: The Run Request Handler

**File:** `loop_run.go`

The agent execution begins with `Run(ctx context.Context, req RunRequest)`. This is the blocking entry point for processing a single user message.

**Core Idea:** Single-message processing with full lifecycle management including tracing, event emission, and error handling. The request encapsulates all routing metadata (delegation context, team scope, workspace channel), user identity, media attachments, and optional model/provider overrides.

**Key Pattern - Trace Lifecycle:**

```go
// Pre-generate root span ID so child spans can reference it
agentSpanID = store.GenNewID()
ctx = tracing.WithParentSpanID(ctx, agentSpanID)

// Emit agent span start
l.emitAgentSpanStart(ctx, agentSpanID, runStart, req.Message, agentSpanOpts...)

// V3 pipeline path (always enabled)
result, err := l.runViaPipeline(ctx, req)

// Finalize span with result/error
if err != nil {
  l.emitAgentSpanEnd(ctx, agentSpanID, runStart, nil, err)
} else {
  l.emitAgentSpanEnd(ctx, agentSpanID, runStart, result, nil)
}
```

**AGH Benefit:** Tracing should be first-class and baked into the loop foundation. Pre-generate span IDs upfront so all downstream operations can nest under a deterministic parent. Use context propagation for per-run metadata.

---

### 1.2 Context Injection: The Foundation Layer

**File:** `loop_context.go`

Before the main loop starts, all execution context is injected via `injectContext()`. This is a pure function that enriches the request context with:

- Agent identity (UUID + key dual identity)
- Tenant + user scoping
- Workspace resolution (user/chat/team layers)
- Tool configuration (per-agent, per-tenant overrides)
- Security guards (input validation, message truncation)

**Core Idea:** Context injection is a single checkpoint where all per-run state is immutably captured. This prevents concurrent runs from interfering with each other's tool execution context.

**Key Pattern - Layered Workspace Resolution:**

```go
// Layer order: tenant → team → user/chat
// ResolveWorkspace applies transformations in sequence
effectiveWorkspace := tools.ResolveWorkspace(l.dataDir,
  tools.TenantLayer(tenantID, tenantSlug),
  tools.TeamLayer(team.ID),
  tools.UserChatLayer(userID, isShared),
)
```

**Key Pattern - Dual Identity for Agent:**

```go
// DB PKs + foreign keys use UUID
agentUUID := store.WithAgentID(ctx, l.agentUUID)

// Logs, paths, filesystem use agent_key
agentKey := store.WithAgentKey(ctx, l.id)

// Tools routing uses agent_key to disambiguate which agent's spawn/delegate targets
ctx = tools.WithToolAgentKey(ctx, l.id)
```

**Key Pattern - Credential User Resolution:**

```go
// UserID stays unchanged (session/workspace scoping)
ctx = store.WithUserID(ctx, req.UserID)

// CredentialUserID is resolved separately for per-user features (MCP, SecureCLI)
credUserID := l.resolveCredentialUserID(ctx, *req)
if credUserID != "" && credUserID != req.UserID {
  ctx = store.WithCredentialUserID(ctx, credUserID)
}
```

**AGH Benefit:**

1. Separate workspace resolution from tool execution — make it a pure function with layered configuration (tenant/team/user/chat)
2. Dual identity pattern prevents silent scope leaks (UUID for DB/FK, key for paths/logs/UI)
3. Credential user resolution enables per-user authentication (MCP, SSH keys, cloud APIs) independent of session identity

---

### 1.3 The V3 Pipeline: 8-Stage Execution Flow

**File:** `loop_pipeline_adapter.go`

All agents use the v3 pipeline (the v2 loop was removed). The pipeline is a composable sequence of stages:

**Pipeline Stages:**

1. **Context** – Inject per-run context (loop_context.go)
2. **History** – Load session history, apply memory injection
3. **Prompt** – Build system prompt, assemble message list
4. **Think** – Call LLM, parse tool calls
5. **Act** – Execute tools in parallel, handle results
6. **Observe** – Update conversation state, drain injection channel
7. **Memory** – Flush episodic memory before compaction
8. **Summarize** – Compact old history, preserve recent context

**Key Pattern - Dependency Injection:**

```go
deps := pipeline.PipelineDeps{
  TokenCounter: l.tokenCounter,
  EventBus: l.domainBus,
  Config: pipeline.PipelineConfig{
    MaxIterations: maxIter,
    MaxToolCalls: l.maxToolCalls,
    ContextWindow: l.contextWindow,
  },

  // Callbacks for each stage
  InjectContext: cb.injectContext,
  LoadSessionHistory: cb.loadSessionHistory,
  BuildMessages: cb.buildMessages,
  CallLLM: cb.callLLM,
  ExecuteToolCall: cb.executeToolCall,
  PruneMessages: cb.pruneMessages,
  // ... more callbacks
}
p := pipeline.NewDefaultPipeline(deps)
result, err := p.Run(ctx, state)
```

**AGH Benefit:**

1. Dependency injection makes the loop testable and composable
2. Callbacks allow loop logic to live in agent/ while pipeline lives in pipeline/
3. Per-model context window resolution happens at run-time via modelRegistry
4. Token counting (tiktoken) is pluggable and used for context pruning accuracy

---

## 2. MESSAGE HISTORY MANAGEMENT

### 2.1 History Construction: Context Files + Session History + Current Message

**File:** `loop_history.go`

The `buildMessages()` function assembles the full LLM prompt by:

1. Building system prompt (via BuildSystemPrompt)
2. Injecting context files (SOUL.md, IDENTITY.md, BOOTSTRAP.md, etc.)
3. Applying history limits (last N user turns)
4. Sanitizing history (tool pairing repair)
5. Injecting current user message

**Core Idea:** The history pipeline is strictly sequential and stateless. Each stage transforms the message list, with the last 3-stage group (limit → sanitize → append current) happening at request time.

**Key Pattern - History Sanitization:**

```go
// limitHistoryTurns: keep last N user turns + all associated assistant/tool
trimmed := limitHistoryTurns(history, historyLimit)

// sanitizeHistory: repair tool_use/tool_result pairing
sanitized, droppedCount := sanitizeHistory(trimmed)
if droppedCount > 0 {
  // Persist cleaned history back to prevent re-triggering on next request
  l.sessions.SetHistory(ctx, sessionKey, sanitized)
}

// Final message list
messages = append(messages, sanitized...)
messages = append(messages, providers.Message{
  Role: "user",
  Content: userMessage,
})
```

**Key Pattern - Orphan Tool Message Repair:**

```go
// Drops leading tool messages (no preceding assistant with tool_calls)
start := 0
for start < len(msgs) && msgs[start].Role == "tool" {
  dropped++
  start++
}

// Dedup tool call IDs that were persisted as duplicates before uniquifyToolCallIDs
// Maps origID → []newID so multiple results for same orig can pair correctly
idQueue := make(map[string][]string)
for j := range msg.ToolCalls {
  origID := msg.ToolCalls[j].ID
  newID := origID
  if globalSeen[origID] {
    newID = fmt.Sprintf("%s_dedup_%d", origID, j)
  }
  msg.ToolCalls[j].ID = newID
  idQueue[origID] = append(idQueue[origID], newID)
}

// Synthesize missing tool results with placeholder
for _, tc := range msg.ToolCalls {
  if expectedIDs[tc.ID] {
    result = append(result, providers.Message{
      Role: "tool",
      Content: "[Tool result missing — session was compacted]",
      ToolCallID: tc.ID,
    })
  }
}
```

**AGH Benefit:**

1. Sanitize on read, not on write — repair history lazily at request time
2. Track dropped count and re-persist to DB so the same repairs don't repeat
3. Merge consecutive same-role messages to satisfy LLM strict alternation requirement
4. Tool call ID deduping ensures cross-turn uniqueness without history rewriting

---

### 2.2 Context File Resolution and Bootstrap

**File:** `loop_context.go`, `loop_history.go`

Context files (BOOTSTRAP.md, SOUL.md, IDENTITY.md, USER.md) come from two sources:

- **Base context**: Agent-level files (resolver-injected, auto-generated delegation info)
- **Per-user context**: User-specific files (seeded on first request, cached per Loop instance)

**Core Idea:** Lazy seeding + in-memory fallback ensure bootstrap always works even if DB writes fail (e.g., SQLITE_BUSY). Fallback is used once, then cleared so subsequent requests read from DB.

**Key Pattern - Lazy User Setup:**

```go
// sync.Map tracks (workspace, seeded, fallbackBootstrap) per user per Loop instance
setup := l.getOrCreateUserSetup(ctx, req.UserID, req.Channel, isTeamSession, channelMeta)

if !isTeamSession && l.ensureUserProfile != nil && l.seedUserFiles != nil {
  // Preferred: separate profile + seed callbacks
  ws, isNew, err := l.ensureUserProfile(ctx, l.agentUUID, userID, l.workspace, channel)
  if err := l.seedUserFiles(ctx, l.agentUUID, userID, l.agentType, isNew, channelMeta); err != nil {
    // Seeding failed → inject embedded templates in-memory
    setup.fallbackBootstrap = bootstrap.EmbeddedUserFiles(l.agentType)
  } else if l.cacheInvalidate != nil {
    // Invalidate context file cache so LoadContextFiles sees newly seeded files
    l.cacheInvalidate(l.agentUUID, userID)
  }
}

// Merge fallback into contextFiles on first request (and clear after use)
if val, ok := l.userSetups.Load(userID); ok {
  if fb := val.(*userSetup).fallbackBootstrap; len(fb) > 0 {
    contextFiles = l.mergeContextFallback(contextFiles, fb)
    val.(*userSetup).fallbackBootstrap = nil // clear after first use
  }
}
```

**AGH Benefit:**

1. Separate profile creation from file seeding — allows different retry/caching strategies
2. In-memory fallback bootstrap ensures first turn never blocks on DB write
3. Per-instance user setup cache (sync.Map) avoids N+1 DB queries for repeated calls
4. Cache invalidation callback bridges raw agentStore writes with ContextFileInterceptor cache

---

## 3. SYSTEM PROMPT CONSTRUCTION

### 3.1 Dynamic System Prompt Building

**File:** `systemprompt.go`, `systemprompt_sections.go`, `prompt_builder_impl.go`

The system prompt is built dynamically at request time based on:

- Agent identity, model, workspace, channel type
- Tool availability (filtered by orchestration mode)
- Skills summary + pinned skills
- Context files (SOUL.md, IDENTITY.md, TEAM.md)
- Sandbox/execution environment
- Per-provider contributions (thinking budget, extended reasoning, etc.)

**Core Idea:** System prompt is split at a cache boundary marker to separate stable (agent config) from dynamic (per-turn) content. Anthropic's provider uses this to apply cache_control to the stable section.

**Key Pattern - Cache Boundary:**

```go
const CacheBoundaryMarker = "<!-- GOCLAW_CACHE_BOUNDARY -->"

// Everything before marker: cached (agent config, skills, context files)
// Everything after marker: not cached (runtime channel, team members, user info)
systemPrompt := stableSystemPrompt + CacheBoundaryMarker + dynamicSystemPrompt
```

**Key Pattern - Prompt Mode Resolution (3-layer):**

```go
// Layer 1: Runtime override (per-request)
if runtimeOverride != "" {
  return runtimeOverride
}

// Layer 2a: Session auto-detect
if bootstrap.IsHeartbeatSession(sessionKey) {
  return minMode(configMode, PromptMinimal)
}
if bootstrap.IsSubagentSession(sessionKey) || bootstrap.IsCronSession(sessionKey) {
  return minMode(configMode, PromptTask)
}

// Layer 3: Agent config
if configMode != "" {
  return configMode
}

// Layer 4: Default
return PromptFull
```

**Key Pattern - Tool Filtering by Orchestration Mode:**

```go
// spawn: only self-clone (hide delegate + team_tasks)
// delegate: allows inter-agent delegation (hide team_tasks)
// team: full orchestration (no hiding)
orchModeDenyTools := func(mode OrchestrationMode) map[string]bool {
  switch mode {
  case ModeSpawn:
    return map[string]bool{"delegate": true, "team_tasks": true}
  case ModeDelegate:
    return map[string]bool{"team_tasks": true}
  default:
    return nil
  }
}
```

**AGH Benefit:**

1. Cache boundary marker allows partial prompt caching even with dynamic team/user sections
2. 4-layer prompt mode resolution (runtime > auto-detect > config > default) covers all use cases
3. Orchestration mode gating prevents tool misuse (spawn agents can't delegate to non-existent links)
4. Per-provider contributions enable model-specific optimizations (thinking budget, extended reasoning)

---

## 4. TOOL EXECUTION AND LOOP DETECTION

### 4.1 Tool Loop Detection: Multi-Level Defense

**File:** `toolloop.go`

The agent loop implements three independent loop detectors to catch different failure modes:

**1. Identical Arguments + Identical Results (Same Tool):**

```go
// Detects: tool called N times with same args → same result
// Warning threshold: 3, Critical: 5
if noProgressCount >= toolLoopCriticalThreshold {
  rs.loopKilled = true
  rs.finalContent = "I was unable to complete this task — I got stuck repeatedly calling " +
    toolName + " without making progress."
  return toolMsg, nil, toolResultBreak
}
```

**2. Read-Only Streak with Uniqueness Tracking:**

```go
// Detects: consecutive read-only tools (no write/edit/spawn)
// Stuck mode (unique ratio ≤ 0.6): warn 8, kill 12
// Exploration mode (unique ratio > 0.6): warn 24, kill 36

readOnlyRatio := float64(uniqueCount) / float64(readOnlyStreak)
if readOnlyRatio > readOnlyUniquenessThreshold {
  // Exploration: agent reading many unique files
  if readOnlyStreak >= readOnlyExplorationCritical {
    return "critical", "CRITICAL: N consecutive read-only tool calls (M unique files). Stopping..."
  }
} else {
  // Stuck mode: agent re-reading same files
  if readOnlyStreak >= readOnlyStreakCritical {
    return "critical", "CRITICAL: N consecutive read-only (only M unique)..."
  }
}
```

**3. Same Tool, Different Arguments, Identical Results:**

```go
// Detects: tool.read_file(path1) → same result, tool.read_file(path2) → same result
// Warning threshold: 4, Critical: 6
if count >= sameResultCritical {
  return "critical", fmt.Sprintf(
    "CRITICAL: %s returned identical results %d times (with different arguments).",
    toolName, count)
}
```

**Key Pattern - Deterministic Tool Call Hashing:**

```go
// Sorted JSON serialization ensures deterministic dedup
func hashToolCall(toolName string, args map[string]any) string {
  keys := make([]string, 0, len(args))
  for k := range args {
    keys = append(keys, k)
  }
  sort.Strings(keys) // stable ordering

  parts := make([]string, len(keys))
  for i, k := range keys {
    parts[i] = fmt.Sprintf("%q:%s", k, stableJSON(args[k]))
  }
  return "{" + strings.Join(parts, ",") + "}"
}
```

**AGH Benefit:**

1. Three-layer loop detection catches different failure modes (stuck, exploring, same-result)
2. Uniqueness ratio distinguishes exploration from loops
3. Warn before kill: inject system message so agent can adapt before forced break
4. Deterministic hashing with sorted keys ensures portable loop detection across implementations

---

### 4.2 Tool Result Processing

**File:** `loop_tools.go`

After tool execution, `processToolResult()` is a pure function that:

1. Records tool call + result in loop detector
2. Collects media from result
3. Emits tool_result event
4. Checks for prompt injection in web tool results
5. Returns warning messages + action signal (continue/warn/break)

**Key Pattern - Three-Phase Detection:**

```go
toolMsg, warningMsgs, action := l.processToolResult(ctx, rs, req, emitRun, tc,
  registryName, result, hadBootstrap)

// Phase 1: Same-tool same-args same-result
if level, msg := rs.loopDetector.detect(registryName, argsHash); level != "" {
  if level == "critical" {
    return toolMsg, nil, toolResultBreak  // hard stop
  }
  warningMsgs = append(warningMsgs, msg)  // inject warning, continue
}

// Phase 2: Same tool different results
if rh := hashResult(result.ForLLM); rh != "" {
  if level, msg := rs.loopDetector.detectSameResult(registryName, rh); level != "" {
    if level == "critical" {
      return toolMsg, nil, toolResultBreak
    }
    warningMsgs = append(warningMsgs, msg)
  }
}

// Phase 3: Read-only streak (checked between iterations, not here)
if l.checkReadOnlyStreak(rs, req) ... // called at iteration boundary
```

**AGH Benefit:**

1. Pure function allows testing in isolation
2. Three-phase approach (same-args, same-result, read-only) catches overlapping failure modes
3. Return action signal so caller decides break vs continue
4. Warning messages injected into conversation allow agent to self-correct

---

## 5. SANITIZATION AND SECURITY

### 5.1 Input Guard: Prompt Injection Detection

**File:** `input_guard.go`

The InputGuard scans user messages for known injection patterns. Action is configurable:

- "log": info-level (quiet)
- "warn": warning-level (default)
- "block": reject message with error
- "off": disable entirely

**Patterns:**

```go
{
  name: "ignore_instructions",
  pattern: `(?i)ignore\s+(all\s+)?(previous|prior|above|earlier|preceding)\s+(instructions?|rules?|prompts?|directives?|guidelines?)`
},
{
  name: "role_override",
  pattern: `(?i)(you are now|from now on you are|pretend you are|act as if you are|imagine you are)\s+`
},
{
  name: "system_tags",
  pattern: `(?i)</?system>|\[SYSTEM\]|\[INST\]|<<SYS>>|<\|im_start\|>system`
},
// ... more patterns
```

**AGH Benefit:**

1. Detection-only by default (warn action) — doesn't break legitimate use cases
2. Configurable action levels allow security/usability trade-offs
3. Web tool results are scanned too (scanWebToolResult)
4. Per-tenant or per-agent overrides via configuration

---

### 5.2 Output Sanitization: Comprehensive Pipeline

**File:** `sanitize.go`

Before sending to user, assistant content is sanitized through 8 stages:

1. **Strip garbled tool-call XML** (DeepSeek, GLM emit `<function_calls>` as text)
2. **Strip downgraded tool call text** (`[Tool Call: ...]`, `[Tool Result ...]`)
3. **Strip thinking/reasoning tags** (`<thinking>`, `<antThinking>`, etc.)
4. **Strip `<final>` tags** (keep content)
5. **Strip echoed [System Message] blocks** (LLM hallucinations)
6. **Collapse duplicate blocks** (repeated paragraphs)
7. **Strip MEDIA: paths** (delivered separately)
8. **Strip leading blank lines**

**Key Pattern - Line-Based vs Regex Scanning:**

```go
// Fast pre-check: look for indicator strings
if !strings.Contains(content, "[Tool Call:") &&
   !strings.Contains(content, "[Tool Result") &&
   !strings.Contains(content, "[Historical context:") {
  return content  // short-circuit
}

// Detailed scan: walk lines
lines := strings.Split(content, "\n")
var result []string
skipping := false
for _, line := range lines {
  if strings.HasPrefix(strings.TrimSpace(line), "[Tool Call:") {
    skipping = true
    continue
  }
  if skipping && strings.TrimSpace(line) == "" {
    skipping = false  // empty line ends block
    continue
  }
  if !skipping {
    result = append(result, line)
  }
}
```

**Key Pattern - Config Leak Detection (Predefined Agents):**

```go
// Only for predefined agents (l.agentType == "predefined")
// Strip code blocks before checking (mentions in code are architectural, not leaks)
plain := stripMarkdownCode(content)

// Count distinct leaked files
hits := 0
for _, name := range configLeakFileNames {  // SOUL.md, IDENTITY.md, AGENTS.md, etc.
  if strings.Contains(plain, name) {
    hits++
  }
}

// If 3+ distinct files mentioned → replace entire response
if hits >= 3 {
  return "🔒 Security check not passed."
}
```

**AGH Benefit:**

1. Sanitization is domain-specific: different models emit different garbage patterns
2. Line-based scanning for downgraded tool calls (regex can't handle multi-line properly)
3. Fast pre-checks (indicator string lookup) before expensive regex matching
4. Config leak detection prevents predefined agents from dumping internal config

---

## 6. CONTEXT PRUNING AND COMPACTION

### 6.1 Context Pruning: Two-Pass Approach

**File:** `pruning.go`

When context window usage exceeds threshold, context pruning reduces old tool results while preserving recent assistant messages:

**Pass 1: Soft Trim (head + tail):**

```go
// Find cutoff: protect last N assistant messages
cutoffIndex := findAssistantCutoff(msgs, settings.keepLastAssistants)

// Check: if ratio < softTrimRatio, skip pruning
ratio := float64(totalTokens) / float64(tokenWindow)
if ratio < settings.softTrimRatio {
  return msgs
}

// Soft trim long tool results: keep head + tail, drop middle
if msgTokens > trimThreshold {
  head := takeHead(msg.Content, settings.softTrimHeadChars)
  tail := takeTail(msg.Content, settings.softTrimTailChars)
  msg.Content = fmt.Sprintf("%s\n...\n%s\n\n[Tool result trimmed: kept first %d chars and last %d chars of %d chars.]",
    head, tail, headChars, tailChars, msgChars)
}
```

**Pass 2: Hard Clear (replace with placeholder):**

```go
// Only if ratio still > hardClearRatio after soft trim
if ratio < settings.hardClearRatio || !settings.hardClearEnabled {
  return output
}

// Skip media tools (read_image, read_document, etc.) — they contain irreplaceable vision descriptions
if mediaToolNames[toolCallNames[msg.ToolCallID]] {
  continue
}

// Replace entire tool result with placeholder
output[idx] = providers.Message{
  Role: msg.Role,
  Content: settings.hardClearPlaceholder,
  ToolCallID: msg.ToolCallID,
}
```

**Key Pattern - Token Counting Accuracy:**

```go
type pruningEstimator struct {
  counter tokencount.TokenCounter  // tiktoken if available
  model   string
}

func (e *pruningEstimator) estimateTokens(content string) int {
  if e.counter != nil {
    return e.counter.Count(e.model, content)  // accurate tiktoken
  }
  return utf8.RuneCountInString(content)  // fallback: rune count
}
```

**AGH Benefit:**

1. Two-pass approach (soft trim before hard clear) preserves important tail content
2. Media tools get higher soft-trim budget (8K chars) because vision descriptions are irreplaceable
3. Token counting is pluggable — fallback to rune count when tiktoken unavailable
4. Protect last N assistant messages (don't prune recent thinking)

---

### 6.2 Compaction: In-Memory History Summarization

**File:** `loop_compact.go`, `loop_history_sanitize.go`

When session history exceeds token threshold, compaction summarizes old messages:

```go
// Split: summarize old messages (70%), keep recent (30%)
keepCount := 4  // configurable
splitIdx := len(messages) - keepCount

// Walk backward to find clean boundary (avoid splitting tool_use/tool_result pairs)
for splitIdx > 0 {
  m := messages[splitIdx]
  if m.Role == "tool" || (m.Role == "assistant" && len(m.ToolCalls) > 0) {
    splitIdx--
    continue
  }
  break
}

// Call LLM to summarize old messages
resp, err := l.provider.Chat(sctx, providers.ChatRequest{
  Messages: []providers.Message{{
    Role: "user",
    Content: compactionSummaryPrompt + oldMessagesText,
  }},
  Model: l.model,
  Options: map[string]any{"max_tokens": 1024, "temperature": 0.3},
})

// Build result: summary + recent messages
summary := providers.Message{
  Role: "user",
  Content: "[Summary of earlier conversation]\n" + resp.Content,
}
result := append([]providers.Message{summary}, messages[splitIdx:]...)
```

**Compaction Summary Prompt Preservation Rules:**

```
MUST PRESERVE:
- Active tasks and their current status (in-progress, blocked, pending)
- Pending subagent tasks (IDs, labels, statuses)
- Pending team task results awaiting delivery
- Any "waiting for..." state
- Batch operation progress (e.g., "5/17 items completed")
- The last thing the user requested
- Decisions made and their rationale
- TODOs, open questions, and constraints
- Commitments or follow-ups promised

IDENTIFIER PRESERVATION:
- Preserve all opaque identifiers exactly as written (no reconstruction)
- UUIDs, hashes, IDs, tokens, API keys, hostnames, IPs, ports, URLs, file names
```

**AGH Benefit:**

1. Compaction preserves pending task IDs and state (crucial for delegation)
2. Summarization is parameterized (temperature 0.3 for consistency)
3. MediaRefs are preserved from compacted messages (media links don't disappear)
4. Summary prefix helps LLM understand it's reading historical context, not recent events

---

## 7. MEMORY MANAGEMENT

### 7.1 Memory Flush: Pre-Compaction Episodic Capture

**File:** `memoryflush.go`

Before automatic compaction, a memory flush turn runs to capture durable memories to disk:

```go
// Build flush messages: system prompt + history + flush prompt
systemPrompt := BuildSystemPrompt(flushPromptConfig)
systemPrompt += "\n\n" + flushSystemPrompt  // "capture durable memories to memory/YYYY-MM-DD.md"

messages := append(messages, providers.Message{
  Role: "system",
  Content: systemPrompt,
})

if summary != "" {
  messages = append(messages, providers.Message{
    Role: "user",
    Content: "[Previous conversation summary]\n" + summary,
  })
}

messages = append(messages, providers.Message{
  Role: "user",
  Content: flushPrompt,  // "Append durable memories... If nothing, reply with NO_REPLY"
})

resp, err := l.provider.Chat(ctx, providers.ChatRequest{
  Messages: messages,
  Model: l.model,
})
```

**Deduplication Guard:**

```go
// Skip if already flushed in this compaction cycle
compactionCount := l.sessions.GetCompactionCount(ctx, sessionKey)
lastFlushAt := l.sessions.GetMemoryFlushCompactionCount(ctx, sessionKey)
if lastFlushAt >= 0 && lastFlushAt == compactionCount {
  return false  // already flushed
}
```

**AGH Benefit:**

1. Memory flush happens inside maybeSummarize's per-session lock (no concurrent duplicates)
2. Dedup by compaction cycle prevents redundant flushes
3. NO_REPLY detection suppresses empty flush output
4. Extractive memory fallback (regex) saves context when LLM flush returns nothing

---

### 7.2 Extractive Memory Fallback

**File:** `extractive_memory.go`

If LLM-based flush fails or returns NO_REPLY, extractive memory patterns capture key information:

```go
// Pattern: decisions
reDecision = `(?i)(?:decided\s+to|let'?s\s+go\s+with|approved|agreed\s+on|chose|we'?ll\s+use)\s+.{5,120}`

// Pattern: user preferences
rePreference = `(?i)(?:I\s+prefer|don'?t\s+do|always\s+|never\s+|I\s+want|please\s+remember)\s+.{5,120}`

// Pattern: technical facts
reTechFact = `(?i)(?:the\s+API\s+is|endpoint\s+is|version\s+is|uses?\s+\S+\s+for)\s+.{3,120}`

// URLs + file paths + dates

// Output: structured memory file
## Extracted Context (auto-saved before compaction)

### Decisions
- [matched decisions...]

### Key Facts
- [matched facts...]

### User Preferences
- [matched preferences...]
```

**AGH Benefit:**

1. Regex extraction is fast and doesn't require LLM calls
2. Structured output (decisions, facts, preferences) is easier to search than plain text
3. Safety net when LLM flush fails or returns NO_REPLY
4. Identifier preservation (URLs, dates, paths) maintains accuracy

---

## 8. MEDIA HANDLING

### 8.1 Media Persistence and Sanitization

**File:** `media.go`, `loop_media.go`

Incoming media files are:

1. **Sanitized** (images cleaned of metadata/malware)
2. **Persisted** to per-user `.uploads/` directory
3. **Tracked** via MediaRefs with MIME type and kind (image/document/audio/video)

**Key Pattern - Persistent Workspace Storage:**

```go
uploadsDir := filepath.Join(workspace, ".uploads")

// Verify .uploads is real directory (not symlink) to prevent symlink attacks
if fi, err := os.Lstat(uploadsDir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
  slog.Warn("media: .uploads is a symlink, refusing to use")
  return nil
}

// Sanitize images before storage
srcPath := f.Path
if kind == "image" {
  sanitized, err := SanitizeImage(f.Path)
  if err == nil {
    srcPath = sanitized
  }
}

// Traversal guard: ensure dstPath is inside uploadsDir
cleanDst := filepath.Clean(dstPath) + string(os.PathSeparator)
cleanUploads := filepath.Clean(uploadsDir) + string(os.PathSeparator)
if !strings.HasPrefix(cleanDst, cleanUploads) {
  slog.Warn("media: refusing to persist outside uploadsDir")
  return nil
}
```

**Key Pattern - MediaRef Tracking:**

```go
refs = append(refs, providers.MediaRef{
  ID: id,
  MimeType: mime,
  Kind: kind,  // "image", "document", "audio", "video"
  Path: dstPath,
})

// Preserved across history compaction so media links don't break
msg.MediaRefs = append(msg.MediaRefs, refs...)
```

**AGH Benefit:**

1. Workspace isolation: media stored in per-user folder prevents cross-user access
2. Symlink detection prevents symlink attacks
3. Image sanitization removes metadata + embedded content
4. MediaRef preservation across compaction maintains media continuity

---

## 9. ROUTER AND SESSION MANAGEMENT

### 9.1 Agent Router: Caching and TTL-Based Expiration

**File:** `router.go`

The Router manages multiple agent Loop instances with caching + TTL-based invalidation:

```go
type Router struct {
  agents map[string]*agentEntry  // agentKey → Agent
  mu sync.RWMutex
  activeRuns sync.Map            // runID → *ActiveRun
  sessionRuns sync.Map           // sessionKey → runID (secondary index)
  agentActivity sync.Map         // sessionKey → *AgentActivityStatus
  resolver ResolverFunc          // lazy creation from DB
  ttl time.Duration              // default 10 minutes
}

// Get with TTL-based expiration
func (r *Router) Get(ctx context.Context, agentID string) (Agent, error) {
  cacheKey := agentCacheKey(ctx, agentID)  // tenant:agentID

  r.mu.RLock()
  entry, ok := r.agents[cacheKey]
  resolver := r.resolver
  r.mu.RUnlock()

  if ok && time.Since(entry.cachedAt) < r.ttl {
    return entry.agent, nil  // cache hit
  }

  // Cache miss or expired → resolver (DB lookup + Loop construction)
  ag, err := resolver(ctx, agentID)
  if err != nil {
    return nil, err
  }

  // Store in cache under canonical key (tenantID:agentKey)
  r.mu.Lock()
  r.agents[cacheKey] = &agentEntry{agent: ag, cachedAt: time.Now()}
  r.mu.Unlock()

  return ag, nil
}
```

**Canonicalization:** Cache key is always `tenantID:agentKey` (never raw UUID) so all callers hit the cache.

**AGH Benefit:**

1. TTL-based expiration is safety net for multi-instance deployments
2. Canonicalization ensures UUIDs still work (via resolver) but convert to agent_key on storage
3. Per-session activity tracking enables force-abort and status queries
4. Secondary index (sessionKey → runID) allows O(1) IsSessionBusy checks

---

## 10. ORCHESTRATION AND DELEGATION

### 10.1 Orchestration Mode Resolution

**File:** `orchestration_mode.go`

The orchestration mode determines which inter-agent tools are available:

```go
type OrchestrationMode string

const (
  ModeSpawn    = "spawn"      // self-clone only
  ModeDelegate = "delegate"   // inter-agent delegation
  ModeTeam     = "team"       // full team orchestration
)

// Resolve by priority: team > delegate > spawn
func ResolveOrchestrationMode(ctx context.Context, agentID uuid.UUID,
  teamStore store.TeamStore, linkStore store.AgentLinkStore) OrchestrationMode {

  // Team membership takes priority
  if teamStore != nil {
    if team, err := teamStore.GetTeamForAgent(ctx, agentID); err == nil && team != nil {
      return ModeTeam
    }
  }

  // Delegate links
  if linkStore != nil {
    if targets, err := linkStore.DelegateTargets(ctx, agentID); err == nil && len(targets) > 0 {
      return ModeDelegate
    }
  }

  return ModeSpawn
}

// Tool visibility gating
func orchModeDenyTools(mode OrchestrationMode) map[string]bool {
  switch mode {
  case ModeSpawn:
    return map[string]bool{"delegate": true, "team_tasks": true}
  case ModeDelegate:
    return map[string]bool{"team_tasks": true}
  default:  // ModeTeam
    return nil
  }
}
```

**AGH Benefit:**

1. Clear hierarchy: team membership is strongest indicator
2. Delegate targets are injected into system prompt for discovery
3. Tool gating prevents agents from calling non-existent inter-agent features
4. Mode is resolved once per request, not on-demand

---

## 11. TRACING AND OBSERVABILITY

### 11.1 Structured Tracing Integration

**File:** `loop_run.go`

Tracing is integrated at the Loop.Run boundary:

```go
// Create trace (or reuse parent trace for announce runs)
if isChildTrace {
  // Announce: reuse parent trace, don't create new record
  traceID = req.ParentTraceID
  ctx = tracing.WithTraceID(ctx, traceID)
  agentSpanID = store.GenNewID()
  ctx = tracing.WithParentSpanID(ctx, agentSpanID)
} else if l.traceCollector != nil {
  // New trace
  traceID = store.GenNewID()
  trace := &store.TraceData{
    ID: traceID,
    RunID: req.RunID,
    SessionKey: req.SessionKey,
    Name: traceName,
    InputPreview: truncateStr(req.Message, previewMaxLen),
    Status: store.TraceStatusRunning,
    StartTime: time.Now().UTC(),
  }

  // Link to parent trace (delegation or team task)
  if delegateParent := tracing.DelegateParentTraceIDFromContext(ctx); delegateParent != uuid.Nil {
    trace.ParentTraceID = &delegateParent
  }

  l.traceCollector.CreateTrace(ctx, trace)

  // Notify gateway so it can associate traceID with active run
  if req.OnTraceCreated != nil {
    req.OnTraceCreated(traceID)
  }
}

// Emit agent span (covers entire run)
l.emitAgentSpanStart(ctx, agentSpanID, runStart, req.Message, agentSpanOpts...)

// ... v3 pipeline execution ...

// Finalize span
if err != nil {
  l.emitAgentSpanEnd(ctx, agentSpanID, runStart, nil, err)
} else {
  l.emitAgentSpanEnd(ctx, agentSpanID, runStart, result, nil)
}

// Safety net: ensure root traces are always finalized
defer func() {
  if !traceFinalized {
    l.traceCollector.FinishTrace(safeCtx, traceID, store.TraceStatusError,
      "trace finalized by safety net (likely panic or goroutine leak)", "")
  }
}()
```

**AGH Benefit:**

1. Dual trace modes (new vs child) support both standalone and delegated runs
2. OnTraceCreated callback bridges loop and gateway so force-abort can mark correct trace
3. Safety-net finalization ensures no orphaned traces on panic/leak
4. Span hierarchy (root agent span → child LLM/tool spans) enables drill-down debugging

---

## 12. FINALIZATION AND SESSION PERSISTENCE

### 12.1 Final Run Processing

**File:** `loop_finalize.go`

After the pipeline completes, `finalizeRun()` does post-loop processing:

```go
// 1. Sanitize final content
rs.finalContent = SanitizeAssistantContent(rs.finalContent)

// 2. Handle NO_REPLY (silent output)
isSilent := IsSilentReply(rs.finalContent)

// 3. Skill evolution postscript (if enabled)
if l.skillEvolve && rs.totalToolCalls >= l.skillNudgeInterval {
  rs.finalContent += "\n\n---\n_" + i18n.T(locale, i18n.MsgSkillNudgePostscript) + "_"
}

// 4. Fallback: ensure non-empty content
if rs.finalContent == "" {
  rs.finalContent = "..."
}

// 5. Append content suffix (e.g. image markdown for WS)
if req.ContentSuffix != "" {
  rs.finalContent += deduplicateMediaSuffix(rs.finalContent, req.ContentSuffix)
}

// 6. Build assistant message with output media refs
assistantMsg := providers.Message{
  Role: "assistant",
  Content: rs.finalContent,
  Thinking: rs.finalThinking,
}
for _, mr := range rs.mediaResults {
  assistantMsg.MediaRefs = append(assistantMsg.MediaRefs, providers.MediaRef{
    ID: filepath.Base(mr.Path),
    MimeType: mr.ContentType,
    Kind: kind,
    Path: mr.Path,
  })
}
rs.pendingMsgs = append(rs.pendingMsgs, assistantMsg)

// 7. Bootstrap cleanup
if hadBootstrap && userTurns >= bootstrapAutoCleanupTurns {
  if cleanErr := l.bootstrapCleanup(ctx, l.agentUUID, req.UserID); cleanErr != nil {
    slog.Warn("bootstrap auto-cleanup failed", "error", cleanErr)
  }
}

// 8. Flush messages to session atomically
for _, msg := range rs.pendingMsgs {
  l.sessions.AddMessage(ctx, req.SessionKey, msg)
}

// 9. Update metadata
l.sessions.UpdateMetadata(ctx, req.SessionKey, l.model, l.provider.Name(), req.Channel)
l.sessions.AccumulateTokens(ctx, req.SessionKey, int64(rs.totalUsage.PromptTokens), int64(rs.totalUsage.CompletionTokens))

// 10. Emit session.completed for consolidation pipeline
if l.domainBus != nil {
  l.domainBus.Publish(eventbus.DomainEvent{
    Type: eventbus.EventSessionCompleted,
    Payload: &eventbus.SessionCompletedPayload{
      SessionKey: req.SessionKey,
      MessageCount: len(history) + len(rs.pendingMsgs),
      TokensUsed: rs.totalUsage.PromptTokens + rs.totalUsage.CompletionTokens,
      CompactionCount: l.sessions.GetCompactionCount(ctx, req.SessionKey),
    },
  })
}

return &RunResult{
  Content: rs.finalContent,
  Thinking: rs.finalThinking,
  RunID: req.RunID,
  Iterations: rs.iteration,
  Usage: &rs.totalUsage,
  Media: rs.mediaResults,
  Deliverables: rs.deliverables,
  LoopKilled: rs.loopKilled,
}
```

**AGH Benefit:**

1. Bootstrap auto-cleanup runs once per turn cycle without requiring model intervention
2. NO_REPLY detection allows silent operations (subagent progress updates, heartbeats)
3. Session flush is atomic: all messages added together so DB snapshot is consistent
4. Domain event publishing triggers downstream consolidation (episodic → semantic memory)

---

## 13. TOKEN COUNTING AND ESTIMATION

### 13.1 Calibrated Token Estimation

**File:** `loop_history_sanitize.go`, `loop_utils.go`

Token estimation for compaction threshold uses calibration:

```go
// Use calibrated token estimation, adjusted for overhead
lastPT, lastMC := l.sessions.GetLastPromptTokens(ctx, sessionKey)
adjustedLastPT := max(lastPT - l.estimateOverhead(history, lastPT, lastMC), 0)
tokenEstimate := EstimateTokensWithCalibration(history, adjustedLastPT, lastMC)

// Estimate overhead (system prompt + tools + context files)
func (l *Loop) estimateOverhead(history []providers.Message, lastPromptTokens, lastMsgCount int) int {
  if lastPromptTokens <= 0 || lastMsgCount <= 0 {
    // No calibration data — use conservative default (20% of context)
    fallback := min(int(float64(l.contextWindow)*0.2), 40000)
    return fallback
  }

  // Overhead = total prompt tokens - estimated history tokens at calibration time
  count := min(lastMsgCount, len(history))
  historyEstAtCalibration := EstimateHistoryTokens(history[:count])
  overhead := max(lastPromptTokens - historyEstAtCalibration, 0)

  // Clamp to 40% of context window
  maxOverhead := int(float64(l.contextWindow) * 0.4)
  if overhead > maxOverhead {
    overhead = maxOverhead
  }
  return overhead
}
```

**Key Pattern - Calibration Persistence:**

```go
// After each run, store actual prompt tokens + message count
l.sessions.SetLastPromptTokens(ctx, req.SessionKey, rs.totalUsage.PromptTokens, msgCount)

// Next run uses calibration to estimate overhead more accurately
```

**AGH Benefit:**

1. Calibration improves estimate accuracy after first run
2. Overhead clamping (40% max) prevents over-aggressive compaction
3. Conservative fallback (20% or 40K tokens) when no historical data
4. Overhead = system + tools + context files, allowing history-only comparison against threshold

---

## 14. INTEGRATION POINTS AND CALLBACKS

### 14.1 Pipeline Dependency Injection

**File:** `loop_pipeline_adapter.go`, `loop_pipeline_callbacks.go`

The pipeline accepts callbacks for all major operations:

```go
type PipelineDeps struct {
  // Config
  Config pipeline.PipelineConfig

  // Token counting
  TokenCounter tokencount.TokenCounter

  // Stage callbacks
  InjectContext func(ctx, req) (context, error)
  LoadSessionHistory func(ctx, sessionKey) []Message
  ResolveWorkspace func(ctx, req) string
  LoadContextFiles func(ctx, userID) []ContextFile
  BuildMessages func(...) []Message

  BuildFilteredTools func(ctx) []Tool
  CallLLM func(ctx, messages, model) ChatResponse

  ExecuteToolCall func(ctx, tc, ...) *Result
  ProcessToolResult func(ctx, tc, result) (toolMsg, warnings, action)

  PruneMessages func(msgs, tokenWindow) []Message
  SanitizeHistory func(msgs) []Message
  CompactMessages func(ctx, msgs) []Message

  // Event callbacks
  EmitEvent func(event)
  EmitBlockReply func(content)

  // Finalization
  FlushMessages func(ctx, sessionKey, messages)
  UpdateMetadata func(ctx, sessionKey, model, provider, channel)
  MaybeSummarize func(ctx, sessionKey)
}
```

**AGH Benefit:**

1. All agent-specific logic lives in agent/callbacks
2. Pipeline is generic, reusable across multiple agent frameworks
3. Callbacks are individually testable
4. Easy to instrument/observe at each pipeline stage

---

## 15. KEY PATTERNS FOR AGH ADOPTION

### Pattern 1: Dual Identity (UUID + Key)

**Use case:** Agents, teams, tenants all have dual identities.

- **UUID:** Database primary key, foreign keys, domain events, OTel span attributes
- **Key:** Logs, filesystem paths, UI display, route matching

**Benefit:** Prevents silent scope leaks when moving between storage layers.

---

### Pattern 2: Lazy Setup with In-Memory Fallback

**Use case:** Bootstrap onboarding, user context file seeding.

**Flow:**

1. Fast path: check sync.Map cache
2. Slow path: DB calls (profile creation, file seeding)
3. Fallback: inject in-memory templates if DB fails
4. Cache invalidation: clear fallback after first use

**Benefit:** Resilience to transient DB errors; first turn never blocks on slow writes.

---

### Pattern 3: Two-Pass Context Reduction

**Use case:** Managing large context windows (pruning + compaction).

**Phase 1 (Soft Trim):** Keep head + tail, drop middle of long tool results.
**Phase 2 (Hard Clear):** Replace entire old tool results with placeholder.

**Benefit:** Preserves important error messages + summaries while reclaiming space.

---

### Pattern 4: Multi-Layer Loop Detection

**Use case:** Catching infinite loops without breaking legitimate exploration.

1. Same tool, same args, same result → kill
2. Read-only streak with uniqueness ratio → warn/kill
3. Same tool, different args, same result → warn/kill

**Benefit:** Catches overlapping failure modes; warns before killing to allow self-correction.

---

### Pattern 5: Config Boundary Markers

**Use case:** Enabling partial prompt caching.

**Marker:** `<!-- GOCLAW_CACHE_BOUNDARY -->`

**Split:**

- Before marker: stable (agent config, skills, context files) → cached
- After marker: dynamic (runtime channel, team members, user info) → not cached

**Benefit:** Reduces cache misses when dynamic content changes.

---

### Pattern 6: Deterministic Tool Call Hashing

**Use case:** Portable loop detection across implementations.

**Implementation:**

1. Sort argument keys alphabetically
2. Serialize to JSON with sorted keys
3. Hash with SHA-256, take first 16 bytes (32 hex chars)

**Benefit:** Same tool call → same hash across Go, TS, Python implementations.

---

### Pattern 7: Per-Session Lock for Concurrent Operations

**Use case:** Preventing duplicate memory flush/compaction for same session.

**Implementation:**

```go
muI, _ := l.summarizeMu.LoadOrStore(sessionKey, &sync.Mutex{})
sessionMu := muI.(*sync.Mutex)
if !sessionMu.TryLock() {
  return  // already running
}
defer sessionMu.Unlock()
```

**Benefit:** Non-blocking; next run will trigger compaction again if still needed.

---

### Pattern 8: Annotated Events with Routing Context

**Use case:** Broadcasting agent events to WebSocket clients with filtering.

**Structure:**

```go
type AgentEvent struct {
  Type string              // "run.started", "tool.call", "run.completed"
  AgentID string           // for agent-specific subscriptions
  RunID string             // for run-specific subscriptions

  // Routing context (omitted if not applicable)
  UserID string
  Channel string
  SessionKey string
  TenantID uuid.UUID

  // Delegation context
  DelegationID string
  TeamID string
  ParentAgentID string
}
```

**Benefit:** Clients can filter by agent/user/team/tenant without reimplementing routing logic.

---

## 16. ANTI-PATTERNS AND WHAT TO AVOID

1. **Don't persist tool loop history across sessions** — resets per run allow fresh starts
2. **Don't use raw UUID for filesystem paths** — use agent_key (human-readable, stable)
3. **Don't skip input guard on internal tools** — web_fetch/web_search need injection scanning
4. **Don't assume tool result is complete** — truncate at read time, not write time
5. **Don't prune media tool results** — vision descriptions can't be regenerated cheaply
6. **Don't sync memory flush across compaction cycles** — deduplicate by compaction count
7. **Don't merge role alternation issues silently** — log and persist so they don't recur

---

## 17. RECOMMENDED AGH IMPLEMENTATION ROADMAP

### Phase 1: Core Loop Foundation

- [ ] Implement dual identity pattern (UUID + key)
- [ ] Build context injection layer (workspace, tools, security guards)
- [ ] Set up v3 pipeline with dependency injection

### Phase 2: History & Memory Management

- [ ] History sanitization (tool pairing repair, deduplication)
- [ ] Lazy user setup with in-memory fallback
- [ ] Compaction with threshold-based triggers

### Phase 3: Safety & Observability

- [ ] Input guard (prompt injection detection)
- [ ] Output sanitization (8-stage pipeline)
- [ ] Structured tracing with span hierarchy

### Phase 4: Loop Protection

- [ ] Tool loop detection (3-level multi-pass)
- [ ] Context pruning (soft trim + hard clear)
- [ ] Slow tool timing with adaptive thresholds

### Phase 5: Advanced Features

- [ ] Memory flush (pre-compaction episodic capture)
- [ ] Extractive memory fallback (regex patterns)
- [ ] Orchestration mode resolution (spawn/delegate/team)

---

## Conclusion

GoClaw's agent loop is a comprehensive, battle-tested reference implementation with:

1. **Resilience**: Lazy seeding, in-memory fallbacks, safety-net finalization
2. **Observability**: Structured tracing, event broadcasting, activity status
3. **Safety**: Multi-layer loop detection, input guard, output sanitization
4. **Efficiency**: Calibrated token estimation, two-pass pruning, partial prompt caching
5. **Extensibility**: Dependency injection, callbacks per stage, pluggable token counters

AGH should adopt these patterns as foundational patterns, adapting them as needed for the specific orchestration and agent design goals. The dual identity pattern (UUID + key) and context injection layer are the most transferable and should be implemented first.
