# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create hooks and extensions runtime docs for task 15: event catalog, declaration reference, extension install guide, extension development guide, plus sidebar metadata.
- Required validation includes site build, live browser QA through `agent-browser`, full `make verify`, self-review, tracking updates, and a local commit only if verification is clean.

## Important Decisions
- Current implementation does not provide first-class `http` or `agent` hook executor kinds. Docs must describe implemented executor kinds (`subprocess`, daemon-only `native`, reserved `wasm`) and show HTTP/agent-adjacent workflows through subprocess hooks or extensions.
- Runtime sidebar should add `hooks` and `extensions` after `bridges` and before `cli-reference`.

## Learnings
- Hook event catalog is source-driven by `internal/hooks/events.go` and `internal/hooks/introspection.go`; there are 33 hookable events across 11 families.
- Hook matchers are family-scoped and string matcher fields support exact values plus Go `path.Match` wildcards.
- Subprocess hook executors receive JSON payload on stdin, return JSON patch on stdout, default to 5s timeout, and capture stderr/stdout up to 8 KiB.
- Extension manifests are `extension.toml` or `extension.json`; resource-only extensions do not need a subprocess, while extensions with runtime capabilities/actions do.
- The built-in GitHub extension registry supports release lookup/download but not full-text search; docs should not imply GitHub-backed search is a complete marketplace search.

## Files / Surfaces
- Authored docs: `packages/site/content/runtime/hooks/{event-catalog,declaration}.mdx`, `packages/site/content/runtime/extensions/{install,develop}.mdx`.
- Authored metadata: `packages/site/content/runtime/hooks/meta.json`, `packages/site/content/runtime/extensions/meta.json`.
- Updated nav: `packages/site/content/runtime/meta.json`.

## Errors / Corrections
- Task text mentions shell, HTTP, and agent executors, but current source only implements subprocess/native/wasm stub. Use source truth in docs.
- Required `make verify` failed outside task scope in `web/src/styles.test.ts`: tests still expect `#121212`, `#1C1C1E`, and `#2C2C2E`, while current CSS contains `#141312`, `#1e1c1b`, and `#2e2c2b`. Do not update task status, master tracking, or commit until full verification is clean.

## Ready for Next Run
- Baseline before edits: all four required MDX files were absent; hooks/extensions runtime directories were absent.
- Task-scoped validation passed so far: CLI help checks for documented hook/extension commands, source-vs-doc event coverage comparison (33/33), browser QA via `make site-dev` + `agent-browser`, and `bunx turbo run build --filter=@agh/site`.
- Task-specified `bunx turbo run build --filter=packages/site` failed because `packages/site` is not a package name; use `@agh/site`.
- Tracking files for task_15 and `_tasks.md` are intentionally not marked complete because `make verify` is not clean.
