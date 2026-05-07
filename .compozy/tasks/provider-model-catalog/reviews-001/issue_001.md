---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/acp/client.go
line: 569
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6rx,comment:PRRC_kwDOR5y4QM6-6brA
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don't hard-fail `PreferredModel` when config options are unrelated.**

The current branch makes any non-empty `ConfigOptions` authoritative. An agent that only exposes `reasoning_effort` via config options but still publishes models through `SupportedModels` will now reject a valid `PreferredModel` instead of using the legacy model path. Only take the config-option path when `findModelConfigOption` succeeds.

 
<details>
<summary>Suggested fix</summary>

```diff
 	caps := process.CapsSnapshot()
-	if len(caps.ConfigOptions) > 0 {
-		option, ok := findModelConfigOption(caps.ConfigOptions)
-		if !ok {
-			return fmt.Errorf("acp: model config option is not available for requested model %q", modelID)
-		}
+	if option, ok := findModelConfigOption(caps.ConfigOptions); ok {
 		if !configOptionAllowsValue(option, modelID) {
 			return fmt.Errorf("acp: model %q is not available in config option %q", modelID, option.ID)
 		}
 		return d.applySessionConfigOption(ctx, process, option.ID, modelID)
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	caps := process.CapsSnapshot()
	if option, ok := findModelConfigOption(caps.ConfigOptions); ok {
		if !configOptionAllowsValue(option, modelID) {
			return fmt.Errorf("acp: model %q is not available in config option %q", modelID, option.ID)
		}
		return d.applySessionConfigOption(ctx, process, option.ID, modelID)
	}

	if !legacyModelStateAllows(caps, modelID) {
		return fmt.Errorf("acp: model %q is not available in legacy ACP model state", modelID)
	}

	_, err := acpsdk.SendRequest[acpsdk.UnstableSetSessionModelResponse](
		process.conn,
		ctx,
		acpsdk.AgentMethodSessionSetModel,
		acpsdk.UnstableSetSessionModelRequest{
			SessionId: acpsdk.SessionId(process.SessionID),
			ModelId:   acpsdk.UnstableModelId(modelID),
		},
	)
	return err
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/acp/client.go` around lines 544 - 569, The code currently treats any
non-empty caps.ConfigOptions as authoritative and returns an error when
findModelConfigOption fails; instead, only take the config-option branch when
findModelConfigOption actually finds a matching option. Change the logic in the
block around CapsSnapshot/ConfigOptions so that you call
findModelConfigOption(caps.ConfigOptions) and if it returns ok == true then
enforce configOptionAllowsValue(...) and call applySessionConfigOption(...); if
ok == false, do NOT return an error but continue to the legacy path
(legacyModelStateAllows and SendRequest) so SupportedModels/PreferredModel can
still be used. Reference symbols: CapsSnapshot, caps.ConfigOptions,
findModelConfigOption, configOptionAllowsValue, applySessionConfigOption,
legacyModelStateAllows, acpsdk.UnstableSetSessionModelRequest, modelID.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/acp/client.go:545-549` treats any non-empty `caps.ConfigOptions` as authoritative, even when no model option exists.
  - That rejects valid `PreferredModel` overrides for agents that expose unrelated config options such as `reasoning_effort` while still using legacy `SupportedModels`.
  - Fix: only take the config-option path when `findModelConfigOption(...)` succeeds; otherwise continue into the legacy `legacyModelStateAllows` branch.
  - Extra scoped justification: I added `internal/acp/client_test.go` outside the batch file list as the minimal regression test needed to lock the fallback behavior.
