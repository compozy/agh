---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/core/network_details.go
line: 495
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulJB,comment:PRRC_kwDOR5y4QM680KHL
---

# Issue 005: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Guard `service` before use to avoid nil-interface panic.**

At Line 491, `service.ListPeers` is called without validating `service`. Since `NetworkChannelPayloads` is exported, a nil `service` can reach this path and panic.


<details>
<summary>Suggested fix</summary>

```diff
 func networkChannelAggregates(
 	ctx context.Context,
 	service NetworkService,
 	sessionsManager SessionManager,
 	networkStore NetworkStore,
 ) (map[string]*networkChannelAggregate, error) {
+	if service == nil {
+		return nil, errors.New("api: network service is required")
+	}
 	if networkStore == nil {
 		return nil, errors.New("api: network store is required")
 	}
 	if sessionsManager == nil {
 		return nil, errors.New("api: sessions are required")
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func networkChannelAggregates(
	ctx context.Context,
	service NetworkService,
	sessionsManager SessionManager,
	networkStore NetworkStore,
) (map[string]*networkChannelAggregate, error) {
	if service == nil {
		return nil, errors.New("api: network service is required")
	}
	if networkStore == nil {
		return nil, errors.New("api: network store is required")
	}
	if sessionsManager == nil {
		return nil, errors.New("api: sessions are required")
	}
	runtimePeers, err := service.ListPeers(ctx, "")
	if err != nil {
		return nil, err
	}
	sessions, err := sessionsManager.ListAll(ctx)
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 479 - 495, The function
networkChannelAggregates calls service.ListPeers without validating service; add
a nil check for the service parameter at the top of networkChannelAggregates
(similar to existing checks for networkStore and sessionsManager) and return a
descriptive error like "api: network service is required" to avoid a
nil-interface panic when NetworkService is nil before invoking
service.ListPeers; ensure the check references the service variable so callers
of networkChannelAggregates or exported NetworkChannelPayloads cannot trigger a
panic.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `networkChannelAggregates` validates `networkStore` and `sessionsManager` but calls `service.ListPeers` before checking `service`. Because `NetworkChannelPayloads` is exported, callers can pass a nil `NetworkService` and panic. Add the missing nil check before use.
