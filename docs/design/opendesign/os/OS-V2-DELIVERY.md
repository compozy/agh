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
| Sessions | sidebar-sessions-02 view (recent + Show-all grouped by agent, collapse contract) | none — opens as a normal floating window like every other dock app |
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
2. **Dock** — app buttons + running dots + badges + magnification + New session. Sessions is the first dock icon and opens as a normal window (same WindowFrame as Dashboard etc.). Minimizing folds a window into its own dock icon (dimmed glyph + hollow indicator; click restores) — no tray, so minimized count never affects layout. The dock stays centered; it does not slide when Sessions opens.
3. **WindowFrame** — drag, focus z-order, minimize/zoom/close, resize, rect persistence. Wraps the route outlet; injects controls into the route topbar's leading zone.
4. **Desktop** — wallpaper layer. (Custom desktop widgets are explicitly out of scope for the first delivery.)
5. **Spaces** — workspace switch already exists; v2 adds per-workspace window-rect persistence and the overview overlay.

Deleted: app sidebar. Everything else ships as-is.

## Phasing

- **P1 (first delivery):** MenuBar + Dock + WindowFrame over existing routes. Single workspace behavior identical to today; sidebar retired.
- **P2:** Spaces (per-workspace window state), ⌘K palette (can reuse the existing command registry).
- **P3:** live-session niceties (genie minimize, session-per-window multiplexing), desktop widgets if ever wanted.

## Mobile mode (<960px)

Below 960px the shell reflows instead of blocking (the old "desktop concept" viewport guard was deleted):

- **Windows** become stacked fullscreen surfaces. The media query overrides the WM's inline rects (`!important`), z-order decides which one is visible. Drag, resize, zoom and rect persistence are disabled behind `isMobile()` guards, so the saved desktop layout survives a phone visit untouched and reappears intact above 960px.
- **Dock** renders as a bottom tab bar: horizontally scrollable icon strip (full `app-sidebar.tsx` parity kept — no route omitted), pinned New-session action, safe-area padding. Tap = switch to app, never minimize; magnification, tooltips and separators are off.
- **Sessions** is a normal window: on mobile it stacks fullscreen like every other app (no special sheet).
- **Menu bar** keeps logo · workspace · bell · ⌘K · cog; the Session/View/Help menus hide (their actions stay reachable via ⌘K).

## Tokens

`os-v2.css` copies the production token block verbatim (dashboard.html ← agent-detail.html ← `packages/ui/src/tokens.css`) and adds shell-only variables (`--shell-glass`, dock/window radii under the OS radius waiver). Signal vocabulary follows production: running = pulsing accent dot, needs-you = warning, failed = danger, ok = success.
