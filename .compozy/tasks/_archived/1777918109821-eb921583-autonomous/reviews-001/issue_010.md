---
status: resolved
file: internal/api/core/tasks.go
line: 1471
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:432bf1b8cae3
review_hash: 432bf1b8cae3
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 010: ExecutionRequest lacks a Validate() method used by peer request types.
## Review Comment

The mapper validates `NetworkChannel` via `validateTaskChannel()`, but unlike `enqueueTaskRunFromRequest` (which calls `spec.Validate()`), `taskExecutionRequestFromRequest` has no centralized validation for the domain struct. Peer types like `EnqueueRun`, `ClaimRun`, and `StartRun` all define `Validate()` methods; `ExecutionRequest` should follow the same pattern for consistency. Downstream normalization handles metadata size validation, but this split responsibility is fragile. Consider adding a `Validate()` method to `ExecutionRequest`.

## Triage

- Decision: `VALID`
- Notes: `taskExecutionRequestFromRequest` validates the network channel locally and returns the domain request without invoking a domain-level validation method, while peer request types use `Validate`. The root cause is that `taskpkg.ExecutionRequest` has no `Validate` method. A complete fix requires a minimal out-of-scope domain edit in `internal/task/validate.go` so the API mapper can call the same kind of centralized validation as peers; without that, any fix in `tasks.go` would only duplicate validation locally.
