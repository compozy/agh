# AGH Network Threads & Direct Rooms — Web UI Design

**Status:** Approved 2026-05-04
**Scope:** Web/UI design specification for the `_techspec.md` deliverable in this task directory.
**Authority:** This document is the normative UI source. Where this document and `_techspec.md` overlap, `_techspec.md` wins for protocol/data semantics; this document wins for layout, interaction, visual treatment, and information architecture.
**Audience:** Implementing engineers for the web split (`task_13` shell + IA, `task_14` timeline + thread overlay, `task_15` composer + work + states), and reviewers of `task_07`/`task_08`/`task_16`/`task_17` who must keep their FE/docs/E2E references coherent with this spec.

---

## 1. Purpose & Non-Goals

### 1.1 Purpose

Replace the current flat `/network` UI with a route-driven, channel-pivoted experience that exposes the techspec's two conversation containers — public threads and two-party direct rooms — without contradicting the protocol, the data model, or the chromatic restraint defined in `DESIGN.md`.

This document fills every UX gap left open by `_techspec.md` and `task_13.md`. It does not redefine the protocol, the schema, the RPC surface, or the work lifecycle — those are owned by the techspec and ADRs.

### 1.2 Non-goals

This document does **not** prescribe:

- Private (encrypted) group threads. Out of MVP scope (`_techspec.md:25`).
- Group direct rooms with more than two peers (`_techspec.md:25, 27`).
- Unread-count sync, notification preferences, retention controls, transcript export (`_techspec.md:25`).
- Compatibility shims for `interaction_id` or `kind:"direct"` (`_techspec.md:27`).
- Voice/huddle-style live audio (deliberately rejected; AGH analogue is live transcript tail).
- Mobile-native (iOS/Android) layouts. Responsive web only, with a `<1024px` collapse fallback documented in §3.3.

---

## 2. Decisions Ledger

Three load-bearing decisions taken during the brainstorm. The rest of the document elaborates them.

| # | Decision | Chosen | Why |
|---|----------|--------|-----|
| D1 | Thread navigation model | **Hybrid: route canonical + right-rail overlay** | URL deep-linkable (techspec mandate at `_techspec.md:1113-1119`) while preserving channel-as-ambient context — the load-bearing Slack pattern for human-supervises-N-agents. |
| D2 | Channel rail structure | **Channel-pivot + main-pane tabs (Threads / Directs / Activity), with cross-channel "Recents" pinned to top of rail (max 5)** | Channel is the real unit of scope per techspec (direct rooms are channel-scoped, `_techspec.md:587`). Tabs map to the two distinct route files (`network.$channel.threads.tsx` and `network.$channel.directs.tsx` per `_techspec.md:1113-1119`). |
| D3 | Work lifecycle surfacing | **Combo: inline chip on messages with non-default work state + auto-hiding pinned banner when `open_work_count > 0` + collapsible Work Inspector tab** | Inline gives forensic context, banner gives escalation visibility, inspector gives triage scope. Auto-hide respects flat-depth aesthetic (no false alarms). |

A fourth, equally load-bearing decision sits beneath the first three:

| # | Decision | Chosen | Why |
|---|----------|--------|-----|
| D4 | Chromatic discipline | **Mono-neutral default; one chromatic emphasis per row at rest; tint over solid; conditional chip rendering** | Avoid the "carnaval" that emerges when every chip + avatar tint + work state + kind label fires on every row. Color is a scarce resource. |

The seven discipline rules of D4 are spelled out in §6.

---

## 3. Layout Architecture

### 3.1 Shell — Three Rails + Optional Right Rail

The network surface composes within the existing global `app-sidebar` (no changes to L1). Inside `/_app/network`, the layout is:

```
┌──────┬────────────────┬────────────────────────────────────────────┬──────────────┐
│  L1  │  L2            │  L3                                        │  L4 (opt)    │
│  64- │  Channel rail  │  Main pane                                 │  Right rail  │
│  72px│  240-260px     │  fluid                                     │  420px       │
│      │                │                                            │  (overlay)   │
│ Globl│  Recents (5)   │  Channel header + tabs                     │  Thread or   │
│  nav │  ───────────   │  ┌───────────────────────────────────┐     │  Members     │
│      │  Channels      │  │                                   │     │  or Inspect  │
│ Avatr│   #ops         │  │   Timeline / List / Detail        │     │  ─ slide-in  │
│ Pres-│   #design      │  │                                   │     │  ─ closable  │
│  ence│   #incidents   │  └───────────────────────────────────┘     │  ─ Esc/X     │
│      │                │  Composer (sticky bottom)                  │              │
└──────┴────────────────┴────────────────────────────────────────────┴──────────────┘
```

- **L1 (existing)**: global app sidebar (`web/src/components/app-sidebar.tsx`). Network is one entry among Tasks, Bridges, Jobs, etc. No changes required by this design.
- **L2 (this design)**: channel rail. Lives inside `/_app/network` route. Fixed width, collapsible to icon-only on hover at viewports `<1280px`.
- **L3 (this design)**: main pane. Hosts channel header, tab strip, timeline or list, composer.
- **L4 (this design, optional)**: right rail. Slides in for thread overlay (D1) or contextual inspectors (members, work). Default closed.

### 3.2 Right-Rail Thread Overlay (Hybrid Model)

When the user navigates to `/_app/network/$channel/threads/$threadId`:

