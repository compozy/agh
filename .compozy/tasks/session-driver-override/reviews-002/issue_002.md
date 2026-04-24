---
status: resolved
file: internal/api/httpapi/transport_parity_integration_test.go
line: 254
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167289384,nitpick_hash:abd8c12309ec
review_hash: abd8c12309ec
source_review_id: "4167289384"
source_review_submitted_at: "2026-04-24T01:37:12Z"
---

# Issue 002: Avoid hardcoded provider literal in assertion.
## Review Comment

Line 254 hardcodes `"qa-transport-override"` instead of deriving from `transportOverrideProvider`, making this assertion fragile if the constant changes.

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: The final error assertion hardcodes the override provider literal instead of deriving it from `transportOverrideProvider`, so the test can drift from the configured fixture. I will switch the assertion to use the constant directly.
