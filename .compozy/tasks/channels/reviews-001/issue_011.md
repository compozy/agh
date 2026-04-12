---
status: resolved
file: internal/api/testutil/apitest.go
line: 185
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:ba59ff0436ba
review_hash: ba59ff0436ba
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 011: Add a compile-time assertion for StubChannelService.
## Review Comment

This file already pins the other stubs to their interfaces, but the new channel stub is missing the same guard. Adding `var _ core.ChannelService = (*StubChannelService)(nil)` near the existing assertions will make interface drift fail fast.

As per coding guidelines, "Use compile-time interface verification: `var _ Interface = (*Type)(nil)`".

## Triage

- Decision: `valid`
- Notes:
  - `StubChannelService` is missing the same compile-time interface pinning that the other API test doubles use.
  - I will add a `var _ core.ChannelService = (*StubChannelService)(nil)` assertion so interface drift fails at compile time.
  - Resolution: Added the compile-time interface assertion in [internal/api/testutil/apitest.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/testutil/apitest.go:202); verified with `make verify`.
