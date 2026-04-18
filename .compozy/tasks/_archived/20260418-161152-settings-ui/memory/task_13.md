---
name: task_13 — MCP Servers scoped collection page
description: Implementation notes for the MCP servers scoped/targeted collection page
type: project
---

# Task Memory: task_13.md

## Objective Snapshot

- Build `/_app/settings/mcp-servers` with explicit scope (global/workspace), target (auto/config/sidecar), and source-precedence controls.
- Reuse shared collection primitives from task_12 without hiding MCP-specific semantics.
- Expose `effective_source`, `shadowed_sources`, and `available_targets` end-to-end.

## Important Decisions

- Scope switch is a chip row (Global + one chip per workspace). Switching scope resets the editor/delete state and calls `selectWorkspace(id)` / `selectGlobal()` on the hook.
- Target selector in the editor dialog is an explicit `<select>` rather than segmented buttons. Availability derives from `entry.source_metadata.available_targets` filtered through the current scope.
- Delete dialog has its own target selector + an inline preview of the effective source and the first shadowed source (or a "no other sources" note) so operators see what becomes effective after delete.
- `MCPEditorState` and `MCPDeleteState` are tagged unions like the providers/environments pattern; the editor state carries a `target: SettingsMCPServerTarget` field so the UI can pre-select `auto` and let operators override per-save.
- New-server flows default `target="auto"` (backend interprets this as "write to sidecar by default"), matching the TechSpec collection mutation semantics.
- "Last action" banner surfaces the `write_target` (mapped to short labels like `GLOBAL MCP` / `WS CFG`) plus a `remainingShadowed` count so the operator can tell whether lower-precedence definitions will become effective on reload.

## Learnings

- The route tree's tanstack vite plugin will regenerate the file during dev/build and rename `MCPServers` → `McpServers` in its generated names; the tests import `Route` from `./mcp-servers`, so either spelling is fine. Still worth hand-editing `routeTree.gen.ts` so typecheck/tests work before dev server runs.
- `useWorkspaces()` lives at `@/systems/workspace` and returns `WorkspacePayload[]` with `id`/`name`/`root_dir`. The scope row consumes that directly — no extra adapter needed.
- Route tests that hit `React.useMemo`-derived descriptions need to render the editor dialog inside the relevant `editor.mode` state via `pageState` override — not by calling `openEdit`/`openCreate`, because the component is mocked behind `useSettingsMCPServersPage`.
- Route coverage hit 90%+ only after adding tests for ArgsEditor/EnvEditor add/remove buttons and the delete target selector — the dialog sub-components were the main uncovered paths.

## Files / Surfaces

- `web/src/hooks/routes/use-settings-mcp-servers-page.ts` (new) — scope/target state, editor/delete tagged unions, `resolveAvailableTargets` filter
- `web/src/hooks/routes/use-settings-mcp-servers-page.test.tsx` (new) — 12 hook tests covering scope switches, target submission, precedence-aware delete, validation errors, scope-reset-mid-flow
- `web/src/routes/_app/settings/mcp-servers.tsx` (new) — scope chips, servers table with env/args counts + source badges, editor with Target selector + ArgsEditor + EnvEditor, delete dialog with fallback preview and target selector
- `web/src/routes/_app/settings/-mcp-servers.test.tsx` (new) — 20 route tests covering scope chips, table, editor, delete dialog, action banner
- `web/src/routeTree.gen.ts` (manual edit) — added mcp-servers route import + entries across every map/union; vite plugin later regenerated identical content with `Mcp` casing.

## Errors / Corrections

- Early hook sketch returned `SettingsMCPServerListFilter` with `workspace_id` set unconditionally. Switched to a union that only sets `workspace_id` in workspace scope so the query key and refetch cache line up with the adapter's normalized filter.
- `-mcp-servers.test.tsx` typed `envelope` as `{ mcp_servers: SettingsMCPServerEntry[] } | null` (not the full envelope) to keep test state building terse; the route only reads `page.servers` so the narrower shape is enough.

## Ready for Next Run

- task_14 (hooks & extensions) can continue to reuse the shared collection primitives. The hooks list there will be less precedence-heavy than MCP, so it can likely drop the scoped chip row and only keep the editor/delete dialogs.
- If `writeTargetLabel` helpers need to move (e.g., for task_15 QA), they live inline in `mcp-servers.tsx` right now; extract to `@/systems/settings/lib/write-targets.ts` only if another surface needs them.
