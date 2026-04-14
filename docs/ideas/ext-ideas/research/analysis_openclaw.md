# OpenClaw Extension Research - Analysis for AGH

## Overview

OpenClaw (formerly Clawdbot/Moltbot) is a TypeScript/Node.js open-source personal AI assistant framework created by Peter Steinberger in November 2025. It runs a local Gateway process that connects to LLM providers, exposes the agent across 25+ messaging channels, and has grown to 335,000+ GitHub stars. Its extension model consists of four axes: **Channels** (messaging platform adapters), **Providers** (LLM backends), **Tools** (agent capabilities), and **Skills** (markdown instruction packs distributed via ClawHub). As of April 2026, ClawHub hosts 13,700+ community skills and 3,200+ vetted entries.

OpenClaw's architecture is relevant to AGH because both share a local-first, single-binary (single-process for OpenClaw) gateway design with session management, tool execution, and observability. However, OpenClaw is TypeScript/Node.js while AGH is Go, and OpenClaw uses MCP as its primary tool protocol whereas AGH uses ACP.

---

## Extensions Catalog

### Channels (Messaging Platform Adapters)

| Extension              | Category | Description                                                   | AGH Mapping                                      |
| ---------------------- | -------- | ------------------------------------------------------------- | ------------------------------------------------ |
| WhatsApp (Baileys)     | Channel  | Bidirectional WhatsApp messaging via Baileys library          | Resource: channel adapter                        |
| Telegram (grammY)      | Channel  | Telegram bot integration, most popular after web browsing     | Resource: channel adapter                        |
| Slack (Bolt)           | Channel  | Workspace messaging with Bolt SDK, approval buttons           | Resource: channel adapter                        |
| Discord (discord.js)   | Channel  | Rich interaction support with buttons, slash commands         | Resource: channel adapter                        |
| Google Chat            | Channel  | Google Workspace chat integration via Chat API                | Resource: channel adapter                        |
| Signal (signal-cli)    | Channel  | Privacy-focused Signal messenger support                      | Resource: channel adapter                        |
| iMessage (BlueBubbles) | Channel  | macOS-exclusive iMessage bridge                               | Resource: channel adapter                        |
| Microsoft Teams        | Channel  | Plugin-only as of 2026.1.15 (`@openclaw/msteams`)             | Resource: channel adapter                        |
| Matrix                 | Channel  | Decentralized Matrix protocol support                         | Resource: channel adapter                        |
| IRC                    | Channel  | Legacy IRC channel support                                    | Resource: channel adapter                        |
| WebChat                | Channel  | Built-in web UI channel                                       | Resource: channel adapter (maps to AGH HTTP/SSE) |
| LINE                   | Channel  | LINE messenger (Asia-Pacific markets)                         | Resource: channel adapter                        |
| Mattermost             | Channel  | Self-hosted team chat                                         | Resource: channel adapter                        |
| Feishu                 | Channel  | Lark/Feishu for China market                                  | Resource: channel adapter                        |
| WeChat                 | Channel  | WeChat via Tencent plugin (`@tencent-weixin/openclaw-weixin`) | Resource: channel adapter                        |
| Nostr                  | Channel  | Decentralized social protocol                                 | Resource: channel adapter                        |
| Twitch                 | Channel  | Twitch chat integration for streaming                         | Resource: channel adapter                        |
| Nextcloud Talk         | Channel  | Self-hosted Nextcloud chat                                    | Resource: channel adapter                        |

### LLM Providers

| Extension               | Category | Description                                   | AGH Mapping              |
| ----------------------- | -------- | --------------------------------------------- | ------------------------ |
| Anthropic (Claude)      | Provider | Claude Sonnet 4 recommended as primary        | Capability: agent.driver |
| OpenAI (GPT)            | Provider | GPT-5.4 with forward-compat fallback          | Capability: agent.driver |
| Google (Gemini)         | Provider | Gemini API + Vertex AI support                | Capability: agent.driver |
| Ollama (Local)          | Provider | Auto-detected local models at 127.0.0.1:11434 | Capability: agent.driver |
| OpenRouter              | Provider | Aggregation platform for multiple providers   | Capability: agent.driver |
| Together AI             | Provider | Aggregation/inference provider                | Capability: agent.driver |
| Qwen (via OAuth plugin) | Provider | Alibaba Qwen models                           | Capability: agent.driver |
| Copilot Proxy           | Provider | Microsoft Copilot model proxy                 | Capability: agent.driver |

