# Hermes Agent (hermes-agent) -- Extension & Plugin Research for AGH

## Overview

Hermes Agent is an open-source, self-improving AI agent framework built by Nous Research, released February 2026, written in Python. It has ~23k GitHub stars, 142 contributors, and ships as a single CLI binary with a multi-platform messaging gateway. Hermes is the closest comparable project to AGH in spirit: a daemon-like agent harness with persistent memory, session management, tool orchestration, and a rich extension model.

Key architectural parallels to AGH:

- **Single-binary daemon** with CLI and gateway modes
- **SQLite-backed persistence** with FTS5 for session search
- **Plugin/extension model** spanning tools, hooks, memory backends, and skills
- **MCP integration** as both client and server
- **Subagent delegation** for parallel workstreams
- **Multi-channel communication** (Telegram, Discord, Slack, WhatsApp, Signal, CLI)

Hermes is three months old and already has 80+ community extensions. This analysis extracts concrete extension ideas for AGH's three-dimensional model: Resources, Capabilities, and Actions.

---

## Table of Extensions and Tools

### Built-in Tools (47 registered tools across 20 toolsets)

| Name / Toolset   | Category       | Description                                                                                                                                                       | AGH Mapping                                                                                                                                                |
| ---------------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `terminal`       | Execution      | Six backends: local, Docker, SSH, Daytona, Singularity, Modal. Background process management (list, poll, wait, log, kill, write). PTY mode for interactive CLIs. | **Capability: agent.driver** -- AGH's ACP driver spawns subprocesses; terminal backends map to driver variants. Add Docker/SSH/serverless driver backends. |
| `web`            | Search/Extract | Web search, page extraction, URL safety checking, website policy compliance                                                                                       | **Resource: tool** -- Web search as a built-in tool exposed via ACP. Could be an MCP server integration.                                                   |
| `browser`        | Automation     | Full browser automation via CDP (navigate, click, type, screenshot). Backends: Browserbase cloud, Browser Use cloud, local Chrome, local Chromium.                | **Capability: agent.driver** or **Resource: MCP** -- Browser automation as an MCP server or specialized driver.                                            |
| `file`           | Filesystem     | File read/write/edit with persistent context                                                                                                                      | **Resource: tool** -- Already covered by ACP agent file tools.                                                                                             |
| `vision`         | Multimodal     | Image analysis via vision-capable models. Clipboard paste support.                                                                                                | **Capability: content.validate** or **message.transform** -- Vision as a content processing capability.                                                    |
| `image_gen`      | Creative       | Text-to-image via FAL.ai FLUX 2 Pro with auto-upscaling                                                                                                           | **Resource: MCP** -- Image generation as an MCP server.                                                                                                    |
| `tts`            | Voice          | Text-to-speech with 5 backends (Edge TTS, NeuTTS, ElevenLabs, etc.). Markdown stripping for natural speech.                                                       | **Resource: MCP** -- TTS as an MCP tool server.                                                                                                            |
| `transcription`  | Voice          | STT via faster-whisper (local), Groq, or OpenAI. Hallucination filtering (26 known phrases).                                                                      | **Resource: MCP** -- Transcription as an MCP tool server.                                                                                                  |
| `cronjob`        | Scheduling     | Built-in cron scheduler with natural language. Jobs attach skills, deliver results to any platform. Pause/resume/edit.                                            | **Resource: hook** + **Action: session** -- Cron as a hook trigger that creates scheduled sessions. High value for AGH.                                    |
| `delegation`     | Orchestration  | Spawn isolated subagents (up to 3 concurrent) with own conversation, terminal, and restricted toolsets. Zero-context-cost via RPC.                                | **Action: session** -- Subagent delegation maps directly to AGH session spawning. Critical capability.                                                     |
| `code_execution` | Execution      | Sandboxed Python execution with RPC access to all Hermes tools. 300s timeout, 50 tool calls max, 50KB stdout cap.                                                 | **Capability: agent.driver** -- Code execution sandbox as a driver variant or tool.                                                                        |
| `memory`         | Persistence    | Dual-file memory (MEMORY.md + USER.md) injected into system prompt. 8 pluggable backends.                                                                         | **Capability: memory.backend** -- Direct mapping. AGH already has this dimension.                                                                          |
| `session_search` | Recall         | SQLite FTS5 full-text search over all past sessions with LLM summarization                                                                                        | **Action: observe** -- Session search as an observe/query capability.                                                                                      |
| `skills`         | Knowledge      | On-demand knowledge documents with progressive disclosure. Auto-creation from experience.                                                                         | **Resource: skill** -- Direct mapping. AGH already has skills.                                                                                             |
| `todo`           | Planning       | Task/todo management within agent sessions                                                                                                                        | **Resource: tool** -- Simple tool, low priority.                                                                                                           |
| `moa`            | Routing        | Multi-model orchestration/routing (Mixture of Agents)                                                                                                             | **Capability: prompt.provider** -- Model routing as a prompt/provider capability.                                                                          |
| `homeassistant`  | IoT            | Smart home control: list entities, control devices, watch state changes. Auto-enabled via HASS_TOKEN.                                                             | **Resource: MCP** -- Home Assistant as an MCP server integration.                                                                                          |
| `rl`             | Training       | RL training pipeline with Atropos (trajectory API), Tinker (training service), and custom environments. GRPO with LoRA.                                           | **Capability: observe.exporter** -- Training data export. Unique to Hermes; not directly applicable to AGH v1.                                             |
| `voice_mode`     | Interface      | Push-to-talk terminal, voice messages in messengers, Discord VC join/listen/speak                                                                                 | **Resource: hook** -- Voice as a communication channel hook.                                                                                               |
| `clarify`        | UX             | Ask user for clarification when instructions are ambiguous                                                                                                        | **Capability: permission.gate** -- Clarification as a gating mechanism.                                                                                    |
| `send_message`   | Communication  | Send messages across all connected platforms (Telegram, Discord, Slack, etc.)                                                                                     | **Resource: MCP** -- Messaging as an MCP server (Hermes already does this as MCP server mode).                                                             |

