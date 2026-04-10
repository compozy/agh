---
status: resolved
file: internal/daemon/daemon.go
line: 321
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:eacd326b76ca
review_hash: eacd326b76ca
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 016: Silent nil return when Registry is unavailable may mask configuration issues.
## Review Comment

When `deps.Registry == nil`, the factory returns `nil` without logging or signaling. This silent degradation could make it harder to diagnose why extensions aren't working.

## Triage

- Decision: `invalid`
- Notes: The default `newExtensionManager` factory only returns `nil` when `deps.Registry == nil`, but `bootExtensions` constructs a concrete registry first and logs if the factory still returns `nil`. In other words, the silent branch in the default factory is not reachable in the supported boot path, and the composition root already emits a warning for the externally visible failure mode. There is no masked runtime bug here.
