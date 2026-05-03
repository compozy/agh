---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: internal/network/validate.go
line: 382
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KSGW,comment:PRRC_kwDOR5y4QM69ZeEy
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Scan `env.Proof` for raw secrets too.**

This guard only covers `Body` and `Ext`. `Proof` is still serialized over the wire, so raw `agh_claim_*`, OAuth codes, PKCE verifiers, or secret bindings can bypass the new check by being placed there instead.

 

<details>
<summary>Suggested fix</summary>

```diff
 func validateEnvelopeContainsNoRawSecrets(env Envelope) error {
 	if envelopeRawValueContainsSecret(env.Body) {
 		return fmt.Errorf("%w: raw secret material is not allowed in network body", ErrInvalidBody)
 	}
+	if env.Proof != nil {
+		for _, raw := range *env.Proof {
+			if envelopeRawValueContainsSecret(raw) {
+				return fmt.Errorf("%w: raw secret material is not allowed in network proof", ErrInvalidBody)
+			}
+		}
+	}
 	for _, raw := range env.Ext {
 		if envelopeRawValueContainsSecret(raw) {
 			return fmt.Errorf("%w: raw secret material is not allowed in network ext", ErrInvalidBody)
 		}
 	}
 	return nil
 }
```
</details>

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory; use hash forms (`claim_token_hash`) over the wire".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func validateEnvelopeContainsNoRawSecrets(env Envelope) error {
	if envelopeRawValueContainsSecret(env.Body) {
		return fmt.Errorf("%w: raw secret material is not allowed in network body", ErrInvalidBody)
	}
	if env.Proof != nil {
		for _, raw := range *env.Proof {
			if envelopeRawValueContainsSecret(raw) {
				return fmt.Errorf("%w: raw secret material is not allowed in network proof", ErrInvalidBody)
			}
		}
	}
	for _, raw := range env.Ext {
		if envelopeRawValueContainsSecret(raw) {
			return fmt.Errorf("%w: raw secret material is not allowed in network ext", ErrInvalidBody)
		}
	}
	return nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/validate.go` around lines 373 - 382, The validation
currently checks only Envelope.Body and Envelope.Ext for raw secrets; extend
validateEnvelopeContainsNoRawSecrets to also inspect Envelope.Proof using the
same helper envelopeRawValueContainsSecret and return a wrapped ErrInvalidBody
error when a raw secret is found. Specifically, add a check for env.Proof (and
iterate if Proof is a collection similar to Ext) and produce the same error
message pattern but reference "network proof" (e.g., "%w: raw secret material is
not allowed in network proof") so raw claim_token/OAuth/PKCE/secret-binding
values cannot be transported via Proof.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `validateEnvelopeContainsNoRawSecrets` checks `Body` and `Ext` but not `Proof`, even though `Proof` is serialized over the wire and can carry the same raw secret material.
- Fix plan: extend the validation to inspect `Proof` and add a regression test in `internal/network/validate_test.go`.
