# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose automation to extensions through Host API CRUD/read surfaces plus `automation/triggers/fire`, and add automation lifecycle hook events that can mutate or cancel dispatch before prompt submission.

## Important Decisions
- Reuse canonical automation DTOs and manager methods instead of defining extension-only automation models.
- Put automation pre/post/run hook emission in the shared dispatcher so every activation path uses the same lifecycle.
- Route extension-originated `ext.*` events through the existing trigger engine and dispatcher rather than calling dispatch directly.
- Reuse the existing manager/store semantics for config-backed definitions in the Host API: config jobs and triggers only allow enabled-state changes and reject delete or mutable field updates.

## Learnings
- `internal/extension` currently only exposes sessions/memory/observe/skills Host API methods; automation methods and capabilities are absent.
- `internal/hooks` has no automation event family entries yet, so automation lifecycle hooks need additive event, payload, matcher, and dispatch plumbing.
- `internal/automation` already has manager/runtime seams for webhook and built-in ingress, which are the right place to add extension trigger fire support.
- Hook taxonomy consumers can drift if they hardcode event counts. `internal/observe/hooks_test.go` now derives its expectation from `hookspkg.AllEventDescriptors()` so additive hook families do not create unrelated test failures.
- The extension Host API automation handlers benefit from package-local CRUD tests because integration-tagged tests do not contribute to unit-package coverage in `internal/extension`.

## Files / Surfaces
- `internal/extension/host_api.go`
- `internal/extension/contract/host_api.go`
- `internal/extension/protocol/host_api.go`
- `internal/extension/capability.go`
- `internal/hooks/events.go`
- `internal/hooks/payloads.go`
- `internal/hooks/dispatch.go`
- `internal/automation/dispatch.go`
- `internal/automation/trigger.go`
- `internal/automation/manager.go`
- `internal/automation/extension.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/hooks_bridge.go`
- `internal/observe/hooks_test.go`
- `openapi/agh.json`
- `sdk/typescript/src/generated/contracts.ts`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- `make verify` initially failed on two staticcheck selector cleanups and two integration-only test helpers living in non-integration test files; those helpers were moved under the integration build tag and the selectors were simplified.
- `internal/observe/hooks_test.go` had a stale hardcoded hook-event count after the automation hook family landed; the test now reads the taxonomy size dynamically.
- Package-local coverage for `internal/automation` initially stalled at `79.3%`; targeted helper/observer tests in `internal/automation/{manager_test.go,trigger_test.go}` raised it to `80.1%` without expanding production scope. `internal/extension` remains at `80.0%` and `internal/hooks` at `82.0%`.

## Ready for Next Run
- Implementation, verification, and task tracking are complete. The remaining workspace dirtiness is limited to task-tracking/workflow artifacts that were intentionally left out of the code commit set.
