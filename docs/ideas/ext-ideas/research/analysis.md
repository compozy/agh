# AGH Extension Ideas — Consolidated Research Analysis

**Date**: 2026-04-11
**Sources**: Pi-Mono, Hermes, OpenClaw, Claude Code, OpenFang, GoClaw, broader AI agent ecosystem
**Purpose**: Identify high-impact extension ideas for AGH based on cross-ecosystem research

---

## Executive Summary

Five parallel research agents analyzed the extension ecosystems of six major AI agent frameworks (Pi-Mono, Hermes, OpenClaw, Claude Code, OpenFang, GoClaw) plus the broader MCP/A2A ecosystem. The research surfaced **40+ concrete extension ideas** that map to AGH's three-dimensional extension model (Resources, Capabilities, Actions).

**Three converging industry standards** that AGH must support:

1. **MCP** (Model Context Protocol) — agent-to-tool communication (5,000+ servers)
2. **A2A** (Agent-to-Agent Protocol) — agent-to-agent communication (150+ orgs, Linux Foundation)
3. **OpenTelemetry** — universal agent observability standard

**One critical design principle** discovered across all frameworks:

> _"Hooks guarantee behavior; prompts suggest it."_ Instructions achieve ~70% compliance; hooks achieve 100%. Use hooks for must-enforce rules, instructions for should-follow guidance.

---

## Extension Ideas by AGH Dimension

### Resources (Declarative — bundled with extensions)

#### R1. Agent Packages ("Hands")

**Inspired by**: OpenFang Hands, Goose Custom Distributions, Roo Code Modes
**What**: Self-contained autonomous capability packages bundling agent definition + skills + hooks + MCP configs + settings into a single deployable unit. Each "Hand" is a preconfigured agent persona (e.g., Researcher, Reviewer, DevOps Engineer).
**Why**: Reduces setup friction from "configure 15 things" to "install one package". OpenFang ships 7 bundled Hands; Roo Code's Mode Gallery has hundreds of community modes.
**AGH mapping**: Resource (agents + skills + hooks bundle). TOML manifest (`HAND.toml`) declaring tools, skills, MCP servers, system prompt, dashboard metrics, requirements.
**Priority**: **HIGH** — foundational for ecosystem growth

#### R2. Cron/Scheduled Triggers

**Inspired by**: Hermes cronjob tool, OpenClaw Cron tool, OpenFang scheduled Hands
**What**: Time-based triggers that create sessions on schedule. Natural language cron expressions. Jobs attach skills, deliver results to any connected interface.
**Why**: Makes agents proactive instead of reactive. Nightly code reviews, morning briefings, periodic health checks. Present in every major framework.
**AGH mapping**: Resource (hook trigger type) + Action (session.create). Persist job definitions in `globaldb`. Expose via UDS/HTTP API.
**Priority**: **HIGH** — natural extension of session lifecycle

#### R3. Webhook/Event Bridge

**Inspired by**: OpenClaw Webhooks, Claude Code CI/CD hooks, Goose recipes
**What**: HTTP endpoints that inject messages into sessions on external events (GitHub push, CI failure, Slack mention, email arrival).
**Why**: Enables event-driven agent workflows. Developers want agents triggered by CI events (high-demand feature request).
**AGH mapping**: Resource (hook trigger type) + Action (session.create or session.prompt). Register webhook endpoints in HTTP API.
**Priority**: **MEDIUM** — depends on cron infrastructure

#### R4. Skill Auto-Generation ("Skill Factory")

**Inspired by**: Hermes skill-factory, GoClaw skillEvolve, Pi-Mono self-evolution
**What**: After successful task completion, agent analyzes its steps, extracts reusable patterns, and writes a SKILL.md file. Every N tasks, agent evaluates and refines existing skills.
**Why**: Compounding institutional knowledge. A DevOps agent that deploys 50 times creates a deployment skill capturing all edge cases. Present in Hermes (80+ community extensions in 3 months), GoClaw (nudge prompts at 70%/90% iteration budget).
**AGH mapping**: Resource (skill, auto-generated) + Resource (hook, session.end trigger). Use observe system to capture successful trajectories.
**Priority**: **MEDIUM** — requires observe maturity

