# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build a test-only `internal/testutil/acpmock/` layer that lets runtime E2E tests register one or more deterministic mock ACP agents through normal AGH agent definitions and provider validation, with fixture primitives for permission/tool/network/bridge/environment flows and focused unit/integration coverage.

## Important Decisions
- `brainstorming` was reviewed but not used as a blocking approval loop because `task_02.md`, `_techspec.md`, and the ADRs already provide the approved design contract for this implementation run.
- Mock-agent `AGENT.md` files are registered before the runtime harness boots the daemon, using `RuntimeHarnessOptions.MockAgents`, because the daemon startup path must discover them through the normal catalog and provider resolution flow.
- The mock driver stays intentionally narrow: it emits deterministic ACP session events and diagnostics, while the harness and daemon remain the real system under test.

## Learnings
- The current repository state has no `internal/testutil/acpmock/` package yet, so task_02 is adding the mock layer rather than expanding an existing package in-place.
- The shared runtime harness from task_01 is already in place and is the likely integration point for fixture-backed mock agent registration.
- The direct ACP client seam is sufficient to validate network-origin environment expectations for task_02; a daemon-level environment-exec prompt currently exits with `signal: killed`, which should be treated as follow-up work for later runtime environment scenarios rather than widened here.

## Files / Surfaces
- `.compozy/tasks/e2e/task_02.md`
- `.compozy/tasks/e2e/_techspec.md`
- `.compozy/tasks/e2e/adrs/adr-001.md`
- `.compozy/tasks/e2e/adrs/adr-004.md`
- `.compozy/tasks/e2e/_tasks.md`
- `.compozy/tasks/e2e/memory/MEMORY.md`
- `internal/testutil/e2e/`
- `internal/testutil/acpmock/`
- `internal/daemon/daemon_mock_agents_integration_test.go`

## Errors / Corrections
- The task spec refers to expanding `internal/testutil/acpmock/`, but the package does not exist in the current repo snapshot; implementation must create it while preserving the required behavior.
- A first daemon-level environment-exec fixture test was removed from the final suite after showing the live daemon terminal path returned `signal: killed`; task_02 keeps environment expectations covered at the ACP/client seam and records the daemon behavior as future follow-up instead of weakening the assertions.

## Ready for Next Run
- Verification completed after implementation:
  - `make verify`
  - `go test ./internal/testutil/acpmock -cover -count=1` (`coverage: 80.3% of statements`)
  - `go test -tags integration ./internal/daemon -run 'TestDaemonE2E(FixtureBackedMockAgentLaunchesThroughNormalAgentDefinition|MockAgentsRemainIsolated|ToolPermissionFixtureEventsSurface)' -count=1`
- Local commit created: `f3c553ac` (`test: add fixture-backed ACP mock agents`)
