---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/modelcatalog/redact.go
line: 16
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaqT,comment:PRRC_kwDOR5y4QM6-7HYv
---

# Issue 019: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Redaction rule misses common `key: value` secret forms.**

Current keyed-secret matching/preservation is `=`-only, so colon-delimited secret pairs are not handled equivalently and can leak sensitive values in error text.

 

<details>
<summary>Suggested fix</summary>

```diff
 	regexp.MustCompile(
-		`(?i)\b([A-Z0-9_-]*(?:api[_-]?key|auth[_-]?token|oauth[_-]?token|access[_-]?token|refresh[_-]?token|id[_-]?token|secret|password|credential|private[_-]?key)[A-Z0-9_-]*)=([^&\s]+)`,
+		`(?i)\b([A-Z0-9_-]*(?:api[_-]?key|auth[_-]?token|oauth[_-]?token|access[_-]?token|refresh[_-]?token|id[_-]?token|secret|password|credential|private[_-]?key)[A-Z0-9_-]*)\s*[:=]\s*([^&\s]+)`,
 	),
 }
@@
 func redactMatch(value string) string {
-	if key, _, ok := strings.Cut(value, "="); ok {
-		return key + "=[REDACTED]"
+	if idx := strings.IndexAny(value, "=:"); idx > 0 {
+		key := strings.TrimSpace(value[:idx])
+		sep := string(value[idx])
+		return key + sep + "[REDACTED]"
 	}
 	return "[REDACTED]"
 }
```
</details>

As per coding guidelines: `internal/**/*.go`: "`claim_token` redaction is non-negotiable ... secret bindings MUST NEVER appear in logs, status APIs, settings views, error payloads..."


Also applies to: 28-31

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/redact.go` around lines 15 - 16, The current redaction
regex in redact.go only matches key=value pairs and misses key: value forms;
update the pattern used (the regex literal that currently contains
api[_-]?key|auth[_-]?token|... followed by =([^&\s]+)) to also accept colon
separators and optional whitespace (e.g., match `:` or `=` with possible spaces)
so keyed secrets like `api_key: secret` are redacted; apply the same change to
the sibling regex entries in the same file that handle keyed secrets (the other
patterns around the listed token/secret names) so all colon-delimited forms are
covered.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/modelcatalog/redact.go` only redacts keyed secrets in `key=value` form.
  - Common `key: value` variants are currently missed, which weakens the non-negotiable secret-redaction boundary.
  - Fix plan: extend the keyed-secret matcher and replacement logic to cover both `:` and `=` separators, then add regression tests. The test addition will require a minimal out-of-scope file because no scoped test file currently covers this helper.
  - Fixed in `internal/modelcatalog/redact.go` with regression coverage in `internal/modelcatalog/redact_test.go`, then verified with focused package tests plus `make verify`.
