---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: internal/extension/install_managed_test.go
line: 66
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215828964,nitpick_hash:df21f53b228f
review_hash: df21f53b228f
source_review_id: "4215828964"
source_review_submitted_at: "2026-05-03T03:31:19Z"
---

# Issue 005: Add t.Parallel() to subtests.
## Review Comment

The subtests for unsafe name rejection are missing `t.Parallel()` calls. Per coding guidelines, subtests should use `t.Parallel()` by default.

As per coding guidelines: "Use `t.Run("Should ...")` subtests with `t.Parallel` as default"

## Triage

- Decision: `valid`
- Root cause: the unsafe-name subtests in `TestManagedInstallHelpers` are independent and only read immutable parent state, so they should follow the repository default of `t.Parallel()`.
- Fix plan: add `t.Parallel()` inside each affected subtest.
