---
status: resolved
file: internal/api/contract/tasks.go
line: 479
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133463506,nitpick_hash:68728df3a7b0
review_hash: 68728df3a7b0
source_review_id: "4133463506"
source_review_submitted_at: "2026-04-18T03:54:22Z"
---

# Issue 001: Consider using *bool for Unread to support three-state filtering.
## Review Comment

With `Unread bool`, you cannot distinguish between "not specified" (show all) and "explicitly false" (show only read items). If filtering to only-read items is a valid use case, consider using `*bool`.

Current semantics appear to be:
- `unread` omitted → show all
- `unread=true` → show only unread
- `unread=false` → treated same as omitted

## Triage

- Decision: `INVALID`
- Reasoning: The current inbox query contract intentionally models `unread` as a one-way opt-in filter, not a three-state selector. In `internal/observe/tasks.go`, `TaskInboxQuery.Unread` only gates `taskInboxFromSnapshot` when it is `true`; `false` and omission both mean "do not apply an unread-only filter." There is no existing server behavior, parser contract, or product requirement in this batch for a distinct "read-only" mode.
- Root cause analysis: This is not a correctness bug in the current implementation. It is a speculative API expansion request that would require a transport and domain contract change across inbox parsing, validation, and filtering semantics.
- Intended action: No code change. Close this issue as analysis-complete.
- Resolution: Closed as invalid after code-path review; no change was made because the reported behavior matches the current contract semantics.