### Built-in Tools

| Extension       | Category | Description                                                              | AGH Mapping                  |
| --------------- | -------- | ------------------------------------------------------------------------ | ---------------------------- |
| Browser         | Tool     | Dedicated Chromium instance with snapshots, actions, form fill, scraping | Action: session tool         |
| Canvas (A2UI)   | Tool     | Agent-driven visual workspace with live push/reset/eval/snapshot         | Action: session tool         |
| Cron            | Tool     | Schedule agent actions at specific times (e.g., daily summary at 9 AM)   | Resource: hook (time-based)  |
| Webhooks        | Tool     | External trigger points for agent actions (e.g., Gmail push)             | Resource: hook (event-based) |
| Sessions        | Tool     | List sessions, inspect transcripts, send cross-session messages          | Action: session management   |
| Nodes           | Tool     | Camera snap/clip, screen record, location.get, notifications             | Action: device capability    |
| Shell Execution | Tool     | Run shell commands on host machine                                       | Action: session tool         |
| File System     | Tool     | Read/write/manage files                                                  | Action: session tool         |
| MCPorter        | Tool     | Discover, configure, authenticate, and call MCP servers via NL           | Resource: MCP management     |

### Memory Plugins

| Extension              | Category | Description                                                                                                                                                        | AGH Mapping                               |
| ---------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------- |
| Memory Core            | Memory   | Markdown files + SQLite FTS5 + sqlite-vec (1536-dim embeddings)                                                                                                    | Capability: memory.backend                |
| Memory LanceDB         | Memory   | LanceDB vector storage with auto-recall/auto-capture                                                                                                               | Capability: memory.backend                |
| Memory LanceDB Pro     | Memory   | Hybrid Vector+BM25, cross-encoder rerank, multi-scope isolation, Weibull decay, smart extraction into 6 categories, three-tier lifecycle (Peripheral/Working/Core) | Capability: memory.backend                |
| Dreaming/Consolidation | Memory   | Background consolidation pass: collects short-term signals, scores candidates, promotes to long-term MEMORY.md. Opt-in, scheduled via cron, thresholded.           | Maps to AGH internal/memory/consolidation |

### Popular ClawHub Skills (Community)

| Extension               | Category | Description                                                        | AGH Mapping                                    |
| ----------------------- | -------- | ------------------------------------------------------------------ | ---------------------------------------------- |
| Web Browsing            | Skill    | Navigate pages, extract content, follow links (180K+ installs)     | Resource: skill                                |
| Telegram Integration    | Skill    | Connect agent to Telegram (145K+ installs)                         | Resource: channel skill                        |
| Capability Evolver      | Skill    | Self-improving agent capability evolution (35K+ downloads)         | Resource: skill + Capability: agent meta-skill |
| Self-Improving Agent    | Skill    | Agent that improves its own performance (132 stars, highest rated) | Resource: skill                                |
| GOG (Google Workspace)  | Skill    | Gmail, Calendar, Drive, Contacts, Sheets, Docs in one skill        | Resource: skill                                |
| Agent Browser           | Skill    | Full browser automation for web workflows (43 stars)               | Resource: skill                                |
| Tavily Search           | Skill    | AI-optimized web search for agents                                 | Resource: skill / MCP server                   |
| n8n Workflow Automation | Skill    | Trigger and manage n8n automation workflows                        | Resource: skill / hook                         |
| MCP Integration         | Skill    | General MCP server connectivity wrapper                            | Resource: MCP                                  |

### Native Apps & Companions

