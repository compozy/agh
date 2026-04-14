# Hermes Agent Analysis for AGH

## Overview

Hermes is a Python-based self-improving AI agent by Nous Research, designed as a long-lived background process reachable from any channel (CLI, Telegram, Discord, Slack, WhatsApp, Signal, Email, Matrix, Home Assistant). Its hub-and-spoke architecture centers on a synchronous `AIAgent` core driven by a five-phase `run_conversation()` loop, surrounded by a self-registering tool system, SQLite+FTS5 session store, eight messaging platform adapters, six terminal execution backends, a cron scheduler, a learning loop (persistent memory + skills + session recall), and an ACP adapter for IDE integration.

Hermes represents a "maximalist kitchen-sink" approach: everything is built in, everything shares the same process, and every interface uses the same registry, session database, memory, and skills directory. This contrasts with AGH's philosophy of a robust minimal core with a highly extensible plugin system.

### Key Architectural Differences from AGH

| Dimension       | Hermes                                                | AGH                                                      |
| --------------- | ----------------------------------------------------- | -------------------------------------------------------- |
| Language        | Python (synchronous core, async gateway)              | Go (single binary)                                       |
| Agent coupling  | One hub class (`AIAgent`) that does everything        | Separate packages wired via daemon composition root      |
| Extension model | Module-level self-registration singletons             | Go interfaces + dependency injection                     |
| Session store   | Single SQLite file shared across all interfaces       | Per-session event store + global catalog                 |
| Memory          | Flat markdown file (`memory.md`) + optional Honcho    | Dual-scope (global + workspace) with dream consolidation |
| Skills          | agentskills.io markdown format in `~/.hermes/skills/` | Bundled skill definitions + catalog/loader               |
| Communication   | Direct subprocess or gateway adapters                 | ACP over JSON-RPC/stdio                                  |
| Observability   | `/insights` command + session cost accounting         | Event recording, health metrics, query engine            |

## Key Features Analysis

