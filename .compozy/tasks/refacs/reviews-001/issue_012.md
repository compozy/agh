---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/testutil/bridge_stub.go
line: 26
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:0471ec9ab2ef
review_hash: 0471ec9ab2ef
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 012: *TaskSubscription*Fn fields drop the Bridge prefix that the actual methods carry.
## Review Comment

Every other `*Fn` field in this struct is named after its method verbatim (`CreateInstanceFn` → `CreateInstance`, `ResolveOrCreateRouteFn` → `ResolveOrCreateRoute`, etc.). Four fields break the pattern:

| Field | Method |
|---|---|
| `PutTaskSubscriptionFn` | `PutBridgeTaskSubscription` |
| `GetTaskSubscriptionFn` | `GetBridgeTaskSubscription` |
| `ListTaskSubscriptionsFn` | `ListBridgeTaskSubscriptions` |
| `DeleteTaskSubscriptionFn` | `DeleteBridgeTaskSubscription` |

Test authors following the convention will instinctively look for `PutBridgeTaskSubscriptionFn` and not find it.

## Triage

- Decision: `INVALID`
- Notes:
  This is a naming-consistency nit, not a behavior defect. The existing field names are already the established stub API used across multiple tests. Renaming them would force broad unrelated churn outside the scoped remediation without improving correctness.
