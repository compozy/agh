---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/situation/service.go
line: 870
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:c455c26a110e
review_hash: c455c26a110e
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 020: Keep run.CoordinationChannelID authoritative.
## Review Comment

Leading with `metadata["coordination_channel_id"]` lets duplicated JSON override the structured field on the run. If those ever diverge, the session context can point at the wrong coordination channel and filter the wrong inbox/peer set.

As per coding guidelines, "Authoritative primitives are exclusive — when a primitive owns a state transition (`task.Service.ClaimNextRun`, `Spawn`, `EnsureMigration`), no peer package may replicate it".

## Triage

- Decision: `valid`
- Notes:
  - `coordinationChannelPayload` still prefers duplicated JSON metadata before `run.CoordinationChannelID`.
  - That lets a lossy copy in `run.Metadata` override the structured field that the runtime already treats as authoritative.
  - Planned fix: make `run.CoordinationChannelID` win ahead of metadata and keep `run.NetworkChannel` only as a fallback.
  - Resolved: `coordinationChannelPayload` now prefers `run.CoordinationChannelID`, falls back to duplicated metadata only when needed, and situation tests cover authoritative structured-field precedence.
