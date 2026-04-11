---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 308
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857889,nitpick_hash:f36d4fab2f31
review_hash: f36d4fab2f31
source_review_id: "4093857889"
source_review_submitted_at: "2026-04-11T14:16:28Z"
---

# Issue 001: Add an explicit compile-time interface assertion for the channel wrapper.
## Review Comment

`integrationChannelService` is now a contract boundary for `WithChannelService`; adding `var _ <interface> = (*integrationChannelService)(nil)` here makes drift fail at compile time.

As per coding guidelines, "Use compile-time interface verification: `var _ Interface = (*Type)(nil)`".

## Triage

- Decision: `Valid`
- Notes:
  `integrationChannelService` is the test adapter passed into `WithChannelService`, whose contract is `core.ChannelService`. A compile-time assertion is the minimal way to catch interface drift in this integration-only wrapper. The fix is to add `var _ core.ChannelService = (*integrationChannelService)(nil)`.
  Resolved by adding the compile-time assertion in `internal/api/udsapi/udsapi_integration_test.go` and verifying it through `go test -tags integration ./internal/api/udsapi -count=1` plus the final `make verify` pass.
