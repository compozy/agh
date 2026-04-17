---
status: completed
title: "Docs: Memory (System, Scopes, Dream Consolidation, Best Practices)"
type: docs
complexity: medium
dependencies:
    - task_03
---

# Task 10: Docs: Memory (System, Scopes, Dream Consolidation, Best Practices)

## Overview

Write the four memory documentation pages covering AGH's persistent dual-scope memory system. Memory is one of AGH's distinctive features — it gives agents persistent context across sessions through global and workspace-scoped memory files, with an automatic "dream" consolidation process that compresses and organizes memories. These pages mix Diataxis **Explanation** (how memory works) and **How-to** (how to configure and use memory).

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Memory"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — pages MUST follow Diataxis Explanation + How-to principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to the memory implementation in `internal/memory/` — read the source
- Dream consolidation is a unique feature — explain it clearly with concrete examples
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/memory/system.mdx` — memory system overview, 4 memory types, MEMORY.md index file, how memory is injected into agent context
- MUST create `packages/site/content/runtime/memory/scopes.mdx` — dual-scope architecture (global vs workspace), scope resolution rules, when to use each scope, file locations
- MUST create `packages/site/content/runtime/memory/dream.mdx` — dream consolidation process, triggers, how consolidation compresses and organizes memories, configuration options
- MUST create `packages/site/content/runtime/memory/best-practices.mdx` — effective memory strategies, what to store, memory hygiene, performance considerations
- MUST create `packages/site/content/runtime/memory/meta.json` — sidebar ordering
- MUST document all 4 memory types with their purpose and format
- MUST explain the MEMORY.md index file structure and how it organizes memory files
- MUST include concrete examples of memory files and their content
- MUST read `internal/memory/` and `internal/memory/consolidation/` for implementation accuracy
- SHOULD include a diagram showing memory flow: session → memory write → consolidation → next session
- SHOULD document the dream consolidation trigger conditions
</requirements>

## Subtasks

- [ ] 10.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 10.2 Read `internal/memory/` package — memory types, scopes, MEMORY.md parsing, file operations
- [ ] 10.3 Read `internal/memory/consolidation/` — dream consolidation runtime, triggers, compression
- [ ] 10.4 Read `docs/rfcs/001_agent-md-with-skills-memory.md` for memory configuration in AGENT.md
- [ ] 10.5 Write `system.mdx` — memory system overview with type catalog
- [ ] 10.6 Write `scopes.mdx` — dual-scope explanation with resolution rules
- [ ] 10.7 Write `dream.mdx` — dream consolidation explanation with trigger conditions
- [ ] 10.8 Write `best-practices.mdx` — practical memory usage guidance
- [ ] 10.9 Create `meta.json` for sidebar ordering: system → scopes → dream → best-practices
- [ ] 10.10 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Memory".

AGH's memory system provides persistent context across sessions through markdown files organized by scope (global and workspace). Each scope has a MEMORY.md index file that lists and categorizes memory entries. The dream consolidation process runs automatically (triggered by configurable conditions) to compress, deduplicate, and reorganize memory files, keeping them useful as they grow.

### Relevant Files

- `internal/memory/memory.go` — Memory manager, types, operations
- `internal/memory/scope.go` — Scope resolution (global vs workspace)
- `internal/memory/consolidation/` — Dream consolidation runtime
- `docs/rfcs/001_agent-md-with-skills-memory.md` — Memory specification in AGENT.md context
- `internal/config/config.go` — Memory-related configuration fields
- `internal/session/` — How memory is loaded/saved during session lifecycle

### Dependent Files

- `packages/site/content/runtime/memory/` — Output directory
- Agents pages (task_09) — Memory is configured in AGENT.md
- Skills pages (task_11) — Skills can contribute memory

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/memory/system.mdx` — Memory system overview
- `packages/site/content/runtime/memory/scopes.mdx` — Dual-scope architecture
- `packages/site/content/runtime/memory/dream.mdx` — Dream consolidation explanation
- `packages/site/content/runtime/memory/best-practices.mdx` — Memory best practices
- `packages/site/content/runtime/memory/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All four pages render at their expected URLs under `/runtime/memory/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] All 4 memory types are documented with examples
  - [ ] MEMORY.md index structure is documented accurately
  - [ ] Dream consolidation triggers and process match source code
  - [ ] Dual-scope resolution rules match implementation
  - [ ] No broken internal links

## Success Criteria

- All four MDX pages build and render correctly
- Content follows Diataxis Explanation + How-to principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Memory types and scopes are accurate to `internal/memory/` implementation
- Dream consolidation is explained clearly with concrete examples
- Best practices are actionable and grounded in real usage patterns
- Zero build warnings from Fumadocs
