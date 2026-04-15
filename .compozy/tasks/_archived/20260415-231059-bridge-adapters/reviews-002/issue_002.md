---
status: resolved
file: extensions/bridges/discord/provider_test.go
line: 645
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Odct,comment:PRRC_kwDOR5y4QM64G4Yq
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Avoid `time.Sleep()` for test orchestration here.**

This makes the initialize/shutdown helper test timing-sensitive and prone to flakes under slower CI scheduling.

As per coding guidelines, "Never use `time.Sleep()` in orchestration — use proper synchronization primitives".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/discord/provider_test.go` around lines 641 - 645, The test
currently uses time.Sleep after calling provider.handleInitialize, which is
flaky; instead, wait deterministically for initialization to finish by replacing
the sleep with a synchronization primitive (e.g., wait on a ready/initialized
channel or a sync.WaitGroup) exposed by the provider or returned by
handleInitialize; if no such signal exists, change provider.handleInitialize
(and/or the provider struct) to signal completion (eg. close an initDone channel
or decrement a WaitGroup) and have the test block on that signal before
proceeding or asserting.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The test currently sleeps for `20ms` after `handleInitialize`, but `handleInitialize` only launches `afterInitialize` in a goroutine and `handleShutdown` already synchronizes with that work via `p.wg.Wait()`.
  - Root cause: the test is using a timing delay where no state transition actually needs one.
  - Outcome: removed the `time.Sleep` and left shutdown to provide the deterministic synchronization the test already had available. Verified with `go test ./extensions/bridges/discord ./extensions/bridges/gchat` and `make verify`.