### Plugin System

| Plugin / Feature             | Category     | Description                                                                                                                     | AGH Mapping                                                                                          |
| ---------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `pre_llm_call` hook          | Lifecycle    | Fires before each LLM call. Can inject context into ephemeral system prompt. Used by memory plugins to inject recalled context. | **Resource: hook** -- Pre-processing hook. Maps to AGH's hook system. Critical for memory injection. |
| `post_llm_call` hook         | Lifecycle    | Fires after each LLM response. Used by memory plugins to retain conversation turns.                                             | **Resource: hook** -- Post-processing hook. Maps to AGH's hook system.                               |
| `pre_tool_call` hook         | Lifecycle    | Fires before tool execution. Can intercept/modify tool calls.                                                                   | **Resource: hook** -- Tool interception hook. Maps to AGH's hook system.                             |
| `post_tool_call` hook        | Lifecycle    | Fires after tool execution. Can process/modify tool results.                                                                    | **Resource: hook** -- Tool result processing hook.                                                   |
| `on_session_start` hook      | Lifecycle    | Fires when a session begins. Used for initialization, context loading.                                                          | **Resource: hook** -- Session lifecycle hook. AGH already has session state machine events.          |
| `on_session_end` hook        | Lifecycle    | Fires when a session ends. Used for cleanup, memory extraction.                                                                 | **Resource: hook** -- Session lifecycle hook.                                                        |
| CLI subcommand registration  | Extension    | Plugins can register new CLI subcommands via the plugin context API.                                                            | **Resource: hook** (CLI extension) -- AGH could allow extensions to register CLI commands.           |
| Request-scoped API hooks     | Extension    | Hooks receive correlation IDs for request tracing.                                                                              | **Capability: observe.exporter** -- Observability enhancement.                                       |
| Env var prompting on install | UX           | Plugins prompt for required env vars during installation.                                                                       | **Resource: hook** (install lifecycle) -- Plugin installation UX.                                    |
| Plugin discovery (3 sources) | Architecture | `~/.hermes/plugins/` (user), `.hermes/plugins/` (project), pip entry points                                                     | **Architecture** -- AGH could support user-dir, project-dir, and Go plugin discovery.                |

