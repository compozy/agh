---
status: resolved
file: internal/daemon/channels_test.go
line: 294
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tkbe,comment:PRRC_kwDOR5y4QM624L_K
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Avoid matching errors via `err.Error()` text.**

`containsErrorText(err, ...)` makes this assertion brittle to wrapping and message rewording. Prefer a sentinel/typed error with `errors.Is`/`errors.As`, or a dedicated `ErrorContains` assertion if the contract really is message-based.



As per coding guidelines, `**/*.go`: `Use errors.Is() and errors.As() for error matching — never compare error strings` and `**/*_test.go`: `MUST have specific error assertions (ErrorContains, ErrorAs)`.


Also applies to: 409-414

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/channels_test.go` around lines 291 - 294, The test currently
checks error text via containsErrorText after calling
runtime.ResolveChannelRuntime(instance.ExtensionName); replace this brittle
string match with a robust error assertion: have ResolveChannelRuntime return a
sentinel or typed error (e.g., runtime.ErrMissingSecretResolver or a typed error
value) and in the test use errors.Is (or require.ErrorIs/AssertErrorIs) to
assert that returned err matches that sentinel; if the contract must remain
message-based, replace containsErrorText with the test helper ErrorContains
assertion instead of directly matching err.Error(). Ensure references to
ResolveChannelRuntime and containsErrorText are updated accordingly and
add/export the sentinel error from the runtime package if it does not yet exist.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Valid`
- Notes:
  The test helper `containsErrorText` matches `err.Error()` directly, which is brittle against wrapping and rewording. The runtime can expose a sentinel for the missing-secret-resolver case, letting the test assert the real contract with `errors.Is`.
  Resolved by introducing `errChannelSecretResolverRequired` in `internal/daemon/channels.go` and updating `internal/daemon/channels_test.go` to assert `errors.Is` through the wrapped `ResolveChannelRuntime` error. Verified with `go test ./internal/daemon -count=1` and the final `make verify` pass.
