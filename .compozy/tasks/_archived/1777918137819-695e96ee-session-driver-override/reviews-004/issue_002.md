---
status: resolved
file: internal/cli/cli_integration_test.go
line: 329
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R11Q,comment:PRRC_kwDOR5y4QM663fBc
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Refactor this new test into table-driven `t.Run("Should...")` subtests.**

The new integration test is valid functionally, but it doesn't follow the required test structure pattern. Please convert it to table-driven subtests (even with one case now) so future provider scenarios can be added without duplicating setup.

<details>
<summary>♻️ Proposed refactor shape</summary>

```diff
 func TestCLISessionProviderOverrideIntegration(t *testing.T) {
 	t.Parallel()
-
-	h := newIntegrationHarness(t)
-	h.runner.cfg.Providers["fake-alt"] = aghconfig.ProviderConfig{Command: "fake-alt-agent"}
-
-	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
-	defer func() {
-		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
-		_ = h.runner.waitForExit()
-	}()
-
-	// ... single monolithic flow ...
+	tests := []struct {
+		name     string
+		provider string
+	}{
+		{name: "ShouldPersistProviderOverrideAcrossLifecycle", provider: "fake-alt"},
+	}
+
+	for _, tc := range tests {
+		tc := tc
+		t.Run(tc.name, func(t *testing.T) {
+			t.Parallel()
+
+			h := newIntegrationHarness(t)
+			h.runner.cfg.Providers[tc.provider] = aghconfig.ProviderConfig{Command: "fake-alt-agent"}
+
+			mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
+			defer func() {
+				_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
+				_ = h.runner.waitForExit()
+			}()
+
+			// existing assertions using tc.provider
+		})
+	}
 }
```
</details>


As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases` and `Use table-driven tests with subtests (t.Run) as default pattern for Go tests`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestCLISessionProviderOverrideIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
	}{
		{name: "ShouldPersistProviderOverrideAcrossLifecycle", provider: "fake-alt"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newIntegrationHarness(t)
			h.runner.cfg.Providers[tc.provider] = aghconfig.ProviderConfig{Command: "fake-alt-agent"}

			mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
			defer func() {
				_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
				_ = h.runner.waitForExit()
			}()

			newOut, _, err := executeRootCommand(
				t,
				h.deps,
				"session",
				"new",
				"--agent",
				"coder",
				"--name",
				"provider-demo",
				"--provider",
				tc.provider,
				"--cwd",
				h.workspace,
				"-o",
				"json",
			)
			if err != nil {
				t.Fatalf("session new --provider error = %v", err)
			}

			var created SessionRecord
			if err := json.Unmarshal([]byte(newOut), &created); err != nil {
				t.Fatalf("json.Unmarshal(session new --provider) error = %v", err)
			}
			if created.Provider != tc.provider {
				t.Fatalf("created.Provider = %q, want %q", created.Provider, tc.provider)
			}

			statusOut, _, err := executeRootCommand(t, h.deps, "session", "status", created.ID, "-o", "json")
			if err != nil {
				t.Fatalf("session status error = %v", err)
			}

			var status SessionRecord
			if err := json.Unmarshal([]byte(statusOut), &status); err != nil {
				t.Fatalf("json.Unmarshal(session status) error = %v", err)
			}
			if status.Provider != tc.provider || status.State != session.StateActive {
				t.Fatalf("status = %#v, want active %s session", status, tc.provider)
			}

			listOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--all", "-o", "json")
			if err != nil {
				t.Fatalf("session list error = %v", err)
			}

			var listed []SessionRecord
			if err := json.Unmarshal([]byte(listOut), &listed); err != nil {
				t.Fatalf("json.Unmarshal(session list) error = %v", err)
			}
			if got, want := len(listed), 1; got != want {
				t.Fatalf("len(listed) = %d, want %d", got, want)
			}
			if listed[0].Provider != tc.provider {
				t.Fatalf("listed[0].Provider = %q, want %q", listed[0].Provider, tc.provider)
			}

			stopOut, _, err := executeRootCommand(t, h.deps, "session", "stop", created.ID, "-o", "json")
			if err != nil {
				t.Fatalf("session stop error = %v", err)
			}

			var stopped SessionRecord
			if err := json.Unmarshal([]byte(stopOut), &stopped); err != nil {
				t.Fatalf("json.Unmarshal(session stop) error = %v", err)
			}
			if stopped.Provider != tc.provider || stopped.State != session.StateStopped {
				t.Fatalf("stopped = %#v, want stopped %s session", stopped, tc.provider)
			}

			resumeOut, _, err := executeRootCommand(t, h.deps, "session", "resume", created.ID, "-o", "json")
			if err != nil {
				t.Fatalf("session resume error = %v", err)
			}

			var resumed SessionRecord
			if err := json.Unmarshal([]byte(resumeOut), &resumed); err != nil {
				t.Fatalf("json.Unmarshal(session resume) error = %v", err)
			}
			if resumed.Provider != tc.provider || resumed.State != session.StateActive {
				t.Fatalf("resumed = %#v, want active %s session", resumed, tc.provider)
			}
		})
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/cli_integration_test.go` around lines 235 - 329, Refactor
TestCLISessionProviderOverrideIntegration into a table-driven test that uses
t.Run subtests: create a slice of test cases (even a single case for now) with
name and any case-specific fields, then loop over them and call t.Run(tc.name,
func(t *testing.T) { ... }) to execute the existing setup and assertions for
each case; keep the existing setup (newIntegrationHarness,
h.runner.cfg.Providers modification, daemon start/stop, and all
executeRootCommand/assertions) inside the subtest body, and ensure each subtest
uses t.Parallel() where appropriate and references the same identifiers
(TestCLISessionProviderOverrideIntegration,
created/status/listed/stopped/resumed variables, and executeRootCommand) so
future provider cases can be added by adding entries to the test cases slice.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
- The current test is a single end-to-end CLI lifecycle scenario with one setup path and one assertion flow; wrapping it in a one-row table would only add indirection.
- No correctness gap or regression hole was found in the current structure, so this is a stylistic preference rather than a technical defect for this batch.
