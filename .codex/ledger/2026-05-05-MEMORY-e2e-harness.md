Goal (incl. success criteria):

- Implement Task 17 "E2E Harness and Fixture Alignment" for network-threads PRD in `/Users/pedronauck/Dev/compozy/agh2`.
- Success requires runtime/web E2E harnesses asserting final thread/direct/work semantics, no active legacy peer-room artifact fields, targeted tests, clean `make verify`, tracking updates, and one local commit.

Constraints/Assumptions:

- Must use `cy-workflow-memory`, `cy-execute-task`, and `cy-final-verify`.
- Must read workflow memory, PRD docs, ADRs, AGENTS/CLAUDE guidance, `internal/CLAUDE.md`, and `web/CLAUDE.md` before code edits.
- Destructive git commands are forbidden without explicit user permission.
- `make verify` is the completion gate.
- Automatic commit is enabled only after clean verification, self-review, and tracking updates.

Key decisions:

- Treat the existing PRD/TechSpec/task set as the approved design; no new design approval is needed before implementation.
- Do not author QA report/execution files in Task 17; `.compozy/extensions/cy-qa-workflow/extension.toml` owns Task 18/19 generation.

State:

- Harness implementation, tracking, final pre-commit verification, and requested local implementation commit are complete.

Done:

- Created session ledger for compaction-safe continuity.
- Read required skill entrypoints for `cy-workflow-memory`, `cy-execute-task`, and `cy-final-verify`.
- Read workflow memory, root/internal/web guidance, Task 17, `_techspec.md`, `_design.md`, ADRs 001-003, and dependency task outputs for tasks 06 and 08-16.
- Activated/read task-relevant skills: `agh-code-guidelines`, `agh-test-conventions`, `testing-anti-patterns`, `golang-pro`, `real-scenario-qa`, `qa-report`, `agh-worktree-isolation`, `no-workarounds`, `systematic-debugging`, `vitest`, `typescript-advanced`, and `react`.
- Baseline found remaining stale surfaces:
  - `web/e2e/fixtures/runtime-seed.ts` still sends `kind:"direct"` and `interaction_id`.
  - `web/e2e/fixtures/selectors.ts` and `web/e2e/network.spec.ts` still target old peer-room selectors.
  - `internal/testutil/e2e/runtime_harness.go` still captures flattened channel messages instead of thread/direct/work snapshots.
  - `internal/testutil/acpmock/testdata/network_collaboration_fixture.json` has partial thread/direct/work metadata coverage.
- Hardened acpmock exact metadata tests and fixture correlation fields.
- Added runtime harness helpers/artifacts for network threads, direct rooms, direct-room messages, direct resolve, and work lookup.
- Tightened daemon audit assertions to include surface/container/work metadata when expected.
- Extended daemon collaboration E2E source for direct resolve race, summarize-back-to-thread, and thread/direct/work HTTP read assertions.
- Updated web runtime seed, selector facade, route artifact capture, and Playwright network scenario to use final thread/direct/work surfaces.
- Active stale-term scan across Task 17 Go/web harness targets now finds `interaction_id`/`network_selected_peer` only in explicit negative/assertion checks.
- Focused Go checks passed:
  - `go test ./internal/testutil/acpmock -run 'Test(FixtureLookupAndHelperErrors|TurnMatchNetworkRequiresExactConversationMetadata|ReadDiagnosticsParsesJSONLines)' -count=1`
  - `go test ./internal/testutil/e2e -run TestRuntimeHarnessCaptureHelpersPersistArtifacts -count=1`
  - `go test ./internal/daemon -run 'TestValidateNetwork' -count=1`
- Focused web fixture checks passed:
  - `bunx vitest run web/e2e/fixtures/runtime-seed.test.ts web/e2e/fixtures/browser-artifact-session.test.ts web/e2e/fixtures/selectors.test.ts`
- Targeted daemon integration slice passed after tightening waits/resolvers:
  - `go test -tags integration ./internal/daemon -run 'TestDaemonE2ENetwork(DirectReplyLifecycleWithMockAgents|WhoisAndCapabilityExchange)|TestPromptInputCompositeIntegrationPreservesStoredMessagesAcrossUserAndNetworkTurns' -count=1`
- Focused web checks passed after aligning the seed thread ID grammar and route-state active-tab capture:
  - `bun run typecheck:raw` in `web`
  - `bunx vitest run e2e/fixtures/browser-artifact-session.test.ts`
  - `bunx playwright test e2e/network.spec.ts`
