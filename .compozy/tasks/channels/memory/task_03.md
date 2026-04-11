# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Completed: added a channel-core outbound target resolver that canonicalizes `channel_instance_id`, `peer_id`, `thread_id`, `group_id`, and `mode` from instance defaults plus explicit overrides without importing automation code.
- Completed: proved the seam with unit tests for mode/field validation, integration tests for instance-default resolution and workspace isolation, 80.4% package unit coverage, and a clean `make verify`.

## Important Decisions

- Keep `ChannelInstance.DeliveryDefaults` persisted as JSON for task 03 and introduce typed decode/validation logic in `internal/channels/` rather than expanding schema/store scope.
- Model target resolution as a channel-owned seam that loads instance metadata and returns one canonical `DeliveryTarget`.
- Only the target-related keys in `delivery_defaults` (`peer_id`, `thread_id`, `group_id`, `mode`) participate in outbound resolution; unrelated JSON keys remain ignored by this seam.
- Explicit overrides take precedence over instance defaults, and unresolved mode falls back to `direct-send`.

## Learnings

- Current `DeliveryTarget.Validate()` only requires `channel_instance_id`; it does not enforce mode-specific completeness or incompatible field combinations yet.
- `internal/channels` currently has routing resolution in `registry.go` but no outbound target resolver API.
- A thin unit test through `Service.ResolveDeliveryTarget` was needed to push `internal/channels` unit coverage from 78.5% to 80.4% while still exercising real behavior instead of synthetic branches.

## Files / Surfaces

- `internal/channels/types.go`
- `internal/channels/target.go`
- `internal/channels/target_test.go`
- `internal/channels/target_integration_test.go`
- `internal/channels/registry.go`
- `internal/channels/registry_test.go`
- `internal/channels/registry_integration_test.go`
- `.codex/ledger/2026-04-11-MEMORY-delivery-targets.md`

## Errors / Corrections

- No implementation errors yet. Pre-change signal is structural: missing resolver seam and incomplete target validation.
- Initial unit coverage landed at 78.5%; corrected by adding a unit test that resolves through the service seam and clone path instead of padding branch-only tests.

## Ready for Next Run

- Code verification is complete. Remaining operational step is the local task commit, keeping tracking and memory artifacts out of the staged set unless explicitly requested.
