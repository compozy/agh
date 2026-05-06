---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/daemon/review_router.go
line: 473
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:550fa3e967be
review_hash: 550fa3e967be
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 014: Don't exclude every reviewer that shares the worker's agent name.
## Review Comment

Self-review is already prevented by session ID / peer ID. The extra agent-name exclusion here rejects distinct sessions running the same agent definition and even prevents spawning a second reviewer session for that agent. In a workspace that only has one reviewer-capable agent type, reviews become unroutable even when separate worker instances exist.

Also applies to: 493-500, 600-611

## Triage

- Decision: `valid`
- Notes:
  - The router still excludes candidates purely by matching `AgentName`, both for existing-session reuse and create-time agent selection.
  - Self-review is already blocked by session/peer identity, so the additional name-based exclusion incorrectly rejects distinct sessions that use the same reviewer-capable agent definition.
  - Planned fix: keep self-review checks on concrete session/peer identity only, and allow separate reviewer sessions that share the same agent name.
  - Resolved: name-based reviewer exclusion was removed, self-review protection stays on concrete session/peer identity, and tests cover reuse/create with same-agent reviewers.
