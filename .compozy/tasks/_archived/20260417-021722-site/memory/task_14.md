# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Create Task 14 bridge docs under `packages/site/content/runtime/bridges/`: overview, routing, setup, and `meta.json`.
- Required evidence before completion: site build, browser QA for every touched route, full verification gate, self-review, tracking updates, and local commit if the gate is clean.

## Important Decisions

- Document current implementation first; reconcile any archived/plan material against source before using it.
- Treat bridge instances as bridge API/CLI records, not TOML configuration blocks. `provider_config` and `delivery_defaults` are JSON objects on the instance.
- Document `env:NAME` as the only stock daemon bridge secret resolver.
- Recommend workspace-scoped bridge instances for inbound routing because current bridge session creation requires `workspace_id`.
- Setup docs focus on Slack, Discord, and Telegram per task scope; overview can mention the installed provider catalog is dynamic.

## Learnings

- Shared workflow memory says docs build should use `bunx turbo run build --filter=@agh/site`; the task's literal `packages/site` selector is stale.
- QMD has required prior-context collections: `agh-site-archived`, `agh-site-ledger`, and `agh-site-plans`.
- Bridge provider catalog comes from installed extensions with `bridge.adapter` capability; current provider manifests include Slack, Discord, Telegram, WhatsApp, Teams, Google Chat, GitHub, and Linear.
- Current required secret slots: Slack `bot_token` + `signing_secret`; Discord `bot_token` + `public_key`; Telegram `bot_token` with optional `webhook_secret`.
- Routing policy dimensions are `include_peer`, `include_thread`, and `include_group`; `include_thread` is invalid unless peer or group is also included.
- Every enabled routing dimension must be present in the inbound event. Direct-message and shared-channel traffic often need separate bridge instances because direct events carry `peer_id`, while channel/group events carry `group_id`.
- Delivery broker orders events per route, retries failed transport/ack paths through resume snapshots, coalesces slow deltas, and exposes backlog/drop/failure metrics. No user-facing retry/timeout knobs exist.
- CLI bridge commands exist for `list`, `get`, `create`, `update`, `enable`, `disable`, `restart`, `routes`, and `test-delivery`; secret bindings and provider config require the API.

## Files / Surfaces

- Docs output: `packages/site/content/runtime/bridges/*`.
- Source surfaces to inspect: `internal/bridges/`, bridge config, CLI/API integration, and site MDX/navigation patterns.
- Runtime navigation to update: `packages/site/content/runtime/meta.json`.

## Errors / Corrections

- Corrected initial routing examples to avoid policies that require both `peer_id` and `group_id` on the same event.
- The task's literal build selector `bunx turbo run build --filter=packages/site` is stale and fails because the package is named `@agh/site`; the correct selector passed.

## Verification Evidence

- CLI help verified: `go run ./cmd/agh bridge create --help`, `go run ./cmd/agh bridge routes --help`, `go run ./cmd/agh bridge test-delivery --help`, `go run ./cmd/agh extension install --help`, `go run ./cmd/agh workspace add --help`.
- Provider package compile checks passed for Slack, Discord, and Telegram using `/tmp/agh-bridge-*-docs-check` outputs.
- Site build passed with `bunx turbo run build --filter=@agh/site` after confirming the stale `packages/site` selector failure.
- Browser QA passed through `make site-dev` and `agent-browser`: `/runtime/bridges/overview/`, `/runtime/bridges/routing/`, `/runtime/bridges/setup/`, and `/runtime/cli-reference/bridge/create/` returned 200 and rendered expected content; overview Mermaid text was present.

## Completion Blockers

- Full `make verify` failed outside Task 14 in `web/src/styles.test.ts`: tests still expect `#121212`, `#1C1C1E`, and `#2C2C2E`, while current CSS defines `#141312`, `#1e1c1b`, and `#2e2c2b`.
- Task tracking and local commit remain blocked until the full verification gate is clean.

## Ready for Next Run
