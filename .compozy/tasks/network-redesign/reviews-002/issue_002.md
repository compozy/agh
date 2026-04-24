---
status: resolved
file: internal/network/audit_test.go
line: 166
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167301360,nitpick_hash:6b688d9e463b
review_hash: 6b688d9e463b
source_review_id: "4167301360"
source_review_submitted_at: "2026-04-24T01:39:58Z"
---

# Issue 002: Refactor to a table-driven test pattern to match repository standards.
## Review Comment

The three subtests repeat identical setup: `storeSink`, `NewAuditWriter()`, `writer.now`, and assertions on `storeSink.messages`. Consolidate into a table-driven structure with `t.Run()` for each case. This aligns with the coding guideline: "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests."

## Triage

- Decision: `valid`
- Root cause: `TestAuditWriterPersistsTimelineMessagesForRenderableEnvelopes` has three sibling subtests that repeat the same `recordingAuditStore`, `NewAuditWriter`, and `writer.now` setup with case-specific assertions layered on afterward. The behavior under test is data-driven, but the test shape is still copy-pasted.
- Fix plan: Collapse the renderable-envelope cases in `internal/network/audit_test.go` into a table-driven subtest loop so setup is shared and each case only declares the record operation plus expected timeline fields.
- Outcome: Reworked the renderable-envelope coverage into one table-driven suite with shared setup. Verified with `go test ./internal/network -count=1` and `make verify`.
