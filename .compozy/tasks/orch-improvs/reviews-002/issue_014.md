---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/situation/task_context.go
line: 675
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-f,comment:PRRC_kwDOR5y4QM6-VcC_
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Broaden the structured secret redaction here.**

This only drops a literal `claim_token` field, so event payloads can still carry other sensitive values into the task bundle unchanged. Please strip the known secret-bearing keys before recursing.

 

<details>
<summary>🔒 Suggested hardening</summary>

```diff
+func isSensitiveTaskContextKey(key string) bool {
+	switch strings.ToLower(strings.TrimSpace(key)) {
+	case "claim_token", "mcp_auth_token", "oauth_code", "pkce_verifier", "secret_bindings":
+		return true
+	default:
+		return false
+	}
+}
+
 func redactTaskContextJSONValue(value any) any {
 	switch typed := value.(type) {
 	case map[string]any:
 		cleaned := make(map[string]any, len(typed))
 		for key, nested := range typed {
-			if strings.EqualFold(strings.TrimSpace(key), "claim_token") {
+			if isSensitiveTaskContextKey(key) {
 				continue
 			}
 			cleaned[key] = redactTaskContextJSONValue(nested)
 		}
 		return cleaned
```
</details>

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings MUST NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory — use hash forms (`claim_token_hash`) over the wire".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/situation/task_context.go` around lines 654 - 675,
redactTaskContextJSONValue currently only drops the literal "claim_token" map
key; update it to strip any structured secret-bearing keys before recursing
(e.g. keys that equal "claim_token" case‑insensitive, that have prefix
"agh_claim_", "mcp_auth_token", or that match names like "oauth_code",
"pkce_verifier", "secret_binding" or common secret suffixes like "_secret") so
their values are omitted entirely and not traversed; keep using
taskpkg.RedactClaimTokens for free-form strings, but for map keys detect these
patterns with strings.EqualFold/strings.HasPrefix/strings.HasSuffix and skip
adding those entries to cleaned so they never appear in the returned structure.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: structured task-context redaction only strips the literal `claim_token` key, leaving other known secret-bearing keys intact in nested JSON payloads.
- Fix approach: Centralize secret-key detection for the known sensitive fields/patterns and extend the redaction tests in `internal/situation/service_test.go`.
