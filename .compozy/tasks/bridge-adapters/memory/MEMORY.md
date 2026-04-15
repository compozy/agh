# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

## Shared Decisions
- Task 01 establishes the canonical daemon bridge instance shape with typed `dm_policy`, `provider_config`, and structured degradation metadata persisted in `bridge_instances`, while provider manifests now surface `secret_slots` plus optional `config_schema` hints.
- Task 02 makes the bridge initialize handshake provider-scoped: `runtime.bridge` now carries `runtime_version`, `provider`, `platform`, and `managed_instances[]`, with each managed instance snapshot owning its bound secrets.
- Daemon-owned bridge lifecycle and secret binding resolution remain authoritative; provider runtimes receive clone-safe launch snapshots rather than live mutable bridge state.
- Task 04 expands the bridge Host API to provider-scoped ownership: runtimes now use `bridges/instances/list`, while `bridges/instances/get` and `bridges/instances/report_state` require an explicit `bridge_instance_id` tied to the negotiated runtime ownership set.
- `bridges/instances/report_state` now updates structured degradation atomically with lifecycle state changes; non-degraded statuses automatically clear stored degradation when the contract no longer allows it.
- Task 05 establishes `internal/bridgesdk` as the shared bridge provider substrate; future bridge runtimes should compose its runtime, Host API client, ingress guards, dedup, batching, and classified retry helpers rather than copying `telegram-reference` boot logic.
- Task 08 replaces the old single-instance `telegram-reference` reference path with a provider-scoped conformance runtime built on `internal/bridgesdk`; the reusable harness contract now requires ownership evidence from `bridges/instances/list` plus explicit `bridges/instances/get`, per-instance state markers, and provider-scoped delivery/ingress validation across multiple managed instances.
- Task 09 lands the first real production provider under `extensions/bridges/telegram`; later providers should mirror the shared `internal/bridgesdk` runtime pattern rather than promoting example adapters into production paths.
- Task 10 lands the first interaction-heavy production provider under `extensions/bridges/slack`; later providers can reuse the same `internal/bridgesdk` runtime shape for signed webhook ingress, typed `command`/`action`/`reaction` mapping, and provider-scoped conformance validation.
- Task 11 confirms that a second interaction-heavy provider can stay inside bridge v1 by returning Discord’s required inline interaction ACKs immediately and deferring Host API ingestion asynchronously behind the shared `internal/bridgesdk` ingress guards.
- Task 14 confirms the shared provider runtime can absorb Google Chat’s two ingress families (direct webhook + Pub/Sub push) inside bridge v1 without any daemon protocol fork.
- Task 15 confirms the shared provider runtime can route multiple GitHub bridge instances through one provider-scoped `/github` webhook endpoint while separating repository-scoped ownership and PAT vs App delivery behavior, including App installation-aware delivery using fixed or cached installation IDs.
- Task 16 confirms the shared provider runtime can keep provider-owned mode/auth branching entirely inside `provider_config`; Linear routes one shared webhook endpoint by `(organization_id, mode)` and uses provider-local `auth_mode` (`api_key` vs OAuth client-credentials) without changing daemon-global bridge semantics.
- Task 07 keeps web bridge management progressive: `provider_config` is operator-edited as a validated JSON object because provider manifests currently expose only `config_schema`/`version` hints and `secret_slots`, not field-level form definitions.
- `internal/bridgesdk` instance-cache synchronization must preserve launch-time bound secrets from the initialize handshake because the provider-scoped Host API refreshes bridge instance state only and does not resend secret material.
- Webhook-based provider integration tests can use the shared extension harness by supplying fixed listen/API-base env overrides and driving real HTTP requests into the spawned subprocess instead of relying on the reference adapter’s file-based update stream.
- Task 17 makes the harness record `bridges/instances/report_state` directly at the Host API boundary so classified recovery transitions emitted via `internal/bridgesdk.Session.ReportClassifiedError` are visible to shared conformance tests even when providers do not write explicit state-marker side effects.
- Task 17 defines the reusable conformance matrix as one aggregated row per provider/platform; multiple scenario summaries for the same provider should merge targets and managed-instance outcomes instead of being treated as duplicates.

## Shared Learnings
- Shared bridge contract changes flow through generated artifacts; after editing exported bridge structs, rerun the repo codegen path so `openapi/agh.json` and `sdk/typescript/src/generated/contracts.ts` stay aligned with the Go source.
- Platform routing policies must match the provider’s actual routing dimensions; Telegram forum/group traffic uses `group_id + thread_id`, so conformance fixtures must not require `peer_id` for those routes.
- Mixed Slack interaction fixtures should also align routing policy with the emitted payload dimensions; slash commands do not include thread identity, so multi-family Slack conformance runs should not force thread-based routing.
- Provider tests that boot a subprocess-backed bridge runtime and then swap runtime collaborators after `initialize` must wait until the async `afterInitialize` path has finished publishing instance state first; otherwise `go test -race` can hit factory/teardown races and hang on provider waitgroups.
- Teams confirmed the same routing rule under Bot Framework: keep channel-scoped ingress fixtures on `group_id + thread_id`, and treat proactive DM delivery as a separate path that relies on cached or configured `tenant_id` plus `service_url`.

## Open Risks
- `go test -tags integration ./internal/extension` currently fails in the unrelated `TestReferenceExtensionsEndToEnd` path because `sdk/examples/prompt-enhancer/node_modules/.bin/tsc` resolves outside the example root and trips the extension install symlink guard. Task-specific bridge manifest integration coverage still passes.

## Handoffs
