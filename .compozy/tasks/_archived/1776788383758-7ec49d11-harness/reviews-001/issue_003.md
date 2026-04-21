---
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 201
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:bb11516cd882
review_hash: bb11516cd882
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 003: Bound prompt collection to a test-scoped timeout.
## Review Comment

These prompt submissions use `context.Background()`, and `collectIntegrationPromptEvents` blocks on a plain channel range. If one prompt stops emitting without closing, this test hangs until the package timeout instead of failing at the call site. Use a cancellable test context and abort the drain when it expires.

Also applies to: 2424-2438

## Triage

- Decision: `valid`
- Notes:
  - The HTTP transcript integration test submits prompts with `context.Background()` and drains prompt events with a plain channel range.
  - If prompt dispatch stalls or the channel never closes, this test hangs until the outer package timeout instead of failing at the prompt call site.
  - I will add a bounded prompt context and timeout-based channel drain for the HTTP variant.
