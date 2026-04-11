---
status: resolved
file: internal/api/udsapi/extensions_additional_test.go
line: 119
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:ec135b820ed9
review_hash: ec135b820ed9
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 004: Split unrelated approve-session check into a dedicated test.
## Review Comment

`extensionStatusCode` mapping and `/api/sessions/sess-1/approve` behavior are separate concerns. Splitting them improves failure localization and keeps each test scoped to one business behavior.

## Triage

- Decision: `valid`
- Root cause: `TestExtensionStatusCodeMappingsAndApproveSession` mixes pure status-code mapping coverage with the unrelated approve-session transport behavior.
- Fix approach: extract the approve-session assertion into its own test so failures stay scoped to one behavior.
