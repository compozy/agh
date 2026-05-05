---
status: resolved
file: internal/api/httpapi/transport_parity_integration_test.go
line: 123
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:c5b69118a5f2
review_hash: c5b69118a5f2
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 002: Fold the new provider parity cases into a table-driven t.Run suite.
## Review Comment

These two tests share the same harness/bootstrap work and only vary in the action/assertion path. Converting them to subtests will remove duplicated setup and match the repo's default pattern for Go tests.

As per coding guidelines, `**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests.

## Triage

- Decision: `valid`
- Root cause: the new HTTP provider tests duplicate the same harness/bootstrap flow and diverge only in their assertions, which violates the repository's default table-driven subtest pattern.
- Fix plan: fold the provider create/read and resume-missing-provider cases into a shared parent test with `t.Run("Should ...")` subtests.
