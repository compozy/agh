---
status: resolved
file: internal/cli/docpost/docpost.go
line: 405
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hC_3,comment:PRRC_kwDOR5y4QM64gE4q
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Keep blank lines inside indented code examples.**

`fenceIndentedBlocks()` closes the fence on the first empty line. Markdown indented code blocks can legally contain blank lines, so multi-command examples get split into separate fenced blocks during conversion.



<details>
<summary>Proposed fix</summary>

```diff
 		switch {
 		case !inIndent && isIndented:
 			inIndent = true
 			result = append(result, "```", stripIndent(line))
 		case inIndent && isIndented:
 			result = append(result, stripIndent(line))
 		case inIndent && isEmpty:
-			inIndent = false
-			result = append(result, "```", line)
+			result = append(result, line)
 		default:
 			if inIndent {
 				inIndent = false
 				result = append(result, "```")
 			}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func fenceIndentedBlocks(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	inFence := false
	inIndent := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track existing fenced code blocks.
		if strings.HasPrefix(trimmed, "
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

````
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/docpost/docpost.go` around lines 354 - 405, The function
fenceIndentedBlocks incorrectly closes an opened fenced block when encountering
an empty line inside an indented code example, which drops blank lines from code
samples; in fenceIndentedBlocks, keep inIndent true for empty indented-code
lines and do not emit the closing "```" on the first empty line—specifically, in
the branch handling "case inIndent && isEmpty" remove the lines that set
inIndent = false and append the closing fence and instead append the original
line (preserving the blank line) so that only a non-indented subsequent line or
end-of-input closes the fence.
````

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `fenceIndentedBlocks()` currently closes an indented code fence on the first blank line, which splits one logical example into multiple fenced blocks and drops the original blank-line structure.
  - Root cause: the `case inIndent && isEmpty` branch closes the fence instead of preserving the blank line while the indented block is still active.
  - Fix plan: keep the fence open for blank lines and only close it when a non-indented content line or EOF ends the block.
  - Resolution: `fenceIndentedBlocks()` now preserves blank lines while an indented block remains active and only closes the fence on the next non-indented content line or EOF.
  - Verification: `go test ./internal/cli/...` passed.
