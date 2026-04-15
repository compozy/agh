# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement the production Slack bridge provider under `extensions/bridges/slack` on top of `internal/bridgesdk`.
- Cover Slack message events plus typed `command`, `action`, and `reaction` bridge ingest families, signed webhook ingress, outbound post/edit/delete delivery, and shared conformance validation.

## Important Decisions

- Treat the task PRD, techspec, ADRs, and approved design doc as the design source of truth; do not reopen design approval.
- Reuse the Telegram provider/runtime structure as the local implementation pattern, but keep Slack request parsing, signing verification, routing, and delivery semantics Slack-specific.
- Keep Slack v1 scope limited to Events API messages, slash commands, block actions, reactions, and `chat.postMessage`/`chat.update`/`chat.delete`.
- Cover the production runtime directly under `extensions/bridges/slack` and validate it through both package-level unit tests and subprocess-backed `internal/extension` integration tests.

## Learnings

- The bridge contract already exposes typed inbound `command`, `action`, and `reaction` payloads, so Slack does not need protocol changes before implementation.
- The shared harness already validates provider-scoped runtime ownership, per-instance state reporting, and delivery sequencing; Slack-specific interaction assertions can layer on top through provider tests.
- Mixed Slack interaction scenarios work cleanly through the shared harness when the test bridge routing policy matches the payload dimensions actually emitted by the provider; slash commands do not carry thread identity, so the integration fixture should not require thread routing for those runs.

## Files / Surfaces

- `.compozy/tasks/bridge-adapters/task_10.md`
- `.compozy/tasks/bridge-adapters/_techspec.md`
- `.compozy/tasks/bridge-adapters/_tasks.md`
- `.compozy/tasks/bridge-adapters/adrs/adr-002.md`
- `.compozy/tasks/bridge-adapters/adrs/adr-003.md`
- `docs/plans/2026-04-15-bridge-adapters-design.md`
- `internal/bridges/types.go`
- `internal/bridgesdk/*`
- `internal/extensiontest/bridge_adapter_harness.go`
- `extensions/bridges/telegram/*`
- `.resources/chat/packages/adapter-slack/src/index.ts`
- `.resources/hermes/gateway/platforms/slack.py`
- `extensions/bridges/slack/main.go`
- `extensions/bridges/slack/markers.go`
- `extensions/bridges/slack/extension.toml`
- `extensions/bridges/slack/provider.go`
- `extensions/bridges/slack/provider_test.go`
- `internal/extension/slack_provider_integration_test.go`

## Errors / Corrections

- Initial package coverage stopped at 78.4%; added targeted branch tests around Slack signature validation, API client error paths, helper edge cases, retry shutdown behavior, and marker helpers to push the package to 81.0%.
- The first Slack integration attempt failed ingress verification because the fixture signed requests with the scenario timestamp instead of current wall-clock time; corrected the helper to sign with current UTC time.
- The first mixed interaction integration attempt also failed host ingest validation because the harness routing policy required unsupported dimensions for slash-command traffic; corrected the Slack fixture to use a routing policy aligned with the emitted interaction identities.
- A post-commit rerun exposed a brittle integration assertion that assumed the final two mock Slack API calls were always `chat.postMessage` then `chat.update`; corrected the test to assert required delivery methods by presence instead of fixed tail order.

## Ready for Next Run

- Slack provider runtime, provider-specific unit coverage, subprocess-backed integration coverage, and repository verification are complete.
- Fresh evidence:
  - `go test ./extensions/bridges/slack -count=1 -coverprofile=/tmp/slack.cover` => 81.0% statements
  - `go test ./internal/extension -tags integration -run 'SlackProvider' -count=1`
  - `make verify`
- Remaining operator task is updating task tracking and creating the local commit without staging tracking-only files.
