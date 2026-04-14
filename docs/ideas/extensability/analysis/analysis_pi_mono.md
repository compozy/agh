# Pi-Mono Analysis for AGH Extensibility Design

## Overview

Pi-Mono is a TypeScript monorepo by Mario Zechner that implements an "aggressively extensible" AI coding agent framework. It consists of seven packages organized in three tiers: a foundation LLM API (`pi-ai`), infrastructure packages (`pi-agent-core`, `pi-tui`), and application-level consumers (`pi-coding-agent`, `pi-mom`, `pi-web-ui`, `pi-pods`). The project's guiding thesis is that coding agents should ship a minimal core with comprehensive extension points, letting users compose exactly the features they need rather than accepting a monolithic feature set.

Pi-Mono's philosophy directly opposes "batteries-included" tools: it ships only 4 default tools (read, write, edit, bash), keeps its system prompt under 1,000 tokens, and deliberately omits MCP support, sub-agents, permission systems, plan mode, built-in todos, and background bash -- all of which can be rebuilt via its extension system. This minimalism is driven by a concrete technical constraint: context windows are finite, and every token consumed by framework overhead is unavailable for the user's actual task.

**Relevance to AGH**: Pi-Mono validates the "robust minimal core + extensible plugin system" philosophy that AGH already pursues. It provides a detailed case study of where that boundary should be drawn and what extension surface area looks like in practice for an agent operating system.

---

## Key Features Analysis

### Feature Classification Table

