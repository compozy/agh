---
status: resolved
file: internal/cli/skill_marketplace.go
line: 145
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrW7,comment:PRRC_kwDOR5y4QM62twcP
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use a stable install path during updates.**

`installMarketplaceSkill` always derives `targetDir` from the archive's current `Meta.Name`, and `updateMarketplaceSkill` reuses it for replacements. If the registry package renames the skill, the update will install into a new directory and leave the old install behind instead of replacing it. Updates should either reuse `installed.Dir` or fail when the package name changes.




Also applies to: 234-245

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_marketplace.go` around lines 128 - 145,
installMarketplaceSkill (and updateMarketplaceSkill) compute targetDir from
parsedSkill.Meta.Name which allows a package rename to create a new directory
instead of replacing the existing install; change the logic to prefer the
existing installation directory when updating: in updateMarketplaceSkill, when
an installed record (installed.Dir or installed.Path) exists, reuse that path as
targetDir (or fail if the new package explicitly intends a rename), and only
fall back to deriving a path from parsedSkill.Meta.Name when there is no
existing install; update the call sites around
moveInstalledSkillDir(parsedSkill.Dir, targetDir, replaceExisting) accordingly
so replacements operate on the stable installed.Dir rather than the archive
name.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: Updates currently derive the replacement directory from the newly downloaded package metadata. If the marketplace package name changes, an update can install into a new directory and leave the old installation behind instead of replacing it.
- Fix approach: Reuse the existing installed directory for updates, validating that the target remains inside the user skills root, and keep name-derived paths only for fresh installs.
