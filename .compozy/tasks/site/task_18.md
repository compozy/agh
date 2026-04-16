---
status: pending
title: "Docs: Operations (Daemon, Database, Troubleshooting, Production)"
type: docs
complexity: low
dependencies:
  - task_03
---

# Task 18: Docs: Operations (Daemon, Database, Troubleshooting, Production)

## Overview

Write the four operations documentation pages covering day-to-day AGH daemon management, database administration, troubleshooting common issues, and a production readiness checklist. These are Diataxis **How-to** type — practical, goal-oriented guides for operators who already understand AGH and need to solve specific operational problems.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Operations"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — all pages MUST follow Diataxis How-to principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- How-to pages must be action-oriented — "How to do X" not "X explained"
- Troubleshooting MUST cover real issues users will encounter
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/operations/daemon.mdx` — daemon start/stop/restart, log management, PID file, lock file, health check, running as systemd/launchd service
- MUST create `packages/site/content/runtime/operations/database.mdx` — SQLite database locations (global agh.db, per-session events.db), backup procedures, database inspection, cleanup old sessions
- MUST create `packages/site/content/runtime/operations/troubleshooting.mdx` — common issues with diagnosis and resolution: daemon won't start, agent fails to spawn, permission errors, socket issues, session stuck in state
- MUST create `packages/site/content/runtime/operations/production.mdx` — production readiness checklist: config hardening, log rotation, monitoring, backup strategy, resource limits
- MUST create `packages/site/content/runtime/operations/meta.json` — sidebar ordering
- MUST document the daemon lock file mechanism and how it prevents multiple instances
- MUST document SQLite database file locations and their purpose
- MUST include at least 5 troubleshooting entries with symptoms, diagnosis, and resolution
- MUST read `internal/daemon/` and `internal/store/` for implementation accuracy
- SHOULD include example systemd and launchd service configurations
- SHOULD document the UDS socket location and permissions
</requirements>

## Subtasks

- [ ] 18.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 18.2 Read `internal/daemon/` — boot sequence, lock file, shutdown, health
- [ ] 18.3 Read `internal/store/` — database locations, schema, cleanup
- [ ] 18.4 Read `internal/store/globaldb/` and `internal/store/sessiondb/` for storage details
- [ ] 18.5 Write `daemon.mdx` — daemon management how-to
- [ ] 18.6 Write `database.mdx` — database administration how-to
- [ ] 18.7 Write `troubleshooting.mdx` — common issues with solutions
- [ ] 18.8 Write `production.mdx` — production readiness checklist
- [ ] 18.9 Create `meta.json` for sidebar ordering: daemon → database → troubleshooting → production
- [ ] 18.10 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Operations".

Operations documentation serves users who have AGH running and need to manage it in practice. The daemon management page covers the lifecycle of the AGH process itself. The database page covers the SQLite databases (global catalog in agh.db, per-session event stores). Troubleshooting is a reference of known issues. The production checklist prepares AGH for persistent, unattended use.

### Relevant Files

- `internal/daemon/daemon.go` — Daemon boot, lock file, health check
- `internal/daemon/shutdown.go` — Graceful shutdown sequence
- `internal/store/store.go` — Shared SQLite helpers, schema
- `internal/store/globaldb/global_db.go` — Global database (agh.db)
- `internal/store/sessiondb/session_db.go` — Per-session database (events.db)
- `internal/logger/logger.go` — Log configuration, output locations
- `internal/config/paths.go` — File path resolution (home dir, socket, DB locations)
- `internal/api/udsapi/` — UDS server, socket path

### Dependent Files

- `packages/site/content/runtime/operations/` — Output directory
- Configuration Reference (task_17) — Config fields for daemon, logging, storage

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/operations/daemon.mdx` — Daemon management how-to
- `packages/site/content/runtime/operations/database.mdx` — Database administration how-to
- `packages/site/content/runtime/operations/troubleshooting.mdx` — Troubleshooting guide
- `packages/site/content/runtime/operations/production.mdx` — Production checklist
- `packages/site/content/runtime/operations/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All four pages render at their expected URLs under `/runtime/operations/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] Daemon management covers start, stop, restart, and service installation
  - [ ] Database page documents both agh.db and events.db with file locations
  - [ ] Troubleshooting has at least 5 entries with symptoms → diagnosis → resolution
  - [ ] Production checklist is actionable with clear pass/fail criteria
  - [ ] No broken internal links

## Success Criteria

- All four MDX pages build and render correctly
- Content follows Diataxis How-to principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Daemon management covers all lifecycle operations
- Database administration is practical and accurate
- Troubleshooting entries cover real issues users will encounter
- Production checklist is comprehensive
- Zero build warnings from Fumadocs
