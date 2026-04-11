---
status: resolved
file: internal/automation/manager.go
line: 1705
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0T,comment:PRRC_kwDOR5y4QM623e7f
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's check the exact file and lines mentioned
wc -l internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 90

---

🏁 Script executed:

```shell
# Read the specific lines mentioned in the review (1682-1705)
sed -n '1682,1705p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 673

---

🏁 Script executed:

```shell
# Also check lines around to understand context better
sed -n '1670,1720p' internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 1430

---

🏁 Script executed:

```shell
# Search for triggerSessionObserver to understand the error handling pattern being referenced
rg -n "triggerSessionObserver" internal/automation/manager.go -A 10
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if Manager has a logger field
rg -n "type Manager struct" internal/automation/manager.go -A 20
```

Repository: compozy/agh

Length of output: 778

---

🏁 Script executed:

```shell
# Verify the current imports and slog usage
head -50 internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 1730

---

🏁 Script executed:

```shell
# Search for triggerSessionObserver method
rg -n "func.*triggerSessionObserver" internal/automation/manager.go -A 15
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Also search for similar error logging patterns in the file
rg -n "m.logger" internal/automation/manager.go | head -20
```

Repository: compozy/agh

Length of output: 297

---

🏁 Script executed:

```shell
# Check what FireSessionCreated and FireSessionStopped return
rg -n "FireSessionCreated|FireSessionStopped" internal/automation/manager.go
```

Repository: compozy/agh

Length of output: 166

---

🏁 Script executed:

```shell
# Look for the TriggerEngine interface to understand return types
rg -n "FireSessionCreated|FireSessionStopped" internal/automation/ -A 2 -B 2
```

Repository: compozy/agh

Length of output: 2409

---



</details>

**Handle session-trigger dispatch errors instead of suppressing them.**

The `fireSessionCreated` and `fireSessionStopped` methods suppress errors from `FireSessionCreated` and `FireSessionStopped` with `_, _`, violating the coding guideline "Never ignore errors with `_`". Session-based automations can fail with no log trail. Add error handling with logging consistent with the pattern already used in `trigger.go`:

<details>
<summary>Suggested fix</summary>

```diff
 func (m *Manager) fireSessionCreated(ctx context.Context, sess *session.Session) {
 	if sess == nil {
 		return
 	}
 	engine, runtimeCtx, ok := m.triggerRuntime()
 	if !ok {
 		return
 	}
 	mergedCtx, cancel := mergedRuntimeContext(ctx, runtimeCtx)
 	defer cancel()
-	_, _ = engine.FireSessionCreated(mergedCtx, sess)
+	if _, err := engine.FireSessionCreated(mergedCtx, sess); err != nil {
+		m.logger.Warn("automation.manager.session_created_trigger_failed", "error", err)
+	}
 }
 
 func (m *Manager) fireSessionStopped(ctx context.Context, sess *session.Session) {
 	if sess == nil {
 		return
 	}
 	engine, runtimeCtx, ok := m.triggerRuntime()
 	if !ok {
 		return
 	}
 	mergedCtx, cancel := mergedRuntimeContext(ctx, runtimeCtx)
 	defer cancel()
-	_, _ = engine.FireSessionStopped(mergedCtx, sess)
+	if _, err := engine.FireSessionStopped(mergedCtx, sess); err != nil {
+		m.logger.Warn("automation.manager.session_stopped_trigger_failed", "error", err)
+	}
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/manager.go` around lines 1682 - 1705, The session trigger
calls currently ignore errors from engine.FireSessionCreated and
engine.FireSessionStopped; update fireSessionCreated and fireSessionStopped to
capture the returned error, check if err != nil, and log the failure (including
the error and relevant session info) using the Manager's logger consistent with
trigger.go's pattern; keep the existing triggerRuntime and mergedRuntimeContext
usage and defer cancel, but replace "_, _ = ..." with "if _, err :=
engine.FireSessionCreated(...); err != nil { m.log.Errorf(..., err) }" (and
similarly for FireSessionStopped), referencing the functions fireSessionCreated,
fireSessionStopped, triggerRuntime, mergedRuntimeContext, and the
engine.FireSessionCreated/FireSessionStopped calls.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `fireSessionCreated` and `fireSessionStopped` currently ignore errors returned by the trigger engine, so failed session-based automations leave no operator signal. I will log those failures with the manager logger and session identifiers, matching the observer-side error handling already present in `trigger.go`.
- Notes:
