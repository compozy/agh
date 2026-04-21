---
status: resolved
file: internal/network/router_integration_test.go
line: 119
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:dd96c9c622da
review_hash: dd96c9c622da
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 022: Wrap these new scenarios in t.Run("Should...") subtests.
## Review Comment

Both tests add more capability lifecycle coverage, but they still use standalone test bodies instead of the repo’s required subtest pattern. Moving these into `t.Run("Should ...")` blocks will also make the repeated router setup easier to extend for future cases.

As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `valid`
- Notes:
  The new router integration scenarios are standalone test bodies instead of `t.Run("Should ...")` subtests, which is inconsistent with the repository's required Go test pattern and makes related setup less extensible.
  I will wrap the scoped scenarios in `Should...` subtests without changing their behavior.
  Fixed and verified with targeted package tests plus `make verify`.
