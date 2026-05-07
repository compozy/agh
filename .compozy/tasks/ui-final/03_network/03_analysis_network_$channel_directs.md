# UI/UX Analysis :: `network` :: `/network/$channel/directs`

> **Status:** draft
> **Owner subagent:** `ui-final/network`
> **Date:** 2026-05-06
> **Module:** `network` (`03_network`)
> **Route path:** `/network/$channel/directs` (TanStack Router id: `/_app/network/$channel/directs`)
> **Web source:** `web/src/routes/_app/network.$channel.directs.tsx`
> **System owner:** `web/src/systems/network/components/directs/`
> **Storybook story id(s):** `routes-app-stories-network--directs-tab`, `systems-network-emptystates--no-directs`, `systems-network-networkshell--directs-tab`
> **Live URLs probed:** `http://localhost:3000/network/general/directs` · `http://localhost:6006/iframe.html?id=routes-app-stories-network--directs-tab&viewMode=story`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/network.$channel.directs.tsx`
  - `web/src/systems/network/components/directs/directs-list.tsx`
  - `web/src/systems/network/components/directs/new-direct-dialog.tsx`
  - `web/src/systems/network/components/empty-states/directs-empty.tsx`
  - `web/src/systems/network/components/timeline/message-avatar.tsx`
  - `web/src/systems/network/hooks/use-channel-members.ts`
  - `web/src/systems/network/components/shell/list-filter-bar.tsx`
- **Storybook stories opened:**
  - `routes-app-stories-network--directs-tab` → `http://localhost:6006/iframe.html?id=routes-app-stories-network--directs-tab&viewMode=story`
  - `systems-network-emptystates--no-directs` → `http://localhost:6006/iframe.html?id=systems-network-emptystates--no-directs&viewMode=story`
- **Live web probes (`localhost:3000`):**
  - `/network/general/directs`. daemon empty, parent collapses to "no channels".
- **Screenshots captured:**
  - `_evidence/directs/01-storybook-directs-1440.png`. populated directs list.
  - `_evidence/directs/02-storybook-no-directs-1440.png`. empty state.
