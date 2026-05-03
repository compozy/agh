---
provider: coderabbit
pr: "90"
round: 2
round_created_at: 2026-05-03T03:57:53.330715Z
status: resolved
file: internal/network/validate.go
line: 407
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KWjn,comment:PRRC_kwDOR5y4QM69Zj0P
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Raw secret material can bypass checks when embedded in JSON keys.**

Current recursion only applies redaction-based secret detection to string **values**. A key like `agh_claim_...` can pass if its value is innocuous, which still leaks raw secret material over the wire.

 

<details>
<summary>Suggested fix</summary>

```diff
 func envelopeValueContainsSecret(key string, value any) bool {
+	if envelopeStringContainsSecret(key) {
+		return true
+	}
 	if envelopeKeyCarriesRawSecret(key) && envelopeValueIsNonEmpty(value) {
 		return true
 	}
```
</details>

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory; use hash forms (`claim_token_hash`) over the wire".


Also applies to: 421-424, 452-476

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/validate.go` around lines 404 - 407, The current
envelopeValueContainsSecret function only flags secrets when the value is a
non-empty string, letting raw secret material hide in JSON keys; update
envelopeValueContainsSecret to also check the key itself using
envelopeKeyCarriesRawSecret and treat a matching key as a secret regardless of
the value type, and additionally ensure any recursion/redaction logic (used
elsewhere in the file around envelopeKeyCarriesRawSecret and
envelopeValueIsNonEmpty) is applied to map/object keys as well so that keys like
"agh_claim_*" immediately return true; locate and modify
envelopeValueContainsSecret, envelopeKeyCarriesRawSecret, and any recursive
redaction helpers to enforce key-based detection consistently.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `envelopeValueContainsSecret()` currently only treats a raw-secret key as dangerous when the associated value is non-empty, and the recursive detection relies on value inspection rather than key inspection. A payload like `{"agh_claim_...":""}` or other raw-secret material embedded in JSON keys can therefore slip past the current validation.
- Fix approach: make key inspection authoritative regardless of the value shape, keep the recursive traversal checking nested keys, and add regression coverage for raw secret material encoded in object keys.
- Resolution: nested key names are now scanned for raw secret material before value inspection, and the validation tests cover secret-bearing JSON keys directly.
- Verification: `go test ./extensions/bridges/teams ./internal/network -count=1 -race`, `make verify`.
