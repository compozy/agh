# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add nightly-only combined runtime/browser E2E coverage on the shared harnesses, expand multi-domain artifacts, and preserve credentialed Daytona/provider coverage as nightly-only.

## Important Decisions
- Added `internal/e2elane.NightlyRuntimeE2EPattern` with a `TestDaemonNightlyE2E...` naming contract instead of widening the default runtime selector.
- Added dedicated `combined_flow.json` and `tool_host_diagnostics.json` artifacts so multi-domain failures keep cross-domain IDs plus concrete tool-host outcomes.
- Kept the browser combined-flow scenario on the existing Bridges/session Playwright fixtures and fixed the shipped Bridges UI data-refresh bug instead of adding browser reloads or sleeps to the spec.

## Learnings
- The daemon-served Bridges page can show live health `route_count=1` while the routes query stays cached at `0`; invalidating route queries on bridge health SSE route-count changes fixes the real operator-path bug.
- Combined automation/task resume coverage needed a shared `RuntimeHarness.ResumeSession(...)` helper so nightly runtime tests could stay on the public operator surface.
- Focused evidence that mattered for this task was: `go test ./internal/e2elane ./internal/testutil/e2e`, `go test -tags integration ./internal/daemon -run 'TestDaemonNightlyE2E(AutomationTaskResumesIntoNetworkChannel|BridgeIngressUsesEnvironmentToolBeforeDelivery)$'`, `bunx playwright test web/e2e/combined-flows.spec.ts --grep @nightly`, and `make verify`.

## Files / Surfaces
- `internal/e2elane/{lanes.go,lanes_test.go}`
- `internal/testutil/e2e/{artifacts.go,artifacts_test.go,runtime_harness.go,runtime_harness_helpers_test.go}`
- `internal/daemon/daemon_nightly_combined_integration_test.go`
- `web/e2e/combined-flows.spec.ts`
- `web/src/systems/bridges/hooks/{use-bridge-health-stream.ts,use-bridge-health-stream.test.tsx}`

## Errors / Corrections
- Initial nightly Playwright run failed because ingress updated bridge health before the selected bridge's routes query refreshed, leaving the detail panel on "No routes". Fixed the root cause in `use-bridge-health-stream` by invalidating cached route queries when live `route_count` changes.

## Ready for Next Run
- Implementation, focused validation, and `make verify` are complete. Remaining close-out work in this run is tracking-file updates plus the final post-tracking verification/commit.
