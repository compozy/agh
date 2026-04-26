---
status: resolved
file: web/src/systems/knowledge/components/stories/knowledge-list-panel.stories.tsx
line: 201
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:be3802a4c9e0
review_hash: be3802a4c9e0
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 024: Use the shared key helper for story test-id construction.
## Review Comment

This keeps stories resilient if fallback key derivation rules evolve.

## Triage

- Decision: `valid`
- Notes:
  - The knowledge list panel story constructs a row test id from `defaultMemories[2].key` instead of the shared key helper used by the component.
  - Root cause: story code duplicates the key derivation contract instead of importing the canonical helper, so it can drift if fallback rules change.
  - Fix plan: import `knowledgeMemoryKey()` into the story and update the storybook source regression test to pin the shared helper usage.
