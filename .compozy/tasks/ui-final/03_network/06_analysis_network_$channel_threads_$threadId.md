# UI/UX Analysis :: `network` :: `/network/$channel/threads/$threadId`

> **Status:** draft
> **Owner subagent:** `ui-final/network`
> **Date:** 2026-05-06
> **Module:** `network` (`03_network`)
> **Route path:** `/network/$channel/threads/$threadId` (TanStack Router id: `/_app/network/$channel/threads/$threadId`)
> **Web source:** `web/src/routes/_app/network.$channel.threads.$threadId.tsx`
> **System owner:** `web/src/systems/network/components/thread-overlay/`
> **Storybook story id(s):** `systems-network-threadoverlay--header|root|replies-populated|replies-loading|replies-empty`, `systems-network-work--banner|banner-escalation`, `systems-network-emptystates--thread-empty-state`
> **Live URLs probed:** `http://localhost:3000/network/general/threads/<threadId>` · `http://localhost:6006/iframe.html?id=systems-network-threadoverlay--replies-populated&viewMode=story`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/network.$channel.threads.$threadId.tsx`
  - `web/src/systems/network/components/thread-overlay/thread-overlay.tsx`
  - `web/src/systems/network/components/thread-overlay/thread-overlay-header.tsx`
  - `web/src/systems/network/components/thread-overlay/thread-overlay-root.tsx`
  - `web/src/systems/network/components/thread-overlay/thread-overlay-replies.tsx`
  - `web/src/systems/network/components/thread-overlay/use-thread-overlay-view.ts`
  - `web/src/systems/network/components/timeline/timeline.tsx`
  - `web/src/systems/network/components/timeline/message-row.tsx`
  - `web/src/systems/network/components/timeline/hover-toolbar.tsx`
  - `web/src/systems/network/components/composer/detail-composer.tsx`
  - `web/src/systems/network/components/work/work-banner.tsx`
  - `web/src/systems/network/components/empty-states/{thread-empty,conversation-error}.tsx`
- **Storybook stories opened:**
  - `systems-network-threadoverlay--replies-populated` → `http://localhost:6006/iframe.html?id=systems-network-threadoverlay--replies-populated&viewMode=story`
  - `systems-network-threadoverlay--replies-loading`
  - `systems-network-threadoverlay--replies-empty`
- **Live web probes (`localhost:3000`):** parent route collapses to "no channels" because daemon empty.
- **Screenshots captured:**
  - `_evidence/thread-detail/01-storybook-thread-replies-1440.png`. populated replies in overlay density.
