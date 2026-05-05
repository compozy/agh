---
status: completed
title: Web Composer, Work Surfacing, Empty/Error States & Realtime Polling
type: frontend
complexity: critical
dependencies:
  - task_14
---

# Task 15: Web Composer, Work Surfacing, Empty/Error States & Realtime Polling

## Overview

Close out the web network experience by wiring user input and lifecycle visibility on top of the shell (task_13) and timeline (task_14). This task delivers: the channel-level "New public thread" composer with collision-retry semantics, the detail composer for replies into the active container, slash commands (`/run`, `/mention`, `/attach`), optimistic mutations, the work lifecycle surfaces (inline chip, auto-hiding pinned banner, Work Inspector right-rail tab), all empty / error / disabled states, and the MVP realtime strategy (polling intervals + `refetchOnWindowFocus`).

The normative UI source is `_design.md`. Sections referenced below: §5.7 (composer), §5.8 (work surfacing), §6 (chromatic discipline — especially rules 6.6 and 6.7), §7 (state semantics), §9 (realtime strategy).

<critical>
- ALWAYS READ `_techspec.md`, `_design.md`, all ADRs, `web/CLAUDE.md`, and `DESIGN.md` before editing.
- ACTIVATE `agh-design`, `design-taste-frontend`, `minimalist-ui`, `react`, `tanstack-query-best-practices`, `vitest`, `testing-anti-patterns`, and `app-renderer-systems`.
- REFERENCE `_design.md` §5.7 (composer placement, slash commands, send affordance), §5.8 (three-layer work surfacing with auto-hide), §7 (empty / error / disabled state copy), §9 (polling intervals + optimistic mutation pattern).
- REFERENCE `_techspec.md:1126-1127` for thread-id collision retry and channel-level composer redirect semantics. `_techspec.md:502, 1333, 1343` for `claim_token` / `interaction_id` redaction. `_techspec.md:1332` for `kind:"direct"` rejection.
- FOCUS ON composer + work + states + realtime polling. The shell, routes, and timeline are owned by tasks 13/14.
- TESTS REQUIRED for composer collision retry, optimistic mutation rollback, work chip silence rules, banner auto-hide, polling intervals, all empty/error/disabled state copy.
- NO WORKAROUNDS: do not silently append to an existing thread on collision. Surface the error to the user after one silent retry per `_techspec.md:1127`.
</critical>

<requirements>
- MUST implement the channel-level "New public thread" composer per `_design.md` §5.7.1 — generates fresh `thread_id`, posts root `kind:"say"` with `surface:"thread"`, redirects to detail on success.
- MUST implement collision retry per `_techspec.md:1127` — exactly one silent retry on collision; second failure surfaces a single Sonner toast with the copy `Couldn't open this thread. Try again.` (per `_design.md` §5.7.1).
- MUST implement the detail composer per `_design.md` §5.7.2 — sends to the active container (`thread_id` or `direct_id` from URL).
- MUST implement slash command popover per `_design.md` §5.7.3 — `/run`, `/mention`, `/attach`. Out-of-MVP entries render disabled with `Post-MVP` tooltip.
- MUST implement optimistic mutations per `_design.md` §9.2 — message appears immediately, replaces with server canonical on success, renders `--color-danger-tint` with retry/discard inline on failure.
- MUST never construct a request body containing `kind:"direct"`, `interaction_id`, or raw `claim_token` (`_techspec.md:1332, 1333, 502, 1343`).
- MUST implement work surfacing in three layers per `_design.md` §5.8:
  - Inline chip on messages with `work_id` and state ∉ {`submitted`, `completed`}.
  - Pinned banner appearing only when `open_work_count > 0` for the active container; auto-hides within 400ms when count returns to 0; switches to solid `--color-warning` background when any work is in `needs_input`.
  - Work Inspector right-rail tab (alongside Members / Activity) showing open work entries with state badges, target peer, age, and "jump to message" links.
