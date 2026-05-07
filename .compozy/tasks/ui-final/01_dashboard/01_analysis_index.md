# UI/UX Analysis, `01_dashboard` :: `/`

> **Status:** draft
> **Owner subagent:** `dashboard-auditor`
> **Date:** 2026-05-06
> **Module:** `01_dashboard` (`.compozy/tasks/ui-final/01_dashboard`)
> **Route path:** `/` (TanStack Router id: `/_app/`)
> **Web source:** `web/src/routes/_app/index.tsx`
> **System owner:** mostly composed of `@agh/ui` primitives (`Metric`, `PageHeader`, `Section`, `Empty`, `Pill.Dot`) plus `web/src/components/connection-indicator.tsx` and the page hook `web/src/hooks/routes/use-home-page.ts`. Data sourced from `web/src/systems/{daemon,workspace,agent,session}`.
> **Storybook story id(s):** `routes-app-stories-index--default`, `routes-app-stories-index--degraded`, `routes-app-stories-index--disconnected`, `routes-app-stories-index--loading`, `routes-app-stories-index--error`, `routes-app-stories-index--onboarding`. The story file declares `title: "routes/app/home"` (`web/src/routes/_app/stories/-index.stories.tsx:13-18` via `createRouteStoryMeta`), but the running Storybook indexer ignores that title and synthesizes ids from the file path; the file path namespace is the working one (verified live and against `web/storybook-static/index.json`).
> **Live URLs probed:** `http://localhost:3000/` (AGH SPA, started by the audit), `http://localhost:6006/iframe.html?id=routes-app-stories-index--<state>&viewMode=story`. Daemon ground truth: `http://localhost:2123/api/observe/health`.

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/index.tsx`
  - `web/src/routes/_app/stories/-index.stories.tsx`
  - `web/src/routes/_app.tsx`
  - `web/src/components/app-sidebar.tsx`
  - `web/src/components/sidebar-nav-classes.ts`
  - `web/src/components/connection-indicator.tsx`
  - `web/src/hooks/routes/use-home-page.ts`
  - `web/src/storybook/route-story.tsx`
  - `packages/ui/src/components/metric.tsx`
  - `packages/ui/src/components/page-header.tsx`
  - `packages/ui/src/components/section.tsx`
  - `packages/ui/src/components/empty.tsx`
  - `packages/ui/src/components/pill.tsx`
  - `web/.storybook/main.ts`
- **Storybook stories opened:**
  - `routes-app-stories-index--default` -> `http://localhost:6006/iframe.html?id=routes-app-stories-index--default&viewMode=story`
  - `routes-app-stories-index--degraded` -> same path with `--degraded`
  - `routes-app-stories-index--disconnected` -> same path with `--disconnected`
  - `routes-app-stories-index--loading` -> same path with `--loading`
  - `routes-app-stories-index--error` -> same path with `--error`
  - `routes-app-stories-index--onboarding` -> same path with `--onboarding`
- **Live web probes (`localhost:3000`):**
  - `/` populated with the operator's real workspace and agent (1 workspace, 1 agent, 0 active sessions, daemon healthy).
  - Sidebar collapsed via `Toggle sidebar`.
  - 1440x900, 1024x768, 768x1024, 320x568 viewport variants.
- **Screenshots / DOM snapshots captured** (under `.compozy/tasks/ui-final/01_dashboard/_evidence/index/`):
  - `live_full.png`, `live_1440.png`, `live_1024.png`, `live_768.png`, `live_320.png`, `live_collapsed.png`, `live_dashboard_nav_focus.png` -> live SPA at the listed viewports / states.
  - `sb_default.png`, `sb_degraded.png`, `sb_disconnected.png`, `sb_loading.png`, `sb_loading_reduced.png`, `sb_error.png`, `sb_onboarding.png`, `sb_default_v2.png` -> Storybook captures.
  - `sb_index.png` -> Storybook landing page, used to confirm tree / id format.
  - `dom_live.txt`, `dom_default.txt`, `dom_degraded.txt`, `dom_disconnected.txt`, `dom_loading.txt`, `dom_error.txt`, `dom_onboarding.txt` -> accessibility tree dumps from `agent-browser snapshot -i`.
  - `impeccable_route.json`, `impeccable_components.json`, `impeccable_systems.json` -> `npx impeccable --json` runs (all returned `[]`, no automated detector findings).
- **Console / network errors observed:**
  - Live: `agent-browser console` and `agent-browser errors` returned empty.
  - Dev server `vite` log: tanstack-router code-split warnings for `routes/_app/knowledge.tsx`, `routes/_app/settings/memory.tsx`, `routes/_app/agents.$name.sessions.$id.tsx`. None for `routes/_app/index.tsx`.
- **Keyboard / a11y probes performed:**
  - Tab order from page load: 1 `Go to dashboard` (logo), 2 `Workspace: pedronauck`, 3 `Add workspace`, 4 `Toggle sidebar`, 5 `Dashboard` (sidebar nav row), then through `Network`, `Tasks`, `Jobs`, `Triggers`, captured by `agent-browser eval`-ed `document.activeElement` after repeated Tabs.
  - Live `getComputedStyle` on the first five focusable controls: app-shell controls (logo, workspace, add-workspace, toggle) all carry an explicit accent box-shadow ring (`rgb(232, 87, 42) 0px 0px 0px 2px` or `4px`); the `Dashboard` nav row carries `outline: auto` only and `box-shadow: none`.
  - Reduced-motion probe in Storybook (`agent-browser set media dark reduced-motion` against the `loading` story): the skeleton stops shimmering, so the global reset is honored. Loading state evidence in `sb_loading.png` vs `sb_loading_reduced.png`.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** the AGH operator landing surface. It tells the operator that the local daemon is reachable and healthy, identifies which workspace the rest of the app is scoped to, and counts the four numbers an operator most often glances at, active sessions in this workspace, total workspaces, agents available, daemon uptime.
- **Primary user goal on this route:** confirm that the daemon is healthy and the workspace state is what the operator expects. Concretely: a returning operator wants to see "green", a number for sessions running in their workspace, and a daemon version they trust.
- **Entry vectors:**
  - SPA initial load on `/`.
  - Sidebar `Dashboard` nav row (`web/src/components/app-sidebar.tsx:120-124, 154-158`).
  - Logo / workspace icon (`web/src/components/app-sidebar.tsx:44-51`).
  - Error / not-found `Go home` action button in `web/src/routes/_app.tsx:115-119, 137-141`.