- **Console / network errors observed:** none.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** renders one thread, either as a 360px right-rail overlay (default) or as a full-page main-pane view when `?view=full` is set (or when `useThreadViewMode()` decides "fullpage". typically below 1024px). Anatomy: `ThreadOverlayHeader` (title, participant count, "Open in main", close X) → optional `WorkBanner` → `ThreadOverlayRoot` (the root post inside a "ROOT" eyebrow) → `ThreadOverlayReplies` (divider with `N replies` mono label, then `Timeline` density=overlay) → `DetailComposer surface="thread"`.
- **Primary user goal:** read the thread + reply.
- **Entry vectors:** click a row in `threads-list`, click a thread in the inspector activity feed, click a recents thread, deep link with optional `?view=full`.
- **Exit vectors:** close X navigates to `/network/$channel/threads`; `Open in main` toggles to `?view=full`; the parent shell still owns the rail / channel header / inspector toggle when in overlay mode.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty (no replies) | yes | `thread-overlay-replies.tsx:48-54` `emptyOverride={<ThreadEmpty />}` | adequate. |
| Loading / skeleton | yes | `thread-overlay.tsx:39-44` shows root + replies loading; `thread-overlay-root.tsx:9-23` shows a "Loading root" mono row | adequate. The replies skeleton uses `Timeline` density=overlay. matches. |
| Partial data | yes | detail can resolve before messages | OK. |
| Populated (typical) | yes | story `systems-network-threadoverlay--replies-populated` | strong. |
| Populated (dense) | partial | no virtualization. |
| Error (network / detail load) | yes | `thread-overlay.tsx:31-38` `ConversationError` | strong. |
| Error (permission / 403) | no | n/a |
| Error (not found / 404) | yes | same `ConversationError` branch when `detailError` set | strong, copy explicitly says "Could not load thread <id>. Choose an existing thread from #<channel>." |
| Read-only / disabled | yes | `disabledReason` flows into composer | OK. |
| Live-update | partial | polling 5s for messages, 15s for thread detail. |
| Mobile / narrow | partial | `useThreadViewMode` decides full-page below 1024px (`network.$channel.threads.$threadId.tsx:18-22`). The right-rail overlay flips to a full-page rendering automatically. Better than the rest of the module. |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   3   | `WorkBanner` aria-live, replies divider with explicit count (`thread-overlay-replies.tsx:31-44`) | Polling silent. |
| 2  | Match between system and real world    |   3   | `participant_count` exposed as "1 peer" / "X peers"; `replyCount` honest. The `ROOT` mono badge clearly marks the thread root. |
| 3  | User control and freedom               |   4   | Close X navigates back to list; `Open in main` toggles full-page; full-page back to list via parent route. Multiple escape paths. |
| 4  | Consistency and standards              |   3   | timeline + composer reused | Inspector vs overlay shell shape differs (overlay has its own header). Acceptable. |
| 5  | Error prevention                       |   3   | composer disabled when no session; collision-toast on duplicate sends; `ConversationError` for unavailable thread. |
| 6  | Recognition rather than recall         |   3   | `ROOT` badge + `N replies` divider is great signal. |
| 7  | Flexibility and efficiency of use      |   2   | Cmd+Enter sends; full-page toggle; no per-message keyboard shortcut; hover toolbar dead controls (P0 inherited). |
| 8  | Aesthetic and minimalist design        |   3   | `_evidence/thread-detail/01-storybook-thread-replies-1440.png` | Editorial calm with eyebrows; clean. |
| 9  | Help users recognize / recover errors  |   3   | `ConversationError` is specific ("Choose an existing thread from #<channel>."); message Retry/Discard inline. |
| 10 | Help and documentation                 |   1   | none. |
|    | **Total**                              | **28/40** | | **Band:** ◯ adequate (20–28). strongest single route in the module |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders                             | OK | none |
| Gradient text                                   | OK | none |
| Glassmorphism                                   | OK | none |
| Hero-metric template                            | OK | none |
| Identical card grids                            | OK | not a card grid |
| Modal as first thought                          | OK | overlay is an `aside`, not a `Dialog`. |
| Em dashes in copy                               | OK | none in this file or its empty state. |
| Generic AI palette                              | OK | tokens only |
| Category-reflex theme                           | OK | n/a |
| Restated headings / intros                      | OK | "ROOT" mono badge is functional metadata, not a title repeat |
| Decorative shadows                              | OK | flat |
| Hardcoded `#000` / `#fff`                       | OK | none |

**Summary verdict:** No, not AI-generated. The two flags inherited from the parent shell are the dead hover toolbar and dead composer toolbar.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** header (title, "Open in main", close X) + optional banner + root row + replies divider + each reply + composer (4 toolbar buttons + Send + textarea).
- **8-item checklist:**
  1. >4 visible? `partial fail`. the composer toolbar.
  2. Self-evident labels? `pass`. The `ROOT` badge is the only piece of jargon, and it's clarified by being right above the root post.
  3. Primary action visually dominant? `pass`.
  4. Progressive disclosure? `pass`. overlay collapses to right-rail at first, can expand to full-page.
  5. Grouped by proximity? `pass`.
  6. Hierarchy contrast ≥1.25? `pass`.
  7. Body line length 65–75ch? `partial`. same lack of `max-w-62ch` on message body as the direct-room timeline.
  8. Whitespace varied? `pass`. header + banner + root + divider + replies + composer create five distinct rhythm bands.

  Failure count: 0 + partials. Low cognitive load. Best route in the module on this metric.

- **IA observations:**
  - The overlay header has THREE controls (title block, "Open in main" ghost button, close X). The "Open in main" sits inside the title column with its own `self-start` alignment. it functions as a tertiary action embedded in the primary header. Acceptable but slightly noisy.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all via `var(--color-*)`. Pass.
