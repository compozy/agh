---
status: resolved
file: internal/api/httpapi/helpers_test.go
line: 125
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:e9cf6e39d046
review_hash: e9cf6e39d046
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 013: Consider extracting shared resource-handler test config
## Review Comment

Both constructors duplicate nearly the full `handlerConfig` setup; a small internal helper would reduce maintenance drift.

Also applies to: 157-177

## Triage

- Decision: `INVALID`
- Reason: The shared setup extraction the review asks for is already present. Both constructors delegate through `newTestHandlersWithAutomationBridgesTasksAndWorkspace` in [internal/api/httpapi/helpers_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/api/httpapi/helpers_test.go#L68), so there is no remaining duplication to remove here.

## Resolution

- Analysis complete. No code change was required because the shared helper the review requested already exists.