| Extension          | Category           | Description                                                                                        | AGH Mapping                 |
| ------------------ | ------------------ | -------------------------------------------------------------------------------------------------- | --------------------------- |
| macOS Menu Bar App | Native             | Menu bar control plane, Voice Wake/PTT, Talk Mode overlay, WebChat, debug tools                    | N/A (AGH is CLI-first)      |
| iOS App            | Native             | Canvas, screen snapshot, camera, location, talk mode, voice wake                                   | N/A (potential future node) |
| Android App        | Native             | Customizable wake words ("Hey Claw", "Jarvis", "Computer"), offline Vosk detection, device control | N/A (potential future node) |
| Apple Watch        | Native             | Inbox UI and notification relay                                                                    | N/A                         |
| QuickClaw (iOS)    | Native (community) | Clean native iOS interface with notifications, haptics                                             | N/A                         |
| MacClaw            | Native (community) | All-in-one macOS app integrating WhatsApp, Slack, Teams, iMessage                                  | N/A                         |

### Plugin Hook System

| Hook                                                    | Phase       | Description                                                       | AGH Mapping                                    |
| ------------------------------------------------------- | ----------- | ----------------------------------------------------------------- | ---------------------------------------------- |
| `gateway_start` / `gateway_stop`                        | Lifecycle   | Gateway process start/stop                                        | Daemon boot/shutdown hooks                     |
| `session_start` / `session_end`                         | Session     | Session lifecycle boundaries                                      | Resource: hook                                 |
| `before_agent_start` / `agent_end`                      | Agent       | Agent lifecycle (deprecated in favor of before_prompt_build)      | Resource: hook                                 |
| `before_prompt_build`                                   | Prompt      | Intercept and modify system prompt before sending to LLM          | Capability: prompt.provider                    |
| `before_tool_call` / `after_tool_call`                  | Tool        | Guard or transform tool invocations (block/allow/requireApproval) | Capability: permission.gate + content.validate |
| `before_compaction` / `after_compaction`                | Memory      | Context window compaction events                                  | Capability: message.transform                  |
| `message_received` / `message_sending` / `message_sent` | Message     | Message pipeline interception (cancel support)                    | Capability: message.transform                  |
| `tool_result_persist`                                   | Persistence | Transform tool results before transcript write                    | Capability: observe.exporter                   |
| `before_install`                                        | Plugin      | Guard plugin installation (block support)                         | Resource: hook                                 |

### Tool Access Control Profiles

| Profile     | Description                                                  | AGH Mapping                 |
| ----------- | ------------------------------------------------------------ | --------------------------- |
| `minimal`   | Read-only tools only, for observer agents                    | Capability: permission.gate |
| `coding`    | File ops + execution tools for dev agents                    | Capability: permission.gate |
| `messaging` | Adds cross-platform messaging on top of coding               | Capability: permission.gate |
| `full`      | All tools enabled, for trusted agents only                   | Capability: permission.gate |
| Tool Groups | Named collections (`group:runtime`, `group:fs`, `group:web`) | Capability: permission.gate |

---

## Detailed Analysis of Most Impactful Extensions

### 1. Three-Tier Memory with Weibull Decay (memory-lancedb-pro)

**What it does:** Implements a biologically-inspired memory lifecycle with three tiers (Peripheral, Working, Core). Memories start in Peripheral, get promoted to Working with use, and consolidate into Core through the "dreaming" process. Unused memories decay following a Weibull distribution rather than simple TTL expiration. Smart extraction categorizes memories into 6 types (facts, decisions, technical details, relationships, tasks, insights). Hybrid retrieval combines vector similarity with BM25 keyword search, fused via Reciprocal Rank Fusion.

**Why it matters:** AGH already has `internal/memory` with dual-scope (global + workspace) and dream triggers in `internal/memory/consolidation`. The LanceDB Pro approach adds: (a) principled mathematical decay instead of binary keep/delete, (b) multi-stage retrieval that handles both semantic and keyword queries well, (c) automatic categorization that structures memories for better retrieval, (d) cross-encoder reranking for precision.

