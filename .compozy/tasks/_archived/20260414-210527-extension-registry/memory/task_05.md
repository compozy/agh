# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Migrate `agh skill search/install/update` off `internal/skills/marketplace` onto `internal/registry` (`MultiRegistry`, `Installer`, ClawHub adapter), keep the rest of the skill CLI unchanged, and delete the deprecated package after the new path is verified.
- Required gates: existing `skill_marketplace_integration_test.go`, new/updated unit coverage around search/install/update, no remaining imports of `internal/skills/marketplace`, and final `make verify`.

## Important Decisions
- Follow the extension CLI wiring pattern: command layer loads registry sources, instantiates `MultiRegistry`, and closes it per command.
- Use the shared installer in a staging directory under the skills root, then perform skill-specific provenance writing and the final move in CLI code so the final path still matches the installed skill name.
- Align install behavior with the task test plan: the migrated install path now uses the replace-capable staging/move flow, and update uses `MultiRegistry.CheckUpdate()` plus a new `--check` flag.
- Keep the marketplace provenance sidecar behavior in the skill CLI after the registry migration: `Installer.Install()` handles extraction, then the CLI computes the directory hash and writes `.agh-meta.json` via `skills.WriteSidecar()` before the final move.

## Learnings
- The install refactor required a new command dependency hook (`loadSkillRegistrySources`) so CLI tests can inject registry sources without patching the runtime loader.
- The new install path needs registry info metadata before download-side provenance is written; the CLI test server now synthesizes skill info from download fixtures when tests only care about archives.
- Focused skill CLI coverage on the migrated surfaces now clears the task gate: `installMarketplaceSkill` 88.9%, `updateMarketplaceSkills` 81.0%, `updateMarketplaceSkill` 88.0%, `searchMarketplaceSkills` 90.9%.

## Files / Surfaces
- `internal/cli/skill_commands.go`
- `internal/cli/skill_marketplace.go`
- `internal/cli/skill_output.go`
- `internal/cli/skill_marketplace_integration_test.go`
- `internal/cli/skill_test.go`
- `internal/cli/root.go`
- `internal/registry/clawhub/client.go`
- `internal/skills/marketplace/` (deleted)

## Errors / Corrections
- Corrected the CLI test harness after the refactor surfaced an extra `Info()` lookup ahead of installs; the fake ClawHub server now returns synthesized detail metadata from download fixtures when no explicit info fixture exists.
- Added direct helper tests after the first coverage pass showed `installMarketplaceSkill` and `updateMarketplaceSkills` still below the task threshold; the new cases cover fallback version/registry handling, nil clock/info/detail paths, move failures, and empty `--all` updates.

## Ready for Next Run
- Implementation, targeted verification, and package deletion are complete. Remaining operator work is limited to creating the local code commit after tracking updates.
