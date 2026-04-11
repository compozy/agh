---
status: resolved
file: internal/extension/manifest_test.go
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:edd850912516
review_hash: edd850912516
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 028: Consider using t.Run subtests for the TOML/JSON equivalence checks.
## Review Comment

While the test is functional, using subtests would provide clearer output when only one format fails.

As per coding guidelines: "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  This is test-only work, but it is consistent with the repository’s stated preference for table-driven subtests. Converting the equivalence assertions into subtests improves failure localization without changing the tested behavior.
  Fix approach: split the TOML and JSON equivalence assertions into descriptive subtests in the existing scoped test file.
