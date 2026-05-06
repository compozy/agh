---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/api/core/tasks_test.go
line: 247
severity: minor
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:2768ae3a6256
review_hash: 2768ae3a6256
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 003: Assert response bodies on the new negative and 204 cases.
## Review Comment

These additions introduce several status-only checks. That leaves the response contract untested on the paths most likely to drift — especially the 400/404 JSON payloads and the 204 empty-body cases. Decode `contract.ErrorPayload` on the error branches and assert an empty body for the `204` responses.

As per coding guidelines "Always assert both HTTP status code AND response body in tests; status-code-only assertions are insufficient".

Also applies to: 447-512, 763-780

## Triage

- Decision: `valid`
- Notes:
  - The new handler tests still contain status-only assertions on error and `204 No Content` paths.
  - That leaves JSON error payloads and empty-body guarantees unverified, which violates the repo’s HTTP test contract.
  - Planned fix: decode `contract.ErrorPayload` on the negative branches and assert an empty body on the `204` responses.
  - Resolved: `tasks_test.go` now asserts empty `204` bodies, concrete bad-request and not-found payloads, and the dynamic subscription IDs emitted by the server-owned create flow.
