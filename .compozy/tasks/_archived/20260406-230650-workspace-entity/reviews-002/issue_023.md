---
status: resolved
file: internal/store/schema.go
line: 670
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCr,comment:PRRC_kwDOR5y4QM61T6IL
---

# Issue 023: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Potential infinite loop if all candidate names are taken.**

The `uniqueWorkspaceName` function loops indefinitely trying suffix increments. While practically unlikely to exhaust all integers, a safeguard would be prudent for robustness.


<details>
<summary>🛡️ Suggested safeguard</summary>

```diff
 func uniqueWorkspaceName(rootDir string, taken map[string]struct{}) string {
 	baseName := filepath.Base(filepath.Clean(strings.TrimSpace(rootDir)))
 	switch baseName {
 	case "", ".", string(filepath.Separator):
 		baseName = "workspace"
 	}
 
 	candidate := baseName
-	for suffix := 2; ; suffix++ {
+	for suffix := 2; suffix < 100000; suffix++ {
 		if _, ok := taken[candidate]; !ok {
 			return candidate
 		}
 		candidate = fmt.Sprintf("%s-%d", baseName, suffix)
 	}
+	return fmt.Sprintf("%s-%d", baseName, time.Now().UnixNano())
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func uniqueWorkspaceName(rootDir string, taken map[string]struct{}) string {
	baseName := filepath.Base(filepath.Clean(strings.TrimSpace(rootDir)))
	switch baseName {
	case "", ".", string(filepath.Separator):
		baseName = "workspace"
	}

	candidate := baseName
	for suffix := 2; suffix < 100000; suffix++ {
		if _, ok := taken[candidate]; !ok {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", baseName, suffix)
	}
	return fmt.Sprintf("%s-%d", baseName, time.Now().UnixNano())
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/schema.go` around lines 656 - 670, The uniqueWorkspaceName
function can loop indefinitely; add a safety limit and deterministic fallback:
define a maxAttempts constant (e.g., 1_000_000) and change the for loop to stop
when suffix > maxAttempts, and if no unused candidate is found by then return a
fallback composed from baseName plus a guaranteed-unique token (e.g., timestamp
or random uint64) to avoid an infinite loop; update references in
uniqueWorkspaceName accordingly and ensure any random/timestamp helper used is
imported/seeded where needed.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  This comment is stale against the current file. `internal/store/schema.go` no
  longer defines a local `uniqueWorkspaceName` loop; it delegates to
  `aghworkspace.UniqueWorkspaceName` in `internal/workspace/naming.go`, which is
  outside this batch. The suggested nondeterministic fallback would also change
  workspace naming behavior outside the reported scope. No change in this batch.
