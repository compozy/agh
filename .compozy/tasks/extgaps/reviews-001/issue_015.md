---
status: resolved
file: internal/extension/install_managed_test.go
line: 11
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110509614,nitpick_hash:96803fa212e9
review_hash: 96803fa212e9
source_review_id: "4110509614"
source_review_submitted_at: "2026-04-15T03:04:56Z"
---

# Issue 015: Use t.Run("Should...") subtests for these scenarios.
## Review Comment

These are both single-case top-level tests today. Wrapping the scenarios in `t.Run("Should ...")` keeps them aligned with the repo’s required test shape and makes it easier to extend the symlink matrix without cloning more setup.

As per coding guidelines, `**/*_test.go`: `MUST use t.Run("Should...") pattern for ALL test cases`.

---

## Triage

- Decision: `invalid`
- Reasoning: this is a style-only reshaping request for two already focused single-scenario tests. Wrapping each in an extra one-case `t.Run` block does not strengthen the behavioral contract being exercised and would add ceremony without improving coverage of the symlink-install regressions under review.
- Resolution: no code change required for this review item.
