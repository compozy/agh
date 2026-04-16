---
status: resolved
file: internal/automation/resource_projection.go
line: 669
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123260080,nitpick_hash:dc5ae92ad376
review_hash: dc5ae92ad376
source_review_id: "4123260080"
source_review_submitted_at: "2026-04-16T17:55:39Z"
---

# Issue 018: Webhook secret synced before unchanged check.
## Review Comment

The webhook secret is synced before checking if the trigger definition is unchanged (lines 674-680). This ensures secrets are always consistent but performs potentially redundant work for unchanged triggers.

This appears intentional for consistency guarantees. Consider adding a brief comment explaining this design choice if it's intentional.

## Triage

- Decision: `VALID`
- Root cause: The referenced file was refactored; the equivalent loop now lives in [internal/automation/manager.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager.go#L1450). The code intentionally syncs managed webhook secrets even when the trigger definition itself is unchanged, but that ordering is not obvious to the next reader.
- Fix approach: Add a brief comment in the current implementation explaining why secret reconciliation runs independently of the unchanged-definition fast path.

## Resolution

- Added a short comment in [internal/automation/manager.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager.go) explaining why managed webhook-secret reconciliation still runs when the trigger definition is unchanged.