#### R5. Channel Adapters

**Inspired by**: OpenClaw 25+ channels, OpenFang 40 adapters, Hermes 14 platforms
**What**: Messaging platform bridges (Telegram, Discord, Slack, etc.) that route messages into AGH sessions. Shared session context across channels.
**Why**: Telegram alone has 145K installs in OpenClaw. Makes agents accessible from anywhere. OpenFang's 40 adapters demonstrate strong demand.
**AGH mapping**: New Resource type: channel. `ChannelDriver` interface with `Connect()`, `Send()`, `Receive()`. Session manager handles lifecycle; channels are additional I/O paths.
**Priority**: **LOW** for now (AGH is CLI/web-first), **HIGH** if demand emerges

---

### Capabilities (Runtime interfaces the extension provides)

#### C1. Permission Gate: Risk Classifier

**Inspired by**: Claude Code PreToolUse hooks, Pi-Mono security/safe-git, OpenClaw tool profiles, GoClaw 7-step policy engine
**What**: Classify tool calls by risk level (low/medium/high/critical), require approval for destructive actions. Composable, stackable gates. Three handler tiers: shell command (fast), AI classifier (semantic), sub-agent (deep analysis).
**Why**: Present in **every** framework. Claude Code's insight: CLAUDE.md instructions = ~70% compliance, hooks = 100%. The AI classifier pattern (natural-language rules evaluated by a fast model) is more expressive than regex.
**AGH mapping**: Capability: `permission.gate`. Interface receiving tool call + context, returning allow/deny/modify/require-approval. Multiple gates chain with priority ordering.
**Priority**: **CRITICAL** — table stakes for production use

#### C2. Content Validator: Secret Redaction

**Inspired by**: Pi-Mono filter-output, GoClaw ScrubCredentials, Hermes credential patterns
**What**: Scan tool outputs for API keys, tokens, passwords, PII before they reach the LLM. Configurable regex patterns. Block or redact.
**Why**: Prevents credential leakage into LLM context (and therefore into provider logs, training data, displayed output). Pi-Mono community built this immediately; GoClaw runs it on every tool output by default.
**AGH mapping**: Capability: `content.validate`. Ship as built-in with configurable patterns via TOML. Scan for AWS keys, GitHub tokens, JWT, private keys, etc.
**Priority**: **HIGH** — security baseline

#### C3. Content Validator: Prompt Injection Scanner

**Inspired by**: OpenFang prompt injection scanner, Hermes HermesHub 65+ threat rules, GoClaw GuardSkillContent
**What**: Scan incoming tool results, skill content, and MCP server outputs for prompt injection patterns. Block or flag.
**Why**: As AGH connects to external MCP servers and loads community skills, injection attacks become a real threat. GoClaw wraps all MCP output in `<<<EXTERNAL_UNTRUSTED_CONTENT>>>` markers.
**AGH mapping**: Capability: `content.validate`. Scan at skill loading, MCP tool result processing, and extension installation time.
**Priority**: **HIGH** — security for ecosystem trust

#### C4. Message Transform: Context Pruning

**Inspired by**: Pi-Mono context-pruning + compaction-safeguard, OpenClaw before_compaction hook, Claude Code PostCompact
**What**: TTL-based and token-budget-based pruning of old tool results before LLM calls. Time-decay model: recent = full, old = head+tail, ancient = removed. Custom compaction strategies per domain.
**Why**: Context management is the #1 challenge for long-running sessions. Every framework implements this. Pi-Mono's four-layer strategy (message count, token count, TTL decay, smart compaction) is the most sophisticated.
**AGH mapping**: Capability: `message.transform`. `CompactionStrategy` interface receiving message history, returning compacted version. Ship default strategy, allow override via config.
**Priority**: **HIGH** — essential for long sessions

#### C5. Memory Backend: Tiered with Decay

