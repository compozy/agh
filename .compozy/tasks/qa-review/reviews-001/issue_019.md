---
status: resolved
file: web/src/hooks/routes/use-knowledge-page.ts
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:c8a154181f5b
review_hash: c8a154181f5b
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 019: Don’t overwrite incoming memory.key during decoration.
## Review Comment

Always re-synthesizing the key risks divergence from backend/canonical identity. Prefer preserving provided keys and only backfilling when absent.

## Triage

- Decision: `valid`
- Notes:
  - `decorateKnowledgeMemories()` always rewrites `memory.key` as `${scope}:${filename}` even when a canonical key is already present.
  - Root cause: decoration currently treats `key` as derived-only instead of preserving backend-provided identity and only backfilling when absent.
  - Fix plan: preserve `memory.key` if supplied, synthesize only missing keys, and add a hook regression that asserts the existing key survives decoration.
