---
status: resolved
file: internal/cli/task.go
line: 1369
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsp,comment:PRRC_kwDOR5y4QM67YHDF
---

# Issue 030: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Drop blank `--capability` values after trimming.**

Whitespace-only flags currently survive as `""`, so `agh task next --capability "   "` still sends an empty capability to the daemon instead of normalizing it away locally. That turns a CLI input mistake into a backend validation problem.



<details>
<summary>Suggested fix</summary>

```diff
 func trimAgentTaskCapabilities(values []string) []string {
 	if len(values) == 0 {
 		return nil
 	}
 	trimmed := make([]string, 0, len(values))
 	for _, value := range values {
-		trimmed = append(trimmed, strings.TrimSpace(value))
+		value = strings.TrimSpace(value)
+		if value == "" {
+			continue
+		}
+		trimmed = append(trimmed, value)
 	}
+	if len(trimmed) == 0 {
+		return nil
+	}
 	return trimmed
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/task.go` around lines 1361 - 1369, The trimAgentTaskCapabilities
function currently preserves whitespace-only entries as empty strings; change it
so after calling strings.TrimSpace on each value you skip any result that is
empty (i.e., only append non-empty trimmed strings), ensuring flags like
--capability "   " are dropped locally; update the loop in
trimAgentTaskCapabilities to perform TrimSpace into a local variable and only
append when that variable != "" so the returned slice contains no blank
capabilities.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `trimAgentTaskCapabilities` trims values but keeps whitespace-only inputs as empty strings. The CLI should normalize blank capability flags away instead of sending them to backend validation.
- Fix: Skip empty values after trimming and return `nil` when all provided values are blank.