**Inspired by**: OpenClaw memory-lancedb-pro (Weibull decay, 3-tier lifecycle), Hermes 8 pluggable backends, Mem0, Letta
**What**: Three-tier memory lifecycle (Peripheral/Working/Core) with mathematical decay curves. Smart extraction categorizing memories into 6 types (facts, decisions, technical details, relationships, tasks, insights). Hybrid retrieval: vector similarity + BM25 keyword search + cross-encoder reranking.
**Why**: AGH already has dual-scope memory + dream consolidation (rare advantage). Adding structured decay and hybrid retrieval would make it best-in-class. Hermes proves community will build diverse backends if interface is clean.
**AGH mapping**: Capability: `memory.backend`. Interface with `Store()`, `Recall()`, `Decay()`, `Consolidate()` methods. Ship SQLite+FTS default. Allow vector/graph backends.
**Priority**: **HIGH** — AGH's existing memory is a differentiator to build on

#### C6. Memory Backend: Knowledge Graph

**Inspired by**: Memory MCP (official), Mem0 graph memory, OpenClaw relationship tracking
**What**: Graph-based memory for entity relationships (who works on what, how components connect, dependency chains). Complements vector memory for relationship-aware recall.
**Why**: Graph memory is emerging as production-critical for complex domains. Vector search finds similar content; graph search finds connected entities.
**AGH mapping**: Capability: `memory.backend` (alternative implementation). Could use SQLite with adjacency lists or embed a lightweight graph engine.
**Priority**: **MEDIUM** — valuable but can start with vector-only

#### C7. Observe Exporter: OpenTelemetry

**Inspired by**: Traceloop OTel MCP, FastMCP native OTel, AG2 OTel tracing, broader ecosystem convergence
**What**: Export AGH events as OpenTelemetry traces with GenAI semantic conventions (model name, provider, token usage, cost, temperature, tool call args/results). Pre-built Grafana dashboards.
**Why**: OpenTelemetry is emerging as the universal standard for AI agent observability. AGH's observe system already captures events; OTel export makes them consumable by existing monitoring infrastructure.
**AGH mapping**: Capability: `observe.exporter`. Translate AGH events to OTel spans. Propagate trace context to MCP servers.
**Priority**: **MEDIUM** — important for production deployments

#### C8. Observe Exporter: Cost Tracker

**Inspired by**: Pi-Mono cost-tracker/usage-bar/context, Claude Code cost tracking, developer surveys
**What**: Real-time token consumption, cost per session, context composition breakdown (how much context is system prompt vs. skills vs. memory vs. conversation). Per-session cost limits with automatic stop.
**Why**: "Cost tracking and budgets" is a top developer feature request. Pi-Mono's `context` extension shows which components consume most tokens.
**AGH mapping**: Capability: `observe.exporter`. Track token allocation across prompt components. Expose via HTTP API for web UI.
**Priority**: **MEDIUM** — strong demand signal

#### C9. Prompt Provider: Dynamic Context Injection

**Inspired by**: OpenClaw before_prompt_build hook, Hermes pre_llm_call hook, Claude Code UserPromptSubmit context injection
**What**: Hook that fires before each LLM call, allowing extensions to inject context (memory recall, RAG results, channel-specific instructions, safety rails) into the prompt without modifying persisted history.
**Why**: Every framework implements this pattern. It's the primary mechanism for memory injection, RAG, and dynamic context augmentation.
**AGH mapping**: Capability: `prompt.provider`. Interface with `AugmentPrompt(ctx, session, baseMessages) -> augmentedMessages`. Multiple providers chain, each adding a layer.
**Priority**: **HIGH** — enables memory, RAG, and context enrichment

#### C10. Agent Driver: Multi-Model Consultation ("Oracle")

**Inspired by**: Pi-Mono oracle extension, Hermes MOA (Mixture of Agents), GoClaw multi-model routing
**What**: Send current conversation context to an alternative AI model for a second opinion without switching the active session. Could also implement provider fallback chains (primary -> fallback -> economy).
**Why**: Different models have different strengths. Getting a second opinion on architecture decisions or bug diagnosis is valuable. Provider fallback chains improve reliability.
**AGH mapping**: Capability: `agent.driver` (multi-dispatch). Add `session.consult` action that sends context to a specified driver and returns response as injected message.
**Priority**: **MEDIUM** — useful differentiation

#### C11. Agent Driver: Remote Execution Backends

