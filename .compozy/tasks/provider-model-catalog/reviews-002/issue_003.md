---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/acp/types_test.go
line: 96
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYapm,comment:PRRC_kwDOR5y4QM6-7HX4
---

# Issue 003: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Wrap this case in a `t.Run("Should ...")` subtest.**

Line 47 currently keeps the case flat; this file’s required test pattern uses explicit `Should ...` subtests.




<details>
<summary>Proposed update</summary>

```diff
 func TestAgentProcessCapsSnapshotClonesConfigOptions(t *testing.T) {
 	t.Parallel()
-
-	proc := &AgentProcess{}
-	proc.setCaps(Caps{
-		ConfigOptions: []SessionConfigOption{
-			{
-				ID:      "model",
-				Kind:    SessionConfigOptionKindSelect,
-				Current: "model-a",
-				Values:  []SessionConfigOptionValue{{Value: "model-a"}},
-			},
-		},
-	})
-
-	first := proc.CapsSnapshot()
-	first.ConfigOptions[0].Current = "mutated"
-	first.ConfigOptions[0].Values[0].Value = "mutated"
-
-	second := proc.CapsSnapshot()
-	if second.ConfigOptions[0].Current != "model-a" || second.ConfigOptions[0].Values[0].Value != "model-a" {
-		t.Fatalf("CapsSnapshot() leaked mutable config options: %#v", second.ConfigOptions)
-	}
-	proc.setConfigOptions([]SessionConfigOption{{ID: "reasoning_effort", Kind: SessionConfigOptionKindSelect}})
-	updated := proc.CapsSnapshot()
-	if len(updated.ConfigOptions) != 1 || updated.ConfigOptions[0].ID != "reasoning_effort" {
-		t.Fatalf("setConfigOptions() = %#v", updated.ConfigOptions)
-	}
+	t.Run("Should clone config options in CapsSnapshot", func(t *testing.T) {
+		t.Parallel()
+
+		proc := &AgentProcess{}
+		proc.setCaps(Caps{
+			ConfigOptions: []SessionConfigOption{
+				{
+					ID:      "model",
+					Kind:    SessionConfigOptionKindSelect,
+					Current: "model-a",
+					Values:  []SessionConfigOptionValue{{Value: "model-a"}},
+				},
+			},
+		})
+
+		first := proc.CapsSnapshot()
+		first.ConfigOptions[0].Current = "mutated"
+		first.ConfigOptions[0].Values[0].Value = "mutated"
+
+		second := proc.CapsSnapshot()
+		if second.ConfigOptions[0].Current != "model-a" || second.ConfigOptions[0].Values[0].Value != "model-a" {
+			t.Fatalf("CapsSnapshot() leaked mutable config options: %#v", second.ConfigOptions)
+		}
+		proc.setConfigOptions([]SessionConfigOption{{ID: "reasoning_effort", Kind: SessionConfigOptionKindSelect}})
+		updated := proc.CapsSnapshot()
+		if len(updated.ConfigOptions) != 1 || updated.ConfigOptions[0].ID != "reasoning_effort" {
+			t.Fatalf("setConfigOptions() = %#v", updated.ConfigOptions)
+		}
+	})
 }
```
</details>

As per coding guidelines `**/*_test.go`: “Use `t.Run('Should ...')` subtests with `t.Parallel` as default.”

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/acp/types_test.go` around lines 47 - 75, Wrap the existing test body
inside a t.Run subtest named using the "Should ..." pattern (e.g., t.Run("Should
clone config options without leaking mutations", func(t *testing.T) { ... })),
move the current t.Parallel() call into that subtest (call t.Parallel() as the
first line of the subtest), and keep the rest of the logic intact (creating
proc, calling proc.setCaps, first := proc.CapsSnapshot(), mutating first, second
:= proc.CapsSnapshot(), assertions, proc.setConfigOptions, updated :=
proc.CapsSnapshot(), final assertion). Ensure the top-level
TestAgentProcessCapsSnapshotClonesConfigOptions function now only calls t.Run
with the described subtest so the test follows the required "Should ..." subtest
pattern.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - `internal/acp/types_test.go` already wraps the affected behavior in `t.Run("Should clone config options snapshots", ...)` and `t.Run("Should replace config options through setConfigOptions", ...)`.
  - The flat test body described in the review is no longer present in this branch.
  - No code change is needed.
  - Resolved as invalid after branch inspection and full verification.
