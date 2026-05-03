---
provider: coderabbit
pr: "90"
round: 2
round_created_at: 2026-05-03T03:57:53.330715Z
status: resolved
file: internal/network/validate.go
line: 435
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KWjo,comment:PRRC_kwDOR5y4QM69Zj0Q
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Whitespace-only strings are incorrectly flagged as secrets.**

At **Line 434**, `diagnostics.Redact(value) != value` treats `"   "` as secret because redaction trims blank text to `""`. This rejects valid payloads with optional whitespace-only fields.

 

<details>
<summary>Suggested fix</summary>

```diff
 func envelopeStringContainsSecret(value string) bool {
+	if strings.TrimSpace(value) == "" {
+		return false
+	}
 	return taskpkg.RedactClaimTokens(value) != value || diagnostics.Redact(value) != value
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/validate.go` around lines 433 - 435, The function
envelopeStringContainsSecret incorrectly flags whitespace-only strings as
secrets because diagnostics.Redact trims blanks; update
envelopeStringContainsSecret to early-return false for strings that are only
whitespace (e.g., if strings.TrimSpace(value) == "") before calling
taskpkg.RedactClaimTokens or diagnostics.Redact so whitespace-only optional
fields are not treated as secrets.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `envelopeStringContainsSecret()` still compares `diagnostics.Redact(value)` to the original string without first excluding whitespace-only input. Because the redactor collapses blank strings, optional whitespace-only fields are currently misclassified as secret-bearing values.
- Fix approach: short-circuit whitespace-only strings before any redaction checks and cover that behavior alongside the key-based secret-detection fixes.
- Resolution: whitespace-only strings now bypass the secret detectors, and the network normalization tests verify that optional blank fields remain valid.
- Verification: `go test ./extensions/bridges/teams ./internal/network -count=1 -race`, `make verify`.
