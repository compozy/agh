---
status: resolved
file: internal/daemon/daemon_test.go
line: 3146
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4132976935,nitpick_hash:661e281738f2
review_hash: 661e281738f2
source_review_id: "4132976935"
source_review_submitted_at: "2026-04-18T00:19:15Z"
---

# Issue 004: Extract the memory fixture serializer instead of duplicating raw frontmatter.
## Review Comment

This helper now embeds the same document shape already serialized in `internal/daemon/daemon_memory_e2e_integration_test.go:398-408`. Reusing that formatter, or extracting a shared helper, will keep both test suites aligned if frontmatter fields or newline handling change.

## Triage

- Decision: `valid`
- Notes:
  - `writeDaemonMemoryIndex` duplicates the same memory document serialization shape already provided by `memoryDocument(...)` in the E2E test file.
  - Reusing the shared formatter inside package `daemon` removes format drift risk for frontmatter/newline handling without changing test intent.

## Resolution

- Extracted the shared `memoryDocument(...)` serializer into the non-tagged daemon test file set and reused it from `writeDaemonMemoryIndex`, so both unit and integration test builds share one formatter.
