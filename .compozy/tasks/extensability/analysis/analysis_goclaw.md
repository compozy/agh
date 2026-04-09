# GoClaw Analysis for AGH Extensibility

## Overview

GoClaw is a multi-tenant AI agent gateway written in Go 1.26 that routes end-user messages through a think-act-observe agent loop, executes tools against pluggable LLM providers, and streams responses back into multiple messaging channels (Telegram, Feishu, Zalo, Discord, WhatsApp). It is structured as a single control-plane binary with a six-layer stack: Client, Gateway, Agent Execution, Provider Bridge, Storage, and Shared Infrastructure.

**Key differentiators from AGH:**
- GoClaw is a multi-tenant SaaS gateway; AGH is a single-user local daemon
- GoClaw owns the LLM provider bridge (direct API calls); AGH spawns ACP-compatible agents as subprocesses
- GoClaw uses PostgreSQL + pgvector; AGH uses SQLite
- GoClaw has a central event bus (`bus.MessageBus`); AGH uses a typed Notifier pattern with direct function calls
- GoClaw runs its own agent loop in-process; AGH delegates execution to external agent processes (Claude Code, Codex, etc.)

**What makes GoClaw especially relevant:** Both are Go single-binary systems. GoClaw has solved many extensibility problems (dynamic tools, channel adapters, hook systems, MCP bridging, skills discovery) that AGH will need as it grows through Phase 2 (Memory/Skills/State) and Phase 3 (Agent network protocol).

---

## Key Features Analysis

