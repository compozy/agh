# Task Memory: task_17.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Close task 17 with reusable cross-provider conformance coverage for multi-instance ownership, restart recovery, DM policy, auth degradation, and classified retry behavior.

## Important Decisions

- Record `bridges/instances/report_state` in the harness host forwarder so classified recovery updates are observable in conformance evidence without provider-specific marker writes.
- Aggregate multiple representative scenarios into one matrix row per provider/platform by unioning coverage targets and managed-instance outcomes.
- Keep representative coverage focused on GitHub, Telegram, and WhatsApp because they exercise the required ownership, restart, DM policy, auth, and rate-limit paths without expanding task scope.

## Learnings

- The original task-17 matrix failed because WhatsApp rate-limit degradation went through `Session.ReportClassifiedError`, which updated daemon state but bypassed provider-written state markers.
- The new representative matrix passes quickly when the provider scenarios run as subtests; this localizes future provider regressions to one named conformance target.
- `internal/extensiontest` package coverage remains below the broad package-level threshold because the harness package already contains a large amount of existing helper surface, but the new shared matrix file itself measures `83.3%` statement coverage via `/tmp/extensiontest.cover`.

## Files / Surfaces

- `internal/extensiontest/bridge_adapter_harness.go`
- `internal/extensiontest/bridge_adapter_harness_test.go`
- `internal/extensiontest/bridge_conformance_matrix.go`
- `internal/extensiontest/bridge_conformance_matrix_test.go`
- `internal/extension/provider_conformance_matrix_integration_test.go`
- `.compozy/tasks/bridge-adapters/memory/MEMORY.md`

## Errors / Corrections

- Corrected the initial matrix assumption that each provider scenario should produce a distinct provider row; the reusable matrix now merges scenario summaries by provider/platform.
- Corrected the missing degraded-state marker path by capturing Host API `report_state` calls in the harness instead of depending solely on provider-side marker writes.

## Ready for Next Run

- Verified task-specific suites:
  - `go test ./internal/extensiontest ./internal/extension -count=1`
  - `go test -tags integration ./internal/extensiontest ./internal/extension ./internal/daemon -run 'TestHarnessIntegrationTelegramReferenceConformance|TestRepresentativeProviderConformanceMatrix|TestBridgeRuntimeRestartPreservesRouteContinuity' -count=1`
  - `make verify`
- Update PRD tracking is the only remaining administrative step if the code diff changes again.
