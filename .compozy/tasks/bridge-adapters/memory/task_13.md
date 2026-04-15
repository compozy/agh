# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the production Teams bridge provider on the shared provider-scoped runtime.
- Cover tenant pinning, bot identity, service URL behavior, inbound Bot Framework activity mapping, outbound delivery, and shared conformance/integration coverage.

## Important Decisions
- Treat the existing PRD/TechSpec/ADR/design docs as the approved design artifact for this execution run.
- Keep Teams bridge v1 scope to `message`, `action`, and `reaction` inbound events plus post/edit/delete outbound delivery; do not expand into task modules or richer Teams SDK parity.
- Keep tenant and service URL behavior in per-instance provider config and delivery metadata, not process-wide globals.
- Allow loopback `http://` Teams service URLs only for local verification against test Bot Framework servers; keep non-loopback service URLs `https://`-only.

## Learnings
- The harness already supports `provider_config` on managed bridge instances, so Teams tenant/service-url scenarios can be exercised through the shared subprocess path without new generic harness work.
- The local Teams reference adapter caches `serviceUrl` and `tenantId` from inbound activity metadata and encodes thread identity from `conversationId + serviceUrl`; the Go provider should preserve the same delivery-critical context.
- Teams integration fixtures must match one routing shape per managed instance: channel-scoped message/action/reaction fixtures work with `group_id + thread_id`, while proactive direct delivery depends on cached or configured `tenant_id` plus `service_url`.

## Files / Surfaces
- Touched surfaces: `extensions/bridges/teams/*`, `internal/extension/teams_provider_integration_test.go`, `go.mod`, `go.sum`, `.compozy/tasks/bridge-adapters/task_13.md`, `.compozy/tasks/bridge-adapters/_tasks.md`, `.compozy/tasks/bridge-adapters/memory/task_13.md`.

## Errors / Corrections
- Initial Teams integration coverage mixed channel routing with direct-message routing on one bridge instance and hit Host API invalid-params failures; corrected the fixture to keep ingress events channel-scoped and reserved proactive DM coverage for unit tests.
- Teams package coverage initially stalled below 80%; added focused tests for delivery wrappers, retry/shutdown helpers, Bot Framework auth helpers, reconciliation failures, marker utilities, webhook helper branches, and remote message reference helpers to reach the required threshold.

## Ready for Next Run
- Implementation, focused verification, and `make verify` are complete.
- Remaining action is local tracking update plus local commit creation.
