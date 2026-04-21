---
status: resolved
file: internal/network/router_integration_test.go
line: 196
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135966430,nitpick_hash:36aff77d8442
review_hash: 36aff77d8442
source_review_id: "4135966430"
source_review_submitted_at: "2026-04-19T12:48:57Z"
---

# Issue 007: Add t.Parallel() for independent integration tests.
## Review Comment

The new integration tests are self-contained and can run concurrently. Per coding guidelines, independent tests should use `t.Parallel()`.

## Triage

- Decision: `valid`
- Root cause: `internal/network/router_integration_test.go` adds four top-level integration tests that allocate their own transport, peer registries, channels, and cleanup, but none currently opts into `t.Parallel()`.
- Why this is valid: the scoped tests use isolated state and `testNetworkConfig()` binds transports on ephemeral ports (`Port: -1`), so they match the package convention for independent tests and can run concurrently without shared-state coupling.
- Impact: the router integration suite runs more serially than necessary and diverges from the rest of the `internal/network` test package, where comparable isolated tests already opt into parallel execution.
- Resolution: `internal/network/router_integration_test.go` now calls `t.Parallel()` as the first statement in all four scoped top-level integration tests, preserving their existing setup/cleanup while allowing the test runner to schedule them concurrently.
- Verification:
  - `go test ./internal/network -tags integration -run 'Test(RoutersDiscoverEachOtherAndExchangeDirectAndBroadcastMessages|HeartbeatExpiryAndFreshGreetRecovery|DirectedWhoisRichDiscoveryDeliversPeerCardAndCapabilityCatalog|DirectedWhoisRichDiscoveryFilteringRefreshesRemotePresence)$' -count=1`
  - `make verify`
