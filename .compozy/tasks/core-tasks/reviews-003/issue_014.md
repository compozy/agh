---
status: resolved
file: internal/network/audit.go
line: 149
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM565HzW,comment:PRRC_kwDOR5y4QM63qGau
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Guard zero-value writer paths to avoid panic and silent audit drops.**

At Line 136, `w.now()` can panic if `FileAuditWriter` is zero-initialized, and when both sinks are unset (`w.path == "" && w.store == nil`) this method currently returns `nil` without recording anything.



<details>
<summary>🔧 Proposed hardening</summary>

```diff
 func (w *FileAuditWriter) RecordTaskIngress(ctx context.Context, audit TaskIngressAudit) error {
-	if ctx == nil {
-		return errors.New("network: audit context is required")
-	}
 	if w == nil {
 		return errors.New("network: audit writer is required")
 	}
+	if ctx == nil {
+		return errors.New("network: audit context is required")
+	}
+	if w.path == "" && w.store == nil {
+		return errors.New("network: audit sink is required")
+	}
+
+	now := w.now
+	if now == nil {
+		now = func() time.Time { return time.Now().UTC() }
+	}
 
-	entry, err := normalizeTaskIngressAuditEntry(audit, w.now())
+	entry, err := normalizeTaskIngressAuditEntry(audit, now())
 	if err != nil {
 		return err
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (w *FileAuditWriter) RecordTaskIngress(ctx context.Context, audit TaskIngressAudit) error {
	if w == nil {
		return errors.New("network: audit writer is required")
	}
	if ctx == nil {
		return errors.New("network: audit context is required")
	}
	if w.path == "" && w.store == nil {
		return errors.New("network: audit sink is required")
	}

	now := w.now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	entry, err := normalizeTaskIngressAuditEntry(audit, now())
	if err != nil {
		return err
	}

	var recordErr error
	if w.path != "" {
		recordErr = errors.Join(recordErr, w.appendFile(entry))
	}
	if w.store != nil {
		recordErr = errors.Join(recordErr, w.store.WriteNetworkAudit(ctx, entry))
	}

	return recordErr
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/audit.go` around lines 128 - 149, The RecordTaskIngress
method calls w.now() before verifying the writer's sinks and before guarding
against a nil now func, which can panic on a zero-value FileAuditWriter and also
silently return when no sinks are configured; update RecordTaskIngress (in
FileAuditWriter) to first ensure at least one sink is set (w.path != "" ||
w.store != nil) and return a descriptive error if none are configured, then
compute the timestamp using a safe now: use w.now() only if w.now != nil
otherwise fall back to time.Now(), then pass that timestamp to
normalizeTaskIngressAuditEntry; keep conditional calls to appendFile and
store.WriteNetworkAudit unchanged but ensure they run after these guards to
avoid panics and silent drops.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `RecordTaskIngress` can panic on a zero-value writer because it calls `w.now()` before guarding `w.now`, and it silently returns `nil` when no sinks are configured. I will add sink/clock guards plus regression tests for zero-value and sinkless writer paths.
  Resolution: Hardened `RecordTaskIngress` with nil-writer, sink-required, and nil-clock guards, plus regression tests for sinkless writers and fallback timestamp behavior.