| Feature | GoClaw Implementation | Classification for AGH | Rationale |
|---|---|---|---|
| **Think-Act-Observe Loop** | `internal/agent/loop.go` -- in-process LLM call + tool execution cycle with parallel tool dispatch, iteration limits, budget guards | **N/A (different model)** | AGH delegates execution to external agents via ACP/JSON-RPC over stdio. AGH does not own the agent loop -- the spawned agent (Claude Code, etc.) does. However, the iteration/budget guard patterns are worth borrowing for session-level cost control. |
| **Tool Registry** | `tools.Registry` -- unified `Tool` interface (`Name/Schema/Invoke`) for built-in, dynamic, and MCP-sourced tools | **CORE** | AGH already has a tools concept through ACP. A unified tool registry interface that normalizes tools regardless of source (built-in, MCP, dynamic) should be core infrastructure. |
| **MessageBus (Event Bus)** | `internal/bus/bus.go` -- buffered channels (1000-slot), inbound/outbound message routing, event broadcasting with per-subscriber filtering | **CORE (limited)** | AGH explicitly rejects a generic event bus in its architecture principles ("no event bus, no NATS"). AGH uses a typed Notifier pattern instead. However, the *specific patterns* from GoClaw's bus -- deduplication helpers, debounce helpers, non-blocking publish with drop-on-full -- are worth adopting as utilities within AGH's existing Notifier. |
| **Channel Adapter System** | `Channel` interface (`Start/Stop/Send/Health`) with 5 implementations (Telegram, Discord, etc.), `ChannelManager` orchestrator, `RunContext` for streaming state | **EXTENSION** | AGH's primary interfaces are HTTP/SSE (web UI) and UDS (CLI). Messaging platform adapters are clearly extension territory -- they add reach without changing core behavior. The `Channel` interface pattern is excellent for plugin design. |
| **Hook System (Loop-Level)** | Typed function-pointer fields on `Loop` struct (`EnsureUserFilesFunc`, `SeedUserFilesFunc`, `ContextFileLoaderFunc`, `BootstrapCleanupFunc`) with nil-check invocation | **CORE** | AGH should adopt this pattern for session lifecycle hooks. Function-pointer fields are compile-time safe, zero-reflection, and fit AGH's "direct function calls through interfaces" principle. Perfect for hooks like `OnSessionStart`, `OnSessionEnd`, `OnEventRecorded`, `OnMemoryConsolidation`. |
| **Hook System (Handler-Level)** | Pre/post hooks on RPC method handlers (`preValidate`, `postTurn`) | **CORE** | Handler-level hooks for the API layer (HTTP/UDS) are core infrastructure. AGH's `api/httpapi` and `api/udsapi` should support pre/post hooks for audit, analytics, and custom validation. |
| **Dynamic/Custom Tools** | `DynamicTool` wrapping `CustomToolDef` -- shell command templates with `{{.key}}` substitution, per-tool timeouts, encrypted env vars | **EXTENSION** | Shell-command-based tools are an extension mechanism, not core. AGH should provide a `DynamicTool` plugin point but not bake shell execution into the core. The template rendering and shell escaping patterns are reusable. |
| **MCP Bridge** | `mcp.Manager` with connection pooling, three transports (stdio/SSE/streamable-HTTP), tool namespacing (`mcp__{server}__{tool}`), hybrid search mode (40-tool threshold + BM25 lazy loading), per-agent/user access grants | **CORE** | AGH already spawns ACP agents via stdio -- MCP bridge is a natural extension of the same pattern. Tool namespacing, connection management, health monitoring, and hybrid search mode should be core infrastructure since MCP is becoming the standard interop protocol. |
| **Hybrid Tool Search** | BM25 search over deferred tools when tool count > 40, with `mcp_tool_search` and `mcp_tool_activate` meta-tools | **CORE** | Critical for scaling. As AGH accumulates tools from multiple MCP servers, the context budget pressure becomes real. The search-then-activate pattern should be core. |
| **Memory (Vector Embeddings)** | pgvector-backed semantic search, configurable chunking with overlap, cosine similarity retrieval, top-K injection into system prompt | **CORE** | AGH already has `internal/memory` with dual-scope persistent memory. GoClaw's chunking strategy (configurable chunk size + overlap), dedup-by-hash, and the integration pattern (search at run-start, inject into context) validate AGH's approach. |
| **Knowledge Graph** | LLM-based entity/relation extraction, PostgreSQL storage, BFS path finding, fuzzy entity dedup | **EXTENSION** | Knowledge graphs are expensive to build and maintain (require LLM calls for extraction). This is a Phase 2+ extension that sits on top of the memory system. AGH should provide the interface but not bundle the implementation. |
| **Skills System** | Document-based skills (`SKILL.md` with YAML frontmatter), five-tier loader hierarchy, BM25 + pgvector hybrid discovery, hot-reload, agent self-evolution | **CORE (loader + discovery) / EXTENSION (self-evolution)** | AGH already has `internal/skills` with catalog and loader. GoClaw validates the search-then-load pattern and the separation of skills (procedural knowledge) from tools (executable capabilities). The loader hierarchy and discovery engine are core. Self-evolution (nudges at 70%/90% budget) is an extension. |
| **LLM Provider Bridge** | `Provider` interface (`Chat/ChatStream/DefaultModel/Name`), 5 implementations, `providers.Registry`, provider-specific workarounds (thinking passback, token clamping, synthetic streaming) | **N/A (different model)** | AGH does not own the LLM call -- it delegates to ACP agents. However, the `Registry` pattern (lazy map + RWMutex, O(1) lookup) and the encrypted credential storage pattern are directly applicable to AGH's agent/driver management. |
| **Agent Teams and Delegation** | Subagents (self-cloned goroutines), delegation (permission-gated inter-agent handoffs via AgentLinks), team coordination (Kanban task boards with Lead/Member roles) | **EXTENSION** | Multi-agent coordination is Phase 3 territory. AGH should define the interfaces (delegate, handoff, team) but implement them as extensions. The subagent pattern (spawn a background goroutine running the same loop) maps to AGH spawning additional ACP sessions. |
| **Cron and Scheduling** | In-process scheduler with three modes (`cron`/`at`/`every`), JSON file persistence, exponential backoff retry, 200-entry ring buffer log | **EXTENSION** | Scheduling is an extension. AGH could expose a `Scheduler` interface in core but the cron implementation should be a plugin. GoClaw's pattern of dispatching scheduled jobs through the same agent loop (as synthetic `RunRequest`) is elegant and worth copying. |
| **Heartbeat System** | Per-agent periodic self-check with `HEARTBEAT.md` checklist, `HEARTBEAT_OK` suppression, stagger offset, active-hours window | **EXTENSION** | Specialized scheduling for agent self-monitoring. Extension built on top of the scheduler interface. The `HEARTBEAT_OK` suppression pattern is clever for silent monitoring. |
| **Context Files and Agent Identity** | `SOUL.md`, `IDENTITY.md`, `USER.md`, `BOOTSTRAP.md` -- virtual filesystem interceptor routes agent file reads/writes to DB, per-user vs shared scoping | **EXTENSION** | AGH already has workspace management. The context file interception pattern (virtual FS layer that redirects specific filenames to a different backend) is interesting but heavy. AGH's simpler approach of injecting context through the ACP protocol is more appropriate for its architecture. |
| **Shell Execution Security** | Four-gate pipeline: deny patterns, credentialed binary detection, approval flow, sandbox routing. Output scrubbing with `ScrubCredentials`. Docker sandbox with `--read-only --cap-drop ALL --network none`. | **EXTENSION** | AGH delegates execution to external agents, so shell security is the agent's responsibility. However, if AGH adds dynamic tool execution, the deny-pattern and credential-scrubbing patterns should be borrowed. |
| **Text-to-Speech** | `tts.Manager` with 4 provider backends (OpenAI, ElevenLabs, Edge, MiniMax), AutoMode triggers, `TtsTool` for agent-initiated synthesis | **EXTENSION** | Clearly an extension. No impact on AGH's core. |
| **RBAC and Security** | 5-layer permission cascade (role hierarchy, API key scopes, global tool policy, per-agent tool policy, owner-only tools), AES-256-GCM encryption at rest, input guard (detection-only) | **CORE (partial)** | AGH needs authentication and authorization for its HTTP/UDS APIs. The role hierarchy pattern, API key hashing (SHA-256), and encrypted credential storage are core. The full 5-layer cascade is overkill for a single-user daemon but the patterns are sound for when AGH supports multiple users. |
| **Audit Logging** | Append-only `audit_logs` table, structured `slog` output, tenant-scoped, queryable via API | **CORE** | AGH already has `internal/observe` for event recording. Audit logging of security-relevant actions (config changes, session management) should be core. |
| **Rate Limiting** | Per-IP/per-token token-bucket rate limiter at the gateway | **EXTENSION** | Single-user daemon does not need rate limiting. Extension for when AGH supports remote access. |
| **Multi-Tenant PostgreSQL** | `context.Context` propagation of `tenant_id`, RLS on all tables, encrypted columns for secrets | **N/A** | AGH is local-first, single-tenant. The context propagation pattern is good Go practice but multi-tenancy is out of scope. |
| **OpenAI-Compatible API** | `POST /v1/chat/completions` drop-in replacement for OpenAI clients | **EXTENSION** | Useful for interoperability but not core to AGH's mission. Could be a thin extension layer over AGH's HTTP API. |
| **WebSocket v3 Protocol** | Frame-based protocol with `RequestFrame`/`ResponseFrame`/`EventFrame`, method router, per-client write channels | **N/A** | AGH uses HTTP/SSE + UDS, not WebSocket RPC. The event frame pattern and per-client write channel pattern are already addressed by AGH's SSE implementation. |
| **Inbound Debounce** | Per-chat-ID debounce timer (500ms) to consolidate rapid user messages | **CORE** | Debouncing is essential for AGH's HTTP/SSE interface. When a user types rapidly in the web UI, debouncing prevents N session runs. Should be a small utility in core. |
| **Message Dedup** | Content-hash dedup with 5-second window to prevent duplicate processing on reconnects | **CORE** | Important for AGH's SSE reconnection scenarios. A small utility. |
| **Connection Health Monitoring** | Per-channel/per-MCP-server health checks with status tracking, reconnection with exponential backoff | **CORE** | AGH spawns subprocesses -- monitoring their health, detecting crashes, and reconnecting is core infrastructure. The health check pattern with `ChannelHealth` struct and the exponential backoff retry are directly applicable. |

