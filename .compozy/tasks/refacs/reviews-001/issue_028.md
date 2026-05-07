---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bundles/ids.go
line: 9
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:e7f94f6ac41d
review_hash: e7f94f6ac41d
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 028: strings.TrimSpace called twice per part — consider pre-trimming to a local slice.
## Review Comment

The size loop (lines 11–16) and the append loop (lines 18–23) both call `strings.TrimSpace` on every element. Since this is a deterministic ID utility it isn't a hot-path concern, but trimming once into a local slice makes the intent clearer and removes the redundancy.

## Triage

- Decision: `valid`
- Root cause: `stableID` trims each input twice, once for sizing and once for payload assembly. The behavior is correct, but the implementation duplicates normalization work and obscures the intended canonicalization step.
- Fix plan: pre-trim into a local slice once, reuse it for sizing and payload assembly, and keep the stable ID output unchanged.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
