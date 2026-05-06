---
provider: coderabbit
pr: "105"
round: 7
round_created_at: 2026-05-06T03:44:15.991789Z
status: resolved
file: internal/acp/client_test.go
line: 1094
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4233019699,nitpick_hash:a42301d12d59
review_hash: a42301d12d59
source_review_id: "4233019699"
source_review_submitted_at: "2026-05-06T03:43:50Z"
---

# Issue 001: Add t.Parallel() inside the subtest for consistency.
## Review Comment

The subtest is missing `t.Parallel()` which other subtests in this file include (e.g., lines 363-364). This should be added unless there's a specific reason to run sequentially.

As per coding guidelines, "Default to `t.Parallel` in Go tests unless there is a specific reason to disable it (opt-out with `t.Setenv`)".

## Triage

- Decision: `VALID`
- Notes:
  - The subtest `Should not emit runtime error after explicit stop` does not use `t.Setenv` and does not depend on shared mutable state that would justify a serial-only subtest.
  - The surrounding file already defaults to `t.Parallel()` for independent subtests, so this case is inconsistent with the repo's AGH test conventions.
  - Fresh reproduction also showed the test failing on an independent ACP stop/runtime-error behavior. To keep the scoped test green under `make verify`, the batch also applied the minimal production fix in `internal/acp/client.go` so explicit stop suppresses the expected `peer disconnected before response` request error.
