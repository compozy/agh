---
status: pending
title: Web Network Shell, Routes, Channel-Pivot IA & Query Isolation
type: frontend
complexity: critical
dependencies:
  - task_08
---

# Task 13: Web Network Shell, Routes, Channel-Pivot IA & Query Isolation

## Overview

Build the foundation of the new `/network` web experience: the route tree, the channel-pivot information architecture (channels rail with cross-channel Recents, channel header with Threads / Directs / Activity tabs), and the TanStack Query layer with strict per-surface key isolation. This task delivers the empty shell — no message rendering, no composer, no work surfacing yet. Those land in tasks 14 and 15.

The normative UI source is `_design.md` in this directory. Sections referenced below: §2 (decisions), §3 (layout), §4 (information architecture), §11 (component → file map). Where `_design.md` and `_techspec.md` overlap, `_techspec.md` wins for protocol/data semantics; `_design.md` wins for layout, interaction, visual treatment.

<critical>
- ALWAYS READ `_techspec.md`, `_design.md`, all ADRs, `web/CLAUDE.md`, and `DESIGN.md` before editing.
- ACTIVATE `agh-design`, `design-taste-frontend`, `minimalist-ui`, `react`, `tanstack-router-best-practices`, `tanstack-query-best-practices`, `vitest`, `testing-anti-patterns`, and `app-renderer-systems`.
- REFERENCE `_design.md` §3 (shell architecture), §4 (information architecture), §6 (chromatic discipline), §11 (component → file map and routes).
- REFERENCE `_techspec.md:1113-1119` for the exact route map and `_techspec.md:1124` for query key composition.
- FOCUS ON shell, routes, IA, and the query/cache layer. Do not implement message rows, composers, or work surfacing in this task.
- TESTS REQUIRED for route registration, query-key shape, query isolation across surfaces, channel rail behavior, tab navigation, and Recents merge.
- NO WORKAROUNDS: query keys must include `[network, channel, surface, containerId]`. Threads tab queries must never be readable from the directs tab cache, and vice versa.
</critical>

<requirements>
- MUST register the five route files prescribed by `_techspec.md:1113-1119` plus the Activity tab route from `_design.md` §11.4 (`network.$channel.activity.tsx`).
- MUST regenerate `web/src/routeTree.gen.ts` through the project's route generator; never hand-edit generated output.
- MUST place network components under `web/src/systems/network/components/{shell,...}/` per `_design.md` §11.2; this task owns the `shell/` subtree only.
- MUST implement the L2 channel rail with: `Recents` section (cross-channel, max 5, computed from summary `last_activity_at`), `Channels` section (alphabetical or pinned-first), and rail row anatomy from `_design.md` §4.3.
- MUST implement the L3 channel header with name, identity-mix chip (e.g. `3 agents · 1 human`), kebab menu, and the tab strip from `_design.md` §5.1.
- MUST implement tab navigation as real TanStack route navigations (Links), not internal state.
- MUST include `channel`, `surface`, and container ID in every relevant query key (`_techspec.md:1124`).
- MUST keep threads-tab and directs-tab caches isolated; a directs query must never satisfy a threads query and vice versa.
- MUST replace the existing `network_selected_peer` browser artifact field with `network_selected_thread` and `network_selected_direct` (`_techspec.md:1130`); browser artifact capture wiring lives here, the timeline-side artifact emission lands in task_14.
- MUST adhere to the chromatic discipline rules in `_design.md` §6 — default mono, tint over solid, no shadows.
- MUST keep responsive collapse semantics for `<1280px` (rail collapses to icon-only) and `<1024px` (rail becomes a Sheet) per `_design.md` §3.3.
- MUST avoid settings controls for unsupported config lifecycle features per `_design.md` §10 row A8.
</requirements>

## Subtasks

- [ ] 13.1 Update web generated-contract consumers, network types, API adapters, query keys, and query options for thread/direct/work read paths.
- [ ] 13.2 Register the six route files (`network.tsx`, `network.$channel.threads.tsx`, `network.$channel.threads.$threadId.tsx`, `network.$channel.directs.tsx`, `network.$channel.directs.$directId.tsx`, `network.$channel.activity.tsx`) and regenerate the route tree.
- [ ] 13.3 Build `web/src/systems/network/components/shell/` (`network-shell.tsx`, `channel-rail.tsx`, `channel-rail-recents.tsx`, `channel-rail-row.tsx`, `channel-header.tsx`, `channel-tabs.tsx`, `right-rail.tsx` skeleton).
- [ ] 13.4 Implement `use-network-page.ts` orchestration hook (replaces existing flat one), `use-channels.ts`, `use-recents.ts`, and `use-last-read.ts` (localStorage tracking).
- [ ] 13.5 Wire browser artifact capture for `network_selected_thread` / `network_selected_direct`; remove `network_selected_peer` capture.
- [ ] 13.6 Update settings network impact only for real aggregate metrics, with tests proving unsupported controls are absent (per `_design.md` §10 row A8).

