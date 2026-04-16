# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Migrate bundle catalog records and bundle activations to canonical resource records.
- Replace legacy activation inventory with owner-indexed owned resources using `owner_kind=bundle.activation` and `owner_id=<activation-id>`.
- Make activation fan-out write owned `automation.job`, `automation.trigger`, and `bridge.instance` records through typed resource stores and the explicit mixed-kind `bundle.activation` projector path.

## Important Decisions
- Treat the approved PRD/TechSpec/ADRs as the execution design artifact; no separate design-approval loop is needed for this implementation task.
- Activation fan-out will not use `DependsOn()` edges for automation or bridge kinds; downstream projection is triggered by canonical store writes.

## Learnings
- Baseline has the generic `NewBundleActivationProjectorRegistration` seam in `internal/resources`, but no domain bundle/activation codecs or daemon registration yet.
- Baseline still persists activation inventory through `bundle_activation_inventory`; service fan-out currently calls automation/bridge managed sync APIs.
- Task 10/11 memory confirms `automation.job`, `automation.trigger`, and `bridge.instance` are already canonical desired-state kinds and can be targeted by bundle activation fan-out.
- Bundle fan-out needs daemon-owned owner override stamping so downstream records can keep daemon source authority while using `owner_kind=bundle.activation` and `owner_id=<activation-id>` for cleanup.
- Bundle source sync must run after extension runtime is attached so registered extension bundle declarations publish into `bundle` records and trigger `bundle.activation` through the resource topology.
- `bundle.activation` remains the only mixed-kind projector dependency; downstream automation/bridge projection is reached through canonical writes and post-commit triggers, not through `DependsOn()` edges.

## Files / Surfaces
- Touched: `internal/bundles`, `internal/daemon`, `internal/resources`, `internal/store/globaldb`, `internal/extension`, and affected unit/integration tests.
- Added resource-backed bundle codecs/store/projector files under `internal/bundles` plus daemon bundle source sync/projector wiring.
- Removed legacy globaldb activation/inventory store authority; remaining lifecycle guards count canonical `resource_records`.

## Errors / Corrections
- Initial coverage only exercised resource store behavior under integration tags; added non-integration unit coverage for codecs and `ResourceStore` to meet the package coverage target.
- First full `make verify` failed at Go lint. Corrections: split large daemon codec/projector/resource-store helpers, removed unused context parameters, used the repository's nil-context test variable pattern, and passed heavy automation manager options by pointer.
- A fresh coverage run after lint fixes reported 79.9%, below the task target; added deterministic activation/inventory ordering coverage around the resource store comparators.

## Ready for Next Run
- Implementation, tracking, and final verification are complete. Validation evidence:
  - `go test ./internal/bundles`
  - `go test -tags integration ./internal/bundles`
  - `go test ./internal/daemon ./internal/resources ./internal/store/globaldb ./internal/extension ./internal/api/core`
  - `go test -cover ./internal/bundles` => 81.6%
  - `go test ./internal/bundles ./internal/daemon ./internal/resources ./internal/store/globaldb ./internal/extension ./internal/api/core ./internal/automation`
  - `make lint`
  - `make verify` => exit 0, 4123 Go tests, 82 web test files / 676 web tests, 0 lint issues, package boundary check OK.
- Next required step is the local code commit after self-review.
