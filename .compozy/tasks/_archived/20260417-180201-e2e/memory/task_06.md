# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add composition-root runtime E2E coverage for local-environment tool execution and sandbox denial, plus the shared harness/artifact seams needed to assert those outcomes through public and persisted surfaces.

## Important Decisions
- Reused the existing subprocess-backed `internal/testutil/e2e` harness instead of adding a second environment-specific runtime harness.
- Seeded the runtime config with named sandbox profiles and `defaults.sandbox` so tests exercise the real sandbox registry and session startup path.
- Drove the runtime scenarios with a helper ACP agent embedded in the integration test binary. The same helper runs an allowed or blocked filesystem action depending on the scenario and the seeded agent permissions.
- Captured environment truth from both the public session payload and persisted session metadata, with tool-host diagnostics stored as explicit allowed/blocked operation records.
- Explicitly stopped sessions through the public UDS surface after the ACP turn so the stopped-state sandbox metadata becomes deterministic for assertions and artifact capture.

## Learnings
- The local-provider sandbox denial is stable to assert via three surfaces together: missing host-side file, persisted/public sandbox metadata, and the agent-visible tool error containing the `approve-reads` restriction.
- `internal/testutil/e2e` does not clear the 80% bar on unit-only coverage because the useful behavior is runtime-heavy; integration-inclusive coverage is the meaningful package measure here.
- Reusing the existing `provider_calls.json` artifact slot for tool-host diagnostics keeps failure manifests compact while still surfacing the environment-specific action outcomes needed for debugging.

## Files / Surfaces
- `internal/testutil/e2e/config_seed.go`
- `internal/testutil/e2e/config_seed_test.go`
- `internal/testutil/e2e/artifacts.go`
- `internal/testutil/e2e/artifacts_test.go`
- `internal/testutil/e2e/runtime_harness.go`
- `internal/testutil/e2e/runtime_harness_helpers_test.go`
- `internal/daemon/daemon_environment_sandbox_integration_test.go`
- Public `/api/sessions/:id` reads and `/api/sessions/:id` stop requests over UDS
- Persisted `store.SessionMetaFile(...)` metadata and tool-host workspace side effects under the seeded runtime root

## Errors / Corrections
- Initial runtime tests hung because a user session stays active after a prompt completes; fixed by adding a public `StopSession` helper and waiting for the stopped state explicitly.
- The stopped-state assertion originally expected a completed stop reason, but explicit stop requests are recorded as `store.StopUserCanceled`; tests now assert the real persisted behavior.

## Ready for Next Run
- Final committed code is in `9bc89054` (`test: add local sandbox sandbox e2e`).
- Verification completed on the committed tree:
  - `go test ./internal/testutil/e2e`
  - `go test -tags integration -cover ./internal/testutil/e2e`
  - `go test -tags integration -cover ./internal/daemon`
  - `make verify`
- Tracking and workflow-memory updates were intentionally left unstaged per task instructions; current worktree dirt outside the commit is limited to `.compozy/tasks/e2e` tracking/memory artifacts plus pre-existing unrelated task files.
