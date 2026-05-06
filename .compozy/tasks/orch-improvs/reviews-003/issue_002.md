---
provider: coderabbit
pr: "106"
round: 3
round_created_at: 2026-05-06T06:28:14.497092Z
status: resolved
file: internal/bridges/task_notifier.go
line: 435
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3hZh,comment:PRRC_kwDOR5y4QM6-V-LR
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Redact or allow-list `Payload` before sending bridge metadata.**

`notification.Error` is scrubbed, but `record.Event.Payload` is copied straight into `notification.Payload` and then marshaled into `ProviderMetadata`. That means any secret-bearing event fields will cross the bridge unchanged.

 
As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings MUST NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory — use hash forms (`claim_token_hash`) over the wire"


Also applies to: 501-515

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bridges/task_notifier.go` around lines 422 - 435, The code assigns
record.Event.Payload directly into TerminalTaskNotification.Payload (in the
terminalTaskNotificationResolution returned by
terminalTaskNotificationDeliveryID / the TerminalTaskNotification construction),
which allows secret fields to cross the bridge; instead sanitize the payload
before assignment by applying the same redaction logic used for Error (e.g.,
call the existing taskpkg redaction helper or a new JSON-redaction utility) to
cloneRawJSON(record.Event.Payload) and/or whitelist allowed keys, then set
Notification.Payload to that redacted/allow-listed JSON; update both occurrences
(the block returning terminalTaskNotificationResolution and the similar block
around lines ~501-515) to use the redacted payload variable.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause analysis: `terminalTaskNotificationForRecord` copies `record.Event.Payload` straight into `TerminalTaskNotification.Payload`, and that payload is later marshaled into `ProviderMetadata` without any key/value scrubbing.
- Why this is valid: `orch-improvs` already established secret-redaction guarantees for task context payloads, but the bridge notification path still bypasses them and can leak claim tokens, auth tokens, OAuth/PKCE material, or secret bindings to adapter surfaces.
- Fix approach: add a scoped payload sanitizer in `internal/bridges/task_notifier.go`, use it before assigning `notification.Payload`, and add regression coverage in `internal/bridges/task_notifier_test.go` even though that adjacent test file is outside the explicit production-file list.

## Resolution

- Added terminal notification payload sanitization in `internal/bridges/task_notifier.go` before provider metadata is emitted.
- Added regression coverage in `internal/bridges/task_notifier_test.go` proving secret-bearing payload fields are redacted before delivery.

## Verification

- Focused regression: `go test ./internal/bridges -run 'TestTerminalTaskNotifierDeliverDue|TestTruncateTerminalTaskCursorError' -count=1 -race`
- Fresh full gate: `make verify` exited `0` in this session.