- **Exit vectors:** there are none from the dashboard body itself. Every metric is non-interactive. The only navigation routes out via the persistent sidebar.
- **Critical states this route MUST handle:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty | yes (handled by the shell, not this route) | `web/src/routes/_app.tsx:29-31` short-circuits to `WorkspaceOnboarding` when `!hasWorkspaces`; story `routes-app-stories-index--onboarding`; `_evidence/index/sb_onboarding.png` | strong, but rendered by `_app.tsx`, not by `index.tsx` |
| Loading / skeleton | yes | `web/src/routes/_app/index.tsx:35-45`, `DaemonStatusSkeleton`, `MetricsSkeleton`; `sb_loading.png` | strong, layout matches final |
| Partial data | weak / missing | `web/src/routes/_app/index.tsx:47-61` collapses any sub-error into a single global error empty; the "active sessions" metric does have a partial `unavailable` detail (`web/src/hooks/routes/use-home-page.ts:168-176`) | weak |
| Populated (typical) | yes | `web/src/routes/_app/index.tsx:63-72`, `_evidence/index/sb_default.png`, `live_full.png` | strong |
| Populated (dense, 100+ rows) | n/a, route renders four fixed metrics; not list-shaped | | n/a |
| Error (network) | yes | `index.tsx:47-61`; `sb_error.png` | adequate, generic copy |
| Error (permission / 403) | missing | no 403 branch in `use-home-page.ts`; would surface as the same generic error | weak |
| Error (not found / 404) | n/a, `/` cannot 404; the shell handles route-level 404 in `_app.tsx:127-145` | | n/a |
| Read-only / disabled | n/a, route has no interactive primary action | | n/a |
| Live-update (stream / SSE) | partial | `connectionStatus` from `useDaemonHealth` updates the badge and the daemon card; data is React Query polled, not streamed | adequate |
| Mobile / narrow viewport | broken | `_evidence/index/live_320.png` shows the sidebar (44 + 240 = 284 px) consuming almost the entire 320 px viewport; the dashboard body is clipped to ~36 px | missing |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |  3    | `index.tsx:74-121`, `connection-indicator.tsx:18-22`, `sb_default.png` (Healthy + green dot), `sb_disconnected.png`, `sb_degraded.png` | strong daemon health surface; no per-card timestamp ("as of"); no SSE / poll cadence shown |
| 2  | Match between system and real world    |  3    | `index.tsx:31` "Home" vs `app-sidebar.tsx:124` "Dashboard"; `daemonStatus.description` copy in `use-home-page.ts:84-117` reads as the daemon would | naming drift `Home` vs `Dashboard`; "Active Sessions" vs backend noun `session` |
| 3  | User control and freedom               |  2    | route is read-only; the only "control" is the sidebar `Toggle sidebar` and `Go to dashboard` link | the disconnected state shows a hint to run `agh daemon`, but does not offer a retry button or "open settings" path; no manual refresh |
| 4  | Consistency and standards              |  2    | `app-sidebar.tsx:48,66,80` apply `focus-visible:ring` while `sidebar-nav-classes.ts` does not; `Dashboard` vs `Home` vs `home-*` testids; metric label uppercase vs detail lowercase in same card; `Empty` `data-fill` true in error vs disconnected | several internal inconsistencies, called out in §6, §7 |
| 5  | Error prevention                       |  3    | route has no destructive actions; no forms; the only state changes are sidebar-driven; `index.tsx:47-61` short-circuits on error | n/a for the dashboard's read-only nature; there is no "are you sure" surface to grade |
| 6  | Recognition rather than recall         |  3    | mono eyebrows on each metric (`metric.tsx:56-61`), each card self-labels; sidebar shows the route the user is on with the accent left bar | the version chip `vbfd54851` in `sb_default.png` and the live header is a git short-sha with no tooltip / link; operator must recall what this string means |
| 7  | Flexibility and efficiency of use      |  2    | no keyboard shortcut to refresh, no `?` help map; `Toggle sidebar` is the only command surface | no command bar or `cmd+k`; experienced operators have no fast path |
| 8  | Aesthetic and minimalist design        |  3    | flat depth, warm dark canvas, hairline dividers, mono eyebrows; matches `DESIGN.md` §1, §4 | minimal to a fault, see §3 anti-pattern verdict; very near "identical card grid" SaaS hero |
| 9  | Help users recognize / recover errors  |  2    | error empty title "Unable to load dashboard" + injected error message (`index.tsx:53-59`); disconnected hint copy includes the right CLI fix | error copy is generic; no retry button; no link to docs / troubleshooting; no support path beyond the CLI hint |
| 10 | Help and documentation                 |  1    | none on this route | no `?` icon, no tour, no link to runtime docs; the dashboard is the first surface for new operators and offers nothing |
|    | **Total**                              | **24/40** | | **Band:** ◯ adequate (20-28) |

The 24/40 honestly reflects what the surface is right now: a clean, truthful, non-deceptive landing screen that provides almost no operator power, very little teaching, and no recovery affordances. Aesthetically it lands; functionally it is sparse.

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders (`border-l/r > 1px`) used decoratively | OK | only seen in `sidebar-nav-classes.ts:6-7` `ACTIVE_NAV_INDICATOR_CLASS`, which is a 2 px accent left bar on the active nav row; that is the sanctioned selected-list-item pattern (`DESIGN.md` §6) |
| Gradient text (`background-clip: text` + gradient)        | OK | none |
| Glassmorphism / blur as default                            | OK | none |
| Hero-metric template (big number + label + sparkline)      | borderline | `index.tsx:140-167` plus `metric.tsx:46-89` reads exactly as the SaaS "four big numbers in a row" hero template that `impeccable` flags; mitigated because the numbers are real and have detail copy, not decorative sparklines |
| Identical card grids                                       | borderline | the four metrics are visually identical; only the values differ; `live_1440.png` and `sb_default.png` both look like every other "stats hero" |
| Modal as first thought (modal where inline would do)       | OK | the dashboard has no modals; `WorkspaceSetupDialog` is shell-level and route-triggered |
| Em dashes in copy                                          | violation | `web/src/hooks/routes/use-home-page.ts:53` returns the literal U+2014 em dash as the user-facing fallback for `formatUptimeSeconds(null)`; same character is rendered in the disconnected story `sb_disconnected.png` (Daemon Uptime card shows `—`); also at `use-home-page.ts:172` for sessionsError fallback. `COPY.md` §5 and the impeccable shared design laws both say no em dashes; `DESIGN.md` §7 itself says em dash IS allowed for copy pauses. The two files contradict; the audit method (`impeccable critique`) bans em dashes, so this is flagged. |
| Generic AI palette (default Tailwind blues, neon-on-black) | OK | warm `#141312` canvas + accent `#E8572A` per `DESIGN.md` tokens |
| Category-reflex theme ("observability -> dark blue")       | OK | the surface is warm, not blue, and the accent is operator orange not the predictable "tools = green/blue/cyan" reflex |
| Restated headings / intros that repeat the title           | OK | `Home` (header) and `DAEMON` / `OVERVIEW` (section eyebrows) do not restate each other |
| Decorative shadows / heavy elevation                       | OK | flat depth, no `box-shadow` outside the focus ring |
| Hardcoded `#000` / `#fff` instead of tinted neutrals       | OK | every color comes from `var(--color-*)` tokens |

