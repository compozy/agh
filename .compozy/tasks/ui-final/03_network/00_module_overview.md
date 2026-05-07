# UI/UX Module Overview :: `03_network`

> **Status:** draft
> **Owner subagent:** `ui-final/network`
> **Date:** 2026-05-06
> **Module:** Network (`03_network`)
> **Scope:** All routes under `web/src/routes/_app/network*` and the system in `web/src/systems/network/`.

---

## 0. Module map

| # | Route | Source file | System owner |
|---|-------|-------------|--------------|
| 1 | `/network` | `web/src/routes/_app/network.tsx` | `web/src/systems/network/components/shell/` |
| 2 | `/network/$channel/activity` | `web/src/routes/_app/network.$channel.activity.tsx` | `... /components/activity/` + `... /components/shell/list-filter-bar.tsx` |
| 3 | `/network/$channel/directs` | `web/src/routes/_app/network.$channel.directs.tsx` | `... /components/directs/` |
| 4 | `/network/$channel/directs/$directId` | `web/src/routes/_app/network.$channel.directs.$directId.tsx` | `... /components/directs/direct-room.tsx` |
| 5 | `/network/$channel/threads` | `web/src/routes/_app/network.$channel.threads.tsx` | `... /components/threads/threads-list.tsx` + composer |
| 6 | `/network/$channel/threads/$threadId` | `web/src/routes/_app/network.$channel.threads.$threadId.tsx` | `... /components/thread-overlay/thread-overlay.tsx` |

The shell composes three columns: **channel rail** (left, 260px, `channel-rail.tsx`) → **main pane** (channel header + tabs + outlet content) → **right rail** (360px, `right-rail.tsx`) which hosts either the channel inspector (Members / Work / Activity tabs in `network-inspector.tsx`) or the thread overlay. Page chrome is `PageHeader` ("Network" + count) above the three-column shell.

Shared probes for every route, captured under `_evidence/<route-slug>/`:

- Live root probe at `http://localhost:3000/network`. daemon empty, no channels: `_evidence/network/01-root-empty-1440.png`. URL stays `/network` and the shell renders the "no channels yet" empty state inside the `NetworkShell` (`network.tsx:111-152`).
- Storybook (populated) probes hit `http://localhost:6006/iframe.html?id=... &viewMode=story` and are captured per route. The full story index is in `web/src/routes/_app/stories/-network.stories.tsx` plus the `systems-network-*` stories in `web/src/systems/network/components/stories/`.
- `npx impeccable --json web/src/systems/network/` returned `[]`. no deterministic AI-slop tells. See `_evidence/impeccable.json`.

---

## 1. Information architecture

The Network module is the densest shell in the app. Three things compete for attention:

1. **Discovery** (where to go). channel rail with `Channels`, `Direct Rooms` (scoped to active channel), `Recents` (cross-channel).
2. **Composition** (what to do). main pane with channel header (`Hash` + name + member-count + work-count + purpose), the `Threads / Directs / Activity` tab strip, and a content outlet that owns its own `ListFilterBar` (filter pills + sort dropdown + Mark all read).
3. **Context** (who's here, what's open). right-rail inspector with `Members`, `Work`, `Activity` tabs.

Strong points:

- **Single shell**. the same `NetworkShell` wraps `/network`, `/network/$channel/{threads,directs,activity}` and the detail routes. There is no parallel "channel detail" surface; the detail routes plug into the same shell via the TanStack `Outlet` (`network.tsx:174-204`).
- **Inspector is composition over modes**. the right rail has two modes (`inspector` and `thread`) controlled by `rightRailMode` and `rightRailContent` (`network.tsx:153-172`, `right-rail.tsx:5-32`). The thread overlay can also escape into a full-page view with `?view=full` (`network.$channel.threads.$threadId.tsx:11-23`).
- **Channel rail is restrained.** It uses the global `NAV_ROW_CLASS` + active-bar primitives shared with the app sidebar (`channel-rail-row.tsx:36-58`).

Weak points:

