---
status: pending
title: "Docs: Skills (Overview, SKILL.md, Marketplace, Bundled)"
type: docs
complexity: medium
dependencies:
  - task_03
---

# Task 11: Docs: Skills (Overview, SKILL.md, Marketplace, Bundled)

## Overview

Write the four skills documentation pages covering AGH's skill system — the mechanism for extending agent capabilities through composable, shareable instruction sets. Skills can include MCP sidecars for tool access. These pages mix Diataxis **Explanation** (how skills work), **How-to** (how to create and install skills), and **Reference** (SKILL.md schema).

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Skills"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — pages MUST follow Diataxis Explanation + How-to + Reference principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to the skills implementation — read RFC 002 AND `internal/skills/`
- Document the full skill sources hierarchy (bundled → global → workspace → session)
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/skills/overview.mdx` — what skills are, skill sources hierarchy, how skills are resolved and injected into agent context, relationship to MCP
- MUST create `packages/site/content/runtime/skills/skill-md.mdx` — SKILL.md format reference, frontmatter schema (all fields), markdown body as procedural instructions, complete examples
- MUST create `packages/site/content/runtime/skills/marketplace.mdx` — skill discovery, install/uninstall via CLI, search, version pinning, community marketplace
- MUST create `packages/site/content/runtime/skills/bundled.mdx` — catalog of all bundled skills with descriptions, when to use each, how they are loaded
- MUST create `packages/site/content/runtime/skills/meta.json` — sidebar ordering
- MUST document the complete SKILL.md frontmatter schema with a reference table
- MUST document the skill sources hierarchy with resolution order
- MUST include at least 2 complete SKILL.md examples (one simple, one with MCP sidecar)
- MUST read `docs/rfcs/002_skills-system-final.md` for canonical skill format
- MUST read `internal/skills/` and `internal/skills/bundled/` for implementation details
- SHOULD document MCP sidecar configuration within skills
- SHOULD include a diagram showing skill resolution flow
</requirements>

## Subtasks

- [ ] 11.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 11.2 Read `docs/rfcs/002_skills-system-final.md` for canonical skill specification
- [ ] 11.3 Read `internal/skills/` — skill loader, catalog, resolution
- [ ] 11.4 Read `internal/skills/bundled/` — bundled skill definitions and their content
- [ ] 11.5 Write `overview.mdx` — skill system explanation with sources hierarchy
- [ ] 11.6 Write `skill-md.mdx` — SKILL.md format reference with schema table
- [ ] 11.7 Write `marketplace.mdx` — install, search, manage skills how-to
- [ ] 11.8 Write `bundled.mdx` — bundled skill catalog with descriptions
- [ ] 11.9 Create `meta.json` for sidebar ordering: overview → skill-md → marketplace → bundled
- [ ] 11.10 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Skills".

Skills are composable instruction sets defined in SKILL.md files. They follow a resolution hierarchy: bundled skills (shipped with AGH) → global skills (~/.agh/skills/) → workspace skills (.agh/skills/) → session-level skills. Skills can optionally include MCP sidecar configurations that spawn tool servers alongside the agent. The skills system is one of AGH's key extensibility mechanisms.

### Relevant Files

- `docs/rfcs/002_skills-system-final.md` — Canonical skill system specification
- `internal/skills/skills.go` — Skill catalog, loader, resolution
- `internal/skills/bundled/` — Bundled skill directory with embedded skill definitions
- `internal/config/config.go` — Skill-related configuration fields
- `internal/cli/skill.go` — Skill CLI commands (install, list, search)

### Dependent Files

- `packages/site/content/runtime/skills/` — Output directory
- Agents pages (task_09) — Skills are referenced in AGENT.md
- Configuration Reference (task_17) — SKILL.md schema cross-referenced

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/skills/overview.mdx` — Skills system overview
- `packages/site/content/runtime/skills/skill-md.mdx` — SKILL.md format reference
- `packages/site/content/runtime/skills/marketplace.mdx` — Skill marketplace how-to
- `packages/site/content/runtime/skills/bundled.mdx` — Bundled skill catalog
- `packages/site/content/runtime/skills/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All four pages render at their expected URLs under `/runtime/skills/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] SKILL.md schema table covers all frontmatter fields from RFC 002
  - [ ] Skill sources hierarchy is documented with correct resolution order
  - [ ] At least 2 complete SKILL.md examples are included
  - [ ] All bundled skills are cataloged with descriptions
  - [ ] MCP sidecar configuration is documented
  - [ ] No broken internal links

## Success Criteria

- All four MDX pages build and render correctly
- Content follows Diataxis Explanation + How-to + Reference principles (verified by `/documentation-writer` + `/copywriting` workflow)
- SKILL.md schema is complete and accurate to RFC 002 and `internal/skills/`
- Bundled skill catalog matches actual `internal/skills/bundled/` contents
- Skill resolution hierarchy is clearly explained
- Zero build warnings from Fumadocs