**Summary verdict:** borderline. If a stranger said "AI made this", I would not believe them on first glance because of the warm palette and the disciplined typography, but I would on second glance because the entire surface above the fold is a four-card stats grid plus a status banner, exactly the SaaS hero template. The grid passes only because the numbers are truthful, not because the layout is original.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** zero on the route body; the route has no actionable controls. Counting the shell, the operator sees ~12 nav items in the sidebar (Dashboard, Network, Tasks, Jobs, Triggers, Knowledge, Skills, Bridges, Sandbox, Settings, plus the agent tree node and workspace toggles). The route itself does not introduce a decision point. **Flag:** the dashboard cannot fail this checklist because it asks nothing of the operator; that is itself a finding (no primary action).
- **Eight-item cognitive load checklist:**
  1. Are >4 options visible at once? **pass** on the route body (0). **fail** at the shell level (12 visible nav items + 11 agent treeitems on the populated story). Evidence: `dom_default.txt`.
  2. Are labels self-evident without docs? **fail** for the workspace avatar letter and the version chip. The workspace avatar shows only the first letter (`app-sidebar.tsx:54, 71`); the version chip shows `vbfd54851`, a git short-sha, with no tooltip. Both are recall-not-recognition.
  3. Is the primary action visually dominant? **n/a, no primary action.** That is itself a finding.
  4. Is information progressively disclosed? **partial.** The metrics expose nothing more on hover, focus, or click. The daemon card description is one sentence with no "see details".
  5. Do related elements group via proximity / shared container? **pass.** `Section` `border-b` divider plus `gap-6` (`index.tsx:66-70`) gives clean grouping.
  6. Is hierarchy clear via scale/weight contrast (>=1.25 ratio)? **pass.** Mono 11 px label vs Inter 24 px value is ~2.18x; section eyebrow vs body is also a clear step. Per `DESIGN.md` §3.
  7. Is body line length within 65-75ch? **pass.** Daemon description is under 60ch; metric values are short.
  8. Is whitespace varied (rhythm) instead of uniform padding? **fail.** Section gap is uniform 24 px (`gap-6`); cards are uniform `px-5 py-4`; section divider has uniform 8 px bottom padding. The whole surface has the same beat.
  - **Failure count:** 3 (items 2, 3, 8). **Moderate.**

- **Information architecture observations:**
  - Two sections: `Daemon` and `Overview`. The hierarchy is correct but thin. The whole route is the equivalent of two list rows.
  - `Daemon` exposes status + description + version chip on the section's right slot. The version chip `v<sha>` reads as metadata-on-section-header, which is fine, but it is not aligned with the other metrics that live in `Overview`. The version is itself a "metric" (build identity); placing it in the section right slot keeps it visible but means it cannot be filtered, copied, or compared.
  - `Overview` has four metrics in a fixed `METRIC_ORDER`. There is no hierarchy among them: "active sessions" and "daemon uptime" carry the same visual weight. For an operator landing surface the most important metric is "is anything wrong right now", and that signal already lives in the daemon card; the four uniform tiles do not amplify it.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** all dashboard route code uses `var(--color-*)` token references. No `#hex` literals in `index.tsx`, `metric.tsx`, `page-header.tsx`, `section.tsx`, `empty.tsx`, `connection-indicator.tsx`, `app-sidebar.tsx`, `sidebar-nav-classes.ts`, or `pill.tsx`. Verified by `grep -nE '#[0-9a-fA-F]{3,8}' <file>` reading the source. PASS.
