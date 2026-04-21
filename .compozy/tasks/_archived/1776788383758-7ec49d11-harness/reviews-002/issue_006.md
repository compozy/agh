---
status: resolved
file: internal/daemon/harness_context.go
line: 254
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUG,comment:PRRC_kwDOR5y4QM65IlPH
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't mark every system session as detached task runtime.**

`resolveDetachedRunMode` only looks at session class, so any system session — including startup flows with no detached run metadata — resolves to `task_runtime` once the feature flag is on. That collapses unrelated system sessions into detached-runtime policy and can turn on detached-only behavior in the wrong places. Gate this on actual detached turn metadata, e.g. `turnCtx.Detached != nil` (and likely only on turn resolution).

<details>
<summary>💡 Suggested change</summary>

```diff
-		DetachedRunMode:  r.resolveDetachedRunMode(sessionCtx),
+		DetachedRunMode:  r.resolveDetachedRunMode(sessionCtx, turnCtx),
...
-func (r *HarnessContextResolver) resolveDetachedRunMode(sessionCtx HarnessSessionContext) DetachedRunMode {
+func (r *HarnessContextResolver) resolveDetachedRunMode(
+	sessionCtx HarnessSessionContext,
+	turnCtx HarnessTurnContext,
+) DetachedRunMode {
 	if !r.runtime.DetachedTaskRuntimeEnabled {
 		return DetachedRunModeNone
 	}
+	if turnCtx.Detached == nil {
+		return DetachedRunModeNone
+	}
 	if sessionCtx.SessionClass == SessionClassSystem {
 		return DetachedRunModeTaskRuntime
 	}
 	return DetachedRunModeNone
 }
```
</details>




Also applies to: 514-521

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_context.go` around lines 247 - 254, The current
construction of ResolvedHarnessPolicy sets DetachedRunMode based solely on
session class via r.resolveDetachedRunMode(sessionCtx), which causes all system
sessions to be treated as detached runtimes; change this to determine
DetachedRunMode from the turn context and only when detached metadata is
present: check turnCtx.Detached != nil (or equivalent) and call
r.resolveDetachedRunMode using the turn-level information (or a new turn-aware
resolver) so DetachedRunMode is only enabled for turns that explicitly carry
detached run metadata; update both the policy creation here and the other
occurrence that mirrors lines 514-521 to follow the same guard.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `resolveDetachedRunMode` keyed only on session class plus the feature flag, so every system-session resolution became `task_runtime` even when the turn had no detached metadata. That is a real policy bug because it enables detached-runtime behavior for unrelated system turns and startup flows. I made detached mode turn-aware, required actual detached metadata before returning `DetachedRunModeTaskRuntime`, and added a scoped regression test. Verified with `go test ./internal/daemon -count=1` and `make verify`.