### Memory Providers (8 pluggable backends)

| Provider                           | Category          | Description                                                                                                      | AGH Mapping                                                                                         |
| ---------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| **Built-in** (MEMORY.md + USER.md) | Local             | Two curated markdown files injected into system prompt. Agent-editable.                                          | **Capability: memory.backend** -- Default backend. AGH's current memory system.                     |
| **Honcho**                         | Cloud/Self-hosted | Dialectic reasoning and deep user modeling. Builds model of how user thinks, not just what they said. AGPL v3.0. | **Capability: memory.backend** -- Advanced user modeling backend. High value concept.               |
| **Hindsight**                      | Local/Cloud       | Best recall accuracy (91.4% on LongMemEval). Async prefetch + retain. Semantic, graph, temporal retrieval.       | **Capability: memory.backend** -- High-accuracy retrieval backend.                                  |
| **Holographic**                    | Local SQLite      | HRR (Holographic Reduced Representations). Sub-millisecond retrieval. Zero deps. Trust scoring with decay.       | **Capability: memory.backend** -- Lightweight local backend. Interesting for AGH's SQLite approach. |
| **RetainDB**                       | Cloud (paid)      | Hybrid search (Vector + BM25 + Reranking). 7 memory types. Delta compression.                                    | **Capability: memory.backend** -- Cloud backend option.                                             |
| **Mem0**                           | Cloud             | Fastest setup, free tier. Simple extraction.                                                                     | **Capability: memory.backend** -- Easy onboarding backend.                                          |
| **ByteRover**                      | Local Markdown    | Human-readable, inspectable memory stored as Markdown files.                                                     | **Capability: memory.backend** -- Debug-friendly backend.                                           |
| **OpenViking**                     | Local             | Tiered memory loading (L0/L1/L2) for token efficiency.                                                           | **Capability: memory.backend** -- Tiered loading is a smart optimization.                           |

### Skills System

| Skill Category      | Examples                                              | Description                                                      | AGH Mapping                                                                               |
| ------------------- | ----------------------------------------------------- | ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| Apple/macOS         | iMessage, Reminders, Notes, FindMy                    | macOS-specific automation. Platform-gated (only loads on macOS). | **Resource: skill** -- Platform-conditional skills. AGH could gate skills by OS/platform. |
| Agent Orchestration | Multi-agent workflows, coding agent spawning          | Skills for delegating to and coordinating with other agents.     | **Resource: skill** + **Action: session** -- Multi-agent coordination skills.             |
| Data Science        | Jupyter, data analysis, visualization                 | Interactive exploration and notebook-based workflows.            | **Resource: skill** -- Domain knowledge skills.                                           |
| Creative            | ASCII art, hand-drawn diagrams, visual design         | Creative output skills.                                          | **Resource: skill** -- Domain knowledge skills.                                           |
| DevOps              | Infrastructure automation                             | CI/CD, deployment, infrastructure skills.                        | **Resource: skill** -- Domain knowledge skills.                                           |
| Media               | YouTube transcripts, GIF search, music gen, audio viz | Media processing and generation.                                 | **Resource: skill** -- Domain knowledge skills.                                           |
| MLOps               | Model hub, GPU cloud, eval benchmarks, quantization   | ML workflow automation.                                          | **Resource: skill** -- Domain knowledge skills.                                           |
| Smart Home          | Light/switch/sensor control                           | Home automation skills.                                          | **Resource: skill** -- Domain knowledge skills.                                           |
| Social Platforms    | Posting, reading, monitoring                          | Social media automation.                                         | **Resource: skill** -- Domain knowledge skills.                                           |

### Community Extensions (Selected from 80+)

