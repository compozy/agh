---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: pending
file: internal/settings/collections.go
line: 1195
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaqo,comment:PRRC_kwDOR5y4QM6-7HZN
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Reject curated models without an `id` before writing overlays.**

`providerModelConfigMaps` serializes every entry in `settings.Models.Curated`, even when `model.ID` is blank. That lets the settings API write empty or partial `[[providers.<name>.models.curated]]` entries, and `putProvider` replaces the whole provider block, so one malformed item can persist an invalid provider config.
 
<details>
<summary>One minimal guard</summary>

```diff
 func providerModelConfigMaps(models []aghconfig.ProviderModelConfig) []map[string]any {
 	values := make([]map[string]any, 0, len(models))
 	for _, model := range models {
+		id := strings.TrimSpace(model.ID)
+		if id == "" {
+			continue
+		}
 		entry := make(map[string]any)
-		if strings.TrimSpace(model.ID) != "" {
-			entry["id"] = strings.TrimSpace(model.ID)
-		}
+		entry["id"] = id
 		if strings.TrimSpace(model.DisplayName) != "" {
 			entry["display_name"] = strings.TrimSpace(model.DisplayName)
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
	if len(models.Curated) > 0 {
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
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		entry := make(map[string]any)
		entry["id"] = id
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
		if len(model.ReasoningEfforts) > 0 {
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

In `@internal/settings/collections.go` around lines 1141 - 1195,
providerModelConfigMaps currently serializes every element of
settings.Models.Curated even when model.ID is blank; update
providerModelConfigMaps to skip any model whose strings.TrimSpace(model.ID) ==
"" (i.e., do not create/append entry for that item) so malformed/empty curated
entries are not written into providerModelsSettingsMap and persisted by
putProvider; ensure you still trim model.ID when used and preserve existing
population logic for valid models.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
