# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement the production WhatsApp Cloud API bridge provider for task 12 on top of `internal/bridgesdk`, including verify-challenge handling, signed webhook ingress, inbound DM mapping, outbound delivery, retry classification, and provider-scoped conformance coverage.
- Verified completion requires WhatsApp-specific unit coverage at or above 80%, integration coverage through the shared extension harness, and a clean `make verify`.

## Important Decisions

- Reused the production provider runtime shape established by Telegram, Slack, and Discord instead of extending the old reference adapter path.
- Kept WhatsApp secrets provider-scoped via `access_token`, `app_secret`, and `verify_token`, with `phone_number_id` stored in `provider_config`.
- Treated WhatsApp Cloud API deletes as unsupported in bridge v1 delivery handling and surfaced them as permanent errors rather than faking delete semantics.
- Extended the shared bridge adapter harness with optional `ProviderConfig` so provider-scoped integration tests can provision managed instances without leaking config into delivery defaults.

## Learnings

- WhatsApp resume snapshots must include `LastSentSeq` when `LastAckedSeq` is non-zero or the shared delivery request validator rejects them before provider delivery logic runs.
- The WhatsApp provider package needed additional helper/runtime tests to cover batching, shutdown, marker helpers, graph client calls, and content normalization to reach the task’s 80% coverage requirement.
- `go test -race ./extensions/bridges/whatsapp` exposed a test-only race in `TestResolveInstanceConfigAndDetermineInitialState`; waiting for async initialization to publish instance status before mutating `apiFactory` fixed the race and the cleanup hang.

## Files / Surfaces

- `extensions/bridges/whatsapp/provider.go`
- `extensions/bridges/whatsapp/provider_test.go`
- `extensions/bridges/whatsapp/main.go`
- `extensions/bridges/whatsapp/extension.toml`
- `extensions/bridges/whatsapp/README.md`
- `internal/extension/whatsapp_provider_integration_test.go`
- `internal/extensiontest/bridge_adapter_harness.go`

## Errors / Corrections

- Fixed an invalid resume fixture where `LastAckedSeq` exceeded the snapshot’s zero-valued `LastSentSeq`.
- Removed lint issues in the WhatsApp tests by checking response-body close errors and asserting the auth degradation result explicitly.
- Replaced the fixed `127.0.0.1:9999` listener in the runtime-config test with an ephemeral reserved port to avoid race-sensitive teardown problems.

## Ready for Next Run

- Task 12 implementation is verified complete in the workspace. Remaining close-out is limited to task tracking and the local code commit for the task-owned source changes.
