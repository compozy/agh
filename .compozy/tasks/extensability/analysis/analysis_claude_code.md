# Claude Code Harness Analysis for AGH Extensibility

## Overview

Claude Code is a TypeScript agentic CLI that bridges natural-language intent to code and shell operations. Its architecture is a seven-layer stack: Entry Points, Bootstrap/Configuration, Setup, UI Layer, QueryEngine (async-generator core loop), Tool System (50+ tools), and Services/State. The harness manages a disciplined cycle of "send message -> stream response -> execute tools -> loop" with everything else -- the TUI, the service container, the permission engine -- existing to feed that loop.

This analysis identifies the key features, architectural patterns, and capabilities from Claude Code and classifies each as either **CORE** (essential for any agent OS minimal core) or **EXTENSION** (should be a plugin/extension on top of the core) for AGH's architecture.

AGH's philosophy is a robust minimal core with a highly extensible plugin system. The classification below applies that lens: features that every ACP-compatible agent session needs regardless of agent type belong in core; features that are domain-specific, agent-specific, or can be composed from core primitives belong as extensions.

## Key Features Analysis

### Foundational Architecture

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Core Query/Agent Loop** (async turn cycle: normalize -> call model -> execute tools -> loop) | **CORE** | This is the beating heart of any agent OS. AGH's session package already owns session lifecycle; the turn-based execution loop with tool dispatch is the irreducible minimum for running any ACP agent. Every agent type needs this cycle. |
| **Tool Interface Contract** (uniform schema: identity, input schema, execution, permissions, concurrency metadata) | **CORE** | A uniform tool abstraction is what makes the system extensible without changing the core loop. AGH needs a `ToolDriver` interface (like `AgentDriver`) that all tools implement. The contract must include: name, input validation, execution, read-only/concurrency-safe flags, and permission check. |
| **Tool Registry with Dynamic Loading** | **CORE** | The registry that maps tool names to implementations and supports runtime registration (for MCP tools, plugin tools) is core infrastructure. Without it, extensibility requires recompilation. |
| **Tool Execution Pipeline** (validate -> permission check -> pre-hooks -> execute -> post-hooks -> result truncation) | **CORE** | The ordered pipeline through which every tool call passes is the single enforcement point for safety, validation, and extensibility. This is not optional -- it is how the core guarantees invariants for any extension. |
| **Tool Partitioning** (concurrent reads, serial writes) | **CORE** | Smart concurrency based on `isReadOnly()` and `isConcurrencySafe()` flags is a significant performance optimization that belongs in the core orchestrator. It halves wall-clock latency for read-heavy tool batches and prevents write races. |
| **Message Normalization** (role alternation, tool result hoisting, thinking block rules) | **CORE** | Every ACP provider will have message format requirements. The normalization layer that transforms internal state to provider-compatible format is essential infrastructure that sits between the session store and the model call. |
| **Streaming Response Handling** (async generator yielding events to observers) | **CORE** | AGH already has SSE for the web UI and the notifier pattern for fan-out. The streaming pipeline from model to observers is core -- it is how the daemon surfaces real-time events to all consumers (web UI, CLI, hooks). |
| **Result Truncation / Large Output Handling** (persist to disk, send preview + path) | **CORE** | Preventing a single tool result from consuming the entire context window is a safety invariant. The core should enforce per-tool `maxResultSize` and handle overflow to disk automatically. |

