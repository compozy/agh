---
status: completed
title: "Docs: Overview (What is AGH, Architecture, Why AGH)"
type: docs
complexity: medium
dependencies:
    - task_03
---

# Task 06: Docs: Overview (What is AGH, Architecture, Why AGH)

## Overview

Write the three foundational overview pages that introduce AGH to new users: what it is, how it is architected, and why it exists compared to alternatives. These pages set the tone for the entire documentation site and establish AGH's identity as an "Agent Operating System" with a built-in network protocol. All three pages are Diataxis **Explanation** type — they build understanding rather than instruct.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy" for content scope
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — all pages MUST follow Diataxis Explanation principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure and content placement — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to the current codebase — read source files to verify claims
- Do NOT copy-paste from README or RFCs — adapt and rewrite for the external audience
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/overview/what-is-agh.mdx` — "Agent Operating System" identity, single-binary daemon, ACP-based agent management, key value props
- MUST create `packages/site/content/runtime/overview/architecture.mdx` — architecture diagram (Mermaid), package layout, data flow, daemon composition root pattern
- MUST create `packages/site/content/runtime/overview/comparison.mdx` — competitive positioning against typical agent harnesses, LangChain/LangGraph, CrewAI, raw MCP
- MUST create `packages/site/content/runtime/overview/meta.json` — sidebar ordering and group title
- MUST use Diataxis Explanation principles: explain "why" and "how things fit together", not step-by-step instructions
- MUST include at least one Mermaid architecture diagram in architecture.mdx
- MUST reference the network protocol as AGH's key differentiator (per TechSpec Hero copy)
- MUST be accurate to current codebase state — read CLAUDE.md, README.md, and source packages for accuracy
- SHOULD include the comparison table from TechSpec Appendix A ("AGH vs typical harness")
- SHOULD use approved copy lines from TechSpec Appendix A where appropriate
</requirements>

## Subtasks

- [x] 6.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [x] 6.2 Read source files (CLAUDE.md, README.md, internal/daemon/, internal/session/, internal/acp/) to gather accurate architectural details
- [x] 6.3 Write `what-is-agh.mdx` — identity, value proposition, key concepts introduction
- [x] 6.4 Write `architecture.mdx` — system architecture with ASCII diagram, package layout, data flow
- [x] 6.5 Write `comparison.mdx` — competitive positioning, feature comparison table, "when to use AGH"
- [x] 6.6 Create `meta.json` for sidebar ordering: what-is-agh → architecture → comparison
- [x] 6.7 Verify all pages build without errors (`turbo run build --filter=@agh/site`)

## Implementation Details

See TechSpec sections: "Appendix B: Documentation Taxonomy — Overview", "Appendix A: Landing Page Copy — Key Copy Lines".

The overview section is the first thing users encounter after the landing page. It must establish AGH's identity clearly: a Go single-binary daemon that manages AI agent sessions via ACP (Agent Client Protocol), spawns agent CLIs as subprocesses, persists events in SQLite, and exposes interfaces via HTTP/SSE and UDS. The network protocol (agent-to-agent communication) is the key differentiator.

### Relevant Files

- `README.md` — Current project description and quick start
- `CLAUDE.md` — Project overview, architecture principles, package layout
- `internal/daemon/daemon.go` — Composition root, boot sequence
- `internal/session/` — Session lifecycle, state machine
- `internal/acp/` — ACP client, subprocess spawn, JSON-RPC
- `internal/config/` — Configuration system
- `docs/ideas/market-pair/gap-analysis.md` — Competitive analysis source material

### Dependent Files

- `packages/site/content/runtime/overview/` — Output directory (must exist from task_03 scaffold)
- `packages/site/lib/source.ts` — Runtime docs loader must serve these pages

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — These pages live in the `runtime` collection

## Deliverables

- `packages/site/content/runtime/overview/what-is-agh.mdx` — Identity and value proposition page
- `packages/site/content/runtime/overview/architecture.mdx` — Architecture explanation with Mermaid diagram
- `packages/site/content/runtime/overview/comparison.mdx` — Competitive positioning page
- `packages/site/content/runtime/overview/meta.json` — Sidebar ordering configuration

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All three pages render at `/runtime/overview/what-is-agh`, `/runtime/overview/architecture`, `/runtime/overview/comparison`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] Architecture page contains at least one Mermaid diagram
  - [ ] Comparison page contains a feature comparison table
  - [ ] No broken internal links between overview pages
  - [ ] No references to nonexistent pages or sections

## Success Criteria

- All three MDX pages build and render correctly
- Content follows Diataxis Explanation principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Architecture diagram accurately reflects current package layout from CLAUDE.md
- Comparison table positions AGH clearly against alternatives
- Pages use approved copy lines from TechSpec where appropriate
- Zero build warnings from Fumadocs
