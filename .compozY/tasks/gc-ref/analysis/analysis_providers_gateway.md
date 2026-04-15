# GoClaw Provider & Gateway Architecture Analysis

## Executive Summary

GoClaw implements a sophisticated multi-provider LLM gateway with:

- **Plugin-based provider abstraction** supporting 20+ LLM providers via unified interface
- **ACP (Anthropic Console Proxy) integration** for subprocess-based agent management over JSON-RPC 2.0 stdio
- **Provider resolution chain** with failover, OAuth routing, and forward-compatibility
- **Gateway composition** through dependency injection with phase-based method registration
- **Message processing pipeline** with tenant isolation, channel routing, and post-turn consolidation
- **Consumer pattern** for multi-tenant inbound message handling with debouncing and scheduling

---

## 1. Provider Interface Design

### 1.1 Core Provider Interface

**Location:** `internal/providers/types.go`

```go
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error)
    DefaultModel() string
    Name() string
}
```

**Key Design Pattern: Minimal Interface**

- Only 4 methods required for LLM provider implementation
- Streaming support optional but preferred (many providers implement ChatStream)
- Provider identity via `Name()` (lowercase: "anthropic", "openai", "acp")
- Default model as fallback when not specified in request

### 1.2 Request/Response Model

**ChatRequest Structure:**

```go
type ChatRequest struct {
    Messages []Message        // conversation history with role/content
    Tools    []ToolDefinition // available tools for LLM to call
    Model    string           // override provider's default model
    Options  map[string]any   // extensible options: max_tokens, temperature, thinking_level, etc.
}

type ChatResponse struct {
    Content             string          // assistant text output
    Thinking            string          // extended thinking (when enabled)
    ToolCalls           []ToolCall      // tool invocations requested by LLM
    FinishReason        string          // "stop", "tool_calls", "length"
    Usage               *Usage          // token consumption
    Phase               string          // Codex-specific phase tracking
    RawAssistantContent json.RawMessage // for provider-specific blocks (Anthropic thinking)
}

type StreamChunk struct {
    Content  string // partial text delta
    Thinking string // reasoning delta
    Done     bool   // stream completion flag
}
```

**Design Pattern: Schema Flexibility**

- `Options` map supports provider-specific and middleware-specific keys
- `RawAssistantContent` preserves provider-specific data (Anthropic requires thinking blocks passed back in tool loops)
- `Usage` tracks tokens including cache hits/thinking tokens
- `Phase` field enables Codex model state persistence across turns

### 1.3 Optional Capability Interfaces

**Thinking Capable:**

```go
type ThinkingCapable interface {
    SupportsThinking() bool
}
```

Used to gate thinking_level injection to prevent sending unsupported options.

**Capabilities Aware:**

```go
type CapabilitiesAware interface {
    Capabilities() ProviderCapabilities
}

type ProviderCapabilities struct {
    Streaming        bool   // ChatStream() support
    ToolCalling      bool   // accepts tools in request
    StreamWithTools  bool   // can stream while calling tools
    Thinking         bool   // extended thinking support
    Vision           bool   // image input support
    CacheControl     bool   // Anthropic cache_control blocks
    MaxContextWindow int    // context window size
    TokenizerID      string // for BPE token counting (e.g., "cl100k_base")
}
```

Used by pipeline to select code paths based on provider capabilities (e.g., skip streaming for non-streaming providers).

---

## 2. Provider Registry & Resolution

### 2.1 Registry Pattern

**Location:** `internal/providers/registry.go`

```go
type Registry struct {
    providers     map[string]Provider // keyed as "tenantID/providerName"
    mu            sync.RWMutex
    tenantFromCtx func(context.Context) uuid.UUID // tenant resolver from context
    roundRobinMu  sync.Mutex
    roundRobinCounters map[string]int // per-provider round-robin state
}

// Master tenant for config-based providers (UUID: 0193a5b0-7000-7000-8000-000000000001)
var MasterTenantID = uuid.Must(uuid.Parse("0193a5b0-7000-7000-8000-000000000001"))
```

**Lookup Pattern:**

```go
func (r *Registry) Get(ctx context.Context, name string) (Provider, error) {
    tenantID := r.tenantFromContext(ctx)  // extract from context
    return r.GetForTenant(tenantID, name)  // lookup "tenantID/name"
}

func (r *Registry) GetForTenant(tenantID uuid.UUID, name string) (Provider, error) {
    key := tenantID.String() + "/" + name
    if tenantID != MasterTenantID {
        if p, ok := r.providers[key]; ok {
            return p, nil  // tenant-specific override
        }
    }
    // fallback to master tenant
    masterKey := MasterTenantID.String() + "/" + name
    if p, ok := r.providers[masterKey]; ok {
        return p, nil
    }
    return nil, fmt.Errorf("provider not found: %s", name)
}
```

**Design Pattern: Multi-Tenant Isolation**

- Per-tenant provider overrides with master tenant fallback
- Compound key "tenantID/providerName" ensures isolation
- Round-robin state keyed by "tenantID/providerName" for shared routers

**Registration:**

- `Register(provider)` → registers under MasterTenantID
- `RegisterForTenant(tenantID, provider)` → registers under specific tenant
- On replacement, old provider is closed if it implements `io.Closer`

### 2.2 Provider Registration Flow

**Location:** `cmd/gateway_providers.go`

**Config-Based Providers (from JSON5 config):**

