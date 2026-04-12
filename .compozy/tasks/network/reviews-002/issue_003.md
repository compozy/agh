---
status: resolved
file: internal/acp/client.go
line: 660
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TjvP,comment:PRRC_kwDOR5y4QM624LJP
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In Go's os/exec package, for Cmd.Env with duplicate keys (for example PATH repeated multiple times), which value is effective at process start: the first or the last occurrence?`

💡 Result:

In Go's os/exec package, for Cmd.Env with duplicate keys (e.g., PATH appearing multiple times), only the last occurrence of each key in the slice is effective at process start. The first occurrences are ignored.

Citations:

- 1: https://go.dev/src/os/exec/exec.go?s=10901:10947
- 2: https://pkg.go.dev/os/exec@go1.19
- 3: https://pkg.go.dev/os/exec
- 4: https://pkg.go.dev/os/exec@go1.25.6

---



</details>

**Fix duplicate environment variable handling to respect last occurrence.**

Go's `os/exec` package honors the last occurrence of duplicate environment variable keys at process start. The current implementation updates the first occurrence in `setEnvValue` and reads the first occurrence in `envValue`, creating a mismatch where stale entries can override intended values. If `env` contains duplicates (e.g., multiple `AGH_BIN` or `PATH` entries), the pinning will fail silently.

Both functions must be updated to handle the last occurrence: `envValue` should scan from the end to return the last match, and `setEnvValue` should remove all occurrences and append the new value at the end to ensure it takes effect.

<details>
<summary>🔧 Proposed fix</summary>

```diff
 func envValue(env []string, key string) (string, bool) {
 	prefix := key + "="
-	for _, variable := range env {
-		if strings.HasPrefix(variable, prefix) {
-			return variable[len(prefix):], true
-		}
+	for i := len(env) - 1; i >= 0; i-- {
+		variable := env[i]
+		if strings.HasPrefix(variable, prefix) {
+			return variable[len(prefix):], true
+		}
 	}
 	return "", false
 }
 
 func setEnvValue(env []string, key string, value string) []string {
 	prefix := key + "="
 	entry := prefix + value
-	for idx, variable := range env {
-		if strings.HasPrefix(variable, prefix) {
-			env[idx] = entry
-			return env
-		}
+	filtered := env[:0]
+	for _, variable := range env {
+		if strings.HasPrefix(variable, prefix) {
+			continue
+		}
+		filtered = append(filtered, variable)
 	}
-	return append(env, entry)
+	return append(filtered, entry)
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func envValue(env []string, key string) (string, bool) {
	prefix := key + "="
	for i := len(env) - 1; i >= 0; i-- {
		variable := env[i]
		if strings.HasPrefix(variable, prefix) {
			return variable[len(prefix):], true
		}
	}
	return "", false
}

func setEnvValue(env []string, key string, value string) []string {
	prefix := key + "="
	entry := prefix + value
	filtered := env[:0]
	for _, variable := range env {
		if strings.HasPrefix(variable, prefix) {
			continue
		}
		filtered = append(filtered, variable)
	}
	return append(filtered, entry)
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client.go` around lines 570 - 590, The env handling currently
reads/updates the first matching key but Go uses the last occurrence at process
start; update envValue to iterate from the end of the env slice and return the
last matching key (search backwards), and update setEnvValue to remove all
existing entries with the same prefix then append a single "KEY=VALUE" entry at
the end so the new value is the last occurrence; refer to the envValue and
setEnvValue functions to implement these changes.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `envValue` and `setEnvValue` currently read and update the first matching entry, but `os/exec` applies the last duplicate environment variable when starting a child process.
- Fix plan: Update env reads to scan from the end and rewrite env sets so the final slice contains exactly one trailing `KEY=value` entry, then add regression coverage for duplicate `PATH` and `AGH_BIN` handling.
