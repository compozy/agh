---
status: resolved
file: web/src/systems/knowledge/lib/knowledge-formatters.ts
line: 39
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:c2673579b22a
review_hash: c2673579b22a
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 016: Tighten TYPE_TONE key typing to MemoryType for safer evolution.
## Review Comment

Using `Record<string, PillTone>` weakens compile-time guarantees. Prefer keying by `MemoryType` so invalid/typo keys are rejected by TypeScript.

## Triage

- Decision: `valid`
- Root cause: `TYPE_TONE` is typed as `Record<string, PillTone>`, so TypeScript cannot catch missing or misspelled `MemoryType` keys.
- Fix approach: key the map as `Record<MemoryType, PillTone>` and return directly from the exhaustive map.
