---
status: resolved
file: internal/skills/bundled/content.go
line: 22
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZt,comment:PRRC_kwDOR5y4QM623eaA
---

# Issue 031: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Restrict skill names to prevent path traversal in bundled reads.**

`LoadContent` accepts arbitrary path fragments in `name`. A value like `../...` can escape the intended `skills/<name>/SKILL.md` namespace in the embedded FS.


<details>
<summary>Proposed hardening patch</summary>

```diff
 func LoadContent(name string) (string, error) {
 	trimmed := strings.TrimSpace(name)
 	if trimmed == "" {
 		return "", fmt.Errorf("bundled: skill name is required")
 	}
+	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, `\`) || trimmed == "." || trimmed == ".." {
+		return "", fmt.Errorf("bundled: invalid skill name %q", name)
+	}
+	cleanName := path.Clean(trimmed)
+	if cleanName != trimmed {
+		return "", fmt.Errorf("bundled: invalid skill name %q", name)
+	}
 
-	skillPath := path.Join("skills", trimmed, skillFileName)
+	skillPath := path.Join("skills", cleanName, skillFileName)
 	content, err := fs.ReadFile(FS(), skillPath)
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("bundled: skill name is required")
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, `\`) || trimmed == "." || trimmed == ".." {
		return "", fmt.Errorf("bundled: invalid skill name %q", name)
	}
	cleanName := path.Clean(trimmed)
	if cleanName != trimmed {
		return "", fmt.Errorf("bundled: invalid skill name %q", name)
	}

	skillPath := path.Join("skills", cleanName, skillFileName)
	content, err := fs.ReadFile(FS(), skillPath)
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/bundled/content.go` around lines 16 - 22, LoadContent
currently builds skillPath from arbitrary name and can be abused for path
traversal; validate and restrict name before joining: ensure the trimmed name
contains no path separators or parent-directory segments (no '/' or '\' and no
".."), or compare trimmed to filepath.Base(trimmed) to enforce a single path
component, and if it fails return an error; then construct skillPath with
path.Join("skills", trimmed, skillFileName) and continue to call FS() and
fs.ReadFile as before. Reference symbols: LoadContent, skillPath, skillFileName,
FS(), fs.ReadFile.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `LoadContent` currently joins the caller-provided `name` directly into `skills/<name>/SKILL.md`. Because the embedded FS accepts path navigation semantics, names containing separators or parent segments can escape the intended namespace. The fix is to reject anything that is not a single clean path component before building the embedded path.
  Resolved by hardening `internal/skills/bundled/content.go` with explicit bundled-skill name validation and exported sentinel errors, plus validation coverage in `internal/skills/bundled/bundled_test.go`. Verified with package tests and a clean `make verify`.
