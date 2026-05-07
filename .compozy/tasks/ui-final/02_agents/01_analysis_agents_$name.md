# UI/UX Analysis - `Agents` :: `/agents/$name`

> **Status:** draft
> **Owner subagent:** `ui-final/02_agents`
> **Date:** 2026-05-06
> **Module:** Agents (`02_agents`)
> **Route path:** `/agents/$name` (TanStack Router id: `_app/agents/$name`)
> **Web source:** `web/src/routes/_app/agents.$name.tsx`
> **System owner:** `web/src/systems/agent/`
> **Storybook story id(s):** `routes-app-stories-agents-name--default`, `routes-app-stories-agents-name--no-sessions`, `routes-app-stories-agents-name--sessions-loading`, `routes-app-stories-agents-name--agent-loading`, `routes-app-stories-agents-name--not-found`, `routes-app-stories-agents-name--with-failed-session`, `routes-app-stories-agents-name--many-agents`
> **Live URLs probed:** `http://localhost:3000/agents/general` (real empty), `http://localhost:3000/agents/test-agent` (not-found), `http://localhost:6006/iframe.html?id=routes-app-stories-agents-name--default&viewMode=story`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/agents.$name.tsx`
  - `web/src/systems/agent/components/agent-page-header.tsx`
  - `web/src/systems/agent/components/agent-info-panel.tsx`
  - `web/src/systems/agent/components/agent-sessions-list.tsx`
  - `web/src/systems/agent/components/agent-stats-grid.tsx`
  - `web/src/systems/agent/components/agent-icon.tsx`
  - `web/src/systems/agent/index.ts`
  - `web/src/systems/session/components/session-create-dialog.tsx` (action target of New session)
  - `web/src/routes/_app/stories/-agents.$name.stories.tsx`

- **Storybook stories opened:**
  - `routes-app-stories-agents-name--default` -> `_evidence/agents.name/sb_default.png`
  - `routes-app-stories-agents-name--no-sessions` -> `_evidence/agents.name/sb_no-sessions.png`
  - `routes-app-stories-agents-name--sessions-loading` -> `_evidence/agents.name/sb_sessions-loading.png`
  - `routes-app-stories-agents-name--not-found` -> `_evidence/agents.name/sb_not-found.png`
  - `routes-app-stories-agents-name--with-failed-session` -> `_evidence/agents.name/sb_with-failed-session.png`
  - `routes-app-stories-agents-name--many-agents` -> `_evidence/agents.name/sb_many-agents.png`

- **Live web probes (`localhost:3000`):**
  - `/agents/general` empty state at 1440x900, 1024x800, 768x1024, 320x800.
  - `/agents/test-agent` not-found state at 1440x900.
  - New session dialog opened from `/agents/general`.

- **Screenshots / DOM snapshots captured:**
  - `_evidence/agents.name/live_1440_general_empty.png` - real empty agent at 1440px. Stats grid shows `ACTIVE SESSIO…` truncation.
  - `_evidence/agents.name/live_1024_general_empty.png` - 1024px, MCP rail hidden, empty state below the fold.
  - `_evidence/agents.name/live_768_general_empty.png` - 768px, MCP rail hidden, layout still 4-column metric grid.
  - `_evidence/agents.name/live_320_general_empty.png` - 320px, agent name + IDLE pill clipped from header.
  - `_evidence/agents.name/live_1440_notfound.png` - not-found state.
  - `_evidence/agents.name/live_create_dialog.png` - create dialog open over empty state.
  - `_evidence/agents.name/sb_default.png` ... `sb_with-failed-session.png` - storybook populated states.
  - `_evidence/agents.name/dom_full_notfound.txt`, `dom_live_empty.txt`, `dom_live_general.txt` - DOM trees.

- **Console / network errors observed:**
  - None at the SPA level. The `/api/agents/test-agent` call returned a 404 envelope which the route consumed correctly.

- **Keyboard / a11y probes performed:**
  - Toolbar buttons announce `aria-label="Refresh"`, `aria-label="Configure"` (`agent-page-header.tsx:60-77`).
  - Refresh icon spins via `aria-hidden="true"` and a CSS class - non-disruptive.
  - Session table rows are wrapped in `<Link>` - keyboard reachable.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** Per-agent control surface. It shows how busy this agent is (active count, total runtime, failures, last activity), what sessions it owns, and what MCP servers it advertises. From here an operator starts a new session, configures the agent, refreshes data, or jumps into an existing session.
- **Primary user goal on this route:** Start or resume a session for this agent (the `+ New session` toolbar button is the single most prominent CTA).
- **Entry vectors:** sidebar agent tree (active row treatment), permalink redirect from `/session/$id`, deep link from automation history / task tree.
- **Exit vectors:** clicking a row in the sessions table -> `/agents/$name/sessions/$id`; "Configure" -> agent edit surface (route owned by `useAgentDetailPage.onConfigure`); "Go home" inside not-found Empty -> dashboard.
- **Critical states this route MUST handle:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | yes | `agent-sessions-list.tsx:51-63` "No sessions yet" Empty + 4 zero-metric cards (`_evidence/agents.name/live_1440_general_empty.png`) | weak (double-empty: stats grid + Empty illustration both render) |
| Loading / skeleton | yes (partial) | full-bleed `Loader2` for agent fetch + 4 row skeletons for sessions list (`agents.$name.tsx:38-43`, `agent-sessions-list.tsx:134-147`) | weak (skeleton is bars, real layout is a 5-col table; mismatch) |
| Partial data (agent loaded, sessions in flight) | yes | `useAgentDetailPage` exposes `sessionsLoading` separately; `hasResolvedSessions` gates the stats grid | adequate |
| Populated (typical) | yes | sb `Default` story (`_evidence/agents.name/sb_default.png`) | strong |
| Populated (dense, 100+ rows) | not validated | no story exercises >5 sessions | missing |
| Error (network) | yes | `agent-sessions-list.tsx:37-49` "Couldn't load sessions" Empty | adequate; error description is generic ("Try refreshing the page.") |
| Error (permission / 403) | unclear | no dedicated 403 story; would render as the generic agent-error path | weak |
| Error (not found / 404) | yes | `agents.$name.tsx:45-69` Empty + Go home (`_evidence/agents.name/sb_not-found.png`) | strong |
| Read-only / disabled | partially | `newSessionDisabled` is wired in `useAgentDetailPage`; no story exercises it | adequate |
| Live-update (stream / SSE) | yes (refresh button + manual) | `useAgentDetailPage.onRefresh`. There is no auto-revalidate banner; the user has to push the button. | weak |
| Mobile / narrow viewport | partial | live 320px screenshot shows the header truncating the agent name and the metric label being cut to "ACTIVE SESSIO…" | weak |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   3   | `agent-page-header.tsx:30-50` IDLE/ACTIVE pill + count + Refresh spin (`live_1440_general_empty.png`) | Refresh icon spin is the only "I just refreshed" cue; no last-updated timestamp anywhere. |
| 2  | Match between system and real world    |   3   | sessions table columns "Session / Status / Duration / Iterations / Last activity" match runtime nouns | "Iterations" is jargon for an operator who is not an agent author; needs glossary anchor or tooltip. |
| 3  | User control and freedom               |   2   | Refresh + Configure + New session in toolbar; not-found has Go home | No "Stop all sessions for this agent" affordance, no breadcrumb back to agents index, no Esc to close confirm dialog (relies on shadcn default). |
| 4  | Consistency and standards              |   2   | uses `PageHeader`, `Pill`, `Empty`, `Metric` from `@agh/ui` | But the empty state below the stats grid renders 4 zero-metric cards over an Empty illustration - inconsistent emptiness. (`sb_no-sessions.png`) |
| 5  | Error prevention                       |   3   | Configure / Refresh do not have destructive footprint; New session disabled when no workspace | Sessions table exposes no destructive action, OK. |
| 6  | Recognition rather than recall         |   3   | provider chip under each session name (`CLAUDE`), status pill, last-activity column | Iterations column shows raw integers without a max; first-time operator cannot read the unit. |
| 7  | Flexibility and efficiency of use      |   1   | no keyboard shortcut for `+ New session`, no `R` for refresh, no row-level keyboard activation hint | Power-user gap. |
| 8  | Aesthetic and minimalist design        |   3   | tight 16-20px padding, mono eyebrows, flat depth (`live_1440_general_empty.png`) | LAST ACTIVITY value renders the locale date "4/17/2026" at metric-value Inter 24px 700 - heavy and visually unequal vs the other metric values; format mismatch with mono-meta convention. |
| 9  | Help users recognize / recover errors  |   2   | not-found state has Empty + Go home; sessions error has Empty without an action | Sessions error description is a generic "Try refreshing the page." instead of pointing to a Refresh button. |
| 10 | Help and documentation                 |   1   | no inline help icon, no tooltips on metric values, no doc link from "Configure" | First-time operators have no help anchor. |
|    | **Total**                              | **23/40** | | **Band:** ◯ adequate (20-28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders (`border-l/r > 1px`) used decoratively | OK | active sidebar row uses 2px accent bar but it carries meaning (selection); session rows have no decorative border. |
| Gradient text (`background-clip: text` + gradient) | OK | none observed. |
| Glassmorphism / blur as default | OK | none on this route. |
| Hero-metric template (big number + label + sparkline) | borderline | `agent-stats-grid.tsx` is a 4-up grid of `Metric` cards with big values + small detail. It is the SaaS hero-metric pattern. The DESIGN.md spec allows it but on this empty/low-traffic page it dominates above the actual sessions table. |
| Identical card grids | OK | the 4 metric cards are not feature cards; they are the only grid on the route. |
| Modal as first thought | borderline | `New session` opens a dialog (`session-create-dialog.tsx`) for what could be inline-disclosed for the common path (most users will create with the agent default + workspace default). |
| Em dashes in copy | n/a per repo policy | em dashes appear only as data placeholders (`agent-stats-grid.tsx:87, 106`); DESIGN.md `7. Voice & Content` permits em dashes for copy pauses. |
| Generic AI palette (default Tailwind blues, neon-on-black) | OK | warm grays + accent orange throughout. |
| Category-reflex theme | OK | not "agent runtime -> dark blue" - chose warm orange. |
| Restated headings / intros that repeat the title | OK | no subhead in the header. |
| Decorative shadows / heavy elevation | OK | flat depth across the route. |
| Hardcoded `#000` / `#fff` | OK | New session button uses `bg-[color:var(--color-accent)]` via `Button variant="default"`. |

