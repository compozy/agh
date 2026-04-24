---
status: resolved
file: internal/extension/teams_provider_integration_test.go
line: 73
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1b,comment:PRRC_kwDOR5y4QM67HMWm
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify whether this integration test file is using "Should..." subtests for test cases.
rg -n --type go 'func Test' internal/extension/teams_provider_integration_test.go
rg -n --type go '\bt\.Run\("Should' internal/extension/teams_provider_integration_test.go
```

Repository: compozy/agh

Length of output: 280

---

🏁 Script executed:

```shell
#!/bin/bash
# Check for integration test build tag
head -5 internal/extension/teams_provider_integration_test.go

# Get line count to understand file size
wc -l internal/extension/teams_provider_integration_test.go

# Show the context around lines 65-73 with the full test function
sed -n '40,75p' internal/extension/teams_provider_integration_test.go
```

Repository: compozy/agh

Length of output: 1795

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for ALL t.Run usages (not just "Should" pattern)
rg -n 't\.Run\(' internal/extension/teams_provider_integration_test.go | head -20

# Get context of the second test function to compare structure
sed -n '113,180p' internal/extension/teams_provider_integration_test.go
```

Repository: compozy/agh

Length of output: 3209

---



</details>

**Wrap per-instance assertions in `t.Run()` subtests for failure isolation.**

Lines 65-73 validate each instance sequentially in a plain loop. Per test guidelines, this should use subtests to isolate failures and improve test output clarity.

<details>
<summary>♻️ Suggested refactor</summary>

```diff
-	for _, instanceID := range expectedInstanceIDs {
-		state, ok := teamsProviderLastStateForInstance(states, instanceID)
-		if !ok {
-			t.Fatalf("adapter state for %q missing after wait: %#v", instanceID, states)
-		}
-		if got, want := state.Status.Normalize(), bridgepkg.BridgeStatusReady; got != want {
-			t.Fatalf("adapter state for %q = %q (error=%q), want %q", instanceID, got, state.Error, want)
-		}
-	}
+	for _, instanceID := range expectedInstanceIDs {
+		instanceID := instanceID
+		t.Run("ShouldReportReadyStateFor_"+instanceID, func(t *testing.T) {
+			state, ok := teamsProviderLastStateForInstance(states, instanceID)
+			if !ok {
+				t.Fatalf("adapter state for %q missing after wait: %#v", instanceID, states)
+			}
+			if got, want := state.Status.Normalize(), bridgepkg.BridgeStatusReady; got != want {
+				t.Fatalf("adapter state for %q = %q (error=%q), want %q", instanceID, got, state.Error, want)
+			}
+		})
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	for _, instanceID := range expectedInstanceIDs {
		instanceID := instanceID
		t.Run("ShouldReportReadyStateFor_"+instanceID, func(t *testing.T) {
			state, ok := teamsProviderLastStateForInstance(states, instanceID)
			if !ok {
				t.Fatalf("adapter state for %q missing after wait: %#v", instanceID, states)
			}
			if got, want := state.Status.Normalize(), bridgepkg.BridgeStatusReady; got != want {
				t.Fatalf("adapter state for %q = %q (error=%q), want %q", instanceID, got, state.Error, want)
			}
		})
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/teams_provider_integration_test.go` around lines 65 - 73,
Wrap each per-instance assertion into a t.Run subtest to isolate failures:
iterate expectedInstanceIDs and for each call t.Run(instanceID, func(t
*testing.T) { ... }) and move the existing checks (calling
teamsProviderLastStateForInstance(states, instanceID), verifying ok, and
comparing state.Status.Normalize() to bridgepkg.BridgeStatusReady and
state.Error) inside that subtest; ensure you use the loop variable correctly
(capture instanceID) so t.Fatalf remains inside the subtest to report only that
instance's failure.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `TestTeamsProviderLaunchNegotiatesBridgeRuntime` loops over expected instance IDs and fails from the parent test body, which loses per-instance failure isolation.
  - The fix is to wrap each instance assertion in a named `Should...` subtest while preserving the existing readiness wait and conformance checks.
