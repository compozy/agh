---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/cli/format.go
line: 192
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:57cc8dc2ff65
review_hash: 57cc8dc2ff65
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 033: Wrap the new render errors with context.
## Review Comment

These branches now surface first-class errors, but they return raw `fmt.Fprintf`/`Flush` failures unchanged. Add operation context here so callers can tell whether title emission, row formatting, or flush failed.

As per coding guidelines, "Use `%w` for error wrapping and `errors.Is`/`errors.As` for error comparison in Go code".

## Triage

- Decision: `valid`
- Root cause: `renderHumanSectionResult` returns raw `fmt.Fprintf` and `Flush` errors, so callers cannot tell whether the failure happened while writing the title, underline, rows, or final flush.
- Fix plan: wrap those error returns with operation-specific context using `%w`.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
