# Task Memory: task_17.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Align Task 17 E2E/runtime/web fixtures with the final network conversation model before QA workflow generation.
- Required acceptance: acpmock/prompt assertions, daemon collaboration flows, runtime harness helpers, web E2E selectors/seeds/browser artifacts, old `network_selected_peer` absence tests, targeted E2E slices, clean `make verify`, tracking updates, one local commit.

## Important Decisions
- Treat existing `_techspec.md`, `_design.md`, ADRs, and completed dependency task outputs as the approved design; Task 17 implements harness alignment rather than creating new QA report/execution artifacts.
- Do not author `.compozy/tasks/network-threads/qa/*` here; Task 18/19 are generated/consumed by `cy-qa-workflow`.

## Learnings
- Shared workflow memory confirms tasks 06/12 already implemented wrapper metadata (`surface`, container ID, `work_id`, `reply_to`, `trace_id`, `causation_id`, `trust`); Task 17 should harden acpmock/runtime/browser fixtures around that behavior.
- `_design.md` §13 acceptance signals require browser artifacts to expose `network_selected_thread` and `network_selected_direct`, with no `network_selected_peer`.
- Baseline inspection found active stale web E2E payloads/selectors: `web/e2e/fixtures/runtime-seed.ts` still sends `kind:"direct"`/`interaction_id`, and `web/e2e/network.spec.ts` plus selector helpers target old peer-room UI test IDs.
- Runtime harness capture still uses a flattened channel-message projection; Task 17 needs thread/direct/work capture helpers so downstream QA debugs final conversation containers.
- Daemon collaboration tests already resolve direct rooms and read thread messages indirectly, but audit assertions only match message/direction/kind and should assert surface/container/work/correlation fields where provided.
- Focused Go checks pass after initial harness edits: acpmock exact metadata tests, runtime harness helper artifact test, and daemon network assertion helper tests.
- Web fixture unit tests pass after replacing active `interaction_id`/old peer-room selectors with thread/direct/work seed payloads and current route artifact selectors.
- Active stale-term scan across Task 17 harness targets now finds `interaction_id`/`network_selected_peer` only in intentional negative/assertion checks.
- Targeted daemon integration slice passes after adding an audit-backed wait for `msg_summary_01` summarize-back delivery and aligning the prompt composite test's custom skills augmenter with the harness workspace resolver.
- Focused web route/artifact checks and Playwright network E2E pass after changing the browser seed to a valid `thread_...` ID and deriving `network_active_tab` from final network route paths for detail pages.

## Files / Surfaces
- Initial target surfaces: `internal/testutil/acpmock`, `internal/testutil/e2e`, `internal/daemon` network E2E/integration tests, and `web/e2e` selectors/fixtures/spec.
- Initial concrete edit set: `internal/testutil/acpmock/fixture_test.go`, `internal/testutil/acpmock/testdata/network_collaboration_fixture.json`, `internal/daemon/network_e2e_assertions_test.go`, `internal/daemon/daemon_network_collaboration_integration_test.go`, `internal/testutil/e2e/runtime_harness.go`, `internal/testutil/e2e/runtime_harness_helpers_test.go`, `web/e2e/fixtures/{runtime-seed,browser-artifact-session,selectors}.ts`, related tests, and `web/e2e/network.spec.ts`.
- Touched so far: `internal/testutil/acpmock/fixture_test.go`, `internal/testutil/acpmock/testdata/network_collaboration_fixture.json`, `internal/daemon/network_e2e_assertions_test.go`, `internal/daemon/daemon_network_collaboration_integration_test.go`, `internal/daemon/prompt_input_composite_integration_test.go`, `internal/testutil/e2e/{artifacts.go,runtime_harness.go,runtime_harness_helpers_test.go}`.
- Web files touched: `web/e2e/network.spec.ts`, `web/e2e/fixtures/runtime-seed.ts`, `web/e2e/fixtures/runtime-seed.test.ts`, `web/e2e/fixtures/browser-artifact-session.ts`, `web/e2e/fixtures/browser-artifact-session.test.ts`, `web/e2e/fixtures/selectors.ts`, `web/e2e/fixtures/selectors.test.ts`.

