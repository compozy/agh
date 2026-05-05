---
status: completed
title: Web Message Row, Timeline, Thread Overlay & Author Group Collapse
type: frontend
complexity: critical
dependencies:
  - task_13
---

# Task 14: Web Message Row, Timeline, Thread Overlay & Author Group Collapse

## Overview

Build the message rendering surface on top of the shell from task_13. This task delivers the message row (full row, collapsed continuation, system event variants), timeline composition with author group collapsing (60s window), date pills, the "New" divider, the hover toolbar, and the hybrid right-rail thread overlay (URL-canonical with channel context preserved on `>=1024px`, full-page route swap on `<1024px`). Direct rooms render with the headerless layout. No composer, no work surfacing, no realtime — those land in task_15.

The normative UI source is `_design.md`. Sections referenced below: §3.2 (right-rail thread overlay), §3.3 (responsive collapse), §5.2 (message row anatomy), §5.3 (author group collapsing), §5.4 (date pills + "New" divider), §5.5 (thread overlay), §5.6 (direct room headerless), §6 (chromatic discipline), §8 (motion).

<critical>
- ALWAYS READ `_techspec.md`, `_design.md`, all ADRs, `web/CLAUDE.md`, and `DESIGN.md` before editing.
- ACTIVATE `agh-design`, `design-taste-frontend`, `minimalist-ui`, `react`, `tanstack-router-best-practices`, `tanstack-query-best-practices`, `vitest`, `testing-anti-patterns`, and `app-renderer-systems`.
- REFERENCE `_design.md` §5 (component anatomy) for every component built here.
- REFERENCE `_design.md` §6 (chromatic discipline) — every PR review will check the seven rules.
- FOCUS ON timeline rendering, thread overlay, and author group composition. Do not implement composer, work chip/banner/inspector, empty/error states (those are task_15).
- TESTS REQUIRED for message row variants, group collapse rules (60s window, kind boundary, author boundary), date pills across midnight, "New" divider position, thread overlay open/close behavior, responsive fallback at 1024px.
- NO WORKAROUNDS: animate only `transform` and `opacity` (`_design.md` §8.2). No drop shadows. No `box-shadow` on hover toolbars or overlays.
</critical>

<requirements>
- MUST implement `MessageRow` with three variants per `_design.md` §5.2: full row, collapsed continuation (60s same-author), system event (low-prominence single-line for `kind ∈ {greet, whois, capability, receipt, trace}`).
- MUST suppress kind chip for `kind:"say"` per `_design.md` §6.4 — it is the default.
- MUST suppress role chip on collapsed continuation rows per `_design.md` §6.5.
- MUST collapse author groups within a 60-second window from the previous message by the same `peer_from`, breaking on author change, kind change, or gap `>60s` per `_design.md` §5.3.
- MUST render avatar gutter at 36px in main timeline, 32px inside thread overlay.
- MUST never render avatar as a circle — `border-radius: 4px` (flat-depth, geometric).
- MUST implement date pills (TODAY / YESTERDAY / weekday / dated, with year prefix on year-boundary crossing) per `_design.md` §5.4.
- MUST implement the "New" divider with `--color-accent` line and `NEW` label, positioned at the user's last-read boundary tracked via `use-last-read` from task_13.
- MUST implement the hybrid right-rail thread overlay per `_design.md` §3.2 — URL canonical, channel timeline rendered at `opacity: 0.55` behind it on `>=1024px`.
- MUST collapse the hybrid model to full-page route swap on `<1024px` per `_design.md` §3.3.
- MUST render direct room detail with the headerless layout per `_design.md` §5.6 — peer identity row, no `#name`, no member count, no topic.
- MUST support presence dot semantics from `_design.md` §5.6 (mono at idle, accent pulse at running, warning steady at needs_input, danger steady at errored). Presence source is post-MVP — use the placeholder hook `use-network-presence` from task_13 returning idle.
- MUST render the hover toolbar inline (no shadow, 1px border in `--color-divider`) per `_design.md` §5.2.1; the actual handlers (Reply, Pin, Fork, kebab) can be no-op stubs in this task — wiring lands in task_15.
- MUST honor `prefers-reduced-motion` globally per `_design.md` §14.2.
</requirements>

## Subtasks

- [x] 14.1 Build `web/src/systems/network/components/timeline/` (`timeline.tsx`, `message-row.tsx`, `message-row-collapsed.tsx`, `message-row-system.tsx`, `date-pill.tsx`, `new-divider.tsx`, `hover-toolbar.tsx`).
- [x] 14.2 Implement author group collapsing (60s window) in `lib/group-messages.ts` with break rules (author / kind / gap).
- [x] 14.3 Build `web/src/systems/network/components/thread-overlay/` (`thread-overlay.tsx`, `thread-overlay-header.tsx`, `thread-overlay-root.tsx`, `thread-overlay-replies.tsx`).
- [x] 14.4 Wire hybrid right-rail behavior: open on `$threadId` route, render channel timeline at reduced contrast, close on Esc/X/outside-click; full-page swap on `<1024px`.
- [x] 14.5 Implement direct room headerless layout for `network.$channel.directs.$directId.tsx`; reuse `Timeline` for messages.
- [x] 14.6 Implement `use-threads.ts` (list + detail), `use-directs.ts` (list + detail), `use-messages.ts` (shared paginated query for both surfaces).
- [x] 14.7 Add Storybook fixtures for every message-row variant, every collapse boundary, and the thread overlay states (closed / opening / open / closing).