| Extension                          | Author             | Status       | Description                                                                                                     | AGH Mapping                                                                                                         |
| ---------------------------------- | ------------------ | ------------ | --------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| **hermes-workspace**               | outsourc-e         | Production   | Web-based GUI: chat, terminal, memory browser, skills manager, inspector                                        | **Resource: MCP** / Web UI -- AGH already has web UI via HTTP/SSE. Workspace concept maps to AGH's web layer.       |
| **mission-control**                | builderz-labs      | Production   | Agent fleet orchestration dashboard. Dispatch tasks, track costs, coordinate multi-agent workflows. 3.7k stars. | **Action: session** + **observe** -- Fleet management is a natural AGH extension for multi-session orchestration.   |
| **hermes-payguard**                | nativ3ai           | Experimental | USDC/x402 payment plugin with spending limits and approval flows                                                | **Capability: permission.gate** -- Payment gating/approval as a permission gate.                                    |
| **hindsight** (plugin)             | Vectorize          | Production   | Long-term memory layer. retain/recall/reflect workflows. 8.3k stars.                                            | **Capability: memory.backend** -- Memory backend plugin.                                                            |
| **hermes-web-search-plus**         | robbyczgw-cla      | Beta         | Multi-provider web search with intelligent routing (Serper, Tavily, Exa)                                        | **Resource: MCP** -- Search aggregation as an MCP server.                                                           |
| **lintlang**                       | roli-lpci          | Beta         | Static linter for agent configs/prompts. HERM v1.1 scoring.                                                     | **Resource: tool** -- Config validation tool. Could be a pre-session hook.                                          |
| **hermes-plugins** (4-pack)        | 42-evey            | Beta         | Goal management, inter-agent bridge, model selection, cost control                                              | **Multiple** -- Each maps to different AGH dimensions.                                                              |
| **hermes-skill-factory**           | community          | Beta         | Auto-generates SKILL.md files from successful workflows                                                         | **Resource: skill** -- Skill auto-generation. High value for AGH's skills system.                                   |
| **hermes-weather-plugin**          | FahrenheitResearch | Beta         | Professional weather with NWS model imagery, NEXRAD radar                                                       | **Resource: MCP** -- Domain-specific MCP server.                                                                    |
| **hermes-agent-acp-skill**         | Rainhoole          | Beta         | Multi-agent delegation bridging Hermes, Codex, and Claude Code                                                  | **Resource: skill** + **Capability: agent.driver** -- Cross-agent delegation. Directly relevant to AGH's ACP model. |
| **Anthropic-Cybersecurity-Skills** | community          | Production   | 734+ security skills mapped to MITRE ATT&CK. 3.6k stars.                                                        | **Resource: skill** -- Security skill library.                                                                      |
| **autonovel**                      | NousResearch       | Production   | Autonomous novel-writing pipeline (100k+ words)                                                                 | **Resource: skill** -- Long-running workflow skill.                                                                 |
| **hermes-agent-self-evolution**    | NousResearch       | Research     | Evolutionary self-improvement via DSPy and GEPA                                                                 | **Capability: observe.exporter** -- Self-improvement pipeline. Research-grade.                                      |
| **HermesHub**                      | amanning3390       | Production   | Curated skills marketplace with security scanning (65+ threat rules), creator marketplace, x402 payments        | **Architecture** -- Skills marketplace concept for AGH.                                                             |
| **vessel-browser**                 | unmodeled-tyler    | Experimental | AI-native Linux browser with MCP control                                                                        | **Resource: MCP** -- Browser as MCP server.                                                                         |
| **orahermes-agent**                | jasperan           | Production   | Oracle AI Agent Harness with OCI GenAI integration                                                              | **Capability: agent.driver** -- Enterprise driver variant.                                                          |
| **portable-hermes-agent**          | rookiemann         | Beta         | Windows desktop app bundling 100 tools, GUI, local models                                                       | **Architecture** -- Desktop packaging concept.                                                                      |

### MCP Integration Details

