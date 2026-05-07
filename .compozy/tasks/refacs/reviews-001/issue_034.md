---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/cli/format.go
line: 195
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:0b8322e44b4e
review_hash: 0b8322e44b4e
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 034: Keep underline and separator sizing on the same width metric.
## Review Comment

The new renderer uses `humanTableCellWidth` for column sizing, but these spots still use `len(...)`. For non-ASCII titles or headers that mixes byte length with rune width, so the underline/separator can render too wide and even expand the computed table width.

Also applies to: 218-218, 225-227

## Triage

- Decision: `valid`
- Root cause: the human renderer computes table widths with rune counts but still sizes section underlines and separator rows with `len(...)`, which uses byte length and can mis-size non-ASCII headers or titles.
- Fix plan: switch those width calculations to the same `humanTableCellWidth` metric used for column sizing.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
