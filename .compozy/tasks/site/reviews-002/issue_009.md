---
status: resolved
file: internal/api/core/handlers.go
line: 403
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:e2c0fb71a7b4
review_hash: e2c0fb71a7b4
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 009: Handle os.ErrNotExist in catalog-backed ListAgents
## Review Comment

`GetAgent` already maps not-found correctly, but `ListAgents` currently returns 500 for all catalog errors. Mapping not-found to `200 {agents: []}` would keep behavior consistent with the filesystem fallback.

## Triage

- Decision: `INVALID`
- Reason: The current `ListAgents` implementation is filesystem-backed and already returns `200` with an empty list when `os.ReadDir` hits `os.ErrNotExist` at [internal/api/core/handlers.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/api/core/handlers.go#L395). The "catalog-backed" 500-path described in the comment is not present in this tree.

## Resolution

- Analysis complete. No code change was required because the reviewed failure mode is not present in the current implementation.
