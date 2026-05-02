---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/cli/automation.go
line: 1611
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_Irdd,comment:PRRC_kwDOR5y4QM69Xbzc
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Preserve webhook secret values verbatim in CLI requests.**

`strings.TrimSpace` changes the secret before it ever reaches the API. That breaks any whitespace-sensitive HMAC secret and is very hard to diagnose because the stored value no longer matches what the user provided.

 

<details>
<summary>Suggested fix</summary>

```diff
 	request := AutomationTriggerCreateRequest{
 		Scope:              scope,
 		Name:               strings.TrimSpace(input.Name),
 		AgentName:          strings.TrimSpace(input.AgentName),
 		WorkspaceID:        workspaceID,
 		Prompt:             strings.TrimSpace(input.Prompt),
 		Event:              strings.TrimSpace(input.EventRaw),
 		Filter:             filter,
 		WebhookID:          strings.TrimSpace(input.WebhookID),
 		EndpointSlug:       strings.TrimSpace(input.EndpointSlug),
 		WebhookSecretRef:   strings.TrimSpace(input.WebhookSecretRef),
-		WebhookSecretValue: strings.TrimSpace(input.WebhookSecretValue),
+		WebhookSecretValue: input.WebhookSecretValue,
 	}
@@
 	if cmd.Flags().Changed("webhook-secret-value") {
-		request.WebhookSecretValue = stringPointer(strings.TrimSpace(input.WebhookSecretValue))
+		request.WebhookSecretValue = stringPointer(input.WebhookSecretValue)
 	}
```
</details>


Also applies to: 1675-1679

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/automation.go` around lines 1600 - 1611, The CLI is trimming
whitespace from webhook secret fields so HMAC or whitespace-sensitive secrets
change before reaching the API; in the AutomationTriggerCreateRequest
construction, stop calling strings.TrimSpace on WebhookSecretValue and
WebhookSecretRef so the raw input is preserved verbatim (leave trimming on
non-secret fields like Name/AgentName/Prompt/Event), and make the same change in
the corresponding update request block (the update request that sets
WebhookSecretRef/WebhookSecretValue around lines ~1675-1679) to ensure both
create and update paths send secrets unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
