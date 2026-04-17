---
status: completed
title: "Docs: Workspaces (Resolver, Config Overlays, Multi-Root)"
type: docs
complexity: medium
dependencies:
    - task_03
---

# Task 12: Docs: Workspaces (Resolver, Config Overlays, Multi-Root)

## Overview

Write the three workspace documentation pages covering how AGH discovers, registers, and manages workspaces — the directory-based scoping mechanism that contextualizes everything from configuration to memory to skills. These pages mix Diataxis **Explanation** (how workspace resolution works) and **How-to** (how to register and configure workspaces).

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Workspaces"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — pages MUST follow Diataxis Explanation + How-to principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to `internal/workspace/` — read the source for resolver logic
- "Copy the directory, and the agent works" is a key selling point — reinforce this
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/workspaces/resolver.mdx` — workspace resolution algorithm, directory discovery, registration, the `.agh/` directory convention, auto-detection
- MUST create `packages/site/content/runtime/workspaces/config-overlays.mdx` — how workspace-level config overlays global config, TOML merge semantics, workspace-scoped agent/skill overrides
- MUST create `packages/site/content/runtime/workspaces/multi-root.mdx` — multi-root workspace support, monorepo patterns, nested workspace resolution
- MUST create `packages/site/content/runtime/workspaces/meta.json` — sidebar ordering
- MUST explain the workspace resolution cascade with a concrete example
- MUST document the `.agh/` directory structure and what each file/directory does
- MUST read `internal/workspace/` source code for resolver implementation accuracy
- SHOULD include a tree diagram showing a typical `.agh/` directory layout
- SHOULD include examples for monorepo and multi-project setups
</requirements>

## Subtasks

- [ ] 12.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 12.2 Read `internal/workspace/` — resolver, registration, entity management
- [ ] 12.3 Read `internal/config/` — workspace-level config overlay mechanics
- [ ] 12.4 Write `resolver.mdx` — workspace resolution algorithm and directory discovery
- [ ] 12.5 Write `config-overlays.mdx` — configuration overlay system with merge examples
- [ ] 12.6 Write `multi-root.mdx` — multi-root support and monorepo patterns
- [ ] 12.7 Create `meta.json` for sidebar ordering: resolver → config-overlays → multi-root
- [ ] 12.8 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Workspaces".

Workspaces provide directory-scoped context for all AGH features. A workspace is typically a project directory containing a `.agh/` subdirectory with agents, skills, memory, and config overrides. The resolver walks up the directory tree to find the nearest workspace root. Configuration at the workspace level overlays global config, enabling project-specific agent and skill settings.

### Relevant Files

- `internal/workspace/workspace.go` — Workspace type definition, entity structure
- `internal/workspace/resolver.go` — Resolver algorithm, directory walking, registration
- `internal/config/config.go` — Config overlay merge for workspace-level settings
- `internal/config/merge.go` — Config merge implementation
- `internal/cli/workspace.go` — Workspace CLI commands (add, list, edit)
- `internal/store/globaldb/` — Workspace persistence in global database

### Dependent Files

- `packages/site/content/runtime/workspaces/` — Output directory
- Memory pages (task_10) — Workspace-scoped memory
- Skills pages (task_11) — Workspace-scoped skills
- Configuration Reference (task_17) — Config overlay details

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/workspaces/resolver.mdx` — Workspace resolution explanation
- `packages/site/content/runtime/workspaces/config-overlays.mdx` — Config overlay how-to
- `packages/site/content/runtime/workspaces/multi-root.mdx` — Multi-root patterns
- `packages/site/content/runtime/workspaces/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All three pages render at their expected URLs under `/runtime/workspaces/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] Resolver documentation matches actual algorithm in `internal/workspace/resolver.go`
  - [ ] Config overlay merge semantics are accurately documented
  - [ ] `.agh/` directory structure is documented with tree diagram
  - [ ] Multi-root examples cover monorepo patterns
  - [ ] No broken internal links

## Success Criteria

- All three MDX pages build and render correctly
- Content follows Diataxis Explanation + How-to principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Workspace resolution algorithm is accurately documented
- Config overlay merge semantics match `internal/config/merge.go`
- "Copy the directory" portability story is reinforced
- Zero build warnings from Fumadocs
