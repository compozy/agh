---
status: resolved
file: internal/hooks/matcher.go
line: 191
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrXy,comment:PRRC_kwDOR5y4QM62twdP
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Handle unknown hook modes explicitly**

At Line 186, the `switch` has no `default`; an unexpected mode is silently dropped and never dispatched. Please add an explicit fallback (or validation/error path) so this cannot fail quietly.



<details>
<summary>Proposed defensive fix</summary>

```diff
 		switch hook.Mode {
 		case HookModeAsync:
 			asyncHooks = append(asyncHooks, hook)
 		case HookModeSync:
 			syncHooks = append(syncHooks, hook)
+		default:
+			// Defensive fallback to avoid silent drops for unexpected mode values.
+			syncHooks = append(syncHooks, hook)
 		}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		switch hook.Mode {
		case HookModeAsync:
			asyncHooks = append(asyncHooks, hook)
		case HookModeSync:
			syncHooks = append(syncHooks, hook)
		default:
			// Defensive fallback to avoid silent drops for unexpected mode values.
			syncHooks = append(syncHooks, hook)
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/hooks/matcher.go` around lines 171 - 191, selectMatchingHooks
currently drops hooks with unknown hook.Mode silently; add an explicit default
branch in the switch in selectMatchingHooks that handles unexpected modes: log a
clear warning (including hook identity and hook.Mode) and use a safe fallback
(e.g., append the hook to syncHooks) so the hook is not lost; reference the
switch on hook.Mode and constants HookModeAsync and HookModeSync when
implementing the default branch.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Reasoning: Unknown hook modes are already rejected before they reach `selectMatchingHooks`. `HookMode.Validate`, `RegisteredHook.Validate`, and the normalization path enforce `sync` or `async`, so adding a default branch here would mask an invariant violation rather than fixing a reachable silent-drop bug.
- Fix approach: No code change. Keep the matcher strict and rely on the existing validation boundary to reject invalid modes.
