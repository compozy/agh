# OpenFang Analysis for AGH Extensibility

## Overview

OpenFang is a Rust-based Agent Operating System comprising 14 crates (~137K LoC) that runs as a persistent daemon managing AI agent sessions. It shares AGH's fundamental design philosophy -- single binary, SQLite persistence, daemon model, local-first -- but takes a maximalist approach: 53 builtin tools, 40 channel adapters, 7 bundled Hands, 60+ skills, 130+ model catalog entries, and 25 MCP templates all compiled into one ~32 MB binary.

Where AGH follows "robust minimal core + extensible plugins", OpenFang follows "bundle everything into the binary". This is the central tension in the analysis: OpenFang validates many features AGH should eventually support, but its monolithic compilation strategy is the opposite of AGH's extension-first philosophy. The features are proven; the packaging strategy should be inverted.

### Key Similarities to AGH

- Single-binary daemon with SQLite (WAL mode, `synchronous = FULL`)
- Kernel-as-composition-root pattern (OpenFang's `OpenFangKernel` ~ AGH's `daemon/`)
- Agent lifecycle state machine with session persistence
- TOML configuration with env var interpolation
- JSON-RPC over stdio for agent communication (MCP/ACP)
- Strict dependency direction enforced by module boundaries
- CLI that doubles as HTTP client when daemon is running

### Key Differences from AGH

- Rust vs Go (crate boundaries vs package boundaries)
- Everything compiled in vs extension-first architecture
- 140+ HTTP endpoints vs AGH's focused API surface
- Custom OFP wire protocol vs AGH's ACP-based approach
- In-process LLM drivers vs subprocess-spawned ACP agents
- Built-in web dashboard (Alpine.js) vs AGH's separate React SPA

---

## Key Features Analysis

### Agent Runtime & Execution

| Feature                                                  | OpenFang Implementation                                                                           | Classification | Rationale                                                                                                                                                                                                         |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Agent loop (recv/recall/call/execute)**                | `run_agent_loop()` in openfang-runtime, 50-iteration cap, bounded by loop guard                   | **CORE**       | This is the fundamental execution model. AGH already has this via ACP subprocess agents, but the loop-bounding, stop-reason taxonomy, and structured result type are worth hardening.                             |
| **Loop guard (cycle detection)**                         | SHA256 fingerprinting of recent tool calls, detects ping-pong patterns, forces conclusion         | **CORE**       | Critical safety mechanism. Any agent that can loop must have cycle detection. AGH should implement this in the session/observe layer as a cross-cutting concern, not per-agent.                                   |
| **Context budget allocator**                             | Token allocation across system/tools/history/response regions, 70% compaction, 90% emergency trim | **CORE**       | Essential for long-running sessions. AGH's `transcript` package should own this, with configurable thresholds per agent.                                                                                          |
| **Session repair (7-phase validation)**                  | Validates message continuity, tool call completeness, role alternation, deduplication, timestamps | **CORE**       | Critical for crash recovery of long-running agents. AGH's session store should validate on load. This prevents corrupt state from cascading.                                                                      |
| **Three LLM drivers (Anthropic, Gemini, OpenAI-compat)** | Native HTTP clients with provider-specific adaptation                                             | **EXTENSION**  | AGH delegates LLM interaction to ACP agents (Claude Code, Codex, Gemini CLI). AGH should NOT embed LLM drivers -- the ACP model is superior because it delegates provider-specific logic to purpose-built agents. |
| **Provider routing with fallback chain**                 | ModelRouter with complexity scoring, auth cooldown, fallback traversal                            | **EXTENSION**  | If AGH ever routes between multiple ACP agents based on task complexity, this pattern is useful. But it belongs as an extension, not core -- AGH's philosophy is that the agent handles its own model selection.  |
| **Model catalog (130+ models with pricing)**             | Static catalog compiled into binary, cost per million tokens                                      | **EXTENSION**  | Useful for metering, but should be a loadable resource file, not compiled in. AGH's config system can reference an external catalog.                                                                              |

### Scheduling & Automation