```go
func registerProviders(registry *providers.Registry, cfg *config.Config, modelReg providers.ModelRegistry) {
    // Native HTTP providers (Anthropic, OpenAI-compat variants)
    if cfg.Providers.Anthropic.APIKey != "" {
        registry.Register(providers.NewAnthropicProvider(...))
    }

    // OpenAI-compatible endpoints (20+ providers)
    if cfg.Providers.OpenAI.APIKey != "" {
        registry.Register(providers.NewOpenAIProvider("openai", ...))
    }
    // Groq, DeepSeek, Gemini, Mistral, XAI, MiniMax, Cohere, Perplexity, ...

    // Provider-specific adapters
    if cfg.Providers.DashScope.APIKey != "" {
        registry.Register(providers.NewDashScopeProvider(...))
    }

    // Subprocess-based providers (no API key needed)
    if cfg.Providers.ClaudeCLI.CLIPath != "" {
        // Build MCP config with GoClaw bridge + external MCP servers
        mcpData := providers.BuildCLIMCPConfigData(cfg.Tools.McpServers, gatewayAddr, cfg.Gateway.Token)
        registry.Register(providers.NewClaudeCLIProvider(cliPath, opts...))
    }

    if cfg.Providers.ACP.Binary != "" {
        registerACPFromConfig(registry, cfg.Providers.ACP)
    }
}
```

**Database Providers (from llm_providers table):**

```go
func registerProvidersFromDB(registry *providers.Registry, provStore store.ProviderStore, ...) {
    dbProviders, _ := provStore.ListAllProviders(ctx)
    for _, p := range dbProviders {
        // Per-tenant registration
        registry.RegisterForTenant(p.TenantID, adapter)
    }
}
```

**Design Pattern: Layered Registration**

1. Config-based providers registered first (global defaults)
2. DB providers registered second (overwrite config if same name)
3. Enables per-tenant provider overrides without code changes

---

## 3. ACP (Anthropic Console Proxy) Integration

### 3.1 Architecture Overview

**Purpose:** Orchestrate external ACP-compatible agents (including Claude CLI with MCP) as managed subprocesses.

**Transport:** JSON-RPC 2.0 over stdio (bidirectional)

- Client → Agent: requests (initialize, session/new, session/prompt, session/cancel)
- Agent → Client: responses (PromptResponse), notifications (session/update), requests (fs/readTextFile, terminal/create, permission/request)

### 3.2 Process Pool Management

**Location:** `internal/providers/acp/process.go`

```go
type ProcessPool struct {
    processes   sync.Map                        // sessionKey → *ACPProcess
    spawnMu     sync.Map                        // sessionKey → *sync.Mutex
    agentBinary string
    agentArgs   []string
    workDir     string
    idleTTL     time.Duration                  // 5 min default
    toolHandler RequestHandler                 // bridges agent→client requests
    done        chan struct{}
}

type ACPProcess struct {
    cmd        *exec.Cmd
    conn       *Conn                          // JSON-RPC connection
    sessionID  string                         // ACP session ID
    agentCaps  AgentCaps
    lastActive time.Time
    inUse      atomic.Int32                  // prevents reaping while prompt active
    exited     chan struct{}                 // closed when process exits
    updateFn   func(SessionUpdate)          // callback for streaming updates
}
```

**Lifecycle:**

```go
// GetOrSpawn: returns existing process or spawns new one
proc, err := pool.GetOrSpawn(ctx, sessionKey)

// Reaping loop: every 30s, kills idle processes (inUse check prevents early termination)
go pp.reapLoop()

// On shutdown
pool.Close()  // cancels all processes, waits up to 5s for graceful exit
```

**Design Pattern: Session-Based Lifecycle**

- One process per session (persistent across multiple prompts)
- Per-session mutex prevents concurrent spawns
- Idle TTL (5 min default) allows resource cleanup
- `inUse` counter prevents reaping while prompt is active

### 3.3 JSON-RPC 2.0 Protocol Implementation

**Location:** `internal/providers/acp/jsonrpc.go`

```go
type Conn struct {
    writer  io.Writer
    reader  io.Reader
    nextID  atomic.Int64
    pending sync.Map      // id → chan *jsonrpcMessage
    handler RequestHandler
    notify  NotifyHandler
    done    chan struct{}
    mu      sync.Mutex    // protects writes
}

// Message dispatch (async read loop):
// ID + Method != "" && Method != "" → agent→client request, respond asynchronously
// ID + Method == "" → response to our Call, dispatch to pending caller
// No ID + Method != "" → notification, dispatch to notify handler
```

**Three Message Types:**

1. **Requests** (Agent → Client with ID):

```json
{"jsonrpc":"2.0","id":1,"method":"fs/readTextFile","params":{"path":"/tmp/file.txt"}}
→ {"jsonrpc":"2.0","id":1,"result":{"content":"..."}}
```

2. **Responses** (our Call):

```json
{"jsonrpc":"2.0","id":5,"result":{...}} // or {"error":{"code":-32000,"message":"..."}}
```

3. **Notifications** (Agent → Client without ID):

```json
{"jsonrpc":"2.0","method":"session/update","params":{"kind":"message","message":{...}}}
```

**Design Pattern: Zero-Copy Message Routing**

- Single read loop dispatches to multiple goroutines
- Pending channel per request ID (buffered for quick dispatch)
- Context-aware request handlers with connection lifetime
- Write mutex ensures linearization without goroutine overhead

### 3.4 ACP Session and Prompt Lifecycle

**Location:** `internal/providers/acp/session.go`