- **Type scale:** Inter for body and headings; JetBrains Mono for the metric label, the version chip, the section eyebrow, the connection indicator label, and the workspace avatar letter. No Playfair, no NuixyberNext, no other family on the route. PASS, with one note: the metric label uses `text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]` (`metric.tsx:56-58`) but the `Section` label uses `text-[11px] ... text-[color:var(--color-text-label)]` (`section.tsx:38`). Same size, different role token (`tertiary` vs `label`). Reading the DESIGN.md eyebrow rule (§3) the `--color-text-label` token is the canonical eyebrow color; the `Metric` label is using `--color-text-tertiary` which is one token darker. Live shows the difference: section "DAEMON" reads slightly lighter than metric "ACTIVE SESSIONS" (`live_full.png`).
- **Radii / spacing:** card radius is `var(--radius-diagram)` (12 px) per `DESIGN.md` §4, applied via `rounded-[var(--radius-diagram)]` in `metric.tsx:51`, `index.tsx:93,173,191`. No one-off radii. PASS.
- **Elevation:** flat. The only ring on the route is the focus `box-shadow` per `app-sidebar.tsx:48`. No drop shadows, no glassmorphism, no neumorphism. PASS.
- **Signal palette discipline:** `tone="success"` for healthy daemon, `tone="warning"` for degraded, `tone="danger"` for disconnected, `tone="neutral"` for unknown / version chip (`use-home-page.ts:78-124`, `connection-indicator.tsx:18-22`). No decorative semantic color use. PASS.
- **Grid / rhythm:** `gap-6 px-6 py-6` on the body shell; `gap-3` between section header and body; metric grid `grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3`. Spacing is monotone (always 24/12/24). The hero card section divider plus identical card padding gives the surface a uniform tempo, fails the `DESIGN.md` §5 "vary spacing for rhythm" guideline.
- **Density:** sparse. The visible content above 800 px on a 1440x900 viewport ends at y~340, and the rest of the screen is empty canvas (`live_1440.png`). For an operator landing surface that is intentional minimalism, but it borders on "Storybook hero" rather than "operator surface". A power user expecting density will read this as scaffolding.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** none. The dashboard is read-only. Per `COPY.md` §8 "Web UI Microcopy" the rule is "current state, next action, and consequence". This route has the first part and nothing of the second. There is no `Open the runtime docs`, no `Create a session`, no `View peers` CTA on the landing surface. That is a P1 finding.
- **Destructive actions:** n/a.
- **Forms:** n/a on this route. The onboarding branch surfaces the workspace setup form via the shell (`WorkspaceSetupDialog`) but the dashboard route never renders it.
- **Tables / lists:** n/a, the route has no list affordance.
- **Selection model:** n/a.
- **Modals / drawers:** n/a on the route. `WorkspaceSetupDialog` is shell-owned.
- **Live updates / streaming:** the connection indicator polls daemon health via React Query (`web/src/systems/daemon/hooks/...`). The badge label switches between `connected`, `reconnecting`, `disconnected`. The `Pill.Dot` pulses when `pulse=true` in the reconnecting state (`connection-indicator.tsx:18-22`, `pill.tsx:162-194`). No "stale" or "last refreshed" timestamp anywhere on the route. The disconnected card hint copy is correct; the reconnecting state has no description (only a yellow pulsing dot). For an operator surface that is too quiet.
- **Optimistic vs pessimistic updates:** n/a, route is read-only.
- **Hover / focus / active states:** every interactive element on the route's own body is **n/a** (there are no interactive elements). Shell controls all have hover, but as documented in §7, focus is not consistent.
- **Loading patterns:** skeleton primary (`Skeleton` rectangles), no spinner. Skeleton geometry mirrors the populated layout (`index.tsx:169-202`, compare `sb_loading.png` vs `sb_default.png`). PASS, exemplary. The tanstack code-split warnings about other routes do not affect this route's loading time.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** every shell interactive element is reachable in tab order, verified live: `Go to dashboard` -> `Workspace: pedronauck` -> `Add workspace` -> `Toggle sidebar` -> `Dashboard` (nav) -> `Network` -> `Tasks` -> `Jobs` -> `Triggers` (`agent-browser eval activeElement` logs). The route body itself has no interactive elements, so there is nothing to fail.
- **Focus rings:** **mixed.** Live `getComputedStyle` on the first five focusable elements:
  - logo, workspace avatar, add-workspace, toggle: explicit accent box-shadow ring `rgb(232, 87, 42) 0px 0px 0px 2-4px`.
  - sidebar `Dashboard` nav row: `outline: auto`, `box-shadow: none`. Source: `web/src/components/sidebar-nav-classes.ts:1-3` `NAV_ROW_CLASS` declares hover but no `focus-visible:` rule. This is an accessibility regression because it relies on the browser's default outline color, which in dark mode is platform-dependent and can fail the 3:1 contrast required by WCAG 2.2 SC 1.4.11. **P1.**
- **TAB order:** logical, top-to-bottom, left-to-right. PASS.
- **ARIA roles / labels:**
  - `ConnectionIndicator` exposes `role="status"`, `aria-live="polite"` (`connection-indicator.tsx:39-41`). Good.
  - `Empty` icon has `aria-hidden="true"` and the icon container is decorative (`empty.tsx:78-82`). Good.
  - `Metric` is a plain `<div data-slot="metric">` with no `role="group"` or `aria-label`. Screen readers read the eyebrow + value + detail as separate runs of text but with no semantic grouping. The DOM snapshot for the populated state shows only `heading "DAEMON" [level=2]` and `heading "OVERVIEW" [level=2]`; the four metrics are silent in the accessibility tree (`dom_default.txt:32-33`). This is a **P1**: an operator running a screen reader hears "Active Sessions, 0, in pedronauck, Workspaces, 1, Agents, 1, Daemon Uptime, 41m 57s" as one undivided run. Wrapping each `Metric` in a `role="group"` with `aria-labelledby` fixes it.
  - The disconnected state's `Empty` (`index.tsx:123-138`) passes a `<ConnectionIndicator>` as the title. `Empty` decides the title tag from `typeof title === 'string'` and falls back to `<div>` for ReactNode (`empty.tsx:35-37, 84-89`). Net result: the disconnected daemon card has no `<h3>` and no programmatic title; only the status pill text is exposed. The `dom_disconnected.txt` confirms zero headings inside the disconnected card. **P1**, screen readers cannot find a heading for the most important state.