| #   | Feature                                      | Hermes Implementation                                                                                                                           | Classification                          | Rationale                                                                                                                                                                                                                                                               |
| --- | -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Self-registering tool registry**           | Module-level singleton; tools register at import time with name, toolset, schema, handler, `check_fn()`, `requires_env`                         | **CORE**                                | AGH already has a tools concept via ACP. A typed Go tool registry with availability gating (`check_fn` equivalent) and toolset composition should be core -- it is the primary extensibility surface.                                                                   |
| 2   | **Toolset composition**                      | Named groups (`web`, `research`, `full_stack`) with recursive `includes` for bulk enable/disable                                                | **CORE**                                | Toolset grouping with enable/disable per-session is load-bearing for agent safety and flexibility. Core registry should support grouping.                                                                                                                               |
| 3   | **Availability gating via check_fn**         | Tools withheld from model when API keys missing or deps unavailable -- model never sees tools it cannot use                                     | **CORE**                                | Critical reliability property. AGH's tool definitions should include an availability predicate. Prevents hallucinated calls.                                                                                                                                            |
| 4   | **Session store with FTS5**                  | Single SQLite `state.db` with `sessions`, `messages`, `messages_fts` tables; WAL mode; write retries with jitter                                | **CORE**                                | AGH already has SQLite stores. Adding FTS5 for cross-session search is a core capability for any agent OS -- it enables recall without external search infra. The schema pattern (FTS5 virtual table with content-external triggers) is directly portable to Go+SQLite. |
| 5   | **Cross-session recall (session_search)**    | FTS5 query -> group by session -> LLM summarization per session group                                                                           | **EXTENSION**                           | The two-stage pipeline (FTS5 retrieval + LLM summarization) is an opinionated recall strategy. AGH core should expose the FTS5 search primitive; the LLM summarization layer should be an extension.                                                                    |
| 6   | **Persistent memory (memory_tool)**          | Flat `memory.md` file with categorized facts; injected into system prompt every turn                                                            | **CORE**                                | AGH already has `internal/memory` with dual-scope memory. Hermes validates that simple persistent facts injected into system prompts is table-stakes. Keep in core.                                                                                                     |
| 7   | **Memory provider plugins**                  | `BuiltinMemoryProvider` (markdown) vs `HonchoMemoryProvider` (external API) via `MemoryManager` abstraction                                     | **CORE pattern, EXTENSION providers**   | The provider interface pattern belongs in core. Specific providers (Honcho, vector DB, etc.) are extensions.                                                                                                                                                            |
| 8   | **Skills pipeline (agentskills.io)**         | Markdown files with YAML frontmatter; slash-command activation; injected as user message; auto-proposal from trajectories                       | **CORE**                                | AGH already has `internal/skills`. Hermes reinforces that skills should be: (a) markdown-based, (b) frontmatter-indexed, (c) injected as context not system prompt, (d) discoverable via tools. Keep in core.                                                           |
| 9   | **Skill auto-proposal**                      | Agent calls `skill_manage(action="propose")` after complex tasks to create new skills from completed trajectories                               | **EXTENSION**                           | Self-improvement is powerful but opinionated. The skill CRUD API should be core; the auto-proposal heuristic ("detect complex task completion and propose a skill") should be an extension/hook.                                                                        |
| 10  | **Context compression**                      | 5-step algorithm: prune old tool results, protect head/tail, LLM-summarize middle, rebuild, chain sessions via `parent_session_id`              | **CORE**                                | Context management is fundamental for long-running sessions. AGH should have a compaction interface in core with the default implementation. Session chaining via parent references is a good schema pattern.                                                           |
| 11  | **Prompt caching (Anthropic)**               | `apply_anthropic_cache_control()` marks system prompt + last 3 messages with `cache_control` breakpoints                                        | **CORE**                                | Provider-specific optimization, but the concept of marking stable context for caching is universal. AGH's transcript assembly should support cache-hint annotations.                                                                                                    |
| 12  | **Gateway / platform adapters**              | 8 adapters (Telegram, Discord, Slack, WhatsApp, Signal, Email, Matrix, Home Assistant) via `BaseAdapter` interface                              | **EXTENSION**                           | Definitively an extension. The `BaseAdapter` contract (connect, start, send_text, send_message, edit_message, delete_message, on_message, on_command) is a good interface to define in core and let extensions implement.                                               |
| 13  | **Unified command registry**                 | `COMMAND_REGISTRY` shared across CLI and all gateway platforms; `cli_only` / `gateway_only` flags                                               | **CORE**                                | A shared command dispatch table is core infrastructure. Commands registered once should be available across all interfaces.                                                                                                                                             |
| 14  | **Cron / scheduled automations**             | Natural-language tasks on cron schedule; delivery to any platform; `[SILENT]` marker convention                                                 | **EXTENSION**                           | Scheduling is not part of a minimal agent OS core. It is a compelling extension that uses core primitives (session creation, agent execution, platform delivery).                                                                                                       |
| 15  | **Terminal execution backends**              | 6 pluggable backends (Local, Docker, SSH, Modal, Daytona, Singularity) via `BaseEnvironment` interface                                          | **CORE interface, EXTENSION backends**  | The interface (`execute(command, cwd, timeout) -> {output, returncode}`) belongs in core. Individual backends are extensions. AGH already handles this via ACP subprocess spawning, but a pluggable execution environment concept is valuable.                          |
| 16  | **Subagent delegation**                      | `delegate_task` spawns isolated child `AIAgent` in ThreadPoolExecutor; zero-context-cost for parent; blocked tools prevent recursion            | **CORE**                                | Agent composition is fundamental. AGH should support spawning child sessions with isolated contexts, restricted tool access, and independent iteration budgets. The depth limit and blocked-tool pattern are good safety defaults.                                      |
| 17  | **ACP adapter (IDE integration)**            | JSON-RPC server exposing `initialize`, `tools/list`, `tools/call`, `completion/complete`, `resources/read`; stateful sessions with code context | **CORE**                                | AGH already has ACP as its primary agent communication protocol. Hermes validates the pattern: expose the same tool registry and session semantics over JSON-RPC for IDE integration.                                                                                   |
| 18  | **Security: dangerous command detection**    | Regex patterns for destructive commands; command normalization (ANSI stripping, null byte removal, NFKC); Tirith binary scanner                 | **CORE**                                | Command safety is essential for any agent that executes shell commands. AGH should have a core command-approval interface with default regex patterns.                                                                                                                  |
| 19  | **Approval state machine**                   | Three scopes: once / session / permanent; CLI interactive prompts; gateway async approval via chat buttons                                      | **CORE**                                | The approval interface (check -> prompt -> remember) belongs in core. The persistence scope hierarchy is a good pattern.                                                                                                                                                |
| 20  | **SSRF / URL safety**                        | Block private IP ranges, cloud metadata endpoints, user-defined blocklist                                                                       | **CORE**                                | Network safety for agent web requests is core security.                                                                                                                                                                                                                 |
| 21  | **Gateway authorization**                    | Priority chain: platform allow-all -> DM pairing -> platform allowlist -> global allowlist -> global allow-all -> deny                          | **EXTENSION**                           | Gateway auth is specific to multi-user messaging scenarios. Extension responsibility.                                                                                                                                                                                   |
| 22  | **DM pairing system**                        | Cryptographic pairing codes for granting messaging platform access                                                                              | **EXTENSION**                           | Platform-specific access control. Extension.                                                                                                                                                                                                                            |
| 23  | **Process management**                       | `ProcessRegistry` for background processes; spawn/poll/wait/kill/read_log; PTY support; crash recovery via checkpoint file                      | **EXTENSION**                           | Background process tracking is valuable but not core agent OS. The registry pattern and lifecycle management are good extension material.                                                                                                                               |
| 24  | **Persistent shell state**                   | `PersistentShellMixin` maintains shell state across tool calls; SSH ControlMaster                                                               | **EXTENSION**                           | Implementation detail of terminal execution. Extension.                                                                                                                                                                                                                 |
| 25  | **Token accounting & cost estimation**       | `CanonicalUsage` tracks input/output/cache/reasoning tokens; per-model pricing; session-level cost rollup                                       | **CORE**                                | Usage tracking is core observability. AGH's `internal/observe` should track token economics per session.                                                                                                                                                                |
| 26  | **Diagnostic tools (doctor/status)**         | `hermes doctor` validates config, deps, tools; `hermes status` shows component health; `InsightsEngine` for analytics                           | **EXTENSION**                           | Diagnostics are important but not core agent loop. Good extension that uses core health/metrics APIs.                                                                                                                                                                   |
| 27  | **Batch processing / trajectory generation** | `BatchRunner` with multiprocessing; toolset distribution sampling; JSONL trajectory output; `TrajectoryCompressor`                              | **EXTENSION**                           | Training-data generation is Nous-specific. Not core agent OS.                                                                                                                                                                                                           |
| 28  | **RL training environments**                 | Atropos integration, `HermesAgentBaseEnv`                                                                                                       | **EXTENSION**                           | Research-specific.                                                                                                                                                                                                                                                      |
| 29  | **Voice / TTS system**                       | Multi-provider STT (faster-whisper, Groq, OpenAI) + TTS (Edge, ElevenLabs, OpenAI); Discord voice channels                                      | **EXTENSION**                           | Modality-specific. Extension with provider plugin interface.                                                                                                                                                                                                            |
| 30  | **Honcho user modeling**                     | Dialectic user modeling via external API; semantic search, peer cards, configurable write strategies                                            | **EXTENSION**                           | External memory provider. Extension.                                                                                                                                                                                                                                    |
| 31  | **Authentication / provider system**         | 11 providers; 4 auth types (OAuth device code, OAuth external, API key, external process); credential resolution chain                          | **CORE interface, EXTENSION providers** | A provider resolution interface (credential lookup chain, model validation) belongs in core. Individual provider implementations are extensions.                                                                                                                        |
| 32  | **MCP server integration**                   | External MCP servers discovered at startup; tools namespaced under server name                                                                  | **CORE**                                | MCP tool integration is part of the standard agent protocol ecosystem. AGH should support discovering and proxying MCP servers as a core capability.                                                                                                                    |
| 33  | **User plugins**                             | `~/.hermes/plugins/` directory; Python modules loaded at startup; register tools via the same registry                                          | **CORE mechanism**                      | User-authored tool extensions via a plugin directory is a core extensibility mechanism.                                                                                                                                                                                 |
| 34  | **Streaming response delivery**              | `StreamingResponse` class: buffer 500 chars or 2s timeout; progressive message editing on platforms that support it                             | **EXTENSION**                           | Platform-specific delivery optimization. Extension.                                                                                                                                                                                                                     |
| 35  | **System prompt builder**                    | Ordered concatenation of stable sections (identity, platform hints, skills index, memory, context files, guidance blocks) for cache stability   | **CORE**                                | System prompt assembly order matters for caching. AGH should have a structured prompt builder with ordered sections.                                                                                                                                                    |

