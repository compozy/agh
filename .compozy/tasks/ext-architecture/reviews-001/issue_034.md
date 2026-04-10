---
status: resolved
file: internal/skills/registry_external.go
line: 31
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAaz,comment:PRRC_kwDOR5y4QM62zltC
---

# Issue 034: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Normalize `Skill.Meta.Name` on the stored clone.**

The map key uses the trimmed name, but the cloned `Skill` still keeps the original whitespace. That makes list/workspace flows inconsistent with direct lookups because downstream code compares `skill.Meta.Name` literally.

<details>
<summary>💡 Proposed fix</summary>

```diff
-		registered[name] = cloneSkill(skill)
+		cloned := cloneSkill(skill)
+		cloned.Meta.Name = name
+		registered[name] = cloned
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		name := strings.TrimSpace(skill.Meta.Name)
		if name == "" {
			continue
		}
		cloned := cloneSkill(skill)
		cloned.Meta.Name = name
		registered[name] = cloned
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/registry_external.go` around lines 27 - 31, The map stores
skills under a trimmed name but leaves the cloned Skill.Meta.Name unchanged;
update the code that inserts into registered (where registered[name] =
cloneSkill(skill)) to normalize the cloned object's Meta.Name as well (e.g., set
clone.Meta.Name = strings.TrimSpace(skill.Meta.Name) or set it to the
already-computed name variable) so the stored clone and the map key use the same
trimmed value; ensure you still call cloneSkill(skill) to preserve other fields
and only adjust clone.Meta.Name before assigning into registered.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `RegisterExternal` trims the map key but stores a cloned `Skill` whose `Meta.Name` still contains the original whitespace. That leaves direct lookup behavior inconsistent with downstream comparisons against `skill.Meta.Name`.
  Fix approach: normalize `Meta.Name` on the stored clone before inserting it into the external registry map.
  Additional test scope needed: `internal/skills/registry_test.go` is outside the batch file list but is the minimal place to verify external registration behavior.
