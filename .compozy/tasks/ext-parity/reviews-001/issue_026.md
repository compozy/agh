---
status: resolved
file: internal/bundles/resource_projection.go
line: 91
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQq,comment:PRRC_kwDOR5y4QM64dqG4
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Plan revision needs to include bundle-resource versions too.**

`Build` only advances `revision` from `activationRecords`, but the projected output also depends on `bundleRecords`. A bundle/profile change can materially change `desiredJobs` / `desiredTriggers` / `desiredBridges` while `Revision()` stays unchanged.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/bundles/resource_projection.go` around lines 63 - 91, The current
revision value is computed only from activationRecords (variable revision) so
changes in bundleRecords won't bump the plan revision; update the revision
computation in the function that builds the BundleActivationResourcePlan to
consider bundleRecords as well by iterating bundleRecords and taking the max of
record.Version across both activationRecords and bundleRecords (keep using the
existing revision variable), so the returned
BundleActivationResourcePlan.revision reflects the highest resource version from
both activationRecords and bundleRecords.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `Service.Build` still derives the plan revision only from activation resource versions even though bundle resource records also materially change desired jobs, triggers, and bridges. A newer bundle record can therefore change the output while `Revision()` stays stale. The fix is to compute the max version across both activation and bundle records and update the bundle projection test accordingly.
