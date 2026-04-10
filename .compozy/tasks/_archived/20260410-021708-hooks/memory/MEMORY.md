# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 established the `internal/hooks` base package with stdlib-only types, taxonomy metadata, and test coverage. Follow-on tasks should extend this package without importing other `internal/` packages back into the base layer.

## Shared Decisions
- Event payloads in `internal/hooks` are snapshot structs that duplicate the needed session and ACP fields by value to preserve the package's dependency-free boundary.
- Task 02 mirrors skill-source precedence inside `internal/hooks` with a local `HookSkillSource` enum so ordering can preserve Bundled → Marketplace → User → Additional → Workspace without importing `internal/skills`.
- Task 07 confirmed the base `internal/hooks` package cannot import `internal/session` once `internal/skills` depends on `internal/hooks`; notifier/session bridge code must stay in an upper layer such as `internal/daemon` rather than in the base hooks package.
- Task 08 widened `session.Notifier.OnAgentEvent` to accept `any`; ACP-specific downcasting now stays in upper layers (for example `internal/observe`) so config-driven hook declarations can reuse `internal/hooks` without reintroducing a `config -> hooks -> acp -> config` import cycle.
- Task 09 kept the `session.Notifier` bridge as a daemon-local adapter over `Hooks` instead of moving the interface implementation into `internal/hooks`; `Hooks` remains the authoritative runtime, but the adapter is required to avoid the existing `session -> config/skills -> hooks` import cycle.
- Task 10 added a separate `session.HookDispatcher` seam for the load-bearing runtime dispatch path; `session.Notifier` remains for observer/dream fan-out, while session lifecycle hook execution should go through `SessionManagerDeps.Hooks`.
- Task 11 requires new session-scoped hook families to be wired in lockstep across `session.HookDispatcher`, the daemon-local `hooksNotifier` bridge, and the daemon test fake, because the session package still cannot depend directly on the hooks runtime implementation.

## Shared Learnings
- `Executor` and `HookExecutorKind` now exist in the base package, so later executor and registry tasks can build on the contract instead of redefining it.
- `HookDecl.PrioritySet` is needed to distinguish explicit `priority: 0` from an unset priority; later parsers must set this when a declaration explicitly supplies a priority value.
- `RegisteredHook` intentionally carries normalized dispatch metadata only; declaration-specific shell details such as command, args, env, and working directory must stay bound inside executor instances during normalization/resolution instead of being copied onto the registered hook.
- Go filenames ending in `*_wasm.go` are treated as GOARCH-specific build files; future non-wasm stub seams should avoid that suffix unless they are intentionally wasm-only.
- Task 04 added the package-private generic sync `pipeline[P, R]` plus `encodeJSON`/`decodeJSON` helpers and a depth guard; future typed dispatchers should feed it one hook snapshot per dispatch rather than re-selecting hooks mid-pipeline.
- Native Go hooks now have a typed bypass via `NewTypedNativeExecutor`, while subprocess/Wasm-style executors stay on the byte serialization boundary; future task wiring should prefer the typed native path for in-process callbacks.
- Task 05 added a package-private `asyncPool` that passes a pool-owned worker context into each submitted task; future async dispatch wiring must layer per-hook timeout logic inside the submitted closure instead of expecting the pool to read `RegisteredHook.Timeout` directly.
- Task 06 added source-specific declaration-provider seams plus an executor resolver to `Hooks`; future task_07/task_08 loaders can feed `HookDecl` slices directly into those providers without changing rebuild semantics, while native hooks still require an explicit resolver binding.
- Task 06 matches async hooks against the pre-pipeline payload, then runs the matched async set after the sync pipeline against the final payload snapshot; later tasks should preserve that behavior so already-matched async hooks still fire even when the sync path short-circuits.
- Skill-owned hook declarations need a final normalization pass after `skills.Skill.Source` and provenance are resolved so each `hooks.HookDecl` carries the correct `Source` and `SkillSource` metadata for ordering and marketplace policy checks.
- Task 09 added `skills.Watcher.SetAfterRefresh`; hooks rebuilds should attach there so a watcher refresh updates the skills registry and swaps the hooks snapshot in the same change cycle before the next lifecycle dispatch.
- The permission deny-only invariant must treat ACP `reject-once` and `reject-always` decisions as denied states, not just generic `deny`/`block` strings; otherwise subprocess `permission.request` hooks can escalate rejected requests into allows.
- Task 11 introduced `Manager.runContextCompaction` as the session seam for `context.pre_compact` and `context.post_compact`; future compaction paths should route through that helper so hook patches change the actual compaction inputs instead of being observed out-of-band.
- Task 12 stores hook execution audits in each session DB via a `hook_runs` table; observer introspection reopens the per-session store on demand for `/api/hooks/runs` rather than depending on an in-memory recorder.
- Task 12 routes session-managed hook telemetry through context-carried writers first and only falls back to the observer sink when no live session recorder is available; future hook telemetry should preserve that preference to avoid duplicate write paths.

## Open Risks
- `Hooks.OnAgentEvent` remains intentionally conservative in task_06 because the current notifier surface lacks enough typed data for the full taxonomy; task_10 still owns the direct session/runtime integrations for richer event families.

## Handoffs
