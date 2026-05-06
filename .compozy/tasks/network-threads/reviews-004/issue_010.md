---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/api/udsapi/handlers_test.go
line: 549
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:b8ca47953c07
review_hash: b8ca47953c07
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 010: This doesn't actually verify the HTTP router surface.
## Review Comment

`got` is collected once from the UDS engine, then compared to the documented HTTP and UDS route sets. If HTTP registration drops a network route while the spec remains shared, this still passes because no HTTP router is ever instantiated here.

## Triage

- Decision: `valid`
- Notes: `TestRegisterNetworkRoutesMatchDocumentedHTTPAndUDSSurface` builds only the UDS router and compares that single route set against both documented HTTP and UDS transports. That can miss an HTTP registration drift. The test should instantiate both routers and compare each route set to its corresponding documented transport surface.
