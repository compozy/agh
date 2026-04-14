# TC-REG-001: Existing Skill Install Flow Unchanged After Migration

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Regression |
| **Estimated Time** | 5 min |
| **Module** | `internal/cli/skill_commands.go` |
| **Changed In** | Task 05 — Skill CLI Migration |

## Objective

Validate that the skill install behavior is identical after migration from `internal/skills/marketplace/` to `internal/registry/` pipeline.

## Preconditions

- AGH binary with migrated skill code.
- ClawHub registry accessible.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh skill install <known-skill>` | **Expected:** Skill installed to correct path. `.agh-meta.json` created with provenance. |
| 2 | Verify `SKILL.md` exists in installed skill directory | **Expected:** Present and valid. |
| 3 | Verify `.agh-meta.json` has `source`, `slug`, `version`, `installed_at` | **Expected:** All fields populated. |
| 4 | Run `agh skill install <known-skill>` again | **Expected:** Same behavior as before migration (error or replace). |

## Regression Risk

High — the entire skill marketplace implementation was replaced. Any field name changes, path changes, or behavioral differences would break existing users.
