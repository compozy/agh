# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Task 10 implements the shared contract/codegen layer only: Go DTOs/enums/redaction/conversion tests, OpenAPI schema registration, and generated web/SDK TypeScript surfaces for Soul, Heartbeat, session health, wake state/events, authoring mutations, diagnostics, provenance, and read models.
- Success requires `expected_digest` as body-level CAS on mutation DTOs, closed enums for health/wake/diagnostics/validation status, redaction of raw secrets/provider tokens/claim tokens/disallowed prompt data, generated artifacts current, and clean verification before tracking/commit.
- Status: complete in local commit `1ad85048`; post-commit `make verify` passed.

## Important Decisions
- Route/business logic remains out of scope for task 10 except schema/codegen plumbing needed to expose stable DTOs for task 11+.
- No conflict found between task_10, ADR-002/006/010/011, aggregate `_techspec.md`, `_techspec_soul.md`, and `_techspec_heartbeat.md`.
- Compact `/api/agent/context` Soul projection is now `AgentSoulSectionPayload`; full `AgentSoulPayload` is reserved for dedicated inspect/authoring surfaces.
- SDK TypeScript contract generation is extended by registering API DTOs as root types only; Host API methods/actions stay deferred to task 13.
- OpenAPI registers future HTTP/UDS operations for Soul, Heartbeat, session health/status/inspect, and wake request/response contracts only; route handlers/business logic remain task 11 scope.
- `SessionPayload` now has optional `health` so `/api/sessions?include_health=true` can share the same session-health DTO once route wiring lands.

## Learnings
- Shared workflow memory says task 10/14 may expose wake status/audit from persisted wake events/state and synthetic metadata fields (`wake_event_id`, `policy_snapshot_id`, `policy_digest`, `config_digest`) without introducing queue ownership semantics.
- Task 09 completed advisory wake service semantics; contracts must preserve result/reason enums and no-task-queue framing.
- `internal/heartbeat` already owns the canonical closed enums for wake/session-health behavior; task 10 mirrors them in transport DTO enums instead of exposing domain-package types directly.
- Generated OpenAPI represents pointer fields as nullable in web operation types; TypeScript smoke tests account for `null | undefined` on generated pointer fields.
- `make verify` exposed an unrelated harness lifecycle bug in `internal/extension/host_api_test.go`: `newHostAPITestEnv` opened the global DB under `t.TempDir()` without a happy-path close, allowing `TempDir RemoveAll cleanup: directory not empty` under race/parallel package execution. The fix registers `registry.Close` as a cleanup that runs after sessions, automation, and observer cleanup.

## Files / Surfaces
- Contract/codegen surfaces touched: `internal/api/contract/authored_context.go`, `internal/api/contract/authored_context_test.go`, `internal/api/contract/agents.go`, `internal/api/contract/contract.go`, `internal/api/spec/authored_context.go`, `internal/api/spec/authored_context_test.go`, `internal/api/spec/spec.go`, `internal/extension/contract/sdk.go`, `internal/codegen/sdkts/generate.go`, `internal/situation/service.go`, `internal/situation/render.go`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `web/src/lib/agent-authored-context-contract.test.ts`, `sdk/typescript/src/generated/contracts.ts`, `sdk/typescript/src/authored-context-contracts.test.ts`.
- Verification cleanup touched: `internal/extension/host_api_test.go`.
- Tracking touched: `.compozy/tasks/agent-soul/task_10.md`, `.compozy/tasks/agent-soul/_tasks.md`.

## Errors / Corrections
- First `make verify` failed in `internal/extension TestHostAPIHandlerSessionsEventsSupportsSinceFilter` with `TempDir RemoveAll cleanup: directory not empty`.
- Root cause found in the shared host API test harness: unclosed `globaldb` handle in `newHostAPITestEnv`.
- Corrected by registering `registry.Close(testutil.Context(t))` via `t.Cleanup`; removed the old `_ = registry.Close(...)` error discard on the `observe.New` failure path.
- `.agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/host_api_test.go` still reports many pre-existing convention violations across that large legacy test file; not remediated in Task 10. Focused new task test files pass the convention checker.

## Ready for Next Run
- Final evidence: `make codegen`, `make codegen-check`, focused Go/TS tests, `make bun-typecheck`, `make lint`, `make bun-lint`, `make bun-test`, focused extension race tests, pre-commit `make verify`, and post-commit `make verify` all passed.
- Local task commit: `1ad85048 feat: add agent soul contract surface`.
- Post-commit `make verify` evidence: Go lint `0 issues`; Go test lane `DONE 7675 tests in 13.022s`; boundaries `OK: all package boundaries respected`; Vite emitted only the existing chunk-size warning.
- Remaining dirty files are tracking/memory/pre-existing task artifacts only; no code/generated/test changes remain unstaged after commit.
