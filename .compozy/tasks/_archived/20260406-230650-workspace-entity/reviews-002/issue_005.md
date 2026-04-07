---
status: resolved
file: internal/cli/install_test.go
line: 118
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoB2,comment:PRRC_kwDOR5y4QM61T6HH
---

# Issue 005: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Consider splitting into subtests for better test organization.**

This test covers two distinct concerns: input building and bundle formatting. Using subtests would improve test isolation and make failures easier to diagnose.



<details>
<summary>♻️ Suggested refactor with subtests</summary>

```diff
 func TestBuildInstallWizardInputAndBundleFormats(t *testing.T) {
 	t.Parallel()
 
-	cfg, err := aghconfig.Default()
-	if err != nil {
-		t.Fatalf("aghconfig.Default() error = %v", err)
-	}
-	cfg.Defaults.Provider = "codex"
-	cfg.Providers["custom"] = aghconfig.ProviderConfig{DefaultModel: "custom-model"}
-
-	input := buildInstallWizardInput(cfg)
-	if len(input.Providers) == 0 {
-		t.Fatal("buildInstallWizardInput() providers = empty, want builtin/custom providers")
-	}
-	if input.SelectedProvider != "codex" {
-		t.Fatalf("SelectedProvider = %q, want %q", input.SelectedProvider, "codex")
-	}
-	if input.SuggestedModels["custom"] != "custom-model" {
-		t.Fatalf("SuggestedModels[custom] = %q, want %q", input.SuggestedModels["custom"], "custom-model")
-	}
+	t.Run("buildInstallWizardInput", func(t *testing.T) {
+		t.Parallel()
+		cfg, err := aghconfig.Default()
+		if err != nil {
+			t.Fatalf("aghconfig.Default() error = %v", err)
+		}
+		cfg.Defaults.Provider = "codex"
+		cfg.Providers["custom"] = aghconfig.ProviderConfig{DefaultModel: "custom-model"}
+
+		input := buildInstallWizardInput(cfg)
+		if len(input.Providers) == 0 {
+			t.Fatal("buildInstallWizardInput() providers = empty, want builtin/custom providers")
+		}
+		if input.SelectedProvider != "codex" {
+			t.Fatalf("SelectedProvider = %q, want %q", input.SelectedProvider, "codex")
+		}
+		if input.SuggestedModels["custom"] != "custom-model" {
+			t.Fatalf("SuggestedModels[custom] = %q, want %q", input.SuggestedModels["custom"], "custom-model")
+		}
+	})
 
-	record := installRecord{
+	t.Run("installBundle formats", func(t *testing.T) {
+		t.Parallel()
+		record := installRecord{
 			AgentName:    aghconfig.DefaultAgentName,
 			// ... rest of record
-	}
-	// ... rest of bundle tests
+		}
+		// ... rest of bundle tests
+	})
 }
```
</details>

As per coding guidelines: "Table-driven tests with subtests (`t.Run`) as default in Go tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/install_test.go` around lines 72 - 118, Split
TestBuildInstallWizardInputAndBundleFormats into two subtests using t.Run: one
subtest named "BuildInstallWizardInput" that calls buildInstallWizardInput(cfg)
and asserts Providers, SelectedProvider and SuggestedModels; and a second
subtest named "InstallBundleFormats" that constructs the installRecord and calls
installBundle(record).human() and .toon() with the existing assertions. Move
t.Parallel into each subtest (or keep at top and call t.Parallel inside each
t.Run) so they run in parallel safely, and preserve the same setup
(aghconfig.Default, cfg modifications) shared between the subtests but ensure
each t.Run uses that prepared cfg/record so failures are isolated and easier to
diagnose; keep references to TestBuildInstallWizardInputAndBundleFormats,
buildInstallWizardInput, installBundle, human(), and toon().
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - This is a style-only refactor suggestion.
  - The existing test is short, cohesive, and already fails with precise assertions for both helper outputs.
  - Splitting it into subtests would add churn without fixing a defect or closing a coverage gap.
