---
status: resolved
file: web/src/systems/agent/components/agent-stats-grid.tsx
line: 60
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177569108,nitpick_hash:84f075eef815
review_hash: 84f075eef815
source_review_id: "4177569108"
source_review_submitted_at: "2026-04-26T22:35:58Z"
---

# Issue 008: Share the failure predicate with session-status.ts.
## Review Comment

This condition duplicates the stopped/failed classification from `web/src/systems/agent/lib/session-status.ts`. If one side changes, the badge in `AgentSessionsList` and this counter will drift. A shared helper would keep both views consistent.

## Triage

- Decision: `VALID`
- Notes:
  - `AgentStatsGrid` duplicates the failure classification already embedded in `getAgentSessionStatus`, so the table badge and aggregate failed counter can drift.
  - The minimum correct fix requires a supporting edit outside the listed code files: `web/src/systems/agent/lib/session-status.ts` will export a shared `isAgentSessionFailure` predicate.
  - `AgentStatsGrid` will consume that helper, keeping the counter and badge classification on the same source of truth.
  - Resolution: exported `isAgentSessionFailure` from `session-status.ts` and used it in both the badge classifier and stats counter.
  - Verification: targeted Vitest passed; `make verify` passed.
