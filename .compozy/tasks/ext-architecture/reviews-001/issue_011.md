---
status: resolved
file: internal/cli/extension.go
line: 319
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:b23ebefcccef
review_hash: b23ebefcccef
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 011: Silent error swallowing when loading manifest in localExtensionRecord.
## Review Comment

The manifest loading error at line 329 is silently ignored. While the extension info can be returned without the manifest, this could hide legitimate issues (corrupted manifest, permission errors).

Consider logging the error at debug/warn level for troubleshooting.

Alternatively, add a logger parameter to emit a warning when manifest loading fails.

## Triage

- Decision: `invalid`
- Notes: `localExtensionRecord` is intentionally best-effort for offline CLI rendering: manifest data enriches the payload, but the registry row already contains the core status fields. Propagating or logging manifest-load failures here would require threading logger dependencies through pure formatting helpers and would add noisy user-visible diagnostics for non-fatal offline lookups. The real daemon status path already logs manifest read failures in `daemonExtensionService.populateManifest`, so there is no hidden production correctness bug to fix in this CLI helper.
