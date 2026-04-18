---
status: pending
title: Rewrite Network domain (channels + peers)
type: frontend
complexity: high
dependencies:
  - task_13
  - task_14
---

# Task 23: Rewrite Network domain (channels + peers)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/systems/network/**` as a pure visual refresh over `@agh/ui` primitives. The `/network` route exposes two tabs (Channels / Peers); each is a split-pane view with a searchable list on the left and a rich detail panel on the right. Channel detail surfaces the wire trace table, member peers, and message log; peer detail surfaces capabilities, joined channels, and stats. All TanStack Query hooks, adapters, stores, and MSW fixtures stay untouched — only visual chrome changes.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite every component under `web/src/systems/network/components/**` and the route `web/src/routes/_app/network.tsx` using only `@agh/ui` primitives plus domain code.
- MUST compose the page as `PageHeader` (title + count + `Pills` tab switcher + primary action) above a `SplitPane` whose `list` and `detail` slots swap per active tab.
- MUST render the channel detail as a stack of `Section` blocks: wire trace `Table`, member peers list with `StatusDot` + `MonoBadge`, and message log with `CodeBlock` payloads plus `KindChip` for protocol kinds.
- MUST render the peer detail as `Section` blocks for capabilities (`KindChip` list), joined channels (`Table`), and stats (`Metric` strip).
- MUST use `SearchInput` for list filtering and `Empty` for every no-results / disabled / error state — no ad hoc empty markup.
- MUST preserve existing behavior from `useNetworkPage`, `useNetworkChannels*`, `useNetworkPeers*` hooks, query keys, and stores — props passed to components may adjust shape but data sources stay identical.
- MUST NOT import from `@/components/ui/*` or `@/components/design-system/*` — both folders are deleted after Phase 2.
- MUST use Lucide icons at DESIGN.md §3 stroke + size conventions; `StatusDot` drives every status signal.
- SHOULD extract per-view layouts (channels vs peers) into sibling components instead of branching inside one giant panel.
</requirements>

## Subtasks

- [ ] 23.1 Audit current components + `useNetworkPage` view-model to map every prop, state branch, and test id that must survive the rewrite.
- [ ] 23.2 Rewrite the route `network.tsx` around `PageHeader` + `Pills` + `SplitPane`, removing `WorkspacePageShell`, `PillButton`, `MetricStrip`, and every `@/components/design-system` import.
- [ ] 23.3 Rewrite `network-channels-list-panel.tsx` and `network-peers-list-panel.tsx` as `SearchInput` + list rows using `StatusDot` + `MonoBadge` + `KindChip`, with `Empty` states for no-results and errors.
- [ ] 23.4 Rewrite `network-channel-detail-panel.tsx` with three `Section` blocks (wire trace, members, messages), `Table` for wire trace, `CodeBlock` for payloads.
- [ ] 23.5 Rewrite `network-peer-detail-panel.tsx` with `Section` blocks for capabilities, channels, and stats (`Metric` row).
- [ ] 23.6 Rewrite `network-create-channel-dialog.tsx` on the `@agh/ui` `Dialog` + `Field` primitives and `network-empty-state.tsx` on `Empty`.
- [ ] 23.7 Update or rewrite `web/src/systems/network/components/stories/**` and generate Playwright visual baselines covering default / loading / error / empty / network-disabled states for both tabs.

## Implementation Details

See TechSpec §"Impact Analysis" — `web/src/systems/network/**` is a Phase 5 visual rewrite. Component subdivisions stay one-file-per-panel. DESIGN.md §4 and §5 govern the visual spec (Pills, StatusDot, KindChip, MonoBadge). Route layout is driven by `SplitPane` with the `detailEmpty` slot reserved for "no selection" states.

### Relevant Files

- `web/src/routes/_app/network.tsx` — rewrite target; drop `WorkspacePageShell` + `PillButton` + `MetricStrip`.
- `web/src/systems/network/components/network-channels-list-panel.tsx` — rewrite on `SearchInput` + list rows.
- `web/src/systems/network/components/network-channel-detail-panel.tsx` — rewrite on `Section` + `Table` + `CodeBlock`.
- `web/src/systems/network/components/network-peers-list-panel.tsx` — rewrite on `SearchInput` + list rows.
- `web/src/systems/network/components/network-peer-detail-panel.tsx` — rewrite on `Section` + `Metric` + `KindChip` list.
- `web/src/systems/network/components/network-create-channel-dialog.tsx` — rewrite on `Dialog` + `Field`.
- `web/src/systems/network/components/network-empty-state.tsx` — rewrite on `Empty`.

### Dependent Files

- `web/src/hooks/routes/use-network-page.ts` — view-model stays; only the shape of props handed to panels may change.
- `web/src/systems/network/index.ts` — public barrel; update exports if panel modules split.
- `web/src/systems/network/components/stories/**` — stories rewritten against new primitives.
- `web/e2e/**` Playwright suites referencing `data-testid="network-*"` — test ids MUST survive.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout](adrs/adr-004.md)
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `web/src/systems/network/**` components consuming only `@agh/ui` + domain code.
- Rewritten `web/src/routes/_app/network.tsx` wired to `PageHeader` + `SplitPane`.
- Updated Storybook stories for every component under `components/stories/**`.
- Playwright visual snapshot baselines for `/network` covering: channels default, channels empty, channels loading, channels error, peers default, peers empty, network disabled **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for tab switch, row select, and search filter **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Rendering `NetworkChannelsListPanel` with an empty `channels` array renders the `Empty` no-results state when `searchQuery` is non-empty.
  - [ ] Typing into `SearchInput` calls `onSearchChange` with the typed value.
  - [ ] Selecting a channel row calls `onSelectChannel` with the channel id and marks the row active via `data-state="selected"`.
  - [ ] `NetworkChannelDetailPanel` with a channel and non-empty messages renders one `CodeBlock` per payload and one `KindChip` per protocol kind.
  - [ ] `NetworkChannelDetailPanel` with `isLoading=true` renders the loading skeleton in every `Section`, not the final content.
  - [ ] `NetworkPeerDetailPanel` with stats renders exactly one `Metric` per stat key and one `KindChip` per capability.
  - [ ] `NetworkCreateChannelDialog` with `canSubmit=false` renders the submit button disabled and does not call `onSubmit` on click.
- Integration tests:
  - [ ] Storybook `play()` switches Channels → Peers via the `Pills` tab and asserts the list panel test ids swap.
  - [ ] Storybook `play()` selects a channel in the list and asserts the detail panel renders the selected channel's wire trace `Table` rows.
  - [ ] Storybook `play()` with the network-disabled fixture renders the `Empty` disabled state and never mounts the split pane.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Zero imports from `@/components/ui/*` or `@/components/design-system/*` anywhere under `web/src/systems/network/**` or `web/src/routes/_app/network.tsx`.
- Playwright baseline snapshots committed for the seven listed states.
- Every `data-testid="network-*"` referenced by existing Playwright e2e specs still resolves.
- `make verify` and `make web-lint` pass.
