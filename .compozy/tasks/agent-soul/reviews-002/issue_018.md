---
provider: coderabbit
pr: "88"
round: 2
round_created_at: 2026-05-02T22:54:45.308545Z
status: resolved
file: internal/cli/client.go
line: 830
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_Irde,comment:PRRC_kwDOR5y4QM69Xbzd
---

# Issue 018: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Require non-empty vault `ref` for metadata/delete operations.**

`GetVaultSecret` and `DeleteVaultSecret` allow blank refs; `vaultRefValues` then omits `ref` entirely. These are targeted operations and should fail fast client-side to avoid ambiguous (or overly broad) server behavior.  
 

<details>
<summary>Suggested fix</summary>

```diff
 func (c *unixSocketClient) GetVaultSecret(ctx context.Context, ref string) (VaultRecord, error) {
+	trimmedRef := strings.TrimSpace(ref)
+	if trimmedRef == "" {
+		return VaultRecord{}, errors.New("cli: vault ref is required")
+	}
 	var response struct {
 		Secret VaultRecord `json:"secret"`
 	}
 	if err := c.doJSON(
 		ctx,
 		http.MethodGet,
 		"/api/vault/secrets/metadata",
-		vaultRefValues(ref),
+		vaultRefValues(trimmedRef),
 		nil,
 		&response,
 	); err != nil {
 		return VaultRecord{}, err
 	}
 	return response.Secret, nil
 }

 func (c *unixSocketClient) DeleteVaultSecret(ctx context.Context, ref string) error {
-	return c.doJSON(ctx, http.MethodDelete, "/api/vault/secrets", vaultRefValues(ref), nil, nil)
+	trimmedRef := strings.TrimSpace(ref)
+	if trimmedRef == "" {
+		return errors.New("cli: vault ref is required")
+	}
+	return c.doJSON(ctx, http.MethodDelete, "/api/vault/secrets", vaultRefValues(trimmedRef), nil, nil)
 }
```
</details>


Also applies to: 845-847, 2853-2859

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/client.go` around lines 815 - 830, Add a fast-fail check for an
empty vault ref at the start of targeted vault operations: in
unixSocketClient.GetVaultSecret (and same for DeleteVaultSecret and the other
occurrences referenced) validate that the incoming ref string is non-empty and
return a descriptive error (e.g., "vault ref required" or similar) before
calling vaultRefValues or making the request; this prevents omitting the ref
param and avoids ambiguous/broad server behavior—update each function
(GetVaultSecret, DeleteVaultSecret, and the two other occurrences) to perform
this check and return early on empty ref.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `GetVaultSecret` and `DeleteVaultSecret` currently allow an empty ref, and `vaultRefValues` then silently omits the query parameter.
  - These are targeted operations and should fail fast on the client side; I added `requireVaultRef` in `internal/cli/client.go` so empty refs now return a descriptive client error before any request is issued.
  - Verification: `make verify` passed with the new vault-ref validation.
