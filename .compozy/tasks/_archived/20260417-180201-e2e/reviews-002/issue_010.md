---
status: resolved
file: internal/testutil/acpmock/registration.go
line: 67
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57y10q,comment:PRRC_kwDOR5y4QM644c8v
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Sanitize `runtimeAgentName` before using it in filesystem paths.**

Line 80 and Line 167 use the agent name as a path segment without validation. Values like `../x` or names containing separators can write outside the intended directories.



<details>
<summary>🔒 Proposed fix</summary>

```diff
+func validatePathSegment(value string) (string, error) {
+	trimmed := strings.TrimSpace(value)
+	if trimmed == "" {
+		return "", errors.New("acpmock: agent name is required")
+	}
+	if trimmed == "." || trimmed == ".." || strings.ContainsAny(trimmed, `/\`) {
+		return "", fmt.Errorf("acpmock: invalid agent name %q", trimmed)
+	}
+	return trimmed, nil
+}
+
 func Register(homePaths aghconfig.HomePaths, opts RegisterOptions) (Registration, error) {
@@
 	runtimeAgentName := strings.TrimSpace(opts.AgentName)
 	if runtimeAgentName == "" {
 		runtimeAgentName = fixtureAgentName
 	}
+	runtimeAgentName, err = validatePathSegment(runtimeAgentName)
+	if err != nil {
+		return Registration{}, err
+	}
@@
-	return filepath.Join(dir, strings.TrimSpace(name)+".jsonl"), nil
+	safeName, err := validatePathSegment(name)
+	if err != nil {
+		return "", err
+	}
+	return filepath.Join(dir, safeName+".jsonl"), nil
 }
```
</details>


Also applies to: 80-83, 167-167

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/registration.go` around lines 64 - 67, The agent
name from opts.AgentName is used directly as runtimeAgentName and then as a
filesystem path segment (used in path construction around registration.go);
sanitize it before use by trimming whitespace, rejecting or normalizing
dangerous segments (remove any path separators like '/' and '\\', strip any "."
or ".." segments, and allow only a safe charset such as letters, digits, hyphen
and underscore), and if the sanitized result is empty fall back to
fixtureAgentName; replace the existing runtimeAgentName assignment and any uses
(e.g., where runtimeAgentName is interpolated into file paths) to use this
sanitized value so malicious names cannot escape intended directories.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `runtimeAgentName` is only trimmed before being used as a directory name and diagnostics filename stem. Names containing separators or `.` / `..` can escape the intended directories.
- Fix plan: sanitize runtime agent names as filesystem path segments before using them in `Register()` and diagnostics path construction; reject unsafe values instead of normalizing them into a different agent identity.
- Test impact: requires focused coverage in `internal/testutil/acpmock/fixture_test.go`.
- Resolution: implemented. runtime agent names and diagnostics stems now pass through `sanitizeAgentPathSegment()`, and the acpmock tests reject traversal-style values while still trimming valid names.
- Verification: `go test ./internal/testutil/acpmock`, `make verify`.