- MUST honor chromatic discipline rule 6.6 — silent for `submitted` and `completed`; tinted for `working` / `needs_input` / `failed`; tertiary text only for `canceled`.
- MUST implement empty / error / disabled states per `_design.md` §7 with the exact copy from §7.2.
- MUST implement realtime polling per `_design.md` §9.1: refetch-on-focus globally; interval polling (channels 30s, lists 15s, messages 5s while active route mounted, work entry 3s while inspector open with non-terminal state).
- MUST implement direct room resolution flow per `_design.md` §10 row A5: `[New direct]` action → `Combobox` peer picker → `POST /directs/resolve` → navigate on success.
- MUST keep the Send button as the only solid `--color-accent` surface in the entire network UI (`_design.md` §6.3).
</requirements>

## Subtasks

- [x] 15.1 Build `web/src/systems/network/components/composer/` (`composer.tsx` shared base, `channel-thread-composer.tsx`, `detail-composer.tsx`, `composer-toolbar.tsx`, `composer-slash-popover.tsx`).
- [x] 15.2 Implement `use-network-actions.ts` mutations: `sendMessage`, `createThread` (with collision retry), `resolveDirectRoom`. All mutations are optimistic.
- [x] 15.3 Build `web/src/systems/network/components/work/` (`work-chip.tsx`, `work-banner.tsx`, `work-inspector.tsx`, `work-inspector-row.tsx`).
- [x] 15.4 Implement `use-work.ts` — open work query for the active container; refetches every 3s while inspector is open and any state is non-terminal.
- [x] 15.5 Build `web/src/systems/network/components/empty-states/` (`network-empty.tsx`, `threads-empty.tsx`, `directs-empty.tsx`, `thread-empty.tsx`, `direct-empty.tsx`) with copy verbatim from `_design.md` §7.2.
- [x] 15.6 Implement error states per `_design.md` §7.3 — inline retry for list failures, in-place danger-tint for failed sends with retry/discard, single Sonner for thread collision second-failure, full-page error for daemon-down.
- [x] 15.7 Wire polling intervals and `refetchOnWindowFocus` defaults per `_design.md` §9.1; add manual refresh affordance to channel header kebab.
- [x] 15.8 Wire hover toolbar handlers from task_14 (Reply opens overlay, Pin/Fork as Post-MVP-disabled with tooltip, kebab opens menu).
- [x] 15.9 Add direct resolve flow: `[New direct]` button on Directs tab → `Combobox` peer picker → resolve → navigate.

## Implementation Details

Optimistic message IDs use `crypto.randomUUID()` (browser native). The server validates regardless; client UUID is purely a placeholder for cache replacement on commit (per `_design.md` §14.6).

The work chip live-ticking duration display (`working · 12s`) updates via a single shared `useElapsed` hook to avoid mounting per-message intervals — performance is critical when many messages render.

The pinned banner's `needs_input` solid escalation is the only solid warning surface in the entire UI. The send button is the only solid accent surface. The "New" divider line is the only chromatic divider. These are the three solid-color surfaces site-wide per `_design.md` §6.3.

### Relevant Files

- `web/src/systems/network/components/composer/*` - composer subtree.
- `web/src/systems/network/components/work/*` - work surfacing subtree.
- `web/src/systems/network/components/empty-states/*` - empty state subtree.
- `web/src/systems/network/hooks/use-network-actions.ts` - mutations.
- `web/src/systems/network/hooks/use-work.ts` - work query hook.
- `web/src/systems/network/lib/use-elapsed.ts` - shared interval for live duration display.
- `web/src/systems/network/lib/network-formatters.ts` - state label / duration / preview formatting.
- `web/src/routes/_app/network.$channel.threads.tsx` - mount channel-level composer.
- `web/src/routes/_app/network.$channel.threads.$threadId.tsx` - mount detail composer + work surfaces.
- `web/src/routes/_app/network.$channel.directs.$directId.tsx` - mount detail composer + work surfaces.

