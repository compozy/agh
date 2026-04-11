---
status: resolved
file: internal/config/config.go
line: 378
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:24d18a8ebcdf
review_hash: 24d18a8ebcdf
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 008: Wrap network validation errors with context.
## Review Comment

This is the only nested config validation here that returns the raw error. Wrapping it keeps failures actionable once more network checks land.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

## Triage

- Decision: `valid`
- Root cause: top-level `Config.Validate` wraps hook validation with context but returns raw `Network.Validate` errors, so nested network failures are less actionable.
- Fix approach: wrap the network validation failure with config-level context, matching the surrounding validation pattern.
