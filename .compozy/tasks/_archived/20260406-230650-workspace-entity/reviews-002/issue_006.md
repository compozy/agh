---
status: resolved
file: internal/cli/install_test.go
line: 182
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoB4,comment:PRRC_kwDOR5y4QM61T6HJ
---

# Issue 006: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Inconsistent return value handling from `Update()` calls.**

Several `Update()` calls discard both return values (lines 160, 166, 181, 182), while others in the same test properly capture and verify the `cmd` return (lines 139, 146, 174). This inconsistency may mask unexpected behavior and makes the test harder to reason about.



<details>
<summary>♻️ Suggested fix for consistent return handling</summary>

```diff
 	model.modelInput.SetValue("")
-	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
+	if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
+		t.Fatalf("empty model enter cmd = %v, want nil (validation error)", cmd)
+	}
 	if model.errText != "model is required" {
 		t.Fatalf("errText = %q, want model is required", model.errText)
 	}

 	model.modelInput.SetValue("gpt-5.4")
-	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
+	if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
+		t.Fatal("model enter cmd = nil, want blink command")
+	}
 	if model.step != installWizardStepConfirm {
 		t.Fatalf("step = %v, want confirm", model.step)
 	}
```

```diff
-	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
-	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
+	if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
+		t.Fatal("confirm step enter cmd = nil, want blink command")
+	}
+	if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
+		t.Fatal("final confirm enter cmd = nil, want quit command")
+	}
 	if !model.done {
 		t.Fatal("done = false, want true after confirm enter")
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/install_test.go` around lines 159 - 182, The test inconsistently
ignores the two return values from model.Update(...) which can hide unexpected
commands; update each call to model.Update(tea.KeyMsg{...}) (including the calls
after model.modelInput.SetValue("") and SetValue("gpt-5.4"), and the two final
Enter calls) to capture both returned model and cmd (e.g., newModel, cmd :=
model.Update(...)) and assert cmd is non-nil or validate its expected
type/behavior consistent with earlier checks so all Update() return values are
handled uniformly in the test.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - The suggested expectations do not match the implementation.
  - `updateModelStep("enter")` intentionally returns `nil` both for validation failure and for the successful transition to the confirm step, while only the final confirm enter returns `tea.Quit`.
  - Capturing every ignored return value here would not expose a bug, and asserting non-nil commands on those branches would make the test incorrect.