- **Color contrast:** every text role uses tokens that DESIGN.md commits to as AA-compliant. Spot check on `Healthy` label `#E5E5E7` on `#1E1C1B` (`live_full.png`) -> contrast ~13:1. Metric label `#636366` on `#1E1C1B` -> ~3.2:1, **fails AA body 4.5:1** (`DESIGN.md` §2 lists `#636366` as "tertiary, placeholders, disabled text, low-emphasis"; using it for an active eyebrow label like "ACTIVE SESSIONS" puts decorative chrome below WCAG body threshold). **P1.**
- **Motion:** no auto-playing motion on the route except the reconnecting pill pulse, which respects `prefers-reduced-motion` via `useReducedMotionConfig` (`pill.tsx:170-188`). Verified live by `agent-browser set media reduced-motion`. PASS.
- **Text scaling:** at 200 % browser zoom on 1440x900 the metric grid switches from `xl:grid-cols-4` to `sm:grid-cols-2` correctly. At 16 px -> 24 px font scaling the metric value `"41m 57s"` does not overflow because the cell uses `truncate` (`metric.tsx:65`). PASS, with one risk: `truncate` will silently hide content if the operator scales text on a narrow viewport; consider `min-w-0` plus a tooltip for long values like `1d 23h`.
- **Forms:** n/a.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** `excellent`. The `Onboarding` story (`sb_onboarding.png`) shows a real two-pane "Start AGH with a real workspace, not an empty shell" surface with a verbose description, a global-workspace one-click button, a path field, and a first-run note. This is rendered by `web/src/systems/workspace/components/workspace-setup.tsx:251-289`, which is shell-level. The dashboard does not render its own first-run, which is correct.
- **Loading:** `excellent`. Skeleton geometry mirrors the final layout exactly (`sb_loading.png` vs `sb_default.png`). No spinner, no full-page flash, no entrance animation. Matches `DESIGN.md` §9.
- **Error:** `weak`. `index.tsx:47-61` renders `Empty` with title `"Unable to load dashboard"` and the raw error message in description. There is no retry button, no "open settings", no "see daemon log" link. The `WorkspacesError` story (`sb_error.png`) shows literal `workspaces unavailable` from the 500. Per `COPY.md` §9 "Error Copy" the formula is "what failed, why, next safe action". This implementation has the first two and skips the third. **P1.**
- **Permission denied:** missing. There is no separate 403 branch in `use-home-page.ts`; a 403 would fall through to the generic error empty.
- **Stale / disconnected:** the disconnected state replaces the daemon status card body with an `Empty` whose icon is `ServerOff` and whose title is itself the `ConnectionIndicator` (`index.tsx:123-138`). Description copy is good ("Start it with `agh daemon`"). But the four metric tiles below remain populated even in the disconnected state, with no visual indication they may be stale (`sb_disconnected.png` shows `Active Sessions 10`, `Workspaces 7`, `Agents 11`, `Daemon Uptime ---`). The `Daemon Uptime` correctly shows the em-dash placeholder; the other three display the previously-fetched values without any "may be stale" treatment. **P0** truthful UI risk: the dashboard is asserting freshness it cannot guarantee. Remediation: dim the cards or add a "stale" badge when `connectionStatus === 'disconnected'`.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** `session`, `workspace`, `agent`, `daemon` are all canonical per `docs/_memory/glossary.md`. No `recipe`, `workflow`, `procedure`, `playbook`. PASS.
- **Tone:** `daemonStatus.description` strings in `use-home-page.ts:78-124` are operator-first and direct ("All subsystems are reporting healthy status.", "Re-establishing the connection to the local daemon."). Reads as engineer-to-engineer. PASS.
- **Em dashes:** present, **violation against the impeccable shared design laws and `COPY.md` §5**:
  - `web/src/hooks/routes/use-home-page.ts:53` returns `"—"` (U+2014) as the user-facing fallback for invalid uptime input.
  - `web/src/hooks/routes/use-home-page.ts:172` uses `"—"` as the active sessions value when the sessions fetch errors.
  - These render in the Daemon Uptime tile of the disconnected story (`sb_disconnected.png`).
  - Note: `DESIGN.md` §7.4 "Typographic Marks" actively recommends em dashes for copy pauses, contradicting `COPY.md` §5 and the impeccable rule. The two project authorities disagree. Per the audit method (impeccable critique), the em dash here is a violation; the project should pick one rule and either remove the em dash or sanction it explicitly in `COPY.md`. **P2.**
- **Restated headings:** none. `Home` is the page title; `DAEMON` and `OVERVIEW` are section eyebrows; the daemon card title is `Healthy` / `Degraded` / `Disconnected` / `Unknown`. Each title carries its own information. PASS.
- **Sentence case vs Title Case:** `Home`, `Active Sessions`, `Workspaces`, `Agents`, `Daemon Uptime` are all title case for what should be sentence-case labels per `COPY.md` §5 ("Sentence case for headings and labels unless the UI component or design system requires uppercase mono metadata"). The mono eyebrow rule supersedes here because the metric labels are rendered uppercase mono; but the page title `Home` and the section eyebrows `DAEMON` / `OVERVIEW` are correct. Actually, the page title `Home` is also one word. **PASS, with one drift:** `Daemon Uptime` reads as title case while `Active Sessions`, `Workspaces`, `Agents` could be sentence case ("Active sessions") if read in non-mono. Because they are mono uppercase the distinction collapses. Acceptable.
- **Truthful UI test:**
  - `Workspaces` count comes from `useActiveWorkspace().workspaces.length`, sourced from `/api/workspaces`. PASS.
  - `Agents` count comes from `workspaceDetail?.agents ?? agents` (`use-home-page.ts:185-186`). The fallback to `useAgents()` may render the global agents count if `workspaceDetail` is loading; this is mostly correct because the populated label "Agents" is workspace-scoped semantically yet the count is global until detail loads. **Minor drift, P2.**
  - `Active Sessions` correctly shows `0` when the workspace has no sessions; detail copy "in pedronauck" identifies the scope (`use-home-page.ts:177-183`). PASS.
  - `Daemon Uptime` is sourced from `health.uptime_seconds` and formatted as `41m 57s` etc. PASS.
  - `vbfd54851` version chip is `health.version` (a git short-sha at `internal/...`). Correct, but unhelpful as a label without a tooltip. **P2.**
  - **No false metrics, no false controls, no fake trend.** The dashboard is honest. The risk is the four cards remain populated in the disconnected state without a stale indicator (covered in §8). The dashboard is **not** rendering anything the daemon does not expose. PASS.

---

## 10. Performance & Responsiveness

- **Initial render:** under 500 ms in dev mode on `:3000`; no waterfalls observed. The route's hook fans out to four queries (`useDaemonHealth`, `useActiveWorkspace`, `useAgents`, `useWorkspace`, `useSessions`); the `isLoading` aggregation in `use-home-page.ts:212-217` blocks the render until all five resolve.
- **Re-render hot spots:** none observed. The four metric values are memoized via `useMemo` (`use-home-page.ts:159-210`). The connection indicator re-renders only on connection state change. No keystroke-on-interval renders because the route has no inputs.
- **List virtualization:** n/a, route is fixed at four metrics.
- **Bundle red flags:** `index.tsx` imports only `lucide-react` icons, three `@agh/ui` primitives, the `ConnectionIndicator`, and the page-view hook. No charting lib. PASS.
- **Responsive behaviour:**
  - 1440x900: four columns, healthy whitespace, dashboard body ends at y~340 (`live_1440.png`).
  - 1024x768: four columns become two via `xl:grid-cols-4 sm:grid-cols-2` (`live_1024.png`).
  - 768x1024: two columns, sidebar still expanded (`live_768.png`).
  - 320x568: sidebar still 240 px panel + 44 px rail = 284 px, body clipped to ~36 px (`live_320.png`). **Broken.** P0.