```go
// Initialize: ACP handshake
Initialize(ctx) {
    req := InitializeRequest{
        ClientInfo: {Name: "goclaw", Version: "1.0"},
        Capabilities: {Fs: {Read: true, Write: true}, Terminal: {Enabled: true}}
    }
    conn.Call("initialize", req, &resp)
}

// Create session
NewSession(ctx) {
    conn.Call("session/new", {}, &resp)
    p.sessionID = resp.SessionID
}

// Prompt: send user content, stream updates via callback
Prompt(ctx, content, onUpdate) {
    p.setUpdateFn(onUpdate)  // install streaming callback
    conn.Call("session/prompt", {SessionID, Content}, &resp)
}
```

**Update Streaming:**

```go
// During Prompt, agent sends session/update notifications:
// {kind: "message", message: {role: "assistant", content: [{type: "text", text: "..."}]}}
// {kind: "toolCall", toolCall: {id: "...", name: "...", status: "running"}}

// Client's onUpdate callback processes each one
onUpdate(SessionUpdate{
    Kind: "message",
    Message: &{Role: "assistant", Content: [{Type: "text", Text: "chunk"}]},
})
```

**Design Pattern: Structured Streaming**

- `session/prompt` is a long-running RPC call
- Agent sends `session/update` notifications while computing (not responses)
- Caller installs per-prompt callback for side-effect-free message processing
- Callback receives structured blocks (text, toolCall, etc.) not raw streaming

### 3.5 Tool Bridge: Agent→Client Request Handling

**Location:** `internal/providers/acp/tool_bridge.go`

```go
type ToolBridge struct {
    workspace    string                   // sandbox boundary
    terminals    sync.Map                 // terminalID → *Terminal
    denyPatterns []*regexp.Regexp        // shell deny patterns (e.g., rm, format, etc.)
    permMode     string                  // "approve-all", "approve-reads", "deny-all"
}

// Handle: dispatch agent→client requests
func (tb *ToolBridge) Handle(ctx context.Context, method string, params json.RawMessage) (any, error) {
    switch method {
    case "fs/readTextFile":
        // Permission check + workspace boundary validation
        resolved, _ := tb.resolvePath(req.Path)  // prevent path traversal
        data, _ := os.ReadFile(resolved)
        return &{Content: string(data)}, nil

    case "fs/writeTextFile":
        // Similar but respects permMode checks

    case "terminal/create":
        // Create pseudo-terminal, apply deny patterns to command
        if tb.blockedByDenyPattern(req.Command) {
            return nil, fmt.Errorf("command blocked")
        }
        term := tb.createTerminal(req.Command, req.Args)
        return &{TerminalID: term.ID}, nil

    case "terminal/output":
        // Fetch accumulated output

    case "terminal/waitForExit":
        // Blocking call with context cancellation
        <-term.Done()
        return &{ExitStatus: term.ExitCode}, nil

    case "permission/request":
        // Auto-approve based on permMode
        return &{Outcome: "approved"}, nil
    }
}
```

**Design Pattern: Workspace Sandboxing**

- `resolvePath()` ensures all file operations stay within workspace directory
- Path traversal prevented via `filepath.Clean` + boundary check
- Terminal commands validated against deny patterns (shell injection prevention)
- Permission mode controls baseline approval without UI interaction

**Permission Modes:**

- `"approve-all"`: all operations auto-approved
- `"approve-reads"`: read-only (blocks writes + terminals)
- `"deny-all"`: blocks all except notifications

### 3.6 ACPProvider: Wrapping ACP Process Pool

**Location:** `internal/providers/acp_provider.go`

```go
type ACPProvider struct {
    name         string
    pool         *acp.ProcessPool      // manages subprocess lifecycle
    bridge       *acp.ToolBridge       // handles agent→client requests
    defaultModel string
    sessionMu    sync.Map              // sessionKey → *sync.Mutex
}

// Chat (non-streaming): collect all text blocks from updates
Chat(ctx, req) {
    sessionKey := req.Options[OptSessionKey]
    proc, _ := pool.GetOrSpawn(ctx, sessionKey)

    var buf strings.Builder
    _, _ := proc.Prompt(ctx, content, func(update) {
        if update.Message != nil {
            for _, block := range update.Message.Content {
                if block.Type == "text" {
                    buf.WriteString(block.Text)
                }
            }
        }
    })

    return &ChatResponse{Content: buf.String(), ...}, nil
}

// ChatStream: emit each block to onChunk callback
ChatStream(ctx, req, onChunk) {
    sessionKey := req.Options[OptSessionKey]
    proc, _ := pool.GetOrSpawn(ctx, sessionKey)

    _, _ := proc.Prompt(ctx, content, func(update) {
        if update.Message != nil {
            for _, block := range update.Message.Content {
                if block.Type == "text" {
                    onChunk(StreamChunk{Content: block.Text})
                }
            }
        }
    })

    onChunk(StreamChunk{Done: true})
    return &ChatResponse{...}, nil
}
```

**Design Pattern: Provider Interface Adapter**

- ACPProvider implements `Provider` interface
- Hides complexity of process pool, session management, stream callbacks
- Session lifetime tied to `SessionKey` in Options (caller provides)
- Per-session mutex prevents concurrent prompts on same session

---

## 4. Provider Resolution Chain

### 4.1 Agent Provider Resolution

**Location:** `internal/providerresolve/agent_provider.go`

