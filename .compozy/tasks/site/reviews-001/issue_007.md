---
status: resolved
file: internal/cli/docpost/docpost.go
line: 205
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:24eb12a9887a
review_hash: 24eb12a9887a
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 007: Wrap meta-generation errors with the failing path.
## Review Comment

These helpers return several bare filesystem/JSON errors, so a failure eventually surfaces as a generic post-process error without telling you which directory or file broke. Please add path-specific context before returning.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

## Triage

- Decision: `valid`
- Notes:
  - The meta-generation path currently lets several filesystem/JSON errors bubble up without enough location context, especially when walking nested CLI-reference directories.
  - Root cause: `writeSubdirMetas` / `writeDirMeta` return bare errors from `filepath.Rel`, `os.ReadDir`, `json.MarshalIndent`, and `os.WriteFile`.
  - Fix plan: wrap those failures with the directory or target file path and add a regression test that asserts the returned error names the failing path.
  - Resolution: wrapped meta-generation failures with directory/file context in `internal/cli/docpost/docpost.go` and added `TestWriteDirMeta_ErrorIncludesTargetPath`.
  - Verification: `go test ./internal/cli/...` passed.
