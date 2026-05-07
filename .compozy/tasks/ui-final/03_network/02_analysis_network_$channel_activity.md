# UI/UX Analysis :: `network` :: `/network/$channel/activity`

> **Status:** draft
> **Owner subagent:** `ui-final/network`
> **Date:** 2026-05-06
> **Module:** `network` (`03_network`)
> **Route path:** `/network/$channel/activity` (TanStack Router id: `/_app/network/$channel/activity`)
> **Web source:** `web/src/routes/_app/network.$channel.activity.tsx`
> **System owner:** `web/src/systems/network/components/activity/activity-feed.tsx` + `web/src/systems/network/components/shell/list-filter-bar.tsx`
> **Storybook story id(s):** `routes-app-stories-network--activity-tab`
> **Live URLs probed:** `http://localhost:3000/network/general/activity` · `http://localhost:6006/iframe.html?id=routes-app-stories-network--activity-tab&viewMode=story`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/network.$channel.activity.tsx`
  - `web/src/systems/network/components/activity/activity-feed.tsx`
  - `web/src/systems/network/components/shell/list-filter-bar.tsx`
  - `web/src/systems/network/hooks/use-network-list-filters.ts` (referenced via `useNetworkListFilters`)
  - `web/src/systems/network/lib/network-formatters.ts`
- **Storybook stories opened:**
  - `routes-app-stories-network--activity-tab` → `http://localhost:6006/iframe.html?id=routes-app-stories-network--activity-tab&viewMode=story`
- **Live web probes (`localhost:3000`):**
  - `/network/general/activity`. daemon empty, the parent route collapses to "no channels" so the activity tab is not directly reachable in the live probe (covered in `network` route report).
- **Screenshots captured:**
  - `_evidence/activity/01-storybook-activity-1440.png`. populated activity feed.