```go
func ResolveConfiguredProvider(registry *providers.Registry, agent *store.AgentData) (providers.Provider, error) {
    baseProvider, baseErr := registry.GetForTenant(agent.TenantID, agent.Provider)
    if baseErr == nil {
        // Check if Codex provider with OAuth routing
        if _, ok := baseProvider.(*providers.CodexProvider); !ok {
            return baseProvider, nil  // non-Codex: return directly
        }
    }

    // Codex provider: check for routing config
    var providerDefaults *store.ChatGPTOAuthRoutingConfig
    if codex, ok := baseProvider.(*providers.CodexProvider); ok {
        if defaults := codex.RoutingDefaults(); defaults != nil {
            providerDefaults = &store.ChatGPTOAuthRoutingConfig{
                Strategy: defaults.Strategy,
                ExtraProviderNames: defaults.ExtraProviderNames,
            }
        }
    }

    // Merge routing: agent config override + provider defaults
    if routing := store.ResolveEffectiveChatGPTOAuthRouting(providerDefaults, agent.ParseChatGPTOAuthRouting()); routing != nil {
        if routing.Strategy != store.ChatGPTOAuthStrategyPrimaryFirst || len(routing.ExtraProviderNames) > 0 {
            // Create router to handle multi-profile failover
            router := providers.NewChatGPTOAuthRouter(
                agent.TenantID,
                registry,
                agent.Provider,
                routing.Strategy,
                routing.ExtraProviderNames,
            )
            if router != nil && router.HasRegisteredProviders() {
                return router, nil  // return routing-enabled provider
            }
        }
    }

    if baseErr == nil {
        return baseProvider, nil
    }
    return nil, baseErr
}
```

**Resolution Chain:**

1. **Lookup base provider** from registry using agent's provider name
2. **Check for Codex** (ChatGPT OAuth) with routing config
3. **If routing enabled**, wrap with ChatGPTOAuthRouter for failover
4. **Otherwise**, return base provider directly

**Design Pattern: Decorator Pattern with Configuration**

- Base provider from registry untouched
- Routing configuration wrapped in decorator (router)
- Allows per-agent failover policy without provider changes

### 4.2 Model Registry and Forward Compatibility

**Location:** `internal/providers/model_registry.go`

```go
type ModelRegistry interface {
    Resolve(provider, modelID string) *ModelSpec
    Register(spec ModelSpec)
    Catalog(provider string) []ModelSpec
}

type InMemoryRegistry struct {
    models    sync.Map  // "provider:modelID" → *ModelSpec
    resolvers sync.Map  // provider → ForwardCompatResolver
}

// Forward-compat resolver for unknown models
type ForwardCompatResolver interface {
    ResolveForwardCompat(modelID string, registry ModelRegistry) *ModelSpec
}

// Resolve: direct hit → forward-compat resolver → nil
func (r *InMemoryRegistry) Resolve(provider, modelID string) *ModelSpec {
    if v, ok := r.models.Load("provider:modelID"); ok {
        return v.(*ModelSpec)
    }

    // Forward-compat resolver for unknown models
    if v, ok := r.resolvers.Load(provider); ok {
        if resolver, ok := v.(ForwardCompatResolver); ok {
            if spec := resolver.ResolveForwardCompat(modelID, r); spec != nil {
                r.Register(*spec)  // cache for next lookup
                return spec
            }
        }
    }
    return nil
}
```

**Design Pattern: Two-Tier Model Lookup**

1. Direct cache hit for known models
2. Forward-compat resolver for unknown models (allows providers to infer specs from model ID)
3. Automatic caching of resolved specs

**Use Cases:**

- OpenAI provider infers context window from model ID pattern
- New models adopted without code changes
- Per-model capabilities (reasoning, vision, cache support)

---

## 5. Gateway Dependency Injection

### 5.1 Dependency Structure

**Location:** `cmd/gateway_deps.go`

```go
type gatewayDeps struct {
    cfg              *config.Config
    server           *gateway.Server
    msgBus           *bus.MessageBus
    pgStores         *store.Stores
    providerRegistry *providers.Registry
    channelMgr       *channels.Manager
    agentRouter      *agent.Router
    toolsReg         *tools.Registry
    skillsLoader     *skills.Loader
    permCache        *cache.PermissionCache
    enrichProgress   *vault.EnrichProgress
    enrichWorker     *vault.EnrichWorker
    workspace        string
    dataDir          string
    domainBus        eventbus.DomainEventBus
}
```

**Key Components:**

| Component          | Purpose                                              |
| ------------------ | ---------------------------------------------------- |
| `cfg`              | Configuration (providers, port, models, etc.)        |
| `server`           | WebSocket + HTTP server                              |
| `msgBus`           | Event bus for inter-component communication          |
| `pgStores`         | Store interfaces (agents, sessions, providers, etc.) |
| `providerRegistry` | LLM provider lookup                                  |
| `channelMgr`       | Channel manager (Telegram, Discord, etc.)            |
| `agentRouter`      | Agent loop cache + routing                           |
| `toolsReg`         | Tool registry                                        |
| `skillsLoader`     | Skill definitions + BM25 search                      |
| `permCache`        | Permission cache with sweep goroutines               |
| `domainBus`        | Domain event bus for consolidation                   |

### 5.2 Provider Registration Flow

```go
// Step 1: Config-based providers
registerProviders(registry, cfg, modelRegistry)

// Step 2: Database providers (overwrite config if name matches)
registerProvidersFromDB(registry, stores.ProviderStore, ...)

// Step 3: Available for agent resolution
// When agent runs: providerresolve.ResolveConfiguredProvider(registry, agent)
```

### 5.3 Method Registration (3 Phases)

**Location:** `cmd/gateway_methods.go`

