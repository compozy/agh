---
status: resolved
file: internal/automation/model/template.go
line: 85
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TkG_,comment:PRRC_kwDOR5y4QM624LnG
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In Go text/template parsing, inside `{{with .Data}} ... {{end}}`, is `.foo` represented as a field lookup relative to the current dot, requiring validators to prepend the current dot scope?`

💡 Result:

No, inside {{with .Data}} ... {{end}}, .foo refers to the foo field of .Data (the current dot), which is a relative field lookup from the current scope. Validators or parsers do not need to prepend the current dot scope; field lookups like .foo are always relative to whatever the dot currently points to during execution.

Citations:

- 1: https://pkg.go.dev/text/template@go1.19
- 2: https://pkg.go.dev/text/template

---

🏁 Script executed:

```shell
#!/bin/bash
# Inspect the template validation code
wc -l internal/automation/model/template.go
```

Repository: compozy/agh

Length of output: 96

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at the file to understand the validation logic
cat -n internal/automation/model/template.go | head -100
```

Repository: compozy/agh

Length of output: 3387

---

🏁 Script executed:

```shell
#!/bin/bash
# Examine the validateTemplateArgWithState function and surrounding context
sed -n '160,190p' internal/automation/model/template.go | cat -n
```

Repository: compozy/agh

Length of output: 1238

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at templateValidationState and related functions
sed -n '1,90p' internal/automation/model/template.go | cat -n
```

Repository: compozy/agh

Length of output: 3003

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at the rest of the file, especially validateActivationFieldPath and withTemplateValidationState
sed -n '100,160p' internal/automation/model/template.go | cat -n
```

Repository: compozy/agh

Length of output: 2117

---

🏁 Script executed:

```shell
#!/bin/bash
# See the end of the file and helper functions
tail -n 150 internal/automation/model/template.go | cat -n
```

Repository: compozy/agh

Length of output: 4895

---

🏁 Script executed:

```shell
#!/bin/bash
# Verify the full file structure and check for any other validation calls
wc -l internal/automation/model/template.go
sed -n '26,27p' internal/automation/model/template.go
sed -n '35,38p' internal/automation/model/template.go
```

Repository: compozy/agh

Length of output: 298

---

🏁 Script executed:

```shell
#!/bin/bash
# Check test coverage or examples to understand expected behavior
fd -e "template" -e "Template" internal/automation/model/ | head -20
```

Repository: compozy/agh

Length of output: 464

---

🏁 Script executed:

```shell
#!/bin/bash
# Look for tests to understand expected behavior
fd -type f -name "*test*" internal/automation/model/
```

Repository: compozy/agh

Length of output: 225

---

🏁 Script executed:

```shell
#!/bin/bash
# Check how validateActivationFieldPath handles "Data" and other paths
sed -n '127,143p' internal/automation/model/template.go
```

Repository: compozy/agh

Length of output: 392

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at lines 26-27 and 35-37 to check error handling
sed -n '22,32p' internal/automation/model/template.go | cat -n
sed -n '34,39p' internal/automation/model/template.go | cat -n
```

Repository: compozy/agh

Length of output: 645

---



</details>

**Use scoped-dot state for field/chain validation consistent with index validation.**

Line 171 and line 180 validate field and chain lookups using unscoped paths, causing inconsistency with `validateIndexArgs` at line 56 which correctly uses `scopedTemplateFieldPath`. Inside `{{with .Data}}`, a lookup like `.foo` should resolve relative to the current dot (accessing `.Data.foo` at runtime) but is incorrectly rejected as "unknown activation envelope field 'foo'".

Additionally, error returns at lines 26–27 and 36 propagate validation errors without wrapping context per coding guidelines.

<details>
<summary>🔧 Suggested fix</summary>

```diff
 func validateTemplateArgWithState(node parse.Node, state templateValidationState) error {
 	switch n := node.(type) {
 	case nil:
 		return nil
 	case *parse.FieldNode:
-		return validateActivationFieldPath(n.Ident)
+		path, ok := scopedTemplateFieldPath(n, state)
+		if !ok {
+			return fmt.Errorf("unsupported activation lookup %q; unresolved template scope", n.String())
+		}
+		return validateActivationFieldPath(path)
 	case *parse.VariableNode:
 		if len(n.Ident) > 1 {
 			return fmt.Errorf("unsupported activation lookup %q; variable-rooted lookups are not supported", n.String())
 		}
 		return nil
 	case *parse.ChainNode:
 		if _, ok := variableRootExpression(n.Node); ok {
 			return fmt.Errorf("unsupported activation lookup %q; variable-rooted lookups are not supported", n.String())
 		}
-		path, ok := templateFieldPath(n)
+		path, ok := scopedTemplateFieldPath(n, state)
 		if !ok {
-			return nil
+			return fmt.Errorf("unsupported activation lookup %q; unresolved template scope", n.String())
 		}
 		return validateActivationFieldPath(path)
```

For error propagation, wrap validation errors at call sites (lines 26, 36):
```diff
 		if err := validateTemplateNode(subtemplate.Root); err != nil {
-			return nil, err
+			return nil, fmt.Errorf("validate trigger prompt template: %w", err)
 		}
```

```diff
 func ValidateTriggerPromptTemplate(prompt string) error {
-	_, err := ParseTriggerPromptTemplate(prompt)
-	return err
+	_, err := ParseTriggerPromptTemplate(prompt)
+	if err != nil {
+		return fmt.Errorf("validate trigger prompt template: %w", err)
+	}
+	return nil
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/model/template.go` at line 85, The field/chain validation
in validateTemplateNodeWithState is using unscoped paths; change those checks to
use scopedTemplateFieldPath (the same helper used in validateIndexArgs) so
lookups inside a {{with .Data}} are resolved relative to the current dot (e.g.,
`.foo` -> `.Data.foo`). Also update the two call sites that currently propagate
validation errors directly (the returns that do `return err`) to wrap the error
with context using fmt.Errorf("validate template: %w", err) (or equivalent) so
validation failures include caller context instead of raw errors.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `validateTemplateArgWithState()` currently validates `*parse.FieldNode` and `*parse.ChainNode` using unscoped paths, unlike `validateIndexArgs()` which already resolves lookups through `scopedTemplateFieldPath(...)`.
- That means dot-rebound expressions inside `{{ with .Data }}` such as `{{ .session_id }}` are incorrectly validated as unknown top-level activation fields instead of relative lookups under `.Data`.
- Fix plan: resolve field and chain lookups through the scoped-dot helper for consistent validation and add a regression that accepts dot-relative access inside `with .Data`.
