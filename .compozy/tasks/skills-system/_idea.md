# Skills System for AGH v2

## Overview

AGH needs a skills system — the primary extensibility surface that lets users, teams, and the community customize agent behavior through reusable, portable workflow instructions. Skills follow the AgentSkills open standard (SKILL.md with YAML frontmatter), are 100% compatible with the ecosystem (30+ tools, 15,000+ skills on ClawHub), and integrate deeply with AGH's daemon architecture: memory-aware prompt assembly, declarative MCP server lazy-loading, and session lifecycle hooks.

This is the second Phase 2 feature after the memory system. It ships in 3 increments: core loader + memory integration → MCP lazy-load + hooks → marketplace.

## Problem

AGH v2 currently has no way for users to extend or customize agent behavior beyond raw system prompts in TOML config. Every session starts with the same instructions regardless of task, project, or user preference. This means:

- **No workflow reuse**: Users manually re-explain procedures every session. A debugging workflow, a code review checklist, or a deployment procedure must be re-stated every time.
- **No community leverage**: 15,000+ skills on ClawHub and skills.sh exist for Claude Code, Codex, Gemini CLI, Cursor, and Copilot. AGH cannot use any of them.
- **No differentiation path**: Without skills, AGH's daemon architecture (memory, sessions, observability) has no user-facing extension point. The daemon manages agents but provides no mechanism for agents to become domain-specific.
- **No composability with memory**: The memory system persists knowledge across sessions, but there's no way to package procedural knowledge (how to do things) alongside episodic memory (what happened). Skills are the procedural memory layer.

### Market Data

- **AgentSkills standard** adopted by Microsoft, OpenAI, Anthropic, Google, Block, Atlassian, Figma, Cursor, JetBrains, and 16+ other tools as of December 2025 (via AAIF under Linux Foundation).
- **ClawHub** grew from 127 skills (Nov 2025) to 15,000+ (Mar 2026). **ClawHavoc incident** (Feb 2026): 341 malicious skills distributing Atomic Stealer malware — security scanning is non-negotiable.
- **skills.sh** (Vercel) launched January 2026 as universal distribution hub for 30+ agents.
- **Codex plugins** (March 2026) bundle skills + MCP servers + app integrations — validates our MCP lazy-load direction.
- **All 6 researched harnesses** (Claude Code, GoClaw, Hermes, OpenClaw, OpenFang, AGH v1) use SKILL.md format with progressive disclosure.

## Summary / Differentiator

AGH's skills system differentiates on five axes no competitor combines:

1. **Memory-integrated skills**: Skills read memory context at prompt assembly and can instruct agents to write memories. No other harness couples skills with persistent cross-session memory.
2. **Declarative MCP lazy-load**: Skills declare MCP servers in frontmatter; the daemon spawns them on demand. This is the Codex plugins pattern but at the daemon level with proper lifecycle management.
3. **Daemon-managed lifecycle**: Session hooks, security scanning, and observability events for every skill activation — the daemon provides governance that CLI-only harnesses cannot.
4. **Hot-reload without restart**: The daemon watches skill directories and refreshes the registry live (GoClaw pattern). Skills can be edited, added, or removed while sessions are active — changes are picked up on the next prompt assembly cycle.
5. **Skill auto-proposal**: The daemon detects recurring complex workflows across sessions (via memory integration) and proposes skill creation (Hermes pattern). Combined with a bundled `skillify` meta-skill, this creates a compounding loop where agent usage generates new skills that improve future sessions.

## Core Features

