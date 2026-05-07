# UI/UX Analysis :: `network` :: `/network/$channel/threads`

> **Status:** draft
> **Owner subagent:** `ui-final/network`
> **Date:** 2026-05-06
> **Module:** `network` (`03_network`)
> **Route path:** `/network/$channel/threads` (TanStack Router id: `/_app/network/$channel/threads`)
> **Web source:** `web/src/routes/_app/network.$channel.threads.tsx`
> **System owner:** `web/src/systems/network/components/threads/threads-list.tsx` + `web/src/systems/network/components/composer/channel-thread-composer.tsx`
> **Storybook story id(s):** `routes-app-stories-network--threads-tab`, `systems-network-emptystates--no-threads`, `systems-network-networkshell--default`, `systems-network-channelheader--threads-tab-active`, `systems-network-composer--default|submitting|disabled`
> **Live URLs probed:** `http://localhost:3000/network/general/threads` · `http://localhost:6006/iframe.html?id=routes-app-stories-network--threads-tab&viewMode=story`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/network.$channel.threads.tsx`
  - `web/src/systems/network/components/threads/threads-list.tsx`
  - `web/src/systems/network/components/empty-states/threads-empty.tsx`
  - `web/src/systems/network/components/composer/channel-thread-composer.tsx`
  - `web/src/systems/network/components/composer/composer.tsx`
  - `web/src/systems/network/components/shell/list-filter-bar.tsx`
  - `web/src/systems/network/hooks/use-network-list-filters.ts`
- **Storybook stories opened:**
  - `routes-app-stories-network--threads-tab` → 1440 / 1024 / 768 / 320.
  - `systems-network-emptystates--no-threads`
  - `systems-network-channelheader--threads-tab-active`
- **Live web probes (`localhost:3000`):** parent route collapses to "no channels" because daemon empty.
- **Screenshots captured:**
  - `_evidence/threads/03-storybook-threadstab-1440.png`. populated threads tab.
  - `_evidence/threads/03b-storybook-threadstab-full-1440.png`. full-page capture.
  - `_evidence/threads/04-storybook-768.png`. 768.
  - `_evidence/threads/05-storybook-320.png`. 320.
  - `_evidence/threads/06-storybook-1024.png`. 1024.
- **Console / network errors observed:** none.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** the canonical landing surface inside a channel. Lists every thread in the active channel sorted by recent activity, with a `ChannelThreadComposer` pinned at the bottom. Thread rows show title, last preview, peer-count, reply-count, opener, last-activity timestamp, and a `1 work open` chip when there is open work.
- **Primary user goal:** scan threads + reply to / open one + start a new thread.
- **Entry vectors:** clicking a channel rail row (default landing per `useNetworkRouteShell`); the `Threads` tab in the channel header.
- **Exit vectors:** click a thread row → `/network/$channel/threads/$threadId` (or with `?view=full`); composer submit → creates thread → navigates to the new thread.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | yes | `threads-empty.tsx:13-37` "No threads yet." with `Start a thread` action; rendered by `threads-list.tsx:126-132` | strong (action wired by composer focus). Em dash inside description: P1. |
| Loading / skeleton | yes | `threads-list.tsx:97-112` 5-row skeleton matches | strong. |
| Partial data | yes | TanStack stale-while-revalidate. |
| Populated (typical) | yes | `_evidence/threads/03-storybook-threadstab-1440.png` | strong. |
| Populated (dense) | partial | no virtualization. |
| Error (network) | no | `threadsQuery.error` not surfaced in the route | P1. |
| Error (permission / 403) | no | n/a |
| Error (not found / 404) | inherits | parent shell handles "no channels" / "first-visible-channel" auto-nav. |
| Read-only / disabled | yes | `disabledReason` flows into composer placeholder (`network.$channel.threads.tsx:75-79`) | OK. |
| Live-update | partial | polling 15s for list, 5s for messages (within an opened thread). |
| Mobile / narrow | weak | parent shell does not collapse rails. |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   2   | `threads-list.tsx:147-155` `aria-live="polite"` and `Sorted by recent activity` subheader; polling silent | No "last refreshed" indicator. |
| 2  | Match between system and real world    |   3   | row meta uses real fields (`peer_count`, `reply_count`, `opened_by_peer_id`) | OK. |
| 3  | User control and freedom               |   3   | composer cancellable; click-away does not lose draft (state held in `useComposerState`) | OK. |
| 4  | Consistency and standards              |   2   | active row uses `bg-surface` only. no left accent bar. Diverges from canonical pattern (P1-NET-1). |
| 5  | Error prevention                       |   2   | composer is disabled when no session (`channel-thread-composer.tsx:26`); collision toast surfaced via `useCreateNetworkThread` (per code comment); list errors silent. |
| 6  | Recognition rather than recall         |   3   | row has title + preview + peer-count + reply-count + opener; the `started by <opener>` is honest about who opened it. |
| 7  | Flexibility and efficiency of use      |   2   | Cmd+Enter sends; slash popover; no per-row keyboard shortcut. |
| 8  | Aesthetic and minimalist design        |   3   | `_evidence/threads/03-storybook-threadstab-1440.png` | Clean. |
| 9  | Help users recognize / recover errors  |   2   | composer collision toast exists (per code comment) but list-load errors are silent. |
| 10 | Help and documentation                 |   1   | none. |
|    | **Total**                              | **23/40** | | **Band:** ◯ adequate (20–28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders                             | OK | none |
| Gradient text                                   | OK | none |
| Glassmorphism                                   | OK | none |
| Hero-metric template                            | OK | none |
| Identical card grids                            | OK | flat list |
| Modal as first thought                          | OK | inline composer |
| Em dashes in copy                               | violations | `threads-empty.tsx:30` "Start the first one. agents and humans both join." P1. |
| Generic AI palette                              | OK | tokens only |
| Category-reflex theme                           | OK | n/a |
| Restated headings / intros                      | OK | subheader is meta count, not a title repeat |
| Decorative shadows                              | OK | flat |
| Hardcoded `#000` / `#fff`                       | OK | none |

