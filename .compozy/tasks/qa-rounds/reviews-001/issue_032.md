---
status: resolved
file: web/src/systems/network/mocks/handlers.ts
line: 127
severity: minor
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:5c3c5d9e2376
review_hash: 5c3c5d9e2376
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 032: Guard required fields with runtime type checks before .trim().
## Review Comment

At **Line 127**, non-string payload values can throw (e.g., `channel: 123`) before returning the intended 400 contract response.

## Triage

- Decision: `valid`
- Notes: The `/api/network/send` MSW handler calls `.trim()` through optional chaining on fields typed as optional strings, but malformed JSON can still provide non-string values and throw before the intended 400 response. The fix is to read required string fields through runtime type checks before trimming.
