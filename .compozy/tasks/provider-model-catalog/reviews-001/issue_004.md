---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/api/core/conversions_parsers_test.go
line: 151
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6sG,comment:PRRC_kwDOR5y4QM6-6brd
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Tighten config-option value assertions.**

Current checks can still pass if allowed values are wrong but length remains `2`. Please assert actual values (e.g., `low`, `high`) to harden the contract.
 
<details>
<summary>🔧 Suggested test hardening</summary>

```diff
 	if got := payload.ACPCaps.ConfigOptions[0]; got.ID != "reasoning_effort" || got.Current != "high" ||
 		got.Kind != "select" || len(got.Values) != 2 {
 		t.Fatalf("config option payload = %#v", got)
 	}
+	if got := payload.ACPCaps.ConfigOptions[0].Values; got[0].Value != "low" || got[1].Value != "high" {
+		t.Fatalf("config option values = %#v, want [low high]", got)
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if len(payload.ACPCaps.ConfigOptions) != 1 {
		t.Fatalf("config options = %#v", payload.ACPCaps.ConfigOptions)
	}
	if got := payload.ACPCaps.ConfigOptions[0]; got.ID != "reasoning_effort" || got.Current != "high" ||
		got.Kind != "select" || len(got.Values) != 2 {
		t.Fatalf("config option payload = %#v", got)
	}
	if got := payload.ACPCaps.ConfigOptions[0].Values; got[0].Value != "low" || got[1].Value != "high" {
		t.Fatalf("config option values = %#v, want [low high]", got)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/core/conversions_parsers_test.go` around lines 145 - 151, The
test currently only checks ConfigOptions[0].Values length; tighten it by
asserting the actual allowed values (e.g., that
payload.ACPCaps.ConfigOptions[0].Values contains "low" and "high" or equals
[]string{"low","high"} depending on expected order) in addition to existing
checks for ID == "reasoning_effort", Current == "high", and Kind == "select";
update the assertion around payload.ACPCaps.ConfigOptions[0] to explicitly
verify the concrete value strings rather than just the length so the contract
fails if the allowed values change.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The contract assertion at `internal/api/core/conversions_parsers_test.go:145-151` only checks the number of `Values`.
  - A wrong or reordered value set could keep the same length and still pass, weakening the catalog payload contract.
  - Fix: assert the concrete expected values in addition to the existing shape checks.