| #   | Feature                                                        | Pi-Mono Implementation                                                                                                                                                                       | AGH Classification        | Rationale                                                                                                                                                                                                                                                                                                                                                |
| --- | -------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Unified LLM Streaming API**                                  | `pi-ai`: single `stream()` call over 20+ providers, 10 API protocols, lazy-loaded provider modules, canonical message types                                                                  | **EXTENSION**             | AGH's ACP protocol already abstracts agent communication via JSON-RPC over stdio. AGH spawns complete agent binaries (Claude Code, Codex, Gemini CLI) that handle their own LLM provider connections. A unified LLM API would be useful for future "native" agents AGH spawns directly, but should remain an optional provider package rather than core. |
| 2   | **Extension System (TypeScript modules with lifecycle hooks)** | `ExtensionAPI` with 50+ hooks: tools, commands, shortcuts, events, UI components, providers, state persistence. Extensions loaded via jiti transpiler.                                       | **CORE (design pattern)** | The extension system design is the single most important pattern to adopt. AGH needs a Go-native equivalent: a plugin/extension interface with lifecycle hooks at every boundary (session start/end, tool execution, message streaming, compaction). This should be the core abstraction, not an afterthought.                                           |
| 3   | **Skills (on-demand capability packages)**                     | Markdown files with YAML frontmatter following AgentSkills.io spec. Progressive disclosure: only name+description in system prompt, full content loaded on demand.                           | **CORE**                  | AGH already has `internal/skills` with bundled skill definitions. Pi validates this approach and adds the key insight of progressive disclosure (load-on-demand to save context tokens). AGH's skills loader should follow this pattern.                                                                                                                 |
| 4   | **Prompt Templates**                                           | Markdown files with bash-style variable substitution (`$1`, `$@`, `${@:N:L}`). Expanded via `/name args`.                                                                                    | **EXTENSION**             | Useful but not essential to the daemon core. Should be an extension that prompt template directories can register with the skills system. AGH's workspace/config system can discover these.                                                                                                                                                              |
| 5   | **Session Tree (JSONL with branching)**                        | Append-only JSONL where each entry has `id`/`parentId`, forming a tree. Branching via `leafId` pointer. No data ever deleted.                                                                | **CORE (adapt)**          | AGH already uses SQLite for session events (`store/sessiondb`). The tree-branching concept (navigate to any point, branch without losing history) is valuable for AGH's session model. The `leafId`-based branching pattern could be adapted to SQLite with a `parent_event_id` column.                                                                  |
| 6   | **Context Compaction**                                         | Structured summarization when context exceeds threshold. Walks backwards to find cut point preserving recent tokens. Iterative compaction builds on previous summaries.                      | **CORE**                  | Critical for long-running sessions. AGH's `memory/consolidation` package handles dream consolidation, but per-session compaction for context window management is a separate concern that belongs in core. The structured summary format (Goal, Progress, Key Decisions, Next Steps) is a good template.                                                 |
| 7   | **Auto-generated Model Catalog**                               | Build-time script scrapes provider APIs, writes `models.generated.ts` with type-safe model definitions including pricing, context windows, capabilities.                                     | **EXTENSION**             | AGH delegates model selection to agent binaries. If AGH ever needs to route to specific models, this could be a useful extension. Not core.                                                                                                                                                                                                              |
| 8   | **Cross-Provider Message Handoffs**                            | `transform-messages.ts` converts thinking blocks, normalizes tool call IDs, repairs orphaned tool calls, sanitizes Unicode when switching models mid-conversation.                           | **EXTENSION**             | Relevant only if AGH manages LLM connections directly. Currently agents handle their own provider connections. Could become relevant for Phase 3 agent network protocol.                                                                                                                                                                                 |
| 9   | **TUI Framework (differential rendering)**                     | Standalone package with component tree, differential rendering (only redraws changed regions), synchronized output (CSI 2026), overlay system, Kitty keyboard protocol.                      | **N/A (not applicable)**  | AGH uses a React 19 SPA for its web UI and UDS for CLI. A TUI framework is not needed. However, the differential rendering concept is instructive for SSE-based UI updates.                                                                                                                                                                              |
| 10  | **Theme System (hot-reloadable)**                              | 51 color tokens in JSON, hot-reload via `fs.watch` with debounce, terminal capability detection, syntax highlighting integration.                                                            | **EXTENSION**             | Visual customization belongs in the web UI layer, not the daemon core. AGH's web UI already uses Tailwind/shadcn.                                                                                                                                                                                                                                        |
| 11  | **Package Manager (npm/git/local)**                            | `pi install`, `pi remove`, `pi update`. Packages bundle extensions, skills, prompts, themes. Supports npm, git, and local sources. Auto-install on startup from `.pi/settings.json`.         | **CORE (adapted)**        | AGH needs a package/plugin distribution mechanism. The concept of bundling skills, extensions, and config into installable packages is essential for the extension ecosystem. Should be adapted to Go (e.g., git-based plugin repos with `agh install`).                                                                                                 |
| 12  | **Pi Mom (Self-managing Slack Bot)**                           | Headless agent deployment in Slack. Per-channel isolation with separate workspaces, MEMORY.md files, skills directories. Docker sandbox. Events system (immediate, one-shot, periodic/cron). | **EXTENSION**             | Demonstrates a powerful application pattern: the same agent core deployed headlessly into a chat platform. AGH should enable this via its API layer, not by building it into core. The events system (cron-based agent triggers) is a good extension candidate.                                                                                          |
| 13  | **Steering & Follow-up Message Queues**                        | Two-queue system: steering messages redirect agent mid-turn, follow-up messages queue for after completion. Drain modes: "one-at-a-time" vs "all".                                           | **CORE**                  | Essential for interactive agent sessions. AGH's session manager should support injecting messages into running sessions with priority semantics (interrupt vs. queue). This maps directly to AGH's HTTP/SSE API.                                                                                                                                         |
| 14  | **Tool Execution Pipeline (parallel/sequential with hooks)**   | Preflight sequential, execute parallel, finalize in source order. `beforeToolCall` can block, `afterToolCall` can modify results. File mutation queue for concurrent writes.                 | **CORE (design pattern)** | AGH delegates tool execution to agent subprocesses, but the hook pattern (before/after with block/modify capability) is relevant for the observe layer and for extensions that want to intercept tool calls visible via ACP events.                                                                                                                      |
| 15  | **Custom Message Types (declaration merging)**                 | TypeScript `CustomAgentMessages` interface widened via declaration merging. Custom messages in transcript but filtered from LLM context by `convertToLlm`.                                   | **CORE (adapted)**        | AGH's event store should support custom event types from extensions. The pattern of storing extension-specific data in the event stream (but excluding it from agent context) is directly applicable to `store/sessiondb`.                                                                                                                               |
| 16  | **OAuth Provider System**                                      | 5 built-in OAuth providers (Anthropic, OpenAI Codex, GitHub Copilot, Gemini CLI, Antigravity). `AuthStorage` with file-based locking. Token auto-refresh.                                    | **EXTENSION**             | Auth management for LLM providers. AGH delegates this to agent binaries, so not needed in core. Could be an extension for future native agent support.                                                                                                                                                                                                   |
| 17  | **Web UI Components (mini-lit)**                               | Web components for chat interfaces: ChatPanel, AgentInterface, MessageList, artifacts system, sandboxed iframe execution, IndexedDB storage, custom tool renderers.                          | **N/A (parallel)**        | AGH has its own React 19 SPA. Not directly adoptable, but the artifact system (LLM creates/modifies files rendered interactively) and custom tool renderer registry are patterns worth replicating in AGH's web UI.                                                                                                                                      |
| 18  | **GPU Pod Management**                                         | CLI for deploying vLLM on remote GPU pods. SSH-based provisioning, model lifecycle, health monitoring, OpenAI-compatible endpoints.                                                          | **EXTENSION**             | Infrastructure automation for self-hosted LLMs. Clearly an extension/plugin, not core to an agent OS.                                                                                                                                                                                                                                                    |
| 19  | **Context File Discovery (AGENTS.md)**                         | Loads context files from global, parent directories, and current directory. Both `AGENTS.md` and `CLAUDE.md` recognized. Injected into system prompt.                                        | **CORE**                  | AGH's workspace resolver already handles this pattern. Validates the approach. The progressive discovery (walk up from cwd to root) is the right pattern.                                                                                                                                                                                                |
| 20  | **RPC Mode (JSONL over stdin/stdout)**                         | Headless mode using LF-delimited JSONL for IDE integration. Extension UI forwarded as typed requests.                                                                                        | **CORE (validates)**      | AGH already has UDS for CLI IPC. Pi's RPC mode validates that a structured protocol over stdio is essential for embedding agents in IDEs and other host processes.                                                                                                                                                                                       |
| 21  | **Dual-Scope Memory (Global + Channel/Workspace)**             | MEMORY.md files at global and per-channel levels. Read before every response, injected into system prompt. Editable by both human and agent.                                                 | **CORE**                  | AGH already has `internal/memory` with global + workspace scope. Pi's implementation via plain Markdown files validates the approach and emphasizes that memory should be human-readable and editable.                                                                                                                                                   |
| 22  | **Events/Scheduling System**                                   | Three event types: immediate, one-shot (timestamp), periodic (cron). File-based triggers (`events/` directory). Queue cap per channel. Silent completion for no-op periodic checks.          | **EXTENSION**             | Scheduled agent triggers are a powerful pattern but belong as an extension. AGH's daemon could expose a scheduling API that extensions register with.                                                                                                                                                                                                    |
| 23  | **Cost Tracking**                                              | Per-message `Usage` object with token counts and dollar costs. `calculateCost()` from model pricing metadata. Real-time display in TUI/web UI.                                               | **CORE**                  | Observable cost tracking is essential for an agent OS. AGH's `observe` package should track token usage and cost per session, derived from ACP events that report usage.                                                                                                                                                                                 |