```go
func registerAllMethods(server, agents, stores, cfg, ...) {
    router := server.Router()

    // Phase 1: Core (blocks other components)
    chatMethods := methods.NewChatMethods(agents, sessStore, cfg, ...)
    chatMethods.Register(router)
    methods.NewAgentsMethods(agents, ...).Register(router)
    methods.NewSessionsMethods(sessStore, ...).Register(router)
    methods.NewConfigMethods(cfg, ...).Register(router)

    // Phase 2: Extended (can depend on Phase 1)
    methods.NewSkillsMethods(skillStore, ...).Register(router)
    methods.NewCronMethods(cronStore, ...).Register(router)
    methods.NewHeartbeatMethods(heartbeatStore, ...).Register(router)
    methods.NewExecApprovalMethods(execApprovalMgr, ...).Register(router)
    methods.NewSendMethods(msgBus).Register(router)

    // Phase 3: Auxiliary
    methods.NewLogsMethods(logTee).Register(router)
}
```

**Design Pattern: Staged Initialization**

- Phase 1 provides essential routing (agents, sessions, config)
- Phase 2 depends on Phase 1 dependencies being available
- Enables clear dependency declaration without circular imports

---

## 6. Message Processing Pipeline

### 6.1 Consumer Pattern Overview

**Location:** `cmd/gateway_consumer_*.go`

The consumer handles inbound channel messages through:

1. **Debouncing** → merge rapid messages
2. **Normalization** → extract metadata, resolve agent/user/session
3. **Scheduling** → submit to agent loop with context
4. **Post-turn** → consolidation, task updates, notifications

### 6.2 Normal Message Flow

**Location:** `cmd/gateway_consumer_normal.go`

```go
func processNormalMessage(ctx context.Context, msg bus.InboundMessage, deps *ConsumerDeps) {
    // Step 1: Inject tenant from channel into context
    ctx = store.WithTenantID(ctx, msg.TenantID)

    // Step 2: Resolve target agent
    agentID := msg.AgentID
    if agentID == "" {
        agentID = resolveAgentRoute(deps.Cfg, msg.Channel, msg.ChatID, msg.PeerKind)
    }
    agentLoop, _ := deps.Agents.Get(ctx, agentID)

    // Step 3: Build session key (with thread/topic isolation)
    peerKind := msg.PeerKind  // "direct" or "group"
    sessionKey := sessions.BuildScopedSessionKey(agentID, msg.Channel, peerKind, msg.ChatID)

    // Thread-based override (Slack/Discord threads)
    if lk := msg.Metadata["local_key"]; strings.Contains(lk, ":thread:") {
        parts := strings.SplitN(lk, ":thread:", 2)
        sessionKey = sessions.BuildScopedThreadSessionKey(agentID, msg.Channel, peerKind, msg.ChatID, parts[1])
    }

    // Forum topic isolation (Telegram topics in supergroups)
    if msg.Metadata[tools.MetaIsForum] == "true" && peerKind == string(sessions.PeerGroup) {
        topicID := parseMetaInt(msg.Metadata[tools.MetaMessageThreadID])
        if topicID > 0 {
            sessionKey = sessions.BuildGroupTopicSessionKey(agentID, msg.Channel, msg.ChatID, topicID)
        }
    }

    // Step 4: Resolve user ID (per-user or group-scoped)
    userID := msg.UserID
    if peerKind == string(sessions.PeerGroup) && msg.ChatID != "" {
        if guildID := msg.Metadata["guild_id"]; guildID != "" && msg.SenderID != "" {
            // Discord: per-user per-guild scope
            userID = fmt.Sprintf("guild:%s:user:%s", guildID, msg.SenderID)
        } else {
            // Other platforms: group-scoped
            userID = fmt.Sprintf("group:%s:%s", msg.Channel, msg.ChatID)
        }
    }

    // Step 5: Persist metadata (friendly names, contact info)
    sessionMeta := extractSessionMetadata(msg, peerKind)
    deps.SessStore.SetSessionMetadata(ctx, sessionKey, sessionMeta)

    // Step 6: Auto-collect contacts (for contact selector UI)
    if deps.ContactCollector != nil && msg.SenderID != "" && !bus.IsInternalSender(msg.SenderID) {
        deps.ContactCollector.EnsureContact(ctx, channelType, msg.Channel, senderNumericID, userID, ...)
    }

    // Step 7: Resolve merged tenant user (Slack/Teams → tenant_users)
    // If sender has been merged to tenant_user, use that for DM sessions
    if deps.ContactCollector != nil && peerKind == string(sessions.PeerDirect) {
        tenantUserID := deps.ContactCollector.ResolveTenantUserID(ctx, channelType, senderNumericID)
        if tenantUserID != "" {
            userID = tenantUserID  // override to tenant user ID
        }
    }

    // Step 8: Build run request
    runReq := agent.RunRequest{
        SessionKey: sessionKey,
        UserID: userID,
        Prompt: msg.Text,
        Images: msg.Images,  // vision input
        Tools: msg.Tools,    // tool overrides from channel
        Metadata: msg.Metadata,
    }

    // Step 9: Register run and submit to scheduler
    runID := uuid.New().String()
    injectCh := deps.Agents.RegisterRun(ctx, runID, sessionKey, agentID, cancelFn)

    task := scheduler.Task[agent.RunRequest]{
        ID: runID,
        Data: runReq,
        Lane: lane,  // main/subagent/cron
    }
    deps.Scheduler.Submit(task)
}
```

**Design Pattern: Context-Based Tenant Isolation**

- `store.WithTenantID(ctx, tenantID)` propagates tenant through entire request
- All store operations scoped by tenant (implicit filtering)
- No tenant leakage between requests

**Session Key Format:**