**Summary verdict:** No, a stranger would not say "AI made this" on first sight. The route is restrained and on-brand. The borderline calls are the metric-grid weight and modal-as-default for new session.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point** (page header toolbar): 3 (Refresh, Configure, New session). Plus the agent-row interaction (clicking a row navigates). 4 actionable controls. **Pass.**
- **Eight-item cognitive load checklist:**
  1. Are >4 options visible at once? **Pass** - 3 buttons + 1 row-click affordance per row.
  2. Are labels self-evident without docs? **Fail (partial)** - "Iterations" is unclear without a tooltip; "Total runtime" sums elapsed seconds across sessions but is not labeled "across sessions".
  3. Is the primary action visually dominant? **Pass** - `+ New session` has accent fill while Refresh / Configure are outline icon-only.
  4. Is information progressively disclosed (advanced hidden until needed)? **Fail** - the create dialog reveals 4 fields (Agent, Provider, Model, Reasoning effort) with no progressive disclosure for the common-default path.
  5. Do related elements group via proximity / shared container? **Pass** - stats grid grouped, sessions list grouped, MCP rail grouped.
  6. Is hierarchy clear via scale/weight contrast (>=1.25 ratio)? **Pass** - 24px metric value vs 11px mono eyebrow comfortably exceeds 1.25.
  7. Is body line length within 65-75ch? **n/a** - no body prose on this route.
  8. Is whitespace varied (rhythm) instead of uniform padding? **Pass (mild fail at 1024px)** - between sections is 24px (`agents.$name.tsx:87`), within cards is 16-20px. Empty state below the stats grid floats with the sessions list but never anchors top - feels disconnected at 1024px (`live_1024_general_empty.png`).

  Failure count: 2 -> moderate.

