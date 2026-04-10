---
status: resolved
file: internal/extension/registry.go
line: 285
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAar,comment:PRRC_kwDOR5y4QM62zls3
---

# Issue 030: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate the on-disk manifest before persisting registry metadata.**

`installWithSource` verifies the checksum for `artifactRoot`, but it writes `manifest.Name` and `manifest.Version` from the caller-supplied struct. A mismatched `(manifest, path)` pair can therefore install successfully and only fail later when the manager reloads the manifest from `manifestPath`. Reload the resolved manifest here, or at least compare its identity/version to the supplied one before the insert.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/registry.go` around lines 248 - 285, installWithSource
currently validates the artifact checksum but then persists manifest fields
(manifest.Name, manifest.Version) from the caller-supplied manifest without
verifying they match the manifest actually written to disk at manifestPath;
reload the manifest from manifestPath (or read/parse the on-disk manifest file)
after resolveInstallArtifact and compare its Name and Version to the supplied
manifest (or replace the supplied values) before constructing ExtensionInfo and
inserting into the registry; update installWithSource to fail with a clear error
if the on-disk manifest identity/version differ from
manifest.Name/manifest.Version so the persisted ExtensionInfo and ManifestPath
are guaranteed consistent.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `installWithSource` verifies the on-disk artifact checksum, but it persists name/version/capability data from the caller-supplied manifest instead of the manifest actually present at `manifestPath`. A mismatched `(manifest, path)` pair can therefore persist inconsistent registry metadata.
  Fix approach: reload the on-disk manifest after resolving the install artifact, fail on identity/version mismatch, and persist registry metadata from the on-disk manifest.
  Additional test scope needed: `internal/extension/registry_test.go` is outside the batch file list but is the minimal place to verify the registry persistence behavior.
