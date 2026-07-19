# AGH OS v2 — shell-first delivery plan

**Premise.** v1 (`os/agh-os.html`) proved the OS metaphor but redesigned every surface, which means rewriting the whole frontend. v2 keeps the approved shell disposition (menu bar · bottom dock · floating windows · spaces) and hosts the **existing routes unchanged** inside the windows. Net-new work is the shell only.

## The one structural trick

The window header **is** the route's existing topbar. Production routes already render a 48px 3-zone topbar (breadcrumb leading · center · trailing actions). v2 injects the three OS window controls into the leading zone, before the breadcrumb. No second header, no double chrome, no route rewrite:

```
┌─[● ● ●]  agh / Tasks                    [New task]─┐   ← existing topbar + controls
│  (route outlet renders here, unchanged)            │
└────────────────────────────────────────────────────┘
```

## Window → existing code map

The dock mirrors `app-sidebar.tsx` — same 13 routes, same order, with dock separators where the sidebar breaks its nav groups (operate · catalog · system).

| Window | Existing source (web/) | Change needed |
| --- | --- | --- |
| Dashboard | home route (`/`) — KPI strip, runcards, outcome bars | none |
| Sessions | sidebar-sessions-02 view (recent + Show-all grouped by agent, collapse contract) | renders as the left rounded rail (dock-style shell), not a window; dock glides right while it's open |
| Session (live) | session route — transcript, tool calls, composer | none |
| Agents | `/agents` (agents-list) | none |
| Network | `/network` | none |
| Tasks | `/tasks` — filters + Rows\|Cards toggle | none |
| Loops | `/loops` | none |
| Jobs | `/jobs` | none |
| Triggers | `/triggers` | none |
| Marketplace | `/marketplace` — kind tabs, Marketplace\|Installed, cards | none |
| Bridges | `/bridges` | none |
| Knowledge | `/knowledge` | none |
| Sandbox | `/sandbox` — Rows\|Cards toggle | none |
| Vault | `/vault` | none |
| Settings | `/settings` (nav + panes) | none |

## Net-new inventory (the entire cost of phase 1)

1. **MenuBar** — workspace command-select (existing component, compact), Session/View/Help menus, approvals bell, ⌘K, and a Settings cog (AGH has no user concept, so no avatar; Settings lives here instead of the dock). The bell reuses the dashboard "Needs you" rows (`.att__row`).
1b. **SessionsRail** — the sidebar-sessions-02 view (recent rows with agent·state sub, search, Show all sessions → agent groups with the data-collapsed contract) hosted in a floating rounded panel with the dock's glass/radius, left margin, toggled by the first dock icon. Opening it collapses the dock's right flex spacer so the dock glides to the right edge.
2. **Dock** — app buttons + running dots + badges + magnification + New session. Minimizing folds a window into its own dock icon (dimmed glyph + hollow indicator; click restores) — no tray, so minimized count never affects layout.
3. **WindowFrame** — drag, focus z-order, minimize/zoom/close, resize, rect persistence. Wraps the route outlet; injects controls into the route topbar's leading zone.
4. **Desktop** — wallpaper layer. (Custom desktop widgets are explicitly out of scope for the first delivery.)
5. **Spaces** — workspace switch already exists; v2 adds per-workspace window-rect persistence and the overview overlay.

Deleted: app sidebar. Everything else ships as-is.

## Phasing

- **P1 (first delivery):** MenuBar + Dock + WindowFrame over existing routes. Single workspace behavior identical to today; sidebar retired.
- **P2:** Spaces (per-workspace window state), ⌘K palette (can reuse the existing command registry).
- **P3:** live-session niceties (genie minimize, session-per-window multiplexing), desktop widgets if ever wanted.

## Tokens

`os-v2.css` copies the production token block verbatim (dashboard.html ← agent-detail.html ← `packages/ui/src/tokens.css`) and adds shell-only variables (`--shell-glass`, dock/window radii under the OS radius waiver). Signal vocabulary follows production: running = pulsing accent dot, needs-you = warning, failed = danger, ok = success.
