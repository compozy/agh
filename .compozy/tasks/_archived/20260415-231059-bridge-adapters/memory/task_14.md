# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement the production Google Chat bridge provider under `extensions/bridges/gchat` using the shared provider-scoped runtime.
- Cover both Google Chat direct webhook events and Pub/Sub Workspace Events payloads while preserving bridge v1 routing and delivery semantics.
- Finish with unit coverage >=80%, required integration/conformance evidence, task tracking updates, and one verified local commit.

## Important Decisions

- Reuse the established production-provider runtime shape from Telegram/Slack/Discord/Teams instead of widening `internal/bridgesdk`.
- Normalize Google Chat direct webhook messages, card-click actions, and Pub/Sub reaction/message events into existing bridge v1 families only.
- Keep Google Chat credentials in the provider-declared secret slots and use `provider_config` only for mode, webhook, batching, DM, and API override settings.
- Prefer explicit runtime/env token URL overrides over the service-account JSON `token_uri` so subprocess-backed tests and local overrides can redirect OAuth cleanly.

## Learnings

- Google Chat is the first bridge task here that must accept two inbound payload families under one provider runtime: direct Add-ons-style webhook events and Pub/Sub push messages from Workspace Events.
- The Chat-SDK reference uses bearer-token verification for both modes, with project number used for direct webhook audience checks and a separate expected audience for Pub/Sub push validation.
- Reaction events are Pub/Sub-only in the current reference flow and need message/thread recovery to preserve target identity.
- Bridge v1 action and reaction envelopes must not populate message-family fields such as `platform_message_id`; Google Chat needed explicit normalization fixes there.

## Files / Surfaces

- Added: `extensions/bridges/gchat/main.go`
- Added: `extensions/bridges/gchat/markers.go`
- Added: `extensions/bridges/gchat/extension.toml`
- Added: `extensions/bridges/gchat/provider.go`
- Added: `extensions/bridges/gchat/provider_test.go`
- Added: `internal/extension/gchat_provider_integration_test.go`

## Errors / Corrections

- Fixed Google Chat action/reaction normalization so non-message families no longer set `platform_message_id`, which violated the shared bridge contract and broke Pub/Sub reaction ingestion.
- Fixed config precedence so `AGH_BRIDGE_GCHAT_TOKEN_URL` overrides the service-account `token_uri`, allowing subprocess-backed delivery tests to use the mock OAuth server.
- Expanded provider-local coverage from the initial failing scaffold to `80.6%` by adding config, delivery, webhook, lifecycle, and helper-path tests.

## Ready for Next Run

- Implementation complete and verified.
- Evidence:
  - `go test ./extensions/bridges/gchat -count=1`
  - `go test -coverprofile=/tmp/gchat.cover ./extensions/bridges/gchat` -> `coverage: 80.6% of statements`
  - `go test -tags integration ./internal/extension -run GChatProvider -count=1`
  - `make verify`
