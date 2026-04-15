# GoClaw Pipeline, Hooks, Middleware & Sandbox Architecture Analysis

**Scope:** Agent Operating System patterns for the AGH project  
**Reference:** `.resources/goclaw/` directory structure  
**Date:** 2026-04-15

---

## Table of Contents

1. [Pipeline Architecture](#pipeline-architecture)
2. [Hooks System Design](#hooks-system-design)
3. [Permission Model](#permission-model)
4. [Sandbox Patterns](#sandbox-patterns)
5. [Callback Wiring](#callback-wiring)
6. [Key Architectural Patterns](#key-architectural-patterns)

---

## Pipeline Architecture

### Overview

GoClaw implements a **staged pipeline execution model** for agent runs. The pipeline orchestrates message flow through 8 stages: setup (1), iteration (5), and finalize (1). Each stage is stateless; all mutable state lives in `RunState`.

**Files:** `internal/pipeline/pipeline.go`, `internal/pipeline/stage.go`, `internal/pipeline/run_state.go`

### Stage Execution Flow

```
Setup (once):
  └─ ContextStage
     ├─ Inject context (agent/tenant/user/workspace scoping)
     ├─ Resolve context window per provider/model
     ├─ Resolve workspace
     ├─ Load context files + session history
     ├─ Build system prompt + history
     ├─ Compute overhead tokens
     ├─ Enrich input media
     ├─ Inject team reminders
     └─ Auto-inject L0 memory context

Iteration Loop (MaxIterations):
  ├─ ThinkStage
  │  ├─ Inject iteration budget nudges (70%/90%)
  │  ├─ Build filtered tool definitions
  │  ├─ Call LLM (streaming or sync)
  │  ├─ Accumulate token usage
  │  ├─ Handle truncation retries (max 3)
  │  ├─ Uniquify tool call IDs
  │  └─ Flow control: BreakLoop if no tool calls
  │
  ├─ PruneStage
  │  ├─ Count history tokens vs budget
  │  ├─ Phase 1 (70%): soft prune via PruneMessages
  │  ├─ Phase 2 (100%): memory flush + LLM compaction
  │  ├─ Cache-TTL gate (per-session, provider-aware)
  │  └─ Flow control: AbortRun if still over budget post-compact
  │
  ├─ ToolStage
  │  ├─ Extract tool calls from LastResponse
  │  ├─ Parallel path: ExecuteToolRaw (I/O) → ProcessToolResult (mutation)
  │  ├─ Sequential fallback: ExecuteToolCall (I/O + mutation)
  │  ├─ Check exit conditions: loop kill, read-only streak, tool budget
  │  └─ Flow control: BreakLoop on exit condition
  │
  ├─ ObserveStage
  │  ├─ Drain injected messages from InjectCh
  │  ├─ Track block replies (intermediate tool-iteration responses)
  │  └─ Accumulate final content (final answer)
  │
  └─ CheckpointStage
     └─ Flush pending messages to session store every N iterations

Finalize (once, uses context.WithoutCancel):
  └─ FinalizeStage
     ├─ Sanitize final content
     ├─ Skill evolution postscript
     ├─ NO_REPLY detection
     ├─ Append content suffix with dedup
     ├─ Process + deduplicate media
     ├─ Build final assistant message with MediaRefs
     ├─ Flush remaining pending messages
     ├─ Update session metadata (token usage)
     ├─ Bootstrap cleanup
     ├─ Trigger summarization (async)
     ├─ Emit session.completed for consolidation pipeline
     ├─ Strip message directives
     └─ Suppress NO_REPLY if silent
```

### Core Stage Interface

```go
// Stage is stateless — all mutable state in RunState
type Stage interface {
  Name() string
  Execute(ctx context.Context, state *RunState) error
}

// StageWithResult controls pipeline flow
type StageWithResult interface {
  Stage
  Result() StageResult  // Continue | BreakLoop | AbortRun
}
```

### Exit Control Semantics

- **Continue:** Proceed to next stage (default)
- **BreakLoop:** Exit iteration loop gracefully; run remaining iteration stages, then finalize
- **AbortRun:** Exit immediately (error/kill); skip remaining stages, go to finalize

Pipeline enforces:

- AbortRun breaks inner stage loop immediately
- BreakLoop checked after all iteration stages complete
- Context cancellation (ctx.Err()) triggers AbortRun
- Finalize runs on `context.WithoutCancel(ctx)` for crash recovery

### RunState: Shared Mutable State

```go
type RunState struct {
  // Identity (immutable)
  Input     *RunInput
  Workspace *workspace.WorkspaceContext
  Model     string
  Provider  providers.Provider

  // Context enrichment from ContextStage
  Ctx       context.Context

  // Message buffer (3-tier: system/history/pending)
  Messages  *MessageBuffer

  // Per-stage substates
  Context   ContextState    // system prompt, overhead, memory section
  Think     ThinkState      // LLM response, token usage, truncation retries
  Prune     PruneState      // token budget tracking
  Tool      ToolState       // tool execution, loop detection, media
  Observe   ObserveState    // final content, block replies
  Compact   CompactState    // checkpoint flushes, compaction count
  Evolution EvolutionState  // skill nudges, team task tracking

  // Cross-cutting
  Iteration int
  RunID     string
  ExitCode  StageResult
}
```

### Message Buffer: 3-Tier Architecture

```go
type MessageBuffer struct {
  system  providers.Message   // system prompt (rebuilt per run)
  history []providers.Message // conversation history (persisted)
  pending []providers.Message // new messages this iteration (volatile)
}

// All() = [system] + history + pending  (used for LLM calls)
// FlushPending() = move pending → history + return flushed  (checkpoint/finalize)
// ReplaceHistory() = history = compacted msgs, pending = nil  (post-compact)
```

**Invariant:** history + pending are mutually exclusive with the LLM request — messages.All() includes both so LLM sees complete context.

### PipelineDeps: Callback Injection Surface

The pipeline receives ~50 callbacks bundled in `PipelineDeps`:

```go
type PipelineDeps struct {
  // Infrastructure
  TokenCounter     tokencount.TokenCounter
  EventBus         eventbus.DomainEventBus
  Config           PipelineConfig

  // Resolver callbacks
  ResolveContextWindow func(provider, model string) int
  EmitEvent            func(event any)
  AutoInject           func(ctx, userMessage, userID, recentContext string) (string, error)
  InjectContext        func(ctx context.Context, input *RunInput) (context.Context, error)

  // Context callbacks (ContextStage)
  LoadSessionHistory  func(ctx, sessionKey string) ([]providers.Message, string)
  ResolveWorkspace    func(ctx, input *RunInput) (*workspace.WorkspaceContext, error)
  LoadContextFiles    func(ctx, userID string) ([]bootstrap.ContextFile, bool)
  BuildMessages       func(ctx, input, history, summary) ([]providers.Message, error)
  EnrichMedia         func(ctx, state *RunState) error
  InjectReminders     func(ctx, input, msgs) []providers.Message

  // Think callbacks (ThinkStage)
  BuildFilteredTools  func(*RunState) ([]providers.ToolDefinition, error)
  CallLLM             func(ctx, state, req) (*providers.ChatResponse, error)
  UniqueToolCallIDs   func(calls, runID, iteration) []providers.ToolCall
  EmitBlockReply      func(content string)

  // Prune callbacks (PruneStage)
  PruneMessages       func(msgs, budget) ([]providers.Message, PruneStats)
  SanitizeHistory     func(msgs) ([]providers.Message, int)
  CompactMessages     func(ctx, msgs, model) ([]providers.Message, error)
  GetProviderCaps     func() providers.ProviderCapabilities
  GetPruningConfig    func() *config.ContextPruningConfig
  GetCacheTouch       func(sessionKey string) time.Time
  MarkCacheTouched    func(sessionKey string)

  // Memory flush callbacks
  RunMemoryFlush      func(ctx, state *RunState) error

  // Tool callbacks (ToolStage)
  ExecuteToolCall     func(ctx, state, tc) ([]providers.Message, error)
  ExecuteToolRaw      func(ctx, tc) (providers.Message, any, error)
  ProcessToolResult   func(ctx, state, tc, msg, rawData) []providers.Message
  CheckReadOnly       func(state) (*providers.Message, bool)

  // Observe callbacks
  DrainInjectCh       func() []providers.Message

  // Checkpoint callbacks
  FlushMessages       func(ctx, sessionKey, msgs) error

  // Finalize callbacks
  SkillPostscript     func(ctx, content, toolCount) string
  SanitizeContent     func(string) string
  StripMessageDirectives func(string) string
  DeduplicateMediaSuffix func(content, suffix) string
  IsSilentReply       func(content string) bool
  EmitSessionCompleted func(ctx, sessionKey, msgCount, tokensUsed, compactionCount)
  UpdateMetadata      func(ctx, sessionKey, usage) error
  BootstrapCleanup    func(ctx, state) error
  MaybeSummarize      func(ctx, sessionKey)
}
```

**Key pattern:** Callbacks are wired by the agent loop adapter (`loop_pipeline_adapter.go`), enabling test mocking and dependency injection.

### ToolStage: Parallel vs Sequential Execution

**Parallel path** (2+ tools, `ExecuteToolRaw` + `ProcessToolResult` wired):

1. Phase 1: Parallel I/O (no state mutation via goroutines)
2. Phase 2: Sequential result processing (deterministic order, mutations)

Benefits: Reduces latency for multi-tool iterations (e.g., read 3 files concurrently → merge results).

**Sequential fallback:** `ExecuteToolCall` handles both I/O and mutation atomically.

---

## Hooks System Design

### Overview

GoClaw provides a **flexible, permission-gated hook system** for intercepting agent lifecycle events. Hooks can execute shell commands, HTTP requests, or LLM prompts to approve/block operations.

**Files:** `internal/hooks/` (dispatcher.go, types.go, config.go, edition_gate.go, matcher.go)

### Hook Events & Lifecycle Points

```go
const (
  EventSessionStart       = "session_start"        // (non-blocking)
  EventUserPromptSubmit   = "user_prompt_submit"   // BLOCKING: pre-pipeline
  EventPreToolUse         = "pre_tool_use"         // BLOCKING: pre-execution
  EventPostToolUse        = "post_tool_use"        // (non-blocking)
  EventStop               = "stop"                 // (non-blocking)
  EventSubagentStart      = "subagent_start"       // BLOCKING: spawn approval
  EventSubagentStop       = "subagent_stop"        // (non-blocking)
)

// IsBlocking() = true for UserPromptSubmit, PreToolUse, SubagentStart
// Blocking events fail-closed: timeout or error → block
// Non-blocking events run async; failures logged only
```

### Hook Config & Execution

```go
type HookConfig struct {
  ID          uuid.UUID          // hook ID
  TenantID    uuid.UUID          // tenant scope (or SentinelTenantID for global)
  AgentID     *uuid.UUID         // agent scope (optional)
  Event       HookEvent
  HandlerType HandlerType        // command | http | prompt
  Scope       Scope              // global | tenant | agent
  Config      map[string]any     // handler-specific: command path, HTTP URL, LLM prompt
  Matcher     string             // regex pattern (e.g., "^read_.*", "^exec$")
  IfExpr      string             // CEL boolean expression for tool_name/tool_input/depth
  TimeoutMS   int                // per-hook timeout (default 5s, max 10s)
  OnTimeout   Decision           // allow | block (for blocking events, default block)
  Priority    int
  Enabled     bool
  Version     int
  Source      string             // "ui" | "agent_seeded"
  Metadata    map[string]any
  CreatedBy   *uuid.UUID
  CreatedAt   time.Time
  UpdatedAt   time.Time
}

type Decision string // allow | block | error | timeout
type HandlerType string // command | http | prompt
type Scope string // global | tenant | agent
```

### Dispatcher: Execution Engine

**Architecture:**

```
Fire(ctx, Event)
  ├─ Check loop depth (M5: max 3 levels nested sub-agent events)
  ├─ Resolve hooks from DB (tenant + agent scope)
  ├─ Fail-closed on DB error (blocking events block, non-blocking allow)
  │
  ├─ If blocking event:
  │  └─ runSync(chain, budget=10s)
  │     ├─ Per-hook timeout: default 5s, max 10s
  │     ├─ Pre-filter: matcher (regex) + IfExpr (CEL)
  │     ├─ For each enabled hook:
  │     │  ├─ Check circuit breaker (if tripped, block)
  │     │  ├─ runOne(hook, timeout)
  │     │  │  ├─ Handler.Execute(hctx, cfg, event)
  │     │  │  └─ Return (decision, error, duration)
  │     │  ├─ Write audit row
  │     │  ├─ If decision=block, return block (first block wins)
  │     │  ├─ If decision=timeout:
  │     │  │  ├─ Record hit for circuit breaker
  │     │  │  └─ OnTimeout=block → return block; OnTimeout=allow → continue
  │     │  ├─ If decision=error → fail-closed (return block)
  │     │  └─ If chain budget exhausted (H3) → fail-closed (return block)
  │     └─ Return allow
  │
  └─ If non-blocking event:
     └─ runAsync(chain)
        └─ Spawn goroutine per hook (Phase 2 routes via eventbus worker pool)
           ├─ Handler.Execute(ctx, cfg, event)
           ├─ Write audit row
           └─ Failures logged only
```

### Circuit Breaker (C4 Mitigation)

Hooks that block/timeout frequently are automatically disabled:

```go
type circuitBreaker struct {
  threshold    int           // (default 5 hits)
  window       time.Duration // (default 1 minute)
  hits         map[uuid.UUID][]time.Time // rolling window per hook
  tripped      map[uuid.UUID]bool        // persisted to DB when tripped
}

// record() appends timestamp; if count >= threshold in window → trip + persist
// isTripped() short-circuits Fire() to skip executing tripped hooks
```

### Handler Types

#### 1. Command Handler

Executes local shell command with event JSON on stdin.

- Edition-gated: **Lite only** (operator owns the host)
- Standard edition: blocked (C2 drop decision)
- Requires PATH to command

#### 2. HTTP Handler

Posts event JSON to HTTP endpoint.

- No edition gate
- Supports custom headers, method, retry policy (via handler impl)
- Can block based on status code

#### 3. Prompt Handler

Routes event through LLM prompt for approval.

- No edition gate
- **Requires matcher or if_expr** to prevent runaway LLM cost (runaway-cost guard)
- Returns decision based on LLM classification

### Validation & Edition Policy

```go
// Validate() runs cheap checks first, expensive last
func (h *HookConfig) Validate(ed edition.Edition) error {
  1. Event enum (map lookup)
  2. Scope/tenant/agent invariants
  3. Edition gate (HookEditionPolicy)
  4. Matcher regex + CEL compile (most expensive)
}

// HookEditionPolicy.Allow(handlerType, scope, edition)
// - command:  Lite ✓, Standard ✗
// - http:     Lite ✓, Standard ✓
// - prompt:   Lite ✓, Standard ✓
```

### Audit Trail

Every hook execution writes a row to `hook_executions`:

```go
type HookExecution struct {
  ID          uuid.UUID       // execution ID
  HookID      *uuid.UUID      // (NULL if hook deleted)
  SessionID   string          // session key
  Event       HookEvent
  InputHash   string          // canonical-JSON sha256 of tool_name + tool_input
  Decision    Decision        // allow | block | error | timeout
  DurationMS  int
  Retry       int
  DedupKey    string          // (hook_id, event_id) composite
  Error       string          // truncated to 256 chars
  ErrorDetail []byte          // AES-256-GCM encrypted
  Metadata    map[string]any
  CreatedAt   time.Time
}
```

---

## Permission Model

### Overview

GoClaw uses a **5-layer permission system**:

1. **Gateway Auth** (token/password, scopes)
2. **Global Tool Policy** (tools.allow[], tools.deny[], profile)
3. **Per-Agent Policy** (agents.list[].tools.allow/deny)
4. **Per-Channel/Group Policy** (channels._.groups._.tools.policy)
5. **Owner-Only Tools** (senderIsOwner check)

**File:** `internal/permissions/policy.go`

### Layer 1: Gateway Auth

```go
type Role string

const (
  RoleOwner    Role = "owner"     // Tenant management + full access
  RoleAdmin    Role = "admin"     // Full access to all methods
  RoleOperator Role = "operator"  // Read + write (no admin ops)
  RoleViewer   Role = "viewer"    // Read-only
)

type Scope string

const (
  ScopeAdmin     Scope = "operator.admin"
  ScopeRead      Scope = "operator.read"
  ScopeWrite     Scope = "operator.write"
  ScopeApprovals Scope = "operator.approvals"
  ScopePairing   Scope = "operator.pairing"
  ScopeProvision Scope = "operator.provision"
)
```

**Engine:**

```go
type PolicyEngine struct {
  ownerIDs map[string]bool // sender IDs considered "owner"
  mu       sync.RWMutex
}

// Methods:
IsOwner(senderID string) bool               // checks ownerIDs map
CanAccess(role, method string) bool         // role >= requiredRole
CanAccessWithScopes(scopes, method) bool    // required scopes ⊆ given scopes
```

**RPC Method Mapping:**

- Admin methods: config.apply, config.patch, agents.create/update/delete, teams._, skills.update, api_keys._
- Write methods: chat.send, sessions.delete/reset/patch, cron._, approvals._, etc.
- Read methods: everything else

**Role Hierarchy:**

```
Owner (4) ⊃ Admin (3) ⊃ Operator (2) ⊃ Viewer (1)
```

### Layer 5: Owner-Only Tools

Certain tools check if the sender is an "owner":

```go
if pe.IsOwner(senderID) {
  // Allow sensitive operation (e.g., shell exec, workspace traversal)
} else {
  return errors.New("not authorized")
}
```

**Fail-closed default:** When no owner IDs configured, only "system" is treated as owner.

---

## Sandbox Patterns

### Overview

GoClaw provides **Docker-based code execution isolation** for tool execution (exec, shell). Sandbox modes control which agents are isolated; scope controls container reuse.

**Files:** `internal/sandbox/sandbox.go`, `internal/sandbox/docker.go`

### Configuration

```go
type Config struct {
  // Agent isolation
  Mode            Mode              // off | non-main | all

  // Container setup
  Image           string            // goclaw-sandbox:bookworm-slim
  WorkspaceAccess Access            // none | ro | rw
  Scope           Scope             // session | agent | shared

  // Resource limits
  MemoryMB        int               // (default 512)
  CPUs            float64           // (default 1.0)
  TimeoutSec      int               // (default 300)

  // Network
  NetworkEnabled  bool
  RestrictedDomains []string
  Env             map[string]string

  // Security hardening
  ReadOnlyRoot    bool              // (default true)
  CapDrop         []string          // (default ["ALL"])
  Tmpfs           []string          // (default ["/tmp", "/var/tmp", "/run"])
  TmpfsSizeMB     int
  PidsLimit       int               // (default 256)
  User            string            // non-root user (e.g. "1000:1000")
  MaxOutputBytes  int               // (default 1MB)
  SetupCommand    string            // optional init command
  Workdir         string            // container workdir (default "/workspace")

  // Container lifecycle
  IdleHours       int               // prune idle > N hours (default 24)
  MaxAgeDays      int               // prune > N days old (default 7)
  PruneIntervalMin int              // check interval (default 5 min)
}
```

### Mode: Which Agents Run Sandboxed

```go
const (
  ModeOff     Mode = "off"      // no sandbox (all on host)
  ModeNonMain Mode = "non-main" // all except "main" agent
  ModeAll     Mode = "all"      // every agent
)

// ShouldSandbox(agentID) -> bool
switch c.Mode {
case ModeAll:        return true
case ModeNonMain:    return agentID != "main" && agentID != "default"
default:             return false
}
```

### Scope: Container Reuse Granularity

```go
const (
  ScopeSession Scope = "session" // one container per session
  ScopeAgent   Scope = "agent"   // one container per agent
  ScopeShared  Scope = "shared"  // one container for all agents
)

// ResolveScopeKey(sessionKey) -> scope key
// Extracted from session key format "agent:{agentId}:{rest}"
```

### Security Hardening

Default Docker create args (matching TypeScript `buildSandboxCreateArgs()`):

```bash
docker run -d \
  --name <prefix>-<sanitized-key> \
  --label goclaw.sandbox=true \
  --read-only \                       # read-only root filesystem
  --tmpfs /tmp:noexec,nosuid,nodev \  # tmpfs mounts with security flags
  --tmpfs /var/tmp:noexec,nosuid,nodev \
  --tmpfs /run:noexec,nosuid,nodev \
  --cap-drop ALL \                    # drop all Linux capabilities
  --security-opt no-new-privileges \
  --user <user> \                     # non-root user
  --memory <MemoryMB>m \
  --cpus <CPUs> \
  --pids-limit <PidsLimit> \
  [--network none] \
  [-v <hostPath>:<containerWorkdir>:ro|rw] \
  -w <Workdir> \
  <Image> sleep infinity
```

### Execution Interface

```go
type Sandbox interface {
  // Exec runs a command inside the sandbox
  Exec(ctx context.Context, command []string, workDir string, opts ...ExecOption) (*ExecResult, error)

  // Destroy removes the container
  Destroy(ctx context.Context) error

  // ID returns the container ID
  ID() string
}

type ExecResult struct {
  ExitCode int
  Stdout   string    // truncated to MaxOutputBytes
  Stderr   string    // truncated to MaxOutputBytes
}

// ExecOption for per-call env var injection (credentialed exec)
func WithEnv(env map[string]string) ExecOption
```

### Manager: Lifecycle Management

```go
type Manager interface {
  // Get returns (or creates) a sandbox for the given scope key
  Get(ctx, key, workspace string, cfgOverride *Config) (Sandbox, error)

  // Release destroys a sandbox by key
  Release(ctx, key string) error

  // ReleaseAll destroys all active sandboxes
  ReleaseAll(ctx) error

  // Stop signals pruning goroutine to stop
  Stop()

  // Stats returns info about active sandboxes
  Stats() map[string]any
}
```

**DockerManager:**

- Maintains `map[string]*DockerSandbox` keyed by scope key
- Tracks `createdAt` and `lastUsed` per container
- Spawns background pruning goroutine (interval-based)
- Prunes containers idle > N hours OR older than N days

### Workspace Access Isolation

```go
const (
  AccessNone Access = "none" // no filesystem mount
  AccessRO   Access = "ro"   // read-only workspace mount
  AccessRW   Access = "rw"   // read-write workspace mount
)

// Workspace mount format: `-v <hostPath>:<containerWorkdir>:ro|rw`
// resolveHostWorkspacePath() handles DooD (Docker-out-of-Docker) scenarios
```

---

## Callback Wiring

### Agent Loop Adapter Pattern

The agent loop (`internal/agent/loop.go`) bridges the v3 pipeline to the v2 loop via adapter methods.

**File:** `internal/agent/loop_pipeline_adapter.go`

```go
// Main entry point
func (l *Loop) runViaPipeline(ctx, req) (*RunResult, error) {
  input := convertRunInput(&req)
  bridgeRS := &runState{}  // shared loop-detection state
  deps := l.buildPipelineDeps(&req, bridgeRS)

  p := pipeline.NewDefaultPipeline(deps)
  state := pipeline.NewRunState(input, nil, model, provider)

  pResult, err := p.Run(ctx, state)
  return convertRunResult(pResult), nil
}

// Dependency building
func (l *Loop) buildPipelineDeps(req, bridgeRS) pipeline.PipelineDeps {
  maxIter := l.maxIterations  // (respect per-request override)
  cb := l.pipelineCallbacks(req, bridgeRS)  // build all closures

  return pipeline.PipelineDeps{
    TokenCounter: tokencount.NewTiktokenCounter(),
    EventBus: l.domainBus,
    Config: pipeline.PipelineConfig{
      MaxIterations: maxIter,
      MaxToolCalls: l.maxToolCalls,
      ContextWindow: l.contextWindow,
      MaxTokens: l.effectiveMaxTokens(),
      Compaction: l.compactionCfg,
    },
    ResolveContextWindow: func(provider, model string) int {
      if l.modelRegistry == nil { return 0 }
      spec := l.modelRegistry.Resolve(provider, model)
      if spec == nil { return 0 }
      return spec.ContextWindow
    },
    // ... (50+ callback assignments)
  }
}
```

### Callback Closure Pattern

**File:** `internal/agent/loop_pipeline_callbacks.go`

All callbacks are closures that capture `*Loop` and request context:

```go
func (l *Loop) pipelineCallbacks(req *RunRequest, bridgeRS *runState) pipelineCallbackSet {
  // Shared emitRun enriches events with request routing context
  emitRun := func(event AgentEvent) {
    event.RunKind = req.RunKind
    event.DelegationID = req.DelegationID
    event.TeamID = req.TeamID
    event.TeamTaskID = req.TeamTaskID
    event.ParentAgentID = req.ParentAgentID
    event.UserID = req.UserID
    event.Channel = req.Channel
    event.ChatID = req.ChatID
    event.SessionKey = req.SessionKey
    event.TenantID = l.tenantID
    l.emit(event)
  }

  return pipelineCallbackSet{
    emitRun: emitRun,
    injectContext: l.makeInjectContext(req),
    loadSessionHistory: l.makeLoadSessionHistory(),
    resolveWorkspace: l.makeResolveWorkspace(req),
    loadContextFiles: l.makeLoadContextFiles(),
    buildMessages: l.makeBuildMessages(),
    enrichMedia: l.makeEnrichMedia(req),
    // ... (30+ more)
  }
}
```

### Tool Execution Callbacks

**File:** `internal/agent/loop_pipeline_tool_callbacks.go`

Tool execution has 3-phase callback wiring:

#### Phase 1: ExecuteToolRaw (Parallel-Safe I/O)

```go
func (l *Loop) makeExecuteToolRaw(req *RunRequest) func(ctx, tc) (msg, rawData any, err) {
  emitRun := makeToolEmitRun(l, req)
  return func(ctx context.Context, tc providers.ToolCall) (msg, rawData, err) {
    registryName := l.resolveToolCallName(tc.Name)

    // Emit tool.call event at I/O start
    emitRun(AgentEvent{
      Type: protocol.AgentEventToolCall,
      Payload: map[string]any{"name": tc.Name, "id": tc.ID, "arguments": tc.Arguments},
    })

    // Emit tool span start (goroutine-safe: channel only)
    start := time.Now().UTC()
    spanID := l.emitToolSpanStart(ctx, start, tc.Name, tc.ID, argsJSON)

    // Inject agent audio snapshot (e.g., for TTS tool)
    if l.agentUUID != uuid.Nil {
      ctx = store.WithAgentAudio(ctx, store.AgentAudioSnapshot{
        AgentID: l.agentUUID,
        OtherConfig: append([]byte(nil), l.agentOtherConfig...), // defensive copy
      })
    }

    // Execute tool (parallel-safe: no state mutation)
    result := l.tools.ExecuteWithContext(ctx, registryName, tc.Arguments, ...)
    dur := time.Since(start)

    // Emit tool span end
    l.emitToolSpanEnd(ctx, spanID, start, result)

    // Return message + opaque rawData (toolRawResult wrapper) for ProcessToolResult
    msg := providers.Message{
      Role: "tool",
      Content: result.ForLLM,
      ToolCallID: tc.ID,
      IsError: result.IsError,
    }
    return msg, &toolRawResult{result: result, duration: dur}, nil
  }
}
```

#### Phase 2: ProcessToolResult (Sequential State Mutation)

```go
func (l *Loop) makeProcessToolResult(req, bridgeRS) func(ctx, state, tc, rawMsg, rawData) []msg {
  emitRun := makeToolEmitRun(l, req)
  return func(ctx, state, tc, rawMsg, rawData any) []providers.Message {
    registryName := l.resolveToolCallName(tc.Name)

    // Extract result + timing from toolRawResult
    var result *tools.Result
    var dur time.Duration
    if raw, ok := rawData.(*toolRawResult); ok && raw != nil {
      result = raw.result
      dur = raw.duration
    }

    // Record tool metrics (non-blocking, best-effort)
    l.recordToolMetric(ctx, req.SessionKey, registryName, !result.IsError, dur)

    // Process result (state mutation: loop detection, media, deliverables)
    toolMsg, warningMsgs, action := l.processToolResult(
      ctx, bridgeRS, req, emitRun, tc, registryName, result, hadBootstrap)

    // Sync loop-detection state from bridgeRS to pipeline RunState
    syncBridgeToState(bridgeRS, state, action)

    // Return tool message + warnings
    var msgs []providers.Message
    msgs = append(msgs, toolMsg)
    msgs = append(msgs, warningMsgs...)
    return msgs
  }
}
```

**Loop Detection Bridge:**

The `bridgeRS *runState` captures loop detection state that persists across tool execution:

```go
type runState struct {
  // Shared loop-detection counters (populated by processToolResult)
  loopKilled    bool   // set when loop detector triggers critical
  // ... other bridge fields
}

// After tool execution, sync back to pipeline state
func syncBridgeToState(bridgeRS *runState, state *pipeline.RunState, action loopAction) {
  if bridgeRS.loopKilled {
    state.Tool.LoopKilled = true
  }
}
```

### Tool Registry & Wiring

**File:** `cmd/gateway_tools_wiring.go`

Tool registry is wired at gateway startup:

```go
func wireExtraTools(
  pgStores *store.Stores,
  toolsReg *tools.Registry,
  msgBus *bus.MessageBus,
  workspace, dataDir string,
  agentCfg config.AgentDefaults,
  globalSkillsDir, builtinSkillsDir string,
) (heartbeatTool, hasMemory) {
  // Core tools
  toolsReg.Register(tools.NewDateTimeTool())
  toolsReg.Register(tools.NewCronTool(pgStores.Cron))
  toolsReg.Register(tools.NewHeartbeatTool(...))

  // Session tools
  toolsReg.Register(tools.NewSessionsListTool())
  toolsReg.Register(tools.NewSessionStatusTool())
  // ...

  // Register aliases (backward compat + Claude Code)
  toolsReg.RegisterAlias("Read", "read_file")
  toolsReg.RegisterAlias("Write", "write_file")
  toolsReg.RegisterAlias("Bash", "exec")
  // ...

  // Allow-path setup for filesystem tools
  if readTool, ok := toolsReg.Get("read_file"); ok {
    if pa, ok := readTool.(tools.PathAllowable); ok {
      pa.AllowPaths(skillsAllowPaths...)
      pa.AllowPaths(userAllowPaths...)
    }
  }

  // Wire session store + message bus awareness
  for _, name := range []string{"sessions_list", "session_status", ...} {
    if t, ok := toolsReg.Get(name); ok {
      if sa, ok := t.(tools.SessionStoreAware); ok {
        sa.SetSessionStore(pgStores.Sessions)
      }
      if ba, ok := t.(tools.BusAware); ok {
        ba.SetMessageBus(msgBus)
      }
    }
  }

  return heartbeatTool, hasMemory
}
```

**Builtin Tools Seeding:**

**File:** `cmd/gateway_builtin_tools.go`

Tools are pre-seeded into the database with idempotent logic:

```go
func builtinToolSeedData() []store.BuiltinToolDef {
  defs := []store.BuiltinToolDef{
    {Name: "read_file", DisplayName: "Read File", Category: "filesystem", Enabled: true},
    {Name: "write_file", DisplayName: "Write File", Category: "filesystem", Enabled: true},
    {Name: "exec", DisplayName: "Execute Command", Category: "runtime", Enabled: true,
      Metadata: json.RawMessage(`{"config_hint":"Config → Tools → Exec Approval"}`)},
    // ... (40+ more tools)
  }

  // Lite edition: filter out skill management tools
  if !edition.Current().TeamFullMode {
    liteHidden := map[string]bool{"skill_manage": true, "publish_skill": true}
    filtered := defs[:0]
    for _, d := range defs {
      if !liteHidden[d.Name] {
        filtered = append(filtered, d)
      }
    }
    return filtered
  }
  return defs
}

// Seed is idempotent: preserves user-customized enabled/settings on conflict
func seedBuiltinTools(ctx context.Context, bts store.BuiltinToolStore) {
  seeds := builtinToolSeedData()
  if err := bts.Seed(ctx, seeds); err != nil {
    slog.Error("failed to seed builtin tools", "error", err)
    return
  }
}
```

---

## Key Architectural Patterns

### 1. Callback-Driven Pipeline

**Pattern:** Pass implementation details as callbacks rather than embedding them.

**Benefits:**

- Test mockability: swap callbacks for fixtures
- Dependency injection: decouples pipeline from agent loop
- Composability: callbacks can wrap other callbacks
- Gradual migration: old stages can coexist with new callback-based stages

**Cost:**

- Closure complexity: many closures capture outer scope
- Inference difficulty: IDE can't trace through dynamic dispatch

### 2. Three-Tier Message Buffer

**Pattern:** Separate system prompt, history, and pending messages into distinct buffers.

**Benefits:**

- Clear phase separation: system (setup) → history (persistent) → pending (volatile)
- Efficient pruning: only history is compacted; pending is discarded post-checkpoint
- Crash recovery: history is flushed regularly; pending is re-run
- Media tracking: final content uses pending messages (before flush)

**Cost:**

- Slice copying: All() reconstructs every LLM call
- Invariant management: must maintain system + history + pending as separate views

### 3. Event-Driven Loop Detection

**Pattern:** Bridge loop detection state across the pipeline → tool execution → result processing.

**Benefits:**

- Decouples loop detector from pipeline stages
- Tool execution remains pipeline-agnostic
- State synchronization via explicit bridge object

**Cost:**

- Extra state management layer (runState bridge)
- Potential for state-sync bugs if synchronization misses a field

### 4. Fail-Closed Security by Default

**Pattern:** Blocking events timeout or error → block (don't allow).

**Benefits:**

- Prevents silent bypass on infrastructure failures (DB blip, network timeout)
- Circuit breaker protects against runaway hooks
- No graceful degradation leaks security

**Cost:**

- False positives: legitimate slow operations may be blocked
- Requires tuning: circuit breaker threshold/window must match operational load

### 3. Edition-Gated Features

**Pattern:** Embed edition checks at validation time AND execution time.

**Benefits:**

- Catches misconfigurations at config load (fail-fast)
- Protects at runtime in case config bypass (defense-in-depth)
- Example: command handler blocked on Standard edition only (C2 drop decision)

**Cost:**

- Duplication: gate logic in two places (config.go + dispatcher.go)
- Tight coupling: edition package dependency required

### 6. Context Pruning with Cache-TTL Gate

**Pattern:** 2-phase pruning (soft 70%, hard 100%) + optional cache TTL gate.

**Benefits:**

- Gradual degradation: soft prune before hard limit
- Cache-aware: respects provider cache TTL (keep prefix if cache still live)
- Token-aware: per-model context window resolution

**Cost:**

- Complexity: 3 decision points (soft threshold, hard threshold, cache gate)
- Config surface: requires GetProviderCaps, GetPruningConfig, GetCacheTouch, MarkCacheTouched callbacks

---

## Summary

### Pipeline as Operating System Primitive

GoClaw's pipeline is a **pluggable, callback-driven orchestration engine** for agent runs. The 8-stage model provides clear separation of concerns:

- **ContextStage** = identity & scope resolution
- **ThinkStage** = LLM reasoning
- **PruneStage** = memory budget enforcement
- **ToolStage** = action execution (with parallel option)
- **ObserveStage** = result accumulation
- **CheckpointStage** = crash recovery
- **FinalizeStage** = post-run cleanup

### Hooks as Intent Interception

The hook system intercepts lifecycle events (session start, pre-tool, post-tool) and can approve/block based on custom logic. Hooks are:

- **Type-safe:** Events carry structured payloads
- **Fail-closed:** Timeouts/errors block (blocking events)
- **Audited:** Every execution logged to hook_executions
- **Self-healing:** Circuit breaker disables misbehaving hooks

### Permissions as Layered Guards

A 5-layer permission model controls access:

1. Gateway role (viewer/operator/admin/owner)
2. Global tool policy
3. Per-agent tool policy
4. Per-channel/group tool policy
5. Owner-only tools

### Sandbox as Isolation Boundary

Docker-backed sandboxing provides code execution isolation with:

- **Mode-based control:** off / non-main / all
- **Scope-based reuse:** session / agent / shared
- **Security hardening:** read-only root, tmpfs, cap-drop, pids limit
- **Lifecycle management:** automatic pruning of idle containers

### Callback Wiring as Composition

The agent loop bridges to the pipeline via closures that capture context. This enables:

- Test mocking
- Dependency injection
- Gradual migration (old → new)
- Composable callbacks

For AGH implementation, adopt these patterns for:

- **Pipeline:** 8-stage model with callback injection
- **Hooks:** Event interception + approval/blocking
- **Permissions:** Layered guards (role/policy/owner)
- **Sandbox:** Container-based isolation with scope control
- **Wiring:** Callback closures for composition
