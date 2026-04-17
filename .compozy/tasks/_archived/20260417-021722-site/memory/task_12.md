# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create `packages/site/content/runtime/workspaces/{resolver,config-overlays,multi-root}.mdx` plus `meta.json`, grounded in current `internal/workspace/` and `internal/config/` behavior.
- Required gates before completion: site build, browser QA on all touched routes, full `make verify`, self-review, task tracking updates, then one local commit if clean.

## Important Decisions
- Treat current source as implementation authority; archived workspace-entity specs are useful for terminology but may be stale.
- Document additional roots as agents/skills-only; current implementation loads config, top-level MCP sidecars, and workspace memory from the primary root only.
- Use `bunx turbo run build --filter=@agh/site` for the real site build; keep the stale task selector result as evidence only.

## Learnings
- Shared memory says the docs package is `@agh/site`; the task's literal `turbo run build --filter=packages/site` selector is stale.
- Shared memory records an existing full-gate risk in `web/src/styles.test.ts`; verify current state before completion/commit.
- Resolver lookup accepts workspace ID/name/absolute path; path inputs are canonicalized with `filepath.Abs` + `EvalSymlinks` and must exist as directories.
- `agh session new` without `--workspace` or `--cwd` sends the CLI current working directory as `workspace_path`, which triggers resolve-or-register.
- Workspace cache snapshots include global/workspace config, top-level MCP sidecars, discovered agents, discovered skills, and adjacent MCP sidecars; cache TTL defaults to 10 minutes.

## Files / Surfaces
- Planned docs output: `packages/site/content/runtime/workspaces/*`
- Runtime nav likely needs `packages/site/content/runtime/meta.json` if the new section is not already listed.
- Created `packages/site/content/runtime/workspaces/{resolver,config-overlays,multi-root}.mdx` and `meta.json`.
- Updated `packages/site/content/runtime/meta.json` to include `workspaces`.

## Errors / Corrections
- `bunx turbo run build --filter=packages/site` failed because `packages/site` is not a package name.
- Correct build `bunx turbo run build --filter=@agh/site` passed and generated 153 static pages including `/runtime/workspaces/*`.
- Browser QA via `agent-browser` rendered all three workspace routes and followed sidebar links between touched pages.
- Full `make verify` failed in unrelated `web/src/styles.test.ts` assertions expecting old neutral tokens `#121212/#1C1C1E/#2C2C2E`; current CSS contains `#141312/#1e1c1b/#2e2c2b`.

## Ready for Next Run
- Docs implementation is task-scoped and task-specific validation passed, but do not mark task_12 complete or commit until full `make verify` is clean.
