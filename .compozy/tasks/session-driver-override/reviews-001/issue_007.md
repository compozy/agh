---
status: resolved
file: internal/observe/reconcile.go
line: 73
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581azH,comment:PRRC_kwDOR5y4QM66RFOp
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't fail the whole reconcile on one unrepairable legacy session.**

Returning here means a single stale session whose agent/provider can no longer be resolved prevents every other valid session from being indexed. That is much harsher than the unreadable-metadata path above; please log and skip this session (or mark it orphaned) so one broken legacy record does not take down the entire reconciliation pass.

<details>
<summary>Suggested direction</summary>

```diff
 		meta, err = session.RepairLegacyProvider(ctx, metaPath, meta, session.LegacyProviderRepairOptions{
 			Now:               o.now,
 			Logger:            o.logger,
 			WorkspaceResolver: o.workspaceResolver,
 		})
 		if err != nil {
-			return nil, fmt.Errorf(
-				"observe: repair legacy provider for session %q: %w",
-				strings.TrimSpace(meta.ID),
-				err,
-			)
+			o.logger.Warn(
+				"observe: skipping session with unrecoverable legacy provider metadata",
+				"session_id", strings.TrimSpace(meta.ID),
+				"path", metaPath,
+				"error", err,
+			)
+			continue
 		}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		meta, err = session.RepairLegacyProvider(ctx, metaPath, meta, session.LegacyProviderRepairOptions{
			Now:               o.now,
			Logger:            o.logger,
			WorkspaceResolver: o.workspaceResolver,
		})
		if err != nil {
			o.logger.Warn(
				"observe: skipping session with unrecoverable legacy provider metadata",
				"session_id", strings.TrimSpace(meta.ID),
				"path", metaPath,
				"error", err,
			)
			continue
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/reconcile.go` around lines 62 - 73, The current reconcile
code returns an error when session.RepairLegacyProvider(ctx, metaPath, meta,
session.LegacyProviderRepairOptions{Now: o.now, Logger: o.logger,
WorkspaceResolver: o.workspaceResolver}) fails, which aborts the whole pass;
instead, catch the error, log it with context (including meta.ID and metaPath)
via o.logger, optionally mark the session as orphaned/skipped in meta or emit an
event, and continue processing remaining sessions rather than returning the
error so one unrepairable legacy session does not stop the entire reconcile.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `Observer.loadSessionMetadata` aborts the full reconciliation pass when one legacy provider repair cannot be resolved, even though other unreadable metadata paths are already skipped with warnings.
- Fix plan: log the unrecoverable legacy-provider repair failure with session/path context and continue reconciling the remaining sessions.
