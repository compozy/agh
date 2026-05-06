---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: web/src/systems/agent/components/stories/agent-command-select.stories.tsx
line: 13
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4239418533,nitpick_hash:fe2a6e33ef15
review_hash: fe2a6e33ef15
source_review_id: "4239418533"
source_review_submitted_at: "2026-05-06T20:41:36Z"
---

# Issue 010: Extract the categorized story fixture into a shared helper.
## Review Comment

This category map is now duplicated here and in `web/src/components/stories/app-sidebar.stories.tsx`. Keeping two copies will make the categorized demos drift over time, especially since this PR is using both stories to show the same feature surface.

## Triage

- Decision: `valid`
- Notes:
  - The categorized story fixture is duplicated in both `app-sidebar.stories.tsx` and `agent-command-select.stories.tsx`.
  - Root cause: the same category map is maintained in two story modules that are demonstrating the same feature surface.
  - Fix approach: centralize the categorized story data in one scoped story module and reuse it from the other, staying within the batch’s allowed file set.
