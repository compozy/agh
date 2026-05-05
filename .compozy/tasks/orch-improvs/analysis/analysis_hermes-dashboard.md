# Analysis: hermes-dashboard

Read-only exploration of `.resources/hermes/` (kanban dashboard plugin + UI/TUI surfaces) for AGH task `orch-improvs`. Cross-referenced with AGH `web/`, `internal/api/`, `internal/extension/`, `@agh/extension-sdk`.

## Scope

- Path explored: Hermes' bundled `kanban` dashboard plugin (web tab, REST surface, WebSocket live-tail, drag-drop interactions) plus the broader **dashboard-plugin contract** that lets the kanban tab ship as a drop-in extension instead of as hardcoded UI.
- Topic: plugin contract, real-time updates, permission/scope, drag-drop, TUI/web parity, runtime/UI decoupling, browser-safe imports, stale-state handling.
- Sources read in full: `plugins/kanban/dashboard/{manifest.json,plugin_api.py}`, the kanban plugin test suite, the dashboard `--stop`/`--status` lifecycle tests, the stale-dashboard kill path, the browser-safe-import linter test, and the Profiles nav-label test.
- Sampled: `plugins/kanban/dashboard/dist/index.js` (first 200 lines), `hermes_cli/web_server.py` (auth middleware + plugin discovery + asset serve + API mount), `web/src/plugins/{registry.ts,slots.ts,usePlugins.ts}`, `ui-tui/src/hooks/useQueue.ts`, and `website/docs/user-guide/features/{kanban.md,extending-the-dashboard.md}`.
- Out of scope: the `kanban_db` schema/SQL layer, the dispatcher/run-claim invariants, gateway notifier wiring, and CLI argparse internals — referenced only where they constrain the UI contract.
- Total available files: ~14 plugin/test/web files load-bearing for the contract.

## Overview

Hermes ships kanban as a **bundled dashboard plugin**, not a built-in React route. The dashboard discovers plugins at boot by scanning `plugins/*/dashboard/manifest.json` across three roots (user `~/.hermes/plugins/`, bundled repo `plugins/`, opt-in project `./.hermes/plugins/`); each manifest declares a tab path, an entry JS bundle, optional CSS, and an optional Python `api` file (`plugins.py:55-65`, `web_server.py:3558-3633`). At request time the FastAPI app:

1. Serves the plugin's `dist/index.js` and `dist/style.css` from `/dashboard-plugins/<name>/<file>` with path-traversal protection (`web_server.py:3931-3966`).
2. Mounts the plugin's `APIRouter` under `/api/plugins/<name>/` once at process start (`web_server.py:3969-4013`).
3. **Bypasses session-token auth for `/api/plugins/*`** in the auth middleware (`web_server.py:228`). Plugin REST is unauthenticated by design; the dashboard is loopback-only and the host-header middleware blocks DNS-rebinding.

The browser side fetches `/api/dashboard/plugins`, then injects each plugin's `<script>` tag, waits up to 2 s for the bundle to call `window.__HERMES_PLUGINS__.register(name, Component)`, and renders the resulting React component as a tab (`usePlugins.ts:1-123`, `registry.ts:55-149`). The plugin SDK exposes shared React, shadcn primitives, hooks, `fetchJSON`, `cn`/`timeAgo` utils, and a `PluginSlot` registry on the window so plugin bundles **don't bundle their own React** — they consume the host's. `kanban/dist/index.js` is therefore a plain IIFE, no build step, ~one file (`dist/index.js:12-25`).

Live updates are a **WebSocket polling tail of an append-only `task_events` SQLite table on a 300 ms interval**, over WAL — no fan-out broker, no notify channel (`plugin_api.py:1109-1182`). Cursor is `since=<event_id>`, board is pinned at handshake via `?board=<slug>`, and the WS auth uses a `?token=<session>` query param because browsers can't set `Authorization` on the WS upgrade. When a burst of events arrives the React UI debounces and refetches the cheap `GET /board` endpoint instead of patching local state from each event kind (kanban.md:481-484).

Drag-drop is HTML5 native with a `text/x-hermes-task` MIME and a pointer-based touch fallback (`dist/index.js:64, 198-200`); the actual move PATCHes `/tasks/:id` and re-renders from the WS event echo. The TUI does **not** integrate the kanban — `useQueue` is a separate in-process FIFO for queued composer messages (`ui-tui/src/hooks/useQueue.ts:15-76`). TUI/web parity for kanban is not implemented; the only "two-surface" parity is CLI `hermes kanban` ⇄ web dashboard, both reading/writing `~/.hermes/kanban.db` through the shared `kanban_db` Python layer.

