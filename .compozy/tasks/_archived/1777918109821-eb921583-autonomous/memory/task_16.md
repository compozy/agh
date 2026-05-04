# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add runtime autonomy documentation and generated CLI references for the local autonomy MVP: coordinator bootstrap, manual execution boundary, task claim/lease, coordination channels, safe spawn, config, and autonomy hooks.
- Implementation, verification, tracking updates, and local commit are complete.

## Important Decisions
- Use the repository CLI doc generator (`make cli-docs`) rather than hand-editing generated CLI reference pages.
- Update Cobra `Example` text for new agent-facing commands where needed so generated CLI pages include accurate examples.
- Place conceptual docs under `packages/site/content/runtime/core/autonomy/` and keep broad dashboards/network evolution/eval/replay out of scope.
- Add a Vitest content test that reads the generated CLI docs and conceptual MDX so stale CLI references and weakened autonomy claims fail during site verification.

## Learnings
- The existing CLI reference is generated from Cobra via `go run ./cmd/agh doc --output-dir packages/site/content/runtime/cli-reference`.
- Pre-change docs have no `runtime/core/autonomy` section, and the generated `agh task` page still omits the implemented `next`, `heartbeat`, `complete`, `fail`, `release`, `publish`, `start`, and `approve` commands.
- Coordinator config precedence from the spec is workspace override, global `[autonomy.coordinator]`, then bundled/default coordinator agent definition.
- Cobra treats backticks in flag usage as a metavar hint; `Raw claim token from \`agh task next\`` generated a misleading `--claim-token agh task next` option. The usage text was changed to plain ASCII prose before regenerating CLI docs.
- `make cli-docs` rewrites generated CLI pages from Cobra and would drop manual tails on generated leaves. Existing observability health guidance was moved into `agh observe health` Cobra `Long`/`Example` fields so the generated page remains source-owned.

## Files / Surfaces
- Docs surfaces touched: `packages/site/content/runtime/core/autonomy/`, `packages/site/content/runtime/core/meta.json`, configuration docs, hooks docs, sessions/agents/network cross-links, generated CLI reference pages.
- CLI source surfaces touched for generated examples: `internal/cli/agent_kernel.go`, `internal/cli/task.go`, `internal/cli/spawn.go`.
- Added docs test surface: `packages/site/lib/runtime-autonomy-docs.test.ts`.
- Also touched `internal/cli/observe.go` to preserve existing observability-health CLI guidance through the generator.
- Site static export touched `packages/site/app/opengraph-image.tsx`, `packages/site/app/robots.ts`, and `packages/site/app/sitemap.ts`.

## Errors / Corrections
- Site build surfaced a Next static-export route issue outside autonomy docs: `/opengraph-image` and `/robots.txt` required explicit static route config under `output: "export"`. Added `dynamic = "force-static"` to `opengraph-image.tsx`, `robots.ts`, and `sitemap.ts` to keep metadata routes compatible with export builds.

## Verification Evidence
- `make cli-docs` passed and regenerated `packages/site/content/runtime/cli-reference`.
- `cd packages/site && bun run source:generate` passed.
- `cd packages/site && bun run typecheck` passed.
- `cd packages/site && bun run test` passed, including `runtime-autonomy-docs.test.ts`.
- `cd packages/site && bun run build` passed and generated 242 static pages.
- `env -u FORCE_COLOR make verify` passed: web format/lint/test/build/typecheck, Go lint, 6280 Go tests, and package-boundary checks.
- Local commit created: `a6000932 docs: add runtime autonomy references`.
- Post-commit `env -u FORCE_COLOR make verify` also passed after commit-hook formatting.
- Post-commit `packages/site` source generation, typecheck, test, and build also passed after commit-hook formatting.

## Ready for Next Run
- Task 16 is complete. Tracking-only files and workflow memory stayed out of the implementation commit.