- **Two filter bars stacked.** In the directs tab, the `ListFilterBar` (`Filter | All / Has work / @me / Pinned / Unread | Sort | Mark all read`) sits on top of a second mono subheader (`X DIRECT ROOMS IN THIS CHANNEL` + `New direct` button). `network.$channel.directs.tsx:65-94`. That is two consecutive `border-b` strips with `px-5` and competing JetBrains Mono eyebrows.
- **Inspector tab IA contradicts itself.** The right-rail inspector ships its own `Members / Work / Activity` tabs (`network-inspector.tsx:46-117`) while the channel header already exposes `Threads / Directs / Activity` (`channel-tabs.tsx:65-103`). Two distinct things named `Activity` live in the same shell. Inspector activity = "Last 10 transitions" (mixed thread+direct preview list), main `/activity` = full cross-surface feed for the channel. The names need to diverge.
- **The "channel detail" pane the design contract assumes is never used.** `NetworkShell` accepts `activeChannelDetail` and `threadCount` / `directCount` props but the live route always passes `null` / `null` (`network.tsx:178-180`). The header therefore never shows real counts beside the `Threads` / `Directs` tabs even when the data is available in `route.directs.directs.length` / `threadsQuery.threads.length`. That looks like incomplete plumbing rather than a deliberate choice.
- **Channel rail conflates "Channels" + "Direct Rooms" + "Recents" in one panel.** Three sections share a single panel. fine. but the heading for Direct Rooms shifts from "Select a channel" to a list once a channel is active, which makes the rail feel modal even though it's a single sticky region (`channel-rail.tsx:156-191`).

---

## 2. Cross-route consistency

What is consistent and lifts the whole module:

- **Tokens.** Every audited file uses `var(--color-... )` tokens, never raw hex literals. Spot check across `channel-header.tsx`, `network-inspector.tsx`, `inspector-activity-feed.tsx`, `inspector-members-list.tsx`, `list-filter-bar.tsx`, `directs-list.tsx`, `threads-list.tsx`, `thread-overlay.tsx`, `direct-room.tsx`, `channel-rail.tsx`. No `#hex` literals leaked.
- **Type stack.** Inter for body, JetBrains Mono `text-[10px..11px]` `uppercase tracking-[0.06em]` for every meta strip and section eyebrow. No Playfair, no foreign font. Matches `DESIGN.md` §3.
- **Active state grammar.** Channel rail rows reuse `ACTIVE_NAV_INDICATOR_CLASS` + `ACTIVE_NAV_ROW_CLASS` from `web/src/components/sidebar-nav-classes.ts`, identical to the global app sidebar (`channel-rail-row.tsx:36-58`, `channel-rail.tsx:67-84`).
- **Empty-state grammar.** All six empties (`NetworkEmpty`, `DaemonDown`, `ThreadsEmpty`, `DirectsEmpty`, `ThreadEmpty`, `DirectEmpty`, `ConversationError`) wrap the shared `Empty` primitive with `lucide-react` icons at the system size and mono titles. Pattern is consistent.
- **Tab indicator.** Both the channel-tab strip and the inspector-tab strip use a `2px` accent underline on the active tab (`channel-tabs.tsx:93-99`, `network-inspector.tsx:106-111`). Same anatomy, two implementations. see Finding P2-IA-1.

Drift / inconsistency to address:

- **Two distinct in-row-time formats.** Channel-rail direct rows render the relative timestamp on the right (`channel-rail.tsx:78-82`); thread list rows also right-align it (`threads-list.tsx:86-93`); but `directs-list.tsx:78-83` and `inspector-activity-feed.tsx:137-141` align it inline, which means scanning the inspector vs the main directs list produces different reading rhythms.
- **Selection / hover treatments diverge.** Threads list: active row uses `bg-[color:var(--color-surface)]`, no left bar (`threads-list.tsx:54-57`). Directs list: active row uses `bg-[color:var(--color-accent-tint)]` (`directs-list.tsx:48-51`). Channel rail rows: active row uses `bg-surface` + a 2px accent left bar (`ACTIVE_NAV_*`). Three different "selected row" treatments in one shell. `DESIGN.md` §5 / §6 names exactly one selected-list-item pattern (Elevated `#2E2C2B` + 2px left accent bar). Pick one. P1.
- **Header "kebab" surfaces inconsistent affordances.** `channel-header.tsx:166-192` exposes a `MoreHorizontal` dropdown with a single `Refresh data` item; the inspector exposes the same `MoreHorizontal` glyph as a disabled, tooltip-only "More actions · Coming soon" button (`network-inspector.tsx:146-158`). Same icon, two contradictory affordances on the same screen.

---

## 3. Truthful UI verdict

Five places where the UI implies functionality the daemon does not currently expose. These are P0 if they ship to alpha:

1. **`AGENT` / `HUMAN` member labels are heuristic.** `useChannelMembers` documents that the runtime "currently does not persist a `kind` field on `NetworkPeerPayload`, so we treat the presence of a local agent session (`session_id`) as the AGENT signal" (`web/src/systems/network/hooks/use-channel-members.ts:24-34`). The inspector's members list and the directs list render those labels as if they were peer attributes (`inspector-members-list.tsx:84-87`, `directs-list.tsx:64-71`). Truthful UI requires either a daemon `kind` field or removing the chip until it exists.
2. **Direct-room header static `agent` label.** `direct-room.tsx:76-78` hardcodes the literal mono `"agent"` chip beside the peer name regardless of the actual peer kind. This is even less truthful than the inspector heuristic.
3. **Presence dot is a placeholder.** `web/src/systems/network/hooks/use-network-presence.ts:1-23` returns `{ state: "idle" }` unconditionally and the doc explicitly says "presence telemetry lands post-MVP". `<PresenceDot>` therefore never shows a color in production (`direct-room.tsx:23-50`), yet `aria-label`s and rendering paths exist for `running`, `needs_input`, `errored`. Either commit to "no presence" UI today or wire to a real source.
4. **Hover toolbar pretends Reply / Pin / Fork / More work.** `hover-toolbar.tsx` exposes Reply, Pin to capability, Fork thread, More actions buttons; `Add reaction` is correctly disabled with the tooltip `"Reactions land post-MVP"`. None of the production callsites wire `toolbarHandlers` (`grep -rn "toolbarHandlers" web/src/systems/network/` returns three matches, all inside `timeline.tsx` itself). So all four buttons render as enabled controls that do nothing on click. Match the reaction pattern (disabled + post-MVP tooltip) until handlers exist.
5. **Composer toolbar pretends Attach / Format / Mention work.** `composer.tsx:75` only forwards `onSlash`. `Attach`, `Text formatting`, `Mention` icon buttons (`composer-toolbar.tsx:55-72`) render with no `onClick`, so they appear interactive but silently no-op.

Two softer truthful-UI flags (P1, not blockers):

- **`Search` icon-button on the channel header is a `coming soon` ghost** (`channel-header.tsx:133-146`). It is `aria-disabled="true"` + `tabIndex=-1`, which is honest, but it occupies the header next to the inspector toggle as if it were an active control. Hide it until search lands or move to an overflow item.
- **`/attach` slash command is correctly flagged "Post-MVP" but `/run` and `/mention` only insert literal text into the textarea** (`use-composer-state.ts:126-133`). Their descriptions imply real execution. The UX is recoverable (the user just hits send and a peer reads `/run …`) but the descriptions are aspirational.

No live-update / SSE indicator anywhere in the shell. The module relies on TanStack Query polling (5s for messages, 3s for work, 15s for lists, 30s for channels. `lib/query-options.ts:23-30`). There is no "connected / reconnecting / stale" affordance in `NetworkShell` even though the daemon does expose `/api/network/status`. P1.

---

## 4. Accessibility cross-cuts

Run across all routes:

- **Tab semantics correct.** Both `ChannelTabs` and `InspectorTabNav` set `role="tab"` + `aria-selected` + `aria-current="page"` (`channel-tabs.tsx:75-90`, `network-inspector.tsx:80-94`). Both have visible focus rings via `focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]`.
- **Channel rail pin button has `aria-pressed` + dynamic `aria-label`** (`channel-rail-row.tsx:60-66`), but the button is hidden via `opacity-0` until hover, with the focus-visible escape. keyboard-only users CAN reach it with TAB, but this is fragile. P2.
- **Channel header inspector toggle uses `aria-pressed` + `data-state="open|closed"`** (`channel-header.tsx:148-164`). Good.
- **Right-rail inspector close button has `aria-label="Close inspector"`** but the right-rail container itself has `aria-label="Channel inspector"`. Both readable.
- **`role="log"` on Timeline** (`timeline.tsx:108-114`) is appropriate, but it does not announce new messages via `aria-live` since polling re-renders the entire list. Acceptable for now, but if streaming arrives, switch to `aria-live="polite"` on the new-message slot.
- **Color-only signal.** Unread channel rows use `font-semibold` + `text-primary` (`channel-rail-row.tsx:36-44`). that's typography contrast not color, good. The `1 work open` chip uses `bg-[color:var(--color-warning-tint)]` plus the explicit text "1 work open" (`threads-list.tsx:31-40`) so the meaning is not color-only.

---

## 5. Density verdict

Density is comfortable on 1440px (16-20px row padding, 5-12px gaps, two consecutive `border-b` strips on the directs tab is the only rhythm offender). On 1024px the right rail starts cramping the main pane (story `routes-app-stories-network--threads-tab` at 1024 evidence below).