```
agent:{agentID}:{channel}:{peerKind}:{chatID}
agent:{agentID}:{channel}:{peerKind}:{chatID}:thread:{threadID}    // thread isolation
agent:{agentID}:{channel}:{peerKind}:{chatID}:topic:{topicID}      // forum topic isolation
```

**User ID Scoping:**

- Direct messages: user ID from sender
- Groups (Discord): `guild:{guildID}:user:{senderID}` (per-user per-guild context)
- Groups (other): `group:{channel}:{chatID}` (shared group context)

### 6.3 Post-Turn Processing

**Location:** `cmd/gateway_consumer_post_turn.go`

```go
func processPostTurn(ctx context.Context, outcome scheduler.RunOutcome, deps *ConsumerDeps) {
    // Step 1: Handle teammate task outcomes
    if meta := outcome.TeamTaskMeta; meta.TaskID != uuid.Nil {
        team := resolveTeamTaskOutcome(ctx, deps, outcome, flags, meta)

        // Task status updates:
        // error/loopKilled → fail
        // completed → auto-complete
        // reviewed → renew lock
        // escalated → skip

        // Dispatch unblocked dependent tasks
        deps.PostTurn.DispatchUnblockedTasks(ctx, meta.TeamID)
    }

    // Step 2: Memory consolidation (episodic → semantic)
    if outcome.ConsolidationItems != nil {
        deps.PostTurn.QueueConsolidation(ctx, outcome.AgentID, outcome.SessionKey, items)
    }

    // Step 3: Outbound notifications
    if outcome.Notifications != nil {
        deps.PostTurn.SendNotifications(ctx, outcome.Notifications)
    }
}
```

**Design Pattern: Post-Turn Effects**

- Separates agent loop (request→response) from side effects (memory, tasks, notifications)
- Allows background processing without blocking client response
- Enables retry/rollback semantics for consolidation

### 6.4 Scheduler Integration

**Location:** `cmd/gateway_consumer_process.go`

```go
func makeSchedulerRunFunc(agents *agent.Router, cfg *config.Config) scheduler.RunFunc {
    return func(ctx context.Context, req agent.RunRequest) (*agent.RunResult, error) {
        // Step 1: Extract agent ID from session key
        agentID := cfg.ResolveDefaultAgentID()
        if parts := strings.SplitN(req.SessionKey, ":", 4); len(parts) >= 2 {
            switch parts[0] {
            case "agent":
                agentID = parts[1]
            case "delegate":
                if len(parts) >= 3 {
                    agentID = parts[2]
                }
            }
        }

        // Step 2: Get agent loop
        loop, _ := agents.Get(ctx, agentID)

        // Step 3: Register run (enables IsSessionBusy + AbortRunsForSession)
        runCtx, cancel := context.WithCancel(ctx)
        injectCh := agents.RegisterRun(runCtx, req.RunID, req.SessionKey, agentID, cancel)
        defer agents.UnregisterRun(req.RunID)
        defer cancel()

        req.InjectCh = injectCh
        return loop.Run(runCtx, req)
    }
}
```

**Design Pattern: Run Registration and Cancellation**

- Each run tracked in agent router
- Enables query: IsSessionBusy(sessionKey)
- Enables control: AbortRunsForSession(sessionKey)
- Abort closes injectCh → agent loop detects and stops

---

## 7. Agent Resolution and Routing

### 7.1 Agent Router Cache

**Location:** `internal/agent/resolver.go` (partial)

```go
type ResolverDeps struct {
    AgentStore     store.AgentStore
    ProviderStore  store.ProviderStore
    ProviderReg    *providers.Registry
    ModelRegistry  providers.ModelRegistry
    Bus            bus.EventPublisher
    Sessions       store.SessionStore
    Tools          *tools.Registry

    // Callbacks for dynamic resolution
    EnsureUserProfile EnsureUserProfileFunc
    SeedUserFiles     SeedUserFilesFunc
    ContextFileLoader ContextFileLoaderFunc

    // Configuration
    CompactionCfg     *config.CompactionConfig
    ContextPruningCfg *config.ContextPruningConfig
    SandboxEnabled    bool

    // Stores for extended features
    AgentLinkStore     store.AgentLinkStore    // delegation
    TeamStore          store.TeamStore         // teammate context
    MCPStore           store.MCPServerStore    // MCP servers
    SkillAccessStore   store.SkillAccessStore  // skill visibility
    MediaStore         *media.Store            // persistent images
    TracingStore       store.TracingStore      // budget tracking
}
```

**Resolution Steps:**

1. Lookup agent by ID (or agent_key)
2. Load agent configuration (name, provider, tools, context files)
3. Load user profile (per-user context files, model overrides)
4. Seed per-user files if first use
5. Build agent loop with all dependencies

### 7.2 Tool Policy Evaluation

Tools are evaluated per agent/user with:

- Tool enabled/disabled status per agent
- Tool visibility per skill (user may not see all skills)
- Tool execution approval requirements
- Secure CLI credential binding per agent

---

## 8. Key Architectural Patterns

### 8.1 Composition Patterns

**Plugin Pattern (Providers):**

- Provider interface minimal (4 methods)
- Each provider is a concrete implementation
- Registry maps provider names to instances
- Forward-compatibility via ModelRegistry resolver

**Decorator Pattern (Routing):**

- ChatGPTOAuthRouter wraps base provider
- Adds failover logic without modifying base
- Enables per-agent routing policies

**Strategy Pattern (Middleware):**

- RequestMiddleware transforms request bodies
- Composed left-to-right
- Nil-safe (skip if not configured)
- Examples: cache middleware, service tier