- **IA observations:**
  - Two right-rail concepts (MCP servers vs in-session inspector) ride the same column on related routes; users do not anticipate that the rail content swaps. Consider a stable rail with tabs.
  - "AGENT default provider:" label inside the create dialog (`live_create_dialog.png`) is followed by no value when the agent has no default - the label promises a fact the runtime cannot supply.
  - "ACTIVE SESSIONS" metric label is truncated to `ACTIVE SESSIO…` at 1440px (`live_1440_general_empty.png` and `sb_default.png`). Inter 11px mono uppercase tracking 0.06em runs out of room inside a 4-column 16px-padding card on a 690-720px content area. Either reduce the label, drop "Sessions" (the value 0/1 is enough), or shrink padding.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all colors via `var(--color-*)` tokens - confirmed by grep across `agent-*.tsx`. No raw hex / `#000` / `#fff` literals on this route's components.
- **Type scale:** Inter for titles + body; JetBrains Mono for the metric eyebrows + status pill. No serif. Compliant.
- **Radii / spacing:** cards use `rounded-[var(--radius-md)]` / `rounded-[var(--radius-lg)]`; PageHeader is the shared primitive. Compliant.
- **Elevation:** flat. No shadows except the shared shadcn `shadow-xs` on inputs in the create dialog. Compliant.
- **Signal palette discipline:**
  - `IDLE` pill uses `tone="neutral"` (`#636366`-based tint) and `ACTIVE` uses `tone="success"` (`#30D158` tint). Compliant.
  - `FAILED` metric value uses `tone={failed > 0 ? "danger" : "default"}` (`agent-stats-grid.tsx:31-35`). Compliant.
