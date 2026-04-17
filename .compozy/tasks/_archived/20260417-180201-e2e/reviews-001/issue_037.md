---
status: resolved
file: web/e2e/session-onboarding.spec.ts
line: 76
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:ecd9a07e81ff
review_hash: ecd9a07e81ff
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 037: Consider simplifying URL assertion to avoid regex construction.
## Review Comment

Same pattern flagged by static analysis as in `network.spec.ts`. The escaped regex from variable input has minimal ReDoS risk here but can be simplified.

## Triage

- Decision: `valid`
- Notes:
  This reload assertion has the same avoidable regex construction as the
  network flow. Direct pathname equality is sufficient and easier to reason
  about for this session-continuity check.

## Resolution

- Replaced the reload URL regex with a direct pathname equality assertion.