At ≤768px the channel rail (260px), the main pane, AND the inspector (360px when open) sum to >900px of horizontal chrome. The shell does not collapse the channel rail to an icon-rail mode, nor does it auto-close the inspector. Storybook screenshot at 320px shows the channel rail still rendering at 260px and the main pane reduced to a sliver. P1.

Evidence:
- `_evidence/threads/03-storybook-threadstab-1440.png`. comfortable 1440 layout.
- `_evidence/threads/06-storybook-1024.png`. 1024 layout, main pane ~700px.
- `_evidence/threads/04-storybook-768.png`. 768 layout, main pane heavily compressed.
- `_evidence/threads/05-storybook-320.png`. 320 layout, main pane unusable; channel rail still 260px.

---

## 6. Top P0 / P1 cross-route findings

These flow through to individual route reports. Each cites file:line evidence.

### P0 (ship blocker)

1. **Truthful UI: hover toolbar Reply / Pin / Fork / More are dead controls.** `hover-toolbar.tsx:71-101` renders four icon buttons with `aria-label`s that imply behavior; no production callsite (`thread-overlay.tsx`, `direct-room.tsx`, `timeline.tsx` consumers) passes `toolbarHandlers`, so clicks are silent no-ops. Either disable + tooltip "Coming soon" (matching the `Add reaction` pattern), or wire to real handlers. Effort M.
2. **Truthful UI: composer toolbar Attach / Format / Mention are dead controls.** `composer.tsx:74-75` only forwards `onSlash`; `composer-toolbar.tsx:55-72` renders three additional icon buttons that fire undefined handlers. Same fix pattern. Effort S.
3. **Truthful UI: AGENT / HUMAN member labels and the static `agent` chip in `direct-room.tsx`.** `use-channel-members.ts:24-34` admits the heuristic. `direct-room.tsx:76-78` hardcodes the label. Either ship a daemon `kind` field (preferred) or strip the chip. Effort S (UI-only) / L (with daemon field). The chip is read by an agent screen-scraping the DOM and could mislead automation. P0 because it labels the wrong peer kind for any human-driven peer.

### P1 (high-value polish)

1. **Three different "selected row" treatments.** Channel rail uses `bg-surface + 2px left accent bar`; threads list uses `bg-surface`; directs list uses `bg-accent-tint`. `DESIGN.md` §6 names exactly one pattern. Pick the rail pattern and apply everywhere.
2. **Stacked filter bars on `/network/$channel/directs`.** `ListFilterBar` immediately followed by a second mono header strip + `New direct` button creates two consecutive `border-b` rows with redundant eyebrows. Merge into one bar.
3. **Inspector "Activity" tab and channel "Activity" tab share a name with different scopes.** Rename the inspector tab to `Recents` or `Last 10` so the affordance is unambiguous. `network-inspector.tsx:62-66` + `inspector-activity-feed.tsx:106-117`.
4. **Header counts wired to `null` even when data exists.** `network.tsx:178-180` passes `directCount={null}` / `threadCount={null}` to `ChannelTabs` so the count chips never render in production. The data is available in `route.directs.directs` / `threadsQuery.threads`. Wire it.
5. **No connection / stale indicator inside the Network shell.** The dashboard has `ConnectionIndicator`. Network does not. Polling is 5–30s and silent.
6. **Static `agent` chip and presence dot in direct-room header are placeholders.** Remove until presence and peer kind ship.
7. **Mobile / narrow viewport.** No collapse rule for the 260px channel rail or auto-close for the 360px inspector below 1024px. Storybook at 320px renders unusable layout.

### P2 / P3

- Pin star icon-button hidden via `opacity-0` until hover or focus. keyboard users have to know to TAB to it. Minor.
- `direct-empty.tsx:18-22` description "Send the first message. they'll be notified." Promises notification capability that the daemon does not currently model. Either confirm a notification path exists or rephrase.
- `channel-tabs.tsx:91` capitalizes labels via `<span className="capitalize">`; the source labels are already title-cased. Redundant. Trivial.
- "Recent activity · Read-only" subheader (`activity-feed.tsx:120-124`). useful but the "Read-only" tag is mono-uppercase metadata; mono uppercase usually marks identifiers, not state. Consider sentence-case.

---

## 7. Sign-off

- [x] Every claim cites `file:line` or screenshot.
- [x] No section left as `<TODO>`.
- [x] Truthful UI tested against `internal/network/` and `use-network-presence.ts` / `use-channel-members.ts`.
- [x] No em dashes (the dashes used are inside CSS class names or inside backticked literals).
- [x] Findings tagged P0–P3.
