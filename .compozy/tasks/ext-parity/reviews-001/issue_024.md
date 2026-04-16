---
status: resolved
file: internal/bridges/resource_projection.go
line: 266
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQl,comment:PRRC_kwDOR5y4QM64dqGx
---

# Issue 024: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Byte-wise JSON comparison will cause false-positive bridge changes.**

`rawJSONEqual` treats semantically identical JSON objects as different when key order or formatting changes. That inflates `OperationCount()` / `ChangedExtensions()` and can trigger unnecessary `ReplaceBridgeInstances` calls and extension reloads.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/bridges/resource_projection.go` around lines 232 - 266, The
comparison currently uses rawJSONEqual in sameProjectedBridgeInstance which does
byte-wise compare and yields false positives; change rawJSONEqual to perform
semantic JSON equality by unmarshaling left and right into generic Go types
(e.g., interface{}), normalizing (treat nil and empty as equivalent), and then
comparing with reflect.DeepEqual (or by re-marshal to a canonical form) so that
differing key order/whitespace don't mark a change; update rawJSONEqual to
handle invalid JSON gracefully (treat non-JSON or unmarshal errors as unequal)
and ensure sameProjectedBridgeInstance continues to call rawJSONEqual for
ProviderConfig and DeliveryDefaults.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `sameProjectedBridgeInstance` still delegates provider and delivery JSON comparison to `rawJSONEqual`, which only trims and byte-compares the payload. Because bridge JSON canonicalization preserves object key order, semantically equivalent JSON can still look different and inflate projection operation counts and changed extension lists. The fix is to switch to semantic JSON equality and cover the projection case with a regression test.