### 8.2 Isolation Patterns

**Tenant Isolation:**

- Context-based: `store.WithTenantID(ctx, tenantID)`
- Registry keying: `"tenantID/providerName"`
- SQL WHERE clauses: `WHERE tenant_id = $1`
- No shared mutable state

**Session Isolation:**

- Per-session key in registry lookup
- Thread/topic overrides for sub-channels
- Session history never crosses keys

**Process Isolation (ACP):**

- One process per session (no cross-session message leakage)
- Workspace sandboxing prevents directory traversal
- Tool bridge validates all operations

### 8.3 Concurrency Patterns

**Sync.Map for Hot Paths:**

- Registry lookups (reader-heavy)
- Process pool (many sessions)
- Zero-alloc on read in happy path

**Per-Key Mutex:**

- Process pool spawn lock (prevents concurrent spawns for same session)
- Session locks in ACPProvider (prevents concurrent prompts on same session)

**Context Cancellation:**

- Request-level cancellation flows through agent loop
- Abort channels notify agent of cancellation
- Run registration enables scheduler-level cancellation

### 8.4 Streaming Patterns

**Event-Driven Updates:**

- Provider emits StreamChunk via onChunk callback
- Chunks forwarded to WebSocket client as events
- Client accumulates chunks for display

**Long-Running Requests:**

- HTTP POST with streaming response (SSE)
- WebSocket with event emission
- Both preserve streaming semantic (first response without buffering)

---

## 9. Integration Points Summary

### 9.1 Chat Request Flow

```
Client Request
    ↓
ChatMethods.Chat()
    ↓
Agent.Run() [blocking]
    ├─ Load agent config + provider
    ├─ Resolve configured provider
    │   └─ providerresolve.ResolveConfiguredProvider(registry, agent)
    │       └─ Check for Codex routing → may wrap in ChatGPTOAuthRouter
    ├─ Build messages from session history
    ├─ Call provider.ChatStream(ctx, req, onChunk)
    │   └─ Provider-specific request building
    │   └─ HTTP/stdio request transmission
    │   └─ Streaming response with onChunk callbacks
    └─ Return ChatResponse
    ↓
Post-turn processing
    ├─ Memory consolidation
    ├─ Task lifecycle
    └─ Notifications
```

### 9.2 Inbound Channel Message Flow

```
Channel → Inbound Message → Message Bus
    ↓
Consumer Debouncer
    ├─ Merge rapid messages (by session key)
    └─ Flush → processNormalMessage()
    ↓
Message Normalization
    ├─ Resolve agent from bindings
    ├─ Build session key (with thread/topic isolation)
    ├─ Resolve user ID (per-user or group-scoped)
    ├─ Persist metadata + collect contacts
    └─ Resolve merged tenant user
    ↓
Scheduler
    ├─ Register run in agent router
    ├─ Submit to scheduler (lane-based concurrency)
    └─ Run agent via agent loop
    ↓
Post-turn
    ├─ Task outcome resolution
    ├─ Memory consolidation
    └─ Outbound notifications
```

### 9.3 ACP Subprocess Lifecycle

```
Agent request with session_key option
    ↓
ACPProvider.Chat/ChatStream()
    ↓
ProcessPool.GetOrSpawn(sessionKey)
    ├─ Check if process exists → return
    └─ If not, spawn:
        ├─ exec.Command(binary, args...)
        ├─ Create pipes (stdin/stdout)
        ├─ Start process
        ├─ Create JSON-RPC Conn
        ├─ Call initialize()
        ├─ Call session/new()
        └─ Store in processes map
    ↓
proc.Prompt(ctx, content, onUpdate)
    ├─ Set inUse counter (prevents reaping)
    ├─ Install update callback
    ├─ Send session/prompt request
    ├─ Receive session/update notifications
    │   └─ Dispatch via onUpdate callback
    └─ Receive session/prompt response
    ↓
Reaping loop (every 30s)
    ├─ Skip processes with inUse > 0
    ├─ Kill idle processes (> 5 min inactive)
    └─ Allow room for new sessions
```

---

## 10. Extensibility Points

### 10.1 Adding a New Provider

1. **Implement Provider interface:**

```go
type MyProvider struct {
    apiKey string
    baseURL string
}

func (p *MyProvider) Chat(ctx, req) (*ChatResponse, error) { ... }
func (p *MyProvider) ChatStream(ctx, req, onChunk) (*ChatResponse, error) { ... }
func (p *MyProvider) DefaultModel() string { return "..." }
func (p *MyProvider) Name() string { return "myprovider" }
```

2. **Register in config:**

```go
registerProviders(registry, cfg, modelReg) {
    if cfg.Providers.MyProvider.APIKey != "" {
        registry.Register(providers.NewMyProvider(...))
    }
}
```

3. **Add to ModelRegistry if needed:**

```go
modelReg.Register(ModelSpec{
    ID: "mymodel-v1",
    Provider: "myprovider",
    ContextWindow: 100000,
    ...
})
```

### 10.2 Adding a New Tool Type

1. **Register in ToolRegistry**
2. **Implement RequestHandler** to execute tool
3. **Gate with ToolPolicy** for visibility/approval

### 10.3 Adding a New Channel

1. **Implement channel.Manager interface**
2. **Emit bus.InboundMessage** for incoming messages
3. **Subscribe to notifications** for outbound routing

---

## 11. Configuration Examples

### ACP Provider Configuration

