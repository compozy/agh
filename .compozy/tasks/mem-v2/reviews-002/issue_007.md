---
provider: coderabbit
pr: "108"
round: 2
round_created_at: 2026-05-06T04:43:32.489895Z
status: resolved
file: internal/config/config.go
line: 1710
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2cO2,comment:PRRC_kwDOR5y4QM6-Uf-A
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Canonicalize `allow_origins` after accepting it.**

This validator accepts values like `" CLI "` or `"Tool"`, but it leaves the original strings in `AllowOrigins`. Any later exact match against canonical origin keys can still reject a supposedly valid config.

<details>
<summary>Suggested fix</summary>

```diff
-func (c MemoryControllerPolicyConfig) Validate() error {
+func (c *MemoryControllerPolicyConfig) Validate() error {
 	if c.MaxContentChars <= 0 {
 		return fmt.Errorf("memory.controller.policy.max_content_chars must be positive: %d", c.MaxContentChars)
 	}
 	if c.MaxWritesPerMin <= 0 {
 		return fmt.Errorf("memory.controller.policy.max_writes_per_min must be positive: %d", c.MaxWritesPerMin)
@@
-	seen := make(map[string]struct{}, len(c.AllowOrigins))
+	seen := make(map[string]struct{}, len(c.AllowOrigins))
+	canonical := make([]string, len(c.AllowOrigins))
 	for i, origin := range c.AllowOrigins {
 		normalized := strings.ToLower(strings.TrimSpace(origin))
 		if _, ok := allowedOrigins[normalized]; !ok {
 			return fmt.Errorf("memory.controller.policy.allow_origins[%d] is invalid: %q", i, origin)
 		}
 		if _, ok := seen[normalized]; ok {
 			return fmt.Errorf("memory.controller.policy.allow_origins[%d] duplicates %q", i, origin)
 		}
 		seen[normalized] = struct{}{}
+		canonical[i] = normalized
 	}
+	c.AllowOrigins = canonical
 	return nil
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (c *MemoryControllerPolicyConfig) Validate() error {
	if c.MaxContentChars <= 0 {
		return fmt.Errorf("memory.controller.policy.max_content_chars must be positive: %d", c.MaxContentChars)
	}
	if c.MaxWritesPerMin <= 0 {
		return fmt.Errorf("memory.controller.policy.max_writes_per_min must be positive: %d", c.MaxWritesPerMin)
	}
	allowedOrigins := map[string]struct{}{
		"cli":       {},
		"http":      {},
		"uds":       {},
		"tool":      {},
		"extractor": {},
		"dreaming":  {},
		"file":      {},
		"provider":  {},
	}
	if len(c.AllowOrigins) == 0 {
		return errors.New("memory.controller.policy.allow_origins must not be empty")
	}
	seen := make(map[string]struct{}, len(c.AllowOrigins))
	canonical := make([]string, len(c.AllowOrigins))
	for i, origin := range c.AllowOrigins {
		normalized := strings.ToLower(strings.TrimSpace(origin))
		if _, ok := allowedOrigins[normalized]; !ok {
			return fmt.Errorf("memory.controller.policy.allow_origins[%d] is invalid: %q", i, origin)
		}
		if _, ok := seen[normalized]; ok {
			return fmt.Errorf("memory.controller.policy.allow_origins[%d] duplicates %q", i, origin)
		}
		seen[normalized] = struct{}{}
		canonical[i] = normalized
	}
	c.AllowOrigins = canonical
	return nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/config/config.go` around lines 1679 - 1710, The validator currently
normalizes origins for checking but leaves
MemoryControllerPolicyConfig.AllowOrigins unchanged, causing later exact-match
checks to fail; change Validate to accept a pointer receiver (func (c
*MemoryControllerPolicyConfig) Validate() error) and, inside the loop over
c.AllowOrigins in Validate, replace each entry with its canonical form
(strings.ToLower(strings.TrimSpace(origin))) after validating it (use the same
allowedOrigins/seen logic), so the stored AllowOrigins are normalized for
downstream use.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `MemoryControllerPolicyConfig.Validate` normalizes `allow_origins` only for comparison and uses a value receiver, so accepted mixed-case or padded inputs remain uncanonicalized in the stored config.
- Evidence: downstream code can observe the original strings after validation because `AllowOrigins` is never rewritten.
- Fix plan: switch to a pointer receiver and rewrite accepted origins to their canonical lowercase/trimmed forms during validation, with tests covering the normalization.
- Resolution: implemented in `internal/config/config.go` and validated with targeted config coverage in `internal/config/memory_v2_config_test.go`, which was the minimal extra file needed to prove normalization of accepted origins.
- Verification: targeted `go test` for `internal/config` passed, and fresh `make verify` passed on 2026-05-06.
