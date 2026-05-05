# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement agent-facing task lease verbs over UDS and CLI: claim-next, heartbeat, complete, fail, release.
- Preserve existing operator `agh task ...` and `agh task run ...` flows.
- Required evidence: focused UDS/CLI/unit/integration coverage, raw-token redaction checks, permission/identity denial checks, stale token/reconnect flow checks, and `make verify`.

## Important Decisions
- Reuse existing Task 02 contract DTOs and Task 08 service methods; no new public HTTP/OpenAPI DTOs are planned unless implementation proves a gap.
- UDS claim-next should return HTTP 204 when no work is claimable; CLI maps that to stable JSON `{ "claimed": false }`.
- Caller session/workspace/agent identity comes from Task 05 `agentidentity` resolution; the CLI must not implement claim rules directly.
- Raw claim tokens may appear only in claim-next success payloads and command request inputs. Shared API/CLI error formatting will redact `agh_claim_*` tokens defensively.
- Lease complete/fail result metadata must reject raw `claim_token` fields at the task-domain validation layer, not only at CLI validation.
- Workspace-scoped queued runs with a network channel now persist the same value as `coordination_channel_id`, so claim responses can include durable channel metadata for `--channel`-bound work.

## Learnings
- `internal/api/contract` already defines `AgentTask*` request/response DTOs and OpenAPI already includes `/api/agent/tasks/*`.
- `internal/task.Service` already owns authoritative `ClaimNextRun`, token-fenced lease mutation methods, coordination channel metadata, and expired lease recovery.
- `session.Info` does not carry declared agent capabilities; agent task claim can infer session/workspace/agent identity directly and can opportunistically union capabilities from `AgentContextService` when available.
- `make verify` exercises web format/lint/typecheck/test/build and Go lint/test/build/package-boundary checks; no extra generated contract update was needed for Task 09 because the DTOs were already present.

## Files / Surfaces
- Implemented code: `internal/api/core/agent_tasks.go`, `internal/api/core/errors.go`, `internal/api/udsapi/routes.go`, `internal/cli/client.go`, `internal/cli/task.go`, `internal/task/lease.go`, `internal/task/validate.go`, `internal/task/manager.go`, `internal/store/globaldb/global_db_task_aux.go`.
- Implemented tests: `internal/api/udsapi/agent_tasks_test.go`, `internal/cli/task_test.go`, `internal/cli/client_test.go`, `internal/cli/agent_kernel_test.go`, `internal/cli/cli_integration_test.go`, `internal/task/*_test.go`, `internal/store/globaldb/global_db_task_claim_test.go`.

## Errors / Corrections
- Initial CLI integration exposed that `--channel` enqueue stored only `network_channel`; fixed production enqueue persistence to also bind workspace queued runs to `coordination_channel_id`.
- First `make verify` failed on `gocritic hugeParam` and `lll` in `internal/api/core/agent_tasks.go`; changed claim payload conversion to accept a pointer and split the actor literal, then reran verification.
- Self-review found missing explicit stale-token-after-recovery coverage; expanded `TestCLIAgentTaskLeaseLifecycleIntegration` to restart the daemon after lease expiry and prove old tokens cannot `release` or `fail`.

## Ready for Next Run
- Task 09 implementation and tracking updates are complete. Verification evidence:
  - `go test ./internal/task ./internal/store/globaldb ./internal/api/core ./internal/api/udsapi ./internal/cli -count=1` passed.
  - `go test -tags integration ./internal/cli -run TestCLIAgentTaskLeaseLifecycleIntegration -count=1` passed after the recovery coverage addition.
  - Fresh `make verify` passed with `0 issues`, `DONE 6178 tests`, and package boundaries respected.
