---
provider: coderabbit
pr: "106"
round: 3
round_created_at: 2026-05-06T06:28:14.497092Z
status: resolved
file: internal/situation/service.go
line: 597
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3hZ3,comment:PRRC_kwDOR5y4QM6-V-L0
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Validate that the review-bound run actually belongs to the loaded task.**

This path trusts `review.TaskID` and `review.RunID` independently. If those records drift, the session can receive task metadata from one task and coordination state from another run. Guard `run.TaskID == taskRecord.ID` before building the bundle or channel payload.

<details>
<summary>Suggested fix</summary>

```diff
 	run, err := store.GetTaskRun(ctx, review.RunID)
 	if err != nil {
 		if isContextError(err) {
 			return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
 		}
 		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
 	}
+	if strings.TrimSpace(run.TaskID) != strings.TrimSpace(taskRecord.ID) {
+		slog.Warn(
+			"situation: skip review-bound context for mismatched task/run",
+			"review_id", strings.TrimSpace(review.ReviewID),
+			"task_id", strings.TrimSpace(taskRecord.ID),
+			"run_id", strings.TrimSpace(run.ID),
+			"run_task_id", strings.TrimSpace(run.TaskID),
+		)
+		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", nil
+	}
 	bundle, err := s.sessionContextBundle(ctx, taskRecord, run, workspaceSnapshot, strings.TrimSpace(sessionID))
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/situation/service.go` around lines 558 - 597, The code fetches
taskRecord via store.GetTask and run via store.GetTaskRun but never verifies
they belong together; add a guard after retrieving run to compare run.TaskID and
taskRecord.ID and if they differ return the same zero-value triple used
elsewhere (contract.AgentTaskContextPayload{},
contract.AgentCoordinationChannelContextPayload{}, "", nil) to avoid mixing task
metadata and run coordination state before calling s.sessionContextBundle or
coordinationChannelPayload; place this check between the GetTaskRun call and the
s.sessionContextBundle(...) invocation and use the existing return pattern used
for non-context errors.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause analysis: the review-bound context path loads `taskRecord` from `review.TaskID` and `run` from `review.RunID` but never verifies that `run.TaskID` still matches the loaded task.
- Why this is valid: if review metadata drifts, the current implementation can assemble a bundle from one task and a coordination channel from another run, which is cross-task contamination.
- Fix approach: add a task/run consistency guard in `internal/situation/service.go` before `sessionContextBundle`, and add a regression test in `internal/situation/service_test.go` that returns no review-bound task/channel context for mismatched records.

## Resolution

- Added a review-bound task/run consistency guard in `internal/situation/service.go`.
- Added a regression case in `internal/situation/service_test.go` proving mismatched review/task/run records do not surface task or coordination context.

## Verification

- Focused regression: `go test ./internal/situation -run 'TestTaskStoreStubListRunReviewsSortsBeforeApplyingLimit|TestTaskRunPromptOverlayByIDRejectsMismatchedRunTaskPair|TestContextForSessionIncludesReviewerTaskBundleWithoutActiveLease' -count=1 -race`
- Fresh full gate: `make verify` exited `0` in this session.
