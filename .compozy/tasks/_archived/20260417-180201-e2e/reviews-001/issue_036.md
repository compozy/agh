---
status: resolved
file: web/e2e/network.spec.ts
line: 124
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:dbc6de681e4f
review_hash: dbc6de681e4f
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 036: Consider simplifying URL assertion to avoid regex construction.
## Review Comment

Static analysis flagged potential ReDoS risk. While the input is controlled (from `appPage.url()`), you could simplify this by using Playwright's built-in string URL matching or extracting the path comparison.

## Triage

- Decision: `valid`
- Notes:
  The regex-based path assertion is safe in practice here, but it is more
  complex than needed and keeps triggering static-analysis noise. Comparing the
  reloaded pathname directly is simpler and removes the regex construction
  entirely.

## Resolution

- Replaced the reload URL regex with a direct pathname equality assertion.
