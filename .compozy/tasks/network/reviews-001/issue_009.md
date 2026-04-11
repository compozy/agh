---
status: resolved
file: internal/config/config.go
line: 560
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZd,comment:PRRC_kwDOR5y4QM623eZu
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Bound second-based network settings before converting them to `time.Duration`.**

`GreetIntervalDuration()` and `MaxReplayAgeDuration()` multiply validated `int` values by `time.Second`, but `Validate()` never caps those fields. Large configs can overflow into negative or otherwise corrupted durations while still passing validation. Please add an upper bound for both values in `NetworkConfig.Validate()`, or store them as `time.Duration` directly.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/config.go` around lines 540 - 560, The GreetIntervalDuration
and MaxReplayAgeDuration can overflow when converting large int seconds to
time.Duration because Validate() only enforces positivity; update
NetworkConfig.Validate() to also enforce an upper bound so
time.Duration(c.X)*time.Second cannot overflow (e.g., compute maxSeconds :=
int64(math.MaxInt64)/int64(time.Second) and ensure c.GreetInterval and
c.MaxReplayAge are <= maxSeconds, performing safe casts), and return a clear
error message if they exceed the allowed maximum; adjust error text for
GreetInterval/MaxReplayAge to reflect the new range and keep
GreetIntervalDuration()/MaxReplayAgeDuration() unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `NetworkConfig.Validate` enforces positivity for `GreetInterval` and `MaxReplayAge` but never bounds them before `time.Duration(seconds) * time.Second`, so very large values can overflow.
- Fix approach: add an upper-bound check based on the maximum representable seconds in `time.Duration` and keep the duration helper methods unchanged.
