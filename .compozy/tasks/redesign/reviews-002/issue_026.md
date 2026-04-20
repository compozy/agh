---
status: resolved
file: packages/ui/src/components/pills.tsx
line: 127
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4136983224,nitpick_hash:aeb0946757fc
review_hash: aeb0946757fc
source_review_id: "4136983224"
source_review_submitted_at: "2026-04-20T02:30:11Z"
---

# Issue 026: Add a defensive disabled guard in the click handler.
## Review Comment

Right now `Line 129` relies on native disabled-button behavior to suppress `onChange`. Adding an explicit guard makes intent resilient to future refactors.

## Triage

- Decision: `invalid`
- Notes:
  - `Pills` renders native `<button disabled>` elements. The platform already suppresses activation for disabled buttons, and the existing `pills.test.tsx` coverage verifies that `onChange` is not fired for disabled items.
  - Adding a second disabled guard in the click handler would be redundant defensive code without a current defect to fix.
