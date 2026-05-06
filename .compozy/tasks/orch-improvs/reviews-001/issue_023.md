---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/situation/task_context.go
line: 122
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:6988bea375d3
review_hash: 6988bea375d3
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 023: Verify that runID belongs to taskID.
## Review Comment

`GetTask` and `GetTaskRun` are loaded independently here and then combined without checking `run.TaskID == taskRecord.ID`. A mismatched pair will render task A with run B data and can leak the wrong run context.

## Triage

- Decision: `valid`
- Notes: `TaskRunPromptOverlayByID` loads `GetTask(taskID)` and `GetTaskRun(runID)` independently, then passes both to `TaskRunPromptOverlay` without verifying that the run belongs to the task. A mismatched pair would render task A with run B state and leak the wrong context. Fix by rejecting mismatched `run.TaskID` / `taskRecord.ID` pairs with a validation error before building the overlay.
- Resolution: Added the explicit task/run pairing check in `TaskRunPromptOverlayByID` and covered it with a regression test.
