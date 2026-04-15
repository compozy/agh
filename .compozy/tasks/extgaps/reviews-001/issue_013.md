---
status: resolved
file: internal/extension/install_managed.go
line: 72
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__B4,comment:PRRC_kwDOR5y4QM63zbyZ
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
find . -name "install_managed.go" -type f
```

Repository: compozy/agh

Length of output: 94

---

🏁 Script executed:

```shell
cat -n ./internal/extension/install_managed.go | sed -n '60,120p'
```

Repository: compozy/agh

Length of output: 2203

---



</details>

**Wrap the new checksum/install failures with operation context.**

These error paths return raw errors from `ComputeDirectoryChecksum` and `registry.Install`, violating the requirement to wrap with context. Lines 71, 108, and 116 need wrapped returns to help callers diagnose which install phase failed.

<details>
<summary>Suggested change</summary>

```diff
 	actualSourceChecksum, err := ComputeDirectoryChecksum(sourceDir)
 	if err != nil {
-		return err
+		return fmt.Errorf("extension: compute source checksum %q: %w", sourceDir, err)
 	}
@@
 	installedChecksum, err := ComputeDirectoryChecksum(finalDir)
 	if err != nil {
+		checksumErr := fmt.Errorf("extension: compute installed checksum %q: %w", finalDir, err)
 		removeErr := os.RemoveAll(finalDir)
 		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
-			return errors.Join(err, fmt.Errorf("extension: remove failed local install %q after checksum error: %w", finalDir, removeErr))
+			return errors.Join(checksumErr, fmt.Errorf("extension: remove failed local install %q after checksum error: %w", finalDir, removeErr))
 		}
-		return err
+		return checksumErr
 	}
 
 	if err := registry.Install(manifest, finalDir, installedChecksum, opts...); err != nil {
+		installErr := fmt.Errorf("extension: persist managed extension %q: %w", manifest.Name, err)
 		removeErr := os.RemoveAll(finalDir)
 		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
-			return errors.Join(err, fmt.Errorf("extension: remove failed local install %q: %w", finalDir, removeErr))
+			return errors.Join(installErr, fmt.Errorf("extension: remove failed local install %q: %w", finalDir, removeErr))
 		}
-		return err
+		return installErr
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/install_managed.go` around lines 69 - 72, Wrap errors
returned from ComputeDirectoryChecksum and registry.Install with contextual
messages using error wrapping (e.g., fmt.Errorf("...: %w", err)) so callers know
which install phase failed; for the ComputeDirectoryChecksum call that produces
actualSourceChecksum use a message like "compute checksum for sourceDir
<sourceDir>" and for registry.Install wrap its errors with messages indicating
the operation and extension/ID (e.g., "registry install for extension
<extensionName/ID>" or "verify install for <extensionName>"). Ensure you include
the original error via %w to preserve the cause.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the new checksum/install paths in `InstallLocalManaged` still return raw errors from `ComputeDirectoryChecksum` and `registry.Install`, which violates the repository error-wrapping rule and obscures which install phase failed.
- Fix plan: wrap source-checksum, installed-checksum, and registry-install failures with phase-specific context, preserving cleanup errors via `errors.Join`.
- Resolution: wrapped checksum/install failures with explicit install-phase context and preserved cleanup failures via `errors.Join`.
- Verification: added coverage in `internal/extension/install_managed_test.go` and passed `go test ./internal/extension` plus `make verify`.