### Permission and Security Model

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Multi-Level Permission Rule Cascade** (policy > user > project > local > session) | **CORE** | The layered permission system where higher-level sources cannot be overridden by lower ones is essential for any production agent OS, especially one targeting enterprise deployment. AGH's config package already does TOML merge -- extending it to permission rules with cascade semantics is a natural fit. |
| **Permission Decision Waterfall** (deny rules -> tool-specific check -> mode -> allow rules -> classifier -> user prompt) | **CORE** | The ordered evaluation that short-circuits on definitive answers is the enforcement mechanism. Without it, every tool either runs unchecked or requires manual approval. The waterfall structure belongs in core; specific classifiers can be extensions. |
| **Permission Modes** (default, plan, auto, acceptEdits, bypass) | **CORE** | Modes define baseline strictness and are critical for both interactive and automated use. `bypass` mode enables CI/automation; `plan` mode enables safe exploration; `auto` mode reduces prompt fatigue. These are fundamental operational modes, not domain-specific extensions. |
| **Plan Mode as Hard Constraint** (restricts tools to read-only, requires approval to escalate) | **CORE** | Plan mode enforced at the tool layer (not as a suggestion to the model) is a key safety pattern. The core must support tool-scope restriction based on session mode. |
| **Multi-Resolver Race Pattern** (parallel permission resolvers with first-safe-answer-wins) | **EXTENSION** | The sophisticated `createResolveOnce` pattern with parallel resolvers (user click, hook classifier, bridge UI) is an optimization. The core needs a permission resolution interface; the parallel race with multiple resolver types is an advanced capability. |
| **LLM-Based Safety Classifier** (transcript classifier for auto-approve) | **EXTENSION** | Using a separate LLM to classify tool safety is powerful but expensive and model-dependent. The core should define a `PermissionClassifier` interface; the LLM-based implementation is an extension. |
| **Iron Gate** (hardcoded categorical restrictions that no classifier can bypass) | **CORE** | Certain actions must be categorically refused regardless of any classifier, user setting, or mode. A small set of hardcoded deny rules that cannot be overridden is a safety floor that belongs in core. |
| **Permission Explanation** (LLM-generated risk assessment for user prompts) | **EXTENSION** | Natural-language command risk explanation is a UX enhancement that uses side-queries. Not essential for the minimal core. |

### Hook System (Lifecycle Extensibility)

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Lifecycle Event Bus** (25+ events: PreToolUse, PostToolUse, SessionStart, SessionEnd, UserPromptSubmit, etc.) | **CORE** | The hook event taxonomy is the primary extensibility surface. AGH's notifier pattern is already a typed interface for fan-out -- extending it with a formal lifecycle event bus (pre/post tool, session lifecycle, prompt submission) is core infrastructure that enables all extensions. |
| **Hook Output Protocol** (structured JSON: continue/block, updatedInput, additionalContext, transformedResult) | **CORE** | The protocol by which hooks communicate decisions back to the core loop is the contract. Without a structured protocol, hooks are fire-and-forget side effects. With it, hooks can block, modify, and transform -- making them load-bearing extensibility points. |
| **Hook Types** (command, prompt, agent, http, function) | **EXTENSION** | The five execution engines for hooks are implementations of the hook contract. The core needs to define the `HookExecutor` interface and ship a basic `command` executor. `prompt`, `agent`, `http`, and `function` types are extensions that plug into the same interface. |
| **Hook Matcher Syntax** (regex/glob filters for event-specific keys like tool names) | **CORE** | Matchers determine which hooks fire for which events. A simple but expressive matching syntax (exact name, pipe-separated, glob) is core because it determines hook specificity. |
| **PreToolUse Blocking and Modification** (hooks can block execution or rewrite tool inputs) | **CORE** | The ability for pre-execution hooks to block or modify is essential for verification gates, policy enforcement, and input sanitization. This is not a nice-to-have -- it is how organizations enforce coding standards, security policies, and workflow rules. |
| **PostToolUse Result Transformation** (hooks can redact or augment tool results) | **CORE** | Result transformation enables secret redaction, output enrichment, and audit logging. This is a security-critical capability that belongs in core. |
| **Enterprise Hook Enforcement** (MDM-managed hooks that users cannot remove) | **EXTENSION** | MDM enforcement is an enterprise deployment concern. The core should support hook source precedence; the MDM-specific enforcement is an enterprise extension. |

