# Cross-Cutting Analysis: Universal Patterns for AGH Extensibility

This analysis synthesizes concepts from three knowledge domains -- AI agent harness patterns, agent-to-agent network protocols, and AI memory systems -- to identify what belongs in AGH's minimal core, what should be an extension, and what should be a protocol-level interface bridging the two.

---

## Universal Core Patterns

These patterns appear in every agent framework surveyed and represent the irreducible kernel that AGH must implement directly in its daemon.

### 1. The Agentic Loop (CORE)

Every agent system, from the simplest ReAct implementation to full multi-agent orchestrations, relies on a tight cycle: receive input, construct prompt, call model, parse response, dispatch tool calls, append results, repeat. This loop is the execution primitive on which everything else is built.

AGH already implements this via its ACP subprocess model (spawn agent, JSON-RPC over stdio, event persistence). The loop itself must remain core. However, the loop's *policy* -- how many iterations are allowed, what happens on tool errors, when to compact context -- should be configurable per-session via the TOML config layer.

**What specifically must be core:**
- The turn cycle (prompt construction -> model call -> response parsing -> tool dispatch -> observation append)
- Step budgets and hard termination limits
- Error classification (transient vs. permanent) and basic retry
- Streaming response handling
- Context window occupancy tracking (token counting)

### 2. Session Lifecycle and State Machine (CORE)

Every framework implements a session/task state machine. A2A defines: submitted -> working -> input-required -> completed/failed/canceled. LangGraph has checkpointed graph state. CrewAI tracks task status per crew member. The state machine is universal because agents are inherently stateful processes.

AGH's `internal/session` package already owns this. The state machine must be core, but the *set of states* should be extensible. The base states (created, running, paused, completed, failed) are universal. Extensions should be able to register custom states (e.g., "awaiting-human-approval", "delegated-to-peer") without modifying the session package.

**What specifically must be core:**
- State transitions with validation (no illegal jumps)
- Session creation, suspension, resumption, termination
- Event persistence per session (the `sessiondb` pattern AGH already has)
- Session metadata (agent type, workspace, creation time)

### 3. Tool Dispatch and Schema Registry (CORE)

Tool dispatch is the action primitive. Every agent loop terminates in either a tool call or a final answer. The dispatch mechanism -- validate the call against a schema, execute it, capture the result -- is universal. JSON Schema for tool definitions has converged as the industry standard (MCP, OpenAI function calling, Anthropic tool use all use it).

**What specifically must be core:**
- A tool registry that holds (name, description, JSON Schema, handler) tuples
- Schema validation of tool call arguments before dispatch
- Execution with timeout, result capture, and error wrapping
- Parallel tool call support (batch independent calls)
- Tool call audit logging (every invocation recorded)

**What should NOT be core:**
- Specific tool implementations (file read, web search, etc.) -- these are extensions
- Tool discovery from external sources -- this is a protocol concern (MCP)

### 4. Permission Model (CORE)

Every production harness implements layered permissions: tool-level (which tools exist), path-level (allowed directories), command-level (banned operations), and approval gates (human confirmation for writes). The OWASP LLM Top 10 ranks "Excessive Agency" (LLM06) as a critical risk. Permission enforcement belongs in the core because it is the last defense against prompt injection escalation.

**What specifically must be core:**
- Tool-level allow/deny lists
- Approval gates that pause execution for external confirmation
- Per-session capability scoping (an agent gets only the tools it needs)
- Credential isolation (tools manage their own secrets, agents never see raw credentials)

### 5. Event Recording and Observability (CORE)

Every framework requires tracing. OpenTelemetry GenAI semantic conventions are converging as the standard. AGH's `internal/observe` package already handles event recording. This must be core because debugging non-deterministic agent behavior is impossible without traces.

**What specifically must be core:**
- Per-step event recording (tool calls, model calls, state transitions)
- Token usage tracking per call and per session
- Latency measurement per operation
- Structured logging with session/agent correlation IDs
- Cost tracking (input tokens x price + output tokens x price)

### 6. Context Assembly Pipeline (CORE)

