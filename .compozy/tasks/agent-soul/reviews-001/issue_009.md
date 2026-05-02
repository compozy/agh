---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/api/core/settings_test.go
line: 1742
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_IrdR,comment:PRRC_kwDOR5y4QM69XbzQ
---

# Issue 009: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Add an explicit non-leak assertion for raw MCP secret values.**

This test validates secret ingestion, but it does not verify that the HTTP mutation response never echoes `server-token`. Add a response-body guard so this sensitive-path contract is enforced.

<details>
<summary>Suggested patch</summary>

```diff
 	if got, want := service.LastPutCollectionRequest.MCPSecrets.SecretEnv["TOKEN"], "server-token"; got != want {
 		t.Fatalf("LastPutCollectionRequest.MCPSecrets.SecretEnv[TOKEN] = %q, want %q", got, want)
 	}
+	if strings.Contains(putResp.Body.String(), "server-token") {
+		t.Fatalf("PUT response leaked raw secret value: %s", putResp.Body.String())
+	}
```
</details>

   
As per coding guidelines: "Raw ... secret bindings must NEVER appear in logs, status APIs, settings views ...".


Also applies to: 1770-1772

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/settings_test.go` around lines 1736 - 1742, The test that
validates MCP secret ingestion sets SecretValues with SecretEnv
{"TOKEN":"server-token"} but does not assert the HTTP mutation response never
echoes that raw secret; update the test to capture the HTTP response body (the
mutation/handler response used in this test) and add an explicit assertion that
the body does not contain the raw secret value (e.g., "server-token" or
SecretValues.SecretEnv["TOKEN"]) and/or the secret binding key, and apply the
same non-leak assertion to the other occurrence around lines 1770-1772; locate
the test that constructs contract.SettingsMCPSecretValuesPayload and add the
guard immediately after the request/response step so any response or logged
output is checked for absence of the raw secret.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
