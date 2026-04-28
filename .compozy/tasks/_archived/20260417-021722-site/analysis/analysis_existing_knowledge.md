# Existing Knowledge Base Analysis for Documentation Site

**Date**: 2026-04-15
**Scope**: All specs, RFCs, ideas, plans, design docs, and README across the AGH repository
**Purpose**: Inform the documentation site structure, copy, and content strategy

---

## 1. Content Inventory

### 1.1 Root-Level Documentation

| File        | Type              | Content Summary                                                                                                        |
| ----------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `README.md` | Product overview  | Feature list, architecture diagram, quick start, CLI command tree, configuration reference, project structure, roadmap |
| `CLAUDE.md` | Engineering guide | Build commands, architecture principles, package layout, coding style, testing conventions, skill dispatch table       |

### 1.2 RFCs (`docs/rfcs/`)

| File                                 | Status     | Topic                                                                                             |
| ------------------------------------ | ---------- | ------------------------------------------------------------------------------------------------- |
| `001_agent-md-with-skills-memory.md` | Draft      | Self-contained agent definitions with scoped skills and memory (AGENT.md format)                  |
| `002_skills-system-final.md`         | Draft      | Daemon-managed skills with lifecycle hooks, MCP bridge, security scanning, marketplace            |
| `003_agh-network-v0.md`              | Draft      | AGH Network v0 — full wire format, 7 message kinds, interaction lifecycle, NATS transport binding |
| `004_agh-network-v1.md`              | Draft      | AGH Network v1 — Ed25519 trust profile, conformance levels, extension model processing            |
| `003_agh-network-OLD.md`             | Superseded | Earlier network RFC iteration (pre-v0/v1 split)                                                   |

### 1.3 Design Plans (`docs/plans/`)

| File                                       | Topic                                                                                          |
| ------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| `2026-04-06-workspace-entity-design.md`    | Workspace resolver and entity management                                                       |
| `2026-04-08-agh-network-design.md`         | AGH Network v1 approved direction — protocol name, layered architecture, product moat strategy |
| `2026-04-08-rfc-examples-design.md`        | RFC worked examples and conformance examples                                                   |
| `2026-04-10-automation-techspec-design.md` | Automation system design (cron, triggers, webhooks)                                            |
| `2026-04-15-bridge-adapters-design.md`     | Bridge adapters V1 — messaging channel integration (Telegram, Slack, Discord)                  |

### 1.4 Ideas (`docs/ideas/`)

| Directory             | Content                                                                                                                                                                                                                                                                  |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `network/`            | 5 protocol drafts (draft_1 through draft_5) evolving from "AGORA" metaphor to formal AGH Network; 2 council debate rounds; agora-spec v0.1/v0.2; recipe design                                                                                                           |
| `from-claude-code/`   | 8 analysis documents reverse-engineering Claude Code internals: multi-agent orchestration, memory/autonomous patterns, prompt architecture, query engine, services/infrastructure, tool system, filtered recommendations, and system-level analyses (CLI, infra, kernel) |
| `extensability/`      | 8 analysis documents examining extension patterns across 6+ frameworks (Claude Code, OpenClaw, OpenFang, GoClaw, Hermes, Pi-Mono) plus cross-cutting patterns                                                                                                            |
| `ext-ideas/research/` | Consolidated extension research: analysis of 6 ecosystems, integration catalogs (communication, data/AI, DevOps, specialized)                                                                                                                                            |
| `orchestration/`      | Multi-agent orchestration patterns analysis (choreography, orchestration, immutable state, circuit breaker, saga/compensation) mapped to AGH phases                                                                                                                      |
| `market-pair/`        | Competitive gap analysis: AGH vs OpenClaw, OpenFang, GoClaw, Hermes — full capability matrix                                                                                                                                                                             |
| `qa-e2e/`             | QA end-to-end testing README                                                                                                                                                                                                                                             |
| `anp/`                | Agent Network Protocol conversation logs and HTML artifact                                                                                                                                                                                                               |