## Mechanisms / Patterns

### 1. Plugin manifest contract (`manifest.json`)

```json
{
  "name": "kanban",
  "label": "Kanban",
  "icon": "Package",
  "version": "1.0.0",
  "tab":   { "path": "/kanban", "position": "after:skills" },
  "entry": "dist/index.js",
  "css":   "dist/style.css",
  "api":   "plugin_api.py"
}
```

- `tab.position` accepts `"after:<routeName>"`, `"end"` (default), and an optional `tab.override: "/<built-in>"` to **replace** an existing route. `tab.hidden: true` lets a plugin register slot components without adding a tab (`web_server.py:3598-3607`).
- `slots: ["header-left", "sessions:top", ...]` declares which named locations the plugin populates; the React shell renders `<PluginSlot name="header-left" />` and stacks all registered components in registration order (`slots.ts:60-93, 124-199`).
- `api: "plugin_api.py"` makes the FastAPI mount at `/api/plugins/<name>/` opt-in.

### 2. REST surface (`plugin_api.py:202-984`)

| Verb | Route | Notes |
|---|---|---|
| `GET` | `/board?tenant=&include_archived=&board=` | Single denormalized payload: columns, tenants, assignees, **`latest_event_id`** (cursor seed for WS), `now`, plus per-task `link_counts`, `comment_count`, and a `progress` `{done, total}` rollup computed in one SQL pass over `task_links` join `tasks` (`plugin_api.py:222-289`). |
| `GET` | `/tasks/{id}` | Hydrates `{task, comments, events, links, runs}` for the side drawer in one round trip (`plugin_api.py:308-323`). |
| `POST` | `/tasks` | Creates and **opportunistically returns a `warning` field** if the task is `ready+assigned` and the dispatcher probe says no gateway is running (`plugin_api.py:373-383`). |
| `PATCH` | `/tasks/{id}` | Status / assignee / priority / title / body / result / **structured `summary` + `metadata`** for `done`. Refuses `status=running` directly (only the dispatcher's `claim_task` may write it — preserves the run-row invariant) (`plugin_api.py:430-465, 452-456`). |
| `POST` | `/tasks/bulk` | Per-id outcome list — partial failures don't abort siblings (`plugin_api.py:635-705`). |
| `POST` | `/links`, `DELETE` | Cycle attempts rejected at the DB layer with 400. |
| `POST` | `/dispatch?dry_run=&max=` | Manual dispatcher nudge so the UI doesn't wait the 60 s tick. |
| `GET` | `/config` | Reads `dashboard.kanban.*` from `~/.hermes/config.yaml` (server-rendered defaults) (`plugin_api.py:712-733`). |
| `GET/POST/DELETE` | `/home-channels`, `/tasks/:id/home-subscribe/{platform}` | Per-task notification toggles into telegram/discord/slack home channels — reads the live `GatewayConfig` so env-var overlays work (`plugin_api.py:752-882`). |
| `GET` | `/tasks/{id}/log?tail=N` | Worker stdout/stderr with byte cap; on-disk log auto-rotates at 2 MiB. |
| `GET/POST/PATCH/DELETE` | `/boards`, `/boards/{slug}` | Multi-board CRUD; client picks active board via `localStorage` separately from the CLI's on-disk pointer. |
| `WS` | `/events?since=<id>&board=&token=` | Live event tail. |

Every handler opens a per-call `kanban_db.connect()`, calls `init_db()` first to **self-heal a fresh install** (no "no such table" if the user opens the dashboard before running `kanban init`), and closes in `finally` (`plugin_api.py:97-113, 477-490 in tests`).

### 3. WebSocket live-tail design (`plugin_api.py:1109-1182`)

- **Polling, not push.** The handler runs `asyncio.to_thread(_fetch_new, cursor)` every 300 ms. Inline comment: *"SQLite WAL + 300 ms polling is the simplest and most robust approach; it adds a fraction of a percent of CPU and has no shared state to synchronize across workers."*
- **Cursor is the `task_events.id` monotonic.** Initial seed is `latest_event_id` from the prior `GET /board` so the client never replays history.
- **Batch read** with `LIMIT 200` per tick; `created_at`, `kind`, JSON `payload`, `task_id`, `run_id`, `id` all returned per row.
- **Frontend doesn't reconcile event-by-event.** Per the docs: "When a burst of events arrives, the frontend reloads the (very cheap) board endpoint — simpler and more correct than trying to patch local state from every event kind" (`kanban.md:481-484`).
- **Board pinned at handshake.** Switching boards opens a new WS instead of reconciling two cursors (`plugin_api.py:1132-1140`).
- **Auth = constant-time HMAC compare against `_SESSION_TOKEN`** loaded lazily from `hermes_cli.web_server`; in the bare-FastAPI test harness the import fails and the check returns `True` so tests run without the dashboard module (`plugin_api.py:53-72`).

### 4. Frontend plugin SDK (`web/src/plugins/registry.ts`)

The host exposes on `window.__HERMES_PLUGIN_SDK__`:

- **`React`** + **`hooks`** dict (`useState/useEffect/useCallback/useMemo/useRef/useContext/createContext`) — plugins write `const { React } = SDK; const h = React.createElement` and never bundle React.
- **`components`**: `Card, Button, Input, Label, Select, SelectOption, Badge, Tabs, Separator, PluginSlot` (shadcn-flavoured).
- **`api`** (typed Hermes client) and **`fetchJSON`** for plugin-defined endpoints.
- **`utils`**: `cn`, `timeAgo`, `isoTimeAgo`.
- **`useI18n`** hook.

Plugin lifecycle:
1. Manifest fetch → CSS `<link>` injected → `<script src=/dashboard-plugins/<name>/dist/index.js>` injected with `data-hermes-plugin` attribute and dev-mode cache-bust (`usePlugins.ts:38-86`).
2. Plugin IIFE calls `window.__HERMES_PLUGINS__.register(name, Component)` to hand its React component to the host registry.
3. `onload` triggers `notifyPluginRegistry()`; if the plugin failed to register a microtask later, error state `NO_REGISTER` is set and rendered as an inline failure card.
4. On unmount in dev, injected `<script>` tags are removed (HMR-safe). Production keeps a `loadedScripts` set so no duplicate execution on remount.
5. `unregisterPluginSlots(plugin)` clears slot registrations for HMR (`slots.ts:152-162`).

### 5. Browser-safe-import lint (`test_dashboard_browser_safe_imports.py`)

A static test asserts no `web/src/**/*.{ts,tsx}` imports from the **root barrel** `"@nous-research/ui"` — only deep paths like `"@nous-research/ui/ui/components/badge"` are allowed. Reason: the root barrel re-exports server/Node-targeted utilities that break in the browser bundle. This is essentially **the dashboard's "tree-shake-by-policy" rule, enforced by a unit test instead of bundler config.**

### 6. Lifecycle: `--status` / `--stop` / post-`update` cleanup

`hermes dashboard --status` and `--stop` are precedence-routed inside `cmd_dashboard` so neither falls through to the server-start path (`test_dashboard_lifecycle_flags.py:126-159`). The detection is a `ps -A -o pid=,command=` scan on POSIX and a `wmic process` + `taskkill /F` pair on Windows; the kill path SIGTERMs, waits up to 3 s on `time.monotonic` clock, then SIGKILLs survivors (`test_update_stale_dashboard.py:189-291`). Self-PID and `grep` lines are excluded; commands containing the word "dashboard" but not the actual `dashboard` subcommand are rejected to avoid catching user chats. **`hermes update` reuses the same kill path** so a fresh frontend bundle never serves against a stale Python backend — the rationale is in the file docstring (`test_update_stale_dashboard.py:1-12`). Windows wmic is invoked with `encoding="utf-8"` + `errors="ignore"` to survive non-UTF-8 locales; a `None` stdout (Python 3.11 reader-thread crash artefact) short-circuits to `[]` instead of raising (`test_update_stale_dashboard.py:347-394`).

### 7. Permission / scope filtering

- **Per-board**: query param `?board=<slug>` validated by `kanban_db._normalize_board_slug` (lowercase alphanumerics + hyphens + underscores, 1-64 chars, no `..`, no `/`); 400 on malformed, 404 on missing (`plugin_api.py:75-94`). The WS pins the board at handshake.
- **Per-tenant**: `?tenant=<name>` filters the board view; tenant list returned in `GET /board` to populate the dropdown (`plugin_api.py:275-281`).
- **Per-profile (assignee)**: `assignees` array returned by `GET /board`, `GET /assignees` returns the union of disk profiles + active board assignees so newly-created profiles appear in pickers before they have any tasks (`plugin_api.py:905-919`).
- **No per-user authz.** Single-host single-operator model. A `--host 0.0.0.0` exposes the entire kanban surface to the LAN, documented as "don't do that".

### 8. Optimistic update / rollback

There is **no client-side optimistic update path**. Every drag-drop fires a PATCH; the WS event echo eventually arrives and triggers the board refetch. Destructive transitions (`done`, `archived`, `blocked`) prompt for confirmation in `DESTRUCTIVE_TRANSITIONS` (`dist/index.js:57-61`). Bulk PATCH returns per-id results and the UI surfaces partials. The trade-off is paid in latency, and bought back in the "two surfaces can never drift" invariant — every UI mutation goes through the same `kanban_db` write that the CLI uses.

### 9. Stale state via fresh-DB auto-init

Every endpoint calls `_conn() → kanban_db.init_db()` first (idempotent CREATE TABLE IF NOT EXISTS) so the plugin self-heals. A test asserts that hitting `GET /board` on a tmp_path with no prior `kb.init_db()` call **creates `kanban.db`** instead of returning 500 (`test_kanban_dashboard_plugin.py:477-490`). This converts "first-time UX bug" into a non-issue — there's no "open the CLI first" requirement.

## Relevant Code Paths

**Hermes — plugin contract & runtime:**
- `.resources/hermes/plugins/kanban/dashboard/manifest.json:1-15` — manifest schema
- `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:53-72` — WS token compare
- `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:97-113` — fresh-install auto-init
- `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:202-301` — `GET /board` payload shape
- `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:409-506` — PATCH transitions + `running`-guard
- `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:509-557` — direct status write + run reclamation
- `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:635-705` — bulk batch with per-id failure
- `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:1109-1182` — WebSocket tail loop
- `.resources/hermes/plugins/kanban/dashboard/dist/index.js:12-200` — plugin entry, SDK consumption, drag-drop scaffolding, safe-markdown renderer
- `.resources/hermes/hermes_cli/web_server.py:97-234` — auth middleware (skip `/api/plugins/`), host-header DNS-rebind defense
- `.resources/hermes/hermes_cli/web_server.py:3558-3666` — `_discover_dashboard_plugins`, `/api/dashboard/plugins{,/rescan}`
- `.resources/hermes/hermes_cli/web_server.py:3931-3966` — `/dashboard-plugins/<name>/<file>` static route + path-traversal block
- `.resources/hermes/hermes_cli/web_server.py:3969-4013` — `_mount_plugin_api_routes` (importlib spec_from_file_location pattern)

**Hermes — frontend SDK:**
- `.resources/hermes/web/src/plugins/registry.ts:1-149` — registry + `exposePluginSDK`
- `.resources/hermes/web/src/plugins/usePlugins.ts:1-123` — manifest fetch → script inject → registration race
- `.resources/hermes/web/src/plugins/slots.ts:60-199` — slot registry + `<PluginSlot>`

**Hermes — tests as evidence:**
- `.resources/hermes/tests/plugins/test_kanban_dashboard_plugin.py:256-281` — `running`-PATCH must be rejected (run-row invariant)
- `.resources/hermes/tests/plugins/test_kanban_dashboard_plugin.py:477-490` — auto-init on first `/board`
- `.resources/hermes/tests/plugins/test_kanban_dashboard_plugin.py:498-533` — WS token enforcement
- `.resources/hermes/tests/plugins/test_kanban_dashboard_plugin.py:599-617` — bulk partial-failure semantics
- `.resources/hermes/tests/hermes_cli/test_dashboard_browser_safe_imports.py:1-17` — root-barrel import lint
- `.resources/hermes/tests/hermes_cli/test_dashboard_lifecycle_flags.py:126-159` — `--stop` must not fall through to server-start
- `.resources/hermes/tests/hermes_cli/test_update_stale_dashboard.py:1-12, 189-394` — kill-path semantics, Windows wmic encoding fix

**Hermes — TUI (parity gap):**
- `.resources/hermes/ui-tui/src/hooks/useQueue.ts:1-76` — composer FIFO, **not** kanban-aware
- `.resources/hermes/ui-tui/src/components/queuedMessages.tsx` — TUI renders the in-process queue only

**AGH — equivalent surfaces (cross-reference):**
- `internal/api/contract/` — Go contract surface (typed)
- `internal/api/httpapi/` — HTTP handlers
- `internal/extension/host_api*.go` — agent-facing extension API
- `internal/extension/manifest.go` — AGH extension manifest (typed Go)
- `web/src/{routes,systems,components}` — TanStack Router app

## Transferable Patterns

1. **Manifest-first, drop-in plugin contract.** The `manifest.json` + `dist/index.js` + `plugin_api.py` triplet is the single source of truth: backend mounts router at `/api/plugins/<name>/`, frontend serves `/dashboard-plugins/<name>/` static, browser injects `<script>` and waits for `register()`. AGH already has `internal/extension/manifest.go` and `internal/extension/host_api*.go`; what hermes adds is the **frontend half** — a host-side registry that discovers extension UI manifests, serves the bundle, and renders the registered React component as an extension-owned route. Worth lifting wholesale: the manifest fields (`tab.path`, `tab.position: "after:<route>"`, `tab.override`, `tab.hidden`, `slots: []`), the IIFE-with-shared-React pattern (extensions don't bundle React), and the slot registry.
2. **Append-only event table + WebSocket tail with monotonic cursor.** Hermes' `task_events` is exactly the shape AGH already has in `internal/daemon/hook_*.go` and the autonomy event stream. The pattern of "WS handshake seeds cursor from a `latest_event_id` returned in the initial `GET /board`, then 300 ms polling reads `id > cursor LIMIT 200`" is portable to AGH SQLite-backed event streams. AGH currently uses SSE in places; the cursor seeding and burst-debounced refetch are orthogonal to the transport.
3. **Refetch-on-burst instead of patch-from-event.** Render the cheap denormalized payload from `GET /board` on any event arrival; never write client-side reducers per event kind. Cuts a whole class of "two surfaces drifted" bugs. Especially valuable for AGH where the runtime owns the data shape and the frontend should never be a source of truth.
4. **Plugin REST under a documented unauth bypass.** Hermes carves `/api/plugins/*` out of the auth middleware on purpose, with the loopback-bind + host-header-rebind-defense as the line of defense. AGH binds to UDS for CLI and HTTP for web — translating this means plugin/extension routes can run loopback-only with the same Host-header validation. The exception is documented in code comments and in `plugin_api.py:14-24`.
5. **Self-healing fresh-install endpoints.** `_conn()` calling `init_db()` per request is idempotent and converts "user opened wrong surface first" from an error to a no-op. AGH's pattern of `EnsureSchema` at boot is the opposite direction; the per-request guard pattern is cheap and worth applying selectively.
6. **Run-row invariant guarded at the API layer.** `PATCH status=running` is **rejected with 400** at the plugin layer because only `claim_task` may write that status atomically. AGH's task_runs claim model has the same shape (`ClaimNextRun`); enforcing it at the HTTP boundary instead of relying on downstream rejection is a clearer contract.
7. **Per-id partial-failure bulk endpoint.** `POST /tasks/bulk` returns `{results: [{id, ok, error?}, …]}` instead of failing the whole batch. AGH's autonomy and queue surfaces would benefit — bulk reassign / archive / re-queue all want this.
8. **Stale-process detection + kill at upgrade time.** `hermes update` scans `ps`/`wmic` for stale dashboard daemons, SIGTERMs with grace, then SIGKILLs. AGH's daemon already has restart logic; the `--status` / `--stop` flag pair (precedence-routed so they never fall through to start) and the post-update cleanup are directly applicable to `agh daemon`.
9. **Browser-safe-import lint.** A 17-line static test that scans every `*.tsx`/`*.ts` for forbidden barrel imports. AGH `web/` already has linting; adding a similar zero-cost guard for any "browser-only / server-only" boundary in `@agh/ui` and `@agh/extension-sdk` would prevent class-of-bug regressions.
10. **Plugin SDK on `window` over npm peer-deps.** The host owns React, shadcn, the API client, and i18n; plugins consume them through `window.__HERMES_PLUGIN_SDK__`. Keeps plugin bundles small (no React, no shadcn copies), guarantees React identity (one renderer), and lets a host upgrade libraries without breaking plugin ABI as long as the SDK shape is preserved.
11. **Slot registry with `KNOWN_SLOT_NAMES` allowlist.** Page-scoped slots (`sessions:top`, `chat:bottom`, …) let plugins extend built-in routes without overriding them. AGH's autonomy/inspector pages would benefit from the same shell-slots + page-scoped-slots split.
12. **WS auth via query param + constant-time compare.** Pragmatic browser limitation handling. AGH SSE/WS endpoints in `internal/api/httpapi/` should follow the same pattern with `hmac.compare` over a session token derived from the daemon boot.
13. **`localStorage` for UI-only board selection, decoupled from CLI's on-disk pointer.** The browser user's active board pick doesn't shift the CLI's `~/.hermes/kanban/current` out from under a terminal. AGH's analogous "active project / active session" UI state should follow this rule.

## Risks / Mismatches

- **No optimistic UI.** PATCH-then-WS-echo means visible drag-drop latency on slow disks. AGH targets fast desktop UX; if this is unacceptable, AGH must add optimistic-with-rollback on top of the same surface.
- **300 ms WS polling.** Cheap, but **not push**. AGH already has hooks for real change-feeds at `internal/daemon/`; AGH could keep the cursor-seeded contract but drive the loop from a notify channel instead of a poll.
- **Plugin API is unauthenticated behind loopback.** The model assumes single-host single-operator. AGH ships an open-network protocol (AGH Network) and a bridge SDK; AGH-specific plugin/extension routes must NOT inherit this bypass — they need the daemon's existing token middleware applied uniformly. This is a non-trivial design choice: `/api/extensions/*` should remain authed.
- **Plugin bundles consume the host's React via `window`.** Works for hermes' single-React app; AGH `web/` is React 19 + TanStack Router with Vite. The same SDK pattern works, but AGH must commit to a stable plugin SDK shape and version it (hermes does not version it — they ship master to master).
- **No TUI/web parity.** Hermes' TUI (`useQueue`) is unrelated to kanban. AGH's TUI surfaces (`internal/cli/`) currently have no plugin/extension renderer. If AGH wants kanban-style boards in both TUI and web, that's net-new work — hermes only proves the web half.
- **Plugin Python API is loaded with `importlib.spec_from_file_location` and `sys.modules[module_name] = mod` pre-`exec_module`.** This is necessary for pydantic forward refs but introduces global state and blocks unloading. AGH's Go binary doesn't have this footgun, but the **plugin Python files are imported with no sandbox** — full host privileges. AGH's extension model uses out-of-process bridges and capability-scoped extension APIs (`internal/extension/host_api*.go`), which is strictly safer; do not regress to in-process Python loading.
- **Markdown rendering is a hand-rolled escape-then-replace renderer.** Hermes ships ~80 lines of regex-based rendering with `dangerouslySetInnerHTML` after escape (`dist/index.js:122-185`). It's small but not battle-tested vs `markdown-it` + DOMPurify. AGH should not copy this verbatim — use a vetted renderer.
- **Drag-drop uses `dataTransfer` MIME `text/x-hermes-task`.** Native HTML5 DnD is fine on desktop but accessibility-poor (no keyboard-only path documented). AGH should add `aria-grabbed` / keyboard alternative if shipping a kanban-shaped UI.
- **Plugin discovery is process-startup-cached** with `/api/dashboard/plugins/rescan` for forced refresh. Fine for a dev workstation; less fine if AGH wants live extension install/uninstall — would need cache invalidation hooks tied to the extension manager.
- **`POST /tasks` returns a `warning` field for the dispatcher-down case** (`plugin_api.py:373-383`). This is a soft-typed UI hint that's easy to miss in OpenAPI/codegen. AGH's contract-codegen co-ship rule means any equivalent must be a typed enum field, not a free-string `warning`.
- **`/dashboard-plugins/<name>/<file>` is path-traversal-checked but unauthenticated.** Same loopback assumption as the API. AGH should authenticate static plugin assets if they ever leave the loopback assumption.

## Open Questions

1. Does AGH want an in-process React-extension contract, or out-of-process iframe+postMessage? Hermes proves React-extension works for trusted bundles; AGH's bridge SDK already separates extension code from runtime via UDS. Choosing iframe loses the SDK-on-window ergonomics but isolates blast radius — relevant for third-party extensions.
2. Is the SDK shape versioned? Hermes pins to master-of-master. AGH `@agh/extension-sdk` already exists as a workspace; if extensions consume the SDK at runtime via `window.__AGH_EXTENSION_SDK__`, that channel needs an explicit semver and probably a compat shim layer at the edge.
3. What's the AGH equivalent of `task_events`? Sessions/messages for chat, autonomy events for the kernel, hook events for extensions. The cursor-seeded WS pattern wants one canonical stream per UI surface; need a design pass to identify which.
4. Per-user authz. Hermes is single-operator; AGH Network plus shared workspaces hint at multi-user. The "all `/api/plugins/*` is unauthed" carve-out doesn't survive that; either every plugin/extension HTTP route stays under the daemon's token middleware (cleaner) or AGH adopts hermes' bypass and accepts loopback-only.
5. TUI parity. If AGH wants TUI ⇄ web parity for board-shaped UIs (kanban-style autonomy view, session-list view), Bubbletea has no equivalent of the dashboard's plugin-discovery + slot system. Either build it (significant) or accept TUI as a separate surface that consumes the same REST endpoints but with a custom renderer.
6. Rescan vs hot-reload. Hermes' `/api/dashboard/plugins/rescan` is manual. AGH extensions are `agh extension install/uninstall` — should re-discovery be automatic on extension manager events?
7. Should the AGH equivalent of `_check_dispatcher_presence` ship as a typed `warnings: []` array rather than a single optional `warning` string? Multiple soft-warnings per response is plausible (e.g. dispatcher down AND queue near capacity). Worth deciding before the contract calcifies.

## Evidence

- Plugin manifest (Hermes): `.resources/hermes/plugins/kanban/dashboard/manifest.json:1-15`
- WebSocket tail loop: `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:1109-1182`
- WS token + lazy import: `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:53-72`
- Self-healing init on every connect: `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:97-113`; test `.resources/hermes/tests/plugins/test_kanban_dashboard_plugin.py:477-490`
- `GET /board` denormalized payload (counts, progress rollup, cursor seed): `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:222-301`
- `running`-guard run-row invariant: `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:452-456`; test `.resources/hermes/tests/plugins/test_kanban_dashboard_plugin.py:256-281`
- Direct-status write + run reclamation on drag-off-running: `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:509-557`
- Bulk per-id failure semantics: `.resources/hermes/plugins/kanban/dashboard/plugin_api.py:635-705`; test `.resources/hermes/tests/plugins/test_kanban_dashboard_plugin.py:599-617`
- `/api/plugins/*` carve-out from auth middleware: `.resources/hermes/hermes_cli/web_server.py:228`
- DNS-rebind host-header defense: `.resources/hermes/hermes_cli/web_server.py:139-221`
- Plugin discovery (3 sources, manifest schema, slots, override/hidden): `.resources/hermes/hermes_cli/web_server.py:3558-3633`
- Plugin static asset serving with traversal guard: `.resources/hermes/hermes_cli/web_server.py:3931-3966`
- Plugin API mount via importlib + sys.modules pre-exec: `.resources/hermes/hermes_cli/web_server.py:3969-4013`
- Frontend SDK on window (React, hooks, components, api, utils): `.resources/hermes/web/src/plugins/registry.ts:100-149`
- Plugin loader (manifest fetch → CSS link → script inject → register-race + 2 s timeout): `.resources/hermes/web/src/plugins/usePlugins.ts:38-123`
- Slot registry + `KNOWN_SLOT_NAMES`: `.resources/hermes/web/src/plugins/slots.ts:60-199`
- Browser-safe-import lint: `.resources/hermes/tests/hermes_cli/test_dashboard_browser_safe_imports.py:1-17`
- Lifecycle precedence (`--status` > `--stop` > start): `.resources/hermes/tests/hermes_cli/test_dashboard_lifecycle_flags.py:126-159`
- Stale-PID detection + SIGTERM/SIGKILL grace: `.resources/hermes/tests/hermes_cli/test_update_stale_dashboard.py:189-291`
- Windows wmic UTF-8 / None-stdout fix (#17049): `.resources/hermes/tests/hermes_cli/test_update_stale_dashboard.py:347-394`
- Refetch-on-burst rationale: `.resources/hermes/website/docs/user-guide/features/kanban.md:481-484`
- Architecture diagram & REST surface table: `.resources/hermes/website/docs/user-guide/features/kanban.md:412-454`
- Plugin SDK consumption pattern (IIFE, no React bundle): `.resources/hermes/plugins/kanban/dashboard/dist/index.js:12-25, 64-200`
- TUI parity gap (`useQueue` is composer-only, not kanban): `.resources/hermes/ui-tui/src/hooks/useQueue.ts:15-76`
