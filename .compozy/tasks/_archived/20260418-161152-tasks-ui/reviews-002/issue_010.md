---
status: resolved
file: internal/observe/tasks_integration_test.go
line: 269
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133463506,nitpick_hash:ce0b3966b9cb
review_hash: ce0b3966b9cb
source_review_id: "4133463506"
source_review_submitted_at: "2026-04-18T03:54:22Z"
---

# Issue 010: Exercise WithTaskDashboardConfig instead of mutating internals.
## Review Comment

These tests set `h.observer.taskDashboardConfig.backlogWarnAfter` directly, which skips the normalization and constructor wiring added in this PR. A regression in `WithTaskDashboardConfig(...)` could still leave these integration tests green.

As per coding guidelines, "Never hardcode configuration — use TOML config or functional options".

Also applies to: 460-461

## Triage

- Decision: `VALID`
- Reasoning: `internal/observe/tasks_integration_test.go` mutates `h.observer.taskDashboardConfig.backlogWarnAfter` directly, which bypasses the normalization and option application path introduced by `WithTaskDashboardConfig(...)`.
- Root cause analysis: The integration tests are asserting dashboard behavior through direct internal mutation instead of the public configuration option.
- Intended fix: Update the affected integration tests to configure backlog thresholds through `WithTaskDashboardConfig(...)`, and use that path in coverage for repeated partial overrides as well.
- Resolution: Updated the scoped integration tests to configure dashboard thresholds through `WithTaskDashboardConfig(...)` and added layered-option coverage in the same file.
- Verification:
  - `go test ./internal/extension ./internal/observe`
  - `go test -tags integration ./internal/observe -run 'TestObserveTaskDashboard|TestObserveHealthReflectsRecoveryAndForcedStopOutcomes|TestObserveTaskLifecycleSummaryAndMetrics'`
  - `make verify` still fails outside this batch in the web TypeScript gate on pre-existing Storybook/MSW dependency/type errors unrelated to these Go changes.
