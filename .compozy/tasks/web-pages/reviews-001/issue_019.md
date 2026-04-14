---
status: resolved
file: web/src/systems/bridges/hooks/use-bridge-actions.ts
line: 27
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:5c34dc337222
review_hash: 5c34dc337222
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 019: Consider optimistic updates for improved UX.
## Review Comment

Both mutation hooks correctly use `onSettled` for cache invalidation per guidelines. For `useCreateBridge`, optimistic updates aren't applicable since there's no prior state. However, `useTestBridgeDelivery` could potentially benefit from an optimistic loading indicator pattern via `onMutate`/`onError` snapshots if the UI shows delivery test status.

## Triage

- Decision: `invalid`
- Reasoning: this is a generic UX suggestion, not a correctness defect in the current code. `useTestBridgeDelivery` resolves a one-off target lookup and does not back a cached list/detail status that benefits from optimistic mutation state.
- Reasoning: adding artificial optimistic cache writes here would not fix a real bug and would introduce state the UI does not currently model.
