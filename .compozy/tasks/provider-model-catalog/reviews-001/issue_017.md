---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/extension/tool_runtime.go
line: 123
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6st,comment:PRRC_kwDOR5y4QM6-6bsY
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Manifest-first fallback can unintentionally block valid extension methods**

`provides` is overwritten whenever `ext.manifest != nil`, even if the manifest does not effectively grant any service methods. That can make active extensions fail with `ErrToolUnavailable` despite runtime-advertised capabilities.

 

<details>
<summary>💡 Proposed fix</summary>

```diff
-	provides := ext.info.Capabilities.Provides
-	if ext.manifest != nil {
-		provides = ext.manifest.Capabilities.Provides
-	}
+	provides := ext.info.Capabilities.Provides
+	if ext.manifest != nil {
+		manifestMethods := extensionprotocol.CapabilityServiceMethods(ext.manifest.Capabilities.Provides)
+		if len(manifestMethods) > 0 {
+			provides = ext.manifest.Capabilities.Provides
+		}
+	}
@@
-	if !slices.Contains(extensionprotocol.CapabilityServiceMethods(provides), methodName) {
+	grantedMethods := extensionprotocol.CapabilityServiceMethods(provides)
+	if !slices.Contains(grantedMethods, methodName) {
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/extension/tool_runtime.go` around lines 108 - 123, The current code
unconditionally replaces provides with ext.manifest.Capabilities.Provides
whenever ext.manifest != nil, which can drop runtime-advertised service methods;
change the logic so you only overwrite provides when the manifest actually
advertises service methods: compute manifestProvides :=
ext.manifest.Capabilities.Provides and if
slices.ContainsAny(extensionprotocol.CapabilityServiceMethods(manifestProvides),
methodName) or manifestProvides contains any service-capability entries then set
provides = manifestProvides, otherwise keep the original
ext.info.Capabilities.Provides; update the check that calls
extensionprotocol.CapabilityServiceMethods(provides) and preserve the existing
error return using methodName and toolspkg.ErrToolUnavailable.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/extension/tool_runtime.go:108-111` unconditionally prefers manifest `provides` over runtime `ext.info.Capabilities.Provides` whenever a manifest exists.
  - If the manifest is present but does not actually grant any service methods, valid runtime-advertised methods can be masked and incorrectly fail with `ErrToolUnavailable`.
  - Fix: only let the manifest override when it contributes at least one service method; otherwise preserve the runtime capability set.