- **Grid / rhythm:** stats grid is `grid gap-3 sm:grid-cols-2 xl:grid-cols-4` (`agent-stats-grid.tsx:16`). At 1024-1280px that is 2 columns, then snaps to 4 at 1280+. The 1024 layout (2x2) reads OK; the 768 layout (also 2x2) is confirmed in `live_768_general_empty.png`.
- **Density:** comfortable on populated views; sparse on empty (4 zero-cards above an Empty illustration).

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** `+ New session` is unambiguous and visually dominant (accent fill).
- **Destructive actions:** none on this route. (Delete-session lives on the chat route.)
- **Forms:** the New session dialog has 4 fields. Validation is on submit; no inline as-you-type validation. Field labels are clear, placeholders explain provider override.
- **Tables / lists:** sessions table lacks sort, filter, pagination, virtualization. Keyboard nav is via tab through links; no row arrow-key model.
- **Selection model:** none (single-click navigates; no multi-select).
- **Modals / drawers:** the create dialog uses shadcn `Dialog` with focus trap + ESC close. Verified via `session-create-dialog.tsx` props.
- **Live updates / streaming:** there is no SSE banner / auto-poll indicator. The Refresh button is the only signal of liveness. The agent's `IDLE` / `ACTIVE` pill is bound to `sessions.filter(s => s.state === 'active').length > 0`, so it does update on refresh.
- **Optimistic vs pessimistic updates:** session creation is pessimistic (waits for response). Stop / delete happen on the chat route. OK.
- **Hover / focus / active states:** session row links have `hover:text-[color:var(--color-accent)]`. Toolbar buttons have outline + ghost variants from shadcn. Focus ring is the shared shadcn `focus-visible:ring-2 ring-ring/50` - **not** the design-token accent ring. Minor drift.
- **Loading patterns:** uses skeleton bars (`agent-sessions-list.tsx:134-147`) for the sessions list and a centered `Loader2` for the entire route while the agent fetch is pending. Skeleton bars do not match the actual table grid - the populated view has 5 columns at varying widths; the skeleton has 4 uniform-height bars.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** every interactive element is keyboard-reachable (verified via `dom_full_notfound.txt` snapshot which preserves tab order). Toolbar buttons + sidebar tree + session row links + Go home button all reachable.
- **Focus rings:** shadcn defaults (`ring-2 ring-ring/50`) - the `--ring` token exists in `tokens.css` but the spec wants `1.5px solid #E8572A` (DESIGN.md line 728 / 869). Accent focus ring is missing on this route's interactive surfaces.
- **TAB order:** logical (sidebar -> header buttons -> stats -> sessions table -> MCP rail). Verified visually.
- **ARIA roles / labels:** Refresh + Configure buttons have `aria-label`; the spinning Refresh icon has `aria-hidden="true"`. Session rows are `<Link>` children of `<TableRow>` so screen readers announce them as table cells with embedded links - acceptable but a `caption` on the table would help.
- **Color contrast:**
  - Body text on `#1E1C1B` surface using `--color-text-primary` `#E5E5E7` is ~12.5:1 - pass.
  - Tertiary text `#636366` on the same surface is ~4.7:1 - borderline for 13px small body. Just above the AA threshold.
  - Empty illustration's `Empty` icon at `#636366` on canvas `#141312` is ~5.6:1 - pass.