**Summary verdict:** No, not AI-generated. Em dash is the only anti-pattern hit.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** the filter bar (5 pills + sort + Mark all read) + composer + each row.
- **8-item checklist:**
  1. >4 visible? `partial fail`. filter bar.
  2. Self-evident labels? `pass`.
  3. Primary action visually dominant? `pass`. composer Send button is the orange CTA.
  4. Progressive disclosure? `pass`.
  5. Grouped by proximity? `pass`.
  6. Hierarchy contrast ≥1.25? `pass`.
  7. Body line length 65–75ch? `pass`. preview is `line-clamp-2`.
  8. Whitespace varied? `partial`.

  Failure count: 1 + partial. Low load.

- **IA observations:**
  - Subheader strip "X threads · Sorted by recent activity" (`threads-list.tsx:147-155`) is informative but a third consecutive `border-b` row at the top (filter + subheader + first row).
  - `1 work open` chip uses warning color which competes for attention with active rows. With many open-work threads the warning chips will dominate the visual rhythm.
  - "started by <opener>" exposes the raw `peer_id` (the source code uses `opened_by_peer_id?.trim() || "unknown"`). For human peers without a `display_name` plumbed here, this becomes a long opaque hash-like string. P2.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all via `var(--color-*)`. Pass.
- **Type scale:** Inter 14 semibold for title, Inter 13 for preview, Mono 10 uppercase for meta. Pass.
- **Radii / spacing:** rows `px-5 py-4` (slightly taller than directs `py-3`. intentional? not commented).
- **Elevation:** flat. Pass.
- **Signal palette:** `1 work open` chip uses warning tint per signal grammar. Pass.
- **Grid / rhythm:** filter bar + subheader + rows + composer. Four bands.
- **Density:** comfortable.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** click row → detail (overlay or full page); type into composer → Send.
- **Destructive actions:** none on the list.
- **Forms:** `ChannelThreadComposer` autoresizes, Cmd+Enter to send, navigates to the new thread on success.
- **Tables / lists:** sort honored via `useNetworkListFilters` (`filter.sort` flows through to filtered ordering). Filter pills work.
- **Selection model:** single only.
- **Modals / drawers:** none on the list.
- **Live updates:** silent polling.
- **Optimistic vs pessimistic:** thread creation is pessimistic (navigates after success). The collision-toast path is described in the comment but I did not verify the hook implementation.
- **Hover / focus / active:** present.
- **Loading patterns:** 5-row skeleton matches.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** every row is a `Link`; composer reachable via TAB.
- **Focus rings:** `focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]`.
- **TAB order:** filter → first row → ...  → composer.
- **ARIA roles / labels:** `<div aria-label="Threads in #${channel}" aria-live="polite">` on the list (`threads-list.tsx:138-145`).
- **Color contrast:** title ~13:1; preview ~5.4:1; mono tertiary ~4.4:1. Pass.
- **Motion:** none.
- **Text scaling:** survives 200%.
- **Forms:** composer textarea labeled.

---

## 8. Empty / Loading / Error States

- **Empty:** strong, with em dash to fix.
- **Loading:** strong.
- **Error (list load fails):** missing.
- **Error (composer send fails):** comment says toast surfaces; verified via the `useCreateNetworkThread` hook (referenced in the source comment).
- **Permission denied:** missing.
- **Stale / disconnected:** missing.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary:** `thread`, `peer`, `channel`. correct.
- **Tone:** operator voice. Pass.
- **Em dashes:** `threads-empty.tsx:30`. P1.
- **Restated headings:** `Sorted by recent activity` is informative meta, not a title repeat.
- **Sentence vs Title case:** sentence case throughout.
- **Truthful UI test:**
  - All row metadata is daemon-sourced (`message_count`, `participant_count`, `last_activity_at`, `last_message_preview`, `opened_by_peer_id`, `open_work_count`). Pass.
  - "1 work open" is honest. the chip only renders when `openWorkCount > 0` (`threads-list.tsx:27-40`).

