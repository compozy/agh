---
provider: coderabbit
pr: "85"
round: 2
round_created_at: 2026-04-30T19:49:37.693355Z
status: valid
file: internal/api/udsapi/network_test.go
line: 46
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4207030676,nitpick_hash:10a2ce53031b
review_hash: 10a2ce53031b
source_review_id: "4207030676"
source_review_submitted_at: "2026-04-30T17:01:44Z"
---

# Issue 006: Wrap the raw-token scenario in a dedicated t.Run("Should ...") block.
## Review Comment

This case is currently inline; moving it to a named subtest will align with the test-structure rule and isolate failures.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases."

## Triage

- Decision: `VALID`
- Notes:
  The raw-token rejection scenario is inline in
  `TestNetworkHandlersValidateRequestsAndMapErrors`. Wrap it in a named
  `t.Run("Should ...")` subtest so failures are isolated and the test follows
  AGH's subtest convention.