## Architectural Patterns Worth Adopting

### 1. Learning Loop (Memory -> Skills -> Session Recall)

Hermes implements a three-layer learning loop that feeds back into every subsequent session:

- **Persistent memory**: durable facts saved via tool call, injected into system prompt
- **Skills**: procedural knowledge crystallized from completed tasks, invocable on demand
- **Session recall**: FTS5 search + LLM summarization across all historical sessions

**AGH relevance**: AGH already has dual-scope memory with dream consolidation and a skills catalog. The key pattern to adopt is the **closed feedback loop**: the agent should be able to save memories, create skills, and search past sessions -- all via tool calls within the same conversation. The dream consolidation AGH already has goes beyond Hermes (which has no automatic consolidation). The FTS5 cross-session search pattern is the missing piece AGH should add to its `internal/store` layer.

**Recommendation**: Add FTS5 indexing to sessiondb event content. Expose a `session_search` capability as a core tool. Let the LLM summarization of results be an extension point.

### 2. Cron / Scheduled Automations

Hermes runs a 60-second tick loop in its gateway process, checking `~/.hermes/cron/jobs.json` for due jobs. Each job carries a natural-language command, a cron-syntax trigger, and a delivery target.

**AGH relevance**: For an Agent OS, scheduled execution is a strong differentiator. An agent that can autonomously perform tasks on schedule, route output to platforms, and suppress noise with `[SILENT]` markers transforms from a reactive tool to a proactive assistant.

