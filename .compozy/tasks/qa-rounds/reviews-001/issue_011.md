---
status: resolved
file: internal/network/router_test.go
line: 160
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:68efdaccc85c
review_hash: 68efdaccc85c
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 011: Split the new echo coverage into t.Run subtests.
## Review Comment

These are two independent behaviors—broadcast self-echo and directed self-echo—so separate subtests will isolate failures better and match the repo’s default test style.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern" and "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `VALID`
- Notes: `TestRouterDoesNotDeliverLocalEchoesToSender` checks broadcast and directed self-echo behavior in one body. Fix by splitting these into separate `Should ...` subtests that share the same router setup pattern.
