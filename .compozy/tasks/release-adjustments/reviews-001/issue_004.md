---
status: resolved
file: internal/api/contract/contract.go
line: 67
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk08,comment:PRRC_kwDOR5y4QM67HMV-
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep zero-valued runtime metrics in the JSON payload.**

`omitempty` on `idle_seconds`, `elapsed_seconds`, and the iteration counters drops legitimate `0` values. Clients then cannot distinguish “0 seconds / 0 iterations” from “field not populated”. Either remove `omitempty` for those metrics or make them pointers if absence is intentional.



Also applies to: 188-205

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/contract/contract.go` around lines 53 - 67, The runtime metric
fields in RuntimeActivityPayload (IdleSeconds, ElapsedSeconds, IterationCurrent,
IterationMax) are using `omitempty`, which drops legitimate zero values; update
the struct so clients can distinguish 0 from absent by either removing
`omitempty` from the json tags for those fields or changing their types to
pointers (e.g., *int, *int64) and ensuring construction code sets nil vs zero
appropriately; apply the same change to the corresponding fields in the other
related struct referenced around lines 188-205 so both payloads preserve
zero-valued metrics.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `RuntimeActivityPayload` and `SessionActivityHealthPayload` use `omitempty` on zero-valued runtime metrics, which drops meaningful `0` values from JSON.
  - The fix is to remove `omitempty` from `iteration_current`, `iteration_max`, `idle_seconds`, and `elapsed_seconds`, and add JSON shape coverage for zero metrics.