### 1.5 Refactoring Docs (`docs/_refacs/`)

| File                                   | Topic                                                                           |
| -------------------------------------- | ------------------------------------------------------------------------------- |
| `20260414-bundle-runtime-reconcile.md` | Bundle/bridge reconciliation refactoring analysis using Martin Fowler's catalog |

### 1.6 Design Assets (`docs/design/`)

8 JPEG images — appear to be UI design screenshots/mockups from Twitter/X posts. Could be used as visual references for the site but would need proper naming and context.

### 1.7 Compozy Task Archives (`.compozy/tasks/_archived/`)

27 archived task directories covering the full development history:

**Core System TechSpecs:**

- `agh-v2` — Complete system rewrite spec (the foundational document)
- `web-client-v1` — Web UI implementation
- `web-ui-redesign` — Paper design system integration

**Memory & Skills:**

- `cc-memory` — Cross-session memory system (memdir + dream consolidation)
- `agh-memory-extensibility` — Memory for AGH v2
- `skills-system` — Skills system increment 1 (core)
- `skills-v2` — Skills v2 (MCP bridge, lifecycle hooks, marketplace)

**Architecture:**

- `ext-architecture` — Extension architecture (two-tier: Go native + JSON-RPC subprocess)
- `extension-registry` — Remote discovery, install, and update
- `core-tasks` — Core tasks and subtasks system
- `hooks` — Lifecycle hooks platform
- `session-resilience` — Session resilience (stop reasons + resume repair)
- `workspace-entity` — Workspace entity management

**Infrastructure:**

- `cli-dual-output` — CLI dual output mode (human + toon)
- `global-plugins` — Zero workdir pollution
- `markdown-frontmatter-migration` — TOML to Markdown+YAML migration
- `supervisor-orchestration` — Supervisor orchestration enforcement

**Networking:**

- `network` — AGH Network v0 implementation
- `channels` — Channel adapters
- `automation` — Automation system (schedules + triggers)

**Refactoring:**

- `refac`, `refac-v2`, `kb-refac` — Systematic codebase refactoring

### 1.8 Sandbox Task (`.compozy/tasks/sandbox/`)

- `_idea.md` — Execution sandbox research (local, Daytona, E2B providers)
- `_techspec.md` — Execution sandbox abstraction + Daytona provider implementation spec

---

## 2. Key Messaging Themes Found Across Documents

### 2.1 "Agent Operating System" Identity

The phrase "Agent Operating System" appears consistently across README, CLAUDE.md, and the v2 techspec. AGH positions itself not as an AI framework or wrapper, but as an operating system for agents. Key messaging:

- "AGH is a local-first daemon that manages AI agent sessions" (README)
- "Sessions are processes with lifecycle management" (orchestration analysis)
- "The daemon is an operating system, not an AI framework" (orchestration analysis)
- The metaphor extends: sessions = processes, events = immutable logs, the network protocol = wire format, NATS = transport

### 2.2 "Single Binary, No Sidecars, No External Services"

This phrase appears in the README and reinforces a core value proposition. The local-first, zero-dependency story is repeated across:

- README: "Single binary. No sidecars. No external services."
- v2 TechSpec: "No NATS, no suture, no gobreaker. Gin for HTTP. Minimal dependency footprint."
- Architecture: "Single-binary and local-first. Sidecars or external control planes require a written techspec."

### 2.3 "Orchestrates Real Agent CLIs, Not API Wrappers"

The gap analysis identifies this as AGH's most defensible differentiator:

- "AGH orchestrates CLIs de agentes reais (Claude Code, Codex, Gemini CLI) como subprocessos via JSON-RPC/stdio — concorrentes reimplementam a logica de agente via wrappers de API LLM" (gap analysis)
- "The ACP subprocess model is genuinely unique" (gap analysis)
- This is the "why AGH exists" message: compose existing agent CLIs rather than reimplementing agent logic

### 2.4 "Designed for Incremental Extension"

