---
status: resolved
file: internal/session/log_capture_test.go
line: 97
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:30b62906be01
review_hash: 30b62906be01
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 010: Defensively clone Attrs before returning captured records.
## Review Comment

`Records()` and `FindByMessage()` currently expose mutable `Attrs` maps by reference. A caller mutating the returned map can alter internal captured state.

Also applies to: 103-112

## Triage

- Decision: `valid`
- Root cause: `Records()` and `FindByMessage()` return `capturedLogRecord` values that still reference the handler's mutable `Attrs` maps, so caller mutation can corrupt captured test state.
- Fix plan: deep-clone the record maps before returning records from those helper methods.
