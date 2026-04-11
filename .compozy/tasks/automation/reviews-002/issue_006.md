---
status: resolved
file: internal/automation/model/template.go
line: 152
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TZaF,comment:PRRC_kwDOR5y4QM623-TN
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`index` target validation still has a fail-open branch.**

Line 147–149 returns `nil` when the target path cannot be normalized (or is empty root `.`). That lets unsupported targets pass validation even though this validator is meant to be strict (`only .Data` for dynamic lookups).

<details>
<summary>🔧 Suggested fix</summary>

```diff
 func validateIndexArgs(args []parse.Node) error {
 	if len(args) == 0 {
-		return nil
+		return errors.New("index requires a target expression")
 	}
 	if expression, ok := variableRootExpression(args[0]); ok {
 		return fmt.Errorf("unsupported index target %q; variable-rooted lookups are not supported", expression)
 	}
 
 	path, ok := templateFieldPath(args[0])
-	if !ok || len(path) == 0 {
-		return nil
+	if !ok || len(path) == 0 {
+		return fmt.Errorf("unsupported index target %q; only .Data is supported for dynamic lookups", args[0].String())
 	}
 	if path[0] != "Data" {
 		return fmt.Errorf("unsupported index target %q; only .Data is supported for dynamic lookups", dottedPath(path))
 	}
 	return nil
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/model/template.go` around lines 146 - 152, The validation
currently returns nil when templateFieldPath(args[0]) fails or yields an empty
path, allowing unsupported index targets to slip through; change the early
return so it returns the same kind of error as the subsequent check.
Specifically, in the block that calls templateFieldPath (the variables path,
ok), replace the "return nil" with a formatted error (using fmt.Errorf)
indicating the unsupported index target and that only .Data is allowed (use
dottedPath(path) or args[0] to populate the %q), so both failure-to-normalize
and empty-path cases are rejected consistently before the later path[0] check.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The current `validateIndexArgs()` logic allows `index` targets to fall through whenever the target normalizes to the root dot or cannot be reduced to a field path, so top-level `{{ index . "payload" }}` slips past validation.
  - The naive suggested patch would break the already-supported `{{ with .Data }}{{ index . "key" }}{{ end }}` case, because dot is intentionally rebound to `.Data` inside that scope.
  - Fix approach: tighten validation with scope-aware dot handling so top-level/root-dot index targets are rejected unless the current dot context is explicitly `.Data`, and add regression coverage for both the rejected and still-allowed forms.
  - Resolution: implemented scope-aware `index` validation that preserves `with .Data` while rejecting unsafe root-dot lookups, added template regressions, and verified with focused `go test` runs plus `make verify`.
