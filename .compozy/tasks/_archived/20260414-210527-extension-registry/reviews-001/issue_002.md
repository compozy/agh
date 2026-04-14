---
status: resolved
file: internal/cli/extension_marketplace.go
line: 434
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WUW,comment:PRRC_kwDOR5y4QM63madZ
---

# Issue 002: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Guard `args[0]` before indexing.**

When `updateAll` is false and the command is invoked without a name, this panics with `index out of range` instead of returning a CLI error.  


<details>
<summary>🐛 Proposed fix</summary>

```diff
 func selectMarketplaceExtensionsForUpdate(
 	registry localExtensionRegistry,
 	args []string,
 	updateAll bool,
 ) ([]extensionpkg.ExtensionInfo, error) {
 	if updateAll {
 		infos, err := registry.List()
 		if err != nil {
 			return nil, err
 		}
@@
 		}
 		return items, nil
 	}
 
+	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
+		return nil, errors.New("cli: extension name is required unless updateAll is set")
+	}
+
 	info, err := registry.Get(args[0])
 	if err != nil {
 		return nil, err
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/extension_marketplace.go` around lines 406 - 434,
selectMarketplaceExtensionsForUpdate currently indexes args[0] without checking
length which panics when updateAll is false and no name is provided; update the
function to first guard that len(args) > 0 and return a clear CLI error (e.g.,
fmt.Errorf("cli: no extension name provided")) before calling registry.Get or
marketplaceExtensionInstalled, so that the code path using args[0] is only
executed when an argument is present; adjust the error return to match existing
error style and keep the rest of the logic (registry.Get,
marketplaceExtensionInstalled, and returning the slice) unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `selectMarketplaceExtensionsForUpdate()` dereferences `args[0]` without a length check when `updateAll` is false, so invoking the command without an extension name panics instead of returning a CLI error. I will guard the argument before lookup and add coverage in `internal/cli/extension_marketplace_test.go`; that extra test file is outside the listed scope but is the minimal place to validate the CLI helper.
- Resolution: Added a required-name guard before `registry.Get(...)` in `selectMarketplaceExtensionsForUpdate()` and covered the error path in `internal/cli/extension_marketplace_test.go`.
- Verification: `go test ./internal/cli`; `make verify`
