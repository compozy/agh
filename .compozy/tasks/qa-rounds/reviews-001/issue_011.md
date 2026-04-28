---
status: resolved
file: internal/cli/skill_commands.go
line: 130
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-IGMV,comment:PRRC_kwDOR5y4QM67_zdM
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Normalize skill name before daemon/local dispatch.**

In workspace mode, raw `name` is passed through; in local mode it gets trimmed by `findSkillByName`. This creates inconsistent behavior for inputs with surrounding whitespace.


<details>
<summary>Suggested fix</summary>

```diff
 func runSkillViewCommand(cmd *cobra.Command, deps commandDeps, name string, filePath string) error {
+	name = strings.TrimSpace(name)
+	if name == "" {
+		return errors.New("skill name is required")
+	}
  workspaceRef, err := skillWorkspaceFlag(cmd)
  if err != nil {
    return err
  }
  if workspaceRef != "" {
    return runDaemonSkillViewCommand(cmd, deps, name, filePath, workspaceRef)
  }
  return runLocalSkillViewCommand(cmd, deps, name, filePath)
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_commands.go` around lines 121 - 130, Trim/normalize the
incoming skill name before deciding daemon vs local handling so both paths see
the same value: in runSkillViewCommand call the same normalization used by
findSkillByName (e.g., strings.TrimSpace(name) or the project’s canonical
normalize function) and then pass the normalized name into
runDaemonSkillViewCommand and runLocalSkillViewCommand; update references to the
local variable (name) in runSkillViewCommand so both branches use the normalized
skill name.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `runSkillViewCommand` passes the raw CLI argument to daemon mode, while local mode trims inside `findSkillByName`. A padded skill name therefore behaves differently depending on `--workspace`. Fix by trimming and validating the name once before choosing daemon or local execution.

## Resolution

- Normalized and validated the skill name once before dispatching local or daemon skill view mode.
- Added daemon CLI coverage for padded skill names.
- Verified through targeted CLI tests and `make verify`.