---

## Architectural Patterns Worth Adopting

### 1. Registry Pattern with Lazy Loading and TTL Cache

GoClaw's `agent.Router` is a lazy-loading cache keyed by agent ID with a 10-minute TTL:

```go
type Router struct {
    agents   map[string]*agentEntry
    mu       sync.RWMutex
    resolver ResolverFunc  // lazy-create from DB
    ttl      time.Duration
}
```

**Applicability to AGH:** AGH's `session.Manager` could adopt this pattern for agent driver caching. When AGH spawns an ACP agent, the driver instance could be cached and reused across sessions for the same agent type, with TTL-based eviction for config changes.

**Classification: CORE pattern** -- fits AGH's existing `session/` package.

### 2. Typed Function-Pointer Hooks (Not Event Bus)

GoClaw uses function-pointer fields on structs for lifecycle hooks:

```go
type Loop struct {
    ensureUserProfile EnsureUserProfileFunc
    seedUserFiles     SeedUserFilesFunc
    loadContextFiles  ContextFileLoaderFunc
}
```

Nil-check before invocation makes hooks optional. No reflection, no event bus, compile-time type safety.

**Applicability to AGH:** This is exactly aligned with AGH's "direct function calls through interfaces" and "no event bus" principles. AGH's `session.Manager`, `observe.Recorder`, and `memory.Manager` should expose typed hook fields for extension points like:

