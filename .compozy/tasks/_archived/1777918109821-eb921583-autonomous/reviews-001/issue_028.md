---
status: resolved
file: internal/cli/client_test.go
line: 29
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:e02f080cde67
review_hash: e02f080cde67
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 028: Split these new agent client tests into t.Run("Should...") subtests.
## Review Comment

Each function currently packs several independent request/response paths into one assertion chain, so the first failure hides the rest of the transport coverage and makes regressions harder to localize.

As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `VALID`
- Notes: The new agent client tests batch several request/response paths in single top-level functions. A first failure hides the remaining path coverage and makes transport regressions harder to locate.
- Fix: Split the channel and task client method coverage into `t.Run("Should...")` subtests while keeping per-case request validation.