### Dependent Files

- `web/src/systems/network/components/timeline/message-row.tsx` - from task_14; mounts work chip when state is non-default.
- `web/src/systems/network/components/shell/right-rail.tsx` - from task_13; mounts Work Inspector tab.
- `web/src/systems/network/components/timeline/hover-toolbar.tsx` - from task_14; this task wires its handlers.
- `web/src/systems/network/lib/query-keys.ts` - from task_13.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - composer/route alignment.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - work state surfaces.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - composer payload validation.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: composer mutations call public RPCs that agents also drive via CLI/HTTP/UDS/native tools.
- Agent manageability: every UI action corresponds to a server action; no client-only state mutations.
- Config lifecycle: no new config keys.

### Web/Docs Impact

- Web impact: this task owns composer, work surfacing, empty/error states, and realtime polling.
- Docs impact: task_16 documents the composer flows, work lifecycle surfaces, and operator-visible polling behavior.

## Deliverables

- `composer/`, `work/`, and `empty-states/` component subtrees.
- Mutations with optimistic apply/rollback for send, thread create, direct resolve.
- Three-layer work surfacing with auto-hide banner.
- Empty / error / disabled states with verbatim copy from `_design.md` §7.2.
- Realtime polling with documented intervals.
- Mocks, Storybook, browser artifacts, and E2E fixture support for the new components.

## Tests

- Unit tests:
  - [ ] Channel-level composer generates `thread_id`, retries one collision, surfaces toast on second failure.
  - [ ] Channel-level composer redirects to `$threadId` on success.
  - [ ] Detail composer sends into the active route container (thread or direct).
  - [ ] Optimistic message appears within one frame of submit.
  - [ ] Optimistic message is replaced by server canonical message on success (matched by client `MessageID`).
  - [ ] Failed send keeps optimistic message visible with retry/discard inline; no toast for individual send failures.
  - [ ] Work chip is silent for state ∈ {`submitted`, `completed`}.
  - [ ] Work chip renders tinted for `working` / `needs_input` / `failed`; tertiary-only for `canceled`.
  - [ ] Pinned banner renders only when `open_work_count > 0`; auto-hides within 400ms when count returns to 0.
  - [ ] Pinned banner switches to solid `--color-warning` background when any work is in `needs_input`.
  - [ ] Slash command popover lists `/run`, `/mention`, `/attach`; out-of-MVP entries are disabled with tooltip.
  - [ ] Empty state copy matches `_design.md` §7.2 verbatim.
  - [ ] Daemon-down error renders `Network is unreachable.` full-page with retry.
  - [ ] No request body, response body, or rendered string contains `interaction_id` or raw `claim_token`.
  - [ ] No request can be constructed that submits `kind:"direct"`.
- Integration tests:
  - [ ] MSW fixtures cover send (success, collision, second-collision-failure, validation error), direct resolve (new, existing, race), work lookup (active, terminal).
  - [ ] Storybook scenarios cover composer empty / typing / submitting / failed; work chip per state; work banner default / escalation / dismissing; all empty states.
  - [ ] Web E2E covers a full thread create → reply → close flow and a direct resolve → send flow.
  - [ ] Polling intervals match the design spec (channels 30s, lists 15s, messages 5s, work 3s when inspector open) — verified via mock timer.
  - [ ] `refetchOnWindowFocus` triggers a refetch within 200ms of focus restoration.
- Test coverage target: >=80% for touched composer, work, and empty-state modules.
- All tests must pass.

## Success Criteria

- Composer mutations follow the techspec collision and validation rules with no silent state corruption.
- Work surfacing is three-layered, auto-hides at zero, escalates only on `needs_input`, and never violates chromatic discipline.
- Empty, error, and disabled states are terse, mono, and actionable per `_design.md` §7.
- The MVP realtime model is deterministic, tested, and ready to be replaced by SSE post-MVP without UI changes.
