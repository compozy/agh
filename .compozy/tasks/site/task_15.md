---
status: pending
title: "Docs: Hooks & Extensions (Event Catalog, Matchers, Marketplace)"
type: docs
complexity: medium
dependencies:
  - task_03
---

# Task 15: Docs: Hooks & Extensions (Event Catalog, Matchers, Marketplace)

## Overview

Write four documentation pages covering AGH's hook system and extension framework. Hooks allow declarative reactions to AGH events (session started, memory written, etc.) with pattern matching and executor dispatch. Extensions are installable packages that add capabilities to AGH. These are split across two domains but grouped in one task for coherence. Pages mix Diataxis **Explanation** (how hooks and extensions work) and **Reference** (event catalog, declaration format).

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Hooks, Extensions"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — pages MUST follow Diataxis Explanation + Reference principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to `internal/hooks/` and `internal/extension/` — read the source
- The event catalog must be complete — enumerate all hookable events from the source code
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/hooks/event-catalog.mdx` — complete catalog of all hookable events with event names, payload types, when they fire, and example payloads
- MUST create `packages/site/content/runtime/hooks/declaration.mdx` — hook declaration format, matchers (event patterns, filters), executors (shell, HTTP, agent), configuration examples
- MUST create `packages/site/content/runtime/extensions/install.mdx` — extension installation, discovery, marketplace search, version management, enable/disable
- MUST create `packages/site/content/runtime/extensions/develop.mdx` — how to develop custom extensions, extension API, packaging, publishing to marketplace
- MUST create `packages/site/content/runtime/hooks/meta.json` — sidebar ordering for hooks
- MUST create `packages/site/content/runtime/extensions/meta.json` — sidebar ordering for extensions
- MUST enumerate ALL hookable events from `internal/hooks/` source code in the event catalog
- MUST document all matcher types and executor types with examples
- MUST include at least 2 complete hook declaration examples
- MUST read `internal/hooks/` and `internal/extension/` source code for implementation accuracy
- SHOULD document the hook execution order and concurrency model
- SHOULD include a practical example of an extension that adds a new capability
</requirements>

## Subtasks

- [ ] 15.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 15.2 Read `internal/hooks/` — event types, matchers, executors, hook registration
- [ ] 15.3 Read `internal/extension/` — extension interface, lifecycle, marketplace integration
- [ ] 15.4 Write `event-catalog.mdx` — complete hookable event catalog with payloads
- [ ] 15.5 Write `declaration.mdx` — hook declaration format with matcher and executor reference
- [ ] 15.6 Write `install.mdx` — extension installation and management how-to
- [ ] 15.7 Write `develop.mdx` — extension development guide
- [ ] 15.8 Create `hooks/meta.json` for sidebar ordering: event-catalog → declaration
- [ ] 15.9 Create `extensions/meta.json` for sidebar ordering: install → develop
- [ ] 15.10 Verify all pages build without errors

## Implementation Details

See TechSpec sections: "Appendix B: Documentation Taxonomy — Hooks", "Appendix B: Documentation Taxonomy — Extensions".

Hooks provide a declarative way to react to AGH events. A hook declaration specifies an event matcher (which events to listen for, with optional filters) and an executor (what to do when matched — run a shell command, call an HTTP endpoint, or start an agent session). Extensions are a broader packaging mechanism that can bundle hooks, skills, bridges, and other capabilities into installable units.

### Relevant Files

- `internal/hooks/hooks.go` — Hook system core, event dispatch
- `internal/hooks/matcher.go` — Event pattern matching, filters
- `internal/hooks/executor.go` — Executor types (shell, HTTP, agent)
- `internal/hooks/events.go` — Event type definitions, catalog
- `internal/extension/extension.go` — Extension interface, lifecycle
- `internal/extension/marketplace.go` — Marketplace client, install/search
- `internal/extension/loader.go` — Extension loading and initialization
- `internal/cli/hook.go` — Hook CLI commands (if exists)
- `internal/cli/extension.go` — Extension CLI commands (if exists)

### Dependent Files

- `packages/site/content/runtime/hooks/` — Output directory for hooks pages
- `packages/site/content/runtime/extensions/` — Output directory for extension pages
- Automation pages (task_13) — Triggers may overlap with hook events
- Skills pages (task_11) — Extensions can bundle skills

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/hooks/event-catalog.mdx` — Complete event catalog
- `packages/site/content/runtime/hooks/declaration.mdx` — Hook declaration reference
- `packages/site/content/runtime/hooks/meta.json` — Hooks sidebar ordering
- `packages/site/content/runtime/extensions/install.mdx` — Extension install how-to
- `packages/site/content/runtime/extensions/develop.mdx` — Extension development guide
- `packages/site/content/runtime/extensions/meta.json` — Extensions sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All four pages render at their expected URLs under `/runtime/hooks/` and `/runtime/extensions/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] Event catalog is complete — all events from `internal/hooks/` are documented
  - [ ] All matcher types and executor types are documented with examples
  - [ ] At least 2 complete hook declaration examples included
  - [ ] Extension install/develop workflows are actionable
  - [ ] No broken internal links

## Success Criteria

- All four MDX pages build and render correctly
- Content follows Diataxis Explanation + Reference principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Event catalog is complete and accurate to `internal/hooks/` source
- Hook declaration format is fully documented with matcher and executor reference
- Extension lifecycle is clearly explained
- Zero build warnings from Fumadocs
