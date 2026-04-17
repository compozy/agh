---
status: resolved
file: packages/site/components/landing/primitives/code-block.tsx
line: 28
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:18e054c9b514
review_hash: 18e054c9b514
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 019: Minor: Consider cleaning up the timeout on unmount.
## Review Comment

If the component unmounts during the 1.5-second feedback window, the `setTimeout` callback will still fire and attempt to update state on an unmounted component. While React 18+ handles this gracefully without errors, cleaning up is still good practice.

## Triage

- Decision: `invalid`
- Notes:
  - I did not reproduce a user-visible defect here: the timeout lasts 1.5 seconds and React tolerates a queued state update after unmount without surfacing an error in this setup.
  - The component does not retain subscriptions or long-lived resources, so the remaining callback is transient rather than a persistent leak.
  - Adding timer bookkeeping would increase component complexity for negligible value in this batch.
