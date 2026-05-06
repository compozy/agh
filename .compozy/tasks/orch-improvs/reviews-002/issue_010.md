---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/situation/service.go
line: 510
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-d,comment:PRRC_kwDOR5y4QM6-VcC8
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Treat bundle assembly as optional enrichment.**

These two paths now fail the entire session-context build when `bundleForRun` cannot complete. A single bad historic event payload or review-history read will break `ContextForSession` for worker/reviewer sessions even though the task/channel context is still available.

 

<details>
<summary>🩹 Suggested fallback pattern</summary>

```diff
-	bundle, err := s.bundleForRun(ctx, taskRecord, run, workspaceSnapshot, nil)
-	if err != nil {
-		return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
-	}
+	var bundle *taskpkg.ContextBundle
+	built, err := s.bundleForRun(ctx, taskRecord, run, workspaceSnapshot, nil)
+	if err != nil {
+		if isContextError(err) {
+			return contract.AgentTaskContextPayload{}, contract.AgentCoordinationChannelContextPayload{}, "", err
+		}
+	} else {
+		bundle = &built
+	}
@@
-		Bundle:    &bundle,
+		Bundle:    bundle,
```

Apply the same pattern in `reviewBindingTaskAndChannelContext`.
</details>


Also applies to: 529-529, 571-574, 587-587

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/situation/service.go` around lines 507 - 510, The bundle assembly
via s.bundleForRun is currently treated as fatal and returns an error from
ContextForSession/reviewBindingTaskAndChannelContext; change it to optional
enrichment: if s.bundleForRun returns an error, log the error (using the
existing logger on the service, e.g., s.logger or similar) and continue building
the session context with an empty/nil bundle instead of returning the error, so
task/channel context remains available; apply this fallback pattern to the other
occurrences called out (the calls inside reviewBindingTaskAndChannelContext and
the other spots at the indicated contexts) so bundle failures don't abort the
overall session-context build.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `ContextForSession` and `reviewBindingTaskAndChannelContext` treat `bundleForRun` as mandatory, so a non-context enrichment failure can blank out otherwise-available task/channel context.
- Fix approach: Downgrade non-context bundle failures to optional enrichment, keep context-cancellation errors fatal, emit a bounded warning, and add regression coverage in `internal/situation/service_test.go`.
