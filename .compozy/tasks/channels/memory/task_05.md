# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add working Host API handlers for `channels/messages/ingest`, `channels/instances/get`, and `channels/instances/report_state`, backed by the daemon-owned channel registry and dedup store.
- Keep task scope at inbound ingest, route/session resolution, and prompt initiation; do not add outbound delivery projection here.

## Important Decisions

- Use the extension launch `initialize.runtime.channel` payload as the authorization boundary for channel Host API calls; the handler must only serve the bound `channel_instance_id` for the running extension.
- Put inbound dedup checks and route/session resolution behind a per-routing-key in-process lock so concurrent first messages do not create duplicate sessions or routes.
- Record inbound dedup only after prompt initiation succeeds so retries can re-attempt failed ingests instead of being suppressed by a half-complete first attempt.
- Allow inbound ingest only while the instance is `ready` or `degraded`; treat `disabled`, `starting`, `auth_required`, and `error` as unavailable for prompt creation.

## Learnings

- Task 04 already staged the channel Host API method identifiers, security grants, and some contracts; task 05 is primarily the orchestration/validation layer plus tests.
- `internal/store/globaldb` already exposes `Put/Get/DeleteExpiredChannelIngestDedup`, so no schema work is needed for this task.
- The Host API package needed dedicated coverage tests for `DescribeExtension` to keep package coverage at the required threshold after the channel surface expanded.
- The shared Host API test clock must be synchronized behind helper methods; `make verify` surfaced a real `-race` failure when channel ingest tests advanced the clock while prompt goroutines were still reading it.

## Files / Surfaces

- `internal/extension/host_api.go`
- `internal/extension/host_api_channels.go`
- `internal/extension/host_api_test.go`
- `internal/extension/host_api_integration_test.go`
- `internal/extension/describe_test.go`

## Errors / Corrections

- `make verify` initially failed on lint and then on a real `-race` issue in the Host API test harness; both were fixed before task completion, and the final tree passes `make verify`.

## Ready for Next Run

- Confirm how new channel-created sessions should pick their workspace/agent defaults when the channel instance is workspace-scoped versus global; current code seams only guarantee workspace-scoped creation cleanly.