Architecture documents emphasize that AGH's flat package structure enables new capabilities without modifying existing packages:

- "New capabilities arrive as new packages wired into daemon/, without modifying existing packages" (CLAUDE.md)
- The three-phase roadmap (core -> memory/skills -> network) reflects this incrementalism
- Extension architecture explicitly supports Go-native (L1) and subprocess (L3) tiers

### 2.5 "Daemon as Governor"

RFC 002 introduces this concept for the skills system but it applies broadly:

- "A long-running daemon process (not a CLI wrapper) manages skill lifecycle, enforces security policy, and maintains observability"
- The daemon owns security scanning, capability enforcement, lifecycle hooks
- This positions AGH against CLI-only tools that cannot provide runtime guarantees

### 2.6 "Agent Coordination Is a Distributed Systems Problem"

The orchestration analysis crystallizes this theme:

- "Multi-agent systems are distributed systems problems, not AI problems" (quoting Bhaumik)
- "Agent coordination is not special. It's the same distributed systems engineering that has been solved for decades"
- AGH applies proven patterns: orchestration, immutable state, circuit breakers, sagas

---

## 3. Network Protocol Evolution Story

The network protocol evolved through 5 distinct phases visible in the documentation:

### Phase 1: The Agora Metaphor (draft_1)

The first draft used a pre-modern marketplace metaphor ("agora"). Key concepts: Spaces as acoustic namespaces, Calls as broadcasts, Whispers as direct messages, Greetings as identity, Seeks as discovery queries, Teach as skill transfer, Tribute as payment. The metaphor was powerful for conceptual alignment but produced a design that mixed protocol layers.

### Phase 2: Council Debates (draft_5, agora-council rounds)

