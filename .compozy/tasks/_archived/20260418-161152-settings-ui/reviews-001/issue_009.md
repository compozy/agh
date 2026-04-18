---
status: resolved
file: internal/api/httpapi/transport_parity_integration_test.go
line: 248
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133264038,nitpick_hash:e69501821d16
review_hash: e69501821d16
source_review_id: "4133264038"
source_review_submitted_at: "2026-04-18T02:14:16Z"
---

# Issue 009: Avoid representation-sensitive parity assertions here.
## Review Comment

`reflect.DeepEqual` will treat `nil` and empty slices/maps as different, so this can fail even when HTTP and UDS are semantically identical after JSON decoding. Normalize zero-value collections first, or switch to a comparator that equates empty collections before asserting transport parity.

As per coding guidelines, "MUST test meaningful business logic, not trivial operations".

Also applies to: 285-286

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in the extension parity integration test: it uses `reflect.DeepEqual` on decoded JSON payloads, which treats `nil` and empty collections differently even when the HTTP and UDS responses are semantically identical. I will normalize the compared payloads or use a comparator that equates empty collections before asserting parity.
