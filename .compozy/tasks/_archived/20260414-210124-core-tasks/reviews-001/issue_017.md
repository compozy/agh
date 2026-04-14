---
status: resolved
file: internal/cli/task_test.go
line: 484
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:b832aeb7ce17
review_hash: b832aeb7ce17
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 017: Make the sample run fixture match the requested status.
## Review Comment

`sampleTaskRunRecord` populates end-state fields even for queued/claimed/starting runs. That makes it harder for these tests to catch status-dependent rendering or JSON-shaping bugs.

As per coding guidelines, `**/*_test.go`: Ensure tests verify behavior outcomes, not just function calls.

---

## Triage

- Decision: `valid`
- Root cause: `sampleTaskRunRecord` currently populates claimed, started, ended, error, and result fields for every status, which can hide status-dependent rendering regressions in the CLI tests.
- Fix approach: make the fixture populate lifecycle fields conditionally based on the requested task-run status.

## Resolution

- Reworked `sampleTaskRunRecord` so claimed, session-bound, started, ended, error, and result fields are only populated for statuses that should actually carry them.
- Verified in the final `make verify` run.