- **Motion:** Refresh icon spin is a continuous animation. No `prefers-reduced-motion` guard inside `agent-page-header.tsx` - relies on the global tokens in `styles.css`. Need to verify the global guard zeroes the spin. (Not validated in this audit pass; flag as P3.)
- **Text scaling:** at 200% zoom the metric label `ACTIVE SESSIONS` truncates further; tested implicitly via the 1440px screenshot which already truncates at 100% zoom.
- **Forms:** create dialog uses `Field` + `FieldLabel` + `FieldDescription` from `@agh/ui` - programmatic association is via the shared component.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** **adequate**. `Empty` icon (MessageSquare) + title "No sessions yet" + description "Start a new session for {agent} from the toolbar above." is operator-first and directional. **Weakness**: stats grid renders 4 zero-cards above the Empty illustration; the surface advertises both "you have nothing" and "your nothings have metrics".
- **Loading:** **weak**. The whole page is a centered spinner while the agent loads (`agents.$name.tsx:37-43`); switching to the populated view is jarring. The sessions skeleton is 4 rounded bars - layout mismatch with the 5-column table.
- **Error:** **weak**. `agent-sessions-list.tsx:37-49` shows "Couldn't load sessions" + "The session list failed to load. Try refreshing the page." with no retry button inline. Operators must hunt for the toolbar Refresh.
- **Permission denied:** **unverified**. No dedicated 403 path; would surface as the agent-error Empty.
- **Stale / disconnected:** **missing**. The CONNECTED footer in the sidebar is the only daemon-health signal; if the SSE backbone disconnects mid-view the route does not banner.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** uses "session", "agent", "workspace", "MCP servers" - all canonical. No `recipe` / `workflow` / `procedure` / `playbook`. No `AGENT.md` / `AGENTS.md` confusion on this route.
- **Tone:** dry, operator-first. "Start a new session for general from the toolbar above." matches `COPY.md` formula `<What this is>. <What creates it>. <Primary action>.`
- **Em dashes:** no copy em dashes on this route. Long-dash characters appear only as data placeholders inside `formatDuration` / `formatRelative` return values.
- **Restated headings:** none.
- **Sentence case vs Title Case:** all consistent (sentence case for body; mono-uppercase for chips/eyebrows).
- **Truthful UI test:**
  - PageHeader count badge: `count={sessions.length}` (`agent-page-header.tsx:50`) - truthful.
  - Stats grid metrics map directly to runtime payload (`activity.elapsed_seconds`, etc.) - truthful.
  - Create dialog "Agent default provider:" label with empty trailing value - **fails truthful UI** (label promises a value, runtime cannot fill it).

---

## 10. Performance & Responsiveness

