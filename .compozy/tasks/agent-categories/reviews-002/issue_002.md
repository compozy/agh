---
provider: coderabbit
pr: "113"
round: 2
round_created_at: 2026-05-06T21:09:03.43169Z
status: resolved
file: web/src/systems/agent/components/stories/agent-command-select.stories.tsx
line: 93
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4239579182,nitpick_hash:95af5f4d04c4
review_hash: 95af5f4d04c4
source_review_id: "4239579182"
source_review_submitted_at: "2026-05-06T21:08:32Z"
---

# Issue 002: Grouped story description and assertions are slightly out of sync.
## Review Comment

Lines 93-95 say uncategorized agents render in a root-level “Agents” group, but the play test only asserts categorized groups. Add one assertion for the root uncategorized group to lock in that behavior.

Also applies to: 111-118

## Triage

- Decision: `valid`
- Notes:
  - The grouped-story comment is accurate: `categorizedAgents` assigns `category_path` to every fixture in `agentFixtures`, so the `Grouped` story never renders a root-level uncategorized section.
  - The current play function only checks categorized groups, which leaves the documented root-level `Agents` grouping unverified.
  - The fix is to make the story data explicitly include one uncategorized agent and assert the root group in the story play test.
  - Verification after the fix passed with `make web-lint`, `make web-typecheck`, `make web-test`, and the full `make verify` gate.