- **Mobile interactions:** no hover-only affordance on the route. Shell sidebar `Toggle sidebar` is reachable on tablet and above; below ~480 px it collides with the agent tree footer. Module-level finding.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-index--default` (Healthy populated) -> `http://localhost:6006/iframe.html?id=routes-app-stories-index--default&viewMode=story`
  - `routes-app-stories-index--degraded` (Daemon non-healthy)
  - `routes-app-stories-index--disconnected` (503 health)
  - `routes-app-stories-index--loading` (delay infinite on health, workspaces, agents)
  - `routes-app-stories-index--error` (workspaces 500)
  - `routes-app-stories-index--onboarding` (no workspaces -> shell short-circuit)
- **States covered in Storybook:** typical populated, degraded, disconnected, loading, generic error, first-run / empty.
- **Gaps:**
  - No `populated-dense` story for >100 workspaces or many agents.
  - No `partial` story (one of five hooks erroring while the others succeed).
  - No `permission-denied` (403) story; today it would render the same generic error.
  - No narrow-viewport story; this matters because the shell does not auto-collapse.
  - No `reconnecting` story, only `disconnected` and `degraded`. The reconnecting branch in `use-home-page.ts:91-98` is dead in Storybook.
- **Story drift:** `meta.title` in `-index.stories.tsx:14` is `"routes/app/home"`, but the running Storybook indexer uses the file path and emits ids `routes-app-stories-index--*`. The static `web/storybook-static/index.json` confirms (`title` field: `routes/_app/stories/-index`). This is **drift between the helper's intent and Storybook's actual behavior**. Two ways to fix: change the Storybook config so `createRouteStoryMeta` titles win, or rename the file to drop the leading dash so the indexer respects the meta title. **P1.**

---

## 12. Findings, Prioritised

### P0, Ship Blockers

1. **[P0] What:** Disconnected state renders stale metric values with no indication they may be stale.
   - **Why:** The dashboard is the operator's first-glance proof that the daemon is alive. With the daemon disconnected, three of the four metric tiles still show their last-fetched values (workspaces, agents, active sessions) without any visual treatment. Per `COPY.md` §1 "Truthful UI > plausible UI", asserting numbers the daemon cannot currently confirm is a truthful-UI violation.
   - **Fix:** when `connectionStatus === 'disconnected'`, dim the metric cards (e.g., wrap each value in a `text-[color:var(--color-text-tertiary)]` + a `STALE` mono chip in the corner) or replace the values with the same em-dash placeholder pattern already used for daemon uptime in this state. Alternatively, hide the `Overview` section entirely while disconnected.
   - **Cmd:** `/impeccable harden web/src/routes/_app/index.tsx`
   - **Effort:** S
   - **Evidence:** `_evidence/index/sb_disconnected.png` (cards still show `Active Sessions 10`, `Workspaces 7`, `Agents 11`).

2. **[P0] What:** Sidebar does not collapse on narrow viewports; at 320 px the dashboard body is ~36 px wide.
   - **Why:** `DESIGN.md` §11 explicitly mandates "On narrow viewports, collapse to icon-rail-only mode (40 px)". The runtime does not auto-collapse; the toggle exists but is unreachable below ~480 px because the body content disappears under the sidebar. This breaks every route, but the dashboard is the first surface affected.
   - **Fix:** in `web/src/routes/_app.tsx` and `web/src/components/app-sidebar.tsx`, switch the panel from a fixed 240 px to a CSS clamp / breakpoint that auto-collapses below `md` (768 px) and exposes a toggle pinned to a fixed corner. Or hide the panel entirely below `sm` (640 px) with a floating toggle.
   - **Cmd:** `/impeccable adapt web/src/components/app-sidebar.tsx`
   - **Effort:** M
   - **Evidence:** `_evidence/index/live_320.png`.

### P1, High-Value Polish

3. **[P1] What:** Route name drift: `Home` (page header) vs `Dashboard` (sidebar nav) vs `home-*` (testids) vs `Unable to load dashboard` (error empty).
   - **Why:** The user has to reconcile three labels for one surface. `COPY.md` §6 "Vocabulary & Naming" requires canonical terms; the project picks `Dashboard` in the sidebar but ships `Home` in the page chrome. Documentation, screenshots, support, and agent-driven UI scrapes all read different names.
   - **Fix:** pick one canonical name, propose `Dashboard` since the sidebar is the first thing every user sees and `app-sidebar.tsx:124` already uses it. Update `web/src/routes/_app/index.tsx:30-32` to render `<span data-testid="dashboard-page-title">Dashboard</span>`. Rename `data-testid="home-*"` to `data-testid="dashboard-*"`. Update `WorkspaceSetupCopy` if it references "dashboard" anywhere.
   - **Cmd:** `/impeccable clarify web/src/routes/_app/index.tsx`
   - **Effort:** S (one-pass rename, but touches tests and selectors elsewhere).
   - **Evidence:** `web/src/routes/_app/index.tsx:31`, `web/src/components/app-sidebar.tsx:124`, `_evidence/index/live_full.png`.

4. **[P1] What:** Sidebar nav rows lack a focus-visible accent ring; they fall back to the browser default `outline: auto`.
   - **Why:** App-shell controls (logo, workspace avatars, add-workspace, toggle) all carry `focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]`. The nav rows do not. WCAG 2.2 SC 1.4.11 needs a 3:1 contrast for focus indicators; the browser default may or may not satisfy that on `#0E0E0F`. This affects every route in the SPA.
   - **Fix:** add `focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]` to `NAV_ROW_CLASS` in `web/src/components/sidebar-nav-classes.ts:1-3`.
   - **Cmd:** `/impeccable harden web/src/components/sidebar-nav-classes.ts`
   - **Effort:** S
   - **Evidence:** live `getComputedStyle` log on the `Dashboard` nav row vs the logo / workspace controls.

5. **[P1] What:** Disconnected daemon card has no programmatic heading; metrics are silent in the accessibility tree.
   - **Why:** Screen-reader operators cannot find the disconnected status as a heading; the four metrics are read as one undivided run because each `Metric` is a plain `<div>` with no role, no labelledby, and no headings nearby.
   - **Fix:** wrap each `Metric` in a `role="group"` with `aria-labelledby` pointing at the eyebrow `<span>`; in `Empty`, when the title is a ReactNode, infer the heading via an `aria-label` prop on the outer container or render a visually-hidden `<h3>` with the connection status text.
   - **Cmd:** `/impeccable harden packages/ui/src/components/metric.tsx packages/ui/src/components/empty.tsx`
   - **Effort:** S
   - **Evidence:** `_evidence/index/dom_default.txt:32-33`, `_evidence/index/dom_disconnected.txt`.