| # | Feature | Priority | Description |
|---|---------|----------|-------------|
| F1 | SKILL.md Loader & Registry | Critical | Parse SKILL.md files with YAML frontmatter per AgentSkills spec. In-memory registry with thread-safe access (`sync.RWMutex`). 4-level loading hierarchy: bundled (go:embed) → user (~/.agh/skills/ + ~/.agents/skills/) → .agents/skills/ → workspace (.agh/skills/). Override semantics: higher-precedence sources win on name collision. |
| F2 | Prompt Injection via PromptAssembler | Critical | Compose skills into the existing `PromptAssembler` pipeline alongside memory. Generate skill catalog (name + description) for system prompt. Load full skill body on activation via `agh skill view`. Progressive disclosure keeps baseline prompt small. |
| F3 | Memory Integration | Critical | Skills declare memory dependencies in `metadata.agh.memory`. At prompt assembly, daemon injects relevant memories alongside skill content. Skills can instruct agents to write memories via `agh memory save` CLI. Bidirectional but daemon-mediated. |
| F4 | Security Scanning (VerifyContent) | Critical | Scan skill content for prompt injection patterns. Severity levels: Info, Warning, Critical (blocks loading). Scan MCP declarations for dangerous commands. Mandatory for workspace-level and marketplace skills. |
| F5 | CLI Commands | High | `agh skill list [--source]`, `agh skill view <name> [--file <path>]`, `agh skill info <name>`, `agh skill create [name]`. Registered under `agh skill` subcommand group. |
| F6 | MCP Lazy-Load | High | Skills declare MCP servers in `metadata.agh.mcp_servers`. Daemon validates, applies security controls (user consent gate, env scrubbing, command allowlist for non-local skills), and injects into `StartOpts.MCPServers` at session creation. |
| F7 | Lifecycle Hooks | High | Skills declare hooks in `metadata.agh.hooks`. Three events: `on_session_created` (inject additional context), `on_session_stopped` (consolidate/cleanup), `on_prompt_assembly` (modify prompt content). Daemon triggers via existing Notifier pattern. Explicit execution model: ordering by hierarchy precedence, configurable timeouts, error propagation. |
| F8 | ClawHub Marketplace | Medium | `agh skill install <slug>`, `agh skill remove <name>`, `agh skill search <query>`. ClawHub API client with exponential backoff. Cryptographic provenance verification. Override audit trail (warn when workspace skills shadow bundled). |
| F9 | Bundled Skills | Medium | Ship 3-5 starter skills via `go:embed`: code-review, debugging, test-runner, project-setup. Demonstrate the format and provide immediate value. |
| F10 | Hot-Reload | High | Daemon watches skill directories via filesystem watchers (fsnotify). When a SKILL.md is created, modified, or deleted, the registry refreshes without daemon restart. Active sessions pick up changes on next prompt assembly. Inspired by GoClaw's hot-reload pattern — critical for the admin/developer workflow where skills are edited frequently. |
| F11 | Skill Auto-Proposal | Medium | After complex sessions (high tool-call count or multi-step workflows), the daemon detects patterns and proposes skill creation. Inspired by Hermes's auto-proposal loop and GoClaw's evolution nudges. The daemon appends a suggestion to the agent's context at session end: "This session solved a repeatable problem. Consider creating a skill with `agh skill create`." Uses memory integration to track recurring workflow patterns across sessions. Bundled `skillify` meta-skill guides agents through formalizing a session into a SKILL.md. |

## Integration with Existing Features

| Integration Point | How |
|---|---|
| `session.PromptAssembler` | Skills compose into the assembler pipeline alongside memory. Ordering: Memory context → Skill catalog → Skill content (on activation) |
| `session.Notifier` | Lifecycle hooks feed through existing OnSessionCreated/OnSessionStopped/OnAgentEvent fan-out |
| `acp.StartOpts.MCPServers` | MCP lazy-load injects skill-declared servers into the existing MCP injection path |
| `internal/config` | Skill config (directories, disabled skills, defaults) added to TOML config |
| `internal/daemon` | Skill registry created in boot sequence after memory, before HTTP. Wired via composition root — `skills/` never imports `daemon/` |
| `internal/memory` | Skills read memory at assembly time; write memory via CLI instructions to agents |
| `internal/cli` | `agh skill` subcommand group with 7+ commands |

## KPIs

| KPI | Target | How to Measure |
|---|---|---|
| Skill activation rate | > 60% of sessions use ≥ 1 skill | Session events with skill_activated type |
| Cross-client compatibility | 100% of standard AgentSkills-format skills load | Integration tests with skills from Claude Code, OpenClaw, Hermes |
| ClawHub install success rate | > 95% | CLI install command success/failure ratio |
| Prompt token overhead | < 2K tokens for skill catalog (up to 50 skills) | Measure catalog size in prompt assembly |
| Skill load latency | < 50ms p95 for registry operations | Instrumented loader/registry |
| Security scan coverage | 100% of non-bundled skills scanned | VerifyContent call count vs load count |

## Feature Assessment

