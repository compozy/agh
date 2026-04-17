---
status: completed
title: "Docs: CLI Reference (auto-generated + editorial)"
type: docs
complexity: low
dependencies:
    - task_04
---

# Task 16: Docs: CLI Reference (auto-generated + editorial)

## Overview

Generate the CLI reference documentation from Cobra's GenMarkdownTree output and enhance it with an editorial overview page. The auto-generated pages provide complete command reference; the editorial page adds a command tree overview, common usage patterns, and navigational context. This task depends on task_04 (CLI doc generation infrastructure) being complete.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — CLI Reference"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — editorial page MUST follow Diataxis Reference principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Auto-generated content is overwritten on every `make cli-docs` run — do NOT edit generated files
- Editorial content goes in a separate index.mdx that will NOT be overwritten
- Each page MUST include proper frontmatter (`title`, `description`)
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST run `make cli-docs` to generate base CLI reference MDX files
- MUST create `packages/site/content/runtime/reference/cli/index.mdx` — editorial overview with full command tree, usage patterns, global flags, environment variables
- MUST review auto-generated pages for completeness and accuracy
- MUST create `packages/site/content/runtime/reference/cli/meta.json` — sidebar ordering with index first, then command groups
- MUST enhance generated pages with additional examples where the auto-generated output is bare
- MUST document global flags (--config, --log-level, --socket, etc.) in the overview page
- MUST document common usage patterns: "Start daemon + create session", "Resume a session", "Manage workspaces"
- SHOULD add a visual command tree showing the full CLI hierarchy
- SHOULD include shell completion setup instructions
</requirements>

## Subtasks

- [ ] 16.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 16.2 Run `make cli-docs` to generate base reference files
- [ ] 16.3 Review all generated files for completeness and accuracy
- [ ] 16.4 Write `index.mdx` — editorial overview with command tree and usage patterns
- [ ] 16.5 Add usage examples to generated pages where content is bare
- [ ] 16.6 Create `meta.json` for sidebar ordering
- [ ] 16.7 Verify all pages build without errors

## Implementation Details

See TechSpec sections: "Integration Points — Go Binary — CLI Reference Generation", "Appendix B: Documentation Taxonomy — CLI Reference".

The `make cli-docs` target runs `doc.GenMarkdownTree(rootCmd, outputDir)` which produces one markdown file per command. A post-processing script adds Fumadocs frontmatter and converts `.md` to `.mdx`. The editorial index.mdx is maintained separately and never overwritten by generation. All CLI commands are defined in `internal/cli/`.

### Relevant Files

- `internal/cli/root.go` — Root command, global flags
- `internal/cli/daemon.go` — Daemon commands
- `internal/cli/session.go` — Session commands
- `internal/cli/agent.go` — Agent commands
- `internal/cli/workspace.go` — Workspace commands
- `internal/cli/skill.go` — Skill commands
- `internal/cli/memory.go` — Memory commands
- `internal/cli/*.go` — All other CLI command files
- `cmd/agh/` — Main entry point, doc generation subcommand (from task_04)
- `Makefile` — `cli-docs` target (from task_04)

### Dependent Files

- `packages/site/content/runtime/reference/cli/` — Output directory for generated + editorial pages
- Task_04 deliverables — Post-processing script, `make cli-docs` target

### Related ADRs

- [ADR-004: Auto-Generated CLI and API Reference](adrs/adr-004.md) — Cobra GenMarkdownTree approach

## Deliverables

- Generated CLI reference MDX files (from `make cli-docs`)
- `packages/site/content/runtime/reference/cli/index.mdx` — Editorial overview page
- `packages/site/content/runtime/reference/cli/meta.json` — Sidebar ordering
- Enhanced examples added to generated pages where needed

## Tests

- Build verification:
  - [ ] `make cli-docs` completes without errors
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All CLI reference pages render at their expected URLs under `/runtime/reference/cli/`
- Content verification:
  - [ ] Each generated page has valid frontmatter (added by post-processing)
  - [ ] Editorial index page includes complete command tree
  - [ ] Global flags are documented
  - [ ] Common usage patterns are included with examples
  - [ ] No broken internal links
  - [ ] Re-running `make cli-docs` does not overwrite the editorial index.mdx

## Success Criteria

- CLI reference is generated and builds correctly
- Editorial overview provides useful navigational context
- Content follows Diataxis Reference principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Generated content is accurate to actual CLI commands
- Sidebar ordering is logical (overview first, then alphabetical/grouped commands)
- Zero build warnings from Fumadocs
