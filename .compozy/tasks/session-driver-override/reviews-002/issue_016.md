---
status: resolved
file: web/src/systems/session/components/session-resume-failure.tsx
line: 26
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167289384,nitpick_hash:04c791026aa8
review_hash: 04c791026aa8
source_review_id: "4167289384"
source_review_submitted_at: "2026-04-24T01:37:12Z"
---

# Issue 016: Normalize missingProvider before deriving provider-specific UI.
## Review Comment

`hasProviderDetail` uses raw length, so whitespace-only values would still trigger provider-specific title/message/badge. Trim once and reuse the normalized value.

Also applies to: 53-55, 62-64

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: `missingProvider` is used raw when deriving provider-specific UI, so whitespace-only values still trigger the provider badge/title copy. I will normalize it once with `trim()` and add regression coverage in `web/src/systems/session/components/session-resume-failure.test.tsx`, which is the minimal out-of-scope test file needed for this behavior.