| Feature                                       | OpenFang Implementation                                                            | Classification | Rationale                                                                                                                                                                         |
| --------------------------------------------- | ---------------------------------------------------------------------------------- | -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Cron scheduler**                            | POSIX 5-field cron expressions, `BackgroundExecutor` with per-schedule Tokio tasks | **CORE**       | A daemon that runs 24/7 needs scheduled execution. AGH should support cron-triggered sessions in the daemon package. Simple: parse cron, sleep until next fire, dispatch session. |
| **Event-driven triggers**                     | `TriggerEngine` subscribes to EventBus, matches event kind + regex + fire limits   | **CORE**       | Reactive execution is the complement to cron. AGH's `observe` package already has event recording; adding pattern-matching trigger dispatch is a natural extension of that.       |
| **Fire limits (rate limiting triggers)**      | Rolling hourly counter prevents thundering-herd from high-frequency events         | **CORE**       | Without fire limits, a misconfigured webhook can spawn hundreds of sessions per second. This is a safety mechanism that belongs in core.                                          |
| **Missed fire policy (skip, don't backfill)** | Deliberate: no catch-up on missed cron fires after daemon restart                  | **CORE**       | Good design decision. Backfilling is complex and budget-dangerous. AGH should adopt the same policy.                                                                              |

### Workflow Engine

| Feature                                        | OpenFang Implementation                                                              | Classification | Rationale                                                                                                                                                                                                                   |
| ---------------------------------------------- | ------------------------------------------------------------------------------------ | -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Multi-step workflow pipelines**              | `WorkflowEngine` with sequential, parallel (fan-out), condition, loop, collect modes | **EXTENSION**  | Powerful but complex. This should be an extension that composes on top of AGH's session primitives. Core should provide the building blocks (session dispatch, result collection); the workflow engine wires them together. |
| **Variable interpolation between steps**       | `{{step_output}}`, `{{global_var}}`, `{{input}}` expansion in step prompts           | **EXTENSION**  | Implementation detail of the workflow engine extension.                                                                                                                                                                     |
| **Error handlers per step (retry/skip/abort)** | Exponential backoff retries, skip-and-continue, abort-workflow                       | **EXTENSION**  | Belongs with the workflow engine extension.                                                                                                                                                                                 |
| **Visual workflow builder**                    | Alpine.js canvas with drag-and-drop step nodes                                       | **EXTENSION**  | Frontend concern, definitely an extension.                                                                                                                                                                                  |

### Hands (Autonomous Agent Packages)

| Feature                                                       | OpenFang Implementation                                                                        | Classification | Rationale                                                                                                                                                                                                                 |
| ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Hand concept (packaged autonomous agents)**                 | HAND.toml manifest + SKILL.md + system prompt + cron schedule + dashboard metrics + guardrails | **EXTENSION**  | Brilliant packaging concept. AGH should support a similar "agent package" format as an extension -- a directory with manifest, instructions, schedule, and tool allowlist. But it should NOT be compiled into the binary. |
| **Hand lifecycle state machine**                              | Discovered -> Dormant -> Active -> Running -> Paused -> Completed -> Error                     | **CORE**       | The lifecycle state machine itself is a core pattern that AGH's session manager already partially implements. The state transitions and persistence-across-restart behavior should be part of AGH's session package.      |
| **Dependency verification (binary, env var, API key checks)** | `check_requirements()` validates system state before activation                                | **EXTENSION**  | Useful for agent packages but not core. Extensions that need external binaries should declare and check their own deps.                                                                                                   |
| **Hand persistence across daemon restarts**                   | JSON state files at `~/.openfang/hands/<id>.json`, recovered at boot                           | **CORE**       | AGH should persist active session configurations so they survive daemon restarts. This is part of the daemon lifecycle, not an extension.                                                                                 |

### Memory & Knowledge

| Feature                                            | OpenFang Implementation                                                          | Classification                                | Rationale                                                                                                                                                                                                                                       |
| -------------------------------------------------- | -------------------------------------------------------------------------------- | --------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Tri-part memory (session + semantic + KG)**      | Structured KV, semantic text search, entity-relation-fact triples, all in SQLite | **CORE (session) + EXTENSION (semantic, KG)** | Session storage is core (AGH has this). Semantic recall and knowledge graph are extensions. AGH already has `memory/` with dual-scope memory and dream consolidation -- this aligns well. The KG is a natural extension of AGH's memory system. |
| **Knowledge graph (entity-relation-fact triples)** | Three SQLite tables, confidence scoring, BFS traversal, per-agent scoping        | **EXTENSION**                                 | Structured knowledge is powerful for long-running agents but adds schema complexity. Should be an opt-in extension that agents can activate.                                                                                                    |
| **Session compaction (70% threshold)**             | LLM-based summarization of old user/assistant pairs, chars/4 heuristic           | **CORE**                                      | Long-running AGH sessions need compaction. The threshold-based approach with graceful degradation belongs in `transcript/`.                                                                                                                     |
| **Memory consolidation on agent clone**            | Dedup entities, merge KV stores, merge KG, report conflicts                      | **EXTENSION**                                 | Useful but not core. Agent cloning is an advanced feature.                                                                                                                                                                                      |

### Channel Adapters

| Feature                                   | OpenFang Implementation                                                          | Classification | Rationale                                                                                                                                                                     |
| ----------------------------------------- | -------------------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Channel adapter trait**                 | `ChannelAdapter` (inbound stream) + `MessageAdapter` (outbound send)             | **EXTENSION**  | The trait design is excellent -- clean separation of inbound/outbound with platform-agnostic message envelope. AGH should define a similar interface in its extension system. |
| **40 messaging platform adapters**        | Telegram, Discord, Slack, WhatsApp, Signal, Matrix, Email, etc.                  | **EXTENSION**  | Obviously extensions. Each adapter is a plugin that implements the channel interface.                                                                                         |
| **Message routing with 5-level priority** | Bindings -> direct routes -> user defaults -> channel defaults -> system default | **EXTENSION**  | Routing logic belongs with the channel system extension, not core.                                                                                                            |
| **Per-channel policies**                  | DM policy, group policy, output format, rate limits, user allow/block lists      | **EXTENSION**  | Configuration for channel extensions.                                                                                                                                         |
| **Hot-reloadable channel config**         | Adapter restart without daemon restart on config change                          | **CORE**       | Hot reload of extension configuration is a core daemon capability. AGH should support this generically for all extensions.                                                    |

### Peer Networking (OFP)

| Feature                                         | OpenFang Implementation                                                       | Classification | Rationale                                                                                                                                                                                                                |
| ----------------------------------------------- | ----------------------------------------------------------------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **OFP wire protocol**                           | Custom TCP binary protocol with length-prefix + JSON, HMAC-SHA256 mutual auth | **EXTENSION**  | Agent-to-agent networking is a Phase 3 feature for AGH. When it arrives, it should be an extension, not a custom wire protocol. AGH should prefer standard protocols (HTTP, gRPC, or A2A spec) over inventing a new one. |
| **Peer discovery (gossip)**                     | PeerDiscovery payload exchange for transitive endpoint discovery              | **EXTENSION**  | Network topology management is clearly an extension.                                                                                                                                                                     |
| **Heartbeat and health monitoring**             | 30s heartbeat with Healthy/Degraded/Unhealthy/Disconnected classification     | **EXTENSION**  | Peer health is part of the networking extension.                                                                                                                                                                         |
| **Inter-agent tools (agent_send, agent_spawn)** | 5 tools for cross-agent and cross-node communication                          | **EXTENSION**  | Agent delegation and orchestration tools are extensions that compose on top of the session system.                                                                                                                       |
| **Recursion guard (MAX_DEPTH=5)**               | `task_local!` depth counter prevents infinite agent delegation chains         | **CORE**       | If AGH supports agent-to-agent delegation, the recursion guard is a safety mechanism that belongs in core. Unbounded recursion is a cost and stability risk.                                                             |

### Security

| Feature                                        | OpenFang Implementation                                                                          | Classification | Rationale                                                                                                                                                                                    |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------ | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **RBAC capability gates**                      | Per-agent tool allowlist, deny-by-default, child inherits subset of parent                       | **CORE**       | Essential for multi-agent safety. AGH should enforce tool/capability scoping per session.                                                                                                    |
| **Approval manager (human-in-the-loop)**       | Tool risk levels (Low/Medium/High/Critical), oneshot channels for blocking approval, 60s timeout | **EXTENSION**  | Approval gates are important for autonomous agents but should be an extension. Core provides the hook point; the approval logic is pluggable.                                                |
| **Merkle hash-chain audit log**                | SHA256 chaining of every significant action, tamper detection, append-only                       | **EXTENSION**  | Powerful for compliance but overkill for most AGH deployments. Should be an optional extension that wraps AGH's observe/event system.                                                        |
| **WASM sandbox (fuel + epoch + watchdog)**     | Triple-metered Wasmtime sandbox for untrusted skills                                             | **EXTENSION**  | AGH doesn't execute untrusted code in-process (ACP agents are subprocesses). If AGH adds a WASM skill runtime, this belongs there.                                                           |
| **SSRF protection**                            | Block private IPs, metadata endpoints, DNS rebinding on web_fetch                                | **CORE**       | If AGH ever exposes web-fetch capabilities, SSRF protection is non-negotiable. But since AGH delegates to ACP agents that have their own sandboxing, this may be the agent's responsibility. |
| **Subprocess sandbox (env_clear + allowlist)** | Clear environment, selective passthrough for child processes                                     | **CORE**       | AGH already spawns ACP agents as subprocesses. Environment isolation is a core safety property.                                                                                              |
| **Secret zeroization**                         | `Zeroizing<String>` wrapper that scrubs memory on drop                                           | **CORE**       | All credential handling in AGH should use Go's equivalent pattern (explicit zeroing of byte slices).                                                                                         |
| **Prompt injection scanner**                   | Scan user messages for instruction overrides, delimiter injection                                | **EXTENSION**  | Defense against prompt injection is valuable but should be a pluggable middleware, not hardcoded.                                                                                            |
| **Taint tracking**                             | Newtype wrappers that label secret data through the call chain                                   | **EXTENSION**  | Sophisticated but heavyweight. Go doesn't have Rust's type-level guarantees for this. Should be an extension if implemented.                                                                 |

### Cost & Metering

| Feature                                           | OpenFang Implementation                                                | Classification | Rationale                                                                                                               |
| ------------------------------------------------- | ---------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------------------------------------------------------------- |
| **Per-agent, per-provider, global cost tracking** | DashMap for per-agent, per-provider, AtomicU64 for global              | **CORE**       | Cost tracking is essential for any agent system. AGH's `observe` package should track token usage and cost per session. |
| **Budget enforcement (daily + monthly limits)**   | Pre-flight check on every LLM call, halt agent or all agents on breach | **CORE**       | Cost runaway is the top operational risk for autonomous agents. Budget gates must be in core.                           |
| **Cost-aware rate limiting**                      | GCRA token bucket where expensive models draw more tokens              | **EXTENSION**  | Sophisticated but an optimization. Basic rate limiting is core; cost-weighted rate limiting is an extension.            |

### MCP Integration

| Feature                                      | OpenFang Implementation                                                        | Classification | Rationale                                                                                                                                          |
| -------------------------------------------- | ------------------------------------------------------------------------------ | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| **MCP client (tool discovery + dispatch)**   | Connect to external MCP servers, discover tools, merge into agent tool catalog | **CORE**       | AGH already has ACP for agent communication. MCP tool discovery and dispatch should be a core capability since it's the standard for tool interop. |
| **MCP server mode (expose agents as tools)** | OpenFang agents consumable by external MCP clients                             | **EXTENSION**  | Exposing AGH agents as MCP tools is valuable for interop but not essential for core operation.                                                     |
| **25 bundled MCP templates**                 | Pre-configured MCP server configs for GitHub, Slack, Notion, etc.              | **EXTENSION**  | Templates are definitionally extensions. AGH should ship none bundled and let users install from a registry.                                       |
| **Tool namespacing (mcp\_ prefix)**          | Prevent collisions between builtins, skills, and MCP tools                     | **CORE**       | Any system that merges tools from multiple sources needs namespacing. This is a core protocol concern.                                             |

### Configuration & CLI

| Feature                                      | OpenFang Implementation                                                     | Classification | Rationale                                                                                                             |
| -------------------------------------------- | --------------------------------------------------------------------------- | -------------- | --------------------------------------------------------------------------------------------------------------------- |
| **Dual execution mode (daemon + ephemeral)** | Same binary works as daemon or single-shot CLI                              | **CORE**       | AGH should support this. `agh chat "question"` should work without a running daemon by booting an ephemeral session.  |
| **Config hot reload**                        | Whitelist of reloadable fields, POST /api/config/reload                     | **CORE**       | Essential for operational flexibility. AGH's daemon should support reloading config subsections without restart.      |
| **Config validation endpoint**               | POST /api/config/validate for dry-run validation                            | **EXTENSION**  | Nice-to-have but not essential for core.                                                                              |
| **Credential vault (AES-256-GCM)**           | Encrypted secret storage with Argon2 key derivation, OS keyring integration | **EXTENSION**  | AGH should integrate with OS keyrings or external secret managers, but a custom vault implementation is an extension. |
| **OAuth2 PKCE flow**                         | Built-in OAuth for Google, GitHub, Slack integrations                       | **EXTENSION**  | Authentication flows for third-party services are clearly extensions.                                                 |

---

## Architectural Patterns Worth Adopting

### 1. Kernel-as-Composition-Root with Explicit Subsystem Fields

OpenFang's `OpenFangKernel` holds ~35 fields representing every subsystem. AGH's `daemon/` package serves the same role. The key insight: make every subsystem visible as a named field, not hidden behind a service locator or DI container.

**AGH implication**: The daemon struct should explicitly list session manager, store, observer, memory, skills, config, etc. as typed fields. New extensions register through the daemon at boot, not through a generic registry.

### 2. Strict Dependency Direction (Foundation -> Subsystems -> Kernel -> API -> CLI)

OpenFang's 14-crate workspace enforces no circular dependencies at compile time. The DAG flows: types (leaf) -> subsystems -> runtime -> kernel -> API -> CLI.

**AGH implication**: AGH already follows this (`daemon/` imports all; nothing imports `daemon/`). Maintain this rigorously as extensions are added. Extensions should depend on core interfaces, never on the daemon or API packages.

### 3. EventBus with Typed Events for Cross-Subsystem Reactions

OpenFang's `EventBus` with correlation IDs connects the metering engine, audit log, trigger engine, and workflow engine without coupling them directly.

**AGH implication**: AGH's `observe` package records events. Adding a pub-sub dispatch mechanism (typed observer/notifier pattern, not a generic bus) would enable the trigger engine and cost tracking to react to session events without importing each other.

### 4. KernelHandle Trait for Testability

OpenFang's `KernelHandle` trait lets the runtime call kernel methods without importing the kernel directly. This enables testing the runtime with a mock kernel.

**AGH implication**: AGH's `session/` package defines `AgentDriver` (implemented by `acp/`). Extend this pattern: define a `KernelHandle` or `DaemonHandle` interface that the session package and extensions use to call back into the daemon. This breaks the dependency arrow and enables testing.

### 5. Dual Execution Mode (Daemon + Ephemeral)

The same binary can boot a full daemon or run a single-shot operation. This is critical for scripting, testing, and CLI ergonomics.

**AGH implication**: AGH should support `agh chat "question"` without a running daemon. The daemon package should expose an ephemeral boot path that initializes just enough state for one session.

### 6. Agent Package Format (HAND.toml Analog)

OpenFang's Hands package system prompt + skills + manifest + schedule + guardrails into a single activatable unit.

**AGH implication**: AGH should define an "agent package" format (TOML manifest, instruction file, tool allowlist, schedule, resource quotas) that extensions can install, activate, and manage. This is the primary extensibility surface for end users.

### 7. Stop Reason Taxonomy

OpenFang's `StopReason` enum (Completed, MaxIterations, LoopDetected, Timeout, QuotaExceeded, BudgetExceeded, Error) gives precise observability into why an agent loop terminated.

**AGH implication**: AGH's session state machine should capture terminal states with the same granularity. This feeds directly into observability, debugging, and billing.

---

## Extension System Insights

### What Should Be the Extension Interface?

OpenFang has no runtime extension loading -- everything is compiled in. This is the opposite of what AGH wants. However, the _boundaries_ between OpenFang's subsystems reveal the natural extension points:

1. **Tool providers** -- anything that adds tools to the agent's catalog (MCP servers, skill runtimes, builtin tools). Interface: tool definition + execute function.

2. **Channel adapters** -- anything that bridges external messaging to agent sessions. Interface: inbound message stream + outbound send.

3. **Memory backends** -- anything that extends the memory substrate (semantic search, knowledge graph, vector DB). Interface: store + recall.

4. **Scheduling triggers** -- anything that dispatches sessions on events or time. Interface: event pattern + session dispatch.

5. **Workflow orchestrators** -- anything that composes multiple sessions into pipelines. Interface: step definition + execution engine.

6. **Security layers** -- anything that adds safety checks to the execution pipeline. Interface: pre-execution hook + post-execution hook.

### Workflow Engine as Extension Pattern

The workflow engine is the best example of a feature that should be an extension, not core. It composes the core session dispatch primitive into multi-step pipelines with fan-out, conditionals, and loops. The key design lesson:

- **Core provides**: session dispatch, result collection, event emission on completion
- **Extension provides**: step ordering, variable interpolation, parallel dispatch, error handling
- **Extension consumes**: only the core interfaces (session dispatch + event bus), never kernel internals

This pattern generalizes: any complex orchestration (workflow engine, Hand lifecycle, channel bridge) should compose on top of core primitives through defined interfaces.

### OFP Peer Network: Cautionary Tale

OpenFang invented a custom TCP wire protocol for agent-to-agent communication. While technically sound (HMAC auth, nonce replay protection), it creates a compatibility island -- only OpenFang instances can speak OFP.

**AGH recommendation**: Do NOT invent a custom protocol. Use the A2A protocol specification (Google/Linux Foundation) or plain HTTP. Agent networking should be an extension that speaks standard protocols, ensuring interoperability with non-AGH systems.

### Hands/Tools/Skills Layering

OpenFang's four-layer tool taxonomy is instructive:

| Layer     | Scope               | Sandbox         | Example                     |
| --------- | ------------------- | --------------- | --------------------------- |
| Builtins  | Core functionality  | In-process      | file_read, memory_store     |
| MCP tools | External interop    | Subprocess/HTTP | GitHub, Slack, Notion       |
| Skills    | Domain expertise    | WASM/subprocess | Data analysis, web scraping |
| Hands     | Autonomous packages | Agent-level     | Researcher, Lead, Collector |

**AGH equivalent layering**:

| Layer           | AGH Scope          | Mechanism        | Example                        |
| --------------- | ------------------ | ---------------- | ------------------------------ |
| ACP agent tools | Core functionality | ACP protocol     | Claude Code, Codex, Gemini CLI |
| MCP tools       | External interop   | MCP protocol     | GitHub, Linear, Notion servers |
| Skills          | Domain expertise   | Bundled SKILL.md | AGH's existing skills package  |
| Agent packages  | Autonomous units   | Package manifest | Researcher, analyst, monitor   |

The key difference: AGH pushes tool execution to ACP agents instead of executing in-process. This is architecturally superior for isolation but means AGH's extension system focuses on configuration and composition rather than runtime execution.

### What AGH Can Skip

Several OpenFang features are consequences of its monolithic design and are unnecessary for AGH:

1. **In-process LLM drivers** -- AGH delegates to ACP agents. No need to implement Anthropic/OpenAI/Gemini HTTP clients.
2. **WASM sandbox** -- AGH doesn't execute untrusted code in-process. ACP agents run as sandboxed subprocesses.
3. **40 channel adapters compiled into the binary** -- These should be installable extensions, not compiled in.
4. **Custom OFP protocol** -- Use standard A2A/HTTP.
5. **Built-in web dashboard** -- AGH already has a separate React SPA. Better separation of concerns.
6. **130+ model catalog** -- AGH's ACP agents handle their own model selection. A config-loadable pricing catalog is sufficient for metering.

---

## Summary: Priority Features for AGH

### Immediate (Core)

1. **Budget enforcement** -- per-session and global cost limits with pre-flight checks
2. **Session stop reason taxonomy** -- precise terminal state classification
3. **Cron scheduler** -- POSIX cron for scheduled session dispatch
4. **Event-driven triggers** -- pattern-matching on session events with fire limits
5. **Loop/recursion guard** -- cycle detection for agent delegation chains
6. **Session compaction** -- threshold-based context trimming for long sessions
7. **Session repair on load** -- validate session state integrity after crash
8. **Dual execution mode** -- ephemeral single-shot alongside persistent daemon
9. **Config hot reload** -- reload extension configs without daemon restart
10. **Subprocess environment isolation** -- env_clear + allowlist for ACP agent spawning

### Near-term (Extension interfaces)

1. **Tool provider interface** -- for MCP servers and custom tool sources
2. **Channel adapter interface** -- for messaging platform bridges
3. **Agent package format** -- manifest + instructions + schedule + guardrails
4. **Workflow engine** -- multi-step session orchestration with fan-out
5. **Memory extension interface** -- for knowledge graph, semantic search backends

### Later (Extensions)

1. **Knowledge graph engine** -- entity-relation-fact triples with confidence scoring
2. **Approval manager** -- human-in-the-loop gates for high-risk operations
3. **Audit log (Merkle chain)** -- tamper-evident action logging
4. **Agent-to-agent networking** -- standard A2A protocol, not custom wire protocol
5. **Credential vault** -- encrypted secret storage with OS keyring integration
