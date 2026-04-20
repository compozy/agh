---
status: resolved
file: internal/cli/workspace_config_test.go
line: 88
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4058705969,nitpick_hash:9cd70674bdef
review_hash: 9cd70674bdef
source_review_id: "4058705969"
source_review_submitted_at: "2026-04-04T17:43:33Z"
---

# Issue 001: Use the real Cobra binding here instead of manually mirroring state.batchSize.
## Review Comment

Production registers `batch-size` with `IntVar(&state.batchSize, ...)` in `internal/cli/root.go` (line 147), but this test uses an unbound `Int()` plus `state.batchSize = 2` (lines 88, 109). This workaround can mask regressions in the actual flag binding while the test still passes. Replace the unbound flag with `IntVar(&state.batchSize, "batch-size", 1, "batch size")` and remove the manual assignment to test the real production path.

## Triage

- Decision: `valid`
- Root cause: `internal/cli/workspace_config_test.go` registers `batch-size` with an unbound `Int()` flag and then manually mirrors the parsed value into `state.batchSize`, so the test can still pass even if the Cobra flag binding path used by production regresses.
- Evidence: production binds `batch-size` with `cmd.Flags().IntVar(&state.batchSize, "batch-size", 1, ...)` in `internal/cli/reviews_exec_daemon.go`, while the test currently uses `cmd.Flags().Int("batch-size", 1, ...)` plus `state.batchSize = 2`.
- Fix approach: update the test to bind `batch-size` directly to `state.batchSize`, remove the manual assignment, and verify the explicit-flag precedence path with the real Cobra wiring.

## Resolution

- Updated `internal/cli/workspace_config_test.go` to register `batch-size` with `cmd.Flags().IntVar(&state.batchSize, "batch-size", 1, "batch size")` and removed the manual `state.batchSize = 2` assignment.
- Full verification initially surfaced an unrelated lint blocker in already-modified `internal/cli/run_observe.go`; I kept that in-flight branch work intact and wired the existing default observe options through the default attach helpers so the branch remained behaviorally equivalent and lint-clean.
- Verification:
  - `go test ./internal/cli -run '^TestApplyWorkspaceDefaultsDoesNotOverrideChangedFlags$'`
  - `make verify`
