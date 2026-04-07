---
status: resolved
file: internal/cli/skill.go
line: 293
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoB6,comment:PRRC_kwDOR5y4QM61T6HM
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Resolve the registered workspace instead of reconstructing it from `${cwd}/.agh/skills`.**

`cliResolvedWorkspace` only looks at the current root's local skills and returns no `AdditionalDirs` or merged workspace config. In a multi-root workspace, `skill list/view/info` will miss skills coming from `add_dirs` and can diverge from the daemon/API behavior introduced in this PR. Please reuse the shared workspace resolver (or workspace API) to build the `ResolvedWorkspace` passed to `registry.ForWorkspace`.



Also applies to: 305-345

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill.go` around lines 288 - 293, The code uses
cliResolvedWorkspace to build a ResolvedWorkspace before calling
registry.ForWorkspace, which only inspects the root's local skills and omits
AdditionalDirs/merged config; replace that call with the shared workspace
resolver or workspace API so the ResolvedWorkspace passed to
registry.ForWorkspace includes add_dirs and merged workspace config (i.e.,
obtain the ResolvedWorkspace from the shared resolver instead of
cliResolvedWorkspace), and apply the same change for the other occurrences
around the skill list/view/info logic (the block at ~305-345) to ensure
behaviour matches the daemon/API.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `loadSkillCommandContext` currently builds its workspace view with `cliResolvedWorkspace(workspace)`, which only scans `<cwd>/.agh/skills`.
  - Registered workspace metadata, especially `AdditionalDirs`, is therefore ignored even though the shared workspace resolver already knows about those skill roots.
  - I will resolve the current workspace through the shared resolver when a registered workspace exists, while keeping the existing local fallback for unregistered workspaces.
  - Test coverage for this requires touching `internal/cli/skill_test.go`, which is outside the listed batch files but is the minimal place to validate the behavior safely.
