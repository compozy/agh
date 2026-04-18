---
status: resolved
file: internal/api/httpapi/server_test.go
line: 515
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133264038,nitpick_hash:4178d2dc00bd
review_hash: 4178d2dc00bd
source_review_id: "4133264038"
source_review_submitted_at: "2026-04-18T02:14:16Z"
---

# Issue 008: Split the non-loopback matrix into t.Run("Should...") subtests.
## Review Comment

This loop packs several independent cases into one test body, so the first failure hides the rest and the failure output is less specific. Wrapping each entry in a named subtest will make this matrix much easier to debug and extend.

As per coding guidelines, "`**/*_test.go`: MUST use t.Run(\"Should...\") pattern for ALL test cases`".

## Triage

- Decision: `valid`
- Notes:
  The non-loopback permission matrix in `internal/api/httpapi/server_test.go` currently runs multiple independent request cases inside shared loops, which hides later failures behind the first failing case and makes debugging noisy. I will rewrite those matrices into explicit `t.Run("Should ...")` subtests so each route assertion reports independently.
