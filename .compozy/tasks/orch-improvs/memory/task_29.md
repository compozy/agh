# Task Memory: task_29

## Status

Completed 2026-05-05 after mandatory Compozy -> Claude Opus docs-lane delegation, local
truth-audit corrections, site validation, and full `make verify`.

## Objective Snapshot

- Author `packages/site` runtime docs for the review gate, bundled orchestration skills, and bridge notification cursor diagnostics.
- Keep review verdict authority in `task.Service.RecordRunReview`; channels, bridge messages, skills, notification cursors, and web UI are not authority.
- Cover agent-operable CLI, HTTP, UDS, native tool, web UI, and generated-reference paths without hand-editing generated CLI/API reference pages.

## Important Decisions

- task_29 is a docs task, so implementation must be delegated through `compozy exec --ide claude --model opus --prompt-file ...`; local Codex remains orchestrator/auditor.
- Review-gate docs should be narrative and operator-facing, while generated CLI/API references remain the exact command/contract source.
- Notification cursor docs must describe confirmed delivery progress and diagnostics only, not task stream authority or review workflow state.

## Learnings

- Claude Opus wrote the first docs pass but stalled before returning evidence or updating tracking;
  local audit was required before accepting the work.
- The local audit caught and corrected false docs claims: non-existent review events
  (`task.run_review_routed`, `task.run_review_circuit_opened`, `task.run_review_canceled`), a
  broken `/runtime/core/agent/context` link, no-route as `error` instead of implemented `blocked`,
  run-detail bridge notification claims, literal "No delivery yet" UI text, and public cursor-reset
  wording.
- Site metadata tests timed out under the full suite because they imported the full blog/changelog
  pages only to inspect static metadata. The fix split static metadata into lightweight modules and
  kept the pages exporting the same metadata.

## Files / Surfaces Touched

- `packages/site/content/runtime/core/autonomy/review-gate.mdx`
- `packages/site/content/runtime/core/autonomy/notification-cursors.mdx`
- `packages/site/content/runtime/core/autonomy/index.mdx`
- `packages/site/content/runtime/core/autonomy/meta.json`
- `packages/site/content/runtime/core/skills/bundled.mdx`
- `packages/site/content/runtime/core/configuration/config-toml.mdx`
- `packages/site/lib/runtime-autonomy-docs.test.ts`
- `packages/site/lib/static-route-metadata.test.ts`
- `packages/site/app/blog/metadata.ts`
- `packages/site/app/blog/page.tsx`
- `packages/site/app/changelog/metadata.ts`
- `packages/site/app/changelog/page.tsx`
- `.compozy/tasks/orch-improvs/task_29.md`
- `.compozy/tasks/orch-improvs/_tasks.md`
- `.compozy/tasks/orch-improvs/memory/MEMORY.md`

## Errors / Corrections

- First Claude docs-lane finalization stalled after writing partial docs; killed only that
  `task_29` process tree and audited the written files locally.
- `packages/site` full test initially failed on a bad config hash link; fixed by removing the
  stale `#task-orchestration-review` anchor and relying on the config page route.
- `packages/site` full test then failed on `static-route-metadata.test.ts` timeout; fixed by moving
  blog/changelog metadata into lightweight modules instead of importing full route component trees.

## Verification Evidence

- `cd packages/site && bun run source:generate` PASS.
- `cd packages/site && bun run content:generate` PASS.
- `cd packages/site && bun run typecheck` PASS.
- `cd packages/site && bunx vitest run lib/runtime-autonomy-docs.test.ts` PASS, `1` file / `15`
  tests.
- `cd packages/site && bunx vitest run lib/static-route-metadata.test.ts` PASS, `1` file / `3`
  tests.
- `cd packages/site && bun run test` PASS, `74` files / `263` tests.
- `cd packages/site && bun run build` PASS, SSG generated `1086` static pages.
- `compozy tasks validate --name orch-improvs --format json` PASS, `scanned: 32`.
- `git diff --check` PASS.
- `make verify` PASS: Vitest monorepo `339` files / `2206` tests, `golangci-lint` `0 issues`, Go
  race gate `DONE 8283 tests in 32.697s`, package boundaries OK.

## Ready for Next Run

- Next detector should move to `task_30` after `state.yaml` is updated through the loop helper.