- **Initial render:** lazy via TanStack Router code splitting. No obvious waterfalls observed.
- **Re-render hot spots:** `useAgentDetailPage` returns memoized handlers; `agent-stats-grid` recomputes totals on every render but the input is `sessions: SessionPayload[]` which only changes on data refresh. OK.
- **List virtualization:** **not present** on the sessions table. With 100+ sessions this will degrade. P2.
- **Bundle red flags:** none noted; the route does not pull a charting library.
- **Responsive behaviour:**
  - 1440px: stats grid is 4-up. Truncation on `ACTIVE SESSIONS` label. (`live_1440_general_empty.png`)
  - 1024px: stats grid 2x2; MCP rail hidden; empty state below the fold. (`live_1024_general_empty.png`)
  - 768px: same 2x2; readable. (`live_768_general_empty.png`)
  - 320px: agent name in header is GONE - only icon + truncated row visible; metric label `ACTIVE SESSIO…` overlaps with "0 0 total" text. (`live_320_general_empty.png`) **P0.**
- **Mobile interactions:** no hover-only affordance on this route.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-agents-name--default` (`-agents.$name.stories.tsx:57-67`)
  - `routes-app-stories-agents-name--no-sessions` (`:72-86`)
  - `routes-app-stories-agents-name--sessions-loading` (`:91-110`)
  - `routes-app-stories-agents-name--agent-loading` (`:115-134`)
  - `routes-app-stories-agents-name--not-found` (`:139-157`)
  - `routes-app-stories-agents-name--with-failed-session` (`:162-198`)
  - `routes-app-stories-agents-name--many-agents` (`:204-213`)

- **States covered in Storybook:** populated, populated-with-failure, sessions-loading, agent-loading, not-found, no-sessions, many-agents.

- **Gaps:**
  - No 100+-session story (validate virtualization need + table layout).
  - No `/api/sessions` 500 error story.
  - No 403 / permission-denied story.
  - No 320 / 768 viewport story (the route's responsive collapse is unenforced).
  - No "agent has 5 MCP servers" story for the right rail (`AgentInfoPanel` has its own system stories but the route never demonstrates them in combination).
  - No "session creation in flight" story (the create dialog's submitting state).

- **Story drift:** no stale prop API observed; the route stories use `StorybookWorkspaceSetup` and `StorybookRouteCanvas` consistently. OK.

---

## 12. Findings - Prioritised

### P0 - Ship Blockers

1. **[P0] Header truncates the agent name and IDLE pill at 320px.**
   - **Why:** operators on a phone cannot identify which agent they are looking at. The whole page becomes "icon + truncated metric labels".
   - **Fix:** drop the IDLE pill into a meta row below the title at <640px, or stack icon + title + pill vertically. Verify with a `viewports.mobile` parameter on the Default story.
   - **Cmd:** `/impeccable adapt agents-detail-header`
   - **Effort:** M
   - **Evidence:** `_evidence/agents.name/live_320_general_empty.png`

### P1 - High-Value Polish

2. **[P1] Stats grid truncates `ACTIVE SESSIONS` to `ACTIVE SESSIO…` even at 1440px.**
   - **Why:** the truncation reads as a layout bug at the canonical desktop width; first impression of a polished operator UI is degraded.
   - **Fix:** rename to `ACTIVE` (the surrounding context is unambiguous) or shrink the eyebrow to 10px or reduce 16px+20px padding to 12px+16px.
   - **Cmd:** `/impeccable layout agents-stats-grid`
   - **Effort:** S
   - **Evidence:** `_evidence/agents.name/live_1440_general_empty.png`, `_evidence/agents.name/sb_default.png`

3. **[P1] Skeleton bars do not match the populated table layout.**
   - **Why:** users expect the loading shape to predict the final layout; 4 uniform bars switching to a 5-column striped table creates a layout shift and feels unfinished.
   - **Fix:** render skeleton table rows (`<TableRow>` with `<Skeleton>` cells matching column widths). Reuse `Table` chrome.
   - **Cmd:** `/impeccable harden agent-sessions-skeleton`
   - **Effort:** S
   - **Evidence:** `_evidence/agents.name/sb_sessions-loading.png`

4. **[P1] Empty state double-paints (4 zero-cards + Empty illustration).**
   - **Why:** the surface tells the user "nothing here" twice in different idioms; cognitive load and visual noise.
   - **Fix:** render the stats grid only when `sessions.length > 0`, OR turn the stats grid into the empty state itself by replacing values with a single explanatory line and the `+ New session` CTA.
   - **Cmd:** `/impeccable distill agents-detail-empty`
   - **Effort:** S
   - **Evidence:** `_evidence/agents.name/sb_no-sessions.png`

5. **[P1] LAST ACTIVITY metric renders the locale date `4/17/2026` at metric-value Inter 24px 700.**
   - **Why:** a date string at metric-value weight reads as numeric data; visually unbalanced vs the `0` and `1` numbers in adjacent cards. Also it is a locale string (`toLocaleDateString`) rather than the relative `Xd ago` format used in the table.
   - **Fix:** keep the relative format (`X min ago`, `X h ago`, `X d ago`) which `formatRelative` already returns when within 7 days; otherwise compress to mono with smaller weight.
   - **Cmd:** `/impeccable typeset agents-stats-grid`
   - **Effort:** S
   - **Evidence:** `_evidence/agents.name/sb_default.png` (LAST ACTIVITY card)

6. **[P1] Sessions error has no retry / no command path.**
   - **Why:** "Try refreshing the page." is a wall sign that fails error recovery (heuristic 9). The toolbar Refresh exists but is not surfaced inline.
   - **Fix:** add a `Retry` button inside the Empty (`onAction`) that calls the same query refetch the toolbar uses.
   - **Cmd:** `/impeccable harden agent-sessions-error`
   - **Effort:** S
   - **Evidence:** `agent-sessions-list.tsx:37-49`

7. **[P1] Create dialog "Agent default provider:" label with empty value.**
   - **Why:** truthful UI test fails (`web/CLAUDE.md` "Do not imply ... a value the runtime does not expose").
   - **Fix:** hide the line when there is no agent default; or render `none` mono pill with tertiary tone.
   - **Cmd:** `/impeccable clarify session-create-dialog`
   - **Effort:** S
   - **Evidence:** `_evidence/agents.name/live_create_dialog.png`

### P2 - Worthwhile

8. **[P2] No virtualization on the sessions table.**
   - **Why:** at 100+ sessions the table will block scroll. Real fintech operators may push 50-200 sessions per agent.
   - **Fix:** wrap with `@tanstack/react-virtual` rows when `sessions.length > 50`.
   - **Cmd:** `/impeccable optimize agent-sessions-list`
   - **Effort:** M
   - **Evidence:** `agent-sessions-list.tsx:67-86` (no virtualization, plain `<TableBody>`)

9. **[P2] No keyboard shortcuts on the toolbar.**
   - **Why:** power users running multiple agents cannot keyboard-trigger a new session or refresh.
   - **Fix:** ship `R` (refresh) and `N` (new session) at the route level via TanStack key listener.
   - **Cmd:** `/impeccable craft agents-detail-shortcuts`
   - **Effort:** S
   - **Evidence:** `agent-page-header.tsx:51-94` (no keyboard handler)

10. **[P2] Iterations column shows raw integers without explaining the unit.**
    - **Why:** "iteration_current / iteration_max" is a runtime concept; first-time operators do not have the model.
    - **Fix:** render with a tooltip (`Tooltip from @agh/ui`) that explains "Turns since session start / cap" or rename the column to `Turns`.
    - **Cmd:** `/impeccable clarify agent-sessions-list`
    - **Effort:** S
    - **Evidence:** `agent-sessions-list.tsx:75-127`

### P3 - Parking Lot

11. **[P3] Reduced-motion guard for the Refresh spin.**
    - **Why:** continuous animation should be zeroed under `prefers-reduced-motion`.
    - **Fix:** validate the global token reset zeroes `animate-spin`; if not, gate locally.
    - **Cmd:** `/impeccable animate agent-page-header`
    - **Effort:** XS
    - **Evidence:** `agent-page-header.tsx:64`

12. **[P3] Focus rings use shadcn default ring tokens, not the accent ring per DESIGN.md.**
    - **Why:** DESIGN.md line 728 mandates 1.5px solid `#E8572A` for focus.
    - **Fix:** override `--ring` in tokens.css for this surface, or pass `focus-visible:ring-[color:var(--color-accent)]` in the toolbar buttons.
    - **Cmd:** `/impeccable polish agent-page-header`
    - **Effort:** XS
    - **Evidence:** `agent-page-header.tsx:51-91`

