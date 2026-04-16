---
status: completed
title: "Docs: Automation (Jobs, Triggers, Webhooks, Runs)"
type: docs
complexity: medium
dependencies:
    - task_03
---

# Task 13: Docs: Automation (Jobs, Triggers, Webhooks, Runs)

## Overview

Write the three automation documentation pages covering AGH's built-in job scheduling, trigger system, and webhook endpoints. Automation enables agents to run on schedules, react to events, and be triggered by external systems — making AGH useful for continuous and unattended agent workflows. These pages mix Diataxis **Explanation** (how automation works) and **How-to** (how to set up jobs and triggers).

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Automation"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — pages MUST follow Diataxis Explanation + How-to principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Content MUST be accurate to `internal/automation/` — read the source for schedule modes, trigger types, retry policies
- Include practical examples for common automation patterns (daily code review, CI integration, etc.)
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/automation/jobs.mdx` — job definition, 3 schedule modes (cron, every, at), job configuration, run history, retry policies
- MUST create `packages/site/content/runtime/automation/triggers.mdx` — trigger types, event filters, trigger-to-job binding, conditional execution
- MUST create `packages/site/content/runtime/automation/webhooks.mdx` — webhook endpoint setup, authentication, payload formats, external system integration
- MUST create `packages/site/content/runtime/automation/meta.json` — sidebar ordering
- MUST document all 3 schedule modes with cron syntax examples
- MUST document retry policy configuration with all available options
- MUST include at least 2 practical automation examples (e.g., scheduled code review, webhook-triggered deployment)
- MUST read `internal/automation/` source code for implementation accuracy
- SHOULD include a diagram showing the automation execution flow
- SHOULD document run status tracking and how to monitor job runs
</requirements>

## Subtasks

- [ ] 13.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 13.2 Read `internal/automation/` — jobs, triggers, webhooks, run tracking, retry policies
- [ ] 13.3 Read relevant CLI commands for automation management
- [ ] 13.4 Write `jobs.mdx` — job definition and schedule modes with examples
- [ ] 13.5 Write `triggers.mdx` — trigger types and event filters
- [ ] 13.6 Write `webhooks.mdx` — webhook setup and external integration
- [ ] 13.7 Create `meta.json` for sidebar ordering: jobs → triggers → webhooks
- [ ] 13.8 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Automation".

Automation in AGH allows agents to be scheduled and triggered without human intervention. Jobs define what agent to run and when, with three schedule modes: cron (standard cron expressions), every (interval-based), and at (one-time future execution). Triggers react to events within AGH and can conditionally start jobs. Webhooks expose HTTP endpoints that external systems can call to trigger agent runs.

### Relevant Files

- `internal/automation/job.go` — Job definition, schedule modes, configuration
- `internal/automation/trigger.go` — Trigger types, event filters, bindings
- `internal/automation/webhook.go` — Webhook endpoint handling
- `internal/automation/run.go` — Run tracking, status, retry logic
- `internal/automation/scheduler.go` — Schedule execution engine
- `internal/cli/automation.go` — Automation CLI commands (if exists)
- `internal/cli/job.go` — Job CLI commands (if exists)
- `internal/api/httpapi/` — Webhook HTTP endpoints

### Dependent Files

- `packages/site/content/runtime/automation/` — Output directory
- Sessions pages (task_08) — Automated sessions follow same lifecycle
- Hooks pages (task_15) — Triggers may relate to hook events

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/automation/jobs.mdx` — Jobs and scheduling documentation
- `packages/site/content/runtime/automation/triggers.mdx` — Triggers documentation
- `packages/site/content/runtime/automation/webhooks.mdx` — Webhooks documentation
- `packages/site/content/runtime/automation/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All three pages render at their expected URLs under `/runtime/automation/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] All 3 schedule modes documented with syntax examples
  - [ ] Retry policy options accurately reflect implementation
  - [ ] At least 2 practical automation examples included
  - [ ] Webhook authentication and payload format documented
  - [ ] No broken internal links

## Success Criteria

- All three MDX pages build and render correctly
- Content follows Diataxis Explanation + How-to principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Schedule modes and syntax are accurate to `internal/automation/` implementation
- Retry policies are fully documented
- Practical examples are realistic and actionable
- Zero build warnings from Fumadocs
