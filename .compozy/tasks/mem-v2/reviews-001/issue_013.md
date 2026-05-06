---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/config/config.go
line: 1950
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Is6,comment:PRRC_kwDOR5y4QM6-UFWM
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Reject or canonicalize mixed-case enum values.**

`validateEnum` accepts `Hybrid`, `LLM`, or ` Jsonl `, but it leaves the original string in the config object. Any downstream exact comparison on these fields can then miss the supposedly validated value and take the wrong path. Either normalize on load or require the canonical literal here.  
 

<details>
<summary>Suggested fix</summary>

```diff
func validateEnum(path string, value string, allowed ...string) error {
-	normalized := strings.ToLower(strings.TrimSpace(value))
+	normalized := strings.TrimSpace(value)
 	if slices.Contains(allowed, normalized) {
 		return nil
 	}
 	return fmt.Errorf("%s must be one of %s: %q", path, strings.Join(allowed, ", "), value)
}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/config/config.go` around lines 1944 - 1950, validateEnum currently
lowercases/trims the input for validation but leaves the original value in
configs, causing downstream exact comparisons to fail; change validateEnum to
return the canonicalized value along with error (signature like
validateEnum(path string, value string, allowed ...string) (string, error)) and
have the function normalize (strings.ToLower/strings.TrimSpace) and return that
normalized string when validation passes, or return an error for invalid input;
update all callers to assign the returned canonical value into the config fields
(or alternatively enforce canonical literals by comparing against the provided
allowed slice without lowercasing and return an explicit error), ensuring the
config stores the normalized/canonical enum rather than the original input.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `validateEnum` still lowercases and trims only for acceptance, but the validated config structs retain the original mixed-case value.
  - Downstream code compares these enum fields to canonical literals such as `"hybrid"`, `"weighted"`, `"post_message"`, and `"jsonl"`, so accepting `Hybrid` or ` Jsonl ` without normalizing the stored value is a real contract bug.
  - Fix approach: canonicalize the stored memory enum fields during validation/load and add a focused config regression test outside the scoped file list because the behavior lives in the config package, not an existing scoped test file.