### Memory and Session Persistence

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Event-Sourced Session Store** (append-only event log per session) | **CORE** | AGH already has this via `sessiondb` (per-session SQLite event store). This is the foundation of session persistence and replay. |
| **Session Resume / Replay** (restore conversation state from persisted events) | **CORE** | The ability to resume a session from persisted state is fundamental for the daemon model. AGH's transcript package already handles replay message assembly -- this belongs in core. |
| **Tiered Memory Architecture** (conversation history, session memory, instruction files, cross-session auto-memory, team memory) | **CORE (framework) / EXTENSION (implementations)** | The framework for tiered memory (AGH's `memory` package with dual-scope global + workspace) is core. The specific implementations -- auto-extraction subagent, Sonnet-based semantic recall, team memory sync -- are extensions that plug into the memory framework. |
| **Persistent Instruction File** (CLAUDE.md / project-level config loaded every session) | **CORE** | AGH's config package handles TOML loading. A mechanism for per-workspace instruction files that agents receive in their system prompt is core infrastructure. |
| **Background Memory Extraction** (forked subagent extracting facts after each turn) | **EXTENSION** | The extraction subagent is a specific implementation of the memory write path. The core needs a `MemoryWriter` interface; the LLM-based extraction is an extension (and AGH already has `dream consolidation` as its analog). |
| **AutoDream / Memory Consolidation** (periodic background merge, dedup, prune) | **EXTENSION** | AGH already has this in `internal/memory/consolidation`. The consolidation runtime is an extension that uses the core memory and session interfaces. The core provides the scheduling, locking, and memory access primitives. |
| **Semantic Recall** (LLM side-query to select relevant memories per turn) | **EXTENSION** | Using a separate model as a relevance filter is a specific recall strategy. The core defines a `MemoryRecaller` interface; LLM-based semantic recall is one implementation. |
| **Session Memory Summary** (structured summary maintained during conversation for compaction) | **EXTENSION** | Session memory as a pre-built summary for fast compaction is a specific optimization strategy. The core provides the compaction trigger; the SM-Compact strategy is an extension. |

### Context Management

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Token Counting** (hybrid API + heuristic estimation) | **CORE** | Context budget awareness is essential for any long-running agent session. The core needs a token estimation facility -- even a rough heuristic -- to know when compaction or truncation is needed. |
| **Context Compaction Cascade** (5-layer: tool result budget -> snip -> microcompact -> context collapse -> autocompact) | **CORE (framework) / EXTENSION (strategies)** | The framework that runs compaction strategies in order of increasing cost/loss is core. The specific strategies (snip, microcompact, SM-compact, full conversation compaction) are extensions that register with the framework. The core provides: threshold detection, strategy ordering, circuit breaker, post-compact cleanup. |
| **Static/Dynamic System Prompt Split** (cached prefix + per-request dynamic tail) | **CORE** | Splitting the system prompt into a cacheable static portion and a per-request dynamic portion is a cost and latency optimization that benefits every session. This belongs in the core prompt-building pipeline. |
| **Tool Result Budget** (per-tool maxResultSize with overflow to disk) | **CORE** | Already classified above under Tool Execution Pipeline. |
| **Circuit Breaker** (halt compaction after N consecutive failures) | **CORE** | Preventing infinite retry on compaction failure is a safety mechanism. The core compaction framework should include a circuit breaker. |

### Plugin and Skills System

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Three-Layer Plugin Reconciliation** (intent in settings, materialization on disk, activation at runtime) | **CORE** | The separation of what-the-user-wants from what-is-installed from what-is-active makes the plugin system robust to partial failures. AGH's skills package should adopt this pattern -- it is the foundation of reliable extension management. |
| **Plugin Lifecycle Operations** (install, uninstall, enable, disable, update) | **CORE** | The CRUD operations for plugins, including the enable/disable vs install/uninstall distinction, are core plugin management. |
| **Marketplace Discovery** | **EXTENSION** | The marketplace UI, browsing, and discovery pipeline are value-added features on top of the core plugin lifecycle. The core needs a plugin registry and loader; marketplace is an extension. |
| **Skills as Markdown Procedures** (SKILL.md with frontmatter for activation, not code) | **CORE** | The concept of skills as prompt-and-procedure pairs (not compiled code) is a key design insight. Skills occupy zero token budget at rest (only metadata visible until activated). AGH already has a skills catalog -- the SKILL.md contract with `description`, `when_to_use`, and `allowed-tools` frontmatter is the right abstraction for the core. |
| **Progressive Disclosure** (skill content materializes into context only when activated) | **CORE** | This is not just a nice optimization -- it is what makes a large skill library practical. The core skill loader must support lazy materialization based on activation, not eager loading. |
| **Skill Improvement** (background process watches for user corrections and proposes skill updates) | **EXTENSION** | Automatic skill refinement based on session corrections is an advanced feature. |
| **Agent Definitions** (markdown files declaring subagent identity, tools, prompts) | **CORE** | Agent definitions are how AGH will support heterogeneous agent types. The markdown-with-frontmatter format for declaring agent capabilities, tool scopes, and system prompts belongs in core. |
| **Plugin-Provided Hooks** | **CORE** | Plugins must be able to register hooks. This is a natural intersection of the plugin and hook systems. |
| **Plugin Policy Enforcement** (allowlist/blocklist per organization) | **EXTENSION** | Enterprise-grade plugin policy enforcement is an enterprise extension. |

### MCP Integration

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **MCP Host Implementation** (connect to multiple servers, expose tools/resources/prompts) | **CORE** | AGH is designed as an ACP-based system. MCP is the standard protocol for agent-to-tool communication. Acting as an MCP host that can connect to external MCP servers and expose their tools identically to built-in tools is core infrastructure. |
| **Tool Namespacing** (`mcp__<server>__<tool>` convention) | **CORE** | When multiple servers can expose same-named tools, namespacing is essential for unambiguous dispatch. This is a core registry concern. |
| **Transport Abstraction** (stdio, SSE, HTTP, WebSocket, in-process) | **CORE (interface) / EXTENSION (transports)** | The transport interface is core. AGH should ship with `stdio` (most common for local tools) and `SSE/HTTP` (for remote). WebSocket and in-process transports are extensions. |
| **Session Recovery** (auto-reconnect on session expiry, 401 handling) | **CORE** | MCP sessions are stateful and servers restart. Transparent reconnection is essential for reliability in long-running daemon sessions. |
| **Output Size Management** (truncate large MCP results, persist to disk) | **CORE** | Already covered under result truncation -- applies uniformly to MCP and built-in tools. |
| **OAuth Flow for Remote MCP Servers** | **EXTENSION** | Browser-based OAuth for remote MCP servers is a specific authentication pattern. The core needs an MCP auth interface; OAuth is one implementation. |
| **MCP Server Approval Dialog** (user must approve new servers before connection) | **CORE** | Security boundary: preventing a malicious workspace config from silently launching subprocesses. The core must gate MCP server activation on explicit approval. |

### Agent Swarm and Subagents

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Subagent Spawning** (fork a new agent loop with its own context, tools, and system prompt) | **CORE** | AGH's ACP layer already spawns agents as subprocesses. The ability to spawn subagent sessions -- whether as forked loops, separate processes, or separate ACP instances -- is core to an agent OS. |
| **Three Execution Models** (fork/cache-shared, teammate/process-isolated, worktree/filesystem-isolated) | **CORE (interface) / EXTENSION (models)** | The core needs a `SubagentExecutionModel` interface. The fork model (cache-shared, same process) is core for efficiency. Teammate (separate process with mailbox) and worktree (git isolation) are extensions. |
| **Tool Scope Restriction per Agent** (subagents get a filtered tool set) | **CORE** | Different agents need different capabilities. The ability to filter the tool registry per-agent based on definitions or mode is core. |
| **File-Based Mailbox** (inter-agent communication via filesystem) | **EXTENSION** | The specific IPC mechanism (filesystem mailbox vs UDS vs channels) is an implementation choice. AGH already has UDS for CLI IPC -- agent-to-agent communication can use the same mechanism. The mailbox pattern is an extension. |
| **Plan Approval Flow** (teammate requests leader approval before escalating to act mode) | **CORE** | The two-phase commit where a subagent must get approval before gaining destructive capabilities is a safety pattern. The core needs a mechanism for capability escalation requests between agents. |
| **Shared Task List** (TodoV2: create, update, list tasks across agents) | **EXTENSION** | Task coordination across agents is a specific orchestration pattern. The core provides session state and messaging; shared task management is an extension. |
| **Swarm UI** (terminal spinners, progress tracking for multiple concurrent agents) | **EXTENSION** | The visualization of multi-agent activity is a UI concern. The core emits events; the UI renders them. |
| **Agent Memory Snapshots** (persist/restore agent knowledge across sessions and worktrees) | **EXTENSION** | Memory snapshotting for subagent continuity is an advanced persistence feature. |

### Settings System

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Hierarchical Settings Cascade** (policy > flag > local > project > user) | **CORE** | AGH's config package already handles TOML loading and merge. Extending it with a formal precedence hierarchy that supports enterprise policy overrides is core infrastructure. |
| **Schema Validation** (Zod-style validation with per-rule error isolation) | **CORE** | Validating configuration against a schema and isolating individual rule errors (so one bad rule does not invalidate the file) is essential for reliability. AGH should use Go struct tags + validation, with the same per-rule isolation principle. |
| **Hot-Reload** (file watcher with stability windows, internal-write suppression) | **CORE** | Settings changes should take effect without daemon restart. The daemon model makes this especially important -- the daemon is long-lived and needs to react to config changes. |
| **MDM / Enterprise Policy Enforcement** (OS-level managed settings that cannot be overridden) | **EXTENSION** | Enterprise MDM integration is a deployment concern. The core supports the precedence hierarchy; MDM-specific readers (plist, registry) are extensions. |
| **Environment Variable Injection** (settings-driven env vars for tool subprocesses) | **CORE** | Tools that spawn subprocesses need controllable environment. The settings system injecting env vars from config is a core feature for corporate proxy, custom paths, and similar concerns. |

### Observability and Diagnostics

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Event Recording** (structured event logging for every tool call, turn, and lifecycle event) | **CORE** | AGH's `observe` package already handles event recording and health metrics. This is core. |
| **Diagnostic Command** (single entry point showing installation health, conflicts, warnings) | **CORE** | A `doctor` equivalent that surfaces configuration errors, agent definition issues, permission conflicts, and health status is essential for operations. |
| **Telemetry Pipeline** (fan-out to multiple sinks) | **EXTENSION** | The specific telemetry sinks (Datadog, analytics collectors) are deployment-specific. The core provides structured events; telemetry export is an extension. |
| **Feature Flags** | **EXTENSION** | Remote feature flag evaluation is an operational concern, not a core requirement. The core can use build tags and config toggles. |
| **Auto-Update** | **EXTENSION** | Self-update mechanisms are distribution-specific and not part of the agent OS core. |
| **PII Redaction** (regex-based credential scrubbing before any data leaves the machine) | **CORE** | Any system that persists or transmits agent transcripts must scrub credentials. The redaction pipeline belongs in core. |

### Remote and Bridge System

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Remote Session Control** (local CLI controllable from web UI via authenticated channel) | **EXTENSION** | Remote control is a specific deployment mode. The core exposes HTTP/SSE and UDS APIs; remote bridge is an extension that uses those APIs. |
| **Session Teleport** (move active session between environments) | **EXTENSION** | Session migration across environments is an advanced operational feature. |
| **SSH Tunnel Integration** | **EXTENSION** | SSH-based remote access is a specific transport. |

### UI and Rendering

| Feature | Classification | Rationale |
|---------|---------------|-----------|
| **Terminal UI** (React/Ink TUI with permission dialogs, progress, streaming) | **EXTENSION** | AGH uses a web SPA (React 19, Vite, TanStack). The specific UI technology is an extension concern. The core provides HTTP/SSE APIs that any UI consumes. |
| **Permission Dialog UX** (tool-specific approval UI with "always allow" options) | **EXTENSION** | The specific UI for permission requests is a frontend concern. The core provides the permission decision API. |

## Architectural Patterns Worth Adopting

### 1. Uniform Tool Interface as Core Abstraction

**Pattern**: Every capability the agent can invoke -- filesystem, shell, web, MCP, custom -- implements the same interface with uniform schema, permissions, and execution semantics.

**Why AGH should adopt this**: AGH's ACP layer handles agent spawning, but tool execution within agent sessions needs the same uniformity. Define a `ToolDriver` interface in Go:

```go
type ToolDriver interface {
    Name() string
    InputSchema() Schema
    IsReadOnly() bool
    IsConcurrencySafe() bool
    CheckPermissions(ctx context.Context, input any) PermissionDecision
    Call(ctx context.Context, input any) (ToolResult, error)
}
```

This becomes the extension point for all tool implementations, including MCP-proxied tools.

### 2. Lifecycle Hook Bus with Structured Protocol

**Pattern**: A typed event bus with 25+ lifecycle events, where hooks can block, modify, or transform operations via a structured JSON output protocol.

**Why AGH should adopt this**: AGH's notifier pattern is already a typed fan-out interface. Extending it to a formal hook bus with:
- Pre/post execution events for tool calls
- Session lifecycle events (start, end, resume)
- Permission decision events
- Context management events (pre/post compact)

The structured output protocol (`continue`, `stopReason`, `updatedInput`, `transformedResult`) is what transforms hooks from passive observers into active participants. This is the pattern that turns AGH from a product into a platform.

### 3. Three-Layer Extension Reconciliation

**Pattern**: Separate intent (what the user configured), materialization (what is installed on disk), and activation (what is live in the runtime).

**Why AGH should adopt this**: AGH's skills catalog should adopt this exact pattern. A skill can be configured but not installed, installed but disabled, or active and running. Each layer reconciles independently, making the system robust to partial failures (corrupted skill file does not crash the daemon).

### 4. Progressive Disclosure for Skills/Capabilities

**Pattern**: Skills and agent definitions declare short metadata (name, description, when_to_use) that stays in context permanently. The full content materializes only when activated.

**Why AGH should adopt this**: With dozens or hundreds of skills, eager loading would blow the context budget. AGH's skill loader should present only metadata to the agent until activation, keeping the per-turn token cost constant regardless of skill library size.

### 5. Permission Cascade with Short-Circuit Evaluation

**Pattern**: A waterfall of permission checks ordered from most-restrictive to most-permissive, where each stage can short-circuit with a definitive answer.

**Why AGH should adopt this**: AGH needs a permission system for tool execution. The waterfall pattern (deny rules first, then tool-specific logic, then mode check, then allow rules, then user prompt) is the right structure because it guarantees that deny rules are always enforced and safe operations auto-approve without user interaction.

### 6. Smart Concurrency via Tool Metadata

**Pattern**: Partition tool calls by `isReadOnly()` and `isConcurrencySafe()` -- run reads in parallel, writes serially.

**Why AGH should adopt this**: AGH manages agent sessions that invoke tools. When an agent requests multiple tool calls in one turn, the daemon should partition them using the same metadata flags. This is a low-effort, high-impact optimization.

### 7. Static/Dynamic Prompt Split for Cache Efficiency

**Pattern**: Split system prompts into a rarely-changing static prefix (cacheable) and a per-request dynamic tail.

**Why AGH should adopt this**: The system prompt for ACP agents includes tool schemas, role instructions, and coding conventions (static) plus environment info, git status, and memory (dynamic). Splitting these lets the API cache the expensive static portion.

## Extension System Insights

### What Makes Claude Code's Extension Model Work

1. **Small, stable core interfaces**: The `Tool` interface, the `HookJSONOutput` protocol, and the `SKILL.md` contract are small and stable. Extensions implement them without needing to understand the rest of the codebase.

2. **Extensions cannot violate core invariants**: The permission waterfall, the tool execution pipeline, and the hook lifecycle all run in the core. Extensions plug into these pipelines -- they do not bypass them. A malicious plugin cannot skip the permission check because the check happens in the core pipeline, not in the plugin.

3. **Extensions are declared, not coded (where possible)**: Skills are Markdown files. Agent definitions are Markdown files. Hook configurations are JSON in settings. Permission rules are strings in settings. This low-code approach to extensions makes the system accessible to non-developers and auditable by security teams.

4. **Progressive complexity**: Simple extensions (a permission rule, a command hook) require zero code. Medium extensions (a skill with a procedure) require Markdown. Complex extensions (a plugin with MCP servers, tools, and hooks) require a manifest and code. The system supports all three levels without forcing everyone to the most complex level.

5. **Fail-safe degradation**: Missing plugins do not crash the daemon. Failed hooks return errors but do not block the pipeline (unless they explicitly return `continue: false`). Unreadable memory files silently degrade to no-memory. The core is designed to keep running even when extensions fail.

### Recommendations for AGH's Extension System

1. **Define the extension contract in Go interfaces, not in plugin APIs**: AGH's extensions should implement Go interfaces (`ToolDriver`, `HookExecutor`, `MemoryRecaller`, `PermissionClassifier`). The daemon loads extensions that fulfill these interfaces. This is Go-native and avoids the complexity of a plugin framework.

2. **Support declarative extensions via TOML/YAML/Markdown**: Not every extension needs compiled code. Skills (Markdown), permission rules (TOML config), hook commands (shell commands in TOML), and agent definitions (Markdown frontmatter) should all work without compilation.

3. **Use the notifier pattern for the hook bus**: AGH's existing notifier pattern is the right foundation. Extend it with typed lifecycle events and the structured output protocol so hooks can participate in decisions, not just observe them.

4. **Make MCP a first-class citizen**: Since AGH speaks ACP, it should also speak MCP for tool access. MCP tools should be indistinguishable from built-in tools in the tool registry, permission system, and hook pipeline. This is what makes the tool ecosystem open-ended.

5. **Ship a minimal set of bundled tools and let everything else be extensions**: The core should ship with: file read, file write, file edit, shell execution, glob, grep, and MCP bridge. Everything else -- web fetch, web search, notebook editing, remote triggers -- should be extensions that demonstrate the tool interface.

6. **Invest in the permission system early**: Claude Code's permission model is its most mature subsystem and arguably its most important. AGH should build the permission cascade and the hook bus before building advanced features, because every advanced feature depends on them.
