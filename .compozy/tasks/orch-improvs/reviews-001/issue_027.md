---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/store/globaldb/global_db_task_claim.go
line: 545
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:2275f4912c4a
review_hash: 2275f4912c4a
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 027: Blank agent names currently bypass execution-profile routing.
## Review Comment

When `criteria.AgentName` is empty, this helper returns before applying the explicit `worker_agent_name` check and the allowed-agent filters. That makes runs with agent-pinned execution profiles claimable by any matching session that has the required capabilities.

## Triage

- Decision: `valid`
- Notes: `appendProfileClaimFilters` returns early when `criteria.AgentName` is blank, which skips both the explicit `worker_agent_name` exclusion and the allowed-agent selector filters. That allows sessions without an agent name to claim runs that were intentionally pinned to a named worker/reviewer population as long as the capability checks pass. Fix by always applying the exact-worker and allowed-agent filters, even when `AgentName` is empty.
- Resolution: Removed the early return so blank agent names still hit the exact-worker and allowed-agent filters, and extended the claim test to cover the blank-agent case.
