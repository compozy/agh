---
provider: coderabbit
pr: "88"
round: 2
round_created_at: 2026-05-02T22:54:45.308545Z
status: resolved
file: internal/api/core/automation.go
line: 1016
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_IrdM,comment:PRRC_kwDOR5y4QM69XbzL
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Keep webhook secret values byte-for-byte.**

These helpers trim the secret before handing it to the manager. That mutates valid secrets and can make HMAC verification fail even though the caller supplied the right value.

 

<details>
<summary>Suggested fix</summary>

```diff
 func webhookSecretWriteFromCreateRequest(req contract.CreateTriggerRequest) automationpkg.WebhookSecretWrite {
 	write := automationpkg.WebhookSecretWrite{Ref: strings.TrimSpace(req.WebhookSecretRef)}
-	if strings.TrimSpace(req.WebhookSecretValue) != "" {
-		value := strings.TrimSpace(req.WebhookSecretValue)
+	if req.WebhookSecretValue != "" {
+		value := req.WebhookSecretValue
 		write.Value = &value
 	}
 	return write
 }
@@
 	if req.WebhookSecretValue != nil {
-		value := strings.TrimSpace(*req.WebhookSecretValue)
+		value := *req.WebhookSecretValue
 		write.Value = &value
 	}
 	return &write
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/automation.go` around lines 1003 - 1024, The helpers
webhookSecretWriteFromCreateRequest and webhookSecretWriteFromUpdateRequest
currently call strings.TrimSpace on webhook secret values which can alter valid
secrets; change them to preserve the secret byte-for-byte by removing TrimSpace
when assigning WebhookSecretValue (keep the pointer semantics and allocate value
variables as before), while still trimming only the WebhookSecretRef (Ref) if
desired; update webhookSecretWriteFromCreateRequest to set write.Value to a
pointer to the raw req.WebhookSecretValue (if non-empty) and
webhookSecretWriteFromUpdateRequest to set write.Value to a pointer to
*req.WebhookSecretValue without trimming so HMAC and exact comparisons remain
correct.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `webhookSecretWriteFromCreateRequest` and `webhookSecretWriteFromUpdateRequest` currently call `strings.TrimSpace` on the secret value before passing it to automation.
  - That mutates valid secrets and breaks exact-match/HMAC semantics; only non-secret routing fields should be normalized.
  - Resolved by removing secret-value trimming in `internal/api/core/automation.go` while preserving the existing normalization of non-secret routing fields.
  - Verification: `make verify` passed with the updated automation conversion path.