**Recommendation**: Implement as an **extension** that registers with the daemon. Core should expose: (a) a way to create sessions programmatically (already exists), (b) a delivery/notification interface for routing output, (c) a timer/scheduler hook in the daemon lifecycle. The cron extension then uses these primitives.

### 3. Gateway / Platform Adapters

Hermes uses a `BaseAdapter` interface with 8 implementations. All adapters normalize incoming messages to `MessageEvent` and route through the same dispatch pipeline.

**AGH relevance**: AGH already has HTTP/SSE and UDS interfaces. Adding messaging platform support should follow the adapter pattern.

**Recommendation**: Define a `PlatformAdapter` interface in core (`internal/api/contract`). Each platform is a separate extension package. The shared command registry pattern (commands registered once, available everywhere) is excellent and should be adopted.

### 4. Pluggable Execution Environments

Hermes separates "what command to run" from "where to run it" via `BaseEnvironment`. The terminal tool delegates to the active backend without knowing whether it is local, Docker, SSH, or serverless.

**AGH relevance**: AGH spawns ACP-compatible agents as subprocesses. The execution environment concept could extend this: agents could run in Docker, on remote machines, or in serverless environments.

**Recommendation**: Define an `ExecutionEnvironment` interface in core. The current local subprocess spawning becomes the default implementation. Docker, SSH, and serverless backends become extensions. This is lower priority than other patterns but valuable for Phase 3 (agent network protocol).

### 5. Approval / Security Pipeline

Hermes implements defense-in-depth: command normalization -> regex detection -> Tirith scanning -> approval callback -> execution backend isolation -> file write safety. The three-scope approval state machine (once/session/permanent) is particularly well-designed.

**AGH relevance**: AGH will need command safety as it supports more agent types. The layered approach is the right architecture.

**Recommendation**: Core should define: (a) a `CommandApproval` interface, (b) default regex patterns for dangerous commands, (c) a scope-based approval memory (once/session/permanent). The Tirith scanner and SSRF protection can be extensions or built-in.

## Extension System Insights

