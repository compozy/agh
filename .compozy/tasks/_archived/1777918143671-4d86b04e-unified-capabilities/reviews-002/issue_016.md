---
status: resolved
file: internal/extension/host_api_test.go
line: 4849
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:721055fe11ed
review_hash: 721055fe11ed
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 016: Consider using t.Errorf instead of t.Fatalf in cleanup hooks.
## Review Comment

Using `t.Fatalf` inside `t.Cleanup` stops execution of subsequent cleanup hooks and can produce confusing output since the test has already completed. Cleanup errors are typically informational—you want to report them but allow other cleanup hooks to run.

## Triage

- Decision: `valid`
- Root cause: using `t.Fatalf` inside the shared environment cleanup can stop later cleanup work and hide additional teardown failures.
- Fix plan: downgrade those cleanup reports to `t.Errorf` so the test still fails, but the remaining cleanup steps continue to run and report their own errors.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
