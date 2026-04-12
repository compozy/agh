---
status: resolved
file: internal/extension/host_api.go
line: 763
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:5b5c9fc618f5
review_hash: 5b5c9fc618f5
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 030: Consider making seed poll configuration injectable.
## Review Comment

The hardcoded 5ms poll interval and 30ms window are quite aggressive. While functional, exposing these as options (similar to `channelIngestDedupTTL`) would improve testability and allow tuning in production.

## Triage

- Decision: `invalid`
- Why: The `5ms`/`30ms` seed polling values are an internal bounded retry window used only to bridge the small async gap between prompt submission and persisted event visibility. They are not part of the production contract and they already sit next to the logic they constrain.
- Why not fix: Adding another public Host API option surface purely to tune this narrow internal retry loop would expand configuration and test setup without a concrete production requirement. The real testability issue in this area is the direct `time.Now()` call, which is addressed separately in issue `031`.
- Resolution: Analysis complete; no configuration-surface change required.
