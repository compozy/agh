# AI Agent Extension Ecosystem Research

## Overview

This document captures research into the AI agent extension ecosystem as of April 2026, with a focus on concrete extension ideas that could be adapted for AGH's three-dimensional extension model (Resources, Capabilities, Actions). The research covers OpenFang (a Rust-based agent OS), the MCP server ecosystem, extension patterns from major AI coding tools, emerging protocols (A2A), agent memory systems, workflow orchestration, permission/sandbox patterns, and developer feature requests.

---

## 1. OpenFang: The Closest Comparable System

OpenFang is an open-source agent operating system built in Rust -- the closest architectural analog to AGH. It compiles to a single ~32MB binary (137K LOC, 14 Rust crates) and runs agents as background daemons.

### 1.1 Built-in Tools (53 tools in openfang-runtime)

OpenFang ships 53 tools in its `openfang-runtime` crate, spanning several categories:

| Category       | Tools                                         | Description                                                                      |
| -------------- | --------------------------------------------- | -------------------------------------------------------------------------------- |
| Web            | web_search, browser_automation, web_fetch     | Search engines, headless browser control, URL fetching                           |
| File           | file_read, file_write, file_list, file_delete | Workspace-confined file operations with path traversal prevention                |
| Code/Process   | process_start, code_execute                   | Subprocess spawning with allowlist validation, env-clearing, timeout enforcement |
| Media          | image_generation, tts (text-to-speech)        | Image creation via AI models, voice synthesis                                    |
| Data           | knowledge_graph, data_analyze                 | Graph-based knowledge storage, structured data analysis                          |
| Infrastructure | docker_run, docker_build                      | Container management for isolated execution                                      |
| Communication  | email_send, notification_push                 | Outbound messaging capabilities                                                  |

All tool code runs inside a WASM sandbox with dual metering (fuel + epoch interruption). File operations are workspace-confined. Subprocesses are env-cleared and timeout-enforced.

### 1.2 Hands System (7 Bundled Agent Packages)

"Hands" are OpenFang's core innovation -- self-contained autonomous capability packages that combine configuration, expert knowledge, operational procedures, and tool access into a single deployable unit.

Each Hand bundles:

- `HAND.toml` manifest
- System prompt with multi-phase operational playbook
- `SKILL.md` expert knowledge
- Configurable settings
- Dashboard metrics

| Hand       | Domain        | What It Does                                                                             |
| ---------- | ------------- | ---------------------------------------------------------------------------------------- |
| Clip       | Content       | Transforms long-form video into short clips with captions, thumbnails, voice-overs       |
| Lead       | Sales         | Discovers, enriches, scores, deduplicates qualified leads on schedule with ICP profiling |
| Collector  | Intelligence  | Monitors targets and gathers competitive intelligence                                    |
| Predictor  | Forecasting   | Makes predictions with Brier score tracking for calibration                              |
| Researcher | Productivity  | Cross-references sources, fact-checks (CRAAP evaluation), generates cited reports        |
| Twitter    | Communication | Manages X/Twitter accounts autonomously                                                  |
| Browser    | Automation    | Web automation for scraping, form-filling, and interaction                               |

**AGH mapping**: Hands map directly to AGH's Resources (agents + skills bundled together). AGH could implement a similar concept where an "agent package" bundles an agent definition, skills, hooks, and MCP servers into a single deployable unit.

### 1.3 Channel Adapters (40 Adapters)

OpenFang connects to 40 messaging platforms: Telegram, Discord, Slack, WhatsApp, Signal, Matrix, Teams, Google Chat, Feishu, DingTalk, Mastodon, Bluesky, LinkedIn, Reddit, IRC, WebChat, and 24+ more.

Each adapter supports per-channel model overrides, DM/group policies, rate limiting, and output formatting. New adapters implement the `ChannelAdapter` trait.

**AGH mapping**: These map to AGH's Capabilities dimension. A `channel.adapter` capability type would allow AGH to expose agent sessions across messaging platforms. The adapter pattern (trait/interface implementation) aligns with AGH's interface-based extension model.

### 1.4 Skills System (60 Bundled Skills)

