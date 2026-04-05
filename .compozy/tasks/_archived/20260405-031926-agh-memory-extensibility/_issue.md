# Cross-Session Memory System With Dream Consolidation for AGH v2

## Overview

AGH v2 needs a persistent memory system that enables agents to accumulate institutional knowledge across sessions. This is the first Phase 2 feature — the foundation on which skills, agent network, and every subsequent capability will be built.

The system comprises three subsystems: (1) a file-based persistent memory store (`memdir`) with dual global/workspace directories and MEMORY.md indexes injected into agent prompts; (2) a dream consolidation service that spawns ephemeral ACP agent sessions to synthesize session transcripts into durable memory files; and (3) team memory for cross-agent knowledge sharing within workspaces.

This is for power developers managing multiple AI agents (Claude Code, Codex, Gemini CLI) through AGH, with interface discipline that enables future harness builders to compose memory primitives into custom workflows.

## Summary / Differentiator

AGH is the only production-grade, local-first daemon that provides **unified cross-agent memory with automatic dream consolidation**. Claude Code ships Auto Dream for single-agent memory. Mem0 ($24M Series A) provides framework-agnostic memory stores. Google ADK has cloud-managed MemoryService. But no existing product consolidates memories from Claude Code + Codex + Gemini CLI sessions into a unified, locally-owned knowledge base. AGH's daemon architecture with SQLite-backed event persistence makes cross-agent dream consolidation architecturally inherent — dream reads all session event databases regardless of agent type.

Dream consolidation reading session events from ALL agent types (Claude, Codex, Gemini) creates unified cross-agent knowledge — something no competitor offers. This compounds across hundreds of sessions, creating a data moat that's hard to replicate.

## Problem

Every AGH session starts from zero context. An agent that spent 30 minutes learning a codebase's conventions, debugging patterns, and team preferences loses all that knowledge when the session ends. The next session repeats the same discovery, the same corrections, the same ramp-up.

This problem compounds across agents. When a user runs Claude Code for backend work and Codex for frontend, neither agent knows what the other learned. Decisions made in one session don't inform the next. The user becomes the sole knowledge bridge between their agents — manually repeating context, re-explaining preferences, and correcting the same mistakes.

The market validates this pain. 88% of AI agent projects fail, and Gartner identifies infrastructure and governance gaps — not the agent technology itself — as the primary cause. Memory is core infrastructure. Without it, agents are disposable commodities that never compound value.

### Market Data

- **$10.9B** global AI agent market in 2026, growing at 40.5% CAGR to $139B by 2034
- **88%** of agent projects fail due to infrastructure/governance gaps (not agent technology)
- **40%** of Fortune 500 firms have adopted CrewAI agents (280% adoption increase in 2025)
- **Claude Code Auto Dream** shipped in v2.1.59+ — consolidates 913 sessions in under 9 minutes, validating the concept
- **Mem0** raised $24M Series A (October 2025) — framework-agnostic memory with 19 vector store backends
- **Google ADK** formalized memory as three services (SessionService, ArtifactService, MemoryService) backed by Vertex AI Memory Bank
- **LangMem SDK** supports episodic/semantic/procedural but creates deep framework lock-in
- **Procedural memory** consistently described as "often overlooked" across all frameworks — deferred to skills phase per ADR-004
- **ICLR 2026 Workshop "MemAgents"** covers hippocampal-cortical consolidation as design inspiration for AI agent memory
- **UC Berkeley/Letta paper** "Sleep-time Compute" showed pre-computation during idle time reduces test-time compute by 2.5x

## Core Features