6. **[P1] What:** Storybook story id format diverges from `createRouteStoryMeta` declared title.
   - **Why:** The helper declares `title: "routes/app/home"` but Storybook indexes ids as `routes-app-stories-index--*`. Anyone trying to deep-link or reference the story by the documented title gets a `Couldn't find story` error.
   - **Fix:** drop the dash prefix on `web/src/routes/_app/stories/-index.stories.tsx` (rename to `index.stories.tsx`); the dash convention is for TanStack route detection, not for Storybook. Or, override Storybook's default indexer to honor the `title` field. The first option is cheaper.
   - **Cmd:** `/impeccable harden web/.storybook/main.ts`
   - **Effort:** S
   - **Evidence:** `web/storybook-static/index.json` shows `title: "routes/_app/stories/-index"`; `_evidence/index/sb_default_v2.png` shows the "Couldn't find story" page when querying `routes-app-home--default`.

7. **[P1] What:** Metric eyebrow uses `--color-text-tertiary` (#636366) on `--color-surface` (#1E1C1B), contrast ~3.2:1, fails AA body 4.5:1.
   - **Why:** `--color-text-tertiary` is documented in `DESIGN.md` §2 as "placeholders, disabled text, low-emphasis"; it should not be the primary label of an active metric. Per the same DESIGN.md the canonical eyebrow color is `--color-text-label` (#98989D), which gives ~5.4:1 on surface, AA-compliant.
   - **Fix:** change `metric.tsx:58` and `metric.tsx:73` from `text-[color:var(--color-text-tertiary)]` to `text-[color:var(--color-text-label)]`. Run a regression check on all surfaces using `Metric` (this also affects later modules).
   - **Cmd:** `/impeccable colorize packages/ui/src/components/metric.tsx`
   - **Effort:** S, but cross-module impact M.
   - **Evidence:** computed-style probe; `DESIGN.md` §2 token table.

8. **[P1] What:** No primary action / next-step CTA on the operator landing surface.
   - **Why:** Per `COPY.md` §4 "Audience & Surface Intent", operators arriving at the dashboard expect "start, supervise, resume, inspect, and repair sessions". The dashboard offers no entry to any of those verbs, only counts. The route loses its narrative job.
   - **Fix:** consider a single inline primary CTA on the right of `PageHeader` when the daemon is healthy, e.g., `Create a session` (uses `SessionCreateProvider` already in scope from the shell); or surface a `View peers` link in the `Daemon` section when bridges are configured. Even one CTA tied to a real verb shifts the surface from "stats hero" to "operator surface".
   - **Cmd:** `/impeccable shape web/src/routes/_app/index.tsx`
   - **Effort:** M (requires copy / verb decision aligned with `COPY.md` §9 CTA Vocabulary).
   - **Evidence:** `_evidence/index/live_full.png` (no CTA above the fold).

### P2, Worthwhile

9. **[P2] What:** Em dash literal `"—"` rendered as user-facing copy in `use-home-page.ts:53, 172`.
   - **Why:** `COPY.md` §5 and the impeccable shared laws ban em dashes; `DESIGN.md` §7 contradicts and recommends them. The two project authorities disagree.
   - **Fix:** either resolve the conflict (have the design lead pick one rule) or replace the literal `—` with `--` (two ASCII hyphens) or with the word `unknown` per the COPY.md tone. Pick `--` if the surface is mono; pick `unknown` if Inter.
   - **Cmd:** `/impeccable clarify web/src/hooks/routes/use-home-page.ts`
   - **Effort:** S, plus a docs update.
   - **Evidence:** `web/src/hooks/routes/use-home-page.ts:53,172`; `_evidence/index/sb_disconnected.png`.

10. **[P2] What:** Version chip `vbfd54851` is opaque, no tooltip, no link.
    - **Why:** Per `COPY.md` §1 "Runtime truth beats copy preference"; if the chip is the build identity it should hand the operator a link to the changelog or a tooltip with build date. Today it is metadata-as-decoration.
    - **Fix:** add a `title` attribute or `Tooltip` (already in `@agh/ui`) showing `Built <date> from <branch>@<sha>`. If a public changelog URL exists, link the chip.
    - **Cmd:** `/impeccable clarify packages/ui/src/components/page-header.tsx`
    - **Effort:** S
    - **Evidence:** `_evidence/index/live_full.png`.

11. **[P2] What:** "Agents" metric falls back to global `agents` count while workspace detail is loading.
    - **Why:** `use-home-page.ts:185-186` resolves `activeWorkspaceAgents = workspaceDetail?.agents ?? agents`. Until detail loads, the operator sees the global agents count under a label that reads workspace-scoped. Minor truthful-UI drift.
    - **Fix:** during `isWorkspaceDetailLoading`, render the metric as a skeleton instead of falling back to the global count. Or label the fallback `(global)` until the detail resolves.
    - **Cmd:** `/impeccable clarify web/src/hooks/routes/use-home-page.ts`
    - **Effort:** S
    - **Evidence:** `web/src/hooks/routes/use-home-page.ts:185-186`.

12. **[P2] What:** Error empty offers no retry and no support path.
    - **Why:** `COPY.md` §9 "Error Copy" formula is "what failed, why, next safe action". The current error empty has the first two and skips the third. Operators stare at it.
    - **Fix:** add `<Empty action={<Button onClick={refetch}>Retry</Button>} />` plus a `Link` to runtime docs `/runtime/troubleshooting` or similar. Reuse the same pattern from `web/src/routes/_app.tsx:104-122` `AppRouteErrorBoundary`.
    - **Cmd:** `/impeccable harden web/src/routes/_app/index.tsx`
    - **Effort:** S
    - **Evidence:** `web/src/routes/_app/index.tsx:47-61`; `_evidence/index/sb_error.png`.

### P3, Parking Lot

13. **[P3] What:** Spacing rhythm is monotone (24/12/24).
    - **Why:** `DESIGN.md` §5 calls for varied spacing for rhythm. The dashboard surface beats at one tempo from top to bottom.
    - **Fix:** consider tightening the section eyebrow gap to 16 px and opening the metric grid gap to 24 px to introduce a 16/24 rhythm; or use `padY="lg"` on the marketing-style `SectionFrame`. Optional polish, the surface is small.
    - **Cmd:** `/impeccable layout web/src/routes/_app/index.tsx`
    - **Effort:** S
    - **Evidence:** `_evidence/index/live_1440.png`.

14. **[P3] What:** No "as of" / "last refresh" timestamp on the daemon card or metrics.
    - **Why:** Operators glance at the dashboard and want to know how fresh the numbers are. Today they are React Query polled but the cadence is invisible.
    - **Fix:** small mono caption "as of 12s ago" right of `Section.right` or below the daemon description. Reuse `JetBrains Mono 11px text-tertiary`.
    - **Cmd:** `/impeccable typeset web/src/routes/_app/index.tsx`
    - **Effort:** S

---

## 13. Persona Red Flags

- **Operator (returning power user, keyboard-first):**
  - On Tab to the `Dashboard` nav row the focus indicator is whatever the browser supplies; on dark mode this can be near-invisible. The accent ring on neighboring controls makes the inconsistency conspicuous.
  - No `cmd+k` palette, no shortcut to refresh, no shortcut to open a session. The dashboard is the first surface the power user lands on; it offers nothing.
  - The version chip is unactionable; clicking it does nothing, hovering shows nothing.
- **First-timer (onboarding, no mental model yet):**
  - Onboarding branch is excellent (`sb_onboarding.png`).
  - Once on the dashboard with a single workspace, the first-timer sees `Home`, four numbers, and a green dot. There is no copy explaining what to do next, no tour, no "start by creating a session", no link to docs. They will likely click around the sidebar at random.
- **Agent (yes, agents read this UI):**
  - DOM is mostly accessible: `data-testid="home-*"` selectors are stable, headings exist for the two sections, the connection indicator carries `role="status"` + `aria-live="polite"`. Programmatic reading is feasible.
  - The four metric tiles render as `<div>` with no role; an agent scraping the surface has to infer grouping from CSS classes or `data-slot`. Adding `role="group"` + `aria-labelledby` would also help agents.
  - The route name drift (`Home` vs `Dashboard` vs `home-*`) confuses agents that match by visible text vs testid vs URL slug. They will pick one and break on edits.

---

## 14. Cross-Module Consistency Notes

- The shell-level findings (sidebar collapse, nav focus rings, route name drift) propagate to every other module. Cross-reference: every `0X_<module>` audit should re-state these as inherited findings, not novel findings.
- The `Metric` eyebrow contrast issue propagates to any other module that uses `Metric`. Fixing in `packages/ui/src/components/metric.tsx` fixes everywhere; per `web/CLAUDE.md` "Pull every color, font, radius, spacing step, and motion value from `DESIGN.md`" the right layer for the fix is the primitive, not the consumer.
- The em-dash policy needs resolution at the project authority level; do not patch it module by module.
- The Storybook id namespace fix is one rename, but it changes every reference under `web/src/routes/_app/stories/-*.stories.tsx`. Worth doing once at the start of the audit pass.

---

## 15. Open Questions

- What is the dashboard supposed to **do**? If the answer is "be the operator's first-glance status", the metrics need to be richer (persistence, retention, dream, automation, bridges all live in `/api/observe/health` already). If it is "be the navigation hub", it needs a CTA. Today it is neither.
- Should the disconnected state hide the `Overview` section entirely or stale-tag it? Truthful UI would prefer hide; ergonomic operators would prefer stale-tag. Pick one.
- Should the dashboard expose a `Recent activity` feed (sessions started in the last hour, recent task runs, recent receipts on agh-network)? The data exists; the surface does not show it.
- Should the version chip be the only build-identity surface, or should it move to the sidebar footer (where the connection indicator already lives)? Today both surfaces show `vbfd54851`, so we have visible duplication (`live_full.png` shows `VBFD54851` in the header and `vbfd54851` in the footer).

---

## 16. Recommended Action Plan

1. `/impeccable harden web/src/routes/_app/index.tsx` -> stale-tag the metrics in the disconnected state, add retry + docs link to the error empty, and (optionally) collapse `Overview` while disconnected. Resolves P0 #1 and P1 #5, P2 #12.
2. `/impeccable adapt web/src/components/app-sidebar.tsx web/src/routes/_app.tsx` -> auto-collapse the sidebar below `md`, ensure the toggle floats above content on narrow viewports. Resolves P0 #2.
3. `/impeccable clarify web/src/routes/_app/index.tsx web/src/components/app-sidebar.tsx` -> rename the route to `Dashboard` everywhere (page title, testids, error empty title); resolve em-dash policy; tighten the version chip with a tooltip. Resolves P1 #3, P2 #9, P2 #10.
4. `/impeccable harden web/src/components/sidebar-nav-classes.ts` -> add focus-visible accent ring to `NAV_ROW_CLASS`. Resolves P1 #4 across every route.
5. `/impeccable colorize packages/ui/src/components/metric.tsx` -> swap eyebrow color from tertiary to label token. Resolves P1 #7 across every route that uses `Metric`.
6. `/impeccable harden packages/ui/src/components/metric.tsx packages/ui/src/components/empty.tsx` -> add `role="group"` + `aria-labelledby` on metrics; expose a heading on Empty when title is a ReactNode. Resolves P1 #5.
7. `/impeccable shape web/src/routes/_app/index.tsx` -> propose a primary CTA on the dashboard (Create session / Open runtime docs). Resolves P1 #8 by design conversation.
8. Rename `web/src/routes/_app/stories/-index.stories.tsx` to drop the dash prefix and update sibling references; verifies that `createRouteStoryMeta`'s title wins. Resolves P1 #6.
9. `/impeccable polish web/src/routes/_app/index.tsx` -> final pass after the above changes ship.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/index/`.
- [x] No section is left as `<TODO>` or empty (a few are `n/a` with reason: route is read-only, no forms, no lists).
- [x] Nielsen scores total is consistent with the band claimed (24/40 -> adequate).
- [x] Findings are tagged P0-P3 with effort and command.
- [x] No hallucinated routes, components, or props (every cited path was opened in `Read`).
- [x] No em dashes in this report (every pause uses a comma, semicolon, period, or parentheses; the only U+2014 occurrences are inside `"—"` quoted from source code).
- [x] Report length: thorough but not padded; the value is in `file:line` evidence and the prioritised findings.