**Inspired by**: Pi-Mono pi-ssh-remote, Hermes 6 terminal backends (Docker/SSH/Daytona/Modal), OpenFang Docker sandbox
**What**: Pluggable execution backends that redirect tool execution to remote hosts (SSH), containers (Docker), or cloud sandboxes (Modal, Daytona). The agent thinks it's running locally but commands execute remotely.
**Why**: Security (isolate agent actions), scale (run on powerful remote machines), compliance (execute in approved environments).
**AGH mapping**: Capability: `agent.driver` variant. Execution backend configured per agent or per session.
**Priority**: **LOW** — important but niche initially

---

### Actions (Host API operations extensions can perform)

#### A1. Session Delegation (Parent-Child)

**Inspired by**: Hermes delegate_task (3 concurrent subagents), Claude Code subagent orchestration, OpenFang inter-agent tools
**What**: Spawn child sessions with isolated context, restricted toolsets, and their own workspace. Results flow back to parent. Up to N concurrent children. Zero context cost to parent.
**Why**: Single-agent context windows are finite. Complex tasks (refactor + test + review) benefit from specialized agents that don't pollute each other's context. Present in every major framework.
**AGH mapping**: Action: Host API `sessions/delegate`. Add parent-child session relationships in `internal/session`. Child inherits workspace, gets restricted tool access.
**Priority**: **HIGH** — core orchestration capability

#### A2. Session Fork with Context Handoff

**Inspired by**: Pi-Mono handoff extension, Claude Code worktree isolation
**What**: Distill current conversation context, open editor for review, spawn new focused session with that context. Also: fork session into isolated git worktree for parallel work.
**Why**: Long sessions accumulate noise. Handoff lets users start fresh without losing progress. Worktree isolation enables parallel agents on the same repo.
**AGH mapping**: Action: Host API `sessions/fork`. Session fork with context summarization. Optional git worktree isolation.
**Priority**: **MEDIUM** — natural extension of session model

#### A3. Session Checkpointing & Rewind

**Inspired by**: Pi-Mono pi-rewind, Git-based snapshots per turn
**What**: Automatic git-based snapshots (stored as refs or stash entries) after file-modifying tools. `/rewind` command with checkpoint browser, diff preview, safe restore, redo stack.
**Why**: AI agents make mistakes. Clean rewind of file changes while preserving conversation state is essential for confidence in agent-assisted coding.
**AGH mapping**: Action: Host API (new checkpoint/restore endpoints). Hook on tool completion to create checkpoints. Per-session checkpoint namespacing.
**Priority**: **MEDIUM** — strong UX improvement

#### A4. MCP Server Mode (AGH as Capability Provider)

**Inspired by**: Hermes MCP server mode, OpenFang bidirectional MCP
**What**: AGH exposes its Host API as MCP tools so external agents (Claude Code, Codex, Cursor) can use AGH's sessions, memory, skills, and observe as tools.
**Why**: Makes AGH composable with other agent systems. Being only an MCP client limits AGH to consuming tools; being also a server makes it a building block.
**AGH mapping**: Action: all Host API actions exposed as MCP tools (`agh_create_session`, `agh_query_memory`, `agh_list_skills`, `agh_get_events`). Add MCP server alongside HTTP and UDS servers.
**Priority**: **MEDIUM** — composability multiplier

#### A5. Workflow Engine (DAG Orchestration)

**Inspired by**: OpenFang WorkflowEngine (5 step modes), GoClaw pipelines, broader ecosystem patterns
**What**: DAG-based task orchestration across sessions. Steps are pure data (prompt template + mode + error handling). Five modes: Sequential, FanOut, Collect, Conditional, Loop. Variable interpolation connects steps.
**Why**: Complex tasks need structured multi-step orchestration. The workflow engine adds no execution capability — it only orchestrates when and how existing sessions run.
**AGH mapping**: Action: new `workflows/*` Host API methods. TOML-defined workflow definitions persisted in `globaldb`.
**Priority**: **MEDIUM** — depends on session delegation (A1)

#### A6. A2A Protocol Gateway

