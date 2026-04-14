---
status: resolved
file: internal/cli/task_test.go
line: 151
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:cfeb56ec3e70
review_hash: cfeb56ec3e70
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 016: Break the command-mapping coverage into table-driven subtests.
## Review Comment

Each of these tests covers many unrelated commands in one flow, so the first failure hides the rest of the parsing surface.

As per coding guidelines, `**/*_test.go`: Use table-driven tests with subtests (t.Run) as default in Go tests.

Also applies to: 257-376

## Triage

- Decision: `valid`
- Root cause: the CLI task command mapping tests pack many unrelated command paths into single monolithic flows, so one failure hides the rest of the parsing surface.
- Fix approach: convert those flows into table-driven `Should...` subtests so each command mapping is isolated.

## Resolution

- Reworked the CLI task command mapping coverage into table-driven `Should...` subtests so each create/list/run/mutation path fails independently.
- Verified in the final `make verify` run.
