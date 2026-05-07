# UI/UX Audit Module Overview, `01_dashboard`

> **Status:** draft
> **Owner subagent:** `dashboard-auditor`
> **Date:** 2026-05-06
> **Module:** `01_dashboard`
> **Routes covered:** `/` (`web/src/routes/_app/index.tsx`)
> **Surfaces also touched:** the app shell that frames every `_app` route (sidebar + header chrome).
> **Live URLs probed:** `http://localhost:3000/` (the AGH Vite dev server, started by this audit), `http://localhost:6006/` (Storybook).

---

## Why the dashboard sets the tone

`/` is the only route in this module, but it is the first surface a returning operator sees, the first surface a new operator hits after onboarding, and the most aggressively-cached preview shown to anyone running `agh-web` at all. Every signal it carries (truthful or not) becomes the reader's mental model for the rest of the app. That makes the module's module-level concerns about the shell and the daemon-truth contract more important than the route's own copy or layout choices.

This overview synthesizes findings that span the route file, the `@agh/ui` primitives (`Metric`, `PageHeader`, `Section`, `Empty`, `Pill.Dot`), the `ConnectionIndicator`, the app shell (`web/src/routes/_app.tsx`, `web/src/components/app-sidebar.tsx`), and the daemon contract at `/api/observe/health`. Per-route detail and Nielsen scoring live in `01_analysis_index.md`.

---

## Probe environment, what was actually live

Per the audit `_README.md` the SPA is supposed to be reachable on `http://localhost:5173`. That port served a different project ("Weather Dashboard", `Vite + React`, see `_evidence/index/live_full.png` first capture before the redirect). The AGH web dev server is wired to `:3000` (`web/package.json:7`, `"dev:raw": "vite --port 3000"`), so this auditor:

1. Started `bun run dev:raw` in `web/`, served on `:3000`.
2. Used Storybook on `:6006` for the populated/degraded/disconnected/loading/error/onboarding states (the daemon currently has only one workspace and one agent).
3. Hit the daemon directly on `:2123/api/observe/health` to confirm what data the dashboard is allowed to render.

This mismatch between the README's target and the codebase's `dev:raw` port is itself a small finding for the audit task harness, not a product finding.

The live SPA loads cleanly at 1440x900 (`_evidence/index/live_1440.png`); no console errors, no page errors (`agent-browser console` and `errors` returned empty).

---

## Cross-cutting themes the dashboard exposes

These are observations that recur once you have the dashboard, the shell, and the daemon contract side by side. They will resurface in every other module's audit, so they are flagged here once.

### 1. Shell sets the navigation grammar; the dashboard is the only surface that obeys it cleanly

The sidebar (`web/src/components/app-sidebar.tsx:120-178`) groups items as `Dashboard` (top-level), `Agents` (tree), `Operate`, `Catalog`, `System`. The dashboard route is the canonical example of "left-aligned page header, mono eyebrow section labels, surface card with hairline divider", which `DESIGN.md` §4 describes verbatim. Any sibling route that diverges from this rhythm (extra container chrome, different header height, glassmorphism, non-mono eyebrows) is wrong by the dashboard's example. The audit should treat this route as the per-module reference for "what the rest of `_app` is supposed to look like".

### 2. Truthful UI is mostly upheld, but the surface is too small to count as proof

The dashboard renders four metrics (`active sessions`, `workspaces`, `agents`, `daemon uptime`) plus a daemon status card and version chip (`web/src/routes/_app/index.tsx:13-18, 140-167`, `web/src/hooks/routes/use-home-page.ts:190-210`). All six are sourced from real daemon endpoints (`/api/observe/health`, `/api/workspaces`, `/api/agents`, `/api/sessions`). No fake metric, no decorative sparkline, no "trend" delta, no synthetic chart. That is correct posture for a greenfield alpha. **But:** the daemon exposes far more (persistence status, retention sweep status, agent_probes, bridges totals, memory/dream status, automation enabled) that the dashboard does not surface. Per `COPY.md` §1 "Runtime truth beats copy preference", and per the product premise "highly observable", the dashboard is currently a near-empty hero, not an operator surface. This is a strategic question more than a defect; it lives as a P1 finding in the per-route report.

### 3. Naming drift: `Dashboard` vs `Home`

Sidebar nav label is `Dashboard` (`app-sidebar.tsx:124`, `DASHBOARD_NAV_ITEM`). `PageHeader` title on the same route is `Home` (`index.tsx:30-32`). Test ids and shell anchors use `home-*` (`index.tsx:31, 37, 50, 64, 95, 99, 107, 112`). The empty-state error title says "Unable to load dashboard" (`index.tsx:57`). Three different names for one route in one screenshot. The reader's mental model gets to pick. This is a P1 cross-cutting consistency issue: the rest of the audit will want to see whether other modules have a single canonical name; the dashboard module sets the precedent that they do not.

### 4. Focus-ring grammar is split between the shell and the page