## Implementation Details

The Activity tab is a design addition (not techspec-mandated). It is a unified reverse-chronological feed across both surfaces in the active channel, derived from `last_activity_at` on summary rows. No new endpoints required.

The right-rail container is registered as a slot in this task; thread overlay content (`thread-overlay/`) is implemented in task_14, work inspector content (`work/`) is implemented in task_15.

### Relevant Files

- `web/src/routes/_app/network.tsx` - layout/shell + redirect to first channel.
- `web/src/routes/_app/network.$channel.threads.tsx` - threads tab list.
- `web/src/routes/_app/network.$channel.threads.$threadId.tsx` - thread detail (rendered as overlay or full-page based on viewport; this task registers the route only).
- `web/src/routes/_app/network.$channel.directs.tsx` - directs tab list.
- `web/src/routes/_app/network.$channel.directs.$directId.tsx` - direct detail.
- `web/src/routes/_app/network.$channel.activity.tsx` - activity tab (design addition).
- `web/src/routeTree.gen.ts` - generated route tree.
- `web/src/systems/network/components/shell/*` - shell components.
- `web/src/systems/network/hooks/use-network-page.ts` - route orchestration.
- `web/src/systems/network/hooks/use-channels.ts` - channels data hook.
- `web/src/systems/network/hooks/use-recents.ts` - cross-channel recents merge.
- `web/src/systems/network/hooks/use-last-read.ts` - localStorage last-read tracking.
- `web/src/systems/network/lib/query-keys.ts` - cache key factory.
- `web/src/systems/network/lib/query-options.ts` - queryOptions for threads/directs/messages.
- `web/src/systems/network/lib/palette.ts` - identity-seeded avatar tints (extracted from current shell; consider moving to `@agh/ui` per `_design.md` §14.1).
- `web/src/systems/network/types.ts` - web network models.
- `web/src/systems/network/adapters/network-api.ts` - API adapter.

### Dependent Files

- `web/src/generated/agh-openapi.d.ts` - generated in task_08.
- `web/src/routes/_app/settings/network.tsx` - settings impact check (no unsupported controls).
- `web/src/routes/_app/settings/-network.test.tsx` - unsupported control assertions.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - UI navigation model.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - tab structure.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: web should display server truth from contract APIs, not invent client-only conversation state.
- Agent manageability: the shell must reflect routes that agents can also drive via CLI/HTTP/UDS.
- Config lifecycle: no new settings controls for thread retention, unread sync, notifications, or new network keys.

### Web/Docs Impact

- Web impact: this task owns the shell, route registration, IA, and query/cache foundation.
- Docs impact: task_16 (renumbered) documents the resulting shell IA and supported controls.

## Deliverables

- Six registered route files with regenerated route tree.
- `shell/` component subtree (rail, header, tabs, right-rail slot).
- Channels and Recents data hooks with strict query-key isolation.
- Browser artifact capture for `network_selected_thread` / `network_selected_direct`.
- Tests proving settings does not expose unsupported controls.

## Tests

- Unit tests:
  - [ ] All six routes register and resolve to correct components.
  - [ ] Query keys include `channel`, `surface`, and container ID.
  - [ ] Threads-tab queries do not satisfy directs-tab queries (cache isolation proof).
  - [ ] Recents merges thread + direct summaries across channels and caps at 5.
  - [ ] Last-read tracking persists per `[channel, surface, containerId]` and resets on visit.
  - [ ] Channel rail renders pinned channels first, alphabetical thereafter.
  - [ ] Tab change is a real route navigation, not internal state (URL changes).
  - [ ] Settings route does not render unsupported thread retention, unread sync, notification, or config controls.
- Integration tests:
  - [ ] Route tree generates cleanly with the project's route generator.
  - [ ] Storybook scenarios cover empty / loading / populated / disabled states for the shell, channel rail, and channel header.
  - [ ] MSW fixtures cover `/api/network/channels` and the thread/direct summary endpoints used by Recents.
- Test coverage target: >=80% for touched web network shell modules.
- All tests must pass.

## Success Criteria

- The web network shell is route-driven and query-isolated.
- Channels rail surfaces cross-channel Recents without merging cache keys.
- Tab navigation reflects URL truth at all times.
- Browser artifacts no longer expose `network_selected_peer`.
