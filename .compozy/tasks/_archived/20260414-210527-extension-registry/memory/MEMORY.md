# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 03 implemented dedicated `internal/registry/clawhub` and `internal/registry/github` adapters; later CLI tasks can wire those packages directly instead of extending the legacy marketplace client.
- Task 04 wired extension CLI remote `search` / `install` / `remove` / `update` flows against `MultiRegistry`, the shared `Installer`, and extension registry metadata persistence.
- Task 05 migrated skill CLI `search` / `install` / `update` onto the same `MultiRegistry` + `Installer` stack and removed `internal/skills/marketplace/`; skill provenance sidecars remain a CLI responsibility after installer extraction.

## Shared Decisions
- GitHub source-archive fallback relies on the existing installer behavior that walks through a single top-level extracted directory until it finds the manifest; later tasks should not add separate root-stripping logic for GitHub tarballs.
- The new ClawHub adapter intentionally preserves the existing ClawHub marketplace semantics and retry behavior while task 05 still owns the legacy-package removal.
- Marketplace-installed extensions now live under `<AGH_HOME>/extensions`, persist `registry_slug` / `registry_name` / `remote_version`, and intentionally print restart guidance instead of notifying the daemon in phase 1.

## Shared Learnings
- Adapter package coverage for task 03 cleared the PRD threshold: `internal/registry/clawhub` at 82.5% and `internal/registry/github` at 81.0%.

## Open Risks
- None currently recorded for the registry adapter layer.

## Handoffs
- Task 04 and task 05 can treat `internal/registry/clawhub` and `internal/registry/github` as the verified backends for `MultiRegistry` wiring.
- Task 05 should preserve the phase-1 restart-message behavior for remote installs unless daemon reload support is explicitly added as new scope.
- Future registry work should treat both extension and skill CLIs as consumers of the shared registry layer; new registry-specific behavior belongs under `internal/registry`, while skill/extension metadata persistence stays in the respective CLI/domain packages.