App-shell controls (logo, workspace avatars, add-workspace, sidebar toggle) all use an explicit `focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]` (`app-sidebar.tsx:48, 66, 80`). The nav rows themselves (`Dashboard`, `Network`, `Tasks`, ...) use `NAV_ROW_CLASS` from `web/src/components/sidebar-nav-classes.ts`, which carries no `focus-visible` rule. The result, captured in `_evidence/index/live_dashboard_nav_focus.png` and `eval`-ed live: nav rows fall back to the browser default `outline: auto` while nearby controls use the project's accent ring. This is one accessibility regression that will appear in every other module since they all share the same shell.

### 5. Responsive, the shell does not yield

At 320 px (`_evidence/index/live_320.png`) the expanded sidebar (rail 44 px + panel 240 px = 284 px) consumes 89% of the viewport; the body content area is roughly 36 px wide and clipped. The shell never collapses automatically; the `Toggle sidebar` control exists but is only reachable on a wider viewport. `DESIGN.md` §11 explicitly states "On narrow viewports, collapse to icon-rail-only mode (40 px)" yet the implementation does not auto-collapse. This is a shell defect that will hit every route in the SPA.

### 6. Storybook id format drift

The home story file at `web/src/routes/_app/stories/-index.stories.tsx` declares `title: "routes/app/home"` via `createRouteStoryMeta`, but the running Storybook indexer ignores the explicit title and synthesises ids of the form `routes-app-stories-index--<state>` (per `web/storybook-static/index.json`). The static URL hint in the audit prompt and most internal references using `routes-app-home--*` will 404 (proven by `_evidence/index/sb_default_v2.png`). This is a Storybook config / route-story-meta issue and should be fixed across modules so a `cy-create-tasks` writer can deep-link to states without trial and error.

---

## Storybook coverage at the module level

The dashboard module ships six stories that map cleanly to the route's required states:

| State | Story id | Coverage |
|---|---|---|
| Populated (typical) | `routes-app-stories-index--default` | strong |
| Degraded | `routes-app-stories-index--degraded` | strong |
| Disconnected | `routes-app-stories-index--disconnected` | strong |
| Loading skeleton | `routes-app-stories-index--loading` | strong |
| Error (workspaces 500) | `routes-app-stories-index--error` | strong, narrow scope |
| First-run / onboarding | `routes-app-stories-index--onboarding` | strong, but the onboarding card is rendered by `_app.tsx`, not by `index.tsx`, so this story tests the shell branch, not the dashboard |

What is missing at the module level:

- No `populated-dense` story (100+ workspaces or sessions). Once the daemon supports that scale, the metric grid becomes the limiting card. There is no story for that.
- No `partial` story where one of the five hooks (`useDaemonHealth`, `useActiveWorkspace`, `useAgents`, `useWorkspace`, `useSessions`) succeeds while another silently fails. Today, any one error short-circuits to the global error empty state (`index.tsx:47-61`). That is a strong choice but it is not visible in Storybook.
- No `narrow-viewport` story. Because the shell does not auto-collapse, this is the only way to exercise the route's responsive layout outside live probing.

---

## How this module sets up the rest of the audit

When the next ten modules audit `_app/network`, `_app/tasks`, `_app/jobs`, `_app/triggers`, `_app/knowledge`, `_app/skills`, `_app/bridges`, `_app/sandbox`, `_app/settings`, the auditor should expect to see the same flat-depth, mono-eyebrow grammar, the same `PageHeader` chrome, and the same `Empty` primitive for failure surfaces. Any divergence is presumptively wrong. The dashboard's clean surface is also a soft warning: the rest of the routes are supposed to do real work; if their per-page detail is no denser than the dashboard's four metrics, they are scaffolded, not shipped, and the audit needs to flag it.

---

## Module-level top three

1. **P0 (shell):** the sidebar does not auto-collapse on narrow viewports and the toggle is unreachable below ~480 px. Documented in `DESIGN.md` §11 as required behavior; absent in code (`web/src/components/app-sidebar.tsx`, `web/src/routes/_app.tsx`). Evidence: `_evidence/index/live_320.png`.
2. **P0 (truthful UI tightness):** the dashboard renders four metrics plus a status card on a route that the daemon could fill with persistence, retention, dream, automation, bridges, agent_probes, and failures evidence. The current state is honest but is not yet an operator surface. Evidence: full daemon health response in `_evidence/index/sb_default.png` versus the Storybook `Default` payload that already includes that data via `daemonHealthFixture` (`web/src/systems/daemon/mocks/fixtures.ts`).
3. **P1 (cross-shell consistency):** route name drift `Dashboard` vs `Home` vs `home-*` testids; nav-row focus rings missing while neighboring controls have them. Both are shell defects that will recur in every module's audit.

Per-route Nielsen scoring, anti-pattern verdict, and prioritised findings are in `01_analysis_index.md`.
