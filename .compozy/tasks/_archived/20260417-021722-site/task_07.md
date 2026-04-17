---
status: completed
title: "Docs: Getting Started (Install, Quick Start, First Agent, Web UI)"
type: docs
complexity: medium
dependencies:
    - task_03
---

# Task 07: Docs: Getting Started (Install, Quick Start, First Agent, Web UI)

## Overview

Write the four Getting Started tutorial pages that guide new users from zero to a running AGH daemon with their first agent session. These pages are Diataxis **Tutorial** type — they walk the user through a concrete learning experience with clear steps, expected output, and a sense of accomplishment at each milestone. This is the primary onboarding path.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Getting Started"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — all pages MUST follow Diataxis Tutorial principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure and content placement — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Tutorials MUST be tested end-to-end — every command shown must actually work
- Include expected CLI output after each command so users can verify they are on track
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/getting-started/installation.mdx` — install methods (go install, brew, binary download), prerequisites, verify installation
- MUST create `packages/site/content/runtime/getting-started/quick-start.mdx` — start daemon, create first session, send a message, observe events — under 5 minutes
- MUST create `packages/site/content/runtime/getting-started/first-agent.mdx` — create AGENT.md, configure a provider (Claude Code), customize system prompt, run a session with the custom agent
- MUST create `packages/site/content/runtime/getting-started/web-ui.mdx` — open web UI, navigate sessions, view events, understand the interface
- MUST create `packages/site/content/runtime/getting-started/meta.json` — sidebar ordering
- MUST follow Diataxis Tutorial principles: learning-oriented, concrete steps, expected outcomes, minimal explanation
- MUST include code blocks with shell commands and expected output
- MUST include prerequisites section in installation page (Go version, OS support)
- SHOULD include callout boxes (Fumadocs Callout component) for common pitfalls
- SHOULD end each page with a "Next steps" section linking to the next tutorial page
</requirements>

## Subtasks

- [x] 7.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [x] 7.2 Read source files (README.md, internal/cli/\*.go, internal/daemon/) to gather accurate CLI commands and flags
- [x] 7.3 Write `installation.mdx` — prerequisites, install methods, verification
- [x] 7.4 Write `quick-start.mdx` — daemon start, first session, message, events
- [x] 7.5 Write `first-agent.mdx` — AGENT.md creation, provider config, custom session
- [x] 7.6 Write `web-ui.mdx` — open UI, navigate, view events
- [x] 7.7 Create `meta.json` for sidebar ordering: installation → quick-start → first-agent → web-ui
- [x] 7.8 Verify all pages build without errors

## Implementation Details

See TechSpec sections: "Appendix B: Documentation Taxonomy — Getting Started".

The Getting Started section is the most critical path for user adoption. Each page should take no more than 5 minutes to complete. The progression is: install AGH binary → start the daemon → create a session with a built-in agent → customize with AGENT.md → explore the web UI. Every CLI command must reflect the actual command structure from `internal/cli/`.

### Relevant Files

- `README.md` — Current quick start instructions (adapt, don't copy)
- `internal/cli/root.go` — Root command and global flags
- `internal/cli/daemon.go` — `agh daemon start` command
- `internal/cli/session.go` — `agh session create`, `agh session list` commands
- `internal/cli/agent.go` — Agent-related CLI commands
- `internal/config/config.go` — Default config location, TOML structure
- `internal/daemon/daemon.go` — Daemon boot sequence
- `internal/api/httpapi/` — Web UI HTTP server

### Dependent Files

- `packages/site/content/runtime/getting-started/` — Output directory
- Overview pages (task_06) — "What is AGH" is conceptual prerequisite reading

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/getting-started/installation.mdx` — Installation tutorial
- `packages/site/content/runtime/getting-started/quick-start.mdx` — 5-minute quick start
- `packages/site/content/runtime/getting-started/first-agent.mdx` — Custom agent tutorial
- `packages/site/content/runtime/getting-started/web-ui.mdx` — Web UI walkthrough
- `packages/site/content/runtime/getting-started/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All four pages render at their expected URLs under `/runtime/getting-started/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] All CLI commands shown match actual `internal/cli/` command structure
  - [ ] Each page includes expected output after commands
  - [ ] Installation page covers at least 2 install methods
  - [ ] Quick start page is completable in under 5 minutes of reading
  - [ ] No broken internal links

## Success Criteria

- All four MDX pages build and render correctly
- Content follows Diataxis Tutorial principles (verified by `/documentation-writer` + `/copywriting` workflow)
- CLI commands are verified against actual codebase command structure
- Each page has a clear "Next steps" section
- Zero build warnings from Fumadocs
