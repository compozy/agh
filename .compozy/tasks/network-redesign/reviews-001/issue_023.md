---
status: resolved
file: web/src/systems/network/lib/network-formatters.ts
line: 286
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4166737115,nitpick_hash:7fd22c6b024f
review_hash: 7fd22c6b024f
source_review_id: "4166737115"
source_review_submitted_at: "2026-04-23T23:14:00Z"
---

# Issue 023: Redundant === true comparisons.
## Review Comment

The `.includes()` method already returns a boolean, so the explicit `=== true` is unnecessary. However, this is a minor stylistic preference.

## Triage

- Decision: `invalid`
- Notes:
- `web/src/systems/network/lib/network-formatters.ts:286-289` does contain redundant `=== true` comparisons, but the code is already correct and unambiguous at runtime.
- This batch is scoped to review remediation, not opportunistic style churn, and no repository lint rule or behavioral bug requires rewriting these expressions.
- Leaving the code unchanged avoids mixing a preference-only edit into a batch that otherwise contains functional fixes.

## Resolution

- No code change was necessary. The report was closed as a style-only preference after confirming the current implementation is behaviorally correct and passes the repo lint/type/test gates unchanged.
