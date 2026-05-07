---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/agentidentity/errors.go
line: 76
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRsI,comment:PRRC_kwDOR5y4QM6-67EK
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Do not surface raw error text in machine-readable CLI payloads.**

Line [74] currently mirrors arbitrary `err.Error()` into `payload.Message`; that can expose secrets in error payloads. Default to a safe generic message, and only use curated text from controlled identity errors.

 
<details>
<summary>Safer default payload message</summary>

```diff
 	payload := ErrorPayload{
 		Code:     "agent_error",
-		Message:  strings.TrimSpace(errorString(err)),
+		Message:  agentCommandFailedMessage,
 		Action:   "inspect the daemon error and retry",
 		ExitCode: ExitCodeForError(err),
 	}
```
</details>
As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings MUST NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory."


Also applies to: 79-83

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/agentidentity/errors.go` around lines 72 - 76, The payload currently
assigns raw errorString(err) into ErrorPayload.Message (see ErrorPayload,
payload.Message and errorString(err)); change this to a safe generic message
like "internal agent error" for all unknown errors and only populate Message
with curated, non-secret text for known/controlled identity error types (e.g.,
match specific error types or sentinel errors before calling ExitCodeForError);
ensure you do not mirror err.Error() into the payload and apply the same change
to the other payload creation sites noted (around the block referencing
ExitCodeForError and the later 79-83 area).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  `ErrorPayloadFor` currently falls back to `err.Error()` for arbitrary failures, which can leak secrets into machine-readable CLI output. The safe default is the generic command-failed message, with curated text only for controlled `*agentidentity.Error` payloads.
