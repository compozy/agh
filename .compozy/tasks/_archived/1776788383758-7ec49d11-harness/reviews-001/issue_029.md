---
status: resolved
file: internal/task/manager_test.go
line: 3068
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:d8dc22927a80
review_hash: d8dc22927a80
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 029: Cover conflicting metadata on duplicate enqueue.
## Review Comment

This only locks down the happy path where the duplicate request sends the same metadata. The risky case for detached runs is reusing the same idempotency key with different metadata; without that assertion, this test will still pass if the enqueue path silently preserves the wrong wake target/payload. As per coding guidelines, focus on critical paths and ensure tests verify behavior outcomes, not just function calls.

## Triage

- Decision: `valid`
- Root cause: the current idempotency test only covers duplicate enqueue requests that repeat the same metadata payload. It does not assert the higher-risk case where the same idempotency key is reused with conflicting detached-run metadata.
- Fix approach: extend the test with a conflicting duplicate request and assert the existing run keeps the original metadata rather than silently adopting the new payload.
