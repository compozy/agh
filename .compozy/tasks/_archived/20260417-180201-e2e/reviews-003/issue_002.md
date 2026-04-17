---
status: resolved
file: internal/api/udsapi/transport_parity_integration_test.go
line: 327
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130744050,nitpick_hash:c248a3312939
review_hash: c248a3312939
source_review_id: "4130744050"
source_review_submitted_at: "2026-04-17T17:17:39Z"
---

# Issue 002: String containment for JSON type matching is fragile but acceptable for tests.
## Review Comment

The `udsSessionEventsContainType` helper uses `strings.Contains(string(event.Content), `"type":"` + want + `"`)` which could false-positive on embedded strings. Consider JSON unmarshaling for robustness (similar to the httpapi implementation), though the current approach is acceptable for test code.

## Triage

- Decision: `valid`
- Notes:
  - `udsSessionEventsContainType` currently uses a raw substring match against JSON payload bytes, so a nested string field containing `"type":"<value>"` can satisfy the helper even when the event type does not match.
  - The same test surface already has a stronger JSON-based pattern elsewhere, so the helper should decode the event payload and inspect the actual `type` field.
  - Implemented: replaced the raw substring match with JSON unmarshaling against the `content.type` field.
  - Verification: `go test -tags integration ./internal/api/udsapi -run 'TestUDSTransport' -count=1`; `make verify`.