**AGH mapping:** This maps directly to `memory.backend` capability. AGH could define a `MemoryBackend` interface with methods for `Store`, `Recall`, `Decay`, and `Consolidate`, letting plugins provide SQLite+FTS (built-in), LanceDB, Qdrant, or other backends. The Weibull decay and smart extraction concepts could be built into the consolidation runtime.

### 2. Tool Access Control Profiles & Permission Gates

**What it does:** OpenClaw implements a three-tier permission model: Base Profile (minimal/coding/messaging/full) -> Allow/Deny Lists -> Provider-Specific Policy. Tool groups (`group:runtime`, `group:fs`, `group:web`) bundle related tools. Per-agent overrides narrow the tool set. The `before_tool_call` hook supports `block: true`, `block: false`, and `requireApproval: true` guard semantics, with approval routing to Telegram buttons, Discord interactions, or CLI commands.

**Why it matters:** AGH's `permission.gate` capability needs exactly this kind of layered access control. The profile concept (named capability bundles) is particularly powerful for multi-agent setups where different agents need different trust levels. The approval routing pattern -- where a tool call can be paused and approved via any connected channel -- is a killer feature for autonomous agents.

**AGH mapping:** This maps to `permission.gate` capability with a `ToolPolicy` system. AGH could implement profiles as TOML-defined capability bundles in config, with allow/deny lists per agent. The `before_tool_call` hook pattern maps to AGH's hook resource, with approval routing handled by the Notifier pattern to fan out to CLI (UDS) or web (HTTP/SSE).

### 3. Multi-Channel Gateway with Shared Session Context

**What it does:** A single OpenClaw Gateway process receives messages from 25+ platforms and routes them into the same session store. A conversation started on WhatsApp can continue on Telegram because the context is shared. Each channel adapter runs in its own lightweight thread. Multi-agent routing lets inbound channels/accounts/peers be routed to isolated agents with separate workspaces and sessions.

**Why it matters:** AGH currently exposes HTTP/SSE (web UI) and UDS (CLI). Adding channel adapters as a resource type would let AGH become reachable from any messaging platform. The shared session context across channels is the key innovation -- it is not "three separate bots" but one agent with multiple interfaces.

**AGH mapping:** This maps to a new Resource type: `channel`. Each channel adapter would implement a `ChannelDriver` interface with `Connect()`, `Send()`, `Receive()`, and `Disconnect()` methods. The session manager already handles session lifecycle; channels would just be additional input/output paths. Channel adapters could be Go plugins or subprocess-based (like ACP agents).

### 4. Capability Evolver / Self-Improving Agent

**What it does:** The most downloaded skill on ClawHub. It monitors agent performance patterns, identifies capability gaps, and automatically suggests or creates new skills to fill those gaps. The Self-Improving Agent (highest community rating) takes this further by letting the agent modify its own instruction set based on success/failure feedback.

**Why it matters:** This is a meta-capability -- an agent that improves itself. For AGH, this could be implemented as a skill that uses the Host API (Actions) to inspect session transcripts via `observe`, identify repeated failures or manual interventions, and generate new skills or modify existing ones via the `skills` API.

**AGH mapping:** This maps to an AGH skill (Resource) that uses Actions (Host API for session/observe/skills) to implement a feedback loop. The observe system provides the data, the skills system provides the target for improvement, and the session system provides the execution context.

### 5. Cron + Webhooks as First-Class Primitives

**What it does:** Cron jobs schedule agent actions at specific times (daily summaries, periodic checks). Webhooks provide external trigger points (e.g., Gmail push notification triggers agent processing). Both are configuration-driven, not code-driven. External events trigger the agent loop just like user messages do.

**Why it matters:** AGH's hook resource could be extended to support time-based triggers (cron) alongside event-based triggers (webhooks). This turns agents from reactive (respond to user) to proactive (act on schedule or external event). The config-driven approach (TOML in AGH's case) keeps it simple.

**AGH mapping:** Time-based hooks as a new hook trigger type in the config. Webhook hooks as HTTP endpoints registered by the daemon that inject messages into sessions. Both would flow through the existing session manager and event system.

### 6. Before-Prompt-Build Hook (Prompt Provider)

