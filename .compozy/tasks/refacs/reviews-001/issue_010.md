---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/httpapi/middleware_refac_test.go
line: 70
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:5e77c6e1795c
review_hash: 5e77c6e1795c
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 010: Assert the response body in the successful branches.
## Review Comment

Both success paths stop after status/header checks, so a regression that starts returning an unexpected payload would still pass.

As per coding guidelines, "Assert both HTTP status code AND response body in tests; status-code-only assertions are insufficient".

Also applies to: 115-120

## Triage

- Decision: `VALID`
- Notes:
  The successful middleware branches only assert status and selected headers. Since both success paths intentionally return `204 No Content`, the tests should also assert an empty body so payload regressions do not go unnoticed.
