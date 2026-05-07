---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/modelcatalog/live_sources_test.go
line: 474
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6s9,comment:PRRC_kwDOR5y4QM6-6bss
---

# Issue 019: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Make the coalescing assertions deterministic instead of time-based.**

The first case relies on `time.Sleep(50 * time.Millisecond)` to give the second refresh time to join, and the second only watches `secondSource.started` for `30ms` before releasing the first source. Both tests can still pass without actually proving the coalescing/serialization behavior on a slow or busy CI machine. An explicit barrier/ack in `blockingProviderSource` would make these checks reliably fail when the concurrency contract regresses.

As per coding guidelines, "Verify tests can fail when business logic changes."
 


Also applies to: 527-538

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/live_sources_test.go` around lines 449 - 474, The
test's coalescing assertions use time.Sleep and short waits which are flaky;
update the blockingProviderSource (created by newBlockingProviderSource) to
expose an explicit synchronization channel/ack (e.g., a "joined" or "ready"
channel) that signals when a goroutine has reached the blocking point, then
change the test to wait on that channel(s) instead of sleeping: in the first
subtest wait for both refresh goroutines to signal they've entered the
provider's block before calling source.release(), and in the second subtest wait
for the second source's "joined" signal (secondSource.joined) or a timeout
before releasing the first; use these explicit acknowledgements to
deterministically assert the coalescing/serialization behavior rather than
relying on time.Sleep.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The coalescing tests use `time.Sleep(...)` and short timing windows to infer concurrency behavior.
  - Those assertions are flaky and can pass or fail based on machine scheduling rather than the intended serialization/coalescing contract.
  - Fix: add explicit synchronization signals to `blockingProviderSource` and assert on those barriers instead of elapsed sleeps.