**What it does:** The `before_prompt_build` hook intercepts and modifies the system prompt before it is sent to the LLM. Plugins can inject context, modify instructions, add safety rails, or transform the prompt based on session state, memory recall results, or channel-specific requirements.

**Why it matters:** This maps directly to AGH's `prompt.provider` capability. It enables: dynamic prompt augmentation based on memory, channel-specific instruction injection (e.g., more concise on mobile), safety rail injection, and RAG-style context injection from external knowledge bases.

**AGH mapping:** Capability: `prompt.provider`. An interface with a `BuildPrompt(ctx, session, basePrompt) -> augmentedPrompt` method. Multiple providers could be chained, each adding their layer (memory context, skill instructions, safety rails).

### 7. MCP as Universal Tool Protocol

**What it does:** Every ClawHub skill is an MCP server. OpenClaw supports both stdio and HTTP/SSE transports. MCPorter provides NL-driven MCP server management. 500+ community MCP servers available.

**Why it matters:** AGH uses ACP (Agent Client Protocol) for agent communication, but MCP is the emerging standard for tool integration. Supporting MCP servers as a resource type would give AGH access to the entire MCP ecosystem (GitHub, Notion, Slack, databases, etc.) without building custom integrations.

**AGH mapping:** Resource: MCP server. AGH already has MCP in its three-dimensional extension model. Implementing MCP client support in Go would let AGH connect to any MCP server (TypeScript, Python, or any language) via stdio or HTTP/SSE transport.

### 8. Smart Memory Extraction (6 Categories)

**What it does:** The LanceDB Pro plugin uses an LLM to automatically categorize and extract memories from conversations into 6 types: facts/preferences, decisions/commitments, technical details, relationships/context, tasks/follow-ups, and insights/learnings. Each category has different retention and retrieval behaviors.

**Why it matters:** AGH's memory system could benefit from structured categorization. Instead of flat memory entries, categorized memories enable more precise recall (e.g., "what technical decisions were made?" vs "what tasks are pending?"). The categories also enable different decay rates -- tasks should not decay the same way facts do.

**AGH mapping:** Extension to `memory.backend` capability. The `Store` method could accept a category enum. The `Recall` method could filter by category. The consolidation runtime could apply category-specific decay curves.

---

## Key Takeaways for AGH Extension Ideas

### High-Priority Extensions to Consider

1. **Channel Adapters** -- Even if AGH starts CLI-first, a `ChannelDriver` interface would let the community add Telegram, Slack, Discord, etc. This is the single highest-impact extension category in OpenClaw's ecosystem (Telegram alone has 145K installs). In AGH, channels would be a Resource type alongside agents, skills, hooks, and MCP.

2. **Tiered Memory with Mathematical Decay** -- AGH already has memory/consolidation. Adding Weibull or exponential decay curves, multi-tier promotion (peripheral/working/core), and structured categorization would make AGH's memory system significantly more sophisticated than flat markdown files.

3. **Tool Permission Profiles** -- Named capability bundles (minimal/coding/messaging/full) with allow/deny lists and per-agent overrides. This maps perfectly to AGH's `permission.gate` capability. Essential for multi-agent setups where agents have different trust levels.

4. **Cron/Webhook Triggers** -- Time-based and event-based triggers that inject messages into sessions. Makes agents proactive, not just reactive. Config-driven via TOML.

5. **MCP Client Support** -- Accessing the 500+ MCP server ecosystem from AGH. Go MCP client libraries exist. This would be a massive force multiplier for available tools.

### Medium-Priority Extensions

6. **Before-Tool-Call Guards with Approval Routing** -- The `requireApproval` pattern that pauses execution and routes approval requests to any connected interface. Critical for autonomous agents running unsupervised.

7. **Prompt Provider Chain** -- Multiple prompt providers that layer context (memory, skills, safety) onto the base prompt before LLM invocation.

8. **Self-Improving Agent / Capability Evolver** -- Meta-skill that uses observe data to identify and fill capability gaps.

9. **Provider Fallback Chains** -- Automatic failover between LLM providers (primary -> fallback -> economy) with different providers for cost optimization.

