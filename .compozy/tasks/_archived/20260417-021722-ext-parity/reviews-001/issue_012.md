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

# Issue 012: Handle os.ErrNotExist in catalog-backed ListAgents
## Review Comment

`GetAgent` already maps not-found correctly, but `ListAgents` currently returns 500 for all catalog errors. Mapping not-found to `200 {agents: []}` would keep behavior consistent with the filesystem fallback.

## Triage

- Decision: `VALID`
- Notes: `ListAgents` already returns `200 {agents: []}` for the filesystem path when the agents directory does not exist, but the catalog-backed path currently turns `os.ErrNotExist` into a 500. That is an observable behavior mismatch and should be normalized to the empty-list response.
