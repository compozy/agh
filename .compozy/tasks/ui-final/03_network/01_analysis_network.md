# UI/UX Analysis :: `network` :: `/network`

> **Status:** draft
> **Owner subagent:** `ui-final/network`
> **Date:** 2026-05-06
> **Module:** `network` (`03_network`)
> **Route path:** `/network` (TanStack Router id: `/_app/network`)
> **Web source:** `web/src/routes/_app/network.tsx`
> **System owner:** `web/src/systems/network/components/shell/`
> **Storybook story id(s):** `routes-app-stories-network--threads-tab`, `routes-app-stories-network--directs-tab`, `routes-app-stories-network--activity-tab`, `routes-app-stories-network--empty-channels`, `routes-app-stories-network--disabled`, `routes-app-stories-network--loading`, `systems-network-networkshell--right-rail-open`, `systems-network-networkshell--default`, `systems-network-channelheader--threads-tab-active`, `systems-network-channelheader--inspector-open`
> **Live URLs probed:** `http://localhost:3000/network` · `http://localhost:6006/iframe.html?id=routes-app-stories-network--threads-tab&viewMode=story`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/network.tsx`
  - `web/src/systems/network/components/shell/network-shell.tsx`
  - `web/src/systems/network/components/shell/channel-rail.tsx`
  - `web/src/systems/network/components/shell/channel-header.tsx`
  - `web/src/systems/network/components/shell/channel-tabs.tsx`
  - `web/src/systems/network/components/shell/right-rail.tsx`
  - `web/src/systems/network/components/shell/network-inspector.tsx`
  - `web/src/systems/network/components/shell/inspector-activity-feed.tsx`
  - `web/src/systems/network/components/shell/inspector-members-list.tsx`
  - `web/src/systems/network/components/shell/list-filter-bar.tsx`
  - `web/src/systems/network/components/shell/channel-rail-row.tsx`
  - `web/src/systems/network/components/shell/channel-rail-recents.tsx`
  - `web/src/systems/network/components/work/work-inspector.tsx`
  - `web/src/systems/network/components/empty-states/{daemon-down,network-empty}.tsx`
  - `web/src/systems/network/hooks/use-network-route-shell.ts`
  - `web/src/systems/network/hooks/use-channel-members.ts`
  - `web/src/systems/network/hooks/use-network-presence.ts`
  - `web/src/systems/network/lib/query-options.ts`
- **Storybook stories opened:**
  - `routes-app-stories-network--threads-tab` → `http://localhost:6006/iframe.html?id=routes-app-stories-network--threads-tab&viewMode=story`
  - `routes-app-stories-network--empty-channels` → `http://localhost:6006/iframe.html?id=routes-app-stories-network--empty-channels&viewMode=story`
  - `routes-app-stories-network--disabled` → `http://localhost:6006/iframe.html?id=routes-app-stories-network--disabled&viewMode=story`
  - `routes-app-stories-network--loading` → `http://localhost:6006/iframe.html?id=routes-app-stories-network--loading&viewMode=story`
  - `systems-network-networkshell--right-rail-open` → `http://localhost:6006/iframe.html?id=systems-network-networkshell--right-rail-open&viewMode=story`
  - `systems-network-channelheader--inspector-open` → `http://localhost:6006/iframe.html?id=systems-network-channelheader--inspector-open&viewMode=story`
- **Live web probes (`localhost:3000`):**
  - `/network` empty (daemon empty). captured.
  - `/network` redirect-to-first-channel branch. not exercised because daemon has no channels.
  - `/network` daemon-down branch. not reproduced live; verified via code path `network.tsx:85-96`.
- **Screenshots captured (`.compozy/tasks/ui-final/03_network/_evidence/`):**
  - `network/01-root-empty-1440.png`. live route, daemon empty, "No channels yet" empty state.
  - `network/01-root-1440.png`. earlier capture during shell mounting.
  - `network/02-storybook-rightrail-1440.png`. storybook shell with the right rail open.
  - `network/03-storybook-empty-1440.png`. empty-channels storybook story.
  - `network/04-storybook-disabled-1440.png`. network disabled storybook story.
  - `network/05-storybook-loading-1440.png`. loading state storybook story.
  - `network/06-storybook-inspector-1440.png`. inspector-open storybook story.
  - `threads/03-storybook-threadstab-1440.png`. populated threads tab (default landing inside the shell).
  - `threads/04-storybook-768.png`. threads tab @ 768.
  - `threads/05-storybook-320.png`. threads tab @ 320.
  - `threads/06-storybook-1024.png`. threads tab @ 1024.
