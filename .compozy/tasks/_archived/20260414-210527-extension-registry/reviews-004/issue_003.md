---
status: resolved
file: internal/cli/extension_marketplace.go
line: 205
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM567Hju,comment:PRRC_kwDOR5y4QM63sxXr
---

# Issue 003: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Validate stored manifest paths before deriving install directories.**

`filepath.Dir("")` returns `"."`. A bad row with an empty or malformed `ManifestPath` makes `remove` and `update` rename the current working directory instead of the extension install directory.

<details>
<summary>Proposed fix</summary>

```diff
+func installedExtensionDir(info extensionpkg.ExtensionInfo) (string, error) {
+	manifestPath := strings.TrimSpace(info.ManifestPath)
+	if manifestPath == "" || filepath.Base(manifestPath) != "extension.toml" {
+		return "", fmt.Errorf("cli: extension %q has an invalid manifest path %q", info.Name, info.ManifestPath)
+	}
+	installDir := filepath.Dir(manifestPath)
+	if installDir == "." || installDir == string(filepath.Separator) {
+		return "", fmt.Errorf("cli: extension %q has an invalid install directory %q", info.Name, installDir)
+	}
+	return installDir, nil
+}
+
 func removeInstalledExtensionWithRegistry(
 	registry localExtensionRegistry,
 	name string,
 	stage func(targetDir string) (extensionDirChange, error),
 ) (_ extensionRemoveItem, err error) {
@@
-	installDir := filepath.Dir(info.ManifestPath)
+	installDir, err := installedExtensionDir(*info)
+	if err != nil {
+		return extensionRemoveItem{}, err
+	}
 	change, err := stage(installDir)
 	if err != nil {
 		return extensionRemoveItem{}, err
 	}
@@
-	item := extensionUpdateItem{
+	installDir, err := installedExtensionDir(info)
+	if err != nil {
+		return extensionUpdateItem{}, err
+	}
+
+	item := extensionUpdateItem{
 		Name:           info.Name,
 		Slug:           slug,
 		Registry:       registryName,
 		CurrentVersion: currentVersion,
 		LatestVersion:  firstNonEmpty(updateInfo.LatestVersion, currentVersion),
-		Path:           filepath.Dir(info.ManifestPath),
+		Path:           installDir,
 	}
```
</details>


Also applies to: 326-333

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/extension_marketplace.go` around lines 204 - 205, The code
derives installDir with filepath.Dir(info.ManifestPath) without validating
info.ManifestPath, so an empty or malformed ManifestPath yields "." and can
cause remove/update to target the CWD; before calling filepath.Dir (in the block
that calls stage(installDir) and the similar block at lines ~326-333), validate
info.ManifestPath (e.g., ensure it is non-empty and looks like a valid path —
optionally check filepath.IsAbs or that filepath.Clean(info.ManifestPath) !=
".") and return an error or skip the row if invalid; update the code that uses
installDir for stage(...), remove, and update to only proceed when ManifestPath
passes validation.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: remove and update derive install directories with `filepath.Dir(info.ManifestPath)` without validating that `ManifestPath` is a real manifest file path.
- Evidence: [`internal/cli/extension_marketplace.go`](internal/cli/extension_marketplace.go) lines 204 and 332 can turn an empty or malformed manifest path into `"."`, which would target the current working directory.
- Fix plan: add a shared manifest-path validator/helper and use it before remove/update stage operations.
- Resolution: Added `installedExtensionDir(...)` validation and regression coverage for invalid manifest paths. Verified with package tests and `make verify`.