| Feature                 | Description                                                                                                  | AGH Mapping                                                                 |
| ----------------------- | ------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------- |
| MCP Client (native-mcp) | Auto-discovers MCP servers, registers tools, supports stdio + HTTP transports                                | **Resource: MCP** -- AGH already supports MCP. Enhance with auto-discovery. |
| MCP Server Mode         | Hermes exposes its messaging capabilities as an MCP server (list conversations, read history, send messages) | **Resource: MCP** -- AGH could expose session/memory/observe as MCP server. |
| Dynamic tool updates    | Handles `notifications/tools/list_changed` for runtime tool registry updates                                 | **Resource: MCP** -- Dynamic tool refresh.                                  |
| Security filtering      | Allow/block lists and attribute-based rules for MCP tools                                                    | **Capability: permission.gate** -- MCP tool filtering as a permission gate. |
| OAuth 2.1 PKCE          | Full OAuth flow for MCP server authentication                                                                | **Resource: MCP** -- Auth for MCP servers.                                  |
| OSV Malware Scanning    | Automatic vulnerability scanning of MCP extension packages                                                   | **Capability: content.validate** -- Security scanning for extensions.       |
| IDE Integration         | VS Code, Zed, JetBrains can register MCP servers that Hermes picks up                                        | **Resource: MCP** -- IDE-sourced MCP server discovery.                      |

---

## Detailed Analysis of High-Impact Extensions

### 1. Pluggable Memory Backends

**What Hermes does:** Memory is abstracted behind a provider ABC (Abstract Base Class). Eight backends implement it, from local SQLite (Holographic) to cloud services (Hindsight, Honcho). Memory providers hook into `pre_llm_call` to inject recalled context and `post_llm_call` to retain new information. The agent sees a unified interface regardless of backend.

**Why it matters for AGH:** AGH already has `memory.backend` as a Capability dimension. Hermes proves that the community will build diverse memory backends if the interface is clean. The key insight is that memory providers need lifecycle hooks (pre/post LLM call) to be truly useful -- simple CRUD is not enough.

**AGH adaptation:**

- Define a `MemoryBackend` interface in `internal/memory/` with `Recall(ctx, query) -> []Memory` and `Retain(ctx, turn) error` methods
- Wire it into the session lifecycle via the existing hook system
- Ship Holographic-style SQLite backend as default (zero deps, local-first)
- Allow registration of additional backends via the plugin system

### 2. Cron/Scheduled Sessions

**What Hermes does:** A built-in cron scheduler lets users define recurring tasks in natural language. Jobs can attach skills, use specific tools, and deliver results to any messaging platform. Jobs support pause/resume/edit.

**Why it matters for AGH:** AGH sessions are currently user-initiated. Scheduled sessions enable autonomous operation: nightly code reviews, morning briefings, periodic health checks, automated testing runs. This is a natural extension of AGH's session lifecycle.

**AGH adaptation:**

- Add a `scheduler` package under `internal/` with cron expression parsing
- Integrate with `internal/session` to create sessions on schedule
- Persist job definitions in `globaldb`
- Expose via UDS API for CLI management and HTTP API for web UI
- Map to **Resource: hook** (cron trigger) + **Action: session.create**

### 3. Subagent Delegation with Isolated Context

**What Hermes does:** The `delegate_task` tool spawns child agent instances with isolated context, restricted toolsets, and their own terminal sessions. Up to 3 concurrent subagents. Results are collected with zero context cost to the parent.

**Why it matters for AGH:** AGH already manages sessions, but parent-child session relationships and context isolation are not yet modeled. Delegation enables complex workflows: a parent session spawns specialized child sessions for parallel tasks, collects results, and synthesizes.

**AGH adaptation:**

- Add parent-child session relationships in `internal/session`
- Add a `delegate` action to the Host API
- Child sessions inherit parent's workspace but get restricted tool access
- Results flow back via the observe system
- Map to **Action: session.delegate** + **Capability: agent.driver** (child driver selection)

### 4. Skill Auto-Generation (Skill Factory)

**What Hermes does:** After completing a task successfully, the agent analyzes its steps, identifies reusable patterns, and writes a SKILL.md file capturing the workflow. Next time a similar task arises, it loads the skill. Every 15 tasks, the agent evaluates and refines skills.

**Why it matters for AGH:** This is the "self-improving" core of Hermes. For AGH, it means agents can build institutional knowledge over time. A DevOps agent that deploys 50 times creates a deployment skill that captures all edge cases.

**AGH adaptation:**

- Add skill generation to `internal/skills/` triggered by session completion hooks
- Use the observe system to capture successful session trajectories
- LLM-based skill extraction as a post-session hook
- Store generated skills in workspace-scoped skill directory
- Map to **Resource: skill** (auto-generated) + **Resource: hook** (session.end trigger)

