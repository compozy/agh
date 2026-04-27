---
status: resolved
file: web/src/systems/agent/components/agent-sessions-list.tsx
line: 162
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177569108,nitpick_hash:2b82716ddc57
review_hash: 2b82716ddc57
source_review_id: "4177569108"
source_review_submitted_at: "2026-04-26T22:35:58Z"
---

# Issue 007: Consider memoizing Date.now() for consistent relative time.
## Review Comment

`formatRelativeTime` calls `Date.now()` on each invocation (line 166). If multiple sessions render close together, they may get slightly different "now" values, causing inconsistent relative times within the same render pass. This is a minor visual inconsistency, not a bug.

## Triage

- Decision: `VALID`
- Notes:
  - `formatRelativeTime` calls `Date.now()` for each rendered row, so sessions in the same table can format against different reference instants.
  - The impact is minor but real near relative-time boundaries and creates avoidable render-pass inconsistency.
  - Fix by capturing one `now` value per `AgentSessionsList` render and passing it to every row formatter.
  - Resolution: captured one render-pass timestamp and passed it through each session row formatter, with a test proving `Date.now()` is called once per table render.
  - Verification: targeted Vitest passed; `make verify` passed.
