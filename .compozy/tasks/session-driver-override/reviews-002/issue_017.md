---
status: resolved
file: web/src/systems/session/hooks/use-session-create-dialog.test.tsx
line: 95
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167289384,nitpick_hash:d7dd2b362f9e
review_hash: d7dd2b362f9e
source_review_id: "4167289384"
source_review_submitted_at: "2026-04-24T01:37:12Z"
---

# Issue 017: Optional: deduplicate provider fixtures to reduce drift risk.
## Review Comment

The provider option literals are repeated in multiple places; extracting one constant keeps future edits safer.

Also applies to: 125-132

## Triage

- Decision: `UNREVIEWED`
- Decision: `invalid`
- Notes: This comment is explicitly optional and proposes only fixture deduplication to reduce maintenance drift. It does not identify a failing behavior, missing assertion, or contract bug in the current tests, so it is not a required remediation item for this batch.
