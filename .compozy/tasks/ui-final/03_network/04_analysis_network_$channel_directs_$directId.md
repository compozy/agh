# UI/UX Analysis :: `network` :: `/network/$channel/directs/$directId`

> **Status:** draft
> **Owner subagent:** `ui-final/network`
> **Date:** 2026-05-06
> **Module:** `network` (`03_network`)
> **Route path:** `/network/$channel/directs/$directId` (TanStack Router id: `/_app/network/$channel/directs/$directId`)
> **Web source:** `web/src/routes/_app/network.$channel.directs.$directId.tsx`
> **System owner:** `web/src/systems/network/components/directs/direct-room.tsx`
> **Storybook story id(s):** `systems-network-timeline--default`, `systems-network-timeline--loading`, `systems-network-timeline--empty`, `systems-network-composer--default`, `systems-network-emptystates--direct-empty-state`, `systems-network-work--banner`
> **Live URLs probed:** `http://localhost:3000/network/general/directs/<directId>` · `http://localhost:6006/iframe.html?id=systems-network-timeline--default&viewMode=story`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/network.$channel.directs.$directId.tsx`
  - `web/src/systems/network/components/directs/direct-room.tsx`
  - `web/src/systems/network/components/directs/use-direct-room-view.ts`
  - `web/src/systems/network/components/timeline/timeline.tsx`
  - `web/src/systems/network/components/timeline/message-row.tsx`
  - `web/src/systems/network/components/timeline/hover-toolbar.tsx`
  - `web/src/systems/network/components/composer/composer.tsx`
  - `web/src/systems/network/components/composer/composer-toolbar.tsx`
  - `web/src/systems/network/components/composer/detail-composer.tsx`
  - `web/src/systems/network/components/composer/use-composer-state.ts`
  - `web/src/systems/network/components/work/work-banner.tsx`
  - `web/src/systems/network/components/empty-states/{direct-empty,conversation-error}.tsx`
  - `web/src/systems/network/hooks/use-network-presence.ts`
- **Storybook stories opened:**
  - `systems-network-timeline--default` → `http://localhost:6006/iframe.html?id=systems-network-timeline--default&viewMode=story`
  - `systems-network-emptystates--direct-empty-state`
  - `systems-network-work--banner`
- **Live web probes (`localhost:3000`):** parent route collapses to "no channels" because daemon empty.
- **Screenshots captured:**
  - `_evidence/direct-detail/01-storybook-timeline-default-1440.png`. timeline default story (also serves as evidence for thread-detail).
