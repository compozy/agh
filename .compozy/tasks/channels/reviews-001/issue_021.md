---
status: resolved
file: internal/daemon/channels_test.go
line: 31
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:e085c18b8300
review_hash: e085c18b8300
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 021: Consider adding t.Parallel() for independent tests.
## Review Comment

None of the tests in this file use `t.Parallel()`. Since each test appears to create its own isolated test fixtures (homePaths, config, DB), they could potentially run in parallel to speed up test execution.

As per coding guidelines: "Use t.Parallel() for independent subtests".

## Triage

- Decision: `valid`
- Why: The tests in `internal/daemon/channels_test.go` build isolated temp directories, configs, sockets, and databases per test and do not share mutable package-level state, so they are safe to run in parallel.
- Root cause: The file omitted `t.Parallel()` on independent top-level tests even though the repo's test conventions call for parallel execution where isolation allows it.
- Fix plan: Add `t.Parallel()` to each top-level test in this file.
- Resolution: Added `t.Parallel()` to the independent top-level tests and verified the package with the targeted runs plus `make verify`.
