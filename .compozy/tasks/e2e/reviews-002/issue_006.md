---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 79
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57y10W,comment:PRRC_kwDOR5y4QM644c8T
---

# Issue 006: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify top-level tests and existing Should-style subtests in this file.
rg -nP '^func Test\w+\(t \*testing\.T\)' internal/daemon/tool_mcp_resources_test.go
rg -nP 't\.Run\("Should[^"]*"' internal/daemon/tool_mcp_resources_test.go
```

Repository: compozy/agh

Length of output: 1034

---



</details>

**Refactor test to use `t.Run("Should...")` pattern as required by guidelines.**

This test violates the mandatory requirement: `**/*_test.go` tests MUST use `t.Run("Should...")` pattern for ALL test cases. Wrap the test logic in a subtest and normalize the structure for consistency with other tests in the file.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/tool_mcp_resources_test.go` around lines 20 - 79,
TestResourceCatalogProjectorBuildAndApply must be refactored to use a
t.Run("Should...") subtest; wrap the existing test logic inside a subtest named
like "Should build and apply resource catalog projector" and move t.Parallel()
into that subtest (or call it both places if needed), keeping all assertions and
usage of newResourceCatalog, newToolProjector, projector.Build, projector.Apply,
catalog.Snapshot, etc. unchanged except for indentation and scope so the test
body executes inside the t.Run callback and the top-level
TestResourceCatalogProjectorBuildAndApply only defines the subtest.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: `TestResourceCatalogProjectorBuildAndApply` is a single cohesive unit test, not a table-driven suite. Wrapping the whole body in one extra `Should...` subtest would not change behavior or improve failure isolation.
- Repository fit: the workspace requires subtests as the default shape for table-driven tests, but it does not require every standalone test to be wrapped in a single subtest shell.
- Resolution: no code change required.