### Architecture Lessons from OpenClaw

- **Skills as Markdown, Plugins as Code**: OpenClaw's distinction between skills (SKILL.md markdown files, no SDK needed) and plugins (TypeScript modules with full API access) is elegant. AGH could mirror this: skills are YAML/markdown instruction packs, while capabilities are Go interfaces requiring compiled code.

- **Local-First with Opt-In Cloud**: Everything runs locally by default. Cloud features (remote MCP servers, hosted models) are opt-in. This aligns perfectly with AGH's single-binary philosophy.

- **Hook Discovery from Multiple Directories**: OpenClaw discovers hooks from workspace, managed, and bundled directories with clear precedence. AGH could do the same with `<workspace>/hooks/`, `~/.agh/hooks/`, and bundled hooks.

- **Security Through Narrowing**: Each policy layer can only narrow the tool set, never expand it. This is the correct security model for autonomous agents.

- **Marketplace with Safety Rails**: ClawHub's experience with the "ClawHavoc" attack (2,400 malicious skills removed) shows that community marketplaces need: minimum account age for publishing, automated scanning (VirusTotal), download count and age as trust signals, and curated "awesome" lists alongside the open registry.

---

## Sources

- [OpenClaw GitHub Repository](https://github.com/openclaw/openclaw)
- [OpenClaw Official Site](https://openclaw.ai/)
- [ClawHub Marketplace](https://clawhub.ai/)
- [OpenClaw Docs - Plugins](https://docs.openclaw.ai/plugins/building-plugins)
- [OpenClaw Docs - Tools](https://docs.openclaw.ai/tools)
- [OpenClaw Docs - Model Providers](https://docs.openclaw.ai/concepts/model-providers)
- [OpenClaw Docs - Memory](https://docs.openclaw.ai/concepts/memory)
- [OpenClaw Docs - ClawHub](https://docs.openclaw.ai/tools/clawhub)
- [OpenClaw Docs - iOS](https://docs.openclaw.ai/platforms/ios)
- [awesome-openclaw-skills (VoltAgent)](https://github.com/VoltAgent/awesome-openclaw-skills)
- [memory-lancedb-pro Plugin](https://github.com/CortexReach/memory-lancedb-pro)
- [LanceDB Blog - OpenClaw Memory](https://www.lancedb.com/blog/openclaw-memory-from-zero-to-lancedb-pro)
- [OpenClaw Architecture Overview (Substack)](https://ppaolo.substack.com/p/openclaw-system-architecture-overview)
- [Reference Architecture: OpenClaw (RobotPaper)](https://robotpaper.ai/reference-architecture-openclaw-early-feb-2026-edition-opus-4-6/)
- [OpenClaw MCP Guide](https://launchmyopenclaw.com/openclaw-mcp-guide/)
- [OpenClaw Hooks Guide (Team 400)](https://team400.ai/blog/2026-04-openclaw-hooks-practical-guide-custom-automations)
- [Tool Access Control Profiles (Stanza)](https://www.stanza.dev/courses/openclaw/tools-browser-automation/openclaw-tool-access-control-profiles)
- [DigitalOcean - What is OpenClaw](https://www.digitalocean.com/resources/articles/what-is-openclaw)
- [PacGenesis - OpenClaw Guide](https://pacgenesis.com/what-is-openclaw-ai-everything-you-need-to-know-about-the-open-source-ai-agent-that-actually-does-things/)
- [Nerd Level Tech - Complete OpenClaw Guide](https://nerdleveltech.com/guides/openclaw-personal-ai-assistant)
- [Milvus Blog - OpenClaw Explained](https://milvus.io/blog/openclaw-formerly-clawdbot-moltbot-explained-a-complete-guide-to-the-autonomous-ai-agent.md)
- [KDnuggets - Essential OpenClaw Skills](https://www.kdnuggets.com/7-essential-openclaw-skills-you-need-right-now)
- [Best Models for OpenClaw (haimaker.ai)](https://haimaker.ai/blog/best-models-for-clawdbot/)
