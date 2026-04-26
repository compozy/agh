---
status: resolved
file: internal/network/greet_summary.go
line: 55
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQq,comment:PRRC_kwDOR5y4QM67VX7H
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**`+N more` can be overstated when capability entries contain blanks.**

The current count uses raw slice length after the first non-empty capability, so empty strings still increase `N`.


<details>
<summary>Suggested fix</summary>

```diff
 	for idx, capability := range card.Capabilities {
 		if trimmed := strings.TrimSpace(capability); trimmed != "" {
-			return trimmed, len(card.Capabilities) - idx - 1
+			extra := 0
+			for _, candidate := range card.Capabilities[idx+1:] {
+				if strings.TrimSpace(candidate) != "" {
+					extra++
+				}
+			}
+			return trimmed, extra
 		}
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	for idx, capability := range card.Capabilities {
		if trimmed := strings.TrimSpace(capability); trimmed != "" {
			extra := 0
			for _, candidate := range card.Capabilities[idx+1:] {
				if strings.TrimSpace(candidate) != "" {
					extra++
				}
			}
			return trimmed, extra
		}
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/greet_summary.go` around lines 52 - 55, The code in the loop
over card.Capabilities returns the first non-empty trimmed capability but
computes the remaining count using the raw slice length, which overstates N when
there are blank entries; update the return to compute remaining as the number of
non-empty (after TrimSpace) entries after the found index by iterating the tail
of card.Capabilities (or using a helper that filters empties) and counting only
trimmed != "" so the returned tuple gives the correct first non-empty capability
and the correct remaining non-empty count.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `greetCapabilityLabel()` returns the first non-empty capability label but computes the remaining count from the raw slice length, so blank capability slots still inflate the `+N more` suffix.
  - Root cause: the fallback path counts tail entries structurally instead of counting only trimmed non-empty capability labels.
  - Fix plan: count only non-empty trailing capabilities after the first selected label and extend greet summary coverage to include blank capability entries.
