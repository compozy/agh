---
status: resolved
file: internal/extension/channel_delivery_notifier.go
line: 59
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLv,comment:PRRC_kwDOR5y4QM623eJJ
---

# Issue 027: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Error ignored without justification.**

`ProjectEvent` error is discarded. This could silently drop delivery projections. Add logging or a justifying comment.


<details>
<summary>🔧 Suggested fix</summary>

```diff
 	if n.broker != nil {
 		if event, ok := payload.(acp.AgentEvent); ok {
-			_ = n.broker.ProjectEvent(ctx, sessionID, projectionEventFromAgentEvent(event))
+			// Projection errors are non-fatal; downstream notifier still receives the event
+			_ = n.broker.ProjectEvent(ctx, sessionID, projectionEventFromAgentEvent(event))
 		}
 	}
```
</details>

As per coding guidelines: "Never ignore errors with `_` — every error must be handled or have a written justification."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if n.broker != nil {
		if event, ok := payload.(acp.AgentEvent); ok {
			// Projection errors are non-fatal; downstream notifier still receives the event
			_ = n.broker.ProjectEvent(ctx, sessionID, projectionEventFromAgentEvent(event))
		}
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/channel_delivery_notifier.go` around lines 56 - 59, The
call to n.broker.ProjectEvent currently swallows the error ( `_ =
n.broker.ProjectEvent(...)` ); change it to capture and handle the error from
n.broker.ProjectEvent(ctx, sessionID, projectionEventFromAgentEvent(event)) —
e.g. `if err := n.broker.ProjectEvent(...); err != nil {
n.logger.Errorf("ProjectEvent failed for session %s: %v", sessionID, err) }` —
or if logging isn’t available add a one-line justified comment explaining why
ignoring the error is safe; ensure the fix references n.broker.ProjectEvent and
projectionEventFromAgentEvent(event) so delivery projection failures are not
silently dropped.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Why: `OnAgentEvent` silently discards `ProjectEvent` errors with `_ =`. The notifier intentionally treats broker projection as best-effort, but that needs to be explicit rather than silently ignored.
- Root cause: The projection path has no error return channel, so failures must be documented at the call site instead of hidden.
- Fix plan: Replace the silent discard with an explicit error branch and justification comment while still forwarding the event downstream.
- Resolution: Broker projection failures are now logged explicitly via `slog` while events still flow downstream, and verification is green.
