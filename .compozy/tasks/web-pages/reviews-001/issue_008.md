---
status: resolved
file: internal/cli/cli_integration_test.go
line: 1170
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:b8d84d9a556a
review_hash: b8d84d9a556a
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 008: Prefer returning an empty provider slice from the test stub.
## Review Comment

Using an empty slice keeps list responses array-shaped in integration flows.

## Triage

- Decision: `valid`
- Root cause: the integration stub returns a `nil` provider slice, which serializes to `null` instead of the array-shaped response contract used by the bridges list endpoint.
- Fix approach: return an empty slice so integration behavior matches the API shape.
