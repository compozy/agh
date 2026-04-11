---
status: resolved
file: internal/api/core/network_test.go
line: 120
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:8339c3854a49
review_hash: 8339c3854a49
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 002: Split the endpoint coverage into subtests.
## Review Comment

This one test currently exercises five handlers plus response-shape assertions. Reusing the shared fixture inside `t.Run(...)` blocks would keep setup cheap while making failures much easier to localize.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default".

## Triage

- Decision: `valid`
- Root cause: `TestBaseHandlersNetworkEndpoints` currently verifies multiple unrelated handlers in one long flow, which makes failures harder to localize and does not follow the repo’s subtest default.
- Fix approach: keep the shared fixture setup, but split the endpoint assertions into dedicated `t.Run("Should...")` subtests.