---

## Architectural Patterns Worth Adopting

### 1. Layered Package Architecture with Strict Dependency Flow

Pi-Mono's three-tier architecture is its most important structural decision:

```
Foundation:    pi-ai (zero internal deps)
Infrastructure: pi-agent-core (depends on pi-ai), pi-tui (standalone)
Application:    pi-coding-agent, pi-mom, pi-web-ui, pi-pods
```

**Key rules**: Dependencies flow strictly downward. No package imports `pi-coding-agent` (the top-level app). The foundation layer has zero internal dependencies. Infrastructure packages depend only on foundation. Application packages pull together lower layers.

**AGH parallel**: AGH already follows this with `daemon/` as sole composition root and downward-only dependency flow. This validates AGH's approach. The additional insight is that AGH's `internal/api/` packages should never be imported by core domain packages (`session/`, `memory/`, `skills/`), which AGH already enforces.

### 2. Progressive Disclosure for Context Optimization

Pi's most impactful design insight is that context windows are finite and expensive. Every feature decision is filtered through "what does this cost in tokens?"

- Skills inject only name+description into the system prompt; full content is loaded on-demand when the agent decides it's relevant.
- No elaborate system prompts -- under 1,000 tokens.
- No MCP tool definitions burning context tokens whether used or not.
- Compaction keeps recent tokens intact while summarizing older ones.

**AGH adoption**: AGH should adopt progressive disclosure as a first-class principle in its skills system. When AGH sends context to agents, skills should be listed as brief descriptors, with the full skill content available via a "read skill" mechanism. This directly reduces the system prompt overhead per agent session.

### 3. Single-File Session Trees

