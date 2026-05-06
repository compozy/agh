---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/config/bootstrap.go
line: 61
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isy,comment:PRRC_kwDOR5y4QM6-UFV9
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Migrate the previous bootstrap-managed dream agent too.**

This only rewrites empty and legacy `"claude"` values. Existing installs that already have `memory.dream.agent = DefaultAgentName` from the earlier bootstrap path will skip the migration and stay on the old general agent.

 

<details>
<summary>Suggested fix</summary>

```diff
-	dreamAgent := ""
-	if strings.TrimSpace(current.Memory.Dream.Agent) == "" ||
-		strings.TrimSpace(current.Memory.Dream.Agent) == legacyDreamAgentName {
+	dreamAgent := ""
+	currentDreamAgent := strings.TrimSpace(current.Memory.Dream.Agent)
+	if currentDreamAgent == "" ||
+		currentDreamAgent == legacyDreamAgentName ||
+		currentDreamAgent == DefaultAgentName {
 		dreamAgent = DefaultMemoryDreamAgentName
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	dreamAgent := ""
	currentDreamAgent := strings.TrimSpace(current.Memory.Dream.Agent)
	if currentDreamAgent == "" ||
		currentDreamAgent == legacyDreamAgentName ||
		currentDreamAgent == DefaultAgentName {
		dreamAgent = DefaultMemoryDreamAgentName
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/config/bootstrap.go` around lines 57 - 61, The migration only
handles empty or legacy ("claude") values; extend the condition that sets
dreamAgent so it also treats the old bootstrap-managed agent value as needing
migration: when current.Memory.Dream.Agent is empty, equals
legacyDreamAgentName, or equals the previous bootstrap agent constant (e.g.
DefaultAgentName), set dreamAgent = DefaultMemoryDreamAgentName; update the
conditional around current.Memory.Dream.Agent/dreamAgent in bootstrap.go
accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/config/bootstrap.go` still migrates the dream agent only when the current value is blank or the legacy `"claude"` alias.
  - That misses the previous bootstrap-managed `DefaultAgentName` value, so already-bootstrapped installs can stay pinned to the old general agent instead of moving to `DefaultMemoryDreamAgentName`.
  - Fix approach: treat `DefaultAgentName` as another migration source and add a focused regression test in `internal/config/bootstrap_test.go` outside the scoped file list because the valid fix needs direct bootstrap coverage.
