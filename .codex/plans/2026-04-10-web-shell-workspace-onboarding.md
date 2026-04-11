# AGH Web Shell and Workspace Onboarding

## Summary

- Bring `skills` and `knowledge` back in line with the Paper shell by standardizing their shared page chrome: the outer app sidebar gets its missing right border, the inner list rail uses a distinct surface from the global sidebar, and the toggle pills become compact mono segments with dark text on the active accent fill.
- Fix the workspace flow at the source by centralizing active-workspace selection. `_app`, `knowledge`, and `skills` should all consume the same active workspace instead of each falling back to `workspaces[0]`.
- Add a first-run onboarding experience plus a reusable workspace setup entrypoint so users can either register their OS home directory as the global workspace in one click or register a custom absolute path.

## Key Changes

- Shell and page structure:
  - Extract a shared workspace page shell for `knowledge` and `skills` with one consistent header row, count badge, compact pill toggle group, inner rail surface, and detail pane layout.
  - Add a right border to the outer `AppSidebar` while keeping the icon-rail divider as an internal separator.
  - Replace the hand-coded toggle/button styling in `knowledge`, `skills`, and marketplace category filters with one shared interactive pill primitive; active pills use a dedicated dark accent-foreground token instead of white.
- Workspace state and setup:
  - Add a shared active-workspace hook/store in the workspace system and migrate `_app.tsx`, `knowledge.tsx`, and `skills.tsx` off local first-workspace fallback logic.
  - Add one reusable `WorkspaceSetup` UI that is rendered in two contexts: a dialog opened by the sidebar `+` button and a full onboarding screen shown whenever workspace loading is complete and the registry is empty.
  - Manual registration stays path-based and uses the existing resolve/register flow with an absolute-path input only. Successful registration invalidates the workspace list, selects the returned workspace, and keeps the user on the current route.
- Backend/API:
  - Extend `GET /api/daemon/status` with `daemon.user_home_dir`, explicitly representing the OS user home directory, not AGH’s `~/.agh` home.
  - Add a small web daemon-status adapter/hook so the “Use global workspace” CTA can call `resolveWorkspace({ path: user_home_dir })` in one click.
  - If daemon status is unavailable, disable only the global-workspace CTA with inline explanation and keep manual path registration available.

## Test Plan

- Web route/component coverage:
  - `_app` renders onboarding instead of the normal shell when `useWorkspaces()` resolves to an empty list.
  - The sidebar add-workspace trigger opens the shared setup flow and selects the returned workspace on success.
  - `knowledge` and `skills` read the shared active-workspace state rather than always using the first workspace.
  - The shared pill primitive covers compact sizing, active dark foreground, and reuse across knowledge, skills, and marketplace filters.
- Web adapter/hook coverage:
  - New daemon-status parsing includes `user_home_dir`.
  - Workspace setup covers success, absolute-path validation, and the disabled global CTA path when daemon status fails.
- Go/API coverage:
  - `getDaemonStatus` returns `user_home_dir`.
  - Handler tests prove `user_home_dir` matches the OS user home and not `homePaths.HomeDir`.
- Verification:
  - Run `make verify`.
  - Run `bun run --cwd web test`.

## Assumptions

- UI copy remains in English to match the current app and the Paper reference.
- First-run onboarding replaces the normal app shell until at least one workspace exists; once a workspace is created, the current route can render normally without forced navigation.
- This pass does not add custom workspace naming or browser directory-picker behavior; custom registration is an absolute-path entry flow.