Five-archetype adversarial council (Pragmatic Engineer, Architect, Devil's Advocate, Product Mind, The Thinker) debated the protocol design across two rounds. Key tensions resolved:

- 6 verbs vs 3 verbs (resolved: 3 core verbs with kinds as schema tags)
- Public discovery vs 1:1 pre-arranged (resolved: both modes in same protocol)
- JSON canonicalization for signing (resolved: reference signing library)
- Shared state primitive (resolved: add replaceable kind)

### Phase 3: Formalization as "AGH Network" (plans/2026-04-08-agh-network-design.md)

The approved design plan renamed from "AGORA" to "AGH Network" and established the layered architecture: semantic core (transport-agnostic) + NATS transport profile + baseline trust profile. Critical product decision: "AGH wins on runtime, SDK, observability, and DX, not by making the wire protocol AGH-only."

### Phase 4: RFC v0 (docs/rfcs/003_agh-network-v0.md)

The full v0 specification: 7 message kinds (greet, whois, say, direct, recipe, receipt, trace), interaction lifecycle with 6 states, NATS transport binding, delivery semantics, reason codes, worked examples. Deliberately excludes cryptographic identity (deferred to v1).

### Phase 5: RFC v1 (docs/rfcs/004_agh-network-v1.md)

Adds Ed25519 + JCS baseline trust profile, proof-stripping defense, formal conformance levels (Core Sender/Receiver/Peer, NATS Peer, Verified Peer), normative extension namespacing, fingerprint-based routing. Wire format identical to v0 — purely additive upgrade.

**This evolution story is excellent documentation site content.** It shows rigorous protocol design process: metaphor -> adversarial debate -> formal spec -> incremental hardening.

---

## 4. Technical Concepts That Recur Across Specs

### 4.1 Dual-Scope Pattern

Appears in memory (global + workspace), skills (global + workspace), config (global + workspace overlay), and agents (user-level + project-level). This is a foundational pattern documented in:

- Memory TechSpec: `~/.agh/memory/` (global) + `<workspace>/.agh/memory/` (project-scoped)
- Skills TechSpec: bundled -> user -> .agents -> workspace (4-level hierarchy)
- Config: global `~/.agh/config.toml` + workspace `.agh/config.toml`

### 4.2 Composition Root Discipline

`daemon/` is the sole composition root. No package imports it. This is stated in CLAUDE.md, the v2 TechSpec, and enforced by CI grep checks. The pattern keeps the dependency graph clean and enables incremental extension.

### 4.3 Notifier Pattern for Fan-Out

Used for observability and SSE streaming. Typed interface (not a generic event bus) for session lifecycle events. Appears in the v2 TechSpec and is the mechanism for observer integration, web UI streaming, and network turn-end delivery.

### 4.4 Prompt Assembly Pipeline

The composed assembler pattern (memory context -> agent prompt -> skill catalog) is documented in the skills TechSpec. Each layer is a `PromptProvider` interface, independently testable. The pipeline is constructed unconditionally regardless of which features are enabled.

### 4.5 Session State Machine

`starting -> active -> stopping -> stopped` with resume support. The v2 TechSpec defines this. The session resilience spec adds stop reason taxonomy and infrastructure-level repair on resume.

### 4.6 ACP as Orchestration Protocol

ACP (Agent Client Protocol) is used internally between daemon and agents. HTTP/SSE is used externally. This separation is documented in ADR-003 of the v2 spec. The daemon is an ACP client; agents are ACP servers spawned as subprocesses.

### 4.7 Self-Contained Agent Directory

Agents as directories with `AGENT.md` (YAML frontmatter + Markdown prompt), optional `skills/`, `memory/`, and `mcp.json`. RFC 001 proposes this as the unit of portability: "copy the directory, and the agent works."

### 4.8 Dream Consolidation

Ephemeral agent sessions spawned to synthesize session transcripts into durable memory files. 3-gate triggering (time, sessions, lock). The consolidation agent reads transcripts + memories and writes updated memory files.

### 4.9 Security Scanning at Load Time

Skills `VerifyContent` scans for prompt injection patterns before loading into the registry. Three severity levels: critical (block), warning (log + allow), info (log). Applied to every non-bundled skill on every load, not just at install time.

### 4.10 Execution Environment Abstraction

The sandbox spec introduces Provider, Launcher, and ToolHost interfaces to decouple ACP from local-only execution. Supports local, Daytona, and E2B backends. Key insight: "The right abstraction is not 'sandbox' but 'execution sandbox', with local as a first-class provider."

---

## 5. Competitive Positioning Found in Gap Analysis

### 5.1 AGH's Defensible Differentiators (from gap analysis)

1. **ACP subprocess model** — orchestrates real agent CLIs (Claude Code, Codex, Gemini CLI), competitors reimplements agent logic via API wrappers
2. **Single Go binary** — only GoClaw shares this, and GoClaw is CC BY-NC 4.0 (non-commercial)
3. **Per-session SQLite isolation** — better than in-memory (OpenClaw) or shared flat DB (Hermes)
4. **Composition root discipline** — stricter than any competitor, CI-enforceable
5. **Workspace-scoped design** — developer-workflow-aligned config overlay

### 5.2 Competitors Analyzed

| Competitor | Language   | Stars | Key Strength                                                             |
| ---------- | ---------- | ----- | ------------------------------------------------------------------------ |
| OpenClaw   | TypeScript | 354k  | 22+ messaging adapters, Canvas UI, ClawHub registry                      |
| OpenFang   | Rust       | 16.5k | 53 built-in tools, WASM sandbox, 16 security layers, P2P protocol        |
| GoClaw     | Go         | 2.4k  | 3-layer memory (working/episodic/semantic), pgvector, team orchestration |
| Hermes     | Python     | 51.8k | Self-improving skills, 6 terminal backends, RL training                  |

### 5.3 Table-Stakes Gaps Identified

- Multi-layer memory with semantic search (all competitors have this)
- Scheduled/cron agent execution (all competitors have this)
- Security hardening (credential vault, RBAC, signed audit logs)
- Messaging channel adapters (22+ in OpenClaw, 40 in OpenFang)

### 5.4 Strategic Positioning Language

From the gap analysis conclusion:

> "A posicao mais defensavel do AGH e como um **agent OS centrado em desenvolvedores que orquestra CLIs de agentes reais** — nao um wrapper de API LLM."

Translation: AGH's most defensible position is as a **developer-centric agent OS that orchestrates real agent CLIs** — not an LLM API wrapper.

The risk: "without structured memory and scheduled execution, AGH looks incomplete compared to even the smallest competitor."

---

## 6. Content Gaps -- What Is NOT Documented Yet but Should Be

### 6.1 No User-Facing Conceptual Documentation

All documentation is either implementation specs (TechSpecs) or protocol RFCs. There is no:

- **Getting Started guide** beyond the README quick-start
- **Conceptual overview** explaining "what is an agent OS" for non-technical stakeholders
- **Architecture overview** aimed at users (the current one in CLAUDE.md is for contributors)
- **Use case documentation** showing real-world workflows

### 6.2 No Configuration Reference

The README shows a sample config but there is no comprehensive configuration reference documenting every TOML field, defaults, and valid values.

### 6.3 No Agent Authoring Guide

RFC 001 proposes the AGENT.md format but there is no tutorial for creating agents — frontmatter fields, prompt best practices, skill scoping, memory configuration.

### 6.4 No Skills Authoring Guide

The skills TechSpec is implementation-focused. No documentation teaches users how to write a SKILL.md, use the security scanning, understand the precedence hierarchy, or publish to a marketplace.

### 6.5 No Extension Development Guide

The extension architecture TechSpec defines the model but there is no developer guide for building extensions in Go or TypeScript, no TypeScript SDK documentation, no "hello world" extension tutorial.

### 6.6 No Network Protocol Human-Readable Spec

The RFCs are comprehensive but dense. A human-readable protocol guide with narrative explanations, use case scenarios, and implementation walkthroughs would make the protocol accessible to third-party implementors.

### 6.7 No Web UI Documentation

The web UI is mentioned in the README and has its own TechSpec but there is no user-facing documentation for the web interface.

### 6.8 No Memory System User Guide

The memory TechSpec covers implementation. No guide explains memory types, scopes, dream consolidation, or how agents interact with the memory system from a user perspective.

### 6.9 No CLI Reference

The README lists commands but there is no comprehensive CLI reference with flags, examples, output formats, and error codes.

### 6.10 No Comparison/Migration Guides

The gap analysis contains rich competitive data but this has not been turned into user-facing "AGH vs X" or "Migrating from X to AGH" content.

---

## 7. Recommended Content to Reuse/Adapt for the Site

### 7.1 Directly Reusable (High Adaptation Value)

| Source                      | Target Page                                | What to Extract                                                             |
| --------------------------- | ------------------------------------------ | --------------------------------------------------------------------------- |
| `README.md`                 | Homepage, Quick Start                      | Feature list, architecture diagram, CLI tree, config sample                 |
| RFC 001 (Agent definitions) | "Agents" docs section                      | AGENT.md format, skills/memory scoping, portability story                   |
| RFC 002 (Skills system)     | "Skills" docs section                      | SKILL.md format, security model, precedence hierarchy, comparison table     |
| RFC 003 (Network v0)        | "Network Protocol" docs section            | Wire format, message kinds, lifecycle states, worked examples, NATS binding |
| RFC 004 (Network v1)        | "Network Protocol" docs section (advanced) | Trust profile, conformance levels, proof verification                       |
| Gap analysis                | "Why AGH" / positioning page               | Defensible differentiators, competitive comparison matrix                   |
| v2 TechSpec                 | Architecture reference                     | Component overview, data flow, package dependencies, API endpoints          |
| Orchestration analysis      | Blog post or "Design Philosophy" page      | Pattern mapping, distributed systems insight                                |
| Network drafts 1-5          | Blog series or "Design Story" page         | Protocol evolution narrative                                                |

### 7.2 Needs Significant Rewriting

| Source                       | Target                         | Rewriting Needed                                                      |
| ---------------------------- | ------------------------------ | --------------------------------------------------------------------- |
| Memory TechSpec              | "Memory System" guide          | Strip implementation details, add user-facing examples and workflows  |
| Skills TechSpec              | "Skills" authoring guide       | Convert API specs to tutorial format, add real SKILL.md examples      |
| Extension TechSpec           | "Extension Development" guide  | Convert interfaces to narrative tutorial, add TypeScript SDK examples |
| Sandbox idea/techspec        | "Execution Environments" guide | Extract user-facing config examples, explain provider model simply    |
| Session resilience TechSpec  | "Session Management" guide     | Extract stop reasons and resume behavior into user-facing docs        |
| Config sections from v2 spec | "Configuration Reference"      | Consolidate all TOML fields into a comprehensive reference            |

### 7.3 Content Worth Creating from Scratch (Not in Existing Docs)

| Page                         | Why                                  | Source Material Available                  |
| ---------------------------- | ------------------------------------ | ------------------------------------------ |
| "What is an Agent OS?"       | No conceptual intro exists           | README + gap analysis positioning language |
| "Getting Started" (tutorial) | README quick-start is too brief      | README + CLI tree + config samples         |
| "Use Cases"                  | No real-world scenarios documented   | Implied by features but never spelled out  |
| "AGH vs Alternatives"        | Gap analysis is internal-only        | Full competitive matrix in gap analysis    |
| "FAQ"                        | No FAQ exists                        | Common questions implied by docs           |
| "Contributing"               | CLAUDE.md serves but is too internal | CLAUDE.md architecture + testing sections  |

### 7.4 Visual and Design Assets

The 8 JPEG files in `docs/design/` appear to be UI screenshots or design mockups. They need:

- Proper naming (currently Twitter/X filename hashes)
- Context documentation (what each shows)
- Evaluation for site use (hero images, feature showcases)

### 7.5 Key Quotes for Marketing Copy

From the existing docs, these phrases have strong messaging potential:

1. "Single binary. No sidecars. No external services." (README)
2. "Agent Operating System" (project identity)
3. "Orchestrates real agent CLIs, not API wrappers" (gap analysis insight)
4. "Copy the directory, and the agent works" (RFC 001 portability)
5. "The daemon is an operating system, not an AI framework" (orchestration analysis)
6. "Security at the boundary" (RFC 002)
7. "Multi-agent systems are distributed systems problems, not AI problems" (orchestration analysis)
8. "Hooks guarantee behavior; prompts suggest it" (extension research)
9. "Agent coordination is not special. It's the same distributed systems engineering solved for decades" (orchestration analysis)
10. "AGH wins on runtime, SDK, observability, and DX, not by making the wire protocol AGH-only" (network design plan)

---

## 8. Content Architecture Recommendation

Based on this inventory, the documentation site should be organized into these major sections:

1. **Overview** — What is AGH, why it exists, key differentiators
2. **Getting Started** — Install, first session, first agent, first prompt
3. **Concepts** — Agent OS model, sessions, workspaces, dual-scope pattern
4. **Guides** — Agent authoring, skill authoring, memory system, extension development
5. **Reference** — CLI commands, configuration, API endpoints, AGENT.md schema, SKILL.md schema
6. **Architecture** — Package layout, data flow, design decisions (adapted from CLAUDE.md/TechSpecs)
7. **AGH Network** — Protocol overview, wire format, message kinds, lifecycle, trust profile, implementation guide
8. **Ecosystem** — Extensions, marketplace, channel adapters, execution sandboxes
9. **Comparisons** — Why AGH, AGH vs alternatives (adapted from gap analysis)
10. **Blog/Design Stories** — Protocol evolution, engineering decisions, pattern analysis

The richest content ready for adaptation is in the RFCs (001-004), the v2 TechSpec, and the gap analysis. The biggest gaps are user-facing tutorials and conceptual introductions.