- `OnSessionCreated func(ctx, session) error`
- `OnEventRecorded func(ctx, event) error`
- `OnConsolidationComplete func(ctx, results) error`

**Classification: CORE pattern** -- directly implements AGH's architectural principles.

### 3. Tool Interface Unification

GoClaw normalizes all tools (built-in, dynamic shell, MCP-sourced) behind a single interface:

```go
type Tool interface {
    Name() string
    Schema() json.RawMessage
    Invoke(ctx context.Context, args map[string]any) (string, error)
}
```

The agent loop does not know where a tool came from. This is achieved through wrapper types like `BridgeTool` for MCP and `DynamicTool` for shell commands.

**Applicability to AGH:** AGH communicates tools to agents via ACP protocol, but it still needs to manage tool registries for MCP bridging, skill-provided tools, and dynamic tools. A unified `Tool` interface in AGH would normalize these sources before exposing them to ACP agents.

**Classification: CORE pattern** -- essential for Phase 2 extensibility.

### 4. Parallel Execution with Deterministic Ordering

GoClaw dispatches tool calls in parallel but sorts results back to original order:

```go
for i, tc := range toolCalls {
    go func(tc, idx int) {
        result := executor.Invoke(ctx, tc)
        resultsChan <- indexedResult{idx: idx, result: result}
    }(tc, i)
}
sort.Slice(results, func(i, j int) bool {
    return results[i].idx < results[j].idx
})
```

**Applicability to AGH:** Useful when AGH needs to execute multiple MCP tool calls or process multiple events concurrently. The `indexedResult` pattern preserves ordering cheaply.

**Classification: CORE utility** -- small helper in `internal/procutil` or similar.

### 5. Non-Blocking Publish with Drop-on-Full

GoClaw's `TryPublishInbound()` is a non-blocking variant that drops messages when the buffer is full rather than blocking producers:

```go
select {
case bus.inbound <- msg:
    return true
default:
    slog.Warn("inbound buffer full, message dropped")
    return false
}
```

**Applicability to AGH:** AGH's Notifier pattern should support this for SSE event delivery. A slow web client should not back-pressure the session execution. AGH's SSE helpers in `api/core` could adopt this.

**Classification: CORE pattern** -- protects core from slow consumers.

### 6. Stagger Offset for Periodic Tasks

GoClaw uses MD5 hash of agent ID to deterministically spread periodic tasks across a time window, preventing thundering herd:

```go
func StaggerOffset(agentID string) time.Duration {
    hash := md5.Sum([]byte(agentID))
    offset := binary.BigEndian.Uint32(hash[:4]) % 30
    return time.Duration(offset) * time.Second
}
```

**Applicability to AGH:** Useful for AGH's dream consolidation triggers when multiple workspaces need consolidation around the same time.

**Classification: CORE utility** -- small helper for scheduling.

### 7. Context Propagation Over Global State

GoClaw propagates tenant ID through `context.Context` rather than a global singleton:

```go
func WithTenantID(ctx context.Context, id uuid.UUID) context.Context {
    return context.WithValue(ctx, ctxKeyTenantID, id)
}
```

**Applicability to AGH:** AGH already uses `context.Context` as first argument everywhere. This validates the approach. AGH should consider propagating session ID, workspace ID, and request ID through context for observability.

**Classification: CORE pattern** -- already partially adopted.

---

## Extension System Insights

### Dynamic Tools: Shell-Command Extension Point