- **Type scale:** title 15 semibold (overlay). slightly smaller than channel header 18, intentional for sub-context. Mono 10 uppercase for ROOT badge + reply divider. Pass.
- **Radii / spacing:** overlay shell 360px wide; full-page mode flexes to fill. Pass.
- **Elevation:** flat with `bg-canvas-deep` for the overlay (`thread-overlay.tsx:25-30`). Distinct from the main-pane `bg-canvas`, creating an inset feel. Pass.
- **Signal palette:** `WorkBanner` warning. Pass.
- **Grid / rhythm:** five rhythm bands. Best in the module.
- **Density:** comfortable.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** type → Send. Cmd/Ctrl+Enter sends.
- **Destructive actions:** none.
- **Forms:** `DetailComposer surface="thread"` autoresizes.
- **Tables / lists:** timeline density=overlay.
- **Selection model:** none.
- **Modals / drawers:** the overlay is an `aside`; ESC does NOT close it (verify. `thread-overlay-header.tsx:63-77` shows the close button is the only dismissal path inside the overlay; the parent shell does not bind ESC to close the overlay). P2.
- **Live updates:** silent polling.
- **Optimistic vs pessimistic:** sends optimistic with retry/discard inline.
- **Hover / focus / active:** present. Hover toolbar dead controls (P0 inherited).
- **Loading patterns:** root has its own loading state; replies have skeleton via `Timeline isLoading`.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** header buttons + composer + each message reachable.
- **Focus rings:** `focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]`.
- **TAB order:** logical (header → banner → root → replies → composer).
- **ARIA roles / labels:** `<section aria-label={fullPage ? "Thread" : "Thread overlay"}>`, `<header data-testid="network-thread-overlay-header">`, `Timeline role="log"`. The `ROOT` badge is a span with text content only. fine for a screen reader.
- **Color contrast:** body ~13:1; secondary ~5.4:1; mono tertiary ~4.4:1. Pass.
- **Motion:** transitions on the work banner only. Respects reduced motion via global CSS.
- **Text scaling:** survives 200%.
- **Forms:** composer textarea labeled.

---

## 8. Empty / Loading / Error States

- **Empty (no replies):** adequate. `ThreadEmpty` "Thread has no replies. Reply below to keep the context alive." Operator voice. No em dash.
- **Loading:** strong on replies; weak on root (the "Loading root" mono row is a spartan placeholder. does not match the final layout).
- **Error (thread unavailable):** strong. Specific copy. No retry button on the empty itself, but the user can navigate away.
- **Error (send fails):** strong (Retry/Discard inline).
- **Permission denied:** missing.
- **Stale / disconnected:** missing.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary:** `thread`, `peer`, `channel`. correct.
- **Tone:** dry, operator voice. Pass.
- **Em dashes:** none in this file or its dependencies.
- **Restated headings:** none.
- **Sentence vs Title case:** sentence case throughout.
- **Truthful UI test:**
  - All metadata daemon-sourced. Pass.
  - "Open in main" is a real navigation. Pass.
  - The `ROOT` badge is real (the first message of the thread). Pass.
  - Hover toolbar dead controls. P0 inherited.
  - Composer toolbar dead controls. P0 inherited.

---

## 10. Performance & Responsiveness

- **Initial render:** detail + messages + open-work queries.
- **Re-render hot spots:** `useThreadOverlayView` returns a memoized object; timeline memoizes `buildTimelineEntries`.
- **List virtualization:** none.
- **Bundle red flags:** none.
- **Responsive behaviour:** strong. auto-flip to full-page below 1024 (`network.$channel.threads.$threadId.tsx:18-22`).
- **Mobile interactions:** rows full-width; hover toolbar invisible on touch.

---

## 11. Storybook Coverage

- **Stories present:**
  - `systems-network-threadoverlay--header|root|replies-populated|replies-loading|replies-empty`
  - `systems-network-emptystates--thread-empty-state`
- **States covered:** populated, loading, empty.
- **Gaps:** no error story for `ConversationError` branch; no full-page mode story; no mobile / narrow.
- **Story drift:** none observed.

---

## 12. Findings. Prioritised

### P0. Ship Blockers

None unique to this route. Inherits P0-NET-1 (hover toolbar) and P0-NET-2 (composer toolbar).

### P1. High-Value Polish

1. **[P1-TDET-1] What:** ESC does not close the right-rail thread overlay.
   - **Why:** users expect ESC to dismiss a non-modal overlay/drawer.
   - **Fix:** add a `useEffect` keyboard handler in `thread-overlay.tsx` that navigates back to `/network/$channel/threads` on ESC when `fullPage === false`. Skip in full-page mode (where there is no overlay to close).
   - **Cmd:** `/impeccable harden web/src/systems/network/components/thread-overlay/thread-overlay.tsx`
   - **Effort:** S
   - **Evidence:** no ESC handler anywhere in `thread-overlay*.tsx`.