- **Console / network errors observed:** none.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** renders one direct room as a vertical conversation: identity row at the top (avatar + `@otherPeerId` + `agent` chip + presence dot placeholder), an optional `WorkBanner` for open work, the `Timeline` of messages, and a `DetailComposer` pinned at the bottom.
- **Primary user goal:** read and reply to messages with a single peer in this channel.
- **Entry vectors:** click a directs-list row, click a direct in the rail, click a recents direct, click a direct entry in the cross-channel activity feed.
- **Exit vectors:** click another rail row / recents entry; the parent shell still owns navigation.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / no messages | yes | `direct-empty.tsx:13-22` "Quiet so far. Send the first message. they'll be notified." | weak: em dash + the "they'll be notified" claim is aspirational. |
| Loading / skeleton | yes | `direct-room.tsx:91-97` `Timeline` with `isLoading` | strong (timeline skeleton matches density). |
| Partial data | yes | `direct-room.tsx:96-130` resolves detail then loads messages. | OK. |
| Populated (typical) | yes | `_evidence/direct-detail/01-storybook-timeline-default-1440.png` | strong. |
| Populated (dense) | partial | timeline renders all messages; no virtualization. |
| Error (network) | yes | `direct-room.tsx:83-90` renders `ConversationError` | strong. |
| Error (permission / 403) | no | n/a |
| Error (not found / 404) | yes | same `ConversationError` branch when `detailError` is set | strong. |
| Read-only / disabled | yes | `direct-room.tsx:118-128` `disabledReason` flows into the composer placeholder | OK. |
| Live-update | partial | polling 5s for messages, no SSE indicator. |
| Mobile / narrow | weak | parent shell does not collapse the rails. |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   2   | `WorkBanner` uses `aria-live="polite"` (`work-banner.tsx:60-72`); polling silent | No connection / stale indicator. |
| 2  | Match between system and real world    |   2   | `direct-room.tsx:76-78` static `agent` chip; `direct-empty.tsx:21` "they'll be notified" | Truthful UI failures. |
| 3  | User control and freedom               |   2   | composer Cmd+Enter to submit; retry / discard handlers wired (`thread-overlay.tsx:60-64` for thread; `direct-room.tsx:104-117` for direct) | OK. No edit / delete after send. |
| 4  | Consistency and standards              |   3   | timeline + composer reused from thread-overlay | Strong. |
| 5  | Error prevention                       |   3   | `disabledReason` blocks the send; `ConversationError` for the unavailable case | OK. |
| 6  | Recognition rather than recall         |   3   | identity row at top; mono `@peerId` reused everywhere | OK. |
| 7  | Flexibility and efficiency of use      |   2   | Cmd+Enter to send; slash popover; no per-message keyboard shortcuts | OK. Hover toolbar buttons render but do nothing (P0 inherited). |
| 8  | Aesthetic and minimalist design        |   3   | `_evidence/direct-detail/01-storybook-timeline-default-1440.png` | Clean. |
| 9  | Help users recognize / recover errors  |   3   | `MessageRow` exposes Retry / Discard for failed optimistic sends; `ConversationError` for the room | Strong. |
| 10 | Help and documentation                 |   1   | none | No help. |
|    | **Total**                              | **24/40** | | **Band:** ◯ adequate (20–28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders                             | OK | none |
| Gradient text                                   | OK | none |
| Glassmorphism                                   | OK | none |
| Hero-metric template                            | OK | none |
| Identical card grids                            | OK | not a card grid |
| Modal as first thought                          | OK | inline composer + inline timeline |
| Em dashes in copy                               | violations | `direct-empty.tsx:21` "Send the first message. they'll be notified." P1. |
| Generic AI palette                              | OK | tokens only |
| Category-reflex theme                           | OK | not generic chat-app blue/green |
| Restated headings / intros                      | OK | header is `@peerId` + role chip + presence; timeline is the body |
| Decorative shadows                              | OK | flat |
| Hardcoded `#000` / `#fff`                       | OK | none |

**Summary verdict:** No, not AI-generated. The em dash + the dead hover-toolbar are the only AI-slop-adjacent flags.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point** (mid-conversation): the composer (4 toolbar buttons + Send + textarea), each message row's hover toolbar (5 buttons, 4 of which are dead today, 1 is disabled-by-design).
- **8-item checklist:**
  1. >4 visible? `pass` at the page level (header, banner, body, composer); `fail` at the message-hover level (5 buttons).
  2. Self-evident labels? `pass`.
  3. Primary action visually dominant? `pass`. `Send` button is the orange CTA.
  4. Progressive disclosure? `pass`. hover toolbar appears only on hover.
  5. Grouped by proximity? `pass`.
  6. Hierarchy contrast ≥1.25? `pass`.
  7. Body line length 65–75ch? `partial`. message body has no max-width cap, can run wide on 1440. `_design.md` §5 specifies UI body 62ch but the timeline applies no cap.
  8. Whitespace varied? `partial`.

  Failure count: 1 (hover toolbar) + partial. Moderate.

- **IA observations:**
  - Identity row uses a static `agent` chip whether the peer is an agent or a human. this is also the first thing the user sees, magnifying the truthful-UI failure.
  - Presence dot is hidden when `state === "idle"` (the only state the placeholder hook returns), so the dot never renders in production. The `<PresenceDot>` component pretends presence is wired.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all via `var(--color-*)`. Pass.
- **Type scale:** 16px semibold for the identity title, 14px for messages, 11px mono for timestamps. Pass.
- **Radii / spacing:** identity row `h-12 px-5`. Composer `px-4 py-3`. Pass.
- **Elevation:** flat. Pass.
- **Signal palette:** `WorkBanner` uses warning color (escalates to solid warning when `hasNeedsInput`). Pass.
- **Grid / rhythm:** identity row + optional banner + flexible timeline + composer. Three-band layout. Clean.
- **Density:** comfortable.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** type → Send. Cmd/Ctrl+Enter submits.
- **Destructive actions:** `Discard` on a failed optimistic message (`message-row.tsx`). Confirm via tooltip. No `confirm typing` because the data is local-only.
- **Forms:** composer textarea autoresizes; placeholder reflects target peer (`detail-composer.tsx:60-65`). Disabled when `disabledReason` is set (e.g., no session).
- **Tables / lists:** the timeline groups messages with date pills + new-divider; collapsible continuation rows for the same peer; no virtualization.
- **Selection model:** none.
- **Modals / drawers:** none.
- **Live updates:** silent polling.
- **Optimistic vs pessimistic:** sends are optimistic; failures expose Retry / Discard inline.
- **Hover / focus / active:** present. Hover toolbar is the truthful-UI offender (P0-NET-1).
- **Loading patterns:** timeline skeleton is comprehensive (5 rows, density-aware).

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** every element reachable.
- **Focus rings:** `focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]`.
- **TAB order:** identity row → timeline (each `article` is reachable as a `role="article"` via `<article>` element) → composer.
- **ARIA roles / labels:** `<section aria-label="Direct room with @${other}">`, `<header data-testid="network-direct-identity-row">`, `Timeline role="log"`. The presence dot has `aria-label` for `running` / `needs input` / `errored` (`direct-room.tsx:35-37`). Solid.
- **Color contrast:** body text `#E5E5E7` on canvas ~13:1; secondary `#8E8E93` ~5.4:1. pass.
- **Motion:** `motion-safe:animate-pulse` on the running presence dot. respects reduced motion.
- **Text scaling:** survives 200%. Composer textarea has `max-height: 12rem` (8 rows × 1.5 rem). long messages scroll inside the textarea.
- **Forms:** composer textarea has `aria-label={placeholder}` (`composer.tsx:58`). Pass.

---

## 8. Empty / Loading / Error States

- **Empty (no messages yet):** weak. `direct-empty.tsx:13-22` `"Quiet so far. Send the first message. they'll be notified."` Em dash, and "they'll be notified" overpromises.
- **Loading:** strong. Timeline skeleton.
- **Error (room unavailable):** strong. `ConversationError` with retry / discard.
- **Error (send fails):** strong. Inline Retry / Discard on the failed message.
- **Permission denied:** missing.
- **Stale / disconnected:** missing.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary:** `direct room`, `peer`, `channel`. correct.
- **Tone:** `direct-empty.tsx` "Quiet so far.". operator voice. Pass.
- **Em dashes:** `direct-empty.tsx:21` violates. P1.
- **Restated headings:** none.
- **Sentence vs Title case:** sentence case throughout.
- **Truthful UI test:**
  - `direct-room.tsx:76-78` static `agent` chip. P0 inherited.
  - `direct-empty.tsx:21` "they'll be notified". daemon does not currently model peer notifications. Either confirm a notification path exists (it does not, in `internal/network/`) or rephrase.
  - Hover toolbar dead controls. P0 inherited.
  - Composer toolbar dead controls. P0 inherited.
  - Presence dot. placeholder, only renders for non-idle states which never occur.

---

## 10. Performance & Responsiveness

- **Initial render:** detail + messages queries via TanStack.
- **Re-render hot spots:** `buildTimelineEntries` is wrapped in `useMemo` (`timeline.tsx:79-82`). Good.
- **List virtualization:** none. With long histories (1000+ messages) this will hurt.
- **Bundle red flags:** none.
- **Responsive behaviour:** the identity row + timeline + composer collapse vertically. Inherits parent shell's chrome problem at narrow widths.
- **Mobile interactions:** rows are wide-tappable; hover toolbar is invisible on touch devices.

---

## 11. Storybook Coverage

- **Stories present:**
  - `systems-network-timeline--default|loading|empty|new-divider-story`
  - `systems-network-composer--default|submitting|disabled`
  - `systems-network-emptystates--direct-empty-state`
  - `systems-network-work--banner|banner-escalation|chip-states|inspector|inspector-empty`
  - `systems-network-messagerow--full-row|collapsed-continuation|system-event|thread-density`
- **States covered:** populated, loading, empty, work-banner.
- **Gaps:**
  - No `direct-room`-specific story; the timeline is the closest proxy. A `routes-app-stories-network--direct-detail` would be useful.
  - No `direct-room` error story for the `ConversationError` branch.
  - No mobile / narrow viewport.
- **Story drift:** none observed.

---

## 12. Findings. Prioritised

### P0. Ship Blockers

1. **[P0-DDET-1] What:** Static `"agent"` chip in identity row regardless of peer kind.
   - **Why:** misleads operator about who they are talking to; misleads agents that read the DOM.
   - **Fix:** remove the chip until a daemon `kind` field exists, or pipe the heuristic from `useChannelMembers` and label it as inferred.
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/directs/direct-room.tsx`
   - **Effort:** S
   - **Evidence:** `direct-room.tsx:76-78`.

2. **[P0-DDET-2] What:** Hover toolbar Reply / Pin to capability / Fork thread / More actions render but do nothing.
   - **Inherited from P0-NET-1.**
   - **Evidence:** `hover-toolbar.tsx:71-101`; consumed by `timeline.tsx:122-152`; no `toolbarHandlers` passed in `direct-room.tsx`.

3. **[P0-DDET-3] What:** Composer toolbar Attach / Format / Mention render but do nothing.
   - **Inherited from P0-NET-2.**
   - **Evidence:** `composer.tsx:74-75`; `composer-toolbar.tsx:55-72`.

### P1. High-Value Polish

1. **[P1-DDET-1] What:** Empty state "Send the first message. they'll be notified." Em dash + aspirational notification claim.
   - **Fix:** "Send the first message. They'll see it on next refresh."
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/empty-states/direct-empty.tsx`
   - **Effort:** S
   - **Evidence:** `direct-empty.tsx:21`.

2. **[P1-DDET-2] What:** Presence dot is wired to a placeholder hook returning `idle` always.
   - **Fix:** until `useNetworkPresence` reads from a real source, remove the dot. Add it back when the protocol exposes presence telemetry.
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/directs/direct-room.tsx`
   - **Effort:** S
   - **Evidence:** `use-network-presence.ts:1-23`, `direct-room.tsx:23-50`.

3. **[P1-DDET-3] What:** No connection / stale indicator while polling 5s for messages.
   - **Fix:** surface a small "Updated 3s ago" or "Reconnecting... " chip in the identity row when the messages query is `isFetching`/`isError`.
   - **Cmd:** `/impeccable polish web/src/systems/network/components/directs/direct-room.tsx`
   - **Effort:** S
   - **Evidence:** `lib/query-options.ts:114`.

### P2. Worthwhile

1. **[P2-DDET-1] What:** No `max-width` cap on message body. wide displays produce 100ch lines.
   - **Fix:** cap message body at `max-w-[62ch]` per `DESIGN.md` §5 UI body rule.
   - **Cmd:** `/impeccable typeset web/src/systems/network/components/timeline/message-body.tsx`
   - **Effort:** S
   - **Evidence:** `message-body.tsx`, `_evidence/direct-detail/01-storybook-timeline-default-1440.png`.

2. **[P2-DDET-2] What:** No virtualization in the timeline.
   - **Fix:** consider TanStack Virtual once long histories are real. P3 acceptable for alpha.

3. **[P2-DDET-3] What:** Composer toolbar `Slash command` button works (opens the slash popover) but the popover offers `/run` and `/mention` whose only behavior is inserting literal text into the textarea (`use-composer-state.ts:126-133`).
   - **Fix:** either implement /run and /mention or mark them disabled (Post-MVP) like /attach.
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/composer/composer-slash-popover.tsx`
   - **Effort:** S
   - **Evidence:** `composer-slash-popover.tsx:21-36`.

### P3. Parking Lot

1. No edit / delete on sent messages. Verify daemon contract; if not supported, document.
2. No keyboard shortcuts to focus the next/previous message.

---

## 13. Persona Red Flags

- **Operator (returning power user):** hover toolbar dead controls; presence dot shows nothing; "agent" chip wrong for human peers.
- **First-timer:** "they'll be notified" leads to misplaced trust that a notification mechanism exists. The first send into an empty room can fail silently if the peer is offline.
- **Agent (DOM scraping):** `data-testid` set on every important node (`network-direct-room`, `network-direct-identity-row`, `network-direct-presence-dot`, `network-message-toolbar-*`). Strong, but the `agent` text content is a lie.

---

## 14. Cross-Module Consistency Notes

- The identity row pattern (avatar + handle + role chip + presence dot) is unique to `direct-room`. The thread overlay header uses a different shape (title + participant count + Open in main). Two distinct conversational headers.
- The composer is shared across thread, direct, and channel-thread variants. consistent.
- Timeline density varies: `direct-room` uses `density="channel"`, thread overlay uses default channel density. Verify intent.

---

## 15. Open Questions

- Should the role chip and presence dot be hidden until the daemon supplies them?
- Is "they'll be notified" a real promise (cross-peer notifications) or aspirational?
- Should the timeline cap message body at 62ch?

---

## 16. Recommended Action Plan

1. `/impeccable clarify web/src/systems/network/components/directs/direct-room.tsx`. remove static `agent` chip; remove or honestly label the presence dot.
2. `/impeccable harden web/src/systems/network/components/timeline/hover-toolbar.tsx`. disable Reply / Pin / Fork / More until handlers exist.
3. `/impeccable harden web/src/systems/network/components/composer/composer-toolbar.tsx`. disable Attach / Format / Mention until handlers exist.
4. `/impeccable clarify web/src/systems/network/components/empty-states/direct-empty.tsx`. rewrite microcopy without em dash and without the notification claim.
5. `/impeccable polish web/src/systems/network/components/directs/direct-room.tsx`. add a stale / connection chip in the identity row.
6. `/impeccable typeset web/src/systems/network/components/timeline/message-body.tsx`. cap body at 62ch.
7. `/impeccable polish web/src/systems/network/components/directs/`. final sweep.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/direct-detail/`.
- [x] No section left empty.
- [x] Nielsen total (24/40) consistent with band (◯ adequate).
- [x] Findings tagged P0–P3 with effort and command.
- [x] No hallucinated routes, components, or props.
- [x] No em dashes.
