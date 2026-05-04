---
status: resolved
file: internal/store/session_liveness.go
line: 45
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyy1,comment:PRRC_kwDOR5y4QM654Np1
---

# Issue 031: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate `StallReason` when `StallState` is set.**

`Validate()` currently accepts any non-empty `StallReason` as long as `StallState` is present, and it also allows `SessionStallStateDetected` with an empty reason. That means invalid persisted combinations like `stalled/"foo"` or `stalled/""` still pass validation and leak impossible state into recovery code.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/session_liveness.go` around lines 29 - 45, The Validate method
must require a non-empty StallReason whenever a StallState is set; update
SessionLivenessMeta.Validate to trim and check m.StallState and if it's
non-empty (including when equal to SessionStallStateDetected) ensure
strings.TrimSpace(m.StallReason) is non-empty and return an error like "store:
session stall reason required when stall state is set" otherwise keep the
existing check that disallows a non-empty StallReason when StallState is empty;
reference the Validate function and the fields StallState, StallReason, and the
constant SessionStallStateDetected when making this change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `SessionLivenessMeta.Validate()` currently rejects a reason without a state, but it still allows a non-empty `StallState` with an empty reason. That admits impossible persisted stall combinations and weakens recovery invariants.
  I will require a trimmed `StallReason` whenever `StallState` is set and cover the behavior through scoped store/globaldb tests.
  Fixed and verified with targeted package tests plus `make verify`.
