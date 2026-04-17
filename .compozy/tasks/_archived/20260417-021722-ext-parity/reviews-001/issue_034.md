---
status: resolved
file: internal/config/mcp_resource.go
line: 48
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQu,comment:PRRC_kwDOR5y4QM64dqG8
---

# Issue 034: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Non-deterministic `Env` canonicalization when trimmed keys collide**

The current in-place map rewrite can produce unstable results if multiple raw keys trim to the same canonical key (map iteration order is random), which can make resource normalization flaky.



<details>
<summary>Suggested deterministic normalization</summary>

```diff
 import (
 	"context"
+	"sort"
 	"strings"
@@
 	if len(normalized.Env) > 0 {
-		for key, value := range normalized.Env {
-			trimmedKey := strings.TrimSpace(key)
-			delete(normalized.Env, key)
-			if trimmedKey == "" {
-				continue
-			}
-			normalized.Env[trimmedKey] = strings.TrimSpace(value)
-		}
-		if len(normalized.Env) == 0 {
-			normalized.Env = nil
-		}
+		keys := make([]string, 0, len(normalized.Env))
+		for key := range normalized.Env {
+			keys = append(keys, key)
+		}
+		sort.Strings(keys)
+		canonicalEnv := make(map[string]string, len(keys))
+		for _, key := range keys {
+			trimmedKey := strings.TrimSpace(key)
+			if trimmedKey == "" {
+				continue
+			}
+			canonicalEnv[trimmedKey] = strings.TrimSpace(normalized.Env[key])
+		}
+		if len(canonicalEnv) == 0 {
+			normalized.Env = nil
+		} else {
+			normalized.Env = canonicalEnv
+		}
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if len(normalized.Env) > 0 {
		keys := make([]string, 0, len(normalized.Env))
		for key := range normalized.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		canonicalEnv := make(map[string]string, len(keys))
		for _, key := range keys {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			canonicalEnv[trimmedKey] = strings.TrimSpace(normalized.Env[key])
		}
		if len(canonicalEnv) == 0 {
			normalized.Env = nil
		} else {
			normalized.Env = canonicalEnv
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/mcp_resource.go` around lines 37 - 48, The in-place rewrite
of normalized.Env is non-deterministic when multiple original keys trim to the
same canonical key; instead, build a new map deterministically by collecting the
original keys, sorting them, then iterating in sorted order to compute
trimmedKey and insert into a new map (skip empty trimmedKey), and finally assign
that new map back to normalized.Env (or nil if empty); refer to normalized.Env
and trimmedKey in your changes to locate and replace the existing loop.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The current `normalized.Env` canonicalization rewrites the existing map in place while iterating it, so when multiple raw keys trim to the same canonical key the winner depends on random map iteration order. That makes normalization nondeterministic and can yield flaky persisted results. The fix is to sort the original keys, build a new canonical map, and add a collision regression test.
