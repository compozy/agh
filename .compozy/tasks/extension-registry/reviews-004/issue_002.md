---
status: resolved
file: internal/cli/extension_marketplace.go
line: 164
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564864,comment:PRRC_kwDOR5y4QM63p4Ap
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Persist the resolved source name from the install result too.**

`updateMarketplaceExtension()` hard-requires `info.RegistryName`, but installs currently only fall back to `detail.Source` or `sourceFilter`. If the adapter populates `InstallResult.Source` and leaves `Detail.Source` empty, the row is saved without source metadata and future updates fail as “missing registry source metadata”.


<details>
<summary>Suggested fix</summary>

```diff
-		registryName := firstNonEmpty(detail.Source, strings.TrimSpace(sourceFilter))
+		registryName := firstNonEmpty(result.Source, detail.Source, strings.TrimSpace(sourceFilter))
+		if registryName == "" {
+			return ExtensionRecord{}, fmt.Errorf("cli: extension registry returned no source for %q", slug)
+		}
 		if err := registry.Install(
 			manifest,
 			finalDir,
```
</details>


Also applies to: 250-253

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/extension_marketplace.go` around lines 154 - 161, The install
currently computes registryName via firstNonEmpty(detail.Source,
strings.TrimSpace(sourceFilter)) which ignores result.Source; change the
selection to include result.Source first (e.g., firstNonEmpty(result.Source,
detail.Source, strings.TrimSpace(sourceFilter))) so registry.Install (and its
extensionpkg.WithInstallRegistryMetadata call that supplies slug, registryName,
remoteVersion) persists the resolved InstallResult.Source when present; update
the same logic in the other install sites (the analogous firstNonEmpty usages
around lines ~250-253) so updateMarketplaceExtension has info.RegistryName
populated.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Root cause analysis: the current install flow already persists a resolved source name because `MultiRegistry.Info()` backfills `detail.Source` from the resolved registry source before returning.
- Evidence: [`internal/registry/multi.go`](internal/registry/multi.go) lines 141-142 normalize `detail.Source` with the source name, and the install path stores that normalized value.
- Resolution: No production change was needed. Verified by package tests and the final `make verify` pass.
