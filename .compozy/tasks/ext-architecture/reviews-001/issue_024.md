---
status: resolved
file: internal/extension/manager.go
line: 1294
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAah,comment:PRRC_kwDOR5y4QM62zlss
---

# Issue 024: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep manifest-controlled paths inside the extension root.**

Absolute paths are accepted verbatim, and relative values like `../bin/run` survive `filepath.Clean(filepath.Join(rootDir, ...))`. That lets an installed extension execute or load files outside its checksummed directory, which defeats the registry integrity boundary for subprocess commands and resource files.



Also applies to: 1715-1720

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/manager.go` around lines 1279 - 1294, The resolveCommand
implementation allows paths that escape the extension root (absolute paths
accepted verbatim and "../" survives Clean+Join); change it to reject any
resolved path that is outside rootDir: after computing candidate :=
filepath.Clean(filepath.Join(rootDir, resolved)) use filepath.Rel(rootDir,
candidate) and if the relative path starts with ".." (or Rel returns an error)
return an error (or nil+error) instead of allowing the path; also apply the same
containment check/fix to the duplicate logic referenced at the other occurrence
(around lines 1715-1720) so all manifest-controlled paths are validated to
remain inside the extension root.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Path-like manifest values can currently escape `ext.rootDir` through absolute paths or `../` traversal, which breaks the extension-artifact integrity boundary for subprocess commands and resource directories. The duplicate resource-path helper has the same issue.
  Fix approach: add a shared containment check that resolves path-like values and rejects anything outside the extension root, while still allowing bare command names such as `node`.
  Additional test scope needed: `internal/extension/manager_test.go` is outside the batch file list but is the minimal place to validate these manager helpers.
