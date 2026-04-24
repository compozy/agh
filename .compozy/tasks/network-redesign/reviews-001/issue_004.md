---
status: resolved
file: internal/api/core/coverage_helpers_test.go
line: 125
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeH,comment:PRRC_kwDOR5y4QM66CAkp
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Add one case for the new persisted-channel input.**

These assertions only adapt the call sites to the new `networkChannelExists(..., ..., ..., channel)` signature, but they still pass `nil` for the new source every time. That leaves the durable-channel branch introduced in this PR untested.


As per coding guidelines, "Focus on critical paths: workflow execution, state management, error handling".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/coverage_helpers_test.go` around lines 118 - 125, The tests
call networkChannelExists(...) with nil for the new persisted-channel parameter,
leaving the durable/persisted-channel branch untested; update
coverage_helpers_test.go to add at least one assertion that passes a non-nil
persistedChannels value (e.g., a slice/map containing "match" or
"sessionChannel") into networkChannelExists and assert true, and add a
complementary case where persistedChannels does not include the channel and
assert false so the persisted-channel branch in networkChannelExists is covered.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: the current helper coverage only exercises `networkChannelExists()` through session and peer matches plus the missing case. The persisted metadata branch (`metadata != nil`) is reachable in production but currently uncovered.
- Fix plan: add explicit persisted-metadata true/false coverage in `coverage_helpers_test.go` and keep the helper assertions organized as named subtests.
- Resolution: added explicit coverage for the persisted-metadata branch in `networkChannelExists()` and kept the helper assertions organized as named subtests.
- Verification: `go test ./internal/api/core` and `make verify`
