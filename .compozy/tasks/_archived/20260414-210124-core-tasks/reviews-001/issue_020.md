---
status: resolved
file: internal/daemon/daemon_test.go
line: 755
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562aoS,comment:PRRC_kwDOR5y4QM63mgRl
---

# Issue 020: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**This only covers the “all survivors are healthy” path.**

Because the fixture set contains just one extension and that one is reported healthy/registered, this test never proves that the fallback excludes unhealthy or unregistered extensions. Add a second installed extension that `runtime.Get()` reports as missing, disabled, or unregistered, then assert it does *not* survive the partial-start recovery.



As per coding guidelines, `**/*_test.go`: "Focus on critical paths: workflow execution, state management, error handling" and "Ensure tests verify behavior outcomes, not just function calls".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 686 - 755, Test only covers the
healthy survivor path; add a second extension fixture and make the
fakeExtensionRuntime return an unhealthy/unregistered result for it so the
partial-start recovery excludes it. Specifically, in
TestBootExtensionsKeepsHealthyRegisteredExtensionsAfterPartialStartFailure add
another installDaemonTestExtension call (e.g., "ext-bad") and update the
fakeExtensionRuntime.getExt / get behavior to return a Status with
Registered=false or Enabled=false (or nil/missing) for "ext-bad"; after
d.bootExtensions assert that the bad extension is not present (e.g.,
state.currentExtensionRuntime() != runtime for that name or check
registry/cleanup entries do not include "ext-bad"), keeping the existing
assertions for the healthy "ext-healthy".
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the partial-start recovery test only proves the happy path where the lone installed extension is healthy and registered, so it never verifies that unhealthy/unregistered extensions are excluded.
- Fix approach: add a second installed extension and make the fake runtime report it as unhealthy or unregistered so the partial-start fallback proves exclusion as well as survival.

## Resolution

- Added a second installed extension to the partial-start recovery test, taught the fake runtime to resolve extensions by name, and asserted that only the healthy extension survives as registered while the bad one remains merely enabled.
- Verified in the final `make verify` run.