## Implementation Details

Author group collapse window is 60 seconds (tighter than Slack's 5 min) because agents emit fast and a longer window would visually merge unrelated tool-call bursts.

The thread overlay's "Open in main →" affordance promotes the overlay to a full-page render at the same URL. Implementation: a UI-only mode flag in route search params (`?view=full`) that the route reads to switch between overlay and full-page rendering — does not affect the canonical URL pattern.

### Relevant Files

- `web/src/systems/network/components/timeline/*` - timeline subtree.
- `web/src/systems/network/components/thread-overlay/*` - thread overlay subtree.
- `web/src/systems/network/hooks/use-threads.ts` - threads list + detail.
- `web/src/systems/network/hooks/use-directs.ts` - directs list + detail.
- `web/src/systems/network/hooks/use-messages.ts` - messages query for both surfaces.
- `web/src/systems/network/hooks/use-network-presence.ts` - placeholder returning idle (post-MVP).
- `web/src/systems/network/lib/group-messages.ts` - author group collapse logic.
- `web/src/systems/network/lib/format-timestamp.ts` - relative time, ISO tooltip, date-pill format.
- `web/src/routes/_app/network.$channel.threads.tsx` - integrate `Timeline`.
- `web/src/routes/_app/network.$channel.threads.$threadId.tsx` - integrate `ThreadOverlay`.
- `web/src/routes/_app/network.$channel.directs.tsx` - integrate `Timeline`.
- `web/src/routes/_app/network.$channel.directs.$directId.tsx` - integrate headerless layout + `Timeline`.
- `web/src/routes/_app/network.$channel.activity.tsx` - integrate read-only `Timeline`.

### Dependent Files

- `web/src/systems/network/components/shell/right-rail.tsx` - from task_13; thread overlay mounts here.
- `web/src/systems/network/lib/query-keys.ts` - from task_13; thread/direct queries follow that schema.
- `web/src/systems/network/hooks/use-last-read.ts` - from task_13; "New" divider reads this.
- `packages/ui/*` - Skeleton, Avatar, Pill, Tabs primitives reused.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - thread vs direct rendering.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - kind chip rules.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: timeline must render any `kind` defined by the protocol; chip rules suppress noise but rendering must be schema-driven, not hardcoded.
- Agent manageability: hover toolbar surfaces actions (Reply/Pin/Fork) that map to the same RPCs agents call.
- Config lifecycle: no new config keys.

### Web/Docs Impact

- Web impact: this task owns timeline, thread overlay, and direct room rendering.
- Docs impact: task_16 documents the resulting message and thread visuals.

## Deliverables

- `timeline/` and `thread-overlay/` component subtrees.
- Author group collapse logic with verified break rules.
- Date pills and "New" divider working across timezone and date boundaries.
- Hybrid right-rail thread overlay with responsive collapse fallback.
- Direct room headerless layout.
- Storybook coverage for every message-row variant and thread overlay state.

## Tests

- Unit tests:
  - [ ] `MessageRow` full variant renders avatar, name, timestamp, role chip, body.
  - [ ] `MessageRow` collapsed variant suppresses avatar visual, name, role chip; reveals timestamp on gutter hover.
  - [ ] `MessageRow` system variant renders single-line, mono, no avatar.
  - [ ] Kind chip not rendered for `kind:"say"`.
  - [ ] Role chip rendered only on first row of an author group.
  - [ ] Author group collapses within 60s and breaks on author / kind / gap.
  - [ ] Date pill renders correctly across midnight and year boundaries.
  - [ ] "New" divider positions at the first message newer than `lastRead[channel:surface:containerId]`.
  - [ ] Thread overlay opens on navigation to `$threadId`, closes on Esc / X / outside click.
  - [ ] On `<1024px`, navigating to `$threadId` yields a full-page render (no overlay).
  - [ ] Direct room renders no `#name`, no member count.
  - [ ] Avatar always renders with `border-radius: 4px` (snapshot).
  - [ ] No `box-shadow` declared on any new component (CSS audit).
- Integration tests:
  - [ ] MSW fixtures cover thread heads, thread detail, direct list, direct detail, message pagination.
  - [ ] Storybook scenarios cover full / collapsed / system message rows; closed / opening / open / closing thread overlay; empty / populated / loading direct room headers.
  - [ ] Web route tests cover thread list, thread detail, direct list, direct detail, activity timeline.
  - [ ] `prefers-reduced-motion` disables overlay slide and pulse animations (verified via Storybook test).
- Test coverage target: >=80% for touched timeline and overlay modules.
- All tests must pass.

## Success Criteria

- Timeline renders all message kinds with correct chip and group semantics.
- Thread overlay preserves channel context on `>=1024px` and falls back cleanly on `<1024px`.
- Direct rooms render with the headerless layout from `_design.md` §5.6.
- No chromatic discipline rule from `_design.md` §6 is violated by any new component.