| Criteria | Question | Score |
|---|---|---|
| **Impact** | How much more valuable does this make AGH? | Must do |
| **Reach** | What % of users would this affect? | Must do |
| **Frequency** | How often would users encounter this value? | Must do |
| **Differentiation** | Does this set us apart or just match competitors? | Strong |
| **Defensibility** | Is this easy to copy or does it compound over time? | Strong |
| **Feasibility** | Can we actually build this? | Strong |

Leverage type: **Compounding Feature** — skill library grows over time, community contributions via ClawHub create network effects, memory integration means skills improve with usage.

## Council Insights

- **Recommended approach:** Ship in 3 increments: (1) Core loader + memory + CLI + security scanning, (2) MCP lazy-load + hooks with security controls, (3) Marketplace with provenance verification.
- **Key trade-offs:** MCP lazy-load is the differentiator but also the biggest security risk. Memory integration is near-free (uses existing PromptAssembler). Full marketplace deferred until demand is validated.
- **Risks identified:** MCP subprocess execution from skill frontmatter is arbitrary code execution — needs user consent gate, env scrubbing, command allowlist. Override hierarchy enables supply-chain attacks — needs audit trail. "Prompt-only" naming is misleading — rename to "prompt-first with declarative MCP."
- **Naming honesty:** Drop "prompt-only runtime" label. Call it "prompt-first runtime with declarative MCP" — accurately describes the execution model.
- **Dissenting view:** Product Mind argued increments 1+2 should ship together because MCP lazy-load is the differentiator. Counter: memory-integrated skills is already unique. If early adopters demand MCP immediately, collapse increments.
- **Stretch goal (V2+):** Skill Bundles — distribution format packaging SKILL.md + MCP + memory templates as one installable unit (like Codex plugins but daemon-managed). Full self-evolution loop with GoClaw-style budget-based nudges at 70%/90% iteration budget.

## Sub-Features

- **Increment 1 — Core**: F1 (Loader/Registry), F2 (Prompt Injection), F3 (Memory Integration), F4 (Security Scanning), F5 (CLI: list/view/info/create), F9 (Bundled Skills), F10 (Hot-Reload)
- **Increment 2 — MCP + Hooks + Auto-Proposal**: F6 (MCP Lazy-Load), F7 (Lifecycle Hooks), F11 (Skill Auto-Proposal + skillify meta-skill), F5 (CLI: extended for hooks/MCP info)
- **Increment 3 — Marketplace**: F8 (ClawHub), F5 (CLI: install/remove/search)

## Out of Scope (V1)

- **Skill orchestration or inter-skill dependencies** — Reserved for Phase 3 (Agent Network Protocol). Skills are independent units; the daemon composes them, but they do not communicate with each other.
- **Executable skill runtimes** (Python, WASM, Node, Shell) — Skills are prompt-first instructions. MCP lazy-load provides the execution surface when needed, but skills themselves are not code.
- **Full self-evolution loop** — GoClaw-style budget-based nudges at 70%/90% of iteration budget. F11 covers the simpler auto-proposal pattern; full evolution with inline nudges is deferred.
- **Skill versioning and pinning** — Lockfile-style deterministic skill sets. Deferred until marketplace usage generates demand for reproducibility.
- **Private registry / enterprise RBAC** — Self-hosted skill registries with role-based access control. Enterprise feature for later.
- **Skill Bundles format** — Packaging SKILL.md + MCP + memory as distributable units. Natural evolution after increment 3 lands.

## Architecture Decision Records

- [ADR-001: Three-Increment Delivery Strategy](adrs/adr-001.md) — Ship core → MCP+hooks → marketplace as independent increments

## Open Questions

- **Prompt assembly ordering**: When both memory and skills inject into the prompt, what's the priority order? Current assumption: Memory context → Skill catalog → Dynamic context. Needs validation with real agent sessions.
- **Hook execution model details**: What happens when two skills declare `on_session_created` hooks? Ordering by hierarchy precedence? Parallel execution with timeout? Needs explicit design in increment 2 techspec.
- **MCP consent UX**: How does the user consent to a skill's MCP server? CLI prompt? Config allowlist? Per-session vs permanent consent? Needs UX design for increment 2.
- **Skill activation trigger**: Claude Code uses `when_to_use` field for auto-triggering. Should AGH support this in `metadata.agh`? Or stick with explicit activation only? The Devil's Advocate flagged this gap.
- **Memory write semantics**: When a skill instructs an agent to write memory, should the daemon validate/scope the write? Or trust the agent's judgment? Relates to security model.
