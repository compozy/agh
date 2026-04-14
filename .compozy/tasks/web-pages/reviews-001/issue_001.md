---
status: resolved
file: internal/api/core/bridges.go
line: 38
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:754f8ef29659
review_hash: 754f8ef29659
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 001: Prefer StatusForBridgeError(err) for consistency.
## Review Comment

This handler currently hard-codes `500` for provider listing failures; using `StatusForBridgeError(err)` would keep bridge error mapping consistent across endpoints.

## Triage

- Decision: `valid`
- Root cause: `ListBridgeProviders` is the only bridge handler still hard-coding `500` for service errors, even though the shared bridge error mapper already defines transport behavior for bridge-domain failures.
- Fix approach: route provider-list failures through `StatusForBridgeError(err)` so bridge endpoints stay consistent.
