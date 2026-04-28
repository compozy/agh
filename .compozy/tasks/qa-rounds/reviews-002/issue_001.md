---
status: resolved
file: internal/api/core/error_paths_test.go
line: 236
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4189206979,nitpick_hash:a7454981cf39
review_hash: a7454981cf39
source_review_id: "4189206979"
source_review_submitted_at: "2026-04-28T13:29:54Z"
---

# Issue 001: Wrap this case in t.Run("Should ...") to match test conventions.
## Review Comment

The test does not use the required subtest pattern. Per coding guidelines, `**/*_test.go` tests must use `t.Run("Should ...")` subtests with `t.Parallel()` by default.

## Triage

- Decision: `VALID`
- Notes:
  - `TestListAgentsWorkspaceResolverUnavailable` is a standalone test body with no `t.Run("Should ...")` subtest.
  - This violates the AGH test-shape rule that each case must live inside a `Should ...` subtest with `t.Parallel()` by default.
  - Fix approach: wrap the current assertions in a `t.Run("Should return service unavailable when workspace resolver is missing", ...)` subtest and keep the subtest parallel because it does not use `t.Setenv` or shared mutable state.

## Resolution

- Wrapped the workspace-resolver-unavailable assertions in a `Should ...` subtest and kept both the parent test and subtest parallel.
- Verified with targeted `go test -race ./internal/api/core -run 'TestListAgentsWorkspaceResolverUnavailable|TestBaseHandlersNetworkChannelMessagesPreserveRemoteAuthors' -count=1`.
- Verified the repository gate with `make verify` after code changes.
