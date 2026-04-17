---
status: pending
title: Skills, Automation, and Network summary pages
type: frontend
complexity: high
dependencies:
  - task_09
---

# Task 11: Skills, Automation, and Network summary pages

## Overview

Implement the settings pages that summarize configuration and runtime state for operational areas that already have dedicated product surfaces: `skills`, `automation`, and `network`. These pages should present settings and status cleanly while linking out to the deeper operational routes instead of duplicating management workflows inside settings.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Data Models", "Runtime apply matrix", and "Key Decisions"
- FOCUS ON "WHAT" — build summary-and-config pages, not replacement operational consoles
- MINIMIZE CODE — reuse existing visual patterns and deep-link to operational routes where the product already has them
- TESTS REQUIRED — route rendering, summary state, and operational links must be covered
- GREENFIELD: settings deve resumir e configurar; gerenciamento profundo continua nas telas operacionais existentes
</critical>

<requirements>
- MUST implement route pages for `skills`, `automation`, and `network`
- MUST render both config state and runtime summaries from the settings section envelopes
- MUST expose restart-required or applied-now feedback according to the runtime-apply matrix
- MUST provide clear links into the existing operational pages for deeper workflows
- MUST preserve the distinction between settings mutations and operational actions
- SHOULD reuse existing design-system and section-shell components for consistency
</requirements>

## Design References

| Screen | Local export | Paper artboard (node id) |
|--------|--------------|--------------------------|
| Skills | `docs/design/paper/settings/AGH Settings — Skills@2x.png` | `AGH Settings — Skills` (`ZDO-0`) |
| Automation | `docs/design/paper/settings/AGH Settings — Automation@2x.png` | `AGH Settings — Automation` (`ZKZ-0`) |
| Network | `docs/design/paper/settings/AGH Settings — Network@2x.png` | `AGH Settings — Network` (`ZSA-0`) |

## Subtasks

- [ ] 11.1 Implement the `skills` settings route with disabled-skill and marketplace/policy state
- [ ] 11.2 Implement the `automation` settings route with engine config and manager summary
- [ ] 11.3 Implement the `network` settings route with config and runtime status summary
- [ ] 11.4 Add deep links from these settings pages to the existing operational routes
- [ ] 11.5 Add tests for save flow, applied-now/restart state, and operational link behavior

## Implementation Details

See TechSpec sections "Data Models", "Runtime apply matrix", and "Key Decisions". Follow the existing `automation`, `network`, and `skills` page language from the app, but keep these settings routes focused on configuration and summary state rather than recreating those operational surfaces.

### Relevant Files

- `web/src/routes/_app/settings/skills.tsx` — new settings route for skills
- `web/src/routes/_app/settings/automation.tsx` — new settings route for automation
- `web/src/routes/_app/settings/network.tsx` — new settings route for network
- `web/src/routes/_app/skills.tsx` — existing operational destination for deep-link behavior
- `web/src/routes/_app/automation.tsx` — existing operational destination for deep-link behavior
- `web/src/routes/_app/network.tsx` — existing operational destination for deep-link behavior

### Dependent Files

- `web/src/systems/settings/components/` — likely shared summary cards, warning banners, and navigation helpers used by these pages
- `web/src/routes/_app/-settings*.test.tsx` — should add route coverage for these three sections
- `web/src/hooks/routes/use-settings-page.ts` — should provide page orchestration and active-section state
- `web/src/routeTree.gen.ts` — may update if route files change in this task

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Keeps settings separate from operational pages
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Defines applied-now versus restart-required behavior and linked-out workflows

## Deliverables

- `skills`, `automation`, and `network` settings routes with config and runtime summaries
- Deep links to the existing operational pages for each area
- Applied-now and restart-required UI for the supported mutations **(REQUIRED)**
- Route and interaction tests with >=80% coverage for modified page logic **(REQUIRED)**
- Verified separation between settings behavior and operational workflows **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] `skills` page surfaces disabled-skill changes and distinguishes `applied_now` from `restart_required`
  - [ ] `automation` page renders manager summary and restart-required save results correctly
  - [ ] `network` page renders runtime status summary and restart-required behavior correctly
  - [ ] Deep-link controls point to the expected operational routes
- Integration tests:
  - [ ] Navigating from each summary page to its operational route works inside the app shell
  - [ ] Saves on these pages invalidate and refetch the matching section queries without mutating unrelated sections
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for the route and component logic touched by these pages
- Users can configure skills, automation, and network settings without losing access to the existing operational screens
- Settings pages communicate clearly when a change applies now versus after restart