### 5. MCP Server Mode (Exposing Agent Capabilities)

**What Hermes does:** Hermes can act as an MCP server, exposing its messaging capabilities to other MCP clients. Other agents (Claude Code, Codex, Cursor) can use Hermes's messaging, conversation history, and platform delivery as tools.

**Why it matters for AGH:** AGH could expose its session management, memory, skills, and observe capabilities as MCP tools. This makes AGH a "capability provider" for any MCP-compatible agent, not just a harness for running agents.

**AGH adaptation:**

- Add MCP server mode to `internal/api/` alongside HTTP and UDS servers
- Expose key Host API actions as MCP tools: `agh_create_session`, `agh_query_memory`, `agh_list_skills`, `agh_get_events`
- This makes AGH composable with other agent systems
- Map to **Resource: MCP** (server role) + **Action: all Host API actions**

### 6. Security Scanning for Extensions (Skills Guard)

**What Hermes does:** All hub-installed skills pass through a security scanner checking 65+ threat rules across 8 categories: data exfiltration, prompt injection, destructive commands, obfuscation, hardcoded secrets, network abuse, env abuse, supply-chain signals. Critical findings block installation.

**Why it matters for AGH:** As AGH's extension ecosystem grows, untrusted extensions become a risk vector. A validation layer for skills, hooks, and MCP servers prevents malicious or buggy extensions from compromising the agent or host system.

**AGH adaptation:**

- Add a `validate` package under `internal/skills/` or a general `internal/security/`
- Implement pattern-based scanning for skill content before loading
- Gate MCP server connections through permission checks
- Map to **Capability: content.validate** + **Capability: permission.gate**

### 7. Multi-Platform Messaging Gateway

**What Hermes does:** A single gateway process handles 14 platform adapters (Telegram, Discord, Slack, WhatsApp, Signal, Feishu/Lark, WeCom, DingTalk, SMS/Twilio, Mattermost, Matrix, Webhook, Home Assistant, CLI). Cross-platform conversation continuity -- start on Telegram, continue on Discord.

**Why it matters for AGH:** AGH currently exposes HTTP/SSE (web) and UDS (CLI). Adding messaging platform adapters would make AGH accessible from anywhere, enabling always-on agent availability.

**AGH adaptation:**

- Add a `gateway` package under `internal/api/` with adapter interface
- Each platform adapter implements message receive/send
- Route incoming messages to session creation/resumption
- Map to **Resource: hook** (message adapters) + **Action: session**

### 8. Credential Pool with Rotation

**What Hermes does:** Same-Provider Credential Pools let you configure multiple API keys for the same provider. Thread-safe least-used strategy distributes load. 401 failures trigger automatic rotation.

**Why it matters for AGH:** Multi-key management is essential for production deployments where rate limits and key rotation are concerns. AGH agents making many API calls benefit from automatic key distribution.

**AGH adaptation:**

- Add credential pool support in `internal/config/`
- Thread-safe rotation with `sync.RWMutex`
- Automatic failover on auth errors
- Map to **Capability: agent.driver** (provider credential management)

---

## Key Takeaways for AGH Extension Ideas

### Highest-Priority Extensions (Immediate Value)

1. **Pluggable memory backends** -- AGH already has the `memory.backend` dimension. Ship a clean interface and one or two backends (SQLite-based local, plus one cloud option). The pre/post LLM call hook pattern is essential.

2. **Cron/scheduled sessions** -- Natural extension of AGH's session lifecycle. Enables autonomous operation without user initiation. Relatively straightforward to implement with AGH's existing session manager.

3. **Subagent delegation** -- AGH manages sessions; parent-child relationships and context isolation unlock complex multi-step workflows. This is a differentiator.

4. **MCP server mode** -- Expose AGH's Host API as MCP tools so other agents can use AGH as a capability provider. Composability multiplier.

### Medium-Priority Extensions (Ecosystem Growth)

5. **Skill auto-generation** -- Self-improving skills from session trajectories. Requires observe system maturity but delivers compounding value.

6. **Security scanning for extensions** -- Content validation for skills and MCP servers. Important as the extension ecosystem grows.

