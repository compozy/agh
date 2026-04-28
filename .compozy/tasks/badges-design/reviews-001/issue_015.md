---
status: resolved
file: web/src/systems/knowledge/components/knowledge-list-panel.tsx
line: 70
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:feed37d675c0
review_hash: feed37d675c0
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 015: Reuse knowledgeScopeShortLabel instead of duplicating scope label logic.
## Review Comment

This keeps scope text formatting centralized and avoids drift between list/detail views.

## Triage

- Decision: `valid`
- Root cause: `knowledge-list-panel.tsx` duplicates short scope label logic inline instead of using the shared `knowledgeScopeShortLabel` formatter.
- Fix approach: import and use `knowledgeScopeShortLabel(scope)` for the list badge label.
