---
status: pending
file: internal/extension/registry.go
line: 359
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM563lMV,comment:PRRC_kwDOR5y4QM63oCs-
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Preserve `enabled` when replacing an existing extension.**

Line 351 always sets `Enabled: true`, and Line 373 writes that value back on conflict. Any update path using `WithInstallReplaceExisting()` will silently re-enable an extension the user had disabled.

<details>
<summary>💡 Proposed fix</summary>

```diff
 		ON CONFLICT(name) DO UPDATE SET
 			version = excluded.version,
 			source = excluded.source,
-			enabled = excluded.enabled,
 			manifest_path = excluded.manifest_path,
 			installed_at = excluded.installed_at,
 			capabilities = excluded.capabilities,
 			actions = excluded.actions,
 			checksum = excluded.checksum,
```
</details>


Also applies to: 368-381

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/registry.go` around lines 347 - 359, The ExtensionInfo
construction in registry.go always sets Enabled: true which causes
WithInstallReplaceExisting() updates to re-enable previously disabled
extensions; change the replacement logic to preserve the existing Enabled state
instead of forcing true: when building the ExtensionInfo for install/replace,
look up the current stored extension (by Name/RegistrySlug or using
r.get.../r.findExistingExtension helper) and set Enabled to that existing value
if present, otherwise default to true for new installs; update the
conflict-upsert path that writes the ExtensionInfo (the code around the
ExtensionInfo literal and the upsert/write-back on conflict) to use this
preserved Enabled value so replaces do not change user-disabled extensions.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