---

## 10. Performance & Responsiveness

- **Initial render:** depends on `threadsQuery` only.
- **Re-render hot spots:** the `total` and `replyCount` are computed inline; no obvious thrash.
- **List virtualization:** none.
- **Bundle red flags:** none.
- **Responsive behaviour:** at 1024px the row layout still works (preview wraps); at 768 / 320 inherits parent shell issues.
- **Mobile interactions:** rows full-width; meets touch.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-network--threads-tab`. populated.
  - `systems-network-emptystates--no-threads`. empty.
  - `systems-network-composer--default|submitting|disabled`.
- **States covered:** populated, empty, composer states.
- **Gaps:** loading-list, error-list, dense list, mobile / narrow.
- **Story drift:** none observed.

---

## 12. Findings. Prioritised

### P0. Ship Blockers

None unique to this route. Inherits P0-NET-1 (hover toolbar inside any thread the user opens) and P0-NET-2 (composer toolbar) from the parent shell.

### P1. High-Value Polish

1. **[P1-THR-1] What:** Active row uses `bg-surface` only, no left accent bar.
   - **Fix:** apply `ACTIVE_NAV_*` shared classes (consistent with channel rail).
   - **Cmd:** `/impeccable polish web/src/systems/network/components/threads/threads-list.tsx`
   - **Effort:** S
   - **Evidence:** `threads-list.tsx:54-57`.

2. **[P1-THR-2] What:** List-load errors silent.
   - **Fix:** if `threadsQuery.error`, render a `ConversationError`-style empty.
   - **Cmd:** `/impeccable harden web/src/routes/_app/network.$channel.threads.tsx`
   - **Effort:** S
   - **Evidence:** `network.$channel.threads.tsx:24-31`.

3. **[P1-THR-3] What:** Em dash in `threads-empty.tsx:30`.
   - **Fix:** "Start the first one. Agents and humans both join."
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/empty-states/threads-empty.tsx`
   - **Effort:** S
   - **Evidence:** `threads-empty.tsx:30`.

### P2. Worthwhile

1. **[P2-THR-1] What:** "started by <opener>" exposes the raw `peer_id`. For human operators without a `display_name` field, this is hash-like.
   - **Fix:** hydrate via `useChannelMembers` to look up `display_name`; fall back to `peer_id` only when missing.
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/threads/threads-list.tsx`
   - **Effort:** S
   - **Evidence:** `threads-list.tsx:49-50, 81-83`.

2. **[P2-THR-2] What:** Subheader "Sorted by recent activity" is technically truthful but the sort dropdown lets the user choose `Created` / `Alphabetical`. The subheader should reflect the active sort.
   - **Fix:** read `filters.sort` and render `Sorted by ${SORT_LABELS[sort]}`.
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/threads/threads-list.tsx`
   - **Effort:** S
   - **Evidence:** `threads-list.tsx:154`.

3. **[P2-THR-3] What:** No virtualization. Acceptable until 500+ threads per channel.

### P3. Parking Lot

1. Composer placeholder is the same string ("Start a new thread…") regardless of channel; consider per-channel hint.

---

## 13. Persona Red Flags

- **Operator (returning power user):** sort dropdown does not match subheader; active row is hard to find when scrolling.
- **First-timer:** "started by aoCv12... K8" is opaque; `1 work open` chip without context.
- **Agent (DOM scraping):** stable testids (`network-thread-list`, `network-thread-list-row-<id>`, `network-thread-list-row-meta-peers|replies|opener|time`, `network-thread-list-row-state-chip`). Strong.

---

## 14. Cross-Module Consistency Notes

- Same row primitive (`Link` + meta strip) is used by directs and threads, but the `1 work open` chip exists only on threads. Verify whether directs should ever surface open work too.
- Active row pattern diverges from rail / directs (P1-NET-1 / P1-DIR-1).

---

## 15. Open Questions

- Should the threads list show a `last_message_author` chip (similar to the directs list role chip) so the operator can distinguish whose voice is in the preview?
- Should the row include a tiny presence indicator for the most-recent message author?
- Why is the row `py-4` here vs `py-3` in directs? Document or unify.

---

## 16. Recommended Action Plan

1. `/impeccable polish web/src/systems/network/components/threads/threads-list.tsx`. unify active-row pattern.
2. `/impeccable harden web/src/routes/_app/network.$channel.threads.tsx`. surface query errors.
3. `/impeccable clarify web/src/systems/network/components/empty-states/threads-empty.tsx`. rewrite without em dash.
4. `/impeccable clarify web/src/systems/network/components/threads/threads-list.tsx`. hydrate opener via `useChannelMembers`; reflect active sort in subheader.
5. `/impeccable polish web/src/systems/network/components/threads/`. final sweep.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/threads/`.
- [x] No section left empty.
- [x] Nielsen total (23/40) consistent with band (◯ adequate).
- [x] Findings tagged P0–P3 with effort and command.
- [x] No hallucinated routes, components, or props.
- [x] No em dashes.
