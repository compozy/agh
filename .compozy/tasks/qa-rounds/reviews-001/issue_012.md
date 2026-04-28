---
status: resolved
file: internal/cli/skill_workspace.go
line: 269
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-IGMX,comment:PRRC_kwDOR5y4QM67_zdR
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Normalize `Source` in `skillInfoItemFromRecord`.**

`skillListItemsFromRecords` trims `record.Source`, but `skillInfoItemFromRecord` returns it raw. This can produce inconsistent `skill list` vs `skill info` output for the same daemon record.


<details>
<summary>Suggested fix</summary>

```diff
 func skillInfoItemFromRecord(record SkillRecord) skillInfoItem {
  return skillInfoItem{
    Name:        record.Name,
    Description: record.Description,
    Version:     record.Version,
-		Source:      record.Source,
+		Source:      strings.TrimSpace(record.Source),
    Path:        record.Dir,
    Enabled:     record.Enabled,
    Metadata:    cloneMetadata(record.Metadata),
  }
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func skillInfoItemFromRecord(record SkillRecord) skillInfoItem {
	return skillInfoItem{
		Name:        record.Name,
		Description: record.Description,
		Version:     record.Version,
		Source:      strings.TrimSpace(record.Source),
		Path:        record.Dir,
		Enabled:     record.Enabled,
		Metadata:    cloneMetadata(record.Metadata),
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_workspace.go` around lines 263 - 269, The Source field is
returned raw in skillInfoItemFromRecord causing inconsistent output versus
skillListItemsFromRecords which trims record.Source; update
skillInfoItemFromRecord to normalize Source the same way (e.g., use
strings.TrimSpace on record.Source) before assigning to skillInfoItem.Source so
both skillInfoItemFromRecord and skillListItemsFromRecords produce consistent
Source values.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `skillListItemsFromRecords` normalizes `record.Source`, but `skillInfoItemFromRecord` returns it raw. This can make `skill list --workspace` and `skill info --workspace` disagree for the same daemon payload. Fix by trimming the source in the info conversion path and pin it with daemon CLI coverage.

## Resolution

- Trimmed daemon skill source values in the `skill info` conversion path.
- Updated daemon CLI coverage to prove list/info source normalization stays consistent.
- Verified through targeted CLI tests and `make verify`.
