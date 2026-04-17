---
status: resolved
file: internal/testutil/acpmock/cmd/acpmock-driver/main.go
line: 457
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57ziMc,comment:PRRC_kwDOR5y4QM645avM
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep async driver-control work bound to the agent lifetime.**

This branch detaches delayed control actions from the prompt that scheduled them, so a late disconnect/raw write can land after the prompt has already finished and interfere with the next prompt or teardown. Please give these goroutines explicit ownership plus shutdown/wait handling instead of letting them run untracked.


As per coding guidelines, "Every goroutine must have explicit ownership and shutdown via context.Context cancellation" and "No fire-and-forget goroutines — track with sync.WaitGroup or equivalent".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/cmd/acpmock-driver/main.go` around lines 448 - 457,
The async branch detaches driver-control goroutines from the agent lifetime;
change it to register each async worker with the agent's lifecycle (e.g., a
shared sync.WaitGroup or a per-agent worker tracker) and use a context derived
from the agent/connection context so cancellation cancels pending delays;
specifically, when control.Async is true, create a child context tied to the
agent lifetime (not context.WithoutCancel), increment the agent's
WaitGroup/worker tracker before spawning the goroutine, run
waitDriverControlDelay and performDriverControl inside that goroutine, and
ensure you call Done()/unregister on exit so the agent can cancel and Wait for
all async driver-control goroutines during teardown. Ensure references to
waitDriverControlDelay, performDriverControl, and DriverControlStep are used to
locate and update the code.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `executeDriverControl` currently launches async control work with `context.WithoutCancel(ctx)` and no tracking, which violates the repo’s goroutine-lifecycle rules and lets delayed actions outlive teardown.
  - That can leak disconnect/raw-write work past the intended agent lifetime and makes the mock driver’s fault scenarios timing-dependent across prompt boundaries.
  - Implemented: async driver-control workers are now tracked with an agent `sync.WaitGroup`, bounded by both the prompt lifetime and the agent lifetime, and drained on connection shutdown.
  - Regression coverage: added `TestAsyncDriverControlIsCanceledWhenPromptCompletes` to prove a completed prompt cancels a delayed async disconnect instead of letting it kill the next prompt.
  - Verification: `go test ./internal/testutil/acpmock -count=1`; `make verify`.
