---
status: resolved
file: internal/observe/observer_test.go
line: 93
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:f22e7360bba5
review_hash: f22e7360bba5
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 014: Prefer one table-driven test with t.Run("Should...") subtests here.
## Review Comment

These two cases only vary by recovery source, so consolidating them would remove duplicated setup/assert logic and match the test structure required in this repo.

As per coding guidelines, `**/*_test.go`: "Table-driven tests with subtests (t.Run) as default pattern" and "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `VALID`
- Notes: The live-source and registry recovery tests duplicate setup and assertions while varying only the recovery source. Fix by consolidating them into one table-driven test with `Should ...` subtests.
