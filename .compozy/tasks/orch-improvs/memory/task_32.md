# Task 32 Memory - Real-Scenario QA Execution

## Decisions

- Used a fresh isolated QA lab for this QA execution instead of reusing older labs. The manifest is persisted at `qa/bootstrap-manifest.json` and the copied environment is `qa/bootstrap.env`.
- Treated Browser Use as unavailable after tool discovery did not expose a callable browser-use action; used daemon-served Playwright for web behavior evidence.
- Did not claim live external bridge/provider delivery. The isolated lab exposed Claude Code auth status, but no eligible worker agent or bridge provider existed for true live delivery; those steps are documented as blocked boundaries in `qa/verification-report.md`.
- Fixed reproduced regressions at their source instead of weakening QA assertions. All bug reports are under `qa/issues/BUG-001..BUG-008*.md`.

## Implementation Notes

- Runtime scenario covered task creation, execution profile inspect/update, active-run profile mutation rejection, run enqueue/claim/start failure, review request/no-route diagnostics, native tool policy, bridge notification diagnostics, and task SSE replay.
- Web scenario covered the daemon-served Playwright suite, including the Orchestration tab on a real seeded task.
- Docs scenario covered runtime-autonomy docs tests plus full site source/content/typecheck/test/build validation.
- Root-cause fixes landed in:
  - `internal/api/core/bridges.go`
  - `internal/api/core/tasks_test.go`
  - `internal/daemon/daemon.go`
  - `internal/daemon/daemon_test.go`
  - `internal/session/query.go`
  - `internal/session/query_test.go`
  - `internal/testutil/acpmock/fixture.go`
  - `internal/testutil/acpmock/cmd/acpmock-driver/main.go`
  - `internal/testutil/acpmock/cmd/acpmock-driver/main_test.go`
  - `internal/testutil/acpmock/testdata/*.json`
  - `internal/api/httpapi/httpapi_integration_test.go`
  - `internal/api/udsapi/udsapi_integration_test.go`
  - `internal/api/udsapi/transport_parity_integration_test.go`
  - `web/e2e/bridges.spec.ts`

## Validation Evidence

- `go test ./internal/api/core -run 'TestBaseHandlersTaskBridgeNotificationSubscription' -count=1`: PASS.
- `go test -race ./internal/api/core -run 'TestBaseHandlersTaskBridgeNotificationSubscription' -count=1`: PASS.
- `go test ./internal/daemon -run TestNewHostAPISessionManagerAdapter -count=1`: PASS.
- `go test ./internal/testutil/acpmock ./internal/testutil/acpmock/cmd/acpmock-driver -count=1`: PASS.
- `go test ./internal/session -run '^TestManagerOpenQueryRecorderValidationAndCleanup$' -count=1`: PASS.
- `go test -race ./internal/session -run '^TestManagerOpenQueryRecorderValidationAndCleanup$' -count=1`: PASS.
- `go test -race -parallel=4 -count=5 -tags integration -run '^TestDaemonE2EACPmockCrashMidStreamProjectsRuntimeFailure$' ./internal/daemon`: PASS.
- `make test-e2e-runtime`: PASS; evidence `qa/evidence/gates/make-test-e2e-runtime-final-pass.txt`.
- `make test-e2e-web`: PASS, 20 Playwright tests; evidence `qa/evidence/gates/make-test-e2e-web-after-fixes.txt`.
- Runtime autonomy docs Vitest: PASS; evidence `qa/evidence/docs/runtime-autonomy-docs-vitest.txt`.
- Full site validation: PASS; evidence `qa/evidence/docs/site-full-validation.txt`.
- Final `make verify`: PASS; evidence `qa/evidence/gates/make-verify-final.txt`.

## Follow-Up

- Continue Phase D review rounds next. CodeRabbit clean streak remains 0/3 until the review loop runs.
- Do not set `progress.deliverables_complete=true` until Phase D reaches three consecutive clean rounds and Phase E final verification passes.

