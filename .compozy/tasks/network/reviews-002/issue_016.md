---
status: resolved
file: internal/network/manager.go
line: 949
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56T_fN,comment:PRRC_kwDOR5y4QM624toc
---

# Issue 016: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Guard `m.stats` before recording audit metrics.**

These helpers panic when `m.stats` is nil. The new tests in this PR build partial `Manager` values and call `recordAuditSent`, `recordAuditReceived`, and `recordAuditRejected` directly, so this currently fails before the assertions run.

<details>
<summary>🐛 Proposed fix</summary>

```diff
 func (m *Manager) recordAuditSent(ctx context.Context, sessionID string, envelope Envelope) {
 	if m == nil || m.auditor == nil {
 		return
 	}
 	if err := m.auditor.RecordSent(ctx, sessionID, envelope); err != nil {
 		m.logger.Warn("network.audit.record_sent_failed", "session_id", sessionID, "envelope_id", envelope.ID, "error", err)
 		return
 	}
-	m.stats.recordSent(envelope)
+	if m.stats != nil {
+		m.stats.recordSent(envelope)
+	}
 	m.logger.Info("network.message.sent", networkLogFields(envelope, "session_id", sessionID)...)
 }
 
 func (m *Manager) recordAuditReceived(ctx context.Context, sessionID string, envelope Envelope) {
 	if m == nil || m.auditor == nil {
 		return
 	}
 	if err := m.auditor.RecordReceived(ctx, sessionID, envelope); err != nil {
 		m.logger.Warn("network.audit.record_received_failed", "session_id", sessionID, "envelope_id", envelope.ID, "error", err)
 		return
 	}
-	m.stats.recordReceived(envelope)
+	if m.stats != nil {
+		m.stats.recordReceived(envelope)
+	}
 	m.logger.Info("network.message.received", networkLogFields(envelope, "session_id", sessionID)...)
 }
 
 func (m *Manager) recordAuditRejected(ctx context.Context, sessionID string, envelope Envelope, reason string) {
 	if m == nil || m.auditor == nil {
 		return
 	}
 	if err := m.auditor.RecordRejected(ctx, sessionID, envelope, reason); err != nil {
 		m.logger.Warn("network.audit.record_rejected_failed", "session_id", sessionID, "envelope_id", envelope.ID, "error", err)
 		return
 	}
-	m.stats.recordRejected(envelope)
+	if m.stats != nil {
+		m.stats.recordRejected(envelope)
+	}
 	fields := networkLogFields(envelope, "session_id", sessionID)
 	fields = append(fields, "reason", strings.TrimSpace(reason))
 	m.logger.Info("network.message.rejected", fields...)
 }
```
</details>



Also applies to: 952-963

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/manager.go` around lines 928 - 949, The recordAuditSent,
recordAuditReceived (and the similar recordAuditRejected) helpers call
m.stats.recordSent/recordReceived/recordRejected without checking m.stats and
can panic when m.stats is nil; update these functions (recordAuditSent,
recordAuditReceived, and recordAuditRejected) to guard m.stats == nil before
calling any m.stats.* methods (skip calling
recordSent/recordReceived/recordRejected if m.stats is nil) so the functions no
longer panic when Manager is partially constructed but still perform
logging/auditing as before.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `recordAuditSent`, `recordAuditReceived`, and `recordAuditRejected` call into `m.stats` without guarding nil, so partially constructed `Manager` values panic before the audit writer assertions execute.
- Fix plan: Make the audit helpers tolerate `nil` stats while preserving audit writes and log emission, and keep regression coverage on the helper behavior.
