---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/api/udsapi/model_catalog_test.go
line: 15
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6sP,comment:PRRC_kwDOR5y4QM6-6brw
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Subtest name should follow `Should ...` convention with spaces.**

Line 15 uses `"ShouldPassModelCatalogServiceToBaseHandlers"` (camelCase, no spaces), while lines 35 and 59 in the same file correctly use `"Should expose..."` / `"Should not register..."`. As per coding guidelines, the pattern must be `t.Run("Should ...")` with a space-separated description.

<details>
<summary>🛠️ Proposed fix</summary>

```diff
-	t.Run("ShouldPassModelCatalogServiceToBaseHandlers", func(t *testing.T) {
+	t.Run("Should pass model catalog service to base handlers", func(t *testing.T) {
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	t.Run("Should pass model catalog service to base handlers", func(t *testing.T) {
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/udsapi/model_catalog_test.go` at line 15, The subtest name in
the t.Run call inside model_catalog_test.go uses camelCase
("ShouldPassModelCatalogServiceToBaseHandlers"); change it to the spaced
convention used elsewhere, e.g. "Should pass ModelCatalogService to base
handlers", by updating the string literal in the t.Run invocation so it follows
the "Should ..." space-separated pattern (locate the t.Run call for the subtest
surrounding the test function name).
```

</details>

<!-- fingerprinting:phantom:poseidon:churro -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The subtest name in `internal/api/udsapi/model_catalog_test.go:15` is camel-cased and does not follow the local `Should ...` style used elsewhere in the file.
  - Fix: rename it to a spaced `Should ...` phrase only.
