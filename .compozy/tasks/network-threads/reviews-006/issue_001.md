---
provider: coderabbit
pr: "105"
round: 6
round_created_at: 2026-05-06T03:03:04.040959Z
status: resolved
file: internal/acp/client_test.go
line: 1097
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232891518,nitpick_hash:000695500363
review_hash: "000695500363"
source_review_id: "4232891518"
source_review_submitted_at: "2026-05-06T03:02:32Z"
---

# Issue 001: Make the subtest parallel and avoid relying on double-stop idempotency
## Review Comment

This subtest skips `t.Parallel()` and currently invokes `Stop()` both explicitly and again in cleanup. Guarding cleanup after a successful explicit stop makes this test less brittle.

As per coding guidelines, "Default to `t.Parallel` in Go tests unless there is a specific reason to disable it (opt-out with `t.Setenv`)."

## Triage

- Decision: `valid`
- Notes:
  The subtest does not call `t.Parallel()` and currently stops the process twice on the success path: once explicitly through `driver.Stop(...)` and again unconditionally in `t.Cleanup`. That makes the test more brittle than needed and violates the AGH default-parallel test shape. I will add subtest parallelism and guard cleanup so it only stops the process when the explicit stop did not already succeed.
  Resolved by guarding cleanup after a successful explicit stop and re-verifying the batch with `make verify`. I tested the `t.Parallel()` suggestion directly with `go test ./internal/acp -run 'TestPromptStopDoesNotEmitRuntimeError' -count=1 -v`, and that path reproduced the exact runtime-error event the test is meant to suppress (`peer disconnected before response`), so the subtest remains serial by evidence.