Pi stores entire conversation histories, including all branches, in a single append-only JSONL file. Branching is achieved by appending entries with `parentId` pointing to earlier entries rather than the current leaf. Nothing is ever deleted.

**Benefits**: No multi-file branch management, complete audit trail, standard format parseable by any tool, no data loss from aborts or crashes (append-only).

**AGH adaptation**: AGH uses SQLite per-session, which is more powerful but less inspectable. Consider adding a `parent_event_id` column to session events to enable tree-structured branching. The append-only guarantee maps naturally to SQLite's INSERT-only pattern. The key insight is that branching should be a core session primitive, not an afterthought.

### 4. Extension Points as First-Class API Surface

Pi's extension API exposes 50+ hooks organized into clear categories:

- **Session lifecycle**: start, before_switch, before_fork, before_compact, compact, shutdown
- **Agent lifecycle**: before_agent_start, agent_start/end, turn_start/end
- **Message lifecycle**: message_start, message_update, message_end
- **Tool lifecycle**: tool_call (can block), tool_result (can modify), tool_execution_start/update/end
- **Input**: transform user input before agent processing
- **Context**: modify messages before LLM call
- **Resources**: contribute additional skill/prompt/theme paths

The critical patterns:

- **`tool_call` can block execution** -- extensions can implement permission gates by returning `{ block: true, reason: "..." }`
- **`tool_result` chains like middleware** -- each extension handler can modify results, patches merge sequentially
- **`before_agent_start` can inject messages** -- extensions add context without modifying core logic

**AGH adoption**: This is the blueprint for AGH's extension system. In Go, these hooks should be typed interfaces that extension packages implement. The `Notifier` pattern AGH already uses is a good foundation; it needs to be extended with blocking/modification semantics for tool call interception.

### 5. Conflict Resolution Rules

Pi has deterministic rules for when extensions collide:

- **Shortcuts**: Reserved keybindings cannot be overridden. Non-reserved conflicts generate warnings.
- **Commands**: Built-in always wins. Extension-vs-extension: first-registered wins, duplicates get numeric suffixes.
- **Tools**: Built-in conflicts produce warnings. First registration wins for extension-vs-extension.
- **Providers**: Can override built-in by ID. Unregister restores defaults.

**AGH adoption**: AGH needs explicit conflict resolution policies defined before the extension system is built. The "built-in always wins" rule is sensible. The "first-registered wins with warnings" approach avoids silent breakage.

### 6. Lazy Loading and Registration

Pi never eagerly imports heavy dependencies. Provider modules are loaded via dynamic `import()` only when first used, with `||=` caching to ensure single-load semantics. Errors during lazy load are encoded as events in the stream, never thrown as unhandled exceptions.

**AGH adoption**: In Go, this translates to lazy initialization of extension packages. Extensions should register intent at startup (name, capabilities, hooks) but defer heavy initialization (database connections, subprocess spawning) until first use. Errors should be captured and reported through the observe layer, not panic.

---

## Extension System Insights

### Architecture: Hook-Based with Full API Surface

Pi's extension system is its defining feature. The core insight is that the extension API should be **exactly as powerful as the internal API**. Extensions import the same packages the agent uses. There is no restricted sandbox, no capability manifest, no permission model. This is a deliberate design choice: the target audience is developers who already run arbitrary code.

**Extension loading flow**:

1. Discovery: scan `~/.pi/agent/extensions/`, `.pi/extensions/`, and package manifests
2. Transpile: use jiti (just-in-time TypeScript transpiler) with virtual modules for bundled packages
3. Execute: call each extension's default export function with `ExtensionAPI`
4. Bind: wire real action methods into the runtime after initialization
5. Dispatch: route events through `ExtensionRunner` which sits between `AgentSession` and extensions

**For AGH in Go**: The equivalent would be:

- Discovery: scan `~/.agh/extensions/`, `.agh/extensions/`, and registered plugin directories
- Load: Go plugins (`plugin.Open()`) or, more practically, subprocess-based plugins communicating via gRPC/JSON-RPC
- Register: each plugin exports a registration function that receives an `ExtensionAPI` interface
- Bind: wire hooks into daemon lifecycle after all plugins register
- Dispatch: route events through an extension runner that sits between `session.Manager` and extensions

### The Four Extension Surfaces

Pi provides four distinct extension mechanisms, each targeting a different user sophistication level:

