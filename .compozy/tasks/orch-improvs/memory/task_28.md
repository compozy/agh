# Task Memory: task_28

## Status

Completed 2026-05-05 through the mandatory Compozy -> Claude Opus frontend/docs lane.

## Objective Snapshot

- Authored `packages/site` runtime docs for task orchestration profiles and configuration.
- Covered `[task.orchestration]`, `[task.orchestration.profile]`, `[task.orchestration.review]`, execution profile shape, selector precedence, sandbox modes, worker runtime selection, continuation runs, and management paths across CLI, HTTP, UDS, native tools, and the operator web UI.
- Kept docs truthful to runtime code (`internal/task/profile.go`, `internal/task/manager.go`, `internal/task/manager_profile.go`, `internal/daemon/task_runtime.go`, `internal/config/task_orchestration.go`) and to generated artifacts (`openapi/agh.json`, generated CLI reference under `packages/site/content/runtime/cli-reference/task/profile/`). No hand edits to generated CLI pages.

## Important Decisions

- Reframed selector precedence step 2 ("Persisted profile") as **write-time** validation and step 4 ("Session start") as **load-only**: session start does not re-run validation; the daemon session bridge maps `worker.agent_name`/`provider`/`model` and sandbox policy from the persisted normalized profile, with provider/model still flowing through `config.ResolveSessionAgentWithRuntime`.
- Replaced the misleading "partial update" wording with explicit PUT-replace semantics: omitted blocks normalize to defaults (modes default to `inherit`, missing selector arrays become empty); to preserve a previously persisted selector you must re-send it.
- Restated the sandbox `none` and provider/model gates as **profile write-time** rejections enforced by `task.Service` validation, not as session-start runtime checks.
- Kept review-gate detail minimal in the profiles doc; task_29 owns full review-gate narrative docs.

## Learnings

- `task.Service.SetExecutionProfile` is the single validation/normalization site for execution profiles; `startTaskExecutionProfile` only loads the persisted (or default `inherit`) profile, so docs that say "validation runs at every session start" are incorrect.
- The PUT request schema in `openapi/agh.json` lists `coordinator`/`worker`/`sandbox`/`participants`/`review`/`task_id`/`created_at`/`updated_at` as required, but the daemon decodes empty objects into zero values and then normalizes them to their defaults; the docs must say "full replace" rather than "partial update".
- Site Vitest `runtime-autonomy-docs.test.ts` is the right place for docs checklist evidence; positive substring assertions cover precedence/CLI/HTTP/native/web/config-lifecycle without coupling to generated CLI page wording.

## Files / Surfaces Touched

- `packages/site/content/runtime/core/autonomy/execution-profiles.mdx` — new narrative page (audited and corrected for runtime truth).
- `packages/site/content/runtime/core/autonomy/index.mdx` — added autonomy navigation row + related-pages link to the profiles page.
- `packages/site/content/runtime/core/autonomy/meta.json` — registered `execution-profiles` in the Fumadocs page list.
- `packages/site/content/runtime/core/configuration/config-toml.mdx` — added `[task.orchestration]`, `[task.orchestration.profile]`, `[task.orchestration.review]` reference tables, defaults, validation-timing note.
- `packages/site/lib/runtime-autonomy-docs.test.ts` — added checklist coverage for the new page and for the generated `agh task profile` CLI references.

## Errors / Corrections

- Initial draft from the prior delegated run claimed "validation runs ... at session start" in `config-toml.mdx`. Corrected to "validation runs in `task.Service` when a profile is created or updated; session start loads the persisted profile without re-running validation".
- Initial draft suggested that omitting a block in PUT preserved previously persisted state. Corrected to a full-replace narrative.
- Initial assertion `expect(profiles).not.toContain("metadata_json")` blocked the legitimate "no `metadata_json`" sentence in the profile-shape paragraph; replaced with the positive checklist coverage already present.

## Verification Evidence

- `cd packages/site && bun run source:generate` → `[MDX] generated files in 27.30ms` PASS.
- `cd packages/site && bun run content:generate` → `[VELITE] build finished in 552.82ms` PASS.
- `cd packages/site && bun run typecheck` → `tsgo --noEmit` PASS (exit 0).
- `cd packages/site && bun run test` → Vitest 74 files / 256 tests PASS.
- `cd packages/site && bun run build` → `next build` PASS (full SSG run, all runtime/protocol/blog routes generated).
- `compozy tasks validate --name orch-improvs --format json` → `{"ok": true, "scanned": 32, "issues": null}`.
- `git diff --check` → clean.
- `make verify` → `DONE 8283 tests in 19.010s`, `OK: all package boundaries respected`.

## Ready for Next Run

- task_29 (Site Docs for Review Gate, Bundled Skills, and Notification Cursors) can begin. The profiles page intentionally keeps review-gate detail minimal so task_29 owns the canonical review narrative.
- Selector text can reuse the precedence frame established here (config defaults → persisted profile → claim eligibility → session start load → coordinator routing → continuation runs).
- The new docs test (`runtime-autonomy-docs.test.ts`) is now the checklist anchor for any further changes to execution-profile/profile-CLI documentation.
