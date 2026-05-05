# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 09: add `agh__autonomy` tools for claim/heartbeat/complete/fail/release and hard-cut AGH-owned CLI/HTTP/UDS/OpenAPI/web/docs contracts away from raw `claim_token` in favor of session-bound `run_id` lookup.
- Acceptance requires preserving existing task-service lease writers as the only write authority, keeping raw tokens out of public payloads/docs/generated types, adding stale/foreign/double/missing lease and redaction coverage, regenerating contracts, and passing focused checks plus `make verify`.

## Important Decisions
- No compatibility bridge: raw-token public request/response fields are residual-check targets per ADR-005 and TechSpec "Post-Implementation Residual Checks".
- `claim_token_hash` remains allowed only as observability metadata and must not become an accepted credential.
- Session-bound lookup will use the existing `task_runs.claim_token` column as internal-only active lease state. Public `Get/ListTaskRun` projections already select `'' AS claim_token`, so the implementation must preserve that redaction while changing claim/heartbeat writes to keep the raw token available for internal lookup until release/terminal/recovery cleanup.

## Learnings
- Baseline search confirms raw `claim_token` still exists in current code/generated/docs for agent task claim/mutation surfaces, so implementation work is required even though shared memory mentions a prior QA hard cut.
- Current worktree has unrelated modified instruction/skill files and an untracked `.compozy/tasks/tools-refac/` bundle; implementation must avoid reverting or staging unrelated changes.
- Existing authoritative writers are `ClaimNextRun`, `HeartbeatRunLease`, `ReleaseRunLease`, `CompleteRunLease`, and `FailRunLease`; heartbeat currently clears `claim_token`, which would break session-bound follow-up mutations after the first heartbeat and must change.
- The hard cut can reuse the existing `task_runs.claim_token` column as internal-only active lease state because public GlobalDB read projections already mask it with `'' AS claim_token`.
- Full package `go test -tags integration ./internal/api/udsapi` is still blocked by pre-existing non-Task-09 failures (`TestUDSToolResourceCRUDRoundTripTriggersProjection` shape drift and observe parity timeout). Focused UDS/API unit coverage and focused CLI integration autonomy flows are the task evidence for now.
- Full `make verify` passed after implementation and lint cleanup with Bun lint/typecheck/test, web build, Go lint, Go test (`DONE 7080 tests`), build, and package boundaries.
- Local code commit `1119d6e4 feat: add session-bound autonomy tools` was created. Post-commit `make verify` passed with Go lint `0 issues`, Go tests `DONE 7080 tests`, and package boundaries respected.

## Files / Surfaces
- Expected: `internal/task`, `internal/api/contract/agents.go`, `internal/api/core/agent_tasks.go`, `internal/api/spec/spec.go`, `internal/cli/task.go`, native tool descriptors/handlers, generated OpenAPI/web types, task fixtures, site autonomy/CLI docs, and focused tests.
- Touched so far: `internal/task`, `internal/store/globaldb`, `internal/api/contract`, `internal/api/core`, `internal/api/testutil`, `internal/api/udsapi`, `internal/cli`, `internal/daemon`, `internal/tools`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `packages/site/content/runtime/*`, and `packages/site/lib/runtime-autonomy-docs.test.ts`.

## Errors / Corrections
- Corrected old CLI/UDS tests that still sent `--claim-token` / `claim_token` public payloads; mutation tests now assert session-bound lookup supplies the internal token to the existing writers.
- Regenerated CLI reference docs after removing `--claim-token`; generated pages now describe session-bound task heartbeat/complete/fail/release commands.
- First full `make verify` attempt failed at Go lint on `funlen`, `gocyclo`, and error-return ordering in the new autonomy lookup/store/native-tool helpers. Fixed by extracting store row-scan/timestamp helpers, splitting lease lookup classification helpers, and returning `error` last from the daemon reason mapper. Follow-up `go test ./internal/task ./internal/store/globaldb ./internal/daemon -count=1` and `make lint` passed.

## Ready for Next Run
- Task 09 is complete in this session. Code commit exists locally as `1119d6e4`; tracking/memory artifacts remain uncommitted per workflow guidance.