7. **Credential pool/rotation** -- Production-grade key management. Important for reliability.

8. **Platform-conditional resource loading** -- Skills/tools that only load on specific OS/platforms (like Hermes's macOS-only skills).

### Lower-Priority but Interesting (Future Phases)

9. **Multi-platform messaging gateway** -- Telegram/Discord/Slack adapters. High effort, niche demand initially.

10. **RL training pipeline** -- Trajectory generation and model fine-tuning from agent interactions. Research-grade, relevant for Phase 3.

11. **Voice mode** -- STT/TTS pipeline. Niche but differentiating for certain use cases.

12. **Skills marketplace** -- Community skill distribution with security scanning. Requires ecosystem scale.

### Architectural Lessons from Hermes

- **Hook-based extension is king.** Hermes's four lifecycle hooks (`pre_llm_call`, `post_llm_call`, `on_session_start`, `on_session_end`) plus two tool hooks (`pre_tool_call`, `post_tool_call`) enable the vast majority of extensions without touching core code. AGH's hook system should prioritize these six hook points.

- **Memory providers need lifecycle integration, not just CRUD.** The ability to inject context before LLM calls and retain information after is what makes memory backends actually useful. Simple read/write APIs are insufficient.

- **Skills as markdown documents with progressive disclosure** is the winning pattern. Low barrier to create (just write markdown), easy to share, and the progressive disclosure pattern minimizes token waste.

- **Plugin discovery from three sources** (user home, project directory, package registry) covers all use cases: personal customization, project-specific tools, and community distribution.

- **MCP dual-role (client + server)** makes the agent composable. Being only an MCP client limits the agent to consuming tools; being also an MCP server makes it a building block for larger systems.

- **Security scanning is not optional.** Hermes learned this early and gates all community extensions through 65+ threat rules. AGH should build this in from the start, not bolt it on later.

---

## Sources

- [NousResearch/hermes-agent GitHub](https://github.com/nousresearch/hermes-agent)
- [Hermes Agent Documentation](https://hermes-agent.nousresearch.com/docs/)
- [Tools & Toolsets Reference](https://hermes-agent.nousresearch.com/docs/user-guide/features/tools)
- [Skills System](https://hermes-agent.nousresearch.com/docs/user-guide/features/skills/)
- [MCP Integration](https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp)
- [Memory Providers](https://hermes-agent.nousresearch.com/docs/user-guide/features/memory-providers)
- [RL Training](https://hermes-agent.nousresearch.com/docs/user-guide/features/rl-training)
- [Voice Mode](https://hermes-agent.nousresearch.com/docs/user-guide/features/voice-mode/)
- [Home Assistant Integration](https://hermes-agent.nousresearch.com/docs/user-guide/messaging/homeassistant)
- [Plugin Guide](https://hermes-agent.nousresearch.com/docs/guides/build-a-hermes-plugin/)
- [awesome-hermes-agent](https://github.com/0xNyk/awesome-hermes-agent)
- [Hermes Agent Ecosystem Map](https://hermes-ecosystem.vercel.app/)
- [HermesHub Skills Marketplace](https://github.com/amanning3390/hermeshub)
- [Hindsight Memory Provider](https://hindsight.vectorize.io/sdks/integrations/hermes)
- [Hermes Agent v0.5.0 Release](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.3.28)
- [Hermes Agent v0.7.0 Release](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.4.3)
- [Bundled Skills Catalog](https://hermes-agent.nousresearch.com/docs/reference/skills-catalog)
- [Creating Skills](https://hermes-agent.nousresearch.com/docs/developer-guide/creating-skills/)
- [Architecture](https://hermes-agent.nousresearch.com/docs/developer-guide/architecture/)
- [Hermes Agent on DEV Community](https://dev.to/arshtechpro/hermes-agent-a-self-improving-ai-agent-that-runs-anywhere-2b7d)
- [Hermes Agent Memory Explained (Vectorize)](https://vectorize.io/articles/hermes-agent-memory-explained)
- [Memory Providers Compared (Vectorize)](https://vectorize.io/articles/hermes-agent-memory-providers-compared)
