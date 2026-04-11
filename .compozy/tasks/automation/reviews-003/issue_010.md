---
status: resolved
file: internal/automation/model/template.go
line: 27
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TkG9,comment:PRRC_kwDOR5y4QM624LnE
---

# Issue 010: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap propagated errors with local context at API boundaries.**

These returns propagate raw errors, which loses where validation failed from the caller’s perspective. Please wrap at these boundary points.



<details>
<summary>🔧 Suggested fix</summary>

```diff
 		if err := validateTemplateNode(subtemplate.Root); err != nil {
-			return nil, err
+			return nil, fmt.Errorf("validate trigger prompt template %q: %w", subtemplate.Name(), err)
 		}
 	}
@@
 func ValidateTriggerPromptTemplate(prompt string) error {
 	_, err := ParseTriggerPromptTemplate(prompt)
-	return err
+	if err != nil {
+		return fmt.Errorf("validate trigger prompt template: %w", err)
+	}
+	return nil
 }
```
</details>

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".


Also applies to: 36-37

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/model/template.go` around lines 26 - 27, The calls that
return raw errors from validateTemplateNode (e.g., the check using
validateTemplateNode(subtemplate.Root)) should wrap the propagated error with
local context before returning so callers know where validation failed; update
the return paths to use fmt.Errorf with descriptive context (for example:
"validate template root: %w") and apply the same wrapping pattern to the other
similar return(s) around the template validation checks (referencing
validateTemplateNode and subtemplate.Root).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `ParseTriggerPromptTemplate()` returns raw validation failures from `validateTemplateNode(...)`, and `ValidateTriggerPromptTemplate()` passes through raw parser/validator errors unchanged.
- At those API boundaries the caller loses local context about whether the failure came from parsing or from activation-envelope validation.
- Fix plan: wrap the propagated validation errors with trigger-template-specific context and update template tests to assert the contextualized messages.