| # | Feature | Priority | Description |
|---|---|---|---|
| F1 | Persistent Memory Store (memdir) | Critical | File-based memory store with dual global/workspace directories, YAML frontmatter metadata, MEMORY.md indexes. Supports CRUD operations via CLI and HTTP API. Ported from proven cc-memory implementation. |
| F2 | Prompt Assembly Pipeline | Critical | New subsystem that injects memory context (global + workspace MEMORY.md indexes) into agent system prompts on session start. Defined as `PromptAssembler` interface for future extensibility (skills, agent network). |
| F3 | Dream Consolidation | Critical | Background service that spawns ephemeral ACP agent sessions to synthesize session transcripts into durable memory files. 3-gate trigger (time, session count, lock). Reads event databases from ALL agent types for cross-agent synthesis. 4-phase prompt: Orient, Gather, Consolidate, Prune. |
| F4 | Team Memory | High | Workspace-scoped memory files with `agent_name` in frontmatter for cross-agent knowledge sharing. All agents in a workspace can read shared memories. Replaces old project's blackboard approach with simpler file-based design. |
| F5 | Memory CLI Commands | High | `agh memory list/read/write/delete/consolidate` subcommand group. Agents interact with memory via CLI passthrough through ACP `terminal/create` calls. Human and JSON output formats. |
| F6 | Memory HTTP/UDS API | High | REST endpoints for memory CRUD and consolidation trigger. Both HTTP (web UI) and UDS (CLI) interfaces. Follows existing API patterns in httpapi/udsapi packages. |
| F7 | Dream Session Type | Medium | Explicit `SessionType` enum (user, dream, system) with different permission defaults. Dream sessions run with `approve-all` permissions internally. Prevents permission model from blocking memory writes during consolidation. |
| F8 | Staleness & Freshness | Medium | Memory age tracking with >1 day warning threshold. Memories older than 1 day get caveat appended in prompt context. Dream consolidation prunes stale memories. |

## Integration with Existing Features

| Integration Point | How |
|---|---|
| `internal/session/` Manager | Session creation triggers memory context loading; session stop is a dream consolidation candidate |
| `internal/acp/` Driver | Agents access memory via `terminal/create` ACP calls running `agh memory` CLI commands |
| `internal/store/` EventRecorder | Dream consolidation reads per-session `events.db` SQLite databases for transcript synthesis |
| `internal/observe/` Notifier | Dream session events recorded like any other session; consolidation visible in observability |
| `internal/daemon/` Composition Root | Daemon boot initializes memory store, dream service, periodic ticker; wires PromptAssembler |
| `internal/config/` Config | New `[memory]` and `[memory.dream]` TOML sections for configuration |
| `internal/httpapi/` + `internal/udsapi/` | New memory endpoint handlers following existing route patterns |

## KPIs

| KPI | Target | How to Measure |
|---|---|---|
| Memory utilization rate | > 60% of sessions read from memory | Count sessions that trigger `memdir.LoadIndex()` vs total sessions via event store |
| Dream consolidation success rate | > 90% completions without error | Track dream session completion vs failures in event store |
| Context ramp-up reduction | > 40% fewer repeated corrections per workspace | Compare correction-type events across sessions pre/post memory |
| Memory freshness | > 80% of active memories < 7 days old | Scan memory files, compute age distribution via staleness module |
| Cross-agent knowledge sharing | > 30% of memories written by one agent type, used by another | Track agent_name on write events vs session agent_name on read |

## Feature Assessment

| Criteria | Question | Score |
|---|---|---|
| **Impact** | How much more valuable does this make the product? | **Must do** — memory is the foundational layer for Phase 2+. Without it, agents are stateless commodities. |
| **Reach** | What % of users would this affect? | **Must do** — 100% of sessions benefit from accumulated context. |
| **Frequency** | How often would users encounter this value? | **Must do** — every session start triggers memory load; every session end is a consolidation candidate. |
| **Differentiation** | Does this set us apart or just match competitors? | **Strong** — cross-agent consolidation is unique. Single-agent memory matches Claude Code/Windsurf. |
| **Defensibility** | Is this easy to copy or does it compound over time? | **Must do** — memory compounds: more sessions = richer knowledge. Cross-agent consolidation creates an increasingly deep knowledge moat. |
| **Feasibility** | Can we actually build this? | **Must do** — proven design from cc-memory (memdir + dream portable), clear integration points in v2. |