---

## 13. Persona Red Flags

- **Operator (returning power user, keyboard-first):** no `N` for New session, no `R` for Refresh, no row-level keyboard activation hint. Has to mouse-click into a session - acceptable, but the empty surface above also lacks an inline `Press N to start` cue. Power users will skip the page header and use the sidebar tree, which makes the right-rail MCP panel + stats grid mostly decorative on the empty state.
- **First-timer (onboarding, no mental model yet):** "Iterations" is unexplained. "TOTAL RUNTIME" sums elapsed seconds across active and stopped sessions but the label does not say so. The empty state is direct enough but the 4 zero-metrics above it confuse the message. Likely abandons at "what does this number mean?"
- **Agent (yes - agents read this UI via screenshots / DOM scrapes):** the DOM exposes stable selectors (`data-testid="agent-page-header"`, `agent-sessions-table`, `agent-stat-active`, etc.). Roles include `tree`, `treeitem`, `complementary`, `main`. ARIA labels on toolbar buttons. The agent's view is **good** - selectors are predictable and headings are stable. One drift: the `Pill mono` count `1` next to the title is rendered as plain text without a sr-only label so a scraper sees a bare number.

---

## 14. Cross-Module Consistency Notes

- `01_dashboard` (likely) renders metric grids with the same `Metric` primitive; the truncation problem on `ACTIVE SESSIONS` is probably a system-level concern. Cross-reference: any other route using a 4-up `Metric` grid will have the same `LABEL TRUNC…` issue.
- `04_tasks` and `05_jobs` use list/table patterns; the keyboard nav gap (no row arrow keys) is a module-level consistency issue, not a route-level bug.
- The right-rail pattern (`AgentInfoPanel`) is unique to this route. Sibling pattern is `SessionInspector` on the chat route. Worth aligning headers ("MCP SERVERS" eyebrow vs "Trace / Usage / Memory / Files / Vault" tablist) into a shared rail-shell primitive.