- Broad runtime E2E prereq failures from acpmock prompt augmentation and fixture permission modes were corrected:
  - acpmock canonicalization now strips situation/current-skills/durable-memory wrappers before `user_text` matching.
  - tool permission fixture uses `approve-reads` so edit permission emits an approvable pending request.
  - driver fault fixture emits permission before disconnect while keeping deny-all for blocked-cancel behavior.
- Fixed ACP stop-request cancellation race: `Driver.runPrompt` now suppresses prompt runtime errors caused after an explicit process stop request; added `TestPromptStopDoesNotEmitRuntimeError`.
- Previously failing broad runtime subset now passes:
  - `go test ./internal/acp -run '^TestPromptStopDoesNotEmitRuntimeError$' -count=1`
  - `go test -tags integration ./internal/daemon -run 'TestDaemonE2E(AutomationPromptTriggerCreatesCompletedSystemSession|AutomationTaskBackedJobDelegatesTaskRun|ACPmock(CrashMidStreamProjectsRuntimeFailure|InvalidFrameProjectsRuntimeFailure|PermissionDisconnectProjectsRuntimeFailure|BlockedCancelStopsPromptWithoutOrphaning)|ToolPermissionFixtureEventsSurface)' -count=1 -parallel=4`
- `make test-e2e-runtime` follow-up failures fixed:
  - HTTP/UDS transport approval parity uses `permission_env_fixture.json`; changed approver permissions to `approve-reads` so edit permission yields a pending request.
  - UDS observe lifecycle parity now expects all three current turn augmenters: durable memory, skills, and situation.
  - Focused checks passed: `go test -race -tags integration ./internal/api/httpapi -run '^TestHTTPTransportApprovalFlowUsesSharedRuntimeHarness$' -count=1`, `go test -race -tags integration ./internal/api/udsapi -run '^TestUDSTransportApprovalFlowMatchesHTTP$' -count=1`, and `go test -race -tags integration ./internal/api/udsapi -run '^TestUDSTransportObserveHarnessLifecycleParityMatchesHTTP$' -count=1`.
- `make test-e2e-runtime` now passes across daemon, HTTP, UDS, and runtime harness packages.
- `make test-e2e-web` first pass had Task 17 network spec passing but exposed unrelated full-suite drift:
  - scoped Bridges dialog close button to avoid toast close buttons.
  - changed browser lifecycle acpmock permissions to `approve-reads` so the onboarding edit permission remains pending for operator approval.
  - Storybook bootstrap now waits for the actual story module before browser navigation.
  - Focused rerun passed for Bridges + session onboarding; Storybook spec passes after module-readiness fix.
- `make test-e2e-web` now passes: 19 Playwright tests passed.
- Stale-term scan across Task 17 harness targets only finds intentional negative/assertion coverage in native tool tests and browser artifact absence assertions.
- Final implementation `make verify` passed with Go lint `0 issues`, `DONE 8400 tests`, and `OK: all package boundaries respected`.
- Updated Task 17 tracking checkboxes/status, master task status, task memory, and shared workflow memory.
- Post-tracking `make test-e2e-runtime` exposed runtime harness races; fixed bridge Host API prompt submission to drain prompt events before stored-event lookup, and fixed runtime harness shutdown to signal the owned daemon process instead of relying on daemon-info PID lookup.
- Focused post-fix checks and `make test-e2e-runtime` pass again.
- Final pre-commit `make verify` rerun passed after bridge prompt-drain and runtime harness shutdown fixes: frontend format/lint/typecheck/build succeeded, Go lint reported `0 issues`, Go tests reported `DONE 8400 tests`, and package boundary checks reported `OK: all package boundaries respected`.
- Created local implementation commit `1897db7e` (`test: align network e2e harness fixtures`) with only Task 17 implementation files staged; tracking/memory artifacts and unrelated pre-existing changes remain unstaged.

Now:

- Final status check and handoff.

Next:

- Report verification evidence, commit hash, and unstaged tracking/unrelated worktree state.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- Task paths under `/Users/pedronauck/Dev/compozy/agh2/.compozy/tasks/network-threads`.
- Initial edit targets: `internal/testutil/acpmock/fixture_test.go`, `internal/testutil/acpmock/testdata/network_collaboration_fixture.json`, `internal/daemon/network_e2e_assertions_test.go`, `internal/daemon/daemon_network_collaboration_integration_test.go`, `internal/testutil/e2e/runtime_harness.go`, `internal/testutil/e2e/runtime_harness_helpers_test.go`, `web/e2e/fixtures/*`, `web/e2e/network.spec.ts`.