| Surface                       | Complexity | Capability               | Token Cost           |
| ----------------------------- | ---------- | ------------------------ | -------------------- |
| **Context files** (AGENTS.md) | Zero code  | Persistent instructions  | Always loaded        |
| **Skills** (SKILL.md)         | Zero code  | On-demand procedures     | Loaded when relevant |
| **Prompt Templates** (\*.md)  | Zero code  | Reusable shortcuts       | Loaded on invocation |
| **Extensions** (\*.ts)        | TypeScript | Full runtime integration | No token cost        |

**AGH mapping**:

- Context files: AGH workspace already supports this via `CLAUDE.md` / config
- Skills: AGH's `internal/skills` package -- validate with progressive disclosure
- Prompt Templates: New extension type, low priority
- Extensions: Primary focus for AGH's extension system design

### Multi-Package Extensibility (Pi Packages)

Pi's package system bundles all four extension surfaces into distributable units:

```json
{
  "pi": {
    "extensions": ["./extensions"],
    "skills": ["./skills"],
    "prompts": ["./prompts"],
    "themes": ["./themes"]
  }
}
```

Packages support three source types (npm, git, local), two scopes (global and project-local), version pinning, selective resource loading via glob patterns, and offline mode.

**AGH design implications**:

- AGH packages should bundle: extensions (Go plugins or subprocess handlers), skills (Markdown), config templates, and web UI components
- Source types: git repositories (primary), local directories (development)
- Scopes: global (`~/.agh/packages/`) and workspace-local (`.agh/packages/`)
- The `agh install <git-url>` command installs a package by cloning the repo and registering its contents
- A manifest file (`agh-package.toml` or similar) declares what the package provides
- Auto-install from workspace config ensures team consistency

### Security Model: Full Trust with Escape Hatches

Pi runs extensions with full trust -- no sandbox, no capability restrictions. The rationale: once an agent can read, write, and execute code, preventing exfiltration while maintaining utility is impossible. Security theater (permission popups) provides false assurance.

Real security comes from:

- **Containers**: Run in Docker/VM for genuine isolation
- **Scope limitation**: Project-local extensions only affect that project
- **Audit**: Package provenance via npm/git audit tools
- **Extension permission gates**: Extensions themselves can add confirmation flows

**AGH consideration**: AGH should follow the same model for extension trust. Since AGH runs as a daemon, the security boundary is the daemon's process permissions. Extensions run in the daemon's process (or as supervised subprocesses) and inherit its permissions. The real security boundary is the container/VM that runs the daemon.

### Event System Design

Pi's event system has two critical properties:

1. **Listeners are awaited sequentially** -- a slow listener blocks subsequent listeners and the loop itself. This is by design: it makes `message_end` processing a barrier before tool preflight, ensuring state consistency.

2. **State is updated before listeners fire** -- when an event arrives, internal state (messages, pending tool calls, streaming state) is updated first, then listeners are invoked. Listeners always see consistent state.

**AGH adoption**: The `observe` package's notifier pattern should follow both rules. Events should be dispatched synchronously through registered handlers in registration order, with state mutations committed before notification. For blocking operations (like permission gates), the handler should be able to return a result that the caller inspects.

---

## Summary of Recommendations for AGH

### Adopt as CORE

1. **Extension hook system** with lifecycle events at session, agent, tool, and message boundaries
2. **Progressive disclosure** for skills (name+description in context, full content on demand)
3. **Session branching** via parent-event relationships in SQLite
4. **Context compaction** with structured summarization and iterative updates
5. **Steering/follow-up message queues** for injecting messages into running sessions
6. **Package distribution** mechanism for bundling extensions, skills, and config
7. **Cost tracking** integrated into the observe layer
8. **Conflict resolution** policies defined upfront for extension collisions

### Adopt as EXTENSION

1. Unified LLM API (for future native agent support)
2. Prompt templates (Markdown with variable substitution)
3. Theme/visual customization
4. Scheduled event triggers (cron-based agent wake-ups)
5. OAuth provider management
6. Chat platform integrations (Slack, Discord, etc.)
7. GPU pod management / self-hosted model deployment
8. Cross-provider message transformation

### Do Not Adopt

1. TUI framework (AGH uses web UI)
2. TypeScript-specific patterns (declaration merging, jiti transpiler)
3. "No MCP" stance (AGH should support MCP as an extension surface since it spawns external agents that may use MCP)
4. Single-file JSONL storage (AGH's SQLite approach is better for the daemon model)