Leverage type: **Compounding Feature** — memory gets more valuable with every session. Cross-agent consolidation creates a data moat that compounds over time and makes switching costs real.

## Council Insights

- **Recommended approach:** Port memdir as `internal/memory/` with minimal changes. Build a minimal prompt assembly layer as a `PromptAssembler` interface. Adapt dream consolidation to spawn ephemeral ACP sessions with trusted internal permissions (SessionType enum). Team memory uses workspace-scoped files with agent metadata in frontmatter. Cross-agent consolidation is architecturally inherent.
- **Key trade-offs:** (1) Minimal prompt assembly will likely need redesign when skills arrives — bounded cost, informed by real usage. (2) Team memory as workspace files is simpler but less queryable than a store-based approach — metadata in frontmatter enables future queryability. (3) Dream as full ACP session is heavier than a lightweight approach — but preserves the daemon's orchestration-only role and provides full observability.
- **Risks identified:** (1) Dream agent permission bypass — mitigated by SessionType enum with different defaults. (2) Prompt assembly pipeline is genuinely new work (~2-3 days) — mitigated by interface-first design.
- **Stretch goal (V2+):** Memory-powered observability dashboard showing accumulated knowledge and decision trends from session event streams. Also: formal extensibility layer extracted from memory + skills patterns.

## Out of Scope (V1)

- **Formal plugin/extension architecture** — Extract extensibility patterns after implementing memory + skills (2+ concrete features). Current Go interfaces and composition root provide sufficient seams.
- **Vector/semantic search over memories** — V1 uses file-based MEMORY.md indexes injected into prompts. Semantic search (embeddings, vector DB) is a V2 enhancement when memory volume justifies it.
- **Memory-powered observability dashboard** — Compelling but better as V1.1 after the memory store exists. Requires web UI work beyond current scope.
- **MCP server for memory** — Agents access memory via CLI passthrough. MCP memory server is a future alternative when MCP adoption warrants it.
- **Interactive dream approval** — Dream runs autonomously with trusted permissions. User approval flow for dream outputs is a future governance feature.
- **Memory encryption/access control** — Local-first system with file permissions. Fine-grained memory access control is a future multi-user feature.
- **Cross-machine memory sync** — Memory is local to `~/.agh/memory/`. Cloud sync or multi-machine memory is out of scope.

## Architecture Decision Records

- [ADR-001: Interleaved Extensibility — Build Memory With Intentional Seams, Defer Formal Plugin System](adrs/adr-001.md) — Build memory with clean interfaces and extension seams but without a formal plugin architecture
- [ADR-002: PromptAssembler Interface in session/ With memory/ Implementation](adrs/adr-002.md) — Define assembly interface where consumed, implement in memory package
- [ADR-003: Frozen Snapshot Memory Injection With Dream-Only Extraction](adrs/adr-003.md) — Load memory once at session start, extract via periodic dream consolidation only
- [ADR-004: Four-Type Memory Taxonomy — Drop Procedural, Defer to Skills Phase](adrs/adr-004.md) — Keep proven 4-type taxonomy, avoid format conflict with future skills system

## Open Questions

- ~~**Dream consolidation agent**~~: **Resolved** — configurable via `[memory.dream] agent = "claude"` in TOML, defaults to Claude for strong summarization. See TechSpec config section.
- ~~**Team memory conflict resolution**~~: **Resolved** — dream consolidation's Consolidate phase resolves contradictions with "newer wins" strategy. Atomic file writes prevent corruption from concurrent access.
- **Memory size budgeting**: How much of the agent's context window should memory consume? MEMORY.md indexes are capped at 200 lines/25KB per scope, but with global + workspace combined, total injection could reach 400 lines/50KB.
- **Prompt assembly ordering**: When skills and agent network arrive, what's the injection order? Memory > Skills > Network context? Need a priority/ordering model for the PromptAssembler.
