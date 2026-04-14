# TC-FUNC-027: Skill Install Via Migrated Pipeline

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Functional |
| **Estimated Time** | 3 min |
| **Module** | `internal/cli/skill_commands.go` |

## Objective

Validate that `agh skill install` works correctly through the migrated `MultiRegistry` + `Installer` pipeline.

## Preconditions

- AGH binary built with migrated skill code.
- ClawHub registry configured.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh skill install <known-skill-slug>` | **Expected:** Skill downloaded from ClawHub, extracted, and installed. |
| 2 | Verify `.agh-meta.json` sidecar created alongside skill | **Expected:** JSON file with `source`, `slug`, `version`, `installed_at` fields. |
| 3 | Verify `SKILL.md` exists in skill directory | **Expected:** SKILL.md present with valid frontmatter. |

## Edge Cases

- Skill already installed: should report "already installed" or offer to replace.
- ClawHub returns 404: should report "skill not found".
