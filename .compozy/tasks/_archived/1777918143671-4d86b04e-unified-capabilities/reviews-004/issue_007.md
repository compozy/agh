---
status: resolved
file: internal/network/capability_catalog.go
line: 203
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4149203771,nitpick_hash:f349f497de18
review_hash: f349f497de18
source_review_id: "4149203771"
source_review_submitted_at: "2026-04-21T16:10:13Z"
---

# Issue 007: Consider logging malformed extension data for observability.
## Review Comment

JSON unmarshal errors are silently converted to "unknown" status. While this is appropriate for defensive handling of external/untrusted extension data, logging at debug level would aid troubleshooting when capability catalogs fail to decode.

## Triage

- Decision: `invalid`
- Reasoning: `decodeWhoisCapabilityCatalogResponseExt()` is a pure decoding helper with no logger dependency and no error return path. Malformed peer extension data is intentionally treated as "catalog unknown" at this layer. Adding logging here would either introduce a global logger or thread observability concerns through a low-level codec API, which is an architectural change rather than a correctness fix for this file.