GoClaw's `DynamicTool` is the most accessible extension mechanism -- operators define tools as shell command templates stored in the database:

```
Command: "curl -s {{.url}} | jq '.results[]'"
Parameters: {"url": {"type": "string"}}
TimeoutSeconds: 30
```

**Insight for AGH:** AGH should provide a similar mechanism where users can define tools via TOML config that get exposed to ACP agents through the protocol. The key security patterns to borrow:
- Shell escaping via single-quote wrapping
- Per-tool configurable timeouts with process-group kill
- Encrypted environment variables for credential injection
- Output scrubbing with both static patterns (API key regexes) and dynamic patterns (injected credential values)

**Recommendation:** Define a `DynamicToolProvider` extension interface in AGH that can be implemented by a shell-command plugin, an HTTP-webhook plugin, or a WASM plugin.

### Channel Adapters: The Minimal Interface

GoClaw's `Channel` interface is remarkably small:

```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg OutboundMessage) error
    Health() ChannelHealth
}
```

**Insight for AGH:** This is the gold standard for a plugin interface -- four methods, clear lifecycle (`Start`/`Stop`), a single operation (`Send`), and a health probe. AGH should define similarly minimal interfaces for its extension points:

- `AgentDriver` (already exists in `session/` -- `Start/Stop/SendMessage`)
- `ToolProvider` -- `ListTools/InvokeTool/Health`
- `MemoryBackend` -- `Store/Search/Delete/Health`
- `NotificationSink` -- `Send/Health`

The `Health()` method returning a struct with `Status`, `LastError`, `LastActivity` is a pattern worth standardizing across all AGH extensions.

### Hook/Event System: Function Pointers > Event Bus

GoClaw's hook system uses two complementary patterns:

1. **Loop-level hooks** -- typed function fields on structs, nil-checked before invocation
2. **Bus-level broadcasting** -- buffered channels with subscriber filtering

**Insight for AGH:** AGH's explicit rejection of event buses is correct for its scope. The function-pointer hook pattern is the right choice. However, AGH should formalize the hook taxonomy:

| Lifecycle Point | Hook Signature | Where |
|---|---|---|
| Session created | `func(ctx, *Session) error` | `session.Manager` |
| Session ended | `func(ctx, *Session) error` | `session.Manager` |
| Event recorded | `func(ctx, *Event) error` | `observe.Recorder` |
| Memory stored | `func(ctx, *MemoryEntry) error` | `memory.Manager` |
| Dream triggered | `func(ctx, *DreamRequest) error` | `memory/consolidation` |
| Skill loaded | `func(ctx, *Skill) error` | `skills.Catalog` |
| Agent spawned | `func(ctx, *AgentProcess) error` | `acp.Driver` |
| Agent crashed | `func(ctx, *AgentProcess, error) error` | `acp.Driver` |

Each hook is a `func` field on the owning struct, set via a `With*` functional option at construction time. Nil hooks are no-ops.

### MCP Bridge: Connection Pooling and Hybrid Search

GoClaw's MCP bridge solves three problems AGH will face:

1. **Connection management** -- pooling server connections across sessions, health monitoring with exponential backoff, cleanup on server crash
2. **Tool namespacing** -- `mcp__{server}__{tool}` prevents collisions when multiple MCP servers expose tools with the same name
3. **Context budget management** -- when tool count > 40, switch to hybrid mode where only the top 40 are inline and the rest are searchable via BM25

**Insight for AGH:** AGH already spawns ACP agents via stdio -- the same transport used for MCP stdio servers. The `mcp.Manager` pattern (server state with atomic connected flag, reconnection with backoff, health checks every 30s) maps directly to AGH's `acp.Driver` lifecycle. Key recommendations:

- Adopt `mcp__{server}__{tool}` namespacing for tool deduplication
- Implement the 40-tool hybrid search threshold -- AGH's ACP agents have finite context windows
- Pool MCP server connections across sessions in the `daemon/` composition root
- Use the `BridgeTool` wrapper pattern to present MCP tools through AGH's native tool interface

### Skills: Search-Then-Load Pattern

GoClaw's skills are document-based (`SKILL.md`) rather than code-based. Discovery uses BM25 + pgvector hybrid search. Loading injects the skill content into the agent's context window.

