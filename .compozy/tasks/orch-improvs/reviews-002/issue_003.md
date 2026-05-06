---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/cli/task.go
line: 920
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-N,comment:PRRC_kwDOR5y4QM6-VcCn
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Allow reviewer/status-only review queries.**

`task review list --status ... --reviewer-session ...` currently fails before it reaches the API because this helper always requires `--task` or `--run`. That blocks the normal “show my pending reviews” flow on a global list endpoint.
 
<details>
<summary>Suggested fix</summary>

```diff
 func parseTaskRunReviewListFilters(
 	taskID string,
 	runID string,
 	statusRaw string,
 	reviewerSessionID string,
 	last int,
 ) (TaskRunReviewListQuery, error) {
 	if strings.TrimSpace(taskID) != "" && strings.TrimSpace(runID) != "" {
 		return TaskRunReviewListQuery{}, errors.New("cli: choose either --task or --run")
 	}
-	if strings.TrimSpace(taskID) == "" && strings.TrimSpace(runID) == "" {
-		return TaskRunReviewListQuery{}, errors.New("cli: task review list requires --task or --run")
+	if strings.TrimSpace(taskID) == "" &&
+		strings.TrimSpace(runID) == "" &&
+		strings.TrimSpace(statusRaw) == "" &&
+		strings.TrimSpace(reviewerSessionID) == "" &&
+		last == 0 {
+		return TaskRunReviewListQuery{}, errors.New("cli: task review list requires at least one filter")
 	}
 	status, err := parseOptionalReviewStatus(statusRaw)
 	if err != nil {
 		return TaskRunReviewListQuery{}, err
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func parseTaskRunReviewListFilters(
	taskID string,
	runID string,
	statusRaw string,
	reviewerSessionID string,
	last int,
) (TaskRunReviewListQuery, error) {
	if strings.TrimSpace(taskID) != "" && strings.TrimSpace(runID) != "" {
		return TaskRunReviewListQuery{}, errors.New("cli: choose either --task or --run")
	}
	if strings.TrimSpace(taskID) == "" &&
		strings.TrimSpace(runID) == "" &&
		strings.TrimSpace(statusRaw) == "" &&
		strings.TrimSpace(reviewerSessionID) == "" &&
		last == 0 {
		return TaskRunReviewListQuery{}, errors.New("cli: task review list requires at least one filter")
	}
	status, err := parseOptionalReviewStatus(statusRaw)
	if err != nil {
		return TaskRunReviewListQuery{}, err
	}
	if err := validateTaskLast(last); err != nil {
		return TaskRunReviewListQuery{}, err
	}
	return TaskRunReviewListQuery{
		TaskID:            strings.TrimSpace(taskID),
		RunID:             strings.TrimSpace(runID),
		Status:            status,
		ReviewerSessionID: strings.TrimSpace(reviewerSessionID),
		Limit:             last,
	}, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/cli/task.go` around lines 893 - 920, parseTaskRunReviewListFilters
currently rejects requests when both taskID and runID are empty, which prevents
reviewer/status-only queries; update the validation so it still errors if both
taskID and runID are provided, but allows both to be empty when the caller
supplied a status or a reviewerSessionID (i.e. permit global queries like "my
pending reviews"). To implement: in parseTaskRunReviewListFilters, call
parseOptionalReviewStatus and trim reviewerSessionID early, then change the
check that enforces a non-empty task or run to only trigger when status and
reviewerSessionID are both empty; keep validateTaskLast and the existing return
structure (TaskID, RunID, Status, ReviewerSessionID, Limit) unchanged and ensure
you still trim inputs when building the TaskRunReviewListQuery.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `parseTaskRunReviewListFilters` hard-requires `--task` or `--run`, which blocks global reviewer/status queries even though the API supports those filters.
- Fix approach: Allow filter-only queries when status, reviewer session, or `--last` is supplied, and cover the new acceptance/rejection matrix in `internal/cli/task_test.go`.