Context engineering is the highest-leverage skill in building production agents. The pipeline that assembles the model's context window -- system prompt + project docs + session state + tool results + user message -- is universal. The ACE framework (Agentic Context Engineering) formalizes this as selection -> formatting -> timing -> lifecycle.

**What specifically must be core:**
- Layered context assembly (system prompt layer, memory layer, tool results layer, user message layer)
- Token budget tracking and enforcement per layer
- Compaction triggers (when occupancy exceeds threshold, compress older turns)
- Basic summarization-based compaction of conversation history

---

## Extension Point Taxonomy

Extensions are capabilities that should plug into AGH's core interfaces without modifying the daemon. The analysis reveals six natural extension categories.

### Category 1: Agent Drivers (Extension)

Different agent runtimes (Claude Code, Codex, Gemini CLI, custom agents) each have their own subprocess protocol, prompt format, and tool-calling convention. AGH already abstracts this via the `AgentDriver` interface in `internal/acp`.

**Extension interface:** `AgentDriver` -- spawn process, send message, receive response, shutdown.
**Why extension, not core:** The set of supported agents grows without bound. Each driver is independent. New drivers require no changes to session management, observability, or permissions.

### Category 2: Memory Backends (Extension)

The knowledge base analysis reveals a universal pattern: pluggable memory backends behind a common interface. Mem0 (vector + graph), Zep (temporal KG), Letta (self-editing blocks), Redis (warm tier), pgvector (cold tier), SQLite+FTS5 (local), and file-based markdown wikis all serve as memory backends. AGH's `internal/memory` package should define the interface; backends are extensions.

**Extension interface:**
```go
type MemoryBackend interface {
    Store(ctx context.Context, entry MemoryEntry) (string, error)
    Recall(ctx context.Context, query string, opts RecallOptions) ([]MemoryEntry, error)
    Forget(ctx context.Context, entryID string) error
    Consolidate(ctx context.Context) (int, error)
}
```

**Why extension, not core:** The diversity of backends (vector stores, knowledge graphs, file-based wikis, cloud services) is enormous and growing. Each makes different trade-offs (latency vs durability, semantic search vs keyword search, graph traversal vs flat retrieval). The consolidation algorithm (dream triggers, importance-weighted pruning, hierarchical compression) also varies by deployment.