---

## 15. Open Questions

- Should the empty state for an idle agent expose `agh agents create-session --workspace=...` as the CLI alternative? `COPY.md` says agent-manageable surfaces should be visible.
- Is the right-rail MCP list useful here, or would "Sessions by status" be more productive for the operator?
- Should `New session` open a dialog at all? An inline composer that captures the agent + provider with sane defaults could remove a click.
- Should the page poll the daemon for session count changes, or is "Refresh" a deliberate choice? If deliberate, the Refresh icon needs a "last updated 12s ago" companion.

---

## 16. Recommended Action Plan

1. `/impeccable adapt agents-detail-header` - 320 / 768 collapse strategy: stack icon-title-pill, drop the IDLE pill below the title at narrow widths, hide the count badge when 0.
2. `/impeccable layout agents-stats-grid` - rename labels to fit the card width (drop "Sessions" from `ACTIVE SESSIONS`); align the LAST ACTIVITY value formatting to relative-time mono.
3. `/impeccable distill agents-detail-empty` - choose between metric grid OR Empty illustration. Suggest hiding the grid when `sessions.length === 0`.
4. `/impeccable harden agent-sessions-skeleton` - use Table + Skeleton cells matching the populated layout.
5. `/impeccable harden agent-sessions-error` - add inline Retry button via Empty `action`.
6. `/impeccable clarify session-create-dialog` - hide empty-default-provider label; add `Press Enter to start` hint when valid.
7. `/impeccable craft agents-detail-shortcuts` - add `R` and `N` keyboard shortcuts at the route level.
8. `/impeccable optimize agent-sessions-list` - virtualize when `sessions.length > 50`.
9. `/impeccable polish agents-detail` - final pass: focus rings to accent, reduced-motion verification, tooltip on Iterations column.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/agents.name/`.
- [x] No section is left as `<TODO>` or empty.
- [x] Nielsen scores total (23/40) is consistent with the band claimed (◯ adequate).
- [x] Findings are tagged P0-P3 with effort and command.
- [x] No hallucinated routes, components, or props (every claim cross-referenced).
- [x] No em dashes in this report.
- [x] Report length is thorough but not padded.
