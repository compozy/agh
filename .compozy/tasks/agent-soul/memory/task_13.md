# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 13 extensibility surfaces for authored context: extension Host API actions/grants, typed hook payloads, agent-operable tools/resources, TypeScript SDK helpers, and tests for Soul, Heartbeat, session health, and wake audit/status.
- Success requires exact grant enforcement, managed service mutation paths, no direct file-write or task-claim bypass, focused tests plus final `make verify`, tracking updates, and one local commit only after clean verification.

## Important Decisions
- Body-level `expected_digest` remains the CAS contract for Soul/Heartbeat mutation across Host API and SDK helpers.
- Host API actions must be exact and separately grant read, validate/history, manage, status, wake, session-health read, and wake-audit read capabilities.
- Implement Host API adapters directly against the managed Soul/Heartbeat/session-health/wake services already composed for Task 11. Do not introduce Host-API-only authoring, filesystem mutation, wake scheduling, or task claim semantics.
- Keep Soul out of `internal/resources`, MCP tools, and canonical built-in tool registry for MVP because the Soul TechSpec explicitly forbids those entries. Heartbeat/session-health read/status tooling can be added only through managed services with no task-claim or direct file-write authority.
- Hook payload work should add Soul snapshot/digest fields to existing session/task/spawn contexts and add typed Heartbeat wake/session-health observation payloads that include digest/status/reason/provenance but no raw prompt or secret data.
- New authored-context hook events must be wired into hook introspection and SDK hook payload maps, not only `hooks.AllHookEvents`; otherwise generated TypeScript `HookPayloadByEvent` no longer covers every `HookEvent`.
- Native tool exposure is limited to `agh__session_health`, `agh__agent_heartbeat_status`, and `agh__agent_heartbeat_wake` in `agh__authored_context`; all three call managed services and do not expose Soul desired-state mutation.

## Learnings
- Shared workflow memory handoff says Task 13 should call Task 11 HTTP/UDS route surface or underlying services with the same body-level `expected_digest` contract; do not add Host-API-only authoring behavior.
- Task 12 added operator-scoped Soul validate route and CLI command parity; Host API/docs should include Soul validate alongside inspect/write/delete/history/rollback.
- TechSpecs/ADRs require no direct `SOUL.md` or `HEARTBEAT.md` writes from extensions, hooks, tools/resources, bundles, MCP sidecars, bridge adapters, or web code.
- Current Host API has no authored-context method constants/specs/grants/handlers and daemon extension wiring does not inject authored-context services into `NewHostAPIHandler`.
- Task 10 contract DTOs already define the required redacted Soul, Heartbeat, session health, wake state/event/decision, and mutation response shapes, so Host API should alias/reuse those DTOs instead of defining new wire shapes.
- Task 11 `internal/api/core` has unexported helper logic; Host API will mirror only target resolution/conversion glue while calling the same service interfaces.
- Daemon-level wrappers now emit bounded observation hooks around managed Soul validation/mutation, Heartbeat validation/status, surfaced Heartbeat wake decisions, and persisted session-health transitions. Hook errors are logged and do not create a direct mutation/claim authority.
- SDK Go had only string constants, so Task 13 adds authored-context Host API constants there while TypeScript gets typed helpers and generated `HostAPIMethodMap` coverage.
- `bun run --cwd sdk/typescript typecheck` initially exposed the missing hook introspection/codegen map entries for new authored-context hook events; fixing `internal/hooks/introspection.go` and `internal/extension/contract/sdk.go` restored type safety.

## Files / Surfaces
- Touched production/test/codegen surfaces: `internal/extension/protocol/host_api.go`, `internal/extension/contract/{host_api.go,sdk.go}`, `internal/extension/host_api*.go`, `internal/extension/capability.go`, `internal/daemon/{authored_context_runtime.go,daemon.go,boot.go,hooks_bridge.go,native_tools.go}`, `internal/hooks/{events.go,introspection.go,payloads.go,dispatch.go,matcher.go}`, `internal/session/{health.go,health_test.go,hooks.go,manager*.go,spawn.go}`, `internal/task/{manager.go,lease_manager.go}`, `internal/tools/builtin*`, `sdk/typescript/src/{host-api.ts,host-api.test.ts,authored-context-contracts.test.ts,generated/contracts.ts}`, `sdk/go/host_api.go`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`.

## Errors / Corrections
- Pre-existing dirty/untracked files before implementation: multiple `.compozy/tasks/agent-soul/task_*.md`, `_tasks.md`, `.compozy/extensions/`, and `.compozy/tasks/agent-soul/memory/`. Do not revert or stage unrelated changes.
- TypeScript typecheck failed when `HookEvent` included new authored-context events but `HookPayloadByEvent`/`HookPatchByEvent` did not; root cause was missing hook introspection and SDK contract registry entries. Fixed at the source and regenerated contracts.
- `make lint` failed on `hugeParam` after adding Soul provenance to hook session context. Fixed without changing exposed JSON names by embedding optional `SessionSoulContext` inside `SessionContext`, and by passing large Soul/Heartbeat hook helper results/policies by pointer.
- The embedded context shape caused generated TypeScript contract drift; ran `make codegen` and confirmed `make codegen-check` is green again.

## Ready for Next Run
- Implementation is applied and Task 13 tracking is marked complete. Focused validation passed:
  - `go test ./internal/hooks ./internal/extension/contract ./internal/extension/protocol ./internal/extension ./internal/session ./internal/daemon ./internal/tools/builtin ./sdk/go -count=1`
  - `bun test sdk/typescript/src/host-api.test.ts sdk/typescript/src/authored-context-contracts.test.ts`
  - `bun run --cwd sdk/typescript typecheck`
  - `make codegen-check`
- `make lint` and a post-fix focused `go test ./internal/hooks ./internal/session ./internal/daemon ./internal/tools/builtin -count=1` are green.
- `make verify` passed end-to-end before commit: Bun lint/typecheck/test/build, Go fmt/lint/test/build, package boundaries; Bun test reported 264 files and 1874 tests, Go test reported 7706 tests.
- Created local commit `a7ff03bb` (`feat: add authored context extension surfaces`) containing only Task 13 code/test/generated files. Tracking and workflow-memory files remain unstaged by policy.
- Post-commit `make verify` passed: Bun lint found 0 warnings/0 errors, Bun tests reported 264 files and 1874 tests, web build completed, Go lint reported 0 issues, Go tests reported 7706 tests, and package boundaries were OK.
- Task 13 is complete for this session. Remaining uncommitted files are tracking/memory updates plus pre-existing `.compozy` task changes and `.compozy/extensions/`.
