# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implemented a production Discord bridge provider on `internal/bridgesdk` with webhook-based ingress, typed v1 interaction mapping, outbound delivery/edit/delete support, conformance coverage, and timing-sensitive interaction ACK coverage.

## Important Decisions
- Reuse the Slack provider runtime/lifecycle pattern as the implementation base so Discord stays within the shared provider substrate instead of adding a Discord-only runtime path.
- Keep Discord inside the approved bridge v1 families: webhook events for message/reaction ingress, interactions for command/action ingress, and standard REST channel messaging for outbound delivery.
- Handle Discord interactions with immediate inline protocol ACKs (`pong`, deferred channel message, deferred update message) and push bridge ingestion asynchronously so the provider stays within Discord timing limits without bypassing shared ingress hardening.

## Learnings
- Telegram and Slack already prove the intended provider runtime pattern: `bridgesdk` owns initialize/deliver/health/shutdown, while the provider owns `resolveInstanceConfig`, webhook mapping, DM policy enforcement, and REST delivery translation.
- The shared webhook guard in `internal/bridgesdk/webhook.go` already covers method/content-type/body-size/rate-limit/in-flight protections, so Discord only needs provider-specific Ed25519 signature verification and payload routing on top.
- Discord interaction handling has a strict 3-second acknowledgment deadline, which makes immediate inline ACK responses a first-class integration concern for this task.
- Provider-scoped integration coverage can exercise Discord webhook verification reliably with deterministic Ed25519 keys as long as signed test requests use a fresh timestamp at send time.

## Files / Surfaces
- `extensions/bridges/discord/*` (new provider package tree)
- `internal/extensiontest/bridge_adapter_harness.go`
- `internal/extension/*discord*_integration_test.go` (new integration coverage)
- `docs/plans/2026-04-15-bridge-adapters-design.md`

## Errors / Corrections
- The task references `.resources/chat/packages/adapter-discord/src/format.ts`, but that file is not present in the workspace. The main Discord reference remains `.resources/chat/packages/adapter-discord/src/index.ts`.

## Ready for Next Run
- Discord provider implementation, unit coverage, and provider-scoped integration coverage are complete.
- Verification evidence: `go test ./extensions/bridges/discord`, `go test -coverprofile=/tmp/discord.cover ./extensions/bridges/discord` (`80.6%`), `go test -tags integration ./internal/extension -run DiscordProvider -count=1`, and `make verify` all passed on 2026-04-15.
