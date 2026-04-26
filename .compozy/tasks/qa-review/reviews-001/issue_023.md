---
status: resolved
file: web/src/systems/knowledge/components/knowledge-list-panel.tsx
line: 44
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:15eef0ef15ec
review_hash: 15eef0ef15ec
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 023: Use one canonical memoryKey for row identity.
## Review Comment

`data-testid` uses `memory.key ?? memory.filename`, while selection/callback/key use `knowledgeMemoryKey(memory)`. Keeping a single key source avoids drift and duplicate IDs in fallback scenarios.

Also applies to: 162-166

## Triage

- Decision: `valid`
- Notes:
  - `KnowledgeListItem` uses `memory.key ?? memory.filename` for the row `data-testid`, while the component's key, selection, and click callback all use `knowledgeMemoryKey(memory)`.
  - Root cause: row identity is derived from a different fallback path than the rest of the component, so a future key derivation change can drift test ids and selection semantics apart.
  - Fix plan: compute the canonical memory key once through `knowledgeMemoryKey(memory)` and reuse it for row identity, selection, and tests, then update `knowledge-list-panel.test.tsx` accordingly.