**Inspired by**: Google A2A protocol (v0.3, Linux Foundation, 150+ orgs)
**What**: Accept/delegate tasks from external agents via A2A protocol. Publish Agent Cards at `/.well-known/agent.json`. Task lifecycle with SSE streaming.
**Why**: A2A maps directly to AGH's Phase 3 (agent network protocol). Complementary to MCP (MCP = agent-to-tool, A2A = agent-to-agent).
**AGH mapping**: Action: new A2A endpoints in HTTP API. Agent Cards generated from agent definitions. Task submission creates sessions.
**Priority**: **LOW** (Phase 3) — but architecture should be ready

---

## Cross-Cutting Patterns

### Pattern 1: Hook Lifecycle Coverage

Every framework converges on these essential hook points:

| Hook Point      | Claude Code      | Hermes           | Pi-Mono                | OpenClaw            | Priority for AGH |
| --------------- | ---------------- | ---------------- | ---------------------- | ------------------- | ---------------- |
| Pre-tool-call   | PreToolUse       | pre_tool_call    | tool_call (block)      | before_tool_call    | **CRITICAL**     |
| Post-tool-call  | PostToolUse      | post_tool_call   | tool_result            | after_tool_call     | **HIGH**         |
| Pre-LLM-call    | UserPromptSubmit | pre_llm_call     | context                | before_prompt_build | **HIGH**         |
| Post-LLM-call   | —                | post_llm_call    | —                      | —                   | **MEDIUM**       |
| Session start   | SessionStart     | on_session_start | session_start          | session_start       | **HIGH**         |
| Session end     | Stop             | on_session_end   | session_shutdown       | session_end         | **HIGH**         |
| Pre-compaction  | —                | —                | session_before_compact | before_compaction   | **MEDIUM**       |
| Post-compaction | PostCompact      | —                | —                      | after_compaction    | **MEDIUM**       |

AGH's hook system should implement **at minimum** the 6 hooks marked CRITICAL/HIGH.

### Pattern 2: Three-Tier Skill System

All frameworks converge on three skill tiers:

| Tier            | Description                                  | Example                    | AGH Implementation       |
| --------------- | -------------------------------------------- | -------------------------- | ------------------------ |
| **Prompt-only** | Markdown instructions injected into context  | SKILL.md with procedures   | Already supported        |
| **Subprocess**  | Code executed via stdin/stdout JSON protocol | Python/Node/Shell scripts  | Via extension subprocess |
| **Sandboxed**   | Code in WASM sandbox with fuel metering      | Untrusted community skills | Future (Extism/wazero)   |

### Pattern 3: Progressive Disclosure for Token Budget

Every framework implements this: only inject skill name + description into the system prompt (~50 tokens per skill). Full skill content loads on-demand when the agent determines relevance. This enables unlimited skills without context bloat.

### Pattern 4: Security Scanning as Default

Hermes (65+ threat rules), GoClaw (GuardSkillContent), OpenFang (Ed25519 signed manifests), OpenClaw (VirusTotal scanning after ClawHavoc attack) — all gate community extensions through automated security scanning. AGH must build this in from the start.

### Pattern 5: MCP Tool Namespacing

Universal convention: `mcp__{server}__{tool}` (Claude Code uses `mcp__`, GoClaw uses `mcp_`). Prevents collisions when multiple servers expose the same tool name. AGH should adopt `mcp__{server}__{tool}`.

---

## Priority Matrix

### Tier 1: Ship First (Critical for v1 extension ecosystem)

| #   | Extension                        | Dimension  | Why First                                              |
| --- | -------------------------------- | ---------- | ------------------------------------------------------ |
| C1  | Permission Gate: Risk Classifier | Capability | Table stakes for production — every framework has this |
| C2  | Secret Redaction                 | Capability | Security baseline — run on every tool output           |
| C4  | Context Pruning / Compaction     | Capability | Essential for long sessions — #1 user pain point       |
| C9  | Dynamic Context Injection        | Capability | Enables memory injection, RAG, and skills              |
| R1  | Agent Packages                   | Resource   | Foundational for ecosystem — reduces setup friction    |
| A1  | Session Delegation               | Action     | Core orchestration — enables complex workflows         |

### Tier 2: Build Next (High ecosystem demand)

