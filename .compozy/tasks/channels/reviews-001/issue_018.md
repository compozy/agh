---
status: resolved
file: internal/channels/types.go
line: 175
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLn,comment:PRRC_kwDOR5y4QM623eJA
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`normalizeRawJSON` is validated here, but its normalized output is discarded.**

Both validators call `normalizeRawJSON(...)` only for error checking, so semantically identical JSON with different formatting remains non-canonical in `ChannelInstance.DeliveryDefaults` and `DeliveryEvent.Metadata`. That breaks the normalization story this package is trying to provide and can cause false diffs or duplicate-state comparisons downstream.



Also applies to: 356-357

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/channels/types.go` around lines 174 - 175, The code calls
normalizeRawJSON(...) only to check for errors but discards the normalized
output; instead capture the normalized result returned by normalizeRawJSON and
assign it back to the canonical fields so they remain canonical (e.g., set
normalized.DeliveryDefaults = <normalized output> after validate call and
similarly set DeliveryEvent.Metadata = <normalized output> where the validator
currently calls normalizeRawJSON for error-only checking); update the validators
that reference normalizeRawJSON to both validate and replace the original value
with the normalized JSON string/bytes returned by normalizeRawJSON.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The persisted channel-instance write paths already canonicalize `DeliveryDefaults` in `CreateInstanceRequest.toInstance(...)` and `UpdateInstance(...)`, which are the authoritative mutation surfaces.
  - `Validate()` on `ChannelInstance` and `DeliveryEvent` is intentionally a pure validation pass over caller-owned values; changing it to mutate/canonicalize would alter that contract without fixing a demonstrated bug in the current write path.
  - Resolution: Closed as invalid after code inspection; the authoritative write paths already canonicalize persisted JSON and `make verify` passed without altering the pure validation contract.
