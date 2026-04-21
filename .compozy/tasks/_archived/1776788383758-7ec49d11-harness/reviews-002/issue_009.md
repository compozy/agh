---
status: resolved
file: internal/daemon/prompt_input_composite.go
line: 208
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUI,comment:PRRC_kwDOR5y4QM65IlPK
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Per-descriptor budgets are never enforced.**

`remainingBudget` is initialized from the sum of all descriptor budgets and then passed unchanged into each step; `descriptor.Budget` is only recorded, never used to cap that descriptor's own contribution. The first augmenter can consume the entire aggregate budget and starve later ones, which breaks the descriptor-level limits this composite is supposed to honor.

<details>
<summary>💡 Suggested change</summary>

```diff
-	bounded, consumed := applyPromptInputAugmenterBudget(
+	descriptorBudget := remainingBudget
+	if descriptor.Budget > 0 {
+		descriptorBudget = min(descriptorBudget, descriptor.Budget)
+	}
+	bounded, consumed := applyPromptInputAugmenterBudget(
 		current,
 		next,
 		limited,
-		remainingBudget,
+		descriptorBudget,
 		descriptor.BudgetBehavior,
 	)
```
</details>




Also applies to: 289-357, 424-473

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/prompt_input_composite.go` around lines 194 - 208, The code
uses a single remainingBudget for all descriptors so per-descriptor limits
(descriptor.Budget) are ignored; change the loop in
aggregatePromptInputBudget/apply flow to enforce each descriptor's cap by
computing a perDescriptorBudget = min(remainingBudget, descriptor.Budget) (or
zero if limited and descriptor.Budget==0) and pass that perDescriptorBudget into
c.applyAugmenterDescriptor instead of the full remainingBudget, then after the
call subtract the actual consumed amount from remainingBudget; update the same
pattern where applyAugmenterDescriptor is invoked (the loop using descriptors,
remainingBudget, limited) so later descriptors cannot consume more than their
own descriptor.Budget.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The composite enforced only the aggregate remaining budget. Any descriptor with a positive `Budget` could consume more than its own cap as long as earlier steps left enough aggregate budget, which defeated the per-descriptor contract. I capped each positive-budget descriptor with `min(remainingBudget, descriptor.Budget)` while preserving the existing aggregate behavior for descriptors without their own cap, and I added a scoped regression test for that case. Verified with `go test ./internal/daemon -count=1` and `make verify`.