| #   | Extension                 | Dimension  | Why Next                                           |
| --- | ------------------------- | ---------- | -------------------------------------------------- |
| C3  | Prompt Injection Scanner  | Capability | Security for ecosystem growth                      |
| C5  | Tiered Memory Backend     | Capability | Build on AGH's existing memory advantage           |
| C8  | Cost Tracker              | Capability | Top developer feature request                      |
| R2  | Cron/Scheduled Sessions   | Resource   | Makes agents proactive — present in all frameworks |
| A2  | Session Fork with Handoff | Action     | Natural session model extension                    |
| A3  | Session Checkpointing     | Action     | Strong UX improvement for coding workflows         |

### Tier 3: Differentiate (Strategic value)

| #   | Extension                | Dimension  | Why Strategic                        |
| --- | ------------------------ | ---------- | ------------------------------------ |
| C7  | OpenTelemetry Exporter   | Capability | Production monitoring standard       |
| C10 | Multi-Model Consultation | Capability | Unique UX differentiation            |
| R4  | Skill Auto-Generation    | Resource   | Compounding institutional knowledge  |
| A4  | MCP Server Mode          | Action     | Composability multiplier             |
| A5  | Workflow Engine          | Action     | Structured multi-agent orchestration |

### Tier 4: Future (Phase 2-3)

| #   | Extension                 | Dimension  | Why Later                          |
| --- | ------------------------- | ---------- | ---------------------------------- |
| C6  | Graph Memory Backend      | Capability | Valuable but can start vector-only |
| C11 | Remote Execution Backends | Capability | Important but niche initially      |
| R3  | Webhook/Event Bridge      | Resource   | Depends on cron infrastructure     |
| R5  | Channel Adapters          | Resource   | High effort, demand-dependent      |
| A6  | A2A Protocol Gateway      | Action     | Phase 3 agent network protocol     |

---

## Architectural Lessons Synthesized

1. **Hook-based extension is king.** Six lifecycle hooks (pre/post tool, pre/post LLM, session start/end) enable the vast majority of extensions without core code changes. Prioritize hook coverage over adding new capability types.

2. **Memory providers need lifecycle integration, not just CRUD.** The ability to inject context before LLM calls (`prompt.provider`) and retain information after (`observe.exporter`) is what makes memory backends actually useful.

3. **Skills as markdown is the winning pattern.** Low barrier to create (just write markdown), easy to share, token-efficient (progressive disclosure). Three tiers (prompt-only, subprocess, sandboxed) cover all use cases.

4. **Deterministic hooks + probabilistic instructions = complete control.** Use hooks for must-enforce rules (security, formatting, testing). Use skills/instructions for should-follow guidance (coding style, architecture preferences).

5. **MCP dual-role (client + server) makes AGH composable.** Being only a client limits AGH to consuming tools. Being also a server makes it a building block for larger agent systems.

6. **Security scanning is not optional.** Every framework with a community ecosystem learned this — some the hard way (OpenClaw's ClawHavoc attack: 2,400 malicious skills). Build content validation into the extension loading pipeline from day one.

7. **BM25 search is sufficient for tool/skill discovery.** GoClaw and Hermes both implement pure-Go BM25 (k1=1.2, b=0.75) for skill and tool search. Zero external dependencies, lexical match is adequate for tool names. Threshold at ~40 inline tools; defer the rest.

8. **Agent Packages are the distribution unit.** Individual skills and hooks are building blocks; packages (OpenFang Hands, Goose distributions, Roo Code Modes) are the installable product that users actually want.

---

## Sources

Detailed per-project research files:

- [analysis_pi_mono.md](research/analysis_pi_mono.md) — 30+ extensions, lifecycle hooks, skill packages
- [analysis_hermes.md](research/analysis_hermes.md) — 80+ community extensions, 8 memory backends, plugin system
- [analysis_openclaw.md](research/analysis_openclaw.md) — 25+ channels, ClawHub marketplace, memory tiers
- [analysis_claude_code.md](research/analysis_claude_code.md) — 3,000+ MCP servers, 12 hooks, plugin marketplace
- [analysis_ecosystem.md](research/analysis_ecosystem.md) — OpenFang, A2A, MCP ecosystem, developer requests

Previous architectural analyses (from extension architecture task):

- `.compozy/tasks/_archived/20260411-014454-ext-architecture/analysis_*.md`
