---
status: resolved
file: internal/network/capability_catalog.go
line: 49
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58BM4F,comment:PRRC_kwDOR5y4QM65LrXr
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't collapse malformed `agh.capability_ids` into “no filter”.**

`decodeExtensionStringList()` returns `nil` for both “extension missing” and “JSON decode failed”. With a valid `agh.include` but malformed `agh.capability_ids`, this request is treated as unfiltered and the responder returns the full capability catalog instead of failing closed. Please preserve presence/error information here and either reject malformed requests or treat them as an empty projection.



Also applies to: 143-157

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/capability_catalog.go` around lines 39 - 49,
parseWhoisCapabilityDiscoveryRequest collapses a malformed agh.capability_ids
into nil (indistinguishable from "missing") so requests are treated as
unfiltered; fix by checking whether the extension key whoisCapabilityIDsExtKey
is present in the ExtensionMap before trusting decodeExtensionStringList: if the
key is present and decodeExtensionStringList returns nil, treat that as a
malformed/empty projection by setting request.capabilityIDs to an empty slice
(not nil) or return an error (choose one policy), and if the key is absent leave
capabilityIDs nil; apply the same presence-check + empty-slice-on-malformed
behavior to the analogous code referenced at lines 143-157 so malformed JSON
does not become "no filter."
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `parseWhoisCapabilityDiscoveryRequest()` used `decodeExtensionStringList()` for `agh.capability_ids`, but that helper returns `nil` for both "extension key absent" and "present but malformed / undecodable". `projectWhoisCapabilityCatalog()` then treated both `nil` and an explicit empty filter as "no filter" because it only checked `len(filter) > 0`, so malformed or empty capability filters could return the full catalog instead of failing closed.
- Resolution: `internal/network/capability_catalog.go` now preserves the absence of `agh.capability_ids` as `nil`, converts present-but-malformed/empty capability filters into an explicit empty slice, and treats that explicit empty slice as an empty projection when building the capability catalog response.
- Verification:
  - `go test ./internal/network -count=1`
  - `make verify`