```json5
{
  providers: {
    acp: {
      binary: "/path/to/acp-agent", // path or "claude" (Claude CLI)
      args: ["--model", "claude-opus"], // additional arguments
      work_dir: "~/.goclaw/acp-workspaces",
      idle_ttl: "5m", // reaping TTL
      perm_mode: "approve-all", // approve-all, approve-reads, deny-all
    },
  },
}
```

### Provider Failover Configuration

```go
// In agent other_config JSON:
{
  "chatgpt_oauth_routing": {
    "strategy": "round-robin",                        // or "primary-first"
    "extra_provider_names": ["openai", "anthropic"]  // failover candidates
  }
}
```

---

## 12. Performance Optimizations

### 12.1 Zero-Alloc Hot Paths

- **Sync.Map for reads:** Registry.Get() allocates zero in happy path
- **Composed middleware:** nil check skips allocation if no middleware
- **Round-robin counter:** atomic.Int64 for rotation without locking all reads

### 12.2 Streaming Efficiency

- **Buffered event channels:** onChunk callbacks don't block sender
- **Newline-delimited JSON:** SSE reader buffers in 256KB chunks
- **Lazy message building:** History loaded on-demand per session

### 12.3 Resource Management

- **Idle timeout for subprocesses:** Reaping loop cleans up after 5 min inactivity
- **Terminal output cap:** 10MB per terminal to prevent memory exhaustion
- **Debouncer flush:** Merges rapid messages (Telegram typing) → fewer scheduled runs

---

## 13. Security Considerations

### 13.1 Sandbox Enforcement

**Path Validation (ACP Tool Bridge):**

```go
resolved, err := tb.resolvePath(userPath)
// prevents: /etc/passwd, ../../../etc/passwd, symlink to /etc

// Implementation: filepath.Clean + IsAbs check + within workspace boundary
```

**Shell Deny Patterns:**

```go
denyPatterns := []string{
    "^rm\\b",           // block rm, rm -rf, etc.
    "^mkfs",            // block filesystem formatting
    "^dd\\b",           // block data destruction
    ":\\(/bin/bash\\|/bin/sh\\)",  // block shell execution
}
```

### 13.2 Tenant Isolation Enforcement

- **Where clause guard:** All DB queries include `WHERE tenant_id = $1`
- **Context propagation:** TenantID in context forces tenant validation
- **No admin bypass:** Even admin reads scoped to tenant (unless master scope)

### 13.3 API Key Management

- **Config secret file:** Encrypted via AES-256-GCM, not checked into Git
- **Database encryption:** Stored in encrypted columns
- **Access control:** Only master scope can list provider secrets

---

## 14. Testing & Verification

### Core Test Areas

1. **Provider Interface Compliance:**
   - Chat and ChatStream both work
   - Error handling (malformed responses, timeouts)
   - Token counting accuracy

2. **Registry Isolation:**
   - Tenant-specific providers override master tenant
   - Fallback to master tenant when not found
   - Round-robin state persists across router instances

3. **ACP Process Pool:**
   - Spawn on demand
   - Reaping after idle TTL
   - Crash recovery (respawn on next use)
   - Concurrent Prompt calls properly serialized

4. **Message Processing Pipeline:**
   - Session key generation (thread/topic isolation)
   - User ID scoping (per-user vs group-scoped)
   - Contact collection and merging
   - Metadata persistence

5. **Post-Turn Effects:**
   - Task status updates (error→fail, completed→auto-complete)
   - Unblocked task dispatch
   - Memory consolidation queueing
   - Notification routing

---

## 15. Glossary & Key Concepts

| Term              | Definition                                                          |
| ----------------- | ------------------------------------------------------------------- |
| **Provider**      | LLM backend (Anthropic, OpenAI, ACP, Claude CLI, etc.)              |
| **Registry**      | Maps provider names to instances, tenant-scoped                     |
| **ACP**           | Anthropic Console Proxy - JSON-RPC subprocess protocol              |
| **Tool Bridge**   | Handles agent→client requests (fs, terminal, permission) in ACP     |
| **Chat Request**  | `{messages, tools, model, options}` sent to provider                |
| **Chat Response** | `{content, thinking, toolCalls, finishReason, usage}` from provider |
| **Stream Chunk**  | Partial response `{content, thinking, done}` during streaming       |
| **Session Key**   | Unique identifier for conversation (agent:channel:peerkind:chat)    |
| **Session ID**    | ACP session identifier (internal to ACP subprocess)                 |
| **Run Request**   | `{sessionKey, userID, prompt, ...}` submitted to scheduler          |
| **Post-Turn**     | Effects processing after agent loop completes                       |
| **Consolidation** | Background memory aggregation (episodic → semantic)                 |
| **Tenant**        | Multi-tenant isolation boundary (per user/org)                      |
| **Message Bus**   | Event publisher for inter-component communication                   |
| **Scheduler**     | Lane-based concurrent run execution (main/subagent/cron)            |

---

## References

- **Provider interface:** `.resources/goclaw/internal/providers/types.go`
- **Registry:** `.resources/goclaw/internal/providers/registry.go`
- **ACP implementation:** `.resources/goclaw/internal/providers/acp/*.go`
- **Provider registration:** `.resources/goclaw/cmd/gateway_providers.go`
- **Message processing:** `.resources/goclaw/cmd/gateway_consumer_*.go`
- **Gateway deps:** `.resources/goclaw/cmd/gateway_deps.go`
- **Agent resolution:** `.resources/goclaw/internal/agent/resolver.go`
- **Provider resolution:** `.resources/goclaw/internal/providerresolve/agent_provider.go`

---

**Generated:** Analysis of GoClaw .resources/goclaw codebase for AGH (Agent Operating System in Go) reference implementation.