**Insight for AGH:** AGH already has `internal/skills` with a catalog and loader, plus `internal/skills/bundled` for built-in skills. GoClaw validates that skills should be:
- Filesystem-based (markdown with YAML frontmatter)
- Discoverable via search (not eagerly loaded)
- Injected as context (not executed as code)
- Hierarchical (workspace > project > global > bundled)

The self-evolution mechanism (agent creates new skills from execution history) is fascinating but should be an extension -- it requires monitoring agent execution patterns and triggering skill creation, which is complex orchestration that does not belong in AGH's minimal core.

### Deduplication and Debounce Helpers

GoClaw provides two small but critical utilities:

1. **`DedupeHelper`** -- content-hash dedup with configurable time window (5s default)
2. **`InboundDebounceHelper`** -- per-key debounce timer (500ms default) that consolidates rapid inputs

**Insight for AGH:** These should be standalone utilities in `internal/` (perhaps `internal/rateutil` or alongside `internal/procutil`). They protect AGH from:
- SSE reconnection storms (dedup)
- Rapid user input in the web UI (debounce)
- Duplicate webhook deliveries from external systems

Both are small, self-contained, and have zero dependencies -- perfect for AGH's core.

### Health Monitoring Pattern

GoClaw standardizes health across all subsystems:

```go
type ChannelHealth struct {
    Status       string    // "connected" | "connecting" | "disconnected" | "error"
    LastError    string
    LastActivity time.Time
    MessageCount int64
}
```

**Insight for AGH:** AGH should define a standard `Health` struct in a shared package and require every subsystem to implement it:
- ACP agent processes: is the process alive, last event time, error count
- MCP server connections: connected/disconnected, last tool call, reconnect attempts
- SQLite databases: writable, size, last vacuum
- Memory system: consolidation status, entry count

This feeds directly into AGH's `/health` endpoint and the `observe` package.

---

## Summary: What AGH Should Take from GoClaw

### Adopt as CORE (build into AGH's minimal robust core)

1. **Typed function-pointer hooks** on `session.Manager`, `observe.Recorder`, `memory.Manager`, `acp.Driver`
2. **Unified Tool interface** for normalizing MCP tools, built-in tools, and dynamic tools
3. **MCP bridge with connection pooling**, health monitoring, namespacing, and hybrid search
4. **Dedup and debounce helpers** as standalone utilities
5. **Standardized Health struct** across all subsystems
6. **Non-blocking publish with drop-on-full** for SSE event delivery
7. **Parallel execution with deterministic ordering** as a utility
8. **Skills search-then-load pattern** (validates AGH's existing approach)

### Adopt as EXTENSION (plugin/extension system)

1. **Channel adapters** -- define the `Channel` interface, let extensions implement Telegram/Discord/etc.
2. **Dynamic shell tools** -- define `DynamicToolProvider`, let a shell plugin implement it
3. **Knowledge graph** -- define the interface, let an extension provide LLM-based extraction
4. **Cron/Scheduling** -- define `Scheduler` interface, let an extension implement it
5. **Agent teams/delegation** -- define coordination interfaces for Phase 3
6. **TTS** -- pure extension, no core impact
7. **Heartbeat system** -- extension on top of scheduler
8. **Skills self-evolution** -- extension on top of skills core
9. **Rate limiting** -- extension for multi-user scenarios
10. **OpenAI-compatible API** -- thin extension layer

### Key Design Principles Validated by GoClaw

- **Small interfaces win.** GoClaw's `Channel` (4 methods), `Provider` (4 methods), and `Tool` (3 methods) interfaces are the right granularity. AGH should target 3-5 methods per extension interface.
- **Nil-check hooks beat event buses.** GoClaw's function-pointer hooks are zero-overhead when unused, compile-time safe, and require no subscription management. This aligns perfectly with AGH's "no event bus" principle.
- **Namespace everything.** GoClaw's `mcp__{server}__{tool}` pattern prevents collisions as the tool catalog grows. AGH should adopt this early.
- **Health is not optional.** Every subsystem in GoClaw reports health. AGH should make `Health()` a required method on every extension interface.
- **Search beats eager loading.** GoClaw's BM25 hybrid search for both skills and MCP tools keeps context budgets manageable. AGH should adopt this pattern before the tool/skill catalog grows large.
