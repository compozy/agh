---
status: resolved
file: internal/skills/catalog.go
line: 112
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:1c4db2d2714e
review_hash: 1c4db2d2714e
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 016: Fix Unicode truncation bug that pads descriptions below the 200-rune limit.
## Review Comment

The current implementation incorrectly truncates descriptions with 198–200 runes when byte length exceeds 200. For multibyte characters like "界" (3 bytes each), a 198-rune string is 594 bytes and triggers truncation, producing 197 runes + ellipsis (200 total)—padding short descriptions rather than preserving them. Only truncate when rune count exceeds `catalogDescriptionLimit` (201+).

## Triage

- Decision: `VALID`
- Notes:
  `truncateCatalogDescription` uses `len(description)` as a byte-length gate, so
  multibyte strings at or below the 200-rune limit are truncated when their byte
  length exceeds 200. Plan: gate truncation on rune count instead and add a
  unicode regression test below the limit.
