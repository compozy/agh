---
status: resolved
file: internal/api/spec/spec.go
line: 280
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQY,comment:PRRC_kwDOR5y4QM64dqGg
---

# Issue 019: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`/api/resources` is documented as UDS-only, but the HTTP server now exposes it too.**

These operations are all tagged with `[]Transport{TransportUDS}`, while `internal/api/httpapi/resources_test.go` now asserts the same routes are registered on the HTTP API when operator auth is configured. That leaves the generated contract understating a remote mutation surface and makes transport-based docs/clients/auth review incorrect. Either add `TransportHTTP` here or stop registering the HTTP routes.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/spec/spec.go` around lines 169 - 280, The API spec routes for
resource operations (OperationID values listResources, listResourcesByKind,
getResource, putResource, deleteResource with Path "/api/resources" and
variants) are currently declared with only TransportUDS; update each route's
Transports slice to include TransportHTTP as well (e.g.,
[]Transport{TransportUDS, TransportHTTP}) so the generated contract matches the
HTTP routes asserted in internal/api/httpapi/resources_test.go, or alternatively
stop registering the HTTP routes — choose the former and add TransportHTTP to
those route specs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The HTTP server now exposes `/api/resources` when operator auth is configured, but the OpenAPI operation registry still marks those routes as UDS-only. That is a real contract drift problem and should be corrected in `internal/api/spec/spec.go` together with the corresponding spec test.
