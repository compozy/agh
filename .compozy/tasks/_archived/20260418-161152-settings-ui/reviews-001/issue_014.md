---
status: resolved
file: internal/config/persistence.go
line: 1390
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRz,comment:PRRC_kwDOR5y4QM65B60h
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Persist config atomically and create the directory privately.**

A crash or concurrent read during `os.WriteFile` can leave a truncated overlay/sidecar on disk, and `0o755` makes the config directory world-executable even though these files can contain secrets.


<details>
<summary>🔒 Proposed fix</summary>

```diff
 func writePersistedFile(path string, contents []byte) error {
-	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
-		return fmt.Errorf("create config directory %q: %w", filepath.Dir(path), err)
-	}
-	if err := os.WriteFile(path, contents, 0o600); err != nil {
-		return fmt.Errorf("write config file %q: %w", path, err)
-	}
-	return nil
+	dir := filepath.Dir(path)
+	if err := os.MkdirAll(dir, 0o700); err != nil {
+		return fmt.Errorf("create config directory %q: %w", dir, err)
+	}
+
+	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
+	if err != nil {
+		return fmt.Errorf("create temp config file in %q: %w", dir, err)
+	}
+	tmpPath := tmp.Name()
+	defer os.Remove(tmpPath)
+
+	if err := tmp.Chmod(0o600); err != nil {
+		_ = tmp.Close()
+		return fmt.Errorf("chmod temp config file %q: %w", tmpPath, err)
+	}
+	if _, err := tmp.Write(contents); err != nil {
+		_ = tmp.Close()
+		return fmt.Errorf("write temp config file %q: %w", tmpPath, err)
+	}
+	if err := tmp.Sync(); err != nil {
+		_ = tmp.Close()
+		return fmt.Errorf("sync temp config file %q: %w", tmpPath, err)
+	}
+	if err := tmp.Close(); err != nil {
+		return fmt.Errorf("close temp config file %q: %w", tmpPath, err)
+	}
+	if err := os.Rename(tmpPath, path); err != nil {
+		return fmt.Errorf("replace config file %q: %w", path, err)
+	}
+	return nil
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/persistence.go` around lines 1384 - 1390, In
writePersistedFile, avoid truncated files and overly-permissive dirs by creating
the parent directory with restrictive mode (0o700) instead of 0o755, then
perform an atomic write: write contents to a temp file in the same directory
(with file mode 0o600), fsync the temp file (and optionally the directory), and
finally rename the temp file to path using os.Rename; update error handling to
report failures for temp file creation/write/sync/rename.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `writePersistedFile`: it creates config directories with mode `0755` and writes files directly with `os.WriteFile`, which can expose secrets through overly broad directory permissions and can leave truncated files behind on interruption. I will switch this path to private directory creation and same-directory atomic temp-file replacement.
