# Task Memory: task_21

## Objective Snapshot

- Closed the bridge notification transport consolidation slice by hard-cutting the transport contract to `/api/tasks/{id}/notifications/bridges`.
- Added single-subscription `show` parity across shared API core, HTTP, UDS, OpenAPI, generated TypeScript contracts, CLI, and generated CLI docs.
- Kept full cursor diagnostic/lifecycle expansion in task_25; task_21 exposes the durable cursor identity and route parity needed by downstream web/docs work.

## Important Decisions

- Removed the divergent `bridge-notification-subscriptions` route shape instead of keeping aliases; the project is greenfield and the TechSpec route is canonical.
- Reused `BridgeService.GetBridgeTaskSubscription` for `show`; no new authority was added in API handlers.
- Used Context7 and Exa when fixing the site OG font regression exposed by the final gate: Next.js documents `readFile(join(process.cwd(), ...))` for `ImageResponse` fonts, while Exa surfaced that Vitest workspaces do not provide independent per-project `cwd`. The runtime now keeps the Next-style `process.cwd()` default, and the site Vitest project injects `AGH_SITE_ROOT` explicitly for root-workspace tests.

## Learnings

- For package-local server assets in `packages/site`, avoid dynamic `existsSync` path probing during Next builds; it can trigger broad Turbopack/NFT tracing warnings.
- `import.meta.url` was not a reliable filesystem base in the root Vitest transform for this OG helper; it resolved to `/lib/og/...` instead of the package source path.
- Root `bunx vitest run` and package-local `bun run test` must both be exercised for site helpers that read local files, because they run with different `process.cwd()` values.

## Files / Surfaces

- `internal/api/core/bridges.go`
- `internal/api/core/tasks.go`
- `internal/api/httpapi/routes.go`
- `internal/api/udsapi/routes.go`
- `internal/api/spec/spec.go`
- `internal/cli/client.go`
- `internal/cli/task.go`
- `internal/api/core/tasks_test.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/api/udsapi/handlers_test.go`
- `internal/api/spec/spec_test.go`
- `internal/cli/task_test.go`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`
- `packages/site/content/runtime/cli-reference/task/notification/show.mdx`
- `packages/site/lib/og/fonts.ts`
- `packages/site/vitest.config.ts`

## Errors / Corrections

- `make site-build` initially passed with a Turbopack/NFT warning caused by dynamic font path probing. Replaced probing with a deterministic site-root path.
- The first deterministic path fix passed package-local tests but failed root Vitest because root tests use the monorepo `cwd`. Added `AGH_SITE_ROOT` in the site Vitest project config instead of changing process `cwd` globally.
- The first full `make verify` failed in `packages/site/lib/opengraph-image.test.tsx`; after the root-cause fix, both focused site contexts and the full gate passed.

## Ready for Next Run

- `task_22` is the next detected task after state update.
- Use `/api/tasks/{id}/notifications/bridges` for all downstream web/docs references.
- Do not reintroduce `bridge-notification-subscriptions` aliases.
- Full cursor diagnostics and notifier lifecycle state tests remain scoped to task_25.

## Verification Evidence

- `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/spec ./internal/cli -run 'TestBaseHandlersTaskBridgeNotificationSubscriptionEndpoints|TestTaskNotificationCommandsMapRequests|TestRegisterExpandedTaskAndObserveOperations|TestRegisterTaskRoutesUseSharedHandlerBindings|TestRegisterRoutesCoversTechSpecEndpoints' -count=1` passed.
- `make codegen` passed.
- `make codegen-check` passed.
- `make cli-docs` passed.
- `rg "bridge-notification-subscriptions" internal/api internal/cli openapi web/src/generated packages/site/content/runtime/cli-reference/task` returned no matches.
- `bun run test lib/opengraph-image.test.tsx` in `packages/site` passed: 1 file / 6 tests.
- `bunx vitest run packages/site/lib/opengraph-image.test.tsx` from repo root passed: 1 file / 6 tests.
- `make site-build` passed and generated 1077 static pages with no Turbopack/NFT warning after the font fix.
- `make verify` passed: Bun lint/typecheck/test, Vitest 329 files / 2092 tests, web build, `golangci-lint` 0 issues, Go race gate `DONE 8262 tests in 95.480s`, and package boundaries OK.