OpenFang ships 60 bundled skills compiled into the binary, using the `SKILL.md` format (YAML frontmatter + Markdown body). Categories include CI/CD, Ansible, Prometheus, Nginx, Kubernetes, Terraform, Helm, Docker, sysadmin, shell-scripting, Linux networking.

Three skill types exist:

1. **Prompt-only skills (SKILL.md)** -- inject expert domain knowledge into system prompt
2. **Python skills** -- run as subprocesses, communicate via JSON over stdin/stdout
3. **Rust/WASM skills** -- compiled to WASM, run in sandboxed environment with fuel metering

Each skill has a `skill.toml` manifest with metadata, runtime config, tool declarations, and capability requirements.

**AGH mapping**: AGH already has a skills system. OpenFang's `skill.toml` manifest pattern (declaring required capabilities like `NetConnect`) is worth adopting. The three-tier skill type system (prompt-only, subprocess, WASM) is a good model for AGH's skill extensibility.

---

## 2. MCP Server Ecosystem

As of March 2026, there are 5,000+ community MCP servers, with 440 curated in the best-of-mcp-servers list (930K total GitHub stars across 34 categories).

### 2.1 Most Popular MCP Servers by Category

| Category          | Server              | Stars/Installs                       | What It Does                                                    | AGH Mapping                                     |
| ----------------- | ------------------- | ------------------------------------ | --------------------------------------------------------------- | ----------------------------------------------- |
| **Documentation** | Context7            | 11K views, 690 installs (FastMCP #1) | Injects fresh, version-specific docs into prompts               | Resource (MCP) + Capability (prompt.provider)   |
| **Browser**       | Playwright MCP      | 30K stars, ~6K views                 | Structured browser automation via accessibility snapshots       | Resource (MCP) + Capability (agent.driver tool) |
| **Git/GitHub**    | GitHub MCP          | Most-starred MCP server              | PR management, issue triaging, code review automation           | Resource (MCP)                                  |
| **Database**      | PostgreSQL MCP      | High adoption                        | Natural language to SQL, schema introspection                   | Resource (MCP)                                  |
| **Database**      | Supabase MCP        | Growing                              | Postgres + edge functions + schema management                   | Resource (MCP)                                  |
| **Filesystem**    | Filesystem MCP      | Official reference                   | Secure file read/write/search within allowed directories        | Resource (MCP)                                  |
| **Memory**        | Memory MCP          | Official reference                   | Persistent knowledge graph across sessions                      | Resource (MCP) + Capability (memory.backend)    |
| **Reasoning**     | Sequential Thinking | Popular                              | Structured step-by-step reasoning                               | Capability (message.transform)                  |
| **Search**        | Firecrawl MCP       | Growing                              | Web scraping with JS rendering, anti-bot, clean markdown output | Resource (MCP)                                  |
| **Cloud**         | AWS MCP             | 8.7K stars                           | Integration with AWS services and resources                     | Resource (MCP)                                  |
| **Automation**    | Zapier MCP          | Growing                              | Connects to thousands of apps via Zapier workflows              | Resource (MCP)                                  |
| **Automation**    | Pipedream MCP       | Growing                              | 2,500 APIs, 8,000+ prebuilt tools                               | Resource (MCP)                                  |
| **Cloud**         | Cloudflare MCP      | Growing                              | Workers/KV/R2/D1 management                                     | Resource (MCP)                                  |
| **Data**          | MindsDB MCP         | 39K stars                            | Unified data platform across databases                          | Resource (MCP)                                  |
| **Search/RAG**    | Pinecone MCP        | Growing                              | Vector similarity search for RAG                                | Resource (MCP) + Capability (memory.backend)    |

### 2.2 MCP Apps (January 2026)

Anthropic launched MCP Apps -- interactive UIs that render dashboards, forms, and charts directly inside Claude. Launch partners: Amplitude, Asana, Box, Clay, Hex, Salesforce.

**AGH mapping**: AGH could support MCP Apps as a UI extension point, where MCP servers can provide rendered components in the web UI.

### 2.3 Recommended Starting Stack for Developers

1. Context7 (documentation injection)
2. Playwright (browser automation)
3. GitHub (PR/issue management)
4. PostgreSQL or Supabase (database)
5. Memory (persistent knowledge graph)

**AGH mapping**: AGH should ship with built-in MCP server support and potentially bundle or recommend these servers as defaults for developer-focused use cases.

---

## 3. AI Coding Agent Extension Patterns

### 3.1 Extension Architectures Across Tools

| Tool             | Extension Mechanism                             | Key Pattern                                                                               | AGH Relevance                                                            |
| ---------------- | ----------------------------------------------- | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| **Claude Code**  | Skills (SKILL.md), hooks, custom slash commands | Markdown-based skills with YAML frontmatter; 12 lifecycle hook events; project/user scope | Directly applicable -- AGH already uses similar patterns                 |
| **Cursor**       | Rules files, MCP, Composer                      | `.cursor/rules` for project context; multi-agent Composer (8 parallel agents)             | Rules files map to AGH config; parallel agents map to session management |
| **Cline**        | MCP Marketplace, subagents, CLI 2.0             | Client-side architecture; 5M+ installs; dedicated MCP marketplace for discovery           | MCP marketplace concept; subagent spawning                               |
| **Roo Code**     | Custom Modes, Mode Gallery                      | Specialized AI personas with scoped tool permissions per mode                             | Maps to AGH agent definitions with capability restrictions               |
| **Continue.dev** | `.continuerc.json`, local indexing, @docs       | Lancet protocol for local vector indexing; semantic codebase search                       | Maps to AGH memory.backend + prompt.provider                             |
| **Aider**        | Git-native, BYOM                                | Terminal-first; git-aware diffs; repository map                                           | Maps to AGH's CLI-first approach                                         |
| **Goose**        | MCP extensions, recipes, custom distributions   | YAML recipe workflows; Extension Manager UI; custom distros                               | Recipes map to AGH skills; distros map to agent configs                  |
| **Windsurf**     | Cascade, rules                                  | Cascade learns project patterns; greenfield-optimized                                     | Pattern learning maps to AGH memory system                               |

### 3.2 Claude Code's Extension Model (Most Relevant to AGH)

Claude Code's model is the most directly relevant since AGH manages Claude Code as a subprocess:

**Skills**: Markdown files in `.claude/skills/<name>/SKILL.md` with supporting scripts, templates, examples. Two invocation controls:

- `disable-model-invocation: true` -- only human can invoke (for side-effect workflows)
- `user-invocable: false` -- only model can invoke (background knowledge)

**Hooks (12 lifecycle events)**:

- PreToolUse, PostToolUse, PostToolUseFailure
- SessionStart, SessionEnd, Stop
- SubagentStart, SubagentStop
- UserPromptSubmit, Notification
- PreCompact, PermissionRequest

"Hooks guarantee behavior; prompts suggest it." This is a critical design principle.

**AGH mapping**: AGH's hooks system should mirror these 12 events. The separation between "hooks guarantee" and "prompts suggest" maps perfectly to AGH's distinction between deterministic hooks and AI-driven skills.

### 3.3 Goose's Extension Model

Goose (29K+ stars, Apache 2.0, now under Linux Foundation's AAIF) provides:

- **MCP-native extensions**: Any MCP server becomes a Goose extension automatically
- **Recipes**: Reusable YAML workflow definitions packaging goals, required extensions, structured inputs, and sub-recipes
- **Custom Distributions**: Preconfigured provider + extension + branding bundles
- **Extension Manager UI**: Desktop app for browsing, toggling, configuring extensions
- **ACP integration**: Connects to VS Code, Cursor, Windsurf, JetBrains via Agent Client Protocol

**AGH mapping**: Goose's recipe system maps to AGH's skills. Custom distributions map to workspace-level agent configurations. The Extension Manager UI concept could inform AGH's web UI design.

### 3.4 Roo Code's Custom Modes

Roo Code's differentiating feature is Custom Modes -- specialized AI personas with:

- Tailored system instructions per mode
- Scoped tool permissions (e.g., security reviewer can read but not write)
- Community Mode Gallery for sharing configurations
- 5 built-in modes: Code, Architect, Ask, Debug, Custom

**AGH mapping**: This maps to AGH agent definitions with per-agent capability restrictions. AGH could implement a "mode" concept as a lightweight agent configuration overlay.

---

## 4. Emerging Protocols and Patterns

### 4.1 A2A (Agent-to-Agent Protocol)

Google's A2A protocol (April 2025, now v0.3 as of July 2025) enables communication between opaque agent systems. Now under the Linux Foundation with 150+ supporting organizations.

**Core concepts**:

- **Agent Cards**: JSON manifests at `/.well-known/agent.json` listing name, endpoint, skills, auth
- **Task lifecycle**: pending -> in-progress -> completed/failed, with SSE streaming
- **Transport**: HTTP, SSE, JSON-RPC (v0.3 adds gRPC)
- **Complementary to MCP**: MCP = agent-to-tool, A2A = agent-to-agent

**AGH mapping**: A2A maps directly to AGH's agent network protocol (Phase 3). AGH sessions could publish Agent Cards, accept tasks from external agents via A2A, and delegate subtasks to remote agents. This is a natural fit for AGH's HTTP/SSE API.

| A2A Concept          | AGH Mapping                                          |
| -------------------- | ---------------------------------------------------- |
| Agent Card           | Agent definition + session metadata exposed via HTTP |
| Task submission      | New session creation or message to existing session  |
| Task streaming       | SSE event stream (already implemented)               |
| Capability discovery | Agent catalog + skills listing                       |

### 4.2 Agent Memory Systems

The memory landscape in 2026 spans three architectural categories:

| Category                      | Examples                        | Description                                | AGH Mapping                 |
| ----------------------------- | ------------------------------- | ------------------------------------------ | --------------------------- |
| Extended attention            | Infini-attention, recursive LMs | Scale the context window itself            | Out of scope (model-level)  |
| Memory-augmented transformers | Hybrid models                   | Learned memory modules in model            | Out of scope (model-level)  |
| External persistent memory    | Mem0, Letta, LangChain Memory   | Store/retrieve/manage memory outside model | Capability (memory.backend) |

**Key frameworks**:

| Framework            | Approach                        | Key Feature                                                                                        | AGH Relevance                           |
| -------------------- | ------------------------------- | -------------------------------------------------------------------------------------------------- | --------------------------------------- |
| **Mem0**             | Dedicated memory layer          | Vector memory (semantic similarity) + graph memory (relationships)                                 | High -- memory.backend implementation   |
| **Letta**            | Long-context agent architecture | Core memory blocks (persistent labeled context), archival memory (DB-backed), memory editing tools | High -- maps to AGH's dual-scope memory |
| **LangChain Memory** | Modular memory types            | Conversation buffer, summary, entity, knowledge graph                                              | Medium -- patterns for memory.backend   |
| **ReMe**             | Open-source memory kit          | Multiple vector store backends, "remember me, refine me"                                           | Medium -- reference implementation      |

**Advanced patterns emerging in 2026**:

- **Conflict resolution**: When user preferences change, compress old memory into temporal reflection summaries rather than deleting
- **Multi-agent shared memory**: Strict access controls to prevent race conditions and cross-agent contamination
- **Graph memory in production**: For complex entity relationships (medical, enterprise hierarchies, technical systems)
- **Memory cost**: 1M-token context window costs ~15x more per turn than equivalent persistent memory retrieval

**AGH mapping**: AGH's existing dual-scope memory (global + workspace) with dream consolidation is well-positioned. Extensions should add:

- Vector-backed memory.backend (semantic search)
- Graph-backed memory.backend (relationship tracking)
- Memory conflict resolution (temporal reflection summaries)
- Cross-session memory sharing with access controls

### 4.3 Workflow Orchestration Patterns

Five dominant patterns have emerged:

| Pattern                  | Description                                          | Use Case                               | AGH Mapping                                   |
| ------------------------ | ---------------------------------------------------- | -------------------------------------- | --------------------------------------------- |
| Sequential pipeline      | Step-by-step, each stage builds on previous          | Progressive refinement tasks           | Session chaining via hooks                    |
| Hierarchical multi-agent | Manager-subordinate delegation                       | Complex multi-department tasks         | Session spawning + parent-child relationships |
| Decentralized swarm      | Peer agents collaborate without central control      | Resilient, flexible problem-solving    | A2A-connected sessions                        |
| Group chat               | Shared conversation thread, chat manager facilitates | Consensus-building (limit to 3 agents) | Multi-agent session with turn management      |
| DAG-based                | Directed acyclic graphs define task dependencies     | Complex pipelines with parallel steps  | Workflow engine as new capability             |

**AGH mapping**: AGH's session model could be extended with a `workflow.engine` capability that supports DAG-based task orchestration across sessions. Parent sessions could spawn child sessions with defined dependencies.

---

## 5. Permission, Sandbox, and Human-in-the-Loop Patterns

### 5.1 Three Levels of Human Oversight

| Level                 | Description                                               | When to Use                       |
| --------------------- | --------------------------------------------------------- | --------------------------------- |
| Human-out-of-the-loop | Agent acts fully autonomously                             | Low-risk, well-defined tasks      |
| Human-in-the-loop     | Agent pauses for approval on specific actions             | High-risk or destructive actions  |
| Human-on-the-loop     | Supervisor monitors overall flow, intervenes on anomalies | Medium-risk continuous operations |

### 5.2 Permission Patterns

| Pattern                        | Description                                                                      | AGH Mapping                                    |
| ------------------------------ | -------------------------------------------------------------------------------- | ---------------------------------------------- |
| Per-tool permission policies   | Read vs. write access per tool                                                   | Capability (permission.gate)                   |
| Environment-scoped permissions | Allow destructive ops in staging only                                            | Config-level permission rules                  |
| Approval vs. suspension        | Gatekeeping (yes/no) vs. clarification (need more info)                          | Action (Host API) with two response types      |
| Planning/execution separation  | Planner proposes under broad permissions, executor acts under strict permissions | Two-phase session with different agent configs |
| Tool trust spectrum            | Classify tools from harmless (search) to destructive (delete)                    | permission.gate with risk-level classification |
| Centralized governance UI      | Dashboard for managing who/what/where/when                                       | Web UI extension                               |

### 5.3 Sandbox Strategies

| Strategy                | Description                             | AGH Mapping                                |
| ----------------------- | --------------------------------------- | ------------------------------------------ |
| WASM sandbox            | Dual-metered execution (fuel + epoch)   | Capability (could wrap tool execution)     |
| MicroVMs                | Firecracker/gVisor for full isolation   | Heavy-weight, for untrusted code           |
| Short-lived credentials | Temporary tokens scoped per task        | Hook (PreToolUse) for credential injection |
| Zero-trust networking   | All connections explicitly allowed      | Config-level network policies              |
| Workspace confinement   | File operations restricted to workspace | Already in AGH's workspace model           |

**AGH mapping**: AGH's permission.gate capability should implement the tool trust spectrum. The planning/execution separation pattern maps to AGH's ability to configure different agent definitions for different phases of a workflow.

---

## 6. Observability and Tracing

### 6.1 OpenTelemetry as the Standard

OpenTelemetry has emerged as the universal standard for AI agent observability. Key developments:

| Project                   | What It Does                                                   | AGH Mapping                             |
| ------------------------- | -------------------------------------------------------------- | --------------------------------------- |
| Traceloop OTel MCP Server | AI agents query distributed traces for automated debugging     | Resource (MCP)                          |
| FastMCP native OTel       | Zero-config tracing for tool/prompt/resource operations        | Capability (observe.exporter)           |
| AG2 OTel Tracing          | Structured hierarchical traces with GenAI semantic conventions | Capability (observe.exporter) reference |
| Grafana Cloud + OpenLIT   | Pre-built dashboards for MCP observability                     | Reference architecture                  |

**Key metrics to track**: Per-tool latency, error rates, call volume anomalies, end-to-end traces connecting agent reasoning to tool execution.

**Proposed MCP protocol change**: Add standardized OTel trace spans directly into MCP protocol, with trace context propagation via HTTP headers (SSE/Streamable HTTP) or explicit parameters (stdio).

**AGH mapping**: AGH's observe.exporter capability should export OpenTelemetry-compatible traces. The GenAI semantic conventions (model name, provider, token usage, cost, temperature, tool call arguments/results) should be adopted for AGH's event recording.

---

## 7. What Developers Most Want

Based on GitHub issues, Reddit discussions, and developer surveys:

### 7.1 Top Feature Requests

| Request                            | Frequency   | Description                                                         | AGH Mapping                                         |
| ---------------------------------- | ----------- | ------------------------------------------------------------------- | --------------------------------------------------- |
| **Better large codebase handling** | Very high   | Index whole repos, semantic search across files                     | Capability (prompt.provider) with codebase indexing |
| **Issue-to-PR automation**         | High        | Assign GitHub issue, agent implements + tests + deploys             | Action (Host API) + workflow orchestration          |
| **Multi-file agentic workflows**   | High        | Parallel agents working on different codebase areas                 | Session management with concurrent agents           |
| **Bring Your Own Model (BYOM)**    | High        | Connect any LLM provider via API keys                               | Capability (agent.driver) with provider abstraction |
| **Fine-grained permissions**       | High        | Approval gates before destructive actions, per-task autonomy levels | Capability (permission.gate)                        |
| **MCP tool discovery**             | Medium-high | Browse, install, configure MCP servers easily                       | Resource (MCP) with registry/marketplace            |
| **Reusable workflows/recipes**     | Medium      | Save and share task automation patterns                             | Resource (skills) with workflow support             |
| **Cost tracking and budgets**      | Medium      | Token usage monitoring, per-session cost limits                     | Capability (observe.exporter) + config              |
| **Audit trails**                   | Medium      | Complete record of every agent action for compliance                | Already in AGH's observe system                     |
| **Local/offline model support**    | Medium      | Run with Ollama, Docker Model Runner                                | Capability (agent.driver)                           |
| **Custom agent personas**          | Medium      | Different "modes" for different tasks (code, review, plan)          | Resource (agents) with mode overlays                |
| **CI/CD integration**              | Medium      | Agents triggered by CI events, results fed back                     | Hook + Action (Host API)                            |

### 7.2 Anti-Patterns to Avoid

- Using AI for architecture decisions (better for implementation)
- Infinite agent loops without cost/iteration limits
- Agents that rewrite entire files instead of surgical diffs
- Hardcoded model dependencies (vendor lock-in)
- Trust-all-tools security model

---

## 8. Consolidated Extension Ideas for AGH

### 8.1 High-Priority Extensions (Strong ecosystem demand, clear AGH mapping)

| Extension                            | Type       | Dimension                | Description                                                                 |
| ------------------------------------ | ---------- | ------------------------ | --------------------------------------------------------------------------- |
| **OTel Observe Exporter**            | Capability | observe.exporter         | Export AGH events as OpenTelemetry traces with GenAI semantic conventions   |
| **Vector Memory Backend**            | Capability | memory.backend           | Semantic similarity search over agent memory using embeddings               |
| **Graph Memory Backend**             | Capability | memory.backend           | Relationship-aware memory using knowledge graphs                            |
| **A2A Protocol Gateway**             | Capability | agent.driver (extension) | Accept/delegate tasks via Google's Agent-to-Agent protocol                  |
| **Permission Gate: Risk Classifier** | Capability | permission.gate          | Classify tool calls by risk level, require approval for destructive actions |
| **Codebase Indexer**                 | Capability | prompt.provider          | Index workspace files for semantic search, inject relevant context          |
| **GitHub MCP Bundle**                | Resource   | MCP                      | Pre-configured GitHub MCP server for PR/issue/code management               |
| **Workflow Engine**                  | Capability | (new)                    | DAG-based task orchestration across sessions                                |
| **Agent Package (Hand-style)**       | Resource   | agents + skills + hooks  | Bundled autonomous capability packages                                      |
| **Channel Adapter Framework**        | Capability | (new)                    | Expose sessions via messaging platforms (Slack, Discord, Telegram)          |

### 8.2 Medium-Priority Extensions (Growing demand, useful differentiation)

| Extension                             | Type       | Dimension         | Description                                                             |
| ------------------------------------- | ---------- | ----------------- | ----------------------------------------------------------------------- |
| **Cost/Budget Tracker**               | Capability | observe.exporter  | Track token usage, enforce per-session cost limits                      |
| **Content Validator: PII**            | Capability | content.validate  | Detect and mask personally identifiable information                     |
| **Content Validator: Secret Scanner** | Capability | content.validate  | Prevent secrets/credentials from leaking into agent context             |
| **Prompt Injection Scanner**          | Capability | content.validate  | Detect prompt injection attempts in skill/tool inputs                   |
| **Custom Distribution Builder**       | Action     | Host API          | Package agent configs + skills + MCP into shareable bundles             |
| **MCP Server Registry**               | Action     | Host API          | Browse, install, configure MCP servers from a catalog                   |
| **Webhook/Event Bridge**              | Resource   | hooks             | Trigger sessions from external events (CI/CD, webhooks, cron)           |
| **Planning/Execution Splitter**       | Capability | message.transform | Separate planning phase (broad tools) from execution (restricted tools) |

### 8.3 Lower-Priority / Exploratory Extensions

| Extension                    | Type       | Dimension           | Description                                                 |
| ---------------------------- | ---------- | ------------------- | ----------------------------------------------------------- |
| **Multi-Agent Group Chat**   | Action     | Host API            | Multiple agents in shared conversation with turn management |
| **Agent Card Publisher**     | Action     | Host API            | Publish `.well-known/agent.json` for A2A discovery          |
| **Mode Gallery**             | Resource   | skills              | Community marketplace for agent mode/persona configurations |
| **Recipe/Workflow YAML**     | Resource   | skills              | Goose-style reusable workflow definitions                   |
| **Memory Conflict Resolver** | Capability | memory.backend      | Temporal reflection summaries when knowledge changes        |
| **Browser Automation Tool**  | Capability | agent.driver (tool) | Playwright-based browser control for agents                 |
| **Local Model Provider**     | Capability | agent.driver        | Connect to Ollama/local models as agent backends            |

---

## 9. Key Takeaways

### 9.1 The ecosystem is converging on three standards

1. **MCP** (Model Context Protocol) for agent-to-tool communication -- 5,000+ servers, universal adoption
2. **A2A** (Agent-to-Agent) for agent-to-agent communication -- 150+ organizations, Linux Foundation governance
3. **OpenTelemetry** for agent observability -- emerging as the universal tracing standard for AI agents

AGH should support all three natively.

### 9.2 Skills/extensions are becoming the primary differentiator

Every major tool (Claude Code, Goose, OpenFang, Cline, Roo Code) has a skills/extension system. The winning pattern is:

- **Markdown-based** skill definitions (low barrier to authorship)
- **YAML manifests** for metadata and capability declarations
- **Three tiers**: prompt-only (cheapest), subprocess (flexible), sandboxed (secure)
- **Community marketplace** for discovery and sharing

AGH's existing skills system is well-aligned. Priority: add manifest-based capability declarations and a registry.

### 9.3 Memory is the next competitive frontier

Persistent memory across sessions is transitioning from experimental to production-critical. The key patterns are:

- **Dual-scope** (global + workspace) -- AGH already has this
- **Vector + graph** hybrid -- AGH should add both backends
- **Dream/consolidation** -- AGH already has this (rare advantage)
- **Cost optimization** -- persistent memory retrieval is 15x cheaper than large context windows

### 9.4 Permission and safety are table stakes

Every production agent system implements:

- Per-tool, per-action permission policies
- Planning/execution separation
- Human-in-the-loop for destructive actions
- Audit trails
- Sandbox isolation

AGH's permission.gate capability should implement risk-based classification with configurable approval thresholds.

### 9.5 Hooks/middleware are the deterministic control layer

The universal pattern across all frameworks: "Hooks guarantee behavior; prompts suggest it." Claude Code's 12 hook events (PreToolUse, PostToolUse, SessionStart, etc.) represent the industry standard. AGH's hook system should match or exceed this coverage.

### 9.6 Workflow orchestration is emerging but not yet standardized

DAG-based workflows, hierarchical agent delegation, and pipeline patterns are common but each framework implements them differently. AGH has an opportunity to provide a clean, Go-native workflow engine that leverages its session model.

### 9.7 Channel adapters are a differentiator for non-IDE use cases

OpenFang's 40 channel adapters demonstrate demand for agent access beyond CLI/IDE. Slack, Discord, and Telegram are the highest-demand channels. AGH could start with 3-5 high-value adapters.

---

## Sources

- [OpenFang -- The Agent Operating System](https://www.openfang.sh/)
- [OpenFang GitHub](https://github.com/RightNow-AI/openfang)
- [OpenFang Skill Development Docs](https://www.openfang.sh/docs/skill-development)
- [OpenFang Channel Adapters Docs](https://www.openfang.sh/docs/channel-adapters)
- [awesome-mcp-servers (GitHub)](https://github.com/wong2/awesome-mcp-servers)
- [best-of-mcp-servers (GitHub)](https://github.com/tolkonepiu/best-of-mcp-servers)
- [MCP Awesome Directory (1200+ servers)](https://mcp-awesome.com/)
- [Top 10 Most Popular MCP Servers -- FastMCP](https://fastmcp.me/blog/top-10-most-popular-mcp-servers)
- [Top 15 MCP Servers -- DEV Community](https://dev.to/jangwook_kim_e31e7291ad98/top-15-mcp-servers-every-developer-should-install-in-2026-n1h)
- [Agent2Agent Protocol (A2A) -- Google Blog](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)
- [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
- [A2A GitHub](https://github.com/a2aproject/A2A)
- [A2A Protocol Upgrade -- Google Cloud Blog](https://cloud.google.com/blog/products/ai-machine-learning/agent2agent-protocol-is-getting-an-upgrade)
- [Linux Foundation A2A Project](https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents)
- [AI Agent Memory Frameworks 2026 -- MachineLearningMastery](https://machinelearningmastery.com/the-6-best-ai-agent-memory-frameworks-you-should-try-in-2026/)
- [Memory for AI Agents -- The New Stack](https://thenewstack.io/memory-for-ai-agents-a-new-paradigm-of-context-engineering/)
- [State of AI Agent Memory 2026 -- Mem0](https://mem0.ai/blog/state-of-ai-agent-memory-2026)
- [Architecture of Memory Systems in AI Agents -- Analytics Vidhya](https://www.analyticsvidhya.com/blog/2026/04/memory-systems-in-ai-agents/)
- [Goose AI Agent -- GitHub](https://github.com/block/goose)
- [Goose Documentation](https://goose-docs.ai/)
- [Goose AI Review 2026](https://aitoolanalysis.com/goose-ai-review/)
- [Cline vs Roo Code vs Continue 2026 -- DevToolReviews](https://www.devtoolreviews.com/reviews/cline-vs-roo-code-vs-continue)
- [Roo Code GitHub](https://github.com/RooCodeInc/Roo-Code)
- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
- [Claude Code Hooks -- Dotzlaw Consulting](https://www.dotzlaw.com/insights/claude-hooks/)
- [Claude Agent SDK Hooks Lifecycle](https://pkg.go.dev/github.com/dotcommander/agent-sdk-go/examples/hooks-lifecycle)
- [OpenTelemetry MCP Server -- Traceloop](https://github.com/traceloop/opentelemetry-mcp-server)
- [MCP Observability with OTel -- SigNoz](https://signoz.io/blog/mcp-observability-with-otel/)
- [Distributed Tracing for Agentic Workflows -- Red Hat](https://developers.redhat.com/articles/2026/04/06/distributed-tracing-agentic-workflows-opentelemetry)
- [How to Sandbox AI Agents 2026 -- Northflank](https://northflank.com/blog/how-to-sandbox-ai-agents)
- [Human-in-the-Loop for AI Agents -- Permit.io](https://www.permit.io/blog/human-in-the-loop-for-ai-agents-best-practices-frameworks-use-cases-and-demo)
- [AI Agent Security Guide 2026 -- MintMCP](https://www.mintmcp.com/blog/ai-agent-security)
- [2026 Guide to Agentic Workflow Architectures -- StackAI](https://www.stackai.com/blog/the-2026-guide-to-agentic-workflow-architectures)
- [Best AI Coding Agents 2026 -- Faros](https://www.faros.ai/blog/best-ai-coding-agents-2026)
- [Best AI for Coding Reddit 2026](https://www.aitooldiscovery.com/guides/best-ai-for-coding-reddit)
- [10 Things Developers Want from Agentic IDEs -- RedMonk](https://redmonk.com/kholterhoff/2025/12/22/10-things-developers-want-from-their-agentic-ides-in-2025/)
