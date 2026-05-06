---
provider: coderabbit
pr: "106"
round: 3
round_created_at: 2026-05-06T06:28:14.497092Z
status: resolved
file: internal/bridges/task_notifier.go
line: 548
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3hZl,comment:PRRC_kwDOR5y4QM6-V-LZ
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Redact cursor diagnostics before persisting them.**

`cause.Error()` is written into `Cursor.LastError` verbatim. Any bridge/provider error that includes a claim token or auth material will end up stored and later exposed through cursor/status surfaces.

 
<details>
<summary>🛡️ Minimal redaction fix</summary>

```diff
 func (n *TerminalTaskNotifier) recordCursorError(
 	ctx context.Context,
 	key notifications.CursorKey,
 	cause error,
 ) error {
+	lastError := taskpkg.RedactClaimTokens(cause.Error())
 	if _, err := n.cursors.RecordError(ctx, notifications.CursorError{
 		Key:       key,
-		LastError: truncateTerminalTaskCursorError(cause.Error()),
+		LastError: truncateTerminalTaskCursorError(lastError),
 		Now:       n.now(),
 	}); err != nil {
 		return fmt.Errorf("bridges: record terminal task notification cursor error: %w", err)
 	}
```
</details>

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings MUST NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory — use hash forms (`claim_token_hash`) over the wire"

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bridges/task_notifier.go` around lines 539 - 548,
TerminalTaskNotifier.recordCursorError currently persists cause.Error() verbatim
into CursorError.LastError; update this to redact sensitive tokens before
storing by passing a sanitized string instead of cause.Error() to
truncateTerminalTaskCursorError. Implement or call a redaction helper (e.g.,
redactSensitiveTokens or similar) that detects and replaces raw claim tokens
(agh_claim_*), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings
with hashed/placeholder forms (claim_token_hash or similar) and then feed the
result into truncateTerminalTaskCursorError when constructing
notifications.CursorError. Ensure the new redaction helper is used wherever
CursorError.LastError is set so no raw secrets are persisted.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause analysis: `recordCursorError` truncates `cause.Error()` directly and persists it as `Cursor.LastError`; there is no diagnostics redaction step before storage.
- Why this is valid: bridge/provider failures can carry raw claim tokens or other credential-shaped fields, and cursor diagnostics are durable state later exposed through status surfaces.
- Fix approach: sanitize error text before truncation/persistence in `internal/bridges/task_notifier.go` and add a regression in `internal/bridges/task_notifier_test.go` that proves cursor diagnostics never retain raw secrets.

## Resolution

- Redacted cursor diagnostics before truncation/persistence in `internal/bridges/task_notifier.go`.
- Added a cursor-regression test in `internal/bridges/task_notifier_test.go` covering claim-token and auth-material redaction.

## Verification

- Focused regression: `go test ./internal/bridges -run 'TestTerminalTaskNotifierDeliverDue|TestTruncateTerminalTaskCursorError' -count=1 -race`
- Fresh full gate: `make verify` exited `0` in this session.
