---
status: resolved
file: internal/daemon/daemon_test.go
line: 3078
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLr,comment:PRRC_kwDOR5y4QM623eJE
---

# Issue 023: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid fixed sleeps in these helper scenarios.**

Both branches now depend on wall-clock timing to force process exit / delivery slowness. That tends to make this harness flaky on slower CI and directly couples the assertions to scheduler timing instead of an explicit synchronization point. As per coding guidelines, "`**/*.go`: Never use `time.Sleep()` in orchestration — use proper synchronization primitives`".



Also applies to: 3091-3097

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 3074 - 3078, The helper
scenarios use time.Sleep to drive a goroutine that calls os.Exit (when
h.scenario == "auto_exit_record_initialize") and a similar sleep elsewhere,
which makes tests flaky; replace the sleep-based orchestration with explicit
synchronization: add a channel or sync.WaitGroup that the test harness can
close/signal at the exact point you want the goroutine to call os.Exit (or
simulate slow delivery) and wire that into the existing goroutine logic that
currently references h.scenario so it blocks on the channel instead of
time.Sleep; update both the "auto_exit_record_initialize" path and the other
scenario (the similar block around the 3091–3097 region) to wait on the same
explicit signal so test timing is deterministic.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Why: The daemon helper uses `time.Sleep()` to force post-initialize exit and delayed delivery behavior. Those sleeps couple the tests to scheduler timing and can make restart/shutdown integration coverage flaky under slower CI.
- Root cause: The helper scenarios encode ordering with wall-clock delays instead of explicit synchronization around "initialize response sent", "delivery in flight", and "shutdown has started".
- Fix plan: Replace sleep-based orchestration with deterministic helper coordination: exit only after the initialize request has been fully handled, and gate delayed delivery acknowledgements on an explicit release signal instead of a timed sleep.
- Resolution: Reworked the daemon helper to use deterministic exit/release coordination and verified the restart/shutdown integration scenarios plus `make verify`.
