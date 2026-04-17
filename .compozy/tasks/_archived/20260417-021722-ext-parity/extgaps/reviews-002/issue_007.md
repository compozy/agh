---
status: resolved
file: internal/extension/install_managed.go
line: 60
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57AO3q,comment:PRRC_kwDOR5y4QM63zyP3
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Normalize the input checksum before this precheck.**

This compares `actualSourceChecksum` against the raw request value, while `registry.Install` normalizes with trim/lowercase first. A valid checksum like `" ABC... "` or uppercase hex will now fail here even though the downstream install path would accept it.

<details>
<summary>Suggested change</summary>

```diff
-	if strings.TrimSpace(checksum) == "" {
+	normalizedChecksum := strings.ToLower(strings.TrimSpace(checksum))
+	if normalizedChecksum == "" {
 		return errors.New("extension: checksum is required")
 	}
@@
-	if actualSourceChecksum != checksum {
+	if actualSourceChecksum != normalizedChecksum {
 		return &ExtensionChecksumMismatchError{
-			ExpectedChecksum: checksum,
+			ExpectedChecksum: normalizedChecksum,
 			ActualChecksum:   actualSourceChecksum,
 		}
 	}
```

</details>

Also applies to: 73-76

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/install_managed.go` around lines 59 - 60, The precheck is
using the raw checksum request value which can have surrounding whitespace or
uppercase hex and will mismatch later normalization in registry.Install;
normalize the input checksum variable (e.g., apply strings.TrimSpace and
strings.ToLower) before performing the empty check and any comparisons against
actualSourceChecksum so that comparisons in places like the block referencing
actualSourceChecksum and the later comparison region (the code around the other
checksum checks) use the same normalized form as registry.Install; update
references to use the normalized checksum variable for the empty validation and
equality checks.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `InstallLocalManaged(...)` validates the checksum requirement and the source checksum precheck against the raw request string, while downstream registry installation normalizes checksum input with trim/lowercase semantics.
- Why this is valid: a caller can provide a semantically correct checksum with uppercase hex or surrounding whitespace and be rejected before the install path reaches the already-normalized registry validation.
- Fix approach: normalize the incoming checksum once at the start of `InstallLocalManaged(...)` and use the normalized value consistently for empty validation, mismatch reporting, and registry installation. Add regression coverage in `internal/extension/install_managed_test.go`.
- Resolution: `internal/extension/install_managed.go` now normalizes the provided checksum before empty checks and source checksum comparison, and `internal/extension/install_managed_test.go` now covers uppercase/whitespace input.
- Verification: targeted `go test ./internal/extension -run 'TestCopyInstallTreeMaterializesSymlinkTargets|TestInstallLocalManagedUsesInstalledChecksumForMaterializedSymlinks|TestInstallLocalManagedNormalizesProvidedChecksum|TestCopyInstallTreeRejectsSymlinkDirectoryCycles|TestCopyInstallTreeRejectsSymlinkTargetsOutsideSourceRoot|TestInstallLocalManagedWrapsPhaseErrors' -count=1` passed, followed by a clean `make verify`.
