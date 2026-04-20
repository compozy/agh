---
status: resolved
file: internal/core/plan/prepare.go
line: 101
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4058705969,nitpick_hash:c9ae70fe4c40
review_hash: c9ae70fe4c40
source_review_id: "4058705969"
source_review_submitted_at: "2026-04-04T17:43:33Z"
---

# Issue 002: Wrap batch build failures with the batch index.
## Review Comment

This path now bubbles `buildBatchJob` errors up unchanged, so failures from `memory.Prepare` or `writeBatchArtifacts` lose which batch failed. Adding the batch number here will make prompt-preparation errors much easier to act on.

As per coding guidelines, `**/*.go`: Use explicit error returns with wrapped context using `fmt.Errorf("context: %w", err)`.

## Triage

- Decision: `valid`
- Root cause: `prepareJobs` calls `buildBatchJob` inside the batch loop and returns any error unchanged, so once control leaves `buildBatchJob` the caller no longer knows which batch failed.
- Evidence: the current implementation at `internal/core/plan/prepare.go` returns `err` directly from the `for idx, batchIssues := range batches` loop, and the existing tests only assert lower-level error wrapping from `buildBatchJob`.
- Fix approach: wrap `buildBatchJob` failures in `prepareJobs` with the 1-based batch index and add a regression test that forces the second PRD batch to fail, proving the batch context survives at the workflow-preparation boundary.
- Resolution: `prepareJobs` now wraps `buildBatchJob` failures as `build batch <index>/<total>: ...`, preserving the failing batch index at the workflow-preparation boundary.
- Regression coverage: added a targeted `prepareJobs` test that lets batch 1 succeed and forces batch 2 to fail on invalid task front matter, asserting the returned error includes both the second-batch wrapper and the underlying task-path parse error.
- Verification: `go test ./internal/core/plan -run 'Test(BuildBatchJobWrapsMemoryPreparationErrorWithTaskPath|PrepareJobsWrapsBatchBuildFailuresWithBatchIndex)$' -count=1` passed. Fresh `make verify` then passed cleanly, including formatting, lint, 2416 tests with 1 skipped, and the final `go build` step.
