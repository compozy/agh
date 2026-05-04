---
status: resolved
file: internal/extension/host_api_test.go
line: 5243
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:9883897aa3dc
review_hash: 9883897aa3dc
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 006: Make the recording session manager fail fast on unexpected methods.
## Review Comment

This stub returns zero values for `Status`, `Events`, `Prompt`, and `ExecEnvironment`, so a future `createBridgeSession` regression can start calling them without this test clearly failing. Returning explicit `"unexpected ... call"` errors would keep the test honest.

## Triage

- Decision: `valid`
- Root cause: the recording session-manager stub returns zero-value success from methods that this test does not intend to exercise, so an accidental new method call could slip through unnoticed.
- Fix plan: make the unused methods fail fast with explicit `unexpected ... call` errors so the test only passes when `createBridgeSession` stays within the intended interaction surface.
