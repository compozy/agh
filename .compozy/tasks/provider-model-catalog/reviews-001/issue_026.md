---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/settings/collections.go
line: 1194
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6tc,comment:PRRC_kwDOR5y4QM6-6btV
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Preserve explicit empty model-slice overrides.**

A replace request that sends `models.curated: []` or `reasoning_efforts: []` is serialized as if those fields were omitted, so inherited builtin values survive reload and the provider overlay cannot actually clear them.  
 

<details>
<summary>Suggested fix</summary>

```diff
 func providerModelsSettingsMap(models aghconfig.ProviderModelsConfig) map[string]any {
 	values := make(map[string]any)
 	if strings.TrimSpace(models.Default) != "" {
 		values["default"] = strings.TrimSpace(models.Default)
 	}
-	if len(models.Curated) > 0 {
+	if models.Curated != nil {
 		values["curated"] = providerModelConfigMaps(models.Curated)
 	}
 	if discovery := providerModelsDiscoveryMap(models.Discovery); len(discovery) > 0 {
 		values["discovery"] = discovery
 	}
 	return values
 }
@@
-		if len(model.ReasoningEfforts) > 0 {
+		if model.ReasoningEfforts != nil {
 			entry["reasoning_efforts"] = cloneStringSlicePreserveNil(model.ReasoningEfforts)
 		}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func providerModelsSettingsMap(models aghconfig.ProviderModelsConfig) map[string]any {
	values := make(map[string]any)
	if strings.TrimSpace(models.Default) != "" {
		values["default"] = strings.TrimSpace(models.Default)
	}
	if models.Curated != nil {
		values["curated"] = providerModelConfigMaps(models.Curated)
	}
	if discovery := providerModelsDiscoveryMap(models.Discovery); len(discovery) > 0 {
		values["discovery"] = discovery
	}
	return values
}

func providerModelConfigMaps(models []aghconfig.ProviderModelConfig) []map[string]any {
	values := make([]map[string]any, 0, len(models))
	for _, model := range models {
		entry := make(map[string]any)
		if strings.TrimSpace(model.ID) != "" {
			entry["id"] = strings.TrimSpace(model.ID)
		}
		if strings.TrimSpace(model.DisplayName) != "" {
			entry["display_name"] = strings.TrimSpace(model.DisplayName)
		}
		if model.ContextWindow != nil {
			entry["context_window"] = *model.ContextWindow
		}
		if model.MaxInputTokens != nil {
			entry["max_input_tokens"] = *model.MaxInputTokens
		}
		if model.MaxOutputTokens != nil {
			entry["max_output_tokens"] = *model.MaxOutputTokens
		}
		if model.SupportsTools != nil {
			entry["supports_tools"] = *model.SupportsTools
		}
		if model.SupportsReasoning != nil {
			entry["supports_reasoning"] = *model.SupportsReasoning
		}
		if model.ReasoningEfforts != nil {
			entry["reasoning_efforts"] = cloneStringSlicePreserveNil(model.ReasoningEfforts)
		}
		if strings.TrimSpace(model.DefaultReasoningEffort) != "" {
			entry["default_reasoning_effort"] = strings.TrimSpace(model.DefaultReasoningEffort)
		}
		if model.CostInputPerMillion != nil {
			entry["cost_input_per_million"] = *model.CostInputPerMillion
		}
		if model.CostOutputPerMillion != nil {
			entry["cost_output_per_million"] = *model.CostOutputPerMillion
		}
		values = append(values, entry)
	}
	return values
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/settings/collections.go` around lines 1141 - 1194, The code
currently omits fields when slices are empty, preventing explicit empty
overrides; update providerModelsSettingsMap to add "curated" when models.Curated
is non-nil (not only when len>0) by assigning
providerModelConfigMaps(models.Curated) so an explicit [] is preserved, and
update providerModelConfigMaps to emit "reasoning_efforts" when
model.ReasoningEfforts is non-nil (even if len==0) by calling
cloneStringSlicePreserveNil(model.ReasoningEfforts) when model.ReasoningEfforts
!= nil; similarly ensure any other slice fields (e.g., discovery) use nil-checks
instead of len()>0 so explicit empty slices are serialized.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `providerModelsSettingsMap` only emits `curated` when `len(models.Curated) > 0`, and `providerModelConfigMaps` only emits `reasoning_efforts` when `len(model.ReasoningEfforts) > 0`.
  - That conflates `nil` with an explicit empty slice and makes it impossible for a settings overlay to clear inherited model arrays on writeback.
  - Fix approach: preserve explicit empty overrides by checking for `nil` instead of non-zero length when serializing those slice fields.
  - Scope expansion was required: `internal/config/persistence.go` and `internal/config/persistence_test.go` now preserve empty array-of-table values as explicit `key = []`, which is necessary for the `curated` override to survive reload.
  - Resolved in `internal/settings/collections.go`, `internal/config/persistence.go`, `internal/settings/service_test.go`, and `internal/config/persistence_test.go`; verified with focused package tests and full `make verify`.
