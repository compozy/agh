---
status: resolved
file: internal/extension/manager_test.go
line: 31
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107909499,nitpick_hash:e5bde992e722
review_hash: e5bde992e722
source_review_id: "4107909499"
source_review_submitted_at: "2026-04-14T17:26:14Z"
---

# Issue 012: Pin the new sink stub to its interface.
## Review Comment

Since `noopBridgeTelemetrySink` is now a reusable test double, add a compile-time interface assertion next to it so interface drift fails immediately instead of only where the stub is wired in.

As per coding guidelines, `**/*.go`: "Use compile-time interface verification: `var _ Interface = (*Type)(nil)`".

## Triage

- Decision: `valid`
- Notes:
  `noopBridgeTelemetrySink` is a reusable test double without compile-time interface pinning, so interface drift would surface later and less locally. I will add a compile-time assertion next to the stub definition.
  Resolution: Added a compile-time `BridgeTelemetrySink` assertion beside the reusable sink stub.
