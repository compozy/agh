---
status: resolved
file: internal/api/contract/contract_test.go
line: 275
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TZZ-,comment:PRRC_kwDOR5y4QM623-TF
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Restructure these new tests to required `t.Run("Should...")` subtests (table-driven by default).**

Lines 159-275 add valid coverage, but the new tests bypass the repo’s required subtest style. Please wrap each case in `t.Run("Should...")`, and use table-driven cases for multi-scenario checks like `HasChanges()`.

<details>
<summary>Suggested refactor pattern</summary>

```diff
 func TestAutomationUpdateRequestsHasChanges(t *testing.T) {
 	t.Parallel()
-
-	name := "updated"
-	secret := "secret"
-	disabled := false
-
-	if (contract.UpdateJobRequest{}).HasChanges() {
-		t.Fatal("UpdateJobRequest{}.HasChanges() = true, want false")
-	}
-	if !(contract.UpdateJobRequest{Name: &name}).HasChanges() {
-		t.Fatal("UpdateJobRequest{Name}.HasChanges() = false, want true")
-	}
-	if !(contract.UpdateJobRequest{Enabled: &disabled}).HasChanges() {
-		t.Fatal("UpdateJobRequest{Enabled:false}.HasChanges() = false, want true")
-	}
-
-	if (contract.UpdateTriggerRequest{}).HasChanges() {
-		t.Fatal("UpdateTriggerRequest{}.HasChanges() = true, want false")
-	}
-	if !(contract.UpdateTriggerRequest{WebhookSecret: &secret}).HasChanges() {
-		t.Fatal("UpdateTriggerRequest{WebhookSecret}.HasChanges() = false, want true")
-	}
-	if !(contract.UpdateTriggerRequest{Enabled: &disabled}).HasChanges() {
-		t.Fatal("UpdateTriggerRequest{Enabled:false}.HasChanges() = false, want true")
-	}
+	name := "updated"
+	secret := "secret"
+	disabled := false
+
+	t.Run("Should report changes for UpdateJobRequest", func(t *testing.T) {
+		t.Parallel()
+		cases := []struct {
+			name string
+			req  contract.UpdateJobRequest
+			want bool
+		}{
+			{name: "Should return false for empty request", req: contract.UpdateJobRequest{}, want: false},
+			{name: "Should return true when Name is set", req: contract.UpdateJobRequest{Name: &name}, want: true},
+			{name: "Should return true when Enabled is set to false", req: contract.UpdateJobRequest{Enabled: &disabled}, want: true},
+		}
+		for _, tc := range cases {
+			tc := tc
+			t.Run(tc.name, func(t *testing.T) {
+				t.Parallel()
+				if got := tc.req.HasChanges(); got != tc.want {
+					t.Fatalf("HasChanges() = %v, want %v", got, tc.want)
+				}
+			})
+		}
+	})
 }
```
</details>

  
As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use `t.Run("Should...")` pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/contract/contract_test.go` around lines 159 - 275, The new tests
(TestAutomationJobPayloadJSONShape, TestAutomationTriggerPayloadJSONShape,
TestAutomationUpdateRequestsHasChanges) must be refactored to use t.Run subtests
with the "Should..." naming convention and table-driven style: wrap each
assertion block in a t.Run("Should ...") call (e.g., t.Run("Should marshal
next_run and scope for JobPayload") and t.Run("Should include endpoint_slug and
webhook_id for TriggerPayload")), and convert
TestAutomationUpdateRequestsHasChanges into a table-driven test where each case
contains an input (e.g., contract.UpdateJobRequest{Name: &name}) and expected
bool, looping over cases and calling t.Run("Should ...", func(t *testing.T){ ...
assert case.HasChanges() == expected }), keeping existing helpers like
marshalJSON and referencing the same types (contract.JobPayload,
contract.TriggerPayload, contract.UpdateJobRequest/UpdateTriggerRequest) and
fields to locate the code.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The reported tests in [internal/api/contract/contract_test.go](/Users/pedronauck/Dev/projects/_worktrees/automation/internal/api/contract/contract_test.go) were added as straight-line assertions and do not follow the repo's required `t.Run("Should ...")` subtest style.
  - `TestAutomationUpdateRequestsHasChanges` is also a natural table-driven case because it checks multiple request shapes with the same assertion pattern.
  - Fix approach: refactor the three automation tests into subtests, and convert the `HasChanges()` assertions into table-driven cases while preserving the current payload coverage.
  - Resolution: refactored the contract tests to `Should ...` subtests and table-driven coverage, then verified with focused `go test` runs and the final `make verify` pass.
