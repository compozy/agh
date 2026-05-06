# Task Memory: task_23

## Objective Snapshot

- Implemented the bounded task context bundle as `task.ContextBundle` and exposed it through `/agent/context`.
- Threaded rendered task context overlays into worker task sessions, review-router-created reviewer sessions, and coordinator sessions.
- Added review-continuation projection with review lineage, missing work, next-round guidance, and review history.
- Added redaction for raw `claim_token` JSON fields and raw `agh_claim_*` bearer-token values in task context text and event payloads.

## Important Decisions

- The Go type is named `task.ContextBundle` instead of `task.TaskContextBundle` to satisfy the repo's zero-warning lint policy while keeping the wire field name `bundle`.
- `situation.TaskStore` now explicitly includes the read models needed to assemble context: task/run, events, execution profile, and run review reads.
- Active worker sessions keep the existing task + lease projection and now also receive the normalized bundle.
- Reviewer sessions without an active worker lease receive review-bound task context through `LookupRunReviewBySession`; they do not receive a worker lease.
- Reviewer and coordinator prompt overlays prepend the same rendered task context bundle to their role-specific instructions rather than introducing separate prompt formats.
- Event payload redaction preserves non-secret fields, removes exact raw `claim_token` keys, and redacts raw claim-token values inside strings.

## Files / Surfaces

- `internal/task/context.go`
- `internal/api/contract/agents.go`
- `internal/situation/task_context.go`
- `internal/situation/service.go`
- `internal/situation/service_test.go`
- `internal/daemon/task_runtime.go`
- `internal/daemon/task_runtime_test.go`
- `internal/daemon/review_router.go`
- `internal/daemon/review_router_test.go`
- `internal/daemon/coordinator_runtime.go`
- `internal/daemon/coordinator_runtime_test.go`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`
- `packages/site/content/runtime/api-reference/agent.mdx`

## Errors / Corrections

- Focused tests first exposed a compile error from the in-progress partial implementation: `activeRunForSession` returned the two-value `selectActiveRun` result despite its three-value signature. It now returns `(run, ok, nil)`.
- Existing render-order tests searched for `"limits"` broadly; adding `bundle.limits` made them match the nested bundle field before the top-level section. Tests now assert the top-level `limits` key.
- `make lint` rejected stuttering exported names (`TaskContextRequest`, `TaskRuntimeLimits`, `TaskContextBundle`) and an unused review-binding helper. The types were renamed to `ContextRequest`, `RuntimeLimits`, and `ContextBundle`; the unused helper was removed.
- The first full `make verify` stopped at `codegen-check` because the contract change required regenerated OpenAPI and TypeScript artifacts. `make codegen` regenerated the artifacts, and `make codegen-check` passed.

## Ready for Next Run

- `task_24` can consume the context bundle and current task/run/event projections when adding latest-event-sequence and cursor-seeded SSE.
- Web tasks should consume the generated `/agent/context` bundle shape from `web/src/generated/agh-openapi.d.ts` instead of duplicating DTOs.
- Docs tasks should describe the bundle as bounded/redacted context and should not imply raw claim tokens or unrestricted transcripts are exposed.
- The `.pyc` artifact remains unresolved and still requires an explicit user decision before cleanup.

## Verification Evidence

- `go test ./internal/situation -run 'Test(ContextForSession|ContextBundle)' -count=1` passed.
- `go test ./internal/daemon -run 'Test(TaskSessionBridgeStartTaskSessionInjectsTaskContextOverlay|ReviewRouterRoutesRunReviewRequests|CoordinatorRuntimeBootstrapsWithTaskContextOverlay)' -count=1` passed.
- `go test ./internal/situation -count=1` passed.
- `go test ./internal/daemon -count=1` passed.
- `make lint` passed with `0 issues`.
- `make codegen` regenerated OpenAPI and TypeScript artifacts.
- `make codegen-check` passed.
- `make web-typecheck` passed.
- `make web-test` passed with 202 files / 1525 tests.
- `packages/site` `bun run typecheck`, `bun run test`, and `bun run build` passed.
- `make verify` passed: Bun lint/typecheck/test, Vitest 329 files / 2092 tests, web build, `golangci-lint` 0 issues, Go race gate `DONE 8276 tests in 130.651s`, and package boundaries OK.