- **Console / network errors observed:** none from `agent-browser console` and `agent-browser errors` on the live `/network` empty state.
- **Keyboard / a11y probes performed:** Tab order verified by reading source (channel rail rows are `Link` elements, pin button is a sibling `button`, channel header buttons + dropdown are reachable). Focus-visible ring colors token-driven.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** the operator-facing entry into AGH Network. It mounts the three-column shell, lists every channel + direct room + cross-channel recents in the rail, and either renders the per-channel surface (threads / directs / activity) when a channel is active or one of three explicit fallback states: daemon down, network disabled in config, no channels yet.
- **Primary user goal on this route:** pick a channel and land inside it (threads tab is the default destination). When there are no channels, create one or accept an invite.
- **Entry vectors:** sidebar "Network" link, deep links (`/network/<channel>/... `), recents.
- **Exit vectors:** click a channel rail row → `/network/$channel/threads`; click a direct → `/network/$channel/directs/$directId`; click a recent → corresponding thread / direct deep link; "Open settings" link from the disabled empty state (`network-empty.tsx:18-30`); the channel-header inspector toggle reveals the right-rail inspector but stays on this route.
- **Critical states:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty (no channels) | yes | `network.tsx:111-152`, `_evidence/network/01-root-empty-1440.png` | strong: shell still mounts; rail shows "No channels yet" hint; main pane shows the `Empty` primitive with "Create one or accept an invite." |
| Loading / skeleton | yes | `network.tsx:67-83`, `_evidence/network/05-storybook-loading-1440.png` | weak: a single centered `Loader2` spinner replaces the entire shell; no skeleton matches the final layout. |
| Partial data | partial | rail handles loading per section (`channel-rail.tsx:115-129`, `:165-177`) | strong on rail, but the main pane has no "channels loaded but channel not found" branch. `useNetworkRouteShell` redirects to the first visible channel via `useEffect` (`use-network-route-shell.ts:47-56`); a deep link to a deleted channel is not surfaced. |
| Populated (typical) | yes | story `routes-app-stories-network--threads-tab` | strong, captured at 1440. |
| Populated (dense, 100+ rows) | unknown | not exercised; rail uses no virtualization (`channel-rail.tsx:130-153`) | weak: hundreds of channels would scroll a non-virtualized list. |
| Error (network / daemon down) | yes | `network.tsx:85-96` + `daemon-down.tsx:13-36` | strong: full-bleed `DaemonDown` empty with retry-when-handler. The handler is missing (no `onRetry` is wired, see Finding P2-NET-1). |
| Error (permission / 403) | no | n/a | none of the network endpoints check 403 in the route. |
| Error (not found / 404) | partial | invalid `$channel` collapses to "no channels yet" empty when the channels list is empty; with channels, `useNetworkRouteShell` auto-navigates to the first visible channel. there is no "channel not found" feedback. |
| Read-only / disabled | yes | `network.tsx:98-108` + `network-empty.tsx:13-38` | strong: explicit "The network is off." with optional Open Settings action. The action is currently never wired (no `onOpenSettings` prop is passed at the callsite. `network.tsx:99-108`), so the button never renders. |
| Live-update (stream / SSE) | no | `lib/query-options.ts:23-30` polling intervals only | missing: no SSE indicator, no "stale" / "reconnecting" affordance. |
| Mobile / narrow viewport | weak | `_evidence/threads/04-storybook-768.png`, `_evidence/threads/05-storybook-320.png` | weak: 260px rail + 360px right rail compete with main pane on ≤1024px; no collapse / drawer pattern below 1024px. |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   2   | `network.tsx:67-83`, `lib/query-options.ts:23-30`, no `ConnectionIndicator` inside shell | Polling is silent; no stale/connected affordance; loading collapses the shell to a spinner instead of skeleton. |
| 2  | Match between system and real world    |   2   | `inspector-members-list.tsx:84-87`, `direct-room.tsx:76-78`, `use-channel-members.ts:24-34` | UI labels members AGENT/HUMAN heuristically; direct-room hardcodes `agent`. |
| 3  | User control and freedom               |   3   | `channel-header.tsx:148-164` (inspector toggle), `network.tsx:174-204` (overlay vs inspector right-rail switch), `?view=full` thread mode (`network.$channel.threads.$threadId.tsx:11-23`) | Plenty of control. No undo for `Mark all read` (`list-filter-bar.tsx:138-149`). |
| 4  | Consistency and standards              |   2   | three different "selected row" treatments. `channel-rail-row.tsx:36-45` vs `threads-list.tsx:54-57` vs `directs-list.tsx:48-51`; two `Activity` tabs at different scopes. `channel-tabs.tsx:45-50` vs `network-inspector.tsx:62-66` | Same shell, multiple grammars for the same concept. |
| 5  | Error prevention                       |   3   | `network.tsx:85-96` daemon-down branch; `network.tsx:98-108` disabled branch; `daemon-down.tsx:13-36`, `network-empty.tsx:13-38` | Solid guards. The truthful-UI dead controls (P0) actively cause user errors. |
| 6  | Recognition rather than recall         |   3   | sidebar shared classes, mono eyebrows, hash icon on channel rows (`channel-rail-row.tsx:46-58`) | Consistent vocabulary; minor confusion from duplicate `Activity`. |
| 7  | Flexibility and efficiency of use      |   2   | composer Cmd/Ctrl+Enter (`use-composer-state.ts:112-124`), pin keyboard reachable (`channel-rail-row.tsx:60-66`); pin button is `opacity-0` until hover | Power-user shortcuts shallow; no hotkey for inspector toggle, no ⌘K-style channel jumper. |
| 8  | Aesthetic and minimalist design        |   3   | `_evidence/network/02-storybook-rightrail-1440.png` | Flat depth, warm tokens, mono eyebrows. The directs tab stacks two header strips that compete (P1). |
| 9  | Help users recognize / recover errors  |   2   | `daemon-down.tsx:13-36` shows generic "Make sure the AGH daemon is running" with no details and no actual onRetry wiring at the callsite | Errors are honest but not actionable: no diagnostic, no copy of the failing endpoint, no link to docs. |
| 10 | Help and documentation                 |   1   | none in module | No tooltip-help, no "what is a channel?" first-run primer, no link to runtime docs from any empty state. |
|    | **Total**                              | **23/40** | | **Band:** ◯ adequate (20–28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders (`border-l/r > 1px`) used decoratively | OK | The 2px accent left bar (`ACTIVE_NAV_INDICATOR_CLASS`) is the documented `DESIGN.md` selected-row pattern, used only on active rows. |
| Gradient text                                   | OK | none |
| Glassmorphism / blur as default                 | OK | none |
| Hero-metric template                            | OK | none |
| Identical card grids                            | OK | shell is column-based, no card grid |
| Modal as first thought                          | OK | `NewDirectDialog` and `NetworkCreateChannelDialog` are appropriate uses (creation confirms intent). |
| Em dashes in copy                               | violations | `direct-empty.tsx:21` `"Send the first message. they'll be notified."`; `threads-empty.tsx:30` `"Start the first one. agents and humans both join."` These are em dashes inside UI microcopy, against `COPY.md` and the no-em-dash rule for product surfaces. P1. |
| Generic AI palette                              | OK | warm tokens via `var(--color-... )` |
| Category-reflex theme                           | OK | not a "messaging app cliche". no green/blue chat bubbles |
| Restated headings / intros that repeat the title | OK | `PageHeader` "Network" + count is unique on page |
| Decorative shadows / heavy elevation            | OK | none |
| Hardcoded `#000` / `#fff`                       | OK | none in network/ source |

**Summary verdict:** No, a stranger would not say "AI made this immediately." The shell has product-specific vocabulary (channel, peer, work) and the warm operator palette. Two genuine issues: em dashes in microcopy and the dead-control toolbars. The deterministic detector confirms (`_evidence/impeccable.json` = `[]`).

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point** (the empty-channels route, daemon empty): in the channel rail there are three section headers (`CHANNELS`, `DIRECT ROOMS`, `RECENTS`) but only two affordances, both passive ("No channels yet.", "Select a channel to see direct rooms."). The main pane shows one heading + one description. Total interactive controls visible: 1 (the workspace switcher in the global app sidebar). Pass.
- **8-item cognitive load checklist:**
  1. >4 options visible at once? `pass`. the rail keeps section headers tight.
  2. Self-evident labels? `pass` for `Network`, `Channels`, `Direct Rooms`, `Recents`, `Threads`, `Directs`, `Activity`. Mild fail for `@me` filter pill (`list-filter-bar.tsx:65-69`). first-timers wouldn't know what `@me` filters.
  3. Primary action visually dominant? `fail` on the empty state. the only CTA in the empty branch is the rail's "No channels yet." text, and the main-pane `Empty` primitive offers no `action` prop. New operators have nowhere obvious to click.
  4. Progressive disclosure? `pass`. inspector hidden until toggled; thread overlay pushable to full page.
  5. Related elements grouped via proximity? `pass`. channel header > tabs > content is a clean column.
  6. Hierarchy contrast ≥1.25? `pass`. `text-[18px] font-semibold` channel title vs `text-[13px] text-secondary` meta vs `text-[10px]` mono labels.
  7. Body line length 65–75ch? n/a. this is a chat shell, not prose. Channel-header `<p>` `metaSegments` is one truncating line.
  8. Whitespace varied? `partial`. `px-5 py-3` channel header → `px-5 py-2` filter bar → `px-5 py-3` directs subheader is mostly the same horizontal step. The directs tab in particular feels uniformly padded.

  Failure count: 2 (label clarity + missing primary CTA on empty). Moderate cognitive load.

- **IA observations:**
  - The empty `/network` page should offer the `New channel` action inline. `NetworkCreateChannelDialog` exists (`network-create-channel-dialog.tsx`) but no entry point on the empty branch.
  - Channels list is alphabetic. not sorted by activity. which is fine but undocumented. Pinned channels rise to the top (`network.tsx:194-196`).
  - The redundant `Activity` naming (channel tab vs inspector tab) trips first-time users.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all values via `var(--color-*)`. Spot-checked across the 12+ shell files; no `#hex` literals in network/. Pass.
- **Type scale:** Inter 13/14/18 px for content, JetBrains Mono 10/11px uppercase tracking-0.06em for eyebrows. Channel title 18px semibold (`channel-header.tsx:109-115`). Pass.
- **Radii / spacing:** rail 260px (`channel-rail.tsx:108`), right rail 360px (`right-rail.tsx:23`). Buttons + nav rows match `DESIGN.md` §5. No one-off radii.
- **Elevation:** flat. The channel rail uses `bg-canvas-deep`, the main pane uses `bg-canvas`, the right rail uses `bg-canvas-deep`. Three-step depth via background only. Pass.
- **Signal palette discipline:** accent for active states + send button + focus rings; warning tint for `1 work open` chip and the work-banner; danger only on errored presence (which never renders today). The static `agent` chip in `direct-room.tsx:76-78` uses tertiary text. neutral, but the meaning is misleading (truthful-UI). Otherwise color is signal. Pass.
- **Grid / rhythm:** the directs tab stacks two header strips (`network.$channel.directs.tsx:65-94`) with `border-b` + `px-5` on each. That's monotone. see Finding P1-NET-2.
- **Density:** comfortable on 1440 (rail 260 + main flexible + rail 360). Cramped at 1024 and broken at 768 (no responsive collapse).

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** `New channel` action is invoked from `NetworkCreateChannelDialog` but the route never surfaces it on the empty `/network`. `New direct` lives in the directs subheader (`network.$channel.directs.tsx:81-93`).
- **Destructive actions:** none on `/network` itself.
- **Forms:** `NewDirectDialog` and `NetworkCreateChannelDialog` are modals. Inline validation present in `network-create-channel-dialog.tsx`.
- **Tables / lists:** no virtualization in channel rail (`channel-rail.tsx:130-153`). Keyboard nav works through DOM order: each row is a `Link`, the pin star is a sibling `button`. No arrow-key list nav (TAB only).
- **Selection model:** single only. No bulk operations on channels.
- **Modals / drawers:** the right rail is a non-modal aside (correct). Modals (`NewDirectDialog`, `NetworkCreateChannelDialog`) wrap shadcn primitives.
- **Live updates:** none. TanStack polling at 5–30s.
- **Optimistic vs pessimistic:** message sends are optimistic in the timeline (`thread-overlay.tsx:60-64` retry/discard hooks). Channel creation is pessimistic.
- **Hover / focus / active:** every interactive control has them via `hover:bg-... ` + `focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]`.
- **Loading patterns:** root loading is a `Loader2` spinner taking the whole shell (`network.tsx:67-83`). Per-section skeletons inside the rail are good. Mismatch: the shell collapses on the first paint, then expands once channels arrive. jarring.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** all interactive controls are reachable. The pin star button has `opacity-0` until hover/focus-visible, so keyboard users get an invisible-by-default control until they TAB to it. P2.
- **Focus rings:** `focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]`. Visible.
- **TAB order:** logical (rail → main pane → right rail).
- **ARIA roles / labels:** `role="tablist"` + `role="tab"` + `aria-selected` + `aria-current="page"` correctly applied on both tab strips. `aria-label` on landmarks (`aside aria-label="Network channels"`, `aside aria-label="Channel inspector"`).
- **Color contrast:** `--color-text-primary #E5E5E7` on `--color-canvas #141312` ≈ 13:1. Mono tertiary `#636366` on canvas ≈ 4.4:1 (just below 4.5 for body but it's used as eyebrow / metadata). Within `DESIGN.md` allowance.
- **Motion:** `motion-safe:animate-pulse` on the presence dot (`direct-room.tsx:46`), and `transition-opacity` on the work-banner (`work-banner.tsx:60-69`). Both respect `prefers-reduced-motion`.
- **Text scaling:** the rail width is fixed at 260px; at 200% zoom it survives but the right rail (360px fixed) overflows the main content area on narrow viewports. P2.
- **Forms:** modals (`NewDirectDialog`, `NetworkCreateChannelDialog`) use shadcn-`Field`s with proper labels.

---

## 8. Empty / Loading / Error States

- **Empty (first-run / no channels):** adequate. `network.tsx:111-152` mounts the shell with empty rails and a centered `Empty` ("No channels yet. Create one or accept an invite."). `Empty` has no action prop here, so there is no `+ New channel` CTA. Adding one would lift this from adequate to strong.
- **Loading:** weak. A single `Loader2` spinner replaces the entire shell (`network.tsx:67-83`). The shell skeleton. rail + main pane + tabs. is not pre-rendered, which forces a full layout shift.
- **Error (daemon down):** adequate. The `DaemonDown` `Empty` is styled correctly but `network.tsx:88-96` does not pass `onRetry`, so the action button never renders. Error message is generic.
- **Permission denied:** missing.
- **Stale / disconnected:** missing. no SSE, no banner, polling is silent.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** `channel` is correct (not `room` / `topic`), `peer` is correct, `direct` matches `direct rooms` / `direct messages` in glossary. `capability` is referenced once in `hover-toolbar.tsx:86` ("Pin to capability"). correct vocabulary, but the action does nothing today (P0).
- **Tone:** generally calm, dry, operator-first. Two slips:
  - `network-empty.tsx:33` "Enable the embedded network in your AGH config to start.". uses "embedded network" which is internal-architecture terminology not exposed elsewhere; suggest "Enable AGH Network in your config to start." to match `COPY.md` vocabulary.
  - `direct-empty.tsx:21` "Send the first message. they'll be notified.". em dash + a notification claim the daemon does not yet implement.
- **Em dashes:** flagged at `direct-empty.tsx:21` and `threads-empty.tsx:30`. P1.
- **Restated headings:** none.
- **Sentence vs Title case:** the `Activity` subheader uses "Recent activity · Read-only" (`activity-feed.tsx:120-124`). sentence case, fine. Section labels (`Channels`, `Direct Rooms`, `Recents`) are title-cased; matches `SidebarSectionLabel` primitive.
- **Truthful UI test:**
  - `inspector-members-list.tsx:84-87` `AGENT` / `HUMAN` chips are heuristic (`use-channel-members.ts:24-34`). P0.
  - `direct-room.tsx:76-78` static `"agent"` chip. P0.
  - `direct-empty.tsx:21` "they'll be notified". daemon does not currently model peer notifications.
  - `hover-toolbar.tsx` Reply / Pin / Fork / More buttons render as enabled but do nothing. P0.
  - `composer-toolbar.tsx` Attach / Format / Mention buttons render as enabled but do nothing. P0.
  - `channel-header.tsx:133-146` `Search` icon. honestly disabled (`aria-disabled="true"` + `tabIndex=-1`). Acceptable.

---

## 10. Performance & Responsiveness

- **Initial render:** the route fetches `useNetworkPage` (`use-network-page.ts`, status + channels + recents) before mounting the shell. While the status query is loading, the entire shell is replaced with a spinner. Forces a layout shift when channels arrive.
- **Re-render hot spots:** `useNetworkRouteShell` returns a memoized object (`use-network-route-shell.ts:58-103`); `NetworkShell` is a presentational component. no obvious render thrash.
- **List virtualization:** none on the channel rail. With the typical AGH operator running fewer than 50 channels this is acceptable; with a few hundred channels the rail would scroll a non-virtualized DOM.
- **Bundle red flags:** none observed; no charting / heavy imports inside `systems/network`.
- **Responsive behaviour:**
  - 1440. comfortable, evidence in `_evidence/network/02-storybook-rightrail-1440.png`.
  - 1024. main pane ~700px, evidence in `_evidence/threads/06-storybook-1024.png`. Tight but usable.
  - 768. main pane ~280px between two 260+360 rails, evidence in `_evidence/threads/04-storybook-768.png`. Cramped.
  - 320. main pane unusable; rail still 260px.
- **Mobile interactions:** pin star is `opacity-0` and `group-hover:opacity-100`. touch devices don't have hover, so the pin affordance is invisible on mobile.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-network--threads-tab` (default landing, populated)
  - `routes-app-stories-network--directs-tab`
  - `routes-app-stories-network--activity-tab`
  - `routes-app-stories-network--empty-channels`
  - `routes-app-stories-network--disabled`
  - `routes-app-stories-network--loading`
  - `systems-network-networkshell--default`
  - `systems-network-networkshell--directs-tab`
  - `systems-network-networkshell--empty-channels`
  - `systems-network-networkshell--right-rail-open`
  - `systems-network-channelheader--*` four stories (one per tab + inspector-open)
  - `systems-network-channelrail--default|loading|empty`
- **States covered:** populated, empty, loading, disabled, right-rail-open.
- **Gaps:**
  - No daemon-down story for the `/network` route (the `Disabled` story is for `enabled: false`, not for `/api/network/status` returning 5xx). Add `routes-app-stories-network--daemon-down`.
  - No mobile / narrow viewport story. Storybook does not cover the responsive collapse case.
  - No "channel not found" story (deep link to a deleted channel).
  - No "stale data" story for what a polling-based timeline looks like when the user goes offline.
- **Story drift:** the Storybook fixtures (`storyHeroNetworkChannel`, `storybookNetworkStatus`) match the route consumers' expectations of `NetworkStatusPayload` / `NetworkChannelsPayload`. No prop drift detected.

---

## 12. Findings. Prioritised

### P0. Ship Blockers

1. **[P0-NET-1] What:** Hover toolbar Reply / Pin to capability / Fork thread / More actions render as enabled icon buttons that silently do nothing.
   - **Why:** the toolbar appears every time an operator hovers a message in the timeline. They look identical to the working `Reply in thread` icon, so users will click them and conclude the app is broken. Truthful UI > plausible UI.
   - **Fix:** mirror the `Add reaction` pattern. pass `disabled` and a tooltip ("Coming soon") for any action that does not have a wired handler. Alternatively, wire real handlers (reply: focus thread composer; pin to capability: open the capability picker).
   - **Cmd:** `/impeccable harden web/src/systems/network/components/timeline/hover-toolbar.tsx`
   - **Effort:** M
   - **Evidence:** `web/src/systems/network/components/timeline/hover-toolbar.tsx:53-101`; `grep -rn "toolbarHandlers" web/src/systems/network/` returns three matches all inside `timeline.tsx` itself.

2. **[P0-NET-2] What:** Composer toolbar Attach / Text formatting / Mention render as enabled but only `onSlash` is wired.
   - **Why:** same truthful-UI failure. The composer is the most-clicked area in the network shell.
   - **Fix:** disable the un-wired buttons with a tooltip until handlers exist. The `Attach` slash command is already correctly disabled in `composer-slash-popover.tsx:31-35`; mirror that.
   - **Cmd:** `/impeccable harden web/src/systems/network/components/composer/composer-toolbar.tsx`
   - **Effort:** S
   - **Evidence:** `web/src/systems/network/components/composer/composer.tsx:74-75` (only `onSlash` forwarded); `composer-toolbar.tsx:55-72` (four buttons rendered).

3. **[P0-NET-3] What:** AGENT / HUMAN member chips and the static `agent` chip in the direct-room header label peers without a daemon-supplied peer kind.
   - **Why:** an agent screen-scraping the DOM (which is a documented goal. `internal/CLAUDE.md` "Agent-manageable by default") will read the wrong peer kind for any non-AGH peer. Humans inferring agent status get the wrong answer.
   - **Fix:** preferred: add a `kind` field to `NetworkPeerPayload` in `internal/api/contract` and stop inferring. Minimum: remove the chip until a real signal exists.
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/shell/inspector-members-list.tsx web/src/systems/network/components/directs/direct-room.tsx`
   - **Effort:** S (UI strip) / L (with daemon field)
   - **Evidence:** `web/src/systems/network/hooks/use-channel-members.ts:24-34`; `web/src/systems/network/components/shell/inspector-members-list.tsx:84-87`; `web/src/systems/network/components/directs/direct-room.tsx:76-78`.

### P1. High-Value Polish

1. **[P1-NET-1] What:** Three different "selected list row" treatments inside one shell.
   - **Why:** `DESIGN.md` §6 names exactly one selected pattern (Elevated bg + 2px left accent bar). Threads list uses `bg-surface` with no bar; directs list uses `bg-accent-tint`; channel rail uses the canonical pattern.
   - **Fix:** unify on the channel-rail pattern (`ACTIVE_NAV_*` shared classes).
   - **Cmd:** `/impeccable polish web/src/systems/network/components/threads/threads-list.tsx web/src/systems/network/components/directs/directs-list.tsx`
   - **Effort:** S
   - **Evidence:** `channel-rail-row.tsx:36-58` vs `threads-list.tsx:54-57` vs `directs-list.tsx:48-51`.

2. **[P1-NET-2] What:** First paint is a centered `Loader2` spinner, not a skeleton matching the shell.
   - **Why:** layout shift when channels arrive; bad perceived performance.
   - **Fix:** render the `NetworkShell` empty + a row of skeleton entries while `page.isStatusLoading` is true; only fall back to `DaemonDown` after the status query errors.
   - **Cmd:** `/impeccable harden web/src/routes/_app/network.tsx`
   - **Effort:** S
   - **Evidence:** `network.tsx:67-83`, `_evidence/network/05-storybook-loading-1440.png`.

3. **[P1-NET-3] What:** No connection / stale indicator inside the Network shell despite 5–30s polling.
   - **Why:** operators cannot tell whether the data they see is current; the dashboard has `ConnectionIndicator`, this shell does not.
   - **Fix:** surface a small "Updated 12s ago" / "Reconnecting... " chip in the channel header right-cluster. Reuse `ConnectionIndicator`.
   - **Cmd:** `/impeccable polish web/src/systems/network/components/shell/channel-header.tsx`
   - **Effort:** M
   - **Evidence:** `web/src/systems/network/lib/query-options.ts:23-30`, `network.tsx` (no `ConnectionIndicator` import).

4. **[P1-NET-4] What:** `network.tsx` passes `directCount={null}` and `threadCount={null}` to the channel-tab strip even though the data is available.
   - **Why:** the count chips next to `Threads` / `Directs` never render in production.
   - **Fix:** pipe `route.directs.directs.length` and `threadsQuery.threads.length` from `useNetworkRouteShell` into `NetworkShell`.
   - **Cmd:** `/impeccable harden web/src/routes/_app/network.tsx`
   - **Effort:** S
   - **Evidence:** `network.tsx:178-180`, `channel-tabs.tsx:65-103`.

5. **[P1-NET-5] What:** The `DaemonDown` empty has an `onRetry` slot but the route does not pass one.
   - **Why:** users see "Network is unreachable" with no action.
   - **Fix:** wire `onRetry` to invalidate `networkKeys.status` + `networkKeys.channels`. Same for `NetworkEmpty`'s `onOpenSettings`.
   - **Cmd:** `/impeccable polish web/src/routes/_app/network.tsx`
   - **Effort:** S
   - **Evidence:** `network.tsx:88-96` (no `onRetry` passed), `daemon-down.tsx:13-36`.

6. **[P1-NET-6] What:** Em dashes in production microcopy.
   - **Why:** `COPY.md` and `DESIGN.md` ban them.
   - **Fix:** rewrite. "Send the first message and they'll see it." / "Start the first one. Agents and humans both join."
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/empty-states/`
   - **Effort:** S
   - **Evidence:** `direct-empty.tsx:21`, `threads-empty.tsx:30`.

### P2. Worthwhile

1. **[P2-NET-1] What:** Inspector "Activity" tab and channel "Activity" tab share the same name.
   - **Fix:** rename inspector tab to `Recents`.
   - **Cmd:** `/impeccable clarify web/src/systems/network/components/shell/network-inspector.tsx`
   - **Effort:** S
   - **Evidence:** `network-inspector.tsx:62-66`, `channel-tabs.tsx:45-50`.

2. **[P2-NET-2] What:** Pin star button is `opacity-0` until hover; touch devices never see it.
   - **Fix:** at touch widths, render the pin button at `opacity: 0.4` always.
   - **Cmd:** `/impeccable adapt web/src/systems/network/components/shell/channel-rail-row.tsx`
   - **Effort:** S
   - **Evidence:** `channel-rail-row.tsx:62-83`.

3. **[P2-NET-3] What:** No mobile / narrow-viewport collapse for the 260px channel rail and 360px right rail.
   - **Fix:** at `<1024px`, auto-close the right rail and convert the channel rail to a sheet/drawer toggle.
   - **Cmd:** `/impeccable adapt web/src/systems/network/components/shell/network-shell.tsx`
   - **Effort:** L
   - **Evidence:** `_evidence/threads/04-storybook-768.png`, `_evidence/threads/05-storybook-320.png`.

4. **[P2-NET-4] What:** The "embedded network" wording in `network-empty.tsx:33` does not match `COPY.md` vocabulary.
   - **Fix:** "Enable AGH Network in your config to start."
   - **Effort:** S
   - **Evidence:** `network-empty.tsx:33`.

### P3. Parking Lot

1. **[P3-NET-1] What:** Channel rail does not virtualize; will hurt at hundreds of channels.
2. **[P3-NET-2] What:** "Recent activity · Read-only" mono uppercase eyebrow for the activity feed (`activity-feed.tsx:120-124`). the "Read-only" tag is mono-uppercase metadata; usually mono uppercase marks identifiers, not state. Consider a sentence-case caption.
3. **[P3-NET-3] What:** "Mark all read" has no undo (`list-filter-bar.tsx:138-149`). Probably fine for an alpha; revisit after telemetry.

---

## 13. Persona Red Flags

- **Operator (returning power user, keyboard-first):**
  - Reaches the channel rail, but no Cmd-K-style channel switcher; they have to TAB or arrow-down through every link.
  - Hover toolbar promises Reply but the icon does nothing; they will assume their keyboard binding is wrong.
  - Pin star is invisible until hover; keyboard users only see it after they tab to it.
  - No connection indicator; if the daemon hangs they don't know.

- **First-timer (onboarding, no mental model yet):**
  - Empty `/network` page reads "No channels yet." with no inline `+ New channel` CTA. The `NetworkCreateChannelDialog` exists but is not surfaced.
  - "Network is off" empty offers "Open settings" but the current callsite never passes `onOpenSettings`, so the button is absent (`network.tsx:99-108`).
  - Two `Activity` tabs with different scopes confuse the meaning of "Activity".

- **Agent (DOM-scraping consumer of the UI):**
  - Stable `data-testid` selectors throughout (`network-channel-rail`, `network-channel-link-<channel>`, `network-channel-meta-<i>`, `network-inspector-tab-<tab>`, `network-direct-list-row-<id>`, etc.). Strong.
  - `AGENT` / `HUMAN` text content in members list does not reflect a daemon field; an agent reading the DOM gets a heuristic. P0.
  - Static `"agent"` mono chip in direct-room header. P0.

---

## 14. Cross-Module Consistency Notes

- **PageHeader vs sectionheader.** `PageHeader` is used here as in other modules (`tasks`, `agents`); the count slot displays `0` when there are no channels. Consistent.
- **Sidebar nav classes.** Channel rail uses the same `NAV_ROW_CLASS` / `ACTIVE_NAV_*` as the global sidebar. Consistent.
- **Dropdown trigger.** The channel header overflow uses the same shadcn `DropdownMenu` as `tasks` route filters. Consistent.

Diverges:

- **Selected list row treatment** (see P1-NET-1) diverges from the unified pattern used by the global app sidebar.
- **Loading state shape**: the dashboard renders skeletons; this route renders a single spinner.

---

## 15. Open Questions

- Should `/network` empty offer an inline `+ New channel` CTA, or is "channels arrive from the network" the intended mental model?
- Is the right-rail inspector primarily a desktop affordance? If yes, document the breakpoint at which it auto-closes.
- The `ChannelHeader` `inspectorOpen` toggle persists nowhere. should the user's preference survive route changes (per-channel) or remain ephemeral?
- Should the inspector's "Activity" tab be renamed `Recents`, or removed entirely (since the main `/activity` tab already exists)?

---

## 16. Recommended Action Plan

Run in this order:

1. `/impeccable harden web/src/systems/network/components/timeline/hover-toolbar.tsx`. disable Reply / Pin / Fork / More until handlers exist; mirror the existing `Add reaction` pattern.
2. `/impeccable harden web/src/systems/network/components/composer/composer-toolbar.tsx`. disable Attach / Format / Mention until handlers exist.
3. `/impeccable clarify web/src/systems/network/components/shell/inspector-members-list.tsx web/src/systems/network/components/directs/direct-room.tsx`. strip the AGENT/HUMAN heuristic chips and the static `agent` chip until a daemon `kind` field exists.
4. `/impeccable polish web/src/systems/network/components/threads/threads-list.tsx web/src/systems/network/components/directs/directs-list.tsx`. unify selected-row treatment with the channel rail pattern.
5. `/impeccable harden web/src/routes/_app/network.tsx`. replace the loading spinner with a shell skeleton; wire `onRetry` and `onOpenSettings` to the `DaemonDown` / `NetworkEmpty` empty states; pipe `directCount` / `threadCount` through.
6. `/impeccable polish web/src/systems/network/components/shell/channel-header.tsx`. add a connection / stale indicator.
7. `/impeccable clarify web/src/systems/network/components/empty-states/`. rewrite em-dash microcopy + replace "embedded network" with "AGH Network".
8. `/impeccable clarify web/src/systems/network/components/shell/network-inspector.tsx`. rename the inspector tab `Activity` to `Recents`.
9. `/impeccable adapt web/src/systems/network/components/shell/network-shell.tsx`. collapse rail / inspector below 1024px.
10. `/impeccable polish web/src/systems/network/`. final sweep.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/network/`.
- [x] No section left as `<TODO>` or empty.
- [x] Nielsen scores total (23/40) is consistent with the band claimed (◯ adequate).
- [x] Findings tagged P0–P3 with effort and command.
- [x] No hallucinated routes, components, or props.
- [x] No em dashes in this report.