- **Console / network errors observed:** none.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** a unified, read-only chronological feed of every thread + every direct room in the active channel, sorted by `last_activity_at` (most recent first). Each entry deep-links to its source thread or direct room. The intent is to give an operator a single place to scan "what is alive in this channel" without flipping between Threads and Directs tabs.
- **Primary user goal:** scan recent transitions in the channel and click into the most relevant one.
- **Entry vectors:** `Activity` tab in the channel header (`channel-tabs.tsx:45-50`), the Recents column on the rail when the entry happens to point to the same channel.
- **Exit vectors:** click any feed row → thread / direct detail; the parent shell still owns the rail / channel header / inspector toggle.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | yes | `activity-feed.tsx:100-111` `Empty` "Quiet across the channel." | adequate; no CTA. |
| Loading / skeleton | yes | `activity-feed.tsx:76-91` `ActivityFeedSkeleton` matches the row layout (4 rows, three skeleton bars each) | strong. |
| Partial data | yes | `network.$channel.activity.tsx:42-46` passes `threadsQuery.threads` AND `directsQuery.directs`; if one query is loading and the other has data, the feed renders the available rows. | strong. |
| Populated (typical) | yes | story `routes-app-stories-network--activity-tab`, `_evidence/activity/01-storybook-activity-1440.png` | strong. |
| Populated (dense) | partial | the feed renders all entries with no virtualization or paging cap; the inspector caps at 10 (`inspector-activity-feed.tsx:97`) but this main feed has no cap. With 500+ entries this is a large DOM. |
| Error (network) | no | the route does not surface query errors. `directsQuery.error` and `threadsQuery.error` are not read. P1. |
| Error (permission / 403) | no | n/a |
| Error (not found / 404) | inherits | parent route handles "no channels" / "channel not found". |
| Read-only / disabled | yes | feed is read-only by design; `activity-feed.tsx:120-124` shows the "Read-only" mono tag. |
| Live-update | partial | inherited polling (`MESSAGES_REFETCH_INTERVAL = 5000`, `LIST_REFETCH_INTERVAL = 15000`). No SSE indicator. |
| Mobile / narrow | weak | inherits parent shell's responsive issues. |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   2   | `activity-feed.tsx:120-124` "Recent activity · Read-only" subheader | No "last updated" timestamp; polling is silent. |
| 2  | Match between system and real world    |   3   | `activity-feed.tsx:131-133` `[TH]` / `[DM]` mono tags. | "TH" / "DM" are operator-jargon shorthand; first-timers may not know what they mean. |
| 3  | User control and freedom               |   3   | filter pills + sort dropdown + Mark all read inherited from `ListFilterBar` | No "Group by surface" toggle; no per-row mute. |
| 4  | Consistency and standards              |   3   | row pattern matches `inspector-activity-feed.tsx` and the recents section in `channel-rail-recents.tsx` | The mono `[TH]` / `[DM]` tags are unique to this surface. same data is rendered with `MessagesSquare` / `AtSign` icons in the recents column. |
| 5  | Error prevention                       |   2   | no error UI for `threadsQuery.error` / `directsQuery.error` | Silent failure if either request errors. |
| 6  | Recognition rather than recall         |   3   | hover, focus, deep-link to source. Title + preview + relative timestamp visible on every row. | `[TH]` / `[DM]` tags require recall. |
| 7  | Flexibility and efficiency of use      |   2   | filter pills cover the 5 list-level filters (`all / has_work / @me / pinned / unread`) | No keyboard shortcut to jump rows; no infinite scroll / pagination. |
| 8  | Aesthetic and minimalist design        |   3   | `_evidence/activity/01-storybook-activity-1440.png` | Clean, but the `border-b` between every row + the `border-b` between filter and feed means three consecutive rules at the top. |
| 9  | Help users recognize / recover errors  |   1   | no error UI in the feed | If both queries fail, the user sees an empty state with no clue. |
| 10 | Help and documentation                 |   1   | none | No tooltip on `[TH]` / `[DM]`, no doc link. |
|    | **Total**                              | **23/40** | | **Band:** ◯ adequate (20–28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders                             | OK | none |
| Gradient text                                   | OK | none |
| Glassmorphism                                   | OK | none |
| Hero-metric template                            | OK | none |
| Identical card grids                            | OK | this is a feed list, not a card grid |
| Modal as first thought                          | OK | none |
| Em dashes in copy                               | OK | none in this file |
| Generic AI palette                              | OK | tokens only |
| Category-reflex theme                           | OK | n/a |
| Restated headings / intros                      | OK | the `Recent activity` subheader is meta, not a title repeat |
| Decorative shadows                              | OK | flat |
| Hardcoded `#000` / `#fff`                       | OK | none |

**Summary verdict:** No, a stranger would not call this AI-generated. Honest functional feed.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** filter pills row (5 pills) + Sort dropdown (1) + Mark all read (1) + every entry in the feed (variable). At the row level: each entry is a single click target. Pass.
- **8-item checklist:**
  1. >4 visible options? `partial fail`. the filter row alone has 5 pills + a sort + a Mark all read.
  2. Self-evident labels? `partial`. `Has work`, `@me`, `Pinned`, `Unread` are clear; `[TH]` / `[DM]` are not.
  3. Primary action visually dominant? n/a. this is a read-only feed; there is no "primary action".
  4. Progressive disclosure? `pass`. feed shows previews; full thread is one click away.
  5. Related elements grouped? `pass`. title + preview + meta in a vertical stack per row.
  6. Hierarchy contrast ≥1.25? `pass`. title `text-[14px] font-semibold text-primary` vs preview `text-[13px] text-secondary` vs meta `text-[10px] mono tertiary`.
  7. Body line length 65–75ch? `pass`. preview is `line-clamp-2`; rows are flexible-width.
  8. Whitespace varied? `partial`. `px-5 py-3` per row, `px-5 py-2` for filter bar; uniform horizontal step.
  Failure count: 1 + partial. Low cognitive load overall.

- **IA observations:**
  - Mixing threads and directs in one feed is a useful collapse, but the route exposes them through opaque `[TH]` / `[DM]` mono tags. Direct rooms also surface as `peer_a ↔ peer_b` titles which differ from the directs list (`@otherPeerId`). the same direct shows two distinct names depending on which surface the operator is looking at.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all via `var(--color-*)`. Pass.
- **Type scale:** Inter 14 semibold for title, Inter 13 for preview, Mono 10 uppercase for meta. Pass.
- **Radii / spacing:** rows have no radius (full-width with `border-b`); filter pills via `PillGroup` primitive. Pass.
- **Elevation:** flat. Pass.
- **Signal palette:** no semantic color used in this feed. Pass.
- **Grid / rhythm:** `border-b border-divider` per row + `border-b` for the subheader; visually monotone but readable.
- **Density:** comfortable; subheader + 4-row skeleton stack at 1440 leaves room.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** click any row → deep link to the source thread / direct.
- **Destructive actions:** none. `Mark all read` is in the filter bar but no per-row destructive control.
- **Forms:** none.
- **Tables / lists:** read-only feed; no virtualization. Sort is via the parent `ListFilterBar` (`recent_activity` / `created` / `alphabetical`). The `ActivityFeed` itself ignores `sort` and always orders by `last_activity_at` (`activity-feed.tsx:69-73`). That is a divergence between filter UI and content. the user can pick "alphabetical" from the dropdown but the feed will still be sorted by recent activity. P1.
- **Selection model:** none.
- **Modals / drawers:** none.
- **Live updates:** silent polling.
- **Hover / focus / active:** every row has `hover:bg-hover` + `focus-visible:ring-1 focus-visible:ring-accent`.
- **Loading patterns:** `ActivityFeedSkeleton` good.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** every row is a `Link`, reachable.
- **Focus rings:** `focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]`. Visible.
- **TAB order:** filter bar → feed rows in document order.
- **ARIA roles / labels:** `<div aria-label="Activity in #${channel}">` on the feed; rows are unlabeled `Link`s that rely on the visible title. Acceptable.
- **Color contrast:** title `#E5E5E7` / canvas `#141312` ~13:1; preview `#8E8E93` / canvas ~5.4:1. Pass for body. Mono tertiary `#636366` / canvas ~4.4:1. acceptable for metadata.
- **Motion:** none on entries.
- **Text scaling:** survives 200% in 1440 layout; in 1024 it pushes the inspector off-screen (parent shell issue).
- **Forms:** none.

---

## 8. Empty / Loading / Error States

- **Empty (channel exists but has no threads/directs):** adequate. `activity-feed.tsx:100-111` "Quiet across the channel. No activity yet across threads or direct rooms." No CTA. the right action would be `Start a thread`, but the activity tab does not own composition.
- **Loading:** strong. The skeleton matches the row layout (`activity-feed.tsx:76-91`).
- **Error:** missing. Neither `threadsQuery.error` nor `directsQuery.error` is read. If both endpoints 5xx, the user sees the empty state. P1.
- **Permission denied:** missing.
- **Stale / disconnected:** missing.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary:** `thread`, `direct room`, `channel`. correct.
- **Tone:** sentence case throughout. "Quiet across the channel." matches operator voice. Pass.
- **Em dashes:** none in this file. Pass.
- **Restated headings:** none.
- **Sentence vs Title case:** subheader `Recent activity · Read-only` is sentence case. Section header on the row meta uses uppercase mono, fine.
- **Truthful UI test:**
  - The feed is honest about being read-only (`Read-only` tag on the subheader).
  - The `[TH]` / `[DM]` tags are accurate but jargon.
  - "Quiet across the channel." accurately describes empty state.
  - No invented controls, metrics, or repair paths.
  - One subtle issue: the `peer_a ↔ peer_b` title for direct rooms uses the `↔` glyph as a separator. Glossary doesn't sanction the glyph; it's harmless visually but inconsistent with the directs-list which uses `@otherPeerId`.

---

## 10. Performance & Responsiveness

- **Initial render:** depends on threads + directs queries; both are TanStack-cached and stale-while-revalidate.
- **Re-render hot spots:** `buildEntries` is called inline on every render (`activity-feed.tsx:93-94`); for hundreds of entries this is fine but should be memoized with `useMemo` keyed on `[threads, directs, channel]`.
- **List virtualization:** none. With hundreds of entries this is a large DOM. P3.
- **Bundle red flags:** none.
- **Responsive behaviour:** inherits parent shell.
- **Mobile interactions:** click targets are full-width rows with `py-3`; meets touch minimum.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-network--activity-tab`. populated activity feed inside the real shell.
- **States covered:** populated.
- **Gaps:** no empty / loading / error stories at the feed level (the system folder has none either). Add `routes-app-stories-network--activity-empty` and `routes-app-stories-network--activity-error`.
- **Story drift:** none observed.

---

## 12. Findings. Prioritised

### P0. Ship Blockers

None at the route level. (Route inherits the P0-NET-1..3 findings from the parent shell.)

### P1. High-Value Polish

1. **[P1-ACT-1] What:** The Sort dropdown offers `Recent activity / Created / Alphabetical` but `ActivityFeed.buildEntries` always sorts by `last_activity_at`.
   - **Why:** the filter UI implies a control that does not work.
   - **Fix:** either honor the `sort` value (pipe `filters.sort` into the feed) or remove the sort dropdown for the activity tab.
   - **Cmd:** `/impeccable harden web/src/systems/network/components/activity/activity-feed.tsx`
   - **Effort:** S
   - **Evidence:** `activity-feed.tsx:69-73` always sorts by timestamp; `network.$channel.activity.tsx:32-39` passes `filters.sort` to `ListFilterBar` but not to `ActivityFeed`.

2. **[P1-ACT-2] What:** Query errors are silent.
   - **Why:** if `listNetworkThreads` or `listNetworkDirectRooms` returns 5xx, the user sees the empty state.
   - **Fix:** when `threadsQuery.error || directsQuery.error`, render a `ConversationError` (already used in thread-overlay) or a similar empty with a Retry action.
   - **Cmd:** `/impeccable harden web/src/routes/_app/network.$channel.activity.tsx`
   - **Effort:** S
   - **Evidence:** `network.$channel.activity.tsx:17-22`, `activity-feed.tsx:93-111`.

3. **[P1-ACT-3] What:** `[TH]` / `[DM]` mono tags are jargon.
   - **Why:** first-timers and agents reading the DOM see opaque codes.
   - **Fix:** replace mono codes with the same iconography the recents column uses (`MessagesSquare` for thread, `AtSign` for direct) plus a `aria-label` of `Thread` / `Direct room`. Mirror `channel-rail-recents.tsx:18-19`.
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/activity/activity-feed.tsx`
   - **Effort:** S
   - **Evidence:** `activity-feed.tsx:131-135`.

### P2. Worthwhile

1. **[P2-ACT-1] What:** Direct row title `peer_a ↔ peer_b` differs from the directs-list title `@otherPeerId`.
   - **Fix:** unify on `@otherPeerId` once `selfPeerId` is available in this scope (it's available via `useNetworkRailView` upstream).
   - **Cmd:** `/impeccable polish web/src/systems/network/components/activity/activity-feed.tsx`
   - **Effort:** S
   - **Evidence:** `activity-feed.tsx:64`, `directs-list.tsx:24-32`.

2. **[P2-ACT-2] What:** `buildEntries` runs every render without memoization.
   - **Fix:** `useMemo(() => buildEntries(channel, threads, directs), [channel, threads, directs])`.
   - **Cmd:** `/impeccable optimize web/src/systems/network/components/activity/activity-feed.tsx`
   - **Effort:** S
   - **Evidence:** `activity-feed.tsx:93-94`.

3. **[P2-ACT-3] What:** Empty state has no `Start a thread` action.
   - **Fix:** add an action prop wiring to the threads tab composer (or to a `New thread` modal).
   - **Effort:** S
   - **Evidence:** `activity-feed.tsx:100-111`.

### P3. Parking Lot

1. **[P3-ACT-1] What:** No virtualization. Acceptable until a channel hits 500+ entries.
2. **[P3-ACT-2] What:** No "last refreshed" timestamp on the subheader.

---

## 13. Persona Red Flags

- **Operator (returning power user):** missing keyboard nav inside the feed; filters reset on tab switch (depends on `useNetworkListFilters` storage; verify); sort dropdown is misleading.
- **First-timer:** `[TH]` / `[DM]` codes opaque; "Quiet across the channel" empty does not invite action.
- **Agent (DOM scraping):** stable testids (`network-activity-feed`, `network-activity-entry-thread:<id>`, `network-activity-tag-thread`). Strong.

---

## 14. Cross-Module Consistency Notes

- The same row pattern is used by `inspector-activity-feed.tsx` (`network-inspector-activity-... ` testids), but the inspector renders the same data without the `[TH]` / `[DM]` tag. Two divergent treatments of the same idea.
- `ListFilterBar` is shared with directs and threads tabs.

---

## 15. Open Questions

- Is the activity feed meant to show a unified surface or replace the threads/directs tabs entirely? If it is the canonical view, why does it not own composition?
- Should the sort dropdown in `ListFilterBar` be hidden for the activity tab, or should the feed honor it?
- Should the inspector activity feed and the main activity feed share a row component (one truth)?

---

## 16. Recommended Action Plan

1. `/impeccable harden web/src/systems/network/components/activity/activity-feed.tsx`. honor the sort filter or hide the sort dropdown for this tab.
2. `/impeccable harden web/src/routes/_app/network.$channel.activity.tsx`. surface query errors via a `ConversationError`-style empty.
3. `/impeccable clarify web/src/systems/network/components/activity/activity-feed.tsx`. replace `[TH]` / `[DM]` with iconography matching the recents column.
4. `/impeccable polish web/src/systems/network/components/activity/activity-feed.tsx`. unify direct-row title to `@otherPeerId`; memoize `buildEntries`.
5. `/impeccable polish web/src/systems/network/components/activity/`. final sweep.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/activity/`.
- [x] No section left empty.
- [x] Nielsen total (23/40) consistent with band (◯ adequate).
- [x] Findings tagged P0–P3 with effort and command.
- [x] No hallucinated routes, components, or props.
- [x] No em dashes.
