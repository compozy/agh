---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/config_test.go
line: 409
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulKS,comment:PRRC_kwDOR5y4QM680KI0
---

# Issue 019: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use a specific error assertion for the invalid JSON case.**

Line 407 currently checks only non-nil error. Assert error details (message/class) so the test proves the expected failure mode instead of any generic error.



<details>
<summary>Suggested assertion improvement</summary>

```diff
 	if _, err := parseStringSliceValue(`["ok",1]`); err == nil {
 		t.Fatal("parseStringSliceValue(invalid json) error = nil, want error")
+	} else if !strings.Contains(err.Error(), "string") {
+		t.Fatalf("parseStringSliceValue(invalid json) error = %v, want element type detail", err)
 	}
```
</details>

As per coding guidelines, "`**/*_test.go`: MUST have specific error assertions (ErrorContains, ErrorAs)".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if _, err := parseStringSliceValue(`["ok",1]`); err == nil {
		t.Fatal("parseStringSliceValue(invalid json) error = nil, want error")
	} else if !strings.Contains(err.Error(), "string") {
		t.Fatalf("parseStringSliceValue(invalid json) error = %v, want element type detail", err)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/config_test.go` around lines 407 - 409, The test currently only
checks that parseStringSliceValue(`["ok",1]`) returns a non-nil error; change it
to assert the specific error class or message so the failure mode is guaranteed.
Update the assertion for parseStringSliceValue to use a specific test helper
(e.g., ErrorContains or ErrorAs) to verify the error is the expected JSON/type
error (match the substring or error type produced by parseStringSliceValue)
instead of a generic non-nil check; keep the call and input the same and replace
t.Fatal(...) with the targeted assertion referencing parseStringSliceValue.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The invalid JSON element case in `TestConfigRenderingAndMutationHelpers` only checks for a non-nil error. That does not prove the expected type-validation failure. Assert the error message includes the element type detail produced by `parseStringSliceValue`.
