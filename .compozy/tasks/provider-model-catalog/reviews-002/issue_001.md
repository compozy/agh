---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/acp/client.go
line: 549
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYapg,comment:PRRC_kwDOR5y4QM6-7HXx
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Fall back to legacy model state when no model config option is advertised.**

This branch treats any non-empty `ConfigOptions` snapshot as proof that model selection must go through a config option. If an agent exposes only unrelated options here, a valid `preferredModel` now fails startup even though `SupportedModels` still allows it. Only take the config-option path when a model option is actually present; otherwise keep the legacy `session/setModel` fallback.

<details>
<summary>Suggested fix</summary>

```diff
 	caps := process.CapsSnapshot()
 	if len(caps.ConfigOptions) > 0 {
-		option, ok := findModelConfigOption(caps.ConfigOptions)
-		if !ok {
-			return fmt.Errorf("acp: model config option is not available for requested model %q", modelID)
-		}
-		if !configOptionAllowsValue(option, modelID) {
-			return fmt.Errorf("acp: model %q is not available in config option %q", modelID, option.ID)
-		}
-		return d.applySessionConfigOption(ctx, process, option.ID, modelID)
+		if option, ok := findModelConfigOption(caps.ConfigOptions); ok {
+			if !configOptionAllowsValue(option, modelID) {
+				return fmt.Errorf("acp: model %q is not available in config option %q", modelID, option.ID)
+			}
+			return d.applySessionConfigOption(ctx, process, option.ID, modelID)
+		}
 	}
 
 	if !legacyModelStateAllows(caps, modelID) {
 		return fmt.Errorf("acp: model %q is not available in legacy ACP model state", modelID)
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/acp/client.go` around lines 544 - 553, The code currently treats any
non-empty caps.ConfigOptions as requiring config-option-based model selection
which causes valid preferredModel to error when no model option is present;
change the logic to call findModelConfigOption(caps.ConfigOptions) first and
only take the config-option path when it returns ok, then validate with
configOptionAllowsValue and call d.applySessionConfigOption; otherwise fall back
to the legacy session/setModel flow (i.e., do not return an error from this
branch if no model config option is found).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - `internal/acp/client.go` already uses `findModelConfigOption(caps.ConfigOptions)` directly and only takes the config-option path when that option exists.
  - When no model config option is advertised, the code already falls back to `legacyModelStateAllows(...)` and `session/setModel`.
  - No code change is needed in this branch.
  - Resolved as invalid after branch inspection and full verification.
