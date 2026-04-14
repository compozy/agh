---
status: resolved
file: internal/registry/extract.go
line: 129
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WUf,comment:PRRC_kwDOR5y4QM63madk
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Preserve archive permission bits.**

Regular files are always created as `0644` and directories as `0755`, so executable assets lose their mode on install. That will break extensions shipped as scripts or binaries inside release archives.


<details>
<summary>🛠️ Suggested fix</summary>

```diff
 		switch header.Typeflag {
 		case tar.TypeDir:
-			if err := os.MkdirAll(targetPath, 0o755); err != nil {
+			dirMode := os.FileMode(header.Mode) & os.ModePerm
+			if dirMode == 0 {
+				dirMode = 0o755
+			}
+			if err := os.MkdirAll(targetPath, dirMode); err != nil {
 				return fmt.Errorf("create archive directory %q: %w", targetPath, err)
 			}
 		case tar.TypeReg, 0:
 			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
 				return fmt.Errorf("create archive parent %q: %w", filepath.Dir(targetPath), err)
 			}
+			fileMode := os.FileMode(header.Mode) & os.ModePerm
+			if fileMode == 0 {
+				fileMode = 0o644
+			}
 
-			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
+			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode)
 			if err != nil {
 				return fmt.Errorf("create archive file %q: %w", targetPath, err)
 			}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		switch header.Typeflag {
		case tar.TypeDir:
			dirMode := os.FileMode(header.Mode) & os.ModePerm
			if dirMode == 0 {
				dirMode = 0o755
			}
			if err := os.MkdirAll(targetPath, dirMode); err != nil {
				return fmt.Errorf("create archive directory %q: %w", targetPath, err)
			}
		case tar.TypeReg, 0:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("create archive parent %q: %w", filepath.Dir(targetPath), err)
			}
			fileMode := os.FileMode(header.Mode) & os.ModePerm
			if fileMode == 0 {
				fileMode = 0o644
			}

			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode)
			if err != nil {
				return fmt.Errorf("create archive file %q: %w", targetPath, err)
			}

			counter := &countingLimitWriter{
				total: &totalExtracted,
				limit: limits.maxDecompressedSize,
			}
			teeReader := io.TeeReader(tarReader, counter)
			if _, err := io.Copy(file, teeReader); err != nil {
				writeErr := fmt.Errorf("write archive file %q: %w", targetPath, err)
				if closeErr := file.Close(); closeErr != nil {
					writeErr = errors.Join(writeErr, fmt.Errorf("close archive file %q after write failure: %w", targetPath, closeErr))
				}
				_ = os.Remove(targetPath)
				return writeErr
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close archive file %q: %w", targetPath, err)
```

</details>

<!-- suggestion_end -->

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: The extractor currently creates regular files with `0644` and directories with `0755` regardless of the tar header, so executable payloads lose their mode bits on install. I will preserve the archive-provided permission bits for created entries and extend `internal/registry/extract_test.go` with mode-preservation coverage; that test file is outside the listed scope because no in-scope test file exercises `ExtractArchive()`.
- Resolution: `internal/registry/extract.go` now derives file and directory permissions from tar headers and reapplies them after creation, with mode-preservation coverage in `internal/registry/extract_test.go`.
- Verification: `go test ./internal/registry/...`; `make verify`
