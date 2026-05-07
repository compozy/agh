# BUG-003: Runtime E2E ACP test agents do not implement the v0.12.2 Agent interface

## Status

Fixed

## Severity

High

## Source

- Task 13 real-scenario QA execution.
- Required command: `make test-e2e-runtime`.

## Reproduction

1. Run the required runtime E2E lane:

   ```bash
   make test-e2e-runtime
   ```

2. Observe the `internal/daemon` integration package failing to build before any E2E test executes.

## Expected Behavior

The runtime E2E lane compiles all integration helpers against the current `github.com/coder/acp-go-sdk` interface and then executes the daemon/runtime E2E tests.

## Actual Behavior

`internal/daemon` failed to compile because helper ACP agents used by daemon integration tests did not implement the full `acpsdk.Agent` interface after the v0.12.2 SDK upgrade:

```text
daemonSessionStopACPAgent does not implement "github.com/coder/acp-go-sdk".Agent (missing method CloseSession)
*daemonNightlyCombinedACPAgent does not implement "github.com/coder/acp-go-sdk".Agent (missing method CloseSession)
*daemonSandboxACPAgent does not implement "github.com/coder/acp-go-sdk".Agent (missing method CloseSession)
```

## Root Cause

The ACP SDK upgrade added required session lifecycle/config methods to the `Agent` interface, but some integration helper agents were not updated with no-op implementations for unsupported optional behavior. The runtime lane builds packages with integration tags, so this gap was invisible to normal unit gates.

## Fix

Updated integration helper agents to satisfy the current `acpsdk.Agent` interface with valid empty responses:

- `internal/daemon/daemon_integration_test.go`
- `internal/daemon/daemon_sandbox_integration_test.go`
- `internal/daemon/daemon_nightly_combined_integration_test.go`
- `internal/testutil/e2e/runtime_harness_integration_test.go`

## Regression Coverage

- `make test-e2e-runtime` must pass after the fix.

## Evidence

- Initial failure log: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-test-e2e-runtime.log`
- Fixed rerun log: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-test-e2e-runtime-rerun.log`
