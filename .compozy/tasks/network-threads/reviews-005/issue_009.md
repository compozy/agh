---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/hooks/network_dispatch_test.go
line: 86
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_033y,comment:PRRC_kwDOR5y4QM6-SXvb
---

# Issue 009: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Split the per-event assertions into `t.Run("Should ...")` subtests.**

This test covers six network-event scenarios in one body, so a failure does not localize cleanly and it misses the repo’s required subtest shape. Serial subtests are fine here if you want to preserve the ordering assertion.

 

As per coding guidelines, "Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/hooks/network_dispatch_test.go` around lines 9 - 86, The test
TestDispatchNetworkHooksUseAsyncObservationPayloads currently asserts all six
events in one flat loop; split each event into its own t.Run subtest (e.g.,
t.Run(fmt.Sprintf("Should dispatch %s", event.String()), func(t *testing.T){ ...
})) so failures localize. Keep the setup (decls, executors, hooks, Rebuild)
outside the loop, then inside each subtest call the appropriate dispatch
function (DispatchNetworkThreadOpened, DispatchNetworkDirectRoomOpened,
DispatchNetworkMessagePersisted, DispatchNetworkWorkOpened,
DispatchNetworkWorkTransitioned, DispatchNetworkWorkClosed) using
networkDispatchTestPayload(event), and assert reading from the shared seen
channel with the same timeout; run subtests serially (do not call t.Parallel())
to preserve ordering.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `TestDispatchNetworkHooksUseAsyncObservationPayloads` dispatches six hook events in one loop and asserts them in one follow-up loop, so the failing scenario is only known after reading the event ordering.
  - These event checks are intentionally order-sensitive, so serial subtests are appropriate, but each event should still have its own `Should ...` case for failure locality.
  - Fix plan: keep the shared hook setup outside the loop, then dispatch and assert each event inside a serial `t.Run("Should dispatch ...")` subtest.

## Resolution

- Split the six async network hook cases into serial `Should dispatch ...` subtests and localized each assertion to the event being dispatched.
- Verified with fresh full `make verify` (passed).