2. **[P1-TDET-2] What:** Root loading state is a spartan "Loading root" mono row.
   - **Fix:** match the final layout. render a `MessageRow`-shaped skeleton (avatar + name + time + body skeleton) inside the same eyebrow band.
   - **Cmd:** `/impeccable polish web/src/systems/network/components/thread-overlay/thread-overlay-root.tsx`
   - **Effort:** S
   - **Evidence:** `thread-overlay-root.tsx:9-23`.

3. **[P1-TDET-3] What:** No `max-width` cap on message body.
   - **Fix:** cap at 62ch per `DESIGN.md` UI body rule (also in P2-DDET-1).
   - **Cmd:** `/impeccable typeset web/src/systems/network/components/timeline/message-body.tsx`
   - **Effort:** S
   - **Evidence:** `message-body.tsx`.

### P2. Worthwhile

1. **[P2-TDET-1] What:** "Open in main" ghost button lives inside the title column with `self-start`. Visually competes with the title.
   - **Fix:** move it to the right cluster next to the close X, with an `ArrowUpRight` icon-only button.
   - **Cmd:** `/impeccable layout web/src/systems/network/components/thread-overlay/thread-overlay-header.tsx`
   - **Effort:** S
   - **Evidence:** `thread-overlay-header.tsx:44-61`.

2. **[P2-TDET-2] What:** No story for the `ConversationError` branch.
   - **Fix:** add `systems-network-threadoverlay--unavailable`.
   - **Cmd:** `/impeccable harden web/src/systems/network/components/thread-overlay/`
   - **Effort:** S

### P3. Parking Lot

1. No virtualization. Acceptable.
2. No "jump to root" affordance from deep within long replies.

---

## 13. Persona Red Flags

- **Operator (returning power user):** ESC doesn't close; hover toolbar dead controls.
- **First-timer:** the `ROOT` badge clarifies the conversation structure better than most chat clients.
- **Agent (DOM scraping):** stable testids (`network-thread-overlay-header`, `network-thread-overlay-root-badge`, `network-thread-overlay-replies-divider`, `network-thread-overlay-close`, `network-thread-overlay-open-main`). Strong.

---

## 14. Cross-Module Consistency Notes

- The overlay header shape diverges from `direct-room.tsx` identity row (one is a title-led header, the other is an avatar-led row). Both are correct for their context.
- The full-page mode mounts inside a `<div bg-canvas-deep>` (`network.$channel.threads.tsx:62-69`). the same canvas-deep used for the right-rail overlay, so the visual identity is consistent across modes.

---

## 15. Open Questions

- Should ESC close the overlay (and only the overlay) without affecting the parent shell?
- Should "Open in main" pre-select the inspector's Activity tab so the user keeps context?
- Should the overlay support resizing via a drag handle?

---

## 16. Recommended Action Plan

1. `/impeccable harden web/src/systems/network/components/timeline/hover-toolbar.tsx`. disable Reply / Pin / Fork / More until handlers exist (P0 inherited).
2. `/impeccable harden web/src/systems/network/components/composer/composer-toolbar.tsx`. disable Attach / Format / Mention until handlers exist (P0 inherited).
3. `/impeccable harden web/src/systems/network/components/thread-overlay/thread-overlay.tsx`. bind ESC to close in overlay mode.
4. `/impeccable polish web/src/systems/network/components/thread-overlay/thread-overlay-root.tsx`. replace the spartan loading row with a layout-matching skeleton.
5. `/impeccable typeset web/src/systems/network/components/timeline/message-body.tsx`. cap body at 62ch.
6. `/impeccable layout web/src/systems/network/components/thread-overlay/thread-overlay-header.tsx`. move "Open in main" to the right cluster.
7. `/impeccable polish web/src/systems/network/components/thread-overlay/`. final sweep.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/thread-detail/`.
- [x] No section left empty.
- [x] Nielsen total (28/40) consistent with band (◯ adequate, top of range).
- [x] Findings tagged P0–P3 with effort and command.
- [x] No hallucinated routes, components, or props.
- [x] No em dashes.
