# SMOKE-005: Skill Search Works After Migration

| Field | Value |
|-------|-------|
| **Priority** | P0 (Critical) |
| **Type** | Smoke |
| **Estimated Time** | 3 min |
| **Module** | CLI / Skill Migration |

## Objective

Validate that the migrated `agh skill search` command functions correctly using the new `MultiRegistry` + `Installer` pipeline.

## Preconditions

- AGH binary built with migrated skill CLI code.
- ClawHub registry configured in `[skills.marketplace]`.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Run `agh skill search "test"` | **Expected:** Results displayed from ClawHub with name, version, author. Exit code 0. |
| 2 | Run `agh skill install <known-skill-slug>` | **Expected:** Skill downloaded, extracted, and installed. `.agh-meta.json` sidecar created. |
| 3 | Verify skill directory at expected path | **Expected:** Directory contains `SKILL.md` and `.agh-meta.json`. |
| 4 | Run `agh skill update --check` | **Expected:** Shows current vs available version for installed skills. |

## Edge Cases

- Skill search with no results: should display "no results" message, not an error.
- ClawHub API down: should fail gracefully with timeout error.
