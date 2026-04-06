---
status: resolved
file: internal/config/agent_test.go
line: 221
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCA,comment:PRRC_kwDOR5y4QM61T6HT
---

# Issue 010: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Add `t.Parallel()` for test independence.**


<details>
<summary>♻️ Proposed fix</summary>

```diff
 func TestLoadWorkspaceAgentDefsAppliesDocumentedPrecedence(t *testing.T) {
+	t.Parallel()
+
 	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestLoadWorkspaceAgentDefsAppliesDocumentedPrecedence(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	root := t.TempDir()
	additionalOne := t.TempDir()
	additionalTwo := t.TempDir()

	writeAgentDefinition(t, filepath.Join(homePaths.AgentsDir, "coder", agentDefName), "coder", "claude", "global")
	writeAgentDefinition(t, filepath.Join(homePaths.AgentsDir, "reviewer", agentDefName), "reviewer", "claude", "global-review")
	writeAgentDefinition(t, filepath.Join(additionalOne, DirName, AgentsDirName, "coder", agentDefName), "coder", "claude", "additional")
	writeAgentDefinition(t, filepath.Join(additionalOne, DirName, AgentsDirName, "pairer", agentDefName), "pairer", "claude", "additional-pair")
	writeAgentDefinition(t, filepath.Join(additionalTwo, DirName, AgentsDirName, "reviewer", agentDefName), "reviewer", "claude", "additional-review")
	writeAgentDefinition(t, filepath.Join(root, DirName, AgentsDirName, "coder", agentDefName), "coder", "claude", "workspace")

	agents, err := LoadWorkspaceAgentDefs(root, []string{additionalOne, additionalTwo}, homePaths)
	if err != nil {
		t.Fatalf("LoadWorkspaceAgentDefs() error = %v", err)
	}

	if got, want := agentModel(agents, "coder"), "workspace"; got != want {
		t.Fatalf("coder model = %q, want %q", got, want)
	}
	if got, want := agentModel(agents, "pairer"), "additional-pair"; got != want {
		t.Fatalf("pairer model = %q, want %q", got, want)
	}
	if got, want := agentModel(agents, "reviewer"), "additional-review"; got != want {
		t.Fatalf("reviewer model = %q, want %q", got, want)
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/agent_test.go` around lines 187 - 221, Add t.Parallel() at
the start of TestLoadWorkspaceAgentDefsAppliesDocumentedPrecedence to allow the
test to run concurrently and avoid interference with other tests; update the
TestLoadWorkspaceAgentDefsAppliesDocumentedPrecedence function (the test that
calls ResolveHomePathsFrom, EnsureHomeLayout, creates temp dirs and calls
LoadWorkspaceAgentDefs) by inserting a t.Parallel() call immediately after the
function begins (before setup) so the test is run in parallel safely.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - This precedence test is independent and uses isolated temp directories plus a test-local home layout.
  - There is no process-wide state mutation, so `t.Parallel()` is safe here as well.
  - I will add `t.Parallel()` at the start of the test.