## Errors / Corrections
- `TestDaemonE2ENetworkDirectReplyLifecycleWithMockAgents` initially read the final audit snapshot before the delivered summary row was guaranteed visible; fixed the wait condition to require the thread-surface delivered audit row for `msg_summary_01`.
- `TestPromptInputCompositeIntegrationPreservesStoredMessagesAcrossUserAndNetworkTurns` initially used the daemon workspace resolver inside its custom composite skills augmenter, which did not resolve the harness workspace. Fixed the fixture to use the same harness workspace resolver shape as the test session manager.
- `web/e2e/network.spec.ts` initially exposed two real fixture mismatches: invalid `thread_id` grammar (`browser_thread_patch_42`) and direct-list summary expecting the first handoff message instead of the latest trace. Fixed both while keeping direct-detail assertions for the handoff and trace messages.
- Browser artifact capture initially omitted `network_active_tab` on direct/thread detail routes because detail pages do not mount the list-section test IDs; fixed route-state capture to derive the tab from `/network/:channel/{threads,directs,activity}`.
- Broad runtime E2E exposed acpmock wrapper and fixture issues outside the network-only slice:
  - augmented prompt matching needed to ignore situation/current-skills/durable-memory wrappers before fixture `user_text` matching;
  - tool-permission fixture needed `approve-reads` so edit permission surfaces as a pending approvable request;
  - permission-disconnect fixture needed permission before disconnect while keeping the fault agent deny-all mode.
- Broad runtime E2E also exposed a real ACP cancellation race: explicit `Stop` during a blocked prompt could record a prompt `error` with process-failure classification depending on timing. Fixed `Driver.runPrompt` to suppress prompt runtime errors after `proc.stopWasRequested()` and added `TestPromptStopDoesNotEmitRuntimeError`; the blocked-cancel daemon assertion now checks for clean `user_canceled` state and rejects non-cancellation error events.
- Previously failing broad runtime subset passes with `-parallel=4` after the ACP cancellation fix.
- `make test-e2e-runtime` then exposed two transport parity fixture drifts:
  - `permission_env_fixture.json` approver also needed `approve-reads` so HTTP/UDS approval flows observe pending permission before approval.
  - UDS observe lifecycle parity needed to expect the current three turn augmenters (`durable_memory`, `skills`, `situation`) instead of two.
- `make test-e2e-runtime` now passes across daemon, HTTP, UDS, and runtime harness slices.
- First full `make test-e2e-web` pass had the Task 17 network browser spec passing and exposed broader web drift fixed during this task:
  - Bridges spec now scopes the test-delivery `Close` button to the dialog to avoid Sonner toast close controls.
  - `browser_session_lifecycle_fixture.json` uses `approve-reads` so the onboarding edit permission remains approvable.
  - Storybook bootstrap waits for the dynamically imported story module before navigating the browser.
- `make test-e2e-web` now passes with 19 Playwright tests, including Task 17 `web/e2e/network.spec.ts`.
- Current stale-term scan across Task 17 harness targets finds only intentional negative/assertion coverage for `interaction_id` and `network_selected_peer`.
- Final `make verify` passed after all implementation changes with Go lint `0 issues`, `DONE 8400 tests`, and `OK: all package boundaries respected`.
- Post-tracking runtime rerun exposed two real harness races:
  - bridge Host API prompt submission queried stored prompt events before draining the prompt stream, so second ingress could return JSON-RPC internal error before a turn ID existed;
  - runtime harness cleanup used `agh daemon stop` through daemon-info PID files, which could signal the wrong daemon under parallel/PID-reuse pressure.
- Fixed both at root: bridge prompt submission now drains the event channel before querying stored events, and runtime harness `Stop` signals its owned process handle directly before waiting.
- Focused post-fix checks passed:
  - `go test -race -tags integration ./internal/daemon -run '^TestDaemonE2EACPmockPermissionDisconnectProjectsRuntimeFailure$' -count=10`
  - `go test -race -tags integration ./internal/daemon -run '^TestDaemonE2EBridgeIngressCreatesAndReusesRouteThroughTelegramExtension$' -count=3`
  - `go test ./internal/extension -run 'TestHostAPIHandler|TestManagerDeliverBridge' -count=1`
  - `go test ./internal/testutil/e2e -run 'TestRuntimeHarnessStop|TestStartRuntimeHarnessRepeatedCyclesLeaveNoStaleDaemonArtifacts' -count=1`
- `make test-e2e-runtime` now passes again after those fixes.
- Final pre-commit `make verify` rerun passed after the post-tracking bridge prompt-drain and runtime harness shutdown fixes: frontend format/lint/typecheck/build passed, Go lint reported `0 issues`, Go tests reported `DONE 8400 tests`, and package boundary checks reported `OK: all package boundaries respected`.
- Local implementation commit created: `1897db7e` (`test: align network e2e harness fixtures`). The commit intentionally contains only Task 17 implementation files; tracking/memory artifacts and unrelated pre-existing changes remain unstaged.

## Ready for Next Run
- Task 17 implementation and tracking are complete.
- Final verification is complete after the last harness race fixes.
- Local implementation commit is complete: `1897db7e`.
- QA report/execution task creation remains delegated to `.compozy/extensions/cy-qa-workflow/extension.toml`; Task 17 did not author QA report or execution artifacts.
- Generated QA tasks can use these evidence commands as deterministic harness slices: `make test-e2e-runtime`, `make test-e2e-web`, and final `make verify`.
