---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: pending
file: internal/modelcatalog/redact_test.go
line: 30
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6tB,comment:PRRC_kwDOR5y4QM6-6bsx
---

# Issue 021: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert all secrets in multi-secret inputs.**

Line 28 includes two secret values, but the test only verifies one. A partial-redaction regression could slip through.
 
<details>
<summary>🔒 Suggested fix</summary>

```diff
 	tests := []struct {
 		name   string
 		input  string
-		secret string
+		secrets []string
 	}{
 		{
 			name:   "Should redact OpenAI style API keys",
 			input:  "models.dev failed with api_key=sk-super-secret-token-123",
-			secret: "sk-super-secret-token-123",
+			secrets: []string{"sk-super-secret-token-123"},
 		},
 		{
 			name:   "Should redact OAuth bearer tokens",
 			input:  "provider returned Authorization: Bearer ya29.secret-oauth-token",
-			secret: "ya29.secret-oauth-token",
+			secrets: []string{"ya29.secret-oauth-token"},
 		},
 		{
 			name:   "Should redact secret shaped environment values",
 			input:  "discovery failed with OPENAI_API_KEY=env-secret-value CLIENT_SECRET=client-secret-value",
-			secret: "env-secret-value",
+			secrets: []string{"env-secret-value", "client-secret-value"},
 		},
 		{
 			name:   "Should redact OAuth token environment values",
 			input:  "extension failed with OAUTH_TOKEN=oauth-secret-value",
-			secret: "oauth-secret-value",
+			secrets: []string{"oauth-secret-value"},
 		},
 	}
@@
 			redacted := RedactString(tc.input)
-			if strings.Contains(redacted, tc.secret) {
-				t.Fatalf("RedactString() = %q, want secret removed", redacted)
+			for _, secret := range tc.secrets {
+				if strings.Contains(redacted, secret) {
+					t.Fatalf("RedactString() = %q, want secret removed: %q", redacted, secret)
+				}
 			}
```
</details>


Also applies to: 43-45

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/redact_test.go` around lines 27 - 30, The test case
"Should redact secret shaped environment values" only asserts one secret; update
the assertions in redact_test.go to verify both secrets in the input string
(e.g., "env-secret-value" and "client-secret-value") are redacted — either by
expanding the expected secrets slice to include both values or by asserting the
output does not contain each secret and that both occurrences are replaced with
the redaction token; make the same change for the other similar case (the test
case around the second multi-secret input) and ensure the test invokes the same
redaction function (e.g., Redact or redactString) for both checks.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
