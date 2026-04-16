---
status: pending
title: "Docs: Bridges (Platform Adapters, Routing, Setup)"
type: docs
complexity: medium
dependencies:
  - task_03
---

# Task 14: Docs: Bridges (Platform Adapters, Routing, Setup)

## Overview

Write the three bridge documentation pages covering AGH's platform adapter system that connects agents to external messaging platforms like Slack, Discord, and Telegram. Bridges allow users to interact with AGH agents through their existing communication tools. These pages mix Diataxis **Explanation** (how bridges work) and **How-to** (how to set up and configure bridges).

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Bridges"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — pages MUST follow Diataxis Explanation + How-to principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to `internal/bridges/` — read the source for adapter types, routing, delivery
- Include step-by-step platform setup guides with screenshots or clear instructions where applicable
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/bridges/overview.mdx` — what bridges are, bridge instance model, adapter architecture, supported platforms, message flow
- MUST create `packages/site/content/runtime/bridges/routing.mdx` — routing policy, message routing from platform to agent session, delivery guarantees, error handling
- MUST create `packages/site/content/runtime/bridges/setup.mdx` — step-by-step setup for each supported platform (Slack, Discord, Telegram), bot token configuration, channel mapping
- MUST create `packages/site/content/runtime/bridges/meta.json` — sidebar ordering
- MUST document the bridge instance configuration format
- MUST include the message flow from platform → bridge → session → response → platform
- MUST document each supported platform's setup requirements (API tokens, bot permissions, etc.)
- MUST read `internal/bridges/` source code for implementation accuracy
- SHOULD include a Mermaid sequence diagram showing the bridge message flow
- SHOULD document error handling and retry behavior for message delivery
</requirements>

## Subtasks

- [ ] 14.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 14.2 Read `internal/bridges/` — adapter types, routing, delivery, platform-specific adapters
- [ ] 14.3 Read relevant CLI commands for bridge management
- [ ] 14.4 Write `overview.mdx` — bridge system explanation with architecture diagram
- [ ] 14.5 Write `routing.mdx` — routing policy and delivery explanation
- [ ] 14.6 Write `setup.mdx` — platform-by-platform setup guides
- [ ] 14.7 Create `meta.json` for sidebar ordering: overview → routing → setup
- [ ] 14.8 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Bridges".

Bridges are AGH's platform adapter layer. Each bridge instance connects to an external messaging platform and routes messages to/from agent sessions. The routing policy determines which agent and session handle incoming messages. Delivery guarantees and error handling ensure reliable communication across the platform boundary.

### Relevant Files

- `internal/bridges/bridge.go` — Bridge instance type, lifecycle
- `internal/bridges/adapter.go` — Adapter interface, platform abstraction
- `internal/bridges/routing.go` — Routing policy, session resolution
- `internal/bridges/slack/` — Slack adapter (if exists as subdirectory)
- `internal/bridges/discord/` — Discord adapter (if exists as subdirectory)
- `internal/bridges/telegram/` — Telegram adapter (if exists as subdirectory)
- `internal/config/config.go` — Bridge configuration fields
- `internal/cli/bridge.go` — Bridge CLI commands (if exists)

### Dependent Files

- `packages/site/content/runtime/bridges/` — Output directory
- Sessions pages (task_08) — Bridge-initiated sessions follow same lifecycle
- Configuration Reference (task_17) — Bridge config cross-referenced

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/bridges/overview.mdx` — Bridge system overview
- `packages/site/content/runtime/bridges/routing.mdx` — Routing and delivery documentation
- `packages/site/content/runtime/bridges/setup.mdx` — Platform setup guides
- `packages/site/content/runtime/bridges/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All three pages render at their expected URLs under `/runtime/bridges/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] All supported platforms are documented with setup steps
  - [ ] Message flow diagram accurately reflects bridge architecture
  - [ ] Routing policy documentation matches implementation
  - [ ] Platform-specific requirements (tokens, permissions) are documented
  - [ ] No broken internal links

## Success Criteria

- All three MDX pages build and render correctly
- Content follows Diataxis Explanation + How-to principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Bridge architecture and routing are accurately documented
- Each supported platform has a complete setup guide
- Message flow is clearly explained with diagrams
- Zero build warnings from Fumadocs
