# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implemented task 10 end to end: new `sdk/typescript/` package with runtime, typed Host API client, test harness, scaffolding CLI/templates, explicit unit/integration tests, coverage >=80%, build success, and clean repo-wide `make verify`.
- Post-change signal: task 10 is ready to remain completed in `_tasks.md`, and task 11 can consume the SDK and scaffolder directly.

## Important Decisions

- The PRD/techspec/examples already define the SDK surface, so this run is using that approved design directly instead of creating a new design document.
- Split the deliverable into two workspace packages: `sdk/typescript` for `@agh/extension-sdk` and `sdk/create-extension` for the scaffolding CLI, both wired into the root workspace and Vitest project list.
- Keep publish output generated at build time via package-local `dist/` directories rather than checking compiled artifacts into source control.

## Learnings

- Previous extension tasks established the canonical protocol and Go-side Host API/manifest/tool contracts that the SDK must mirror; `_protocol.md` section 4 is the initialize source of truth and section 5.2 is the Host API method inventory.
- The real subprocess integration path is sensitive to handshake ordering: `Extension.onReady()` work that calls back into the host must run after the initialize response is written, otherwise the host and extension can race on the shared JSON-RPC channel during startup.
- The existing root `.gitignore` already ignores nested `dist/` and `coverage/` directories, so package build and coverage artifacts stay out of the worktree automatically.

## Files / Surfaces

- `.compozy/tasks/ext-architecture/task_10.md`
- `.compozy/tasks/ext-architecture/_techspec.md`
- `.compozy/tasks/ext-architecture/_protocol.md`
- `.compozy/tasks/ext-architecture/_examples.md`
- `package.json`
- `vitest.config.ts`
- `sdk/typescript/package.json`
- `sdk/typescript/src/`
- `sdk/typescript/scripts/postbuild.mjs`
- `sdk/typescript/vitest.config.ts`
- `sdk/create-extension/package.json`
- `sdk/create-extension/src/`
- `sdk/create-extension/templates/`
- `sdk/create-extension/vitest.config.ts`
- `internal/extension/host_api.go`
- `internal/extension/manifest.go`
- `internal/tools/tool.go`
- `internal/subprocess/transport.go`
- `internal/subprocess/handshake.go`
- `internal/hooks/events.go`
- `internal/hooks/types.go`
- `internal/hooks/payloads.go`

## Errors / Corrections

- The initial integration implementation let `onReady()` run too early via microtask scheduling; the fix was to defer readiness completion with `setImmediate()` so the initialize response reaches the host before any host API callback traffic begins.

## Ready for Next Run

- Task 10 is complete. The next consumer task should build reference extensions against `@agh/extension-sdk` and prefer `@agh/create-extension` templates instead of re-scaffolding from scratch.
