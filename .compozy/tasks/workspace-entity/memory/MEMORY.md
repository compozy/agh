# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 02 moved the global session registry to `sessions.workspace_id` with a real foreign key into `workspaces`; later tasks should treat workspace IDs as the durable join key for global/session-facing data.
- Task 04 made `session.Manager` create/resume resolver-backed: sessions now persist `WorkspaceID` only, and any runtime workspace root path is derived from resolver output rather than metadata or daemon-side `os.Getwd()` fallback.
- Task 05 made workspace config loading explicit-only: `config.Load()` now applies workspace `.env` and `.agh/config.toml` overlays only when callers pass `WithWorkspaceRoot(root_dir)` instead of inferring a workspace from the process current directory.
- Task 06 wired `internal/daemon` to construct a GlobalDB-backed workspace resolver at boot, inject it into `session.Manager`, and run dream consolidation on workspace IDs/refs instead of raw session path strings.
- Task 10 made the HTTP API workspace-aware: `/api/workspaces` now supports register/list/detail/update/delete/resolve, `POST /api/sessions` accepts registered `workspace` refs or absolute `workspace_path`, and `GET /api/sessions` supports `?workspace=` lookup through the resolver.
- Task 11 mirrored that contract over UDS and CLI transport surfaces: `internal/udsapi` now exposes `/api/workspaces` CRUD/resolve plus workspace-aware session create/list semantics, and CLI session payloads/rendering now read `workspace_id` and `workspace_path`.

## Shared Decisions
- Legacy `~/.agh/agh.db` files that still have `sessions.workspace` are upgraded in place by creating workspace rows from distinct roots, copying dependent session tables, and swapping the new schema transactionally.
- `session.CreateOpts` now uses `Workspace` for registered workspace identifiers/names and `WorkspacePath` for explicit filesystem paths that should go through `ResolveOrRegister`.
- Multi-root discovery order now lives in `internal/config` via exported helpers so resolver and future consumers share the same `root_dir -> add_dirs -> global` precedence for agent/resource discovery.
- Task 07 moved prompt assembly onto `workspace.ResolvedWorkspace`: `session.PromptAssembler`, `session.PromptProvider`, the composed assembler, memory assembler, and skills catalog now receive the resolver snapshot directly instead of a raw workspace root string.
- Task 08 extends ACP `session/new` and `session/load` payloads with a top-level `additional_dirs` field; `internal/acp` uses local wire request structs for this because `acp-go-sdk@v0.6.3` does not model the field yet.
- Task 09 made `internal/observe` resolve permission/config state from `WorkspaceID` via the injected workspace resolver; observer snapshots and warning logs now carry `workspace_id` instead of relying on `SessionInfo.Workspace` paths.
- Task 09 made explicit dream consolidation inputs normalize through the workspace resolver inside `internal/memory.Service`; the service ensures workspace memory directories from the resolved root and passes the normalized workspace ID to the dream spawner.
- HTTP and UDS now share session/workspace transport validation and error-to-status mapping through `internal/apisupport`, so future transport changes should update that helper layer first instead of forking semantics per API surface.
- Task 13 added a dedicated `web/src/systems/workspace/` module; the web shell now selects registered workspaces by stable workspace ID, filters session queries through `GET /api/sessions?workspace=`, and sends that selected ID in session-create payloads.

## Shared Learnings
- The `workspace-entity` PRD directory is currently untracked in git; workflow memory and task tracking updates should stay out of auto-commits unless repository policy changes.
- Task 02 staged `WorkspaceID` onto `store.SessionInfo`, `store.SessionMeta`, and runtime session metadata surfaces while temporarily retaining the workspace path field for config/runtime consumers that task_04 will refactor.
- Task 03 confirmed an implementation constraint: because `workspaces.root_dir` stores canonical paths, the resolver can repair stale non-canonical registrations on resolve, but it cannot detect later retargeting of a symlink alias once that alias is no longer persisted.
- Current HTTP/UDS/daemon callers still map their single workspace input to `WorkspacePath`; later session-contract tasks should decide when those surfaces start sending registered workspace refs directly.
- `internal/skills.Registry.ForWorkspace(...)` now overlays only resolver-provided workspace/additional skill paths; bundled and user-level global skills remain sourced from the registry's global snapshot rather than being reloaded from `ResolvedWorkspace.Skills`.
- Resolver-backed HTTP payloads now expose canonical `workspace_path` values from the resolved workspace snapshot rather than echoing the raw path alias submitted by the client.
- Task 12 confirmed the workspace resolver cache invalidation hook is still internal-only (`Resolver.Invalidate`) and is not exposed over the current HTTP/UDS workspace service contract; CLI task scope stopped at the existing transport and did not invent a `--force-refresh` path.
- The web workspace system already wraps `/api/workspaces/resolve`, but task 13 deliberately kept the shell UX on registered workspaces only; there is no browser-side path-entry flow yet.

## Open Risks
- Future symlink-heavy UX should not assume a registered workspace still remembers the original alias path; supporting live alias retarget detection would require storing the alias separately from the canonical root.

## Handoffs