1. URL becomes canonical and deep-linkable (techspec mandate).
2. Channel timeline (the Threads tab list) **remains rendered** in L3 at reduced contrast (`opacity: 0.55`, no pointer events on rows except the row matching `$threadId`, which gets accent-tint background).
3. Right rail (L4) opens with `transform: translateX(0)` from `+100%`, animating over `var(--duration-slow)` (200ms) with `var(--ease-out)`. Width: 420px. Border-left: 1px `--color-divider`.
4. Right rail contents:
   - **Header**: thread title (truncated), peer count chip (e.g. `3 agents · 1 human`), close button (`X`), "Open in main" button (escapes hybrid → full-page route).
   - **Root message** at top with subtle `[root]` badge in JetBrains Mono 11px uppercase tracked 0.06em.
   - **`X replies` divider** (mirrors Slack): 1px line + centered "12 replies" label in `--color-text-tertiary`.
   - **Reply timeline**: same message-row component as channel timeline (§5.2), but with avatar gutter at 32px instead of 36px (denser).
   - **Detail composer** at bottom: replies into `$threadId`. Includes `[ ] Also broadcast to room` checkbox (Slack's "Also send to channel" — mapped to AGH semantics).
5. Closing the rail (Esc, click X, or click outside the rail on `>=1280px` viewports): URL pops back to `/_app/network/$channel/threads`. Channel timeline returns to full opacity. Scroll position is preserved.

**Behavior on direct rooms**: identical, but the right rail is reserved for the **Members** or **Work Inspector** panels — direct rooms never overlay another direct room (two-party rooms have no thread depth).

### 3.3 Responsive Collapse

| Viewport | Behavior |
|----------|----------|
| `>=1280px` | All four rails visible. Hybrid right-rail overlay active. |
| `1024-1279px` | L1 collapses to icon-only (no labels). L2 retains labels. Hybrid overlay still active. |
| `<1024px` | **Hybrid model disabled.** Thread routes navigate as full-page (mode A from brainstorm). Channel timeline is hidden when thread detail is shown. Back button or breadcrumb returns to `/threads`. L2 collapses to a Sheet (`@agh/ui` `Sheet` primitive) opened via hamburger in the channel header. |
| `<640px` | Single-column. Channel header simplified to back button + name. Tabs stack as a horizontal scroll strip. |

The breakpoint constants live as Tailwind defaults — no custom values introduced.

---

## 4. Information Architecture

### 4.1 Channel-Pivot + Main-Pane Tabs

Channel is the unit of scope. The user picks a channel in L2; the main pane shows three tabs:

```
┌─────────────────────────────────────────────────────────────────────┐
│  #ops                                          3 agents · 1 human   │
│  ─────────────────────────────────────────────────────────────────  │
│  [ Threads · 12 ] [ Directs · 4 ] [ Activity ]                      │
└─────────────────────────────────────────────────────────────────────┘
```

- **Threads** tab (default) → renders `network.$channel.threads.tsx` content (`_techspec.md:1115`).
- **Directs** tab → renders `network.$channel.directs.tsx` content (`_techspec.md:1117`).
- **Activity** tab → unified reverse-chronological feed of last activity across both surfaces in this channel. Read-only. No composer. **Note: this is a design addition, not techspec-mandated.** It is built entirely from `last_activity_at` on summary rows (`_techspec.md:231-256`); no new endpoints required.

Tab changes are real route navigations (TanStack `Link`), not internal state — the URL always reflects which surface is active. This satisfies `_techspec.md:1124` ("Query keys must include `channel`, `surface`, and container ID") because each tab's queries are isolated by route file.

The thread/direct **count chips** on each tab use `--color-text-tertiary` text (no chromatic emphasis) until the count changes — then the chip pulses once with `--color-accent-tint` background for 1.2s, then fades back to neutral. This is the only motion on the tab strip.

### 4.2 Cross-Channel Recents

Top of L2, above the channels list, render a fixed `Recents` section with up to 5 entries. Each entry is a one-line row:

```
  RECENTS
  ─────────────
  [TH] #ops · "Codex /goal launch wave"          2m
  [DM] @claude-opus in #design                   12m
  [TH] #incidents · "DB lag on pg-primary"       1h
  [DM] @hermes in #ops                           4h
  [TH] #design · "Brand refresh round 3"         9h
```

- Prefix tag: `[TH]` (thread) or `[DM]` (direct), JetBrains Mono 10px, `--color-text-tertiary`. **Not** a colored chip — keeps Recents mono.
- Title: `LastMessagePreview` truncated to ~36 chars, `--color-text-primary` if has new activity since last view (bold), else `--color-text-secondary`.
- Channel reference: `#channel` for threads, `in #channel` for directs.
- Timestamp: relative, right-aligned, `--color-text-tertiary`.
- Click navigates directly to the corresponding route — no intermediate state.

Recents is a **convenience layer**, not the canonical IA. The data source is a client-side merge of the `/api/network/channels/{channel}/threads` and `/directs` summary lists, sorted by `last_activity_at` across all channels the operator has visible.

### 4.3 Channel Rail Row Anatomy

Below Recents, the channels list:

```
  CHANNELS                                          [+]
  ─────────────
  ★ #ops                                          3
    #design                                       1
    #incidents                                    8
    #brand
    #sandbox                                      —
```

- **Section label**: `CHANNELS`, JetBrains Mono 11px uppercase, `--color-text-label`, tracking 0.06em.
- **Star icon** for pinned channels (Phosphor `Star` weight `fill`, `--color-accent`). Star toggles via right-click menu.
- **Channel name**: `#name`, Inter 14px. Bold (weight 600) if any thread or direct in this channel has activity newer than last visit; regular (weight 400) otherwise.
- **Trailing badge**: combined count of `open_work_count` across all containers in this channel. Renders only if `> 0`. Style: `--color-text-tertiary` text on transparent bg (no pill at rest); pulses to `--color-warning-tint` if any work is in `needs_input` state.
- **No bell, no headphones, no mute toggle** in MVP — those are post-MVP per `_techspec.md:25`.

Selected channel row: `--color-accent-tint` background, no border change. Hover: `rgba(255, 247, 237, 0.04)` overlay.

---

## 5. Component Anatomy

### 5.1 Channel Header + Tabs

Single 56px-tall row sticky at top of L3:

```
  #ops                            3 agents · 1 human · 2 active work    [⋯]
  ──────────────────────────────────────────────────────────────────────────
  [ Threads · 12 ]  [ Directs · 4 ]  [ Activity ]                  [ Search ]
```

- **Channel name**: Inter 18px, weight 600, `--color-text-primary`.
- **Identity chip cluster** (right of name): role-mix label (`3 agents · 1 human`, computed from channel membership), then dot separator, then `open_work_count` summary if `> 0` (`2 active work`, `--color-text-secondary` at rest, `--color-warning` text if any in `needs_input`).
- **Kebab `⋯`**: opens a menu with channel-level actions (rename, archive — the latter is post-MVP; render disabled with tooltip "Post-MVP").
- **Tab strip** (second row): 36px tall, hairline divider above. Each tab is a `Link` to its route. Active tab: 2px accent underline, `--color-text-primary`. Inactive: `--color-text-secondary`, no underline. Count chips behave per §4.1.
- **Search button** at right of tab strip: opens an in-channel `CommandDialog` (already in `@agh/ui`) scoped to the current channel + active tab.

No bell. No headphones. No call. No huddle.

### 5.2 Message Row

The single most reused component. Three states: full row, collapsed continuation, system event.

#### 5.2.1 Full row (first message of an author group, or after a >60s gap)

```
  ┌──────┬───────────────────────────────────────────────────────────┐
  │      │  claude-opus  agent  2:41 PM      [working · 12s]         │
  │ [O]  │                                                            │
  │      │  Working on the migration plan. Found 3 candidates so     │
  │      │  far — running heuristics over the diff to rank them.     │
  │      │                                                            │
  │      │  [↳ reply]  [emoji]  [pin]  [fork]  [⋯]                    │  ← hover
  └──────┴───────────────────────────────────────────────────────────┘
```

- **Avatar gutter**: 36px square, `border-radius: 4px` (never circle — flat-depth). Color seeded deterministically from peer ID (existing `network-workspace-shell.tsx` palette). 8px column gap to content.
- **Display name**: Inter 14px, weight 600, `--color-text-primary`. Click → opens peer card popover (out of MVP — render as no-op with cursor: default).
- **Role chip**: `agent` / `human` / `system`, only on the first row of the author group. JetBrains Mono 10px uppercase, tracked 0.06em, `--color-text-tertiary`. No bg, no border. Mono-only. **Default chip never paints color.**
- **Timestamp**: Inter 12px, weight 400, `--color-text-tertiary`. Hover reveals exact ISO timestamp in tooltip.
- **Work chip** (conditional, see §5.8): inline pill aligned right of timestamp.
- **Body**: Inter 15px, weight 400, line-height 1.5, `--color-text-primary`. Markdown-rendered via existing renderer.
- **Reactions row** (post-MVP — placeholder slot only).
- **Hover toolbar**: appears on row hover. Inline pill with 1px border in `--color-divider`, no shadow. Buttons left-to-right: `Reply in thread` (right-rail overlay D1), `Add reaction` (post-MVP — disabled, tooltip "Post-MVP"), `Pin to capability` (replaces Slack "save"), `Fork thread` (replaces Slack "share"), kebab `⋯`.

#### 5.2.2 Collapsed continuation (same author within 60s)

```
  ┌──────┬───────────────────────────────────────────────────────────┐
  │      │  Found one more. Promoting to top of rank.                │
  │      │                                                            │
  └──────┴───────────────────────────────────────────────────────────┘
```

- Avatar gutter is **empty but reserved** (36px width preserved for alignment).
- Hovering the empty gutter reveals the timestamp inline (`12:41:08 PM`, JetBrains Mono 11px, `--color-text-tertiary`). This satisfies the "agent transcripts are forensic" requirement.
- No name. No role chip. No work chip (work chips are at message granularity, not author-group granularity — but if this specific message has a non-default work state, the chip floats top-right of the body).

**Collapse window: 60 seconds.** Tighter than Slack's 5min because agents emit fast and 5min would visually merge unrelated tool-call bursts.

#### 5.2.3 System event row (low-prominence)

For `kind:"trace"`, `kind:"receipt"`, `kind:"capability"`, `kind:"whois"`, `kind:"greet"`:

```
  ┌──────┬───────────────────────────────────────────────────────────┐
  │   ─  │  CAPABILITY  claude-opus invoked refactor-pass             2:42 PM
  └──────┴───────────────────────────────────────────────────────────┘
```

- No avatar — gutter shows a 1px hairline.
- Single-line, JetBrains Mono 12px, `--color-text-secondary`.
- Kind label first (uppercase tracked 0.06em), then verbatim event description, then timestamp.
- Click → expands inline to show full body.

`kind:"say"` is the default and never paints a kind chip (chromatic discipline rule 4).

### 5.3 Author Group Collapsing

- Grouping window: 60 seconds.
- Group breaks on: different `peer_from`, gap `> 60s`, kind change (a `say` followed by a `capability` always breaks).
- Visual rhythm: 4px between rows in the same group, 16px between groups, 24px around date pills.

### 5.4 Date Pills + "New" Divider

- **Date pill**: floating centered, 24px tall, JetBrains Mono 11px uppercase tracked 0.06em, `--color-text-tertiary`. Hairlines extend left/right of the label (`flex-1 h-px bg-[--color-divider]`). Format: `TODAY`, `YESTERDAY`, `TUESDAY · APR 29`, `2026 · MAR 15` (year prefix only when crossing year boundary).
- **"New" divider**: 1px line in `--color-accent` (not subtle gray — this is the one place we paint the divider with chroma) with centered `NEW` label in Inter 11px weight 600, `--color-accent`. Rendered at the position of the operator's last-read message (client-tracked via local storage; reset on visit). Survives until next visit.

### 5.5 Thread Overlay — Replies & "Also Broadcast" Affordance

Thread overlay structure (cf. §3.2):

```
  ┌─ Thread ───────────────────────────────── [X] ─┐
  │  Codex /goal launch wave        3 agents · 1h  │
  │  [Open in main →]                              │
  ├────────────────────────────────────────────────┤
  │  [root]                                        │
  │  pedronauck  human  2:31 PM                    │
  │  Let's investigate why /goal blew up …         │
  │                                                │
  │  ─────────────  12 replies  ─────────────      │
  │                                                │
  │  claude-opus  agent  2:32 PM                   │
  │  Pulling the run logs now …                    │
  │                                                │
  │  hermes  agent  2:33 PM  [working · 4s]        │
  │  Cross-checking with the prior incident …      │
  │                                                │
  │  …                                              │
  ├────────────────────────────────────────────────┤
  │  ┌──────────────────────────────────────────┐  │
  │  │ Reply…                                   │  │
  │  └──────────────────────────────────────────┘  │
  │  [+] [Aa] [@] [/]              [✓] Broadcast  │
  │                                       [Send]  │
  └────────────────────────────────────────────────┘
```

- `[root]` badge: JetBrains Mono 10px uppercase tracked 0.06em, `--color-text-tertiary`. No background.
- "Open in main →" button: ghost style, takes `$threadId` to a dedicated full-page route (same URL, but a UI mode flag promotes it from overlay to main). Useful for long threads.
- "Broadcast" checkbox: maps to the techspec `kind:"say"` send with both `surface:"thread"` (the reply) and an additional summary message to the channel. The exact RPC pattern is owned by the techspec; the UI commits to the affordance and the wording.

### 5.6 Direct Rooms — Headerless Layout

Direct rooms render with no `#name`, no member count, no topic — just the other party's identity:

```
  ┌─────────────────────────────────────────────────────────────────┐
  │  @claude-opus  agent · ●working                                  │
  │  ───────────────────────────────────────────────────────────────  │
  │                                                                  │
  │  …timeline…                                                      │
  │                                                                  │
  │  ───────────────────────────────────────────────────────────────  │
  │  [Compose…]                                                      │
  └─────────────────────────────────────────────────────────────────┘
```

- Identity row: 48px tall. Peer name (Inter 16px weight 600) + role chip + presence/run-state dot.
- **Presence dot semantics** (D4-compliant):
  - Default state (`idle`, `away`): no dot rendered. Mono-only.
  - `running`: small `--color-accent` dot, 6px, with 2s pulse animation (opacity 0.6 → 1.0).
  - `needs_input`: `--color-warning` dot, 6px, no pulse (steady).
  - `errored`: `--color-danger` dot, 6px, no pulse.
- The `peer_a < peer_b` lex ordering of direct room storage is invisible in UI — render the **other** party as primary identity, the local session's peer is implicit.

### 5.7 Composer

Two composer variants. Both share styling.

#### 5.7.1 Channel-level "New public thread" composer (Threads tab default)

Pinned to bottom of the Threads tab list (not the timeline — the list is **of threads**, not messages). Visual:

```
  ┌─────────────────────────────────────────────────────────────────┐
  │  Start a new thread…                                             │
  │                                                                  │
  │  [+] [Aa] [@] [/]                                       [Send]  │
  └─────────────────────────────────────────────────────────────────┘
```

- Multi-line auto-grow textarea, 2 lines minimum, max 8 lines visible (scroll thereafter).
- Submit generates a fresh `thread_id`, posts `kind:"say"` with `surface:"thread"`, then redirects to `/_app/network/$channel/threads/$threadId` (per `_techspec.md:1126`).
- **Collision UX**: if the server rejects the generated `thread_id` (collision), the client retries once silently with a new ID. If the second attempt also fails, surface the server validation error in a toast (`@agh/ui` `Toaster` / Sonner) with copy: "Couldn't open this thread. Try again." (No technical detail — the operator doesn't need to see UUID collision math.) Per `_techspec.md:1127`.
- Submit button label: `Send` at rest. Hover reveals target hint: `Send to #ops` (or `Send to @peer` for direct rooms). No cost estimate in MVP (no LLM cost source available at the network layer).

#### 5.7.2 Detail composer (thread or direct room)

Pinned to bottom of the timeline pane. Identical visual structure to 5.7.1 but:
- Placeholder copy: `Reply…` (thread) or `Message @peer…` (direct).
- Submits to the active container (`thread_id` or `direct_id` from URL).
- No collision retry path — the container already exists.

#### 5.7.3 Slash commands

Typing `/` opens a `CommandDialog`-style popover anchored above the textarea:

- `/run <capability>` — opens capability picker. Resolves to a `kind:"capability"` send.
- `/mention <peer>` — peer picker. Inserts `@peer` token.
- `/attach` — attach context menu (file, URL, capability ref, prior message). Out-of-MVP entries render disabled with `Post-MVP` tooltip.

The popover is keyboard-only navigable (↑↓ select, Enter confirm, Esc close). No mouse hover commits.

#### 5.7.4 Toolbar buttons

Below the textarea, left-aligned, four icon-only buttons (24px hit target, 16px icon, Phosphor `Regular` weight, `--color-text-secondary` at rest):

- `+` attach context (popover)
- `Aa` text formatting (popover with B / I / S / link / list / quote / code)
- `@` mention
- `/` slash command (same popover as 5.7.3)

Send button right-aligned. Primary style (`--color-accent` bg, `--color-text-primary` on accent). Disabled when textarea is empty. **Never** shows a loading spinner in the button itself — feedback comes from the optimistic message appearing in the timeline (§9).

### 5.8 Work Lifecycle Surfacing

Three layers, all auto-hiding when not relevant.

#### 5.8.1 Inline chip on message row

Conditions for rendering:
- Message has `work_id` set.
- Work state is **not** `submitted` and **not** `completed` (these states are silent — chromatic discipline rule 6).

Visual variants:

| State | Chip text | Background | Text color | Animation |
|-------|-----------|------------|------------|-----------|
| `working` | `working · 12s` (live ticking) | `--color-warning-tint` | `--color-warning` | None |
| `needs_input` | `needs input` | `--color-warning-tint` | `--color-warning` | 2s gentle opacity pulse (0.85 → 1.0) |
| `failed` | `failed` | `--color-danger-tint` | `--color-danger` | None |
| `canceled` | `canceled` | none | `--color-text-tertiary` | None — completely silent |

Position: right of timestamp on full message row (§5.2.1) or top-right floating on collapsed continuation (§5.2.2).

Chip click: opens Work Inspector (§5.8.3) scoped to that `work_id`.

#### 5.8.2 Pinned banner (channel header → timeline)

Renders **only** when `open_work_count > 0` for the active container. Position: between channel header and timeline, full-width, 36px tall.

Visual variants:

```
  Default (working only):
  ┌─────────────────────────────────────────────────────────────────┐
  │  [warning-tint bg] 2 active work in flight              [view]  │
  └─────────────────────────────────────────────────────────────────┘

  Escalation (any in needs_input):
  ┌─────────────────────────────────────────────────────────────────┐
  │  [warning solid bg] 1 needs input · 1 working          [view]  │
  └─────────────────────────────────────────────────────────────────┘
```

- Default: `--color-warning-tint` bg, `--color-warning` text.
- Escalation: `--color-warning` solid bg, `--color-canvas` text. **This is the only place in the entire UI where solid warning chroma paints a large surface area.** That's intentional — it's the alarm.
- "view" link: opens Work Inspector right-rail.
- When `open_work_count` returns to 0, banner animates out: `opacity 1 → 0` over 200ms, then `height 36px → 0` over 200ms. Total 400ms. Gone.

#### 5.8.3 Work Inspector (right-rail tab)

Right rail (L4) has tabs at the top when no thread is open:

```
  [ Members ] [ Work · 2 ] [ Activity ]
```

- `Members`: lists peers in container.
- `Work · N`: lists open work items, each row showing: state badge, target peer, `opened_at` (relative), duration, "jump to message" link. Only renders if `N > 0`.
- `Activity`: micro-feed of recent state transitions. Read-only.

When a thread overlay is open, the right rail is occupied by the thread — the Members/Work/Activity tabs are not shown until the thread closes.

---

## 6. Chromatic Discipline (Load-Bearing)

The seven rules. Violating any one of these in implementation is a code review block.

### 6.1 Default-mono rule

> **Default state of any row, cell, button, or chip is mono-neutral.** Text in `--color-text-primary` or `--color-text-secondary`. Zero chromatic chip in resting state.

Examples:
- A message row with `kind:"say"` from a known peer with `work_id` in `submitted` state: zero chips. Just avatar (tinted by identity, see 6.2), name, timestamp, body.
- A channel rail row with no activity: just `#name` in `--color-text-secondary`. No badge, no dot.

### 6.2 One-emphasis rule

> **At most one chromatic emphasis per row at rest.** Avatar tint counts as the emphasis (it's identity).

This means:
- Avatar is always tinted (identity is permanent emphasis). So no other chip on the row paints color **at rest**.
- Chips that need to paint color (work, mention badge, "New" divider) do so by **replacing** another emphasis or by being conditional (work chip suppressed in default states per 6.6).
- Hover/selection bumps emphasis up by one — still capped at two simultaneous chromatic surfaces.

### 6.3 Tint-over-solid rule

> **Chips and small surfaces use `--color-*-tint` (15% opacity) backgrounds with tinted text.** Solid (full chroma) is reserved for the pinned banner in `needs_input` escalation state (§5.8.2) and the Send button (§5.7.4).

Solid usage inventory across the entire UI:
1. Send button (`--color-accent`).
2. Work banner in escalation state (`--color-warning`).
3. "New" divider line color (`--color-accent`) — but a 1px line, not a fill.

Three places. Total. Nothing else paints solid.

### 6.4 Kind-chip suppression rule

> **`kind:"say"` never renders a kind chip.** It is the default. Chip is rendered only for `greet`, `whois`, `capability`, `receipt`, `trace`. `kind:"direct"` is impossible (rejected at every ingress per `_techspec.md:1332`) — implementations must not even prepare a render path for it.

Removes ~80% of would-be chip noise.

### 6.5 Role-chip group rule

> **Role chip (`agent`/`human`/`system`) renders only on the first message of an author group.** Collapsed continuations (§5.2.2) do not repeat the chip.

### 6.6 Work-chip silence rule

> **Work chip is silent in `submitted` and `completed` states.** Painted only for `working`, `needs_input`, `failed`. `canceled` paints in `--color-text-tertiary` (no fill, no tint — visible as text but not as color).

State `submitted` is silent because every work item starts there; rendering it would mean every `work_id` paints. State `completed` is silent because the lifecycle has resolved; no operator action needed.

### 6.7 Selection & hover restraint rule

> **Selection: `--color-accent-tint` background, no border change, no shadow.**
> **Hover: `rgba(255, 247, 237, 0.04)` warm-white overlay, no border change, no shadow.**
> **Active (`:active`, click): `transform: scale(0.99)` for 80ms, no color change.**

The flat-depth aesthetic from `DESIGN.md` — no shadows anywhere, ever, in the network surface.

---

## 7. State Semantics

### 7.1 Loading

Skeleton shapes that mirror the final layout. No spinners except for two cases:
- `kind:"trace"` rendering an in-flight thinking step (existing `assistant-ui` pattern).
- Initial app shell while routes resolve (existing global pattern).

Specific skeletons:

- **Channel rail** while `/api/network/channels` loads: 5 skeleton rows in the channels list, 3 skeleton rows in Recents. Bone color: `--color-surface-panel`.
- **Threads tab list** while `GET /threads` loads: 6 skeleton thread rows, each ~60px tall (avatar + 2 lines preview).
- **Direct list** same pattern.
- **Timeline** while `GET /messages` loads: 4-6 skeleton message rows of varying body lengths.
- **Right-rail thread overlay** while `GET /threads/{id}` loads: skeleton header + skeleton root message + "loading X replies" placeholder.

Animation: shimmer using `@agh/ui` `Skeleton` primitive (already implements the shimmer pattern).

### 7.2 Empty states

Terse, mono, capability-suggesting. No illustrations. No mascots. Use `@agh/ui` `Empty` component.

| Surface | Title | Description | Action |
|---------|-------|-------------|--------|
| No threads in channel | `No threads yet.` | `Start the first one — agents and humans both join.` | `[Start a thread]` (focuses channel-level composer) |
| No directs in channel | `No direct rooms yet.` | `Open one to talk privately with a peer in this channel.` | `[New direct]` (opens peer picker) |
| Empty thread (no replies) | `Thread has no replies.` | `Reply below to keep the context alive.` | none |
| Empty direct room | `Quiet so far.` | `Send the first message — they'll be notified.` | none |
| No work in container | (banner not rendered — auto-hide per §5.8.2) | — | — |
| Network disabled | `The network is off.` | `Enable the embedded network in your AGH config to start.` | `[Open settings]` |
| Operator has no channels visible | `No channels yet.` | `Create one or accept an invite.` | `[New channel]` |

Title typography: Inter 15px weight 500. Description: Inter 13px, `--color-text-tertiary`, max-width `42ch`. Action button: ghost variant.

### 7.3 Error states

Inline, terse, actionable. Use `@agh/ui` `Alert` with `tone="danger"` for severe failures, ghost row for transient.

- **Failed to load list** (channels/threads/directs/messages): inline error in the empty slot — `Couldn't load. [Retry]`. Retry triggers a `queryClient.invalidateQueries` for that key.
- **Send failed** (composer): the optimistic message stays in place but renders with `--color-danger-tint` background and a tiny `[Retry] [Discard]` cluster appended to the body. No toast for individual send failures (too noisy).
- **Thread collision after second retry** (per `_techspec.md:1127`): Toaster (Sonner) message: `Couldn't open this thread. Try again.` Single sentence. Auto-dismiss 4s.
- **Network unreachable** (daemon down): full-page error in the network route only — `Network is unreachable.` + `Make sure the AGH daemon is running.` + `[Retry connection]`.

### 7.4 Disabled states

When network is off in config:
- L2 channel rail renders the empty state from §7.2 (network disabled).
- L3 main pane renders the same.
- L1 sidebar Network entry renders with `opacity: 0.6` and a `disabled` tooltip.

When operator lacks ACL on a channel:
- Channel does not appear in L2.
- Direct deep-links return 403 → render: `You don't have access to this channel.` (no error tone — informational).

---

## 8. Motion

### 8.1 Tokens

All motion uses `DESIGN.md` tokens — no custom durations or easings:

- `--duration-fast` (100ms): hover state changes, chip pulse start.
- `--duration-base` (150ms): tab change, focus ring fade-in.
- `--duration-slow` (200ms): right-rail slide-in/out, banner reveal/dismiss.
- `--ease-out`: most reveals.
- `--ease-in-out`: rare — used only for the chip pulse and presence dot pulse (must feel symmetric).

### 8.2 Allowed properties

Per `DESIGN.md` and `design-taste-frontend` skill:

- `transform` (translate, scale only — no skew, no rotate at the macro level).
- `opacity`.
- Nothing else. No `top`, `left`, `width`, `height` animations. No `box-shadow` animations.

### 8.3 Specific motions

| Motion | Trigger | Duration | Easing |
|--------|---------|----------|--------|
| Right-rail thread overlay open | Navigate to `$threadId` | 200ms | `--ease-out` |
| Right-rail thread overlay close | Esc / X / outside click | 200ms | `--ease-out` |
| Pinned banner reveal | `open_work_count: 0 → >0` | 200ms (opacity) + 200ms (height) | `--ease-out` |
| Pinned banner dismiss | `open_work_count: >0 → 0` | 200ms (opacity) → 200ms (height) | `--ease-out` |
| Tab count chip pulse | Count change | 1.2s once | `--ease-in-out` |
| Presence dot pulse (running) | Render | 2s loop | `--ease-in-out` |
| Work chip needs_input pulse | Render | 2s loop | `--ease-in-out` |
| Optimistic message arrival | Message sent | 150ms fade-in + 8px translateY | `--ease-out` |
| Date pill fade-in on scroll | Enter viewport | 400ms | `cubic-bezier(0.16, 1, 0.3, 1)` (existing pattern from minimalist-ui skill) |

No magnetic buttons. No hover image trails. No glitch effects. No parallax. No scroll-hijack. The network surface is utilitarian.

---

## 9. Realtime Strategy

### 9.1 MVP — Polling

The techspec does not prescribe SSE or any inbound stream for MVP (`_techspec.md:58-59`, `_techspec.md:1462-1464`). The web layer relies on:

- **TanStack Query refetch on focus**: `refetchOnWindowFocus: true` for all network queries. Operators returning to the tab see fresh data within ~200ms.
- **Interval polling per route**:
  - Channels list: every 30s.
  - Threads/Directs lists for active channel: every 15s.
  - Messages for active thread/direct: every 5s while route is mounted; every 30s when right-rail closed but channel timeline is mounted.
  - Work entry detail: every 3s when Work Inspector is open and showing a non-terminal state.
- **Manual refresh**: subtle refresh affordance in the channel header kebab menu. Not a primary affordance — operators should rarely need it.

### 9.2 Optimistic mutations

All sends optimistically append the message to the active timeline:

- Composer submit → `useMutation` adds the message to the query cache for the active container with status `pending`.
- On success, the optimistic message is replaced with the server's canonical `NetworkConversationMessage`.
- On failure, the message renders with `--color-danger-tint` and retry/discard affordances (§7.3).

The optimistic add MUST include the same `MessageID` the client generated, so the canonical replacement can match by ID.

### 9.3 Post-MVP — SSE path

Reserved. Following the existing pattern from `web/src/systems/bridges/hooks/use-bridge-health-stream.ts`:

- Each container would get a `useNetworkStream({ channel, surface, containerId })` hook.
- Server-side SSE endpoint posts envelope events post-commit (per `_techspec.md:1464` "network hook events provide extension-visible observation after commit").
- Hook updates query cache via `queryClient.setQueryData` rather than refetching — preserves cursors and scroll.

This is **not** a task_13 deliverable. It is a deliberate MVP limitation, not a known shortcut. When SSE arrives, the polling intervals collapse to backstops.

---

## 10. Resolved Ambiguities

The gaps `task_13.md` left unspecified, resolved by this document.

| # | Gap (`task_13.md`) | Resolution (this doc) |
|---|---------------------|------------------------|
| A1 | Component folder structure | `web/src/systems/network/components/{shell,timeline,composer,work,thread-overlay,empty-states}/`. Each subfolder owns its components. See §11. |
| A2 | Breadcrumb / navigation UX | No breadcrumb. Navigation is: L1 sidebar → channel rail → channel header tabs → optional right-rail overlay. Back-button preserved by route history. |
| A3 | Composer UI placement | §5.7. Channel-level pinned to bottom of Threads tab list; detail-level pinned to bottom of timeline; thread overlay has its own composer pinned to bottom of overlay. |
| A4 | Thread creation collision UX feedback | §5.7.1. Silent retry once; on second failure, single Sonner toast `Couldn't open this thread. Try again.` |
| A5 | Direct room resolution UX flow | Tab "Directs" → `[New direct]` action → `Combobox` peer picker → `POST /directs/resolve` → on success navigate to `$directId` route. Loading state is a 200ms inline spinner inside the action button; error is inline below picker. |
| A6 | Work surfacing | §5.8. Inline chip + auto-hide pinned banner + Work Inspector tab. |
| A7 | Kind chips/filters in main pane | No filter chips at rest. Search → `CommandDialog` (§5.1) supports a `kind:` filter via slash modifier. `kind:"direct"` impossible per `_techspec.md:1332`. |
| A8 | Settings controls scope | Network settings page (out of this design's scope) renders ONLY: enable/disable embedded network, channel ACL viewer (read-only in MVP), persisted-message preview length (single number input), timezone selector. No retention, no notification prefs, no unread tracking — all post-MVP per `_techspec.md:25`. **task_07 should reference §7.4 of this doc.** |
| A9 | State management (URL vs Zustand) | URL-first. TanStack Router params are canonical for `channel`, `surface`, `threadId`, `directId`. Zustand exists only for ephemeral UI state: composer draft buffers (per route), Work Inspector tab open/closed, channel rail collapsed state. Never for server state. |
| A10 | "Work" definition for users | Work is "an active piece of agent work bound to this thread or direct room." UI label in copy: `active work` (always lowercased in body, capitalized at start of sentence). Chip label uses state name verbatim (`working`, `needs input`, etc.). |
| A11 | Empty state copy | §7.2 table. |
| A12 | Last-read tracking | Client-side localStorage, key `network:lastRead:{channel}:{surface}:{containerId}`. Reset on visit (when timeline scrolls last message into view). The "New" divider rendered above the first message newer than the stored timestamp. |
| A13 | Direct room peer label order | Always render the **other** party as primary identity. The local session's peer is implicit. The `peer_a < peer_b` storage ordering is never visible. |

---

## 11. Component → File Map

Reuse existing primitives wherever possible. New components live in the network system folder.

### 11.1 Reused from `@agh/ui`

| Use | Primitive |
|-----|-----------|
| Main shell sidebar slot | `Sidebar`, `SidebarSectionLabel` |
| Right-rail overlay | `Sheet` on `<1024px`; on `>=1024px` a custom anchored `div` (Sheet has modal semantics we don't want) |
| Tab strip | `Tabs` (radix-based) |
| Composer textarea | `Textarea` |
| Composer toolbar buttons | `Button` (`variant="ghost"`, `size="icon"`) |
| Slash command popover | `CommandDialog` |
| Mention/peer picker | `Combobox` |
| Send button | `Button` (`variant="default"`, primary) |
| Work chip / role chip / kind chip | `Pill`, `PillGroup` |
| Work banner | inline `div` styled with tokens — no new primitive needed |
| Work Inspector tabs | `Tabs` |
| Empty state | `Empty` |
| Error state | `Alert` (`tone="danger"`) for severe; inline ghost for transient |
| Toast | `Toaster` (Sonner) |
| Date pill | inline — no primitive |
| "New" divider | inline — no primitive |
| Skeleton | `Skeleton` |
| Avatar gutter | `Avatar`, `AvatarFallback`, palette-seed util from existing `network-workspace-shell.tsx` |
| Channel rail row | `Item`, `ItemGroup` (existing list primitive) |
| Search dialog | `CommandDialog` |
| Disabled state | `Empty` with action button |

### 11.2 New components in `web/src/systems/network/components/`

```
shell/
  network-shell.tsx                     // L2+L3+L4 composition
  channel-rail.tsx                       // L2 (channels list + recents)
  channel-rail-recents.tsx               // recents section
  channel-rail-row.tsx                   // single channel row
  channel-header.tsx                     // L3 header + tabs
  channel-tabs.tsx                       // tab strip
  right-rail.tsx                         // L4 container (handles thread/inspector mode)
timeline/
  timeline.tsx                           // ordered list + grouping engine
  message-row.tsx                        // full row (5.2.1)
  message-row-collapsed.tsx              // continuation (5.2.2)
  message-row-system.tsx                 // system event (5.2.3)
  date-pill.tsx
  new-divider.tsx
  hover-toolbar.tsx
composer/
  composer.tsx                           // shared base
  channel-thread-composer.tsx            // 5.7.1 (creates new thread)
  detail-composer.tsx                    // 5.7.2 (replies/sends to active container)
  composer-toolbar.tsx
  composer-slash-popover.tsx
work/
  work-banner.tsx                        // pinned banner (5.8.2)
  work-chip.tsx                          // inline chip (5.8.1)
  work-inspector.tsx                     // right-rail tab content (5.8.3)
  work-inspector-row.tsx
thread-overlay/
  thread-overlay.tsx                     // right-rail thread mode
  thread-overlay-header.tsx
  thread-overlay-root.tsx                // root message presentation
  thread-overlay-replies.tsx
empty-states/
  network-empty.tsx                      // network disabled / no channels
  threads-empty.tsx
  directs-empty.tsx
  thread-empty.tsx
  direct-empty.tsx
```

### 11.3 Hooks & data layer in `web/src/systems/network/`

```
lib/
  query-keys.ts                          // [network, channel, surface, containerId, ...]
  query-options.ts                       // queryOptions per route
  palette.ts                             // identity-seeded avatar tints (existing logic — extract from current shell)
hooks/
  use-network-page.ts                    // route-level orchestration (replaces the existing flat one)
  use-channels.ts
  use-threads.ts                         // list + detail
  use-directs.ts                         // list + detail
  use-messages.ts                        // shared by thread + direct
  use-work.ts
  use-network-actions.ts                 // mutations: send, create thread, resolve direct
  use-last-read.ts                       // localStorage tracking + "New" divider math
  use-recents.ts                         // cross-channel merge
  use-network-presence.ts                // (post-MVP — placeholder; returns static idle for now)
```

### 11.4 Routes

Per `_techspec.md:1113-1119`. No new routes beyond what the techspec mandates.

```
web/src/routes/_app/network.tsx                                  // shell + redirect to first channel
web/src/routes/_app/network.$channel.threads.tsx                 // Threads tab
web/src/routes/_app/network.$channel.threads.$threadId.tsx       // Thread detail (route exists; rendered as overlay or full-page based on viewport)
web/src/routes/_app/network.$channel.directs.tsx                 // Directs tab
web/src/routes/_app/network.$channel.directs.$directId.tsx       // Direct detail
```

The `Activity` tab does **not** require a new route — it composes from the same data feeding the other tabs. It can live as a search-param state on `network.$channel.threads.tsx` (e.g. `?view=activity`) or as its own route file `network.$channel.activity.tsx`. **Decision**: own route file, for clean query-key isolation per `_techspec.md:1124`.

```
web/src/routes/_app/network.$channel.activity.tsx                // Activity tab (design-added, query-isolated)
```

---

## 12. Out of Scope (Mirrored from Techspec)

Explicit alignment with `_techspec.md:25`:

- Private (encrypted) group threads.
- Group direct rooms (>2 peers).
- Thread split, merge, cross-channel move.
- Unread count sync.
- Notification preferences, mute, bell.
- Search ranking.
- Retention policy controls.
- Transcript export.
- Federation policy.
- Analytics dashboards beyond the metrics list above.
- Compatibility aliases for `interaction_id` or `kind:"direct"`.

Surfaces explicitly excluded by this design as well:

- Reactions UI (placeholder slot only — wire post-MVP).
- Peer card popover content (avatar click is no-op in MVP).
- Cost estimates on Send (no LLM cost source at network layer).
- Magnetic / micro-physics buttons.
- Custom mouse cursors.
- Workspace-switching at L1 (AGH switches networks, not workspaces — and that's L1's existing concern, not this design's).

---

## 13. Acceptance Signals

How a reviewer knows this design was implemented faithfully.

### 13.1 Visual

- [ ] No drop shadows anywhere in the network surface. Zero `box-shadow` declarations.
- [ ] Avatar gutter consistent at 36px (full row) / 32px (thread overlay) / 0px-collapsed (continuation).
- [ ] Date pill renders TODAY/YESTERDAY/weekday transitions correctly across midnight.
- [ ] "New" divider paints `--color-accent` on the line and label — the only chromatic divider in the UI.
- [ ] Send button is the only solid `--color-accent` surface in the entire network UI (banner escalation is `--color-warning`).
- [ ] Work banner auto-hides within 400ms of `open_work_count` returning to 0.
- [ ] No chip renders for `kind:"say"`.
- [ ] No chip renders for `work_state ∈ {submitted, completed}` — even when `work_id` is present.
- [ ] Role chip suppressed on collapsed continuation rows.
- [ ] No emoji in any UI string. Phosphor or Radix icons only.

### 13.2 Interaction

- [ ] Navigating to `$threadId` opens the right-rail overlay; channel timeline remains visible at reduced contrast on `>=1024px`.
- [ ] On `<1024px`, navigating to `$threadId` switches to full-page layout; back button returns to threads list.
- [ ] Closing the overlay (Esc/X/click-outside) restores channel timeline to full opacity and preserves scroll.
- [ ] Tab clicks navigate routes (not internal state). The URL always reflects the active surface.
- [ ] Channel-level composer always creates a new thread + redirects on submit.
- [ ] Detail composer always sends to the active container.
- [ ] Thread ID collision triggers exactly one silent retry; second failure surfaces a single toast.
- [ ] Optimistic message arrives in the timeline within one frame of submit.
- [ ] Failed send keeps the optimistic message visible with retry/discard affordances inline.
- [ ] Tab count chip pulses once on count change, then stays neutral.

### 13.3 Data integrity

- [ ] Query keys always include `[network, channel, surface, containerId, ...]` per `_techspec.md:1124`.
- [ ] Threads tab queries never appear in directs tab cache and vice versa.
- [ ] No request body, response body, log line, prompt artifact, or rendered string contains `interaction_id` or raw `claim_token` (`_techspec.md:502, 1333, 1343`).
- [ ] No request can be constructed that submits `kind:"direct"` (`_techspec.md:1332`).
- [ ] `direct_id` rendered anywhere matches `^direct_[a-f0-9]{32}$` (`_techspec.md:587`).
- [ ] Browser artifact capture exposes `network_selected_thread` and `network_selected_direct` keys; `network_selected_peer` does not appear (`_techspec.md:1130`).
- [ ] All listed counters from §5.1 / §5.8 derive from `NetworkThreadSummary` / `NetworkDirectRoomSummary` — no client-side aggregation of full message lists.

### 13.4 Accessibility

- [ ] Every interactive element has a focus ring using `--color-accent` outline-offset.
- [ ] Tab order: L1 → L2 channel list → L3 channel header → L3 tabs → L3 content → composer.
- [ ] Right-rail overlay traps focus when open; Esc returns focus to the row that opened it.
- [ ] All icons have `aria-label`s.
- [ ] Color is never the sole signal for state — work chips include text, presence dots include `aria-label`, the "New" divider includes the literal word "NEW".
- [ ] Live region (`role="status"`) announces optimistic message arrival and work-state transitions.

---

## 14. Open Questions for Implementer

These are not blockers — implementer decides during build with reviewer concurrence.

1. **Avatar palette extraction.** The existing `network-workspace-shell.tsx` has an inline 8-tint palette. Move it to `lib/palette.ts` and add a deterministic seed test. Question: should the palette be exported from `@agh/ui` for reuse by other surfaces (e.g., agents page)? **Recommendation**: yes; create `packages/ui/src/lib/identity-palette.ts`.

2. **Animation respect for `prefers-reduced-motion`.** All §8.3 motions should disable when the OS sets reduced motion. Implementation: a `useReducedMotion` hook gate in each animated component. Question: should this be a global wrapper or per-component? **Recommendation**: global wrapper at the network shell level (`<MotionConfig reduce={prefersReducedMotion}>`).

3. **Right-rail width on ultrawide.** At `>=1920px`, the 420px right rail leaves a lot of horizontal room. Should it grow proportionally? **Recommendation**: cap at 480px; let main pane absorb the rest.

4. **Recents persistence.** Recents (§4.2) is computed client-side. Should the order survive page reload? **Recommendation**: no — it's a function of `last_activity_at`; recompute every mount.

5. **Search scope on Activity tab.** The in-channel search (§5.1) currently scopes to active tab. Activity is cross-surface — does its search scope cross-surface too? **Recommendation**: yes, for ergonomic parity with other agent tools.

6. **Optimistic message ID generation.** `MessageID` must be UUID-format. Use `crypto.randomUUID()` (browser native). Confirm with security review that this is acceptable for client-generated IDs. **Recommendation**: yes — server validates regardless; client UUID is just a placeholder for cache replacement.

7. **Storybook fixtures.** `task_13.md:113` mentions Storybook fixture rewrites. Per §11, every new component should ship a `*.stories.tsx`. Question: which states to cover per component? **Recommendation**: at minimum loading, empty, error, default, hover, selected, focused — a 7-state matrix per interactive component.

---

## Cross-references

- `_techspec.md` — protocol, schema, RPC, work lifecycle (canonical source).
- `_tasks.md` — task index. After this design is reviewed, `task_13` should be split or rewritten to cite this document (see "Task Reshape" appendix in the conversation that produced this doc).
- `DESIGN.md` (repo root) — design tokens, font stack, signal palette.
- `COPY.md` (repo root) — product language; UI copy strings used in this doc are first drafts and must be reviewed for COPY.md compliance before ship.
- `web/CLAUDE.md` — web-specific code conventions.
- `.agents/skills/agh-design/` — AGH design skill.
- `.agents/skills/design-taste-frontend/` — anti-bias rules used as authority for §6 / §8.
- `.agents/skills/minimalist-ui/` — typography, density, motion subtlety used as authority for §5 / §6.

---

**End of `_design.md`.**
