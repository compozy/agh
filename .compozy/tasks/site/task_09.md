---
status: pending
title: "Docs: Agents (Definitions, Providers, Spawning)"
type: docs
complexity: medium
dependencies:
  - task_03
---

# Task 09: Docs: Agents (Definitions, Providers, Spawning)

## Overview

Write the three agent documentation pages covering how agents are defined, which providers are supported, and how agent spawning works via ACP. Agents are the entities that AGH manages — each is defined by an AGENT.md file with YAML frontmatter and markdown body. These pages mix Diataxis **Explanation** (how the agent system works) and **Reference** (AGENT.md schema, provider list).

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Agents"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — pages MUST follow Diataxis Explanation + Reference principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to the AGENT.md format defined in RFC 001 and implemented in `internal/config/`
- Document ALL built-in providers with their configuration options
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/agents/definitions.mdx` — AGENT.md format, frontmatter schema (all fields with types, defaults, valid values), markdown body as system prompt, file discovery rules, inheritance
- MUST create `packages/site/content/runtime/agents/providers.mdx` — all built-in providers (Claude Code, Codex, Gemini CLI, etc.), provider-specific configuration, how to add custom providers
- MUST create `packages/site/content/runtime/agents/spawning.mdx` — ACP subprocess spawn flow, JSON-RPC over stdio, process lifecycle, environment variable injection
- MUST create `packages/site/content/runtime/agents/meta.json` — sidebar ordering
- MUST document the complete AGENT.md frontmatter schema with a reference table
- MUST include a complete AGENT.md example for at least 2 providers
- MUST read `docs/rfcs/001_agent-md-with-skills-memory.md` for canonical AGENT.md format
- MUST read `internal/config/` for actual parsing implementation and supported fields
- SHOULD include a Mermaid sequence diagram for the agent spawn flow
- SHOULD document the agent resolution cascade (workspace → global → built-in)
</requirements>

## Subtasks

- [ ] 9.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 9.2 Read `docs/rfcs/001_agent-md-with-skills-memory.md` for canonical AGENT.md specification
- [ ] 9.3 Read `internal/config/` for agent definition parsing, validation, supported fields
- [ ] 9.4 Read `internal/acp/` for spawn flow, provider registry, ACP client
- [ ] 9.5 Write `definitions.mdx` — AGENT.md format reference with complete schema table
- [ ] 9.6 Write `providers.mdx` — provider catalog with configuration for each
- [ ] 9.7 Write `spawning.mdx` — ACP spawn explanation with sequence diagram
- [ ] 9.8 Create `meta.json` for sidebar ordering: definitions → providers → spawning
- [ ] 9.9 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Agents".

Agents in AGH are defined by AGENT.md files — markdown documents with YAML frontmatter that specify the provider, model, system prompt, skills, memory settings, and other configuration. The frontmatter is parsed by `internal/config/` and the agent is spawned as a subprocess via `internal/acp/`. The ACP client communicates with the agent process over JSON-RPC/stdio.

### Relevant Files

- `docs/rfcs/001_agent-md-with-skills-memory.md` — Canonical AGENT.md format specification
- `internal/config/config.go` — Agent definition types, parsing, validation
- `internal/config/agent.go` — Agent-specific config handling (if exists)
- `internal/acp/client.go` — ACP client implementation
- `internal/acp/process.go` — Subprocess management, spawn flow
- `internal/acp/providers.go` — Provider registry and built-in providers
- `internal/session/manager_start.go` — Agent resolution during session start

### Dependent Files

- `packages/site/content/runtime/agents/` — Output directory
- Sessions pages (task_08) — Sessions reference agents; agents docs explain the definition side
- Skills pages (task_11) — Skills are referenced in AGENT.md frontmatter

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/agents/definitions.mdx` — AGENT.md format reference
- `packages/site/content/runtime/agents/providers.mdx` — Provider catalog
- `packages/site/content/runtime/agents/spawning.mdx` — ACP spawn explanation
- `packages/site/content/runtime/agents/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All three pages render at their expected URLs under `/runtime/agents/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] AGENT.md schema table covers all frontmatter fields from RFC 001
  - [ ] All built-in providers are documented with their config options
  - [ ] At least 2 complete AGENT.md examples are included
  - [ ] Spawn flow documentation matches actual ACP implementation
  - [ ] No broken internal links

## Success Criteria

- All three MDX pages build and render correctly
- Content follows Diataxis Explanation + Reference principles (verified by `/documentation-writer` + `/copywriting` workflow)
- AGENT.md schema is complete and accurate to RFC 001 and `internal/config/`
- All built-in providers are documented
- Spawn flow explanation matches actual ACP implementation
- Zero build warnings from Fumadocs
