---
status: completed
title: "Docs: Sessions (Lifecycle, Resume, Events, Permissions)"
type: docs
complexity: medium
dependencies:
    - task_03
---

# Task 08: Docs: Sessions (Lifecycle, Resume, Events, Permissions)

## Overview

Write the four session documentation pages covering the core runtime unit of AGH. Sessions are the central abstraction — they represent a running agent interaction with a defined lifecycle, event stream, and permission model. These pages mix Diataxis **Explanation** (how sessions work) and **How-to** (how to manage sessions). This is essential reading for any AGH user.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Sessions"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — pages MUST follow Diataxis Explanation + How-to principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to the session state machine in `internal/session/` — read the source
- Include Mermaid state diagrams for the session lifecycle
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/sessions/lifecycle.mdx` — state machine (starting → active → stopping → stopped), transitions, timeout behavior, Mermaid state diagram
- MUST create `packages/site/content/runtime/sessions/resume.mdx` — session resume and replay, transcript reconstruction from events, when and how to resume
- MUST create `packages/site/content/runtime/sessions/events.mdx` — event streaming via SSE, event types, subscribing to session events, event persistence in SQLite
- MUST create `packages/site/content/runtime/sessions/permissions.mdx` — permission modes, approval flow, how permissions are enforced during session execution
- MUST create `packages/site/content/runtime/sessions/meta.json` — sidebar ordering
- MUST include a Mermaid state diagram in lifecycle.mdx showing all session states and transitions
- MUST document all session event types with their payload structures
- MUST read `internal/session/` source code to ensure accuracy of state machine documentation
- SHOULD include code examples showing CLI and HTTP API usage for each operation
- SHOULD include callouts for common gotchas (e.g., resume after crash vs. clean stop)
</requirements>

## Subtasks

- [ ] 8.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 8.2 Read `internal/session/` package thoroughly — state machine, manager, events, permissions
- [ ] 8.3 Read `internal/acp/` for ACP-level session interactions
- [ ] 8.4 Write `lifecycle.mdx` — state machine explanation with Mermaid diagram
- [ ] 8.5 Write `resume.mdx` — resume and replay how-to with examples
- [ ] 8.6 Write `events.mdx` — event streaming explanation with type catalog
- [ ] 8.7 Write `permissions.mdx` — permission model explanation and configuration
- [ ] 8.8 Create `meta.json` for sidebar ordering: lifecycle → resume → events → permissions
- [ ] 8.9 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Sessions".

Sessions are AGH's core runtime unit. A session wraps an ACP-compatible agent subprocess with lifecycle management, event recording, and permission enforcement. The state machine (starting → active → stopping → stopped) governs all transitions. Events are persisted in per-session SQLite databases and streamed via SSE. The transcript package reconstructs message history for resume.

### Relevant Files

- `internal/session/session.go` — Session type, state machine, transitions
- `internal/session/manager.go` — Session manager, create/stop/resume operations
- `internal/session/manager_start.go` — Session start flow
- `internal/session/events.go` — Session event types and recording
- `internal/acp/client.go` — ACP client, JSON-RPC communication
- `internal/acp/process.go` — Subprocess spawn and lifecycle
- `internal/observe/` — Event recording, health metrics
- `internal/transcript/` — Transcript reconstruction for resume
- `internal/store/sessiondb/` — Per-session event persistence
- `internal/api/contract/contract.go` — Session contract types for API

### Dependent Files

- `packages/site/content/runtime/sessions/` — Output directory
- Getting Started pages (task_07) — Users arrive here after quick start

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/sessions/lifecycle.mdx` — Session lifecycle with state diagram
- `packages/site/content/runtime/sessions/resume.mdx` — Resume and replay how-to
- `packages/site/content/runtime/sessions/events.mdx` — Event streaming documentation
- `packages/site/content/runtime/sessions/permissions.mdx` — Permission model documentation
- `packages/site/content/runtime/sessions/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All four pages render at their expected URLs under `/runtime/sessions/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] Lifecycle page contains a Mermaid state diagram
  - [ ] State machine documentation matches actual states in `internal/session/`
  - [ ] Event types documented match actual event types in source code
  - [ ] Permission modes documented match actual implementation
  - [ ] No broken internal links

## Success Criteria

- All four MDX pages build and render correctly
- Content follows Diataxis Explanation + How-to principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Session state machine diagram is accurate to codebase
- Event type catalog is complete and accurate
- Permission modes are fully documented
- Zero build warnings from Fumadocs