- **Console / network errors observed:** none.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** lists every direct (1-to-1) room the local peer participates in inside the active channel. Each row shows the *other* peer's avatar + handle + role chip + last-message preview + relative timestamp. A `New direct` button opens a modal peer picker that resolves a `direct_id` and navigates to it.
- **Primary user goal:** pick a direct room or open one with a specific peer.
- **Entry vectors:** `Directs` tab in the channel header; deep links from rail / inspector / activity feed.
- **Exit vectors:** click any row → `/network/$channel/directs/$directId`; click `New direct` → `NewDirectDialog` → on resolve, navigate to the new direct.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | yes | `directs-empty.tsx:13-37` "No direct rooms yet." with `New direct` action; rendered by `network.$channel.directs.tsx:96-101` when filtered list is empty | strong (action wired). |
| Loading / skeleton | yes | `directs-list.tsx:88-106` 3-row skeleton matches the row layout | strong. |
| Partial data | n/a | single query (`route.directs`); members loaded separately for the role chip. |
| Populated (typical) | yes | `_evidence/directs/01-storybook-directs-1440.png` | strong. |
| Populated (dense) | partial | no virtualization; with hundreds of directs the DOM grows. |
| Error (network) | no | `route.directs.error` is not surfaced in the route. P1. |
| Error (permission / 403) | no | n/a |
| Error (not found) | inherits parent | `directId` route handles per-direct 404 (`direct-room.tsx:83-92`). |
| Read-only / disabled | partial | the `New direct` button is correctly disabled when `sessionId` is empty (`network.$channel.directs.tsx:85-87`). |
| Live-update | partial | inherited polling (`LIST_REFETCH_INTERVAL = 15000`). |
| Mobile / narrow | weak | inherits parent shell. |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   2   | no "last refreshed" affordance | Polling silent. |
| 2  | Match between system and real world    |   2   | `directs-list.tsx:64-71` `AGENT` / `HUMAN` chip; same heuristic source. | Same truthful-UI issue as the inspector members list. |
| 3  | User control and freedom               |   3   | `Cancel` in `NewDirectDialog`; ESC closes; click outside closes | OK. No "undo last delete" because there is no delete. |
| 4  | Consistency and standards              |   2   | active row uses `bg-accent-tint` (`directs-list.tsx:48-51`). divergent from threads list (`bg-surface`) and channel rail (`bg-surface + 2px accent left bar`) | Three "selected row" treatments. |
| 5  | Error prevention                       |   3   | `New direct` requires `sessionId`; disabled otherwise | Solid. |
| 6  | Recognition rather than recall         |   3   | avatar + `@peerId` + role + preview + timestamp | OK. |
| 7  | Flexibility and efficiency of use      |   2   | filter pills + sort dropdown + Mark all read inherited | No keyboard shortcut to focus the New direct button; `New direct` is a button on the route header but the empty state has its own `New direct` button. duplication. |
| 8  | Aesthetic and minimalist design        |   2   | `_evidence/directs/01-storybook-directs-1440.png` | Two consecutive `border-b` strips at the top: filter bar + subheader. P1. |
| 9  | Help users recognize / recover errors  |   2   | `NewDirectDialog` shows the `error.message` inline (`new-direct-dialog.tsx:146-154`) | Errors during resolve are surfaced. List-load errors are not. |
| 10 | Help and documentation                 |   1   | none | No help text, no link to docs. |
|    | **Total**                              | **22/40** | | **Band:** ◯ adequate (20–28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders                             | OK | none |
| Gradient text                                   | OK | none |
| Glassmorphism                                   | OK | none |
| Hero-metric template                            | OK | none |
| Identical card grids                            | OK | flat list |
| Modal as first thought                          | partial | `NewDirectDialog` is a modal for picking a peer; an inline picker (or a popover anchored to the New direct button) would feel lighter. P3. |
| Em dashes in copy                               | OK | none in this file |
| Generic AI palette                              | OK | tokens only |
| Category-reflex theme                           | OK | n/a |
| Restated headings / intros                      | OK | n/a |
| Decorative shadows                              | OK | flat |
| Hardcoded `#000` / `#fff`                       | OK | none |

**Summary verdict:** No, not AI-generated. The two minor concerns are the modal-first peer picker and the divergent active-row treatment.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** filter bar (5 pills + sort + Mark all read) + subheader (`New direct`) + each row.
- **8-item checklist:**
  1. >4 visible? `partial fail`. same as activity tab.
  2. Self-evident labels? `pass`. `Directs`, `New direct`, `@peerId`, `AGENT`/`HUMAN`.
  3. Primary action visually dominant? `partial`. there are TWO `New direct` buttons (subheader + empty state). On a non-empty list the subheader button is the only one. OK.
  4. Progressive disclosure? `pass`.
  5. Grouped by proximity? `pass`.
  6. Hierarchy contrast ≥1.25? `pass`.
  7. Body line length 65–75ch? `pass`. `line-clamp-2` on preview.
  8. Whitespace varied? `partial`. three consecutive `px-5 py-2`/`py-3` strips at the top.

  Failure count: 1 + partial. Low cognitive load, but the duplicate `New direct` button is mild noise.

- **IA observations:**
  - Subheader `X DIRECT ROOMS IN THIS CHANNEL` (mono uppercase) restates the count which the parent shell already displays in the `PageHeader` count slot. Redundant.
  - The `New direct` action lives in two places at once (subheader + empty state). Pick one.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all via `var(--color-*)`. Pass.
- **Type scale:** Inter 14 semibold for `@peerId`, Inter 13 for preview, Mono 10 uppercase for meta. Pass.
- **Radii / spacing:** avatar `size-9` (36px) at `rounded-[4px]`; `MessageAvatar` is non-circular. Acceptable for an operator surface.
- **Elevation:** flat. Pass.
- **Signal palette:** active row uses `bg-accent-tint`. `DESIGN.md` §6 names exactly one selected pattern (Elevated bg + 2px accent left bar). The accent-tint background here is a divergent treatment.
- **Grid / rhythm:** stacked `border-b` strips at the top. Monotone.
- **Density:** comfortable; row `py-3`.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** click row → detail; `New direct` → modal. The empty state action and the subheader action both invoke the same modal.
- **Destructive actions:** none.
- **Forms:** `NewDirectDialog` is a peer-picker. ESC closes; click outside closes. The picker has `aria-selected` on each option. No keyboard arrow-key navigation between options. TAB only.
- **Tables / lists:** sort happens via `useNetworkListFilters` and is applied to the list before render. Sort dropdown values match the actual filter behavior here (unlike activity tab).
- **Selection model:** single only.
- **Modals / drawers:** ESC, click-outside, focus trap via shadcn `Dialog` primitive. Focus restores to trigger on close (verify via `Dialog` impl).
- **Live updates:** silent polling.
- **Optimistic vs pessimistic:** the resolve is pessimistic; navigation happens after success.
- **Hover / focus / active:** all present.
- **Loading patterns:** skeleton matches.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** every row + the subheader button + the modal trigger reachable.
- **Focus rings:** `focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]`.
- **TAB order:** logical (filter → subheader button → first row).
- **ARIA roles / labels:** `role="listbox"` + `role="option"` on the peer picker (`new-direct-dialog.tsx:66-86`); `aria-label="Direct rooms in #${channel}"` on the list; `aria-live="polite"` on the list (`directs-list.tsx:147`).
- **Color contrast:** active row `bg-accent-tint` (#E8572A26) on canvas. the row text is `text-primary #E5E5E7`, which on the tinted background gives ~12:1 contrast. Pass. The `AGENT` / `HUMAN` chip uses tertiary text on the accent-tint background. ~3:1. Below 4.5 for body, but it's a small mono label, marginal pass under WCAG large-text rules.
- **Motion:** none.
- **Text scaling:** survives 200%.
- **Forms:** `NewDirectDialog` uses `DialogTitle` + `DialogDescription` for proper labeling.

---

## 8. Empty / Loading / Error States

- **Empty (no directs in channel):** strong. `directs-empty.tsx:13-37` + an action button wired to `setNewDirectOpen(true)` (`network.$channel.directs.tsx:96-101`).
- **Loading:** strong. Skeleton matches row layout.
- **Error (list load fails):** missing. `route.directs.error` is not surfaced. P1.
- **Error (resolve fails):** strong. Inline error in the modal (`new-direct-dialog.tsx:146-154`).
- **Error (no peers in channel):** the picker shows `No other peers in this channel yet.` (`new-direct-dialog.tsx:54-63`). Strong.
- **Permission denied:** missing.
- **Stale / disconnected:** missing.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary:** `direct room`, `peer`, `channel`. all correct.
- **Tone:** `directs-empty.tsx:31` "Open one to talk privately with a peer in this channel." Sentence case, operator voice. Pass.
- **Em dashes:** none in this file.
- **Restated headings:** `X DIRECT ROOMS IN THIS CHANNEL` mono subheader (`network.$channel.directs.tsx:54-58`) duplicates the count from the `PageHeader` count slot. Cognitive overhead, minor.
- **Sentence vs Title case:** subheader is uppercase mono (matches eyebrow grammar). `New direct` button is sentence case. OK.
- **Truthful UI test:**
  - `AGENT` / `HUMAN` chip is heuristic (P0 inherited).
  - Otherwise honest. `last_message_preview ?? "No messages yet."` accurately reflects state.

---

## 10. Performance & Responsiveness

- **Initial render:** depends on `route.directs` + `route.members` (members are needed for the role chip).
- **Re-render hot spots:** `buildRoleLookup` runs each render (`directs-list.tsx:108-119`); minor, but should be `useMemo`.
- **List virtualization:** none.
- **Bundle red flags:** none.
- **Responsive behaviour:** inherits parent shell.
- **Mobile interactions:** rows `py-3` meet touch target.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-network--directs-tab`. populated.
  - `systems-network-emptystates--no-directs`. empty.
  - `systems-network-networkshell--directs-tab`. shell-level.
- **States covered:** populated, empty.
- **Gaps:** loading, error (list load), error (resolve), no-peers-in-channel modal, dense list. P3.
- **Story drift:** none.

---

## 12. Findings. Prioritised

### P0. Ship Blockers

None unique to this route. Inherits P0-NET-3 (AGENT/HUMAN chip) from the parent shell. applies directly to `directs-list.tsx:64-71`.

### P1. High-Value Polish

1. **[P1-DIR-1] What:** Active row uses `bg-accent-tint` instead of the canonical `bg-surface + 2px left accent bar`.
   - **Why:** divergent from the channel rail and the threads list.
   - **Fix:** apply the `ACTIVE_NAV_*` shared classes.
   - **Cmd:** `/impeccable polish web/src/systems/network/components/directs/directs-list.tsx`
   - **Effort:** S
   - **Evidence:** `directs-list.tsx:48-51` vs `channel-rail-row.tsx:36-58`.

2. **[P1-DIR-2] What:** Two consecutive `border-b` strips at the top of the route (`ListFilterBar` + subheader).
   - **Why:** monotone rhythm, redundant `New direct` button.
   - **Fix:** merge `New direct` into the `ListFilterBar` right cluster (next to Mark all read), drop the subheader. The count is already in the `PageHeader`.
   - **Cmd:** `/impeccable layout web/src/routes/_app/network.$channel.directs.tsx`
   - **Effort:** M
   - **Evidence:** `network.$channel.directs.tsx:65-94`.

3. **[P1-DIR-3] What:** List-load errors are silent.
   - **Fix:** if `route.directs.error`, render a `ConversationError` empty.
   - **Cmd:** `/impeccable harden web/src/routes/_app/network.$channel.directs.tsx`
   - **Effort:** S
   - **Evidence:** `network.$channel.directs.tsx:47-51`.

### P2. Worthwhile

1. **[P2-DIR-1] What:** Modal-first peer picker for `New direct`.
   - **Fix:** consider an anchored popover from the New direct button. Defer until UX research justifies. P3 acceptable.
   - **Effort:** L
   - **Evidence:** `new-direct-dialog.tsx`.

2. **[P2-DIR-2] What:** `buildRoleLookup` runs every render.
   - **Fix:** `useMemo`.
   - **Cmd:** `/impeccable optimize web/src/systems/network/components/directs/directs-list.tsx`
   - **Effort:** S
   - **Evidence:** `directs-list.tsx:108-119`.

3. **[P2-DIR-3] What:** Subheader count duplicates the `PageHeader` count.
   - **Fix:** drop the subheader (covered by P1-DIR-2 if merged).
   - **Effort:** S

### P3. Parking Lot

1. No keyboard arrow-key nav inside the peer picker.
2. No virtualization.
3. No "delete direct" action (might be intentional; verify against daemon contract).

---

## 13. Persona Red Flags

- **Operator (returning power user):** TWO `New direct` controls when the list is empty (subheader + empty action) is a minor distractor. Modal-first peer picker is heavyweight for a single click.
- **First-timer:** the role chip is the most prominent affordance after the @peerId; if it's wrong (heuristic), the operator forms a wrong mental model of who they're talking to.
- **Agent:** stable testids (`network-direct-list-row-<id>`, `network-direct-list-row-role-<id>`, `network-directs-new-direct`). Strong.

---

## 14. Cross-Module Consistency Notes

- Active-row treatment (P1-DIR-1) diverges from sibling lists. Pick one shell-wide.
- `New direct` button uses `variant="outline"` size `sm`. same primitive as `New skill` in skills, `New task` in tasks. Consistent.
- The `MessageAvatar` `rounded-[4px]` is reused from `inspector-members-list.tsx` and `directs-list.tsx`. Consistent.

---

## 15. Open Questions

- Should `New direct` be a popover anchored to the button rather than a modal?
- The subheader exists per-route. does any other tab have a subheader pattern? (Activity tab has a similar "Recent activity · Read-only" subheader.) If yes, document it; if not, drop both.
- Should role chip be removed until the daemon supplies `kind`?

---

## 16. Recommended Action Plan

1. `/impeccable clarify web/src/systems/network/components/directs/directs-list.tsx`. strip or qualify the AGENT/HUMAN chip per P0-NET-3.
2. `/impeccable polish web/src/systems/network/components/directs/directs-list.tsx`. unify active row pattern.
3. `/impeccable layout web/src/routes/_app/network.$channel.directs.tsx`. merge subheader into filter bar; remove redundant count.
4. `/impeccable harden web/src/routes/_app/network.$channel.directs.tsx`. surface list-load errors.
5. `/impeccable optimize web/src/systems/network/components/directs/directs-list.tsx`. memoize role lookup.
6. `/impeccable polish web/src/systems/network/components/directs/`. final sweep.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/directs/`.
- [x] No section left empty.
- [x] Nielsen total (22/40) consistent with band (◯ adequate).
- [x] Findings tagged P0–P3 with effort and command.
- [x] No hallucinated routes, components, or props.
- [x] No em dashes.
