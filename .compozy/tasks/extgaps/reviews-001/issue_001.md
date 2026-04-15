---
status: resolved
file: internal/api/core/bundles.go
line: 279
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Bc,comment:PRRC_kwDOR5y4QM63zbx2
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Return the materialized resource IDs here.**

These payload IDs are built as `job:activation:name` / `trg:...` / `bri:...`, but the bundle service materializes the real managed IDs with the hashed `stableID(...)` helper. That means the IDs in `Jobs`, `Triggers`, and `Bridges` do not match the IDs persisted in `Inventory` or used by the automation/bridge managers, so clients cannot reliably correlate the sections.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/bundles.go` around lines 214 - 279, The
Jobs/Triggers/Bridges IDs in BundleActivationPayload are built with
bundlepkgStableID(...) but the persisted/materialized IDs use the hashed
stableID(...) helper, causing mismatch with Inventory and managers; update the
payload construction in BundleActivationPayload to call
stableID("job"/"trg"/"bri", item.Activation.ID,
job.Name/trigger.Name/bridge.Name) (instead of bundlepkgStableID) so the
returned IDs match the materialized IDs used elsewhere (ensure stableID is
referenced/imported where BundleActivationPayload is defined).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `BundleActivationPayload` still uses the local `bundlepkgStableID` helper, which returns colon-joined IDs like `job:<activation>:<name>`, while bundle materialization persists hashed IDs from `stableID(...)` (`job_<hash>`, `trg_<hash>`, `bri_<hash>`). The payload therefore does not match `Inventory` or the manager/store identifiers.
- Fix plan: update the API payload helper to generate the same hashed stable IDs used by bundle materialization so response IDs, persisted inventory IDs, and managed resource IDs stay aligned.
- Resolution: replaced the local API helper with the same hashed stable-ID algorithm used by bundle materialization.
- Verification: added coverage in `internal/api/core/network_test.go` and passed `go test ./internal/api/core` plus the full `make verify` gate.
