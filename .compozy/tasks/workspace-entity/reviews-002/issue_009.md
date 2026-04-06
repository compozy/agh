---
status: resolved
file: internal/config/agent_test.go
line: 185
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoB_,comment:PRRC_kwDOR5y4QM61T6HR
---

# Issue 009: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Add `t.Parallel()` for test independence.**

Per coding guidelines, independent tests should use `t.Parallel()` to enable parallel execution.


<details>
<summary>♻️ Proposed fix</summary>

```diff
 func TestWorkspaceDiscoveryRootsReturnsWorkspaceAdditionalGlobalOrder(t *testing.T) {
+	t.Parallel()
+
 	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/agent_test.go` around lines 146 - 185, Add t.Parallel() to
the test TestWorkspaceDiscoveryRootsReturnsWorkspaceAdditionalGlobalOrder to
allow it to run concurrently with other independent tests; call t.Parallel()
immediately at the start of the test (right after the function begins) so the
setup using ResolveHomePathsFrom, TempDir(), and WorkspaceDiscoveryRoots still
runs under the parallel test context.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - This test uses only `t.TempDir()`-scoped filesystem state and local values.
  - There is no shared process-global mutation, so running it in parallel is safe and aligned with the project’s test conventions.
  - I will add `t.Parallel()` at the start of the test.