**Backends AGH should ship:**
- SQLite+FTS5 (local default, already aligned with AGH's SQLite architecture)
- File-based markdown (for the Karpathy pattern / CLAUDE.md approach)

**Backends that should be external plugins:**
- Vector store integration (Chroma, Qdrant, pgvector)
- Knowledge graph (Zep/Graphiti, Neo4j)
- Cloud memory services (Mem0, OpenMemory)

### Category 3: Tool Providers (Extension)

Tools are the most natural extension point. MCP has proven that tools can be exposed as independent servers with JSON-RPC + JSON Schema. AGH should treat every tool as a provider that registers with the core tool registry.

**Extension types:**
- **Built-in tools:** File operations, shell execution, basic search -- compiled into the binary
- **MCP servers:** External processes exposing tools via MCP protocol
- **Plugin tools:** Dynamically loaded tool implementations (Go plugins or subprocess)

**Why extension, not core:** The tool catalog is unbounded (450+ MCP servers exist already). The dispatch mechanism is core; the tools themselves are not.

### Category 4: Orchestration Strategies (Extension)

Agent Architecture Patterns reveals at least seven orchestration patterns: ReAct, Plan-and-Execute, Orchestrator-Workers, Evaluator-Optimizer, Routing, Parallelization, and Reflection. Each is a different policy for how the agentic loop operates at the multi-step level.

**Why extension, not core:** The basic loop is core. The strategy that governs *how* the loop runs (single agent vs. orchestrated multi-agent, sequential vs. parallel, with or without replanning) varies by task type. AGH should provide a simple default (single ReAct loop) and allow orchestration strategies to be plugged in.

**Extension interface pattern:** An orchestration strategy receives a task description and produces a plan (sequence of steps, potentially with parallelism). The core loop executes each step. The strategy can observe results and replan.

### Category 5: Skill Packages (Extension)

Skills are reusable packages of instructions that teach agents domain-specific workflows. They include trigger conditions, procedural instructions, tool preferences, and quality criteria. The Voyager skill library, CrewAI agent roles, and Claude Code skills all implement this pattern.

**Extension interface:** A skill is a (trigger, instructions, tools, verification) tuple loaded from a file or registry.
**Why extension, not core:** Skills are domain-specific content, not infrastructure. AGH's core provides the loading mechanism and dispatch (already in `internal/skills`); the actual skill definitions are extensions.

### Category 6: Notification and Output Channels (Extension)

How events and results reach the outside world (HTTP/SSE for web UI, UDS for CLI, webhooks, Slack, email) is inherently extensible. AGH already separates `httpapi` and `udsapi`.

**Why extension, not core:** New output channels (WebSocket, gRPC, push notifications) should be addable without modifying the daemon core. The `Notifier` pattern AGH uses is the right abstraction.

---

## Protocol Layer Recommendations

Protocols are standardized interfaces that sit between core and extensions. They define *how* communication happens without specifying *what* is communicated. AGH should implement protocol support in core and let extensions implement specific protocol endpoints.

### Protocol 1: MCP (Model Context Protocol) -- IMPLEMENT IN CORE

MCP is the universal agent-to-tool protocol. Cross-vendor adoption (Anthropic, OpenAI, Google, Microsoft, GitHub) means it is the de facto standard. AGH should be an MCP host that can connect to any MCP server.

**Core responsibilities:**
- MCP client implementation (JSON-RPC 2.0 over stdio and SSE/Streamable HTTP)
- Connection lifecycle management (initialize, capability exchange, operation, shutdown)
- Tool schema discovery from MCP servers and registration in the core tool registry
- Resource reading from MCP servers
- Credential management for authenticated MCP servers

**Why core, not extension:** MCP is the integration layer. Without it, AGH cannot access the 450+ tool ecosystem. Every agent session benefits from MCP connectivity.

### Protocol 2: A2A (Agent-to-Agent) -- DEFINE INTERFACE IN CORE, IMPLEMENT AS EXTENSION

A2A handles inter-agent communication: discovery via Agent Cards, task delegation with lifecycle management, streaming results, and push notifications. AGH should define the interface for agent-to-agent communication in core but implement the actual A2A protocol handler as an extension.

**Core interface:**
```go
type AgentPeer interface {
    Discover(ctx context.Context, query CapabilityQuery) ([]AgentCard, error)
    Delegate(ctx context.Context, card AgentCard, task TaskSpec) (TaskHandle, error)
    Cancel(ctx context.Context, handle TaskHandle) error
}
```

**Why interface-in-core, implementation-as-extension:** A2A is still maturing (v0.3 as of 2026). AGH should not couple its core to a protocol that may evolve significantly. But the *concept* of peer agent communication is universal -- the interface should be stable.

### Protocol 3: Agent Card / Capability Manifest -- DEFINE IN CORE

Every discovery protocol (A2A Agent Cards, AGNTCY, ANP) requires that agents publish a capability manifest. AGH should define its own agent card format that describes what an AGH-managed agent can do, compatible with A2A Agent Card structure.

**Core responsibilities:**
- Generate Agent Cards from agent configuration (capabilities, skills, supported input/output modes)
- Publish Agent Cards via well-known URI (`.well-known/agent-card.json`)
- Parse incoming Agent Cards for peer discovery

### Protocol 4: Context Transfer / Handoff -- DEFINE IN CORE

When one agent hands off to another, context must transfer. The analysis of handoff patterns reveals four strategies: full history pass-through, summary, structured snapshot, and schema-typed payload. AGH should define a handoff protocol that supports all four.

**Core responsibilities:**
- Handoff primitive (transfer control + context from session A to session B)
- Context packing (serialize current session state into a transferable format)
- Context unpacking (deserialize received context into a new session's starting state)

### Protocol 5: Observability Wire Format -- ALIGN WITH OPENTELEMETRY

OpenTelemetry GenAI semantic conventions are the emerging standard for agent observability. AGH's event recording should emit data in OTel-compatible format so traces can flow to Langfuse, Grafana, Datadog, or any OTel-compatible backend.

**Core responsibilities:**
- OTel-compatible span emission for model calls, tool calls, and state transitions
- Standard attribute naming (`gen_ai.system`, `gen_ai.request.model`, `gen_ai.usage.input_tokens`, etc.)
- Trace context propagation across session boundaries and agent delegations

---

## Memory System Architecture

The memory analysis reveals a three-tier architecture, a four-type cognitive taxonomy, and a pluggable backend pattern that AGH should adopt.

### Three-Tier Memory Hierarchy

| Tier | Latency | Contents | AGH Implementation |
|------|---------|----------|-------------------|
| **Hot (in-context)** | 0ms | Current turn, recent tool results, active scratchpad | Managed by the context assembly pipeline in core |
| **Warm (session-scoped)** | <10ms | Conversation history, session state, recent memories | SQLite per-session DB (AGH's existing `sessiondb`) |
| **Cold (persistent)** | 10-100ms | User preferences, project knowledge, cross-session facts | Global memory store via `MemoryBackend` interface |

### Four Memory Types (CoALA Taxonomy)

| Type | What It Stores | AGH Mapping |
|------|---------------|-------------|
| **Working memory** | Current context window contents | Core: context assembly pipeline |
| **Episodic memory** | Specific past events with timestamps | Extension: event store + recall queries |
| **Semantic memory** | General facts and knowledge | Extension: knowledge graph or vector store |
| **Procedural memory** | Reusable skills and workflows | Extension: skills package + bundled skills |

### Memory Consolidation as a Core Concern

The "dream" consolidation pattern (AGH's `internal/memory/consolidation`) is correctly placed. Consolidation -- the process of extracting high-value facts from raw session transcripts, merging duplicates, resolving contradictions, and pruning stale entries -- is a cross-cutting concern that every memory backend benefits from. The consolidation *trigger* (when to run) and *pipeline* (extract -> merge -> prune -> store) should be core. The specific *algorithm* (LLM-based summarization, importance-weighted pruning, hierarchical compression) should be configurable.

### Dual-Scope Memory (Global + Workspace)

AGH's existing dual-scope model (global memories that follow the user, workspace memories tied to a project directory) maps directly to the production patterns observed:

- **Global scope** = User profile memory (preferences, learned patterns, cross-project knowledge)
- **Workspace scope** = Project knowledge (architecture, conventions, known issues) -- equivalent to CLAUDE.md/project rules files

This dual-scope model is correct and should remain. The extension point is the backend that stores each scope.

### Memory Consistency for Multi-Agent

When AGH eventually supports multiple concurrent agents (Phase 3: Agent Network), it will face the distributed memory consistency problem. The analysis of multi-agent memory consistency (MESI-style coherence, CAP trade-offs) suggests AGH should:

1. Default to **read-your-writes** consistency for session-scoped memory (an agent always sees its own writes)
2. Use **eventual consistency** for the shared global memory (agents may read slightly stale global facts)
3. Define **ownership** per memory entry (the agent that wrote it is the authority)
4. Implement **invalidation signals** when shared memory is updated (pub/sub pattern)

These do not need to be built now but the memory interface should be designed to accommodate them.

---

## Agent Network Considerations

AGH's Phase 3 (Agent Network Protocol) will require decisions about discovery, identity, delegation, and trust. The analysis reveals what should be prepared now vs. deferred.

### Prepare Now (Interface Design)

1. **Agent identity:** AGH-managed agents should have stable identifiers from day one. A session ID is not an agent identity. AGH should assign each configured agent a persistent ID (derivable from its config) that can later become a DID or Agent Card identifier.

2. **Capability description:** Each agent's configuration already describes what it can do (tools, skills, model). AGH should expose this as a structured capability manifest that can later become an A2A Agent Card.

3. **Delegation primitive:** The concept of one agent delegating a task to another should be modeled in the session system now. A session that is "delegated" has a parent session ID and inherits context from it. This prepares for multi-agent orchestration without requiring the full A2A stack.

4. **Context transfer format:** Define a serialization format for session state that can be sent to another agent. This is needed for both local multi-agent (subagent spawning within AGH) and future remote delegation.

### Defer to Phase 3

1. **A2A protocol implementation:** Full A2A client/server is premature for Phase 1. Define the `AgentPeer` interface now; implement it later.

2. **Discovery and registry:** Agent discovery (well-known URIs, registry federation, semantic search over capabilities) is a Phase 3 concern. AGH should publish Agent Cards for its agents but does not need to consume external registries yet.

3. **Payment and settlement:** Agent payment protocols (x402, stablecoin rails) are Phase 3+. No interface needed now.

4. **Cryptographic identity:** DIDs, Verifiable Credentials, and ACNBP-style capability binding are Phase 3 concerns. AGH should use simple string identifiers now, with a migration path to DIDs later.

5. **Multi-agent memory consistency:** Full MESI-style coherence, CRDT-based state synchronization, and directory-based cache protocols are Phase 3. The memory interface designed now should not preclude them but does not need to implement them.

### The Key Insight: Composition Over Monolith

The strongest pattern across all three knowledge domains is **composability through small, well-defined interfaces**. MCP succeeded because it reduced M x N integrations to M + N. A2A succeeded because it separated data model, operations, and transport into three independent layers. Memory systems succeed when they define store/recall/forget/consolidate as a pluggable interface.

AGH's architecture already follows this principle (small interfaces, dependency injection, daemon as sole composition root). The extensibility analysis confirms this is the right approach. The work ahead is defining the specific interfaces at each boundary -- especially `MemoryBackend`, `AgentPeer`, and the MCP host integration -- so that extensions can be developed and composed independently.

---

## Summary Decision Matrix

| Concept | Classification | Rationale |
|---------|---------------|-----------|
| Agentic loop (turn cycle) | **CORE** | Universal execution primitive |
| Session state machine | **CORE** | Every agent needs lifecycle management |
| Tool dispatch + schema registry | **CORE** | Universal action primitive |
| Permission model | **CORE** | Security is non-negotiable |
| Event recording + observability | **CORE** | Debugging non-deterministic behavior requires traces |
| Context assembly pipeline | **CORE** | Highest-leverage quality factor |
| Token budget management | **CORE** | Cost control and context rot prevention |
| Basic context compaction | **CORE** | Required for any session > 10 turns |
| Memory consolidation triggers | **CORE** | Cross-cutting concern for all memory backends |
| Agent drivers (Claude, Codex, etc.) | **EXTENSION** | Unbounded set, each independent |
| Memory backends (SQLite, vector, KG) | **EXTENSION** | Diverse trade-offs per deployment |
| Tool implementations | **EXTENSION** | Unbounded catalog |
| Orchestration strategies | **EXTENSION** | Policy varies by task type |
| Skill packages | **EXTENSION** | Domain-specific content |
| Notification channels | **EXTENSION** | Output format varies by consumer |
| MCP client (host) | **PROTOCOL (core)** | Industry-standard tool integration |
| A2A agent-to-agent | **PROTOCOL (interface in core, impl as extension)** | Still maturing; interface is stable |
| Agent Card / capability manifest | **PROTOCOL (core)** | Self-description is always needed |
| Context transfer / handoff | **PROTOCOL (core)** | Required for any multi-agent scenario |
| OTel observability format | **PROTOCOL (core)** | Cross-vendor tracing standard |
| Agent discovery / registry | **PROTOCOL (deferred)** | Phase 3 concern |
| DID / Verifiable Credentials | **PROTOCOL (deferred)** | Phase 3 concern |
| Payment protocols | **PROTOCOL (deferred)** | Phase 3+ concern |
| Multi-agent memory consistency | **PROTOCOL (deferred)** | Phase 3 concern; design interface now |
