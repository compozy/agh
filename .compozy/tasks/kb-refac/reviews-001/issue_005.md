---
status: resolved
file: internal/cli/skill_commands.go
line: 54
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrWv,comment:PRRC_kwDOR5y4QM62twb8
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**`--source` help text is missing supported values.**

`normalizeSkillSourceFilter` also accepts `marketplace` (plus the `agents` aliases), but the flag description only advertises bundled/user/additional/workspace. The CLI help should match the parser.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_commands.go` at line 54, The --source flag help currently
lists only bundled/user/additional/workspace but normalizeSkillSourceFilter also
accepts marketplace and the agents aliases; update the flag registration (where
cmd.Flags().StringVar(&sourceFilter, "source", ... ) is called) to include all
supported values (e.g., bundled, user, additional, workspace, marketplace,
agents) in the help string so the CLI help matches the
normalizeSkillSourceFilter parser and its accepted aliases.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: The `--source` help text is incomplete. The parser accepts `marketplace` and the `agents` aliases, but the command help omits them, which makes the CLI contract misleading.
- Fix approach: Update the flag help string to advertise the full supported filter vocabulary.