### Skills Pipeline Design

Hermes validates several skills design decisions that AGH should consider:

1. **User-message injection over system-prompt modification**: Skill content injected as a user message preserves prompt caching. System prompt stays stable; only the skill body pays fresh token cost. AGH should adopt this pattern.

2. **Frontmatter-indexed, body-injected**: Only the skill index (names + descriptions) goes into the system prompt. Full skill content is loaded on demand. This keeps system prompts compact.

3. **Platform-conditional skills**: Skills declare which platforms and tools they require via `conditions` in frontmatter. Unavailable skills are withheld. AGH should support skill conditions.

4. **Auto-proposal loop**: After complex tasks, the agent proposes new skills from the trajectory. This should be an opt-in extension, not forced behavior.

### Tool Registry Design

Key patterns from Hermes' tool registry for AGH:

1. **Single-file tool registration**: Each tool is a self-contained file that registers itself. In Go, this maps to an `init()` function or a registry-builder pattern. AGH should make adding a tool a one-package operation.

2. **check_fn for availability gating**: Tools that cannot run (missing API keys, missing deps) are withheld from the model's tool list. This is the single most important reliability property. AGH must implement this.

3. **Toolset composition with recursive includes**: Toolsets compose other toolsets. `resolve_toolset("full_stack")` recursively expands to all leaf tools. This is valuable for configuration ergonomics.

4. **Hidden tools**: Tools that exist in the registry for programmatic use but are not exposed to the model. Useful for internal orchestration tools.

5. **Tool output conventions**: Consistent `{"success": true, "data": {...}}` / `{"error": "..."}` shape. Models learn to parse and retry. AGH should standardize tool result format.

6. **MCP + user plugins as registry citizens**: External MCP servers and user plugin directories are discovered at startup and register into the same registry as built-in tools. AGH should treat MCP tools and user-authored tools as first-class registry entries.

### What AGH Should NOT Copy

1. **Module-level mutable globals** (`_last_resolved_tool_names`): A fragile pattern that causes bugs with subagents. AGH should thread resolved tool context through function parameters.

2. **Synchronous core with async bridging hacks** (`_run_async()`): Hermes' sync/async impedance mismatch creates complexity. AGH's Go concurrency model (goroutines + channels) avoids this entirely.

3. **Single-file session database shared across all interfaces**: AGH's split (global catalog + per-session event store) is architecturally cleaner. It avoids the write contention Hermes must hack around with jitter retries.

4. **Import-time side effects for registration**: In Go, prefer explicit registration in the composition root (`internal/daemon`) rather than relying on `init()` functions.

5. **Flat memory.md without consolidation**: AGH already has dream consolidation, which is superior to Hermes' approach of trusting the model to manually manage memory quality.

6. **Kitchen-sink monolith**: Hermes bundles 8 platform adapters, 6 terminal backends, voice/TTS, RL training, and batch processing into one package. AGH should keep these as extensions.

## Summary: Core vs Extension Classification

### Core (what AGH should build into its minimal robust foundation)

- Tool registry with availability gating, toolset composition, and standardized result format
- FTS5 cross-session search in the session/event store
- Context compression interface with default LLM-summarization implementation
- Prompt caching hints in transcript assembly
- Structured system prompt builder with ordered stable sections
- Command approval interface with scope-based memory (once/session/permanent)
- Provider resolution interface for LLM credentials
- Token accounting and cost estimation in the observe layer
- Subagent delegation with isolated contexts and restricted tools
- MCP tool discovery and proxy
- Unified command dispatch table across interfaces
- Skills: frontmatter-indexed, user-message injected, platform-conditional

### Extension (what AGH should support via its plugin system)

- Gateway platform adapters (Telegram, Discord, Slack, etc.)
- Cron/scheduled automations
- Terminal execution backends (Docker, SSH, Modal, etc.)
- Voice/TTS pipeline
- Batch processing and trajectory generation
- RL training environments
- Honcho user modeling
- Diagnostic commands (doctor, status, insights)
- Skill auto-proposal heuristics
- DM pairing and gateway authorization
- Process management (background process registry)
- Persistent shell state
- Streaming response delivery optimization
