---
status: resolved
file: extensions/bridges/github/provider.go
line: 264
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwm2,comment:PRRC_kwDOR5y4QM64DQ0k
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don’t fail deliveries while async initialization is still building routes.**

`handleInitialize` returns before `afterInitialize` finishes populating `p.routes`, but delivery only waits 500ms for config availability. Any slow host/API call during startup can make the first deliveries fail with "unmanaged bridge instance" even though initialization already succeeded. Either block initialization until routes are ready or wait on an explicit readiness signal instead of a fixed short timeout.



Also applies to: 329-335

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/github/provider.go` around lines 258 - 264,
handleInitialize currently returns before afterInitialize finishes populating
p.routes which allows deliveries to proceed (with only a 500ms wait elsewhere)
and fail as "unmanaged bridge instance"; fix by making initialization wait for
routes to be ready instead of launching afterInitialize in a detached goroutine.
Replace the async goroutine call around p.afterInitialize(session) with a
synchronous call or add an explicit readiness signal (e.g., a ready channel or
use p.wg/WaitGroup with a ready chan) that is closed/fulfilled at the end of
afterInitialize and have the caller (or the delivery wait logic) block on that
readiness signal instead of a fixed timeout; apply the same change for the other
occurrence mentioned (lines around the second async block) to ensure p.routes is
fully populated before handleInitialize returns or before deliveries proceed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `handleInitialize()` still launches `afterInitialize()` asynchronously, while `handleBridgesDeliver()` only waits a fixed 500ms for `p.routes` to appear.
  - The current startup path does enough host/API work before publishing routes that early deliveries can still fail as unmanaged even though initialization has already returned success.
  - Planned fix: introduce explicit initialization readiness for route publication instead of relying on the short polling window, and add a regression test that exercises delayed startup without dropping the first delivery.
  - Resolution: GitHub initialization now exposes an explicit readiness signal, publishes fresh routes before initial-state probing, and `handleBridgesDeliver()` waits on readiness instead of a fixed 500ms timeout; the helper coverage was updated to exercise the delayed-startup path.
  - Verification: `go test ./extensions/bridges/github -count=1` and `make verify` both passed after the fix.
