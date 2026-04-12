---
status: resolved
file: internal/network/manager_test.go
line: 252
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56T_fL,comment:PRRC_kwDOR5y4QM624toa
---

# Issue 018: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Move these standalone cases into `t.Run("Should...")` subtests.**

Most of this file is still organized as long top-level tests instead of the repo’s default table-driven/subtest style. That makes failures harder to localize and misses the required `Should...` naming convention for cases.


```shell
#!/bin/bash
# Compare top-level tests vs. existing "Should..." subtests in this file.
rg -n '^func Test' internal/network/manager_test.go
rg -n 't\.Run\("Should' internal/network/manager_test.go
```

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default", "Use `t.Parallel()` for independent subtests", and "MUST use t.Run(\"Should...\") pattern for ALL test cases".



Also applies to: 305-841

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/manager_test.go` around lines 44 - 252, Tests in
TestNewManagerRequiresEnabledConfigAndPrompter,
TestNewManagerReportsRollbackShutdownFailures,
TestManagerJoinSendStatusAndLeave, and
TestManagerQueuesBusyDeliveriesTracksDisconnectsAndShutsDownIdempotently must be
converted from long top-level test bodies into subtests using t.Run("Should...")
for each independent case; for each case create a t.Run with the descriptive
"Should..." name, call t.Parallel() inside the subtest, move the relevant
assertions/setup into that subtest closure, and preserve existing helpers (e.g.
newFakeDeliveryPrompter, testManagerConfig, mustRawJSON, manager methods) and
defer/cleanup semantics; ensure repeated/shorthand cases (like repeated
LeaveSpace and repeated Shutdown) become their own t.Run cases or are clearly
named subcases so failures localize.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: The first manager tests are long top-level flows that combine multiple independent assertions, which makes failures coarse and diverges from the repository’s `Should...` subtest convention.
- Fix plan: Refactor the affected tests into focused `t.Run("Should...")` cases, using `t.Parallel()` only where the cases are actually independent.
