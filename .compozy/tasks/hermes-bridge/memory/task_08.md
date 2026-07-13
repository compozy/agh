# Task Memory: task_08

## Objective Snapshot

- Bring public bridge documentation to truthful 8/8 provider parity, add operator setup coverage and a bridge-author guide, and co-ship the official AGH skill references without changing runtime behavior.

## Preflight Baseline

- `index.mdx` lists only Slack, Discord, and Telegram secret slots although eight manifests are discovered.
- `setup.mdx` is already CLI-first for Slack, WhatsApp, Telegram, and Discord. Task 08 must preserve those flows, add Microsoft Teams, Google Chat, GitHub, and Linear coverage, and make the shared lifecycle clearer.
- The existing WhatsApp section is behaviorally useful but will become the dedicated provider page required by the approved task shape.
- `meta.json` exposes only routing, progress, and setup. It needs entries for the provider pages and bridge-author guide so the site build owns link/navigation integrity.
- Discord runtime truth is Ed25519 signature verification plus Discord REST create/edit/delete endpoints; its current README already uses those terms, so the audit must keep them exact and remove any residual contradictory wording elsewhere.
- The PRD and user-story companions referenced by the resliced task are not present in this workflow directory. The approved `_techspec.md`, `_tasks.md`, `_tests.md`, task file, and analysis artifacts are the available authoritative catalogs.
- User override: run only the docs drift owner and focused site lint/typecheck/test/build during Task 08. Defer global suites and `make verify` to the workflow tail.

## Documentation Structure

- Keep `setup.mdx` as the shared CLI-first operator path and retain its detailed Slack, Telegram, and Discord sections.
- Move detailed WhatsApp setup into `setup-whatsapp.mdx`; add `setup-teams.mdx`, `setup-gchat.mdx`, `setup-github.mdx`, and `setup-linear.mdx` with behavior first, prerequisites, disabled create/bind/configure/verify/enable/send sequence, and provider-specific troubleshooting.
- Add `adding-a-bridge.mdx` for extension authors and mirror its contract in `internal/bridges/ADDING_A_BRIDGE.md`. The site page is the tutorial; the repository guide is the concise implementation checklist and grep verification recipe.
- Keep `skills/agh/SKILL.md` lean. Update only the existing `native-tools.md` and `runtime-operations.md` references with exact verbs, routes, progress config, and management/native-tool boundaries.

## Documentation Quality Review

- A first truthful pass was still materially shallower than the vendored Hermes guides. The second pass now treats each provider page as an operator journey rather than a command inventory.
- Every new provider guide explains behavior before setup, where credentials originate, the provider-console sequence, public-to-local endpoint topology, disabled and enabled verification checkpoints, route creation plus a real `send-test`, configuration defaults, known limits, security, and troubleshooting by observable symptom.
- The Teams, WhatsApp, Google Chat, GitHub, and Linear console flows were cross-checked against current first-party provider documentation. External console paths remain linked instead of copied as timeless product claims.
- The author guide now explains the daemon/provider boundary, shared SDK ownership, file split, manifest and runtime templates, reconciliation, inbound trust ordering, acknowledgement semantics, checks, progress, owning test suites, public-surface co-ship, and a review grep recipe.
- Runtime truth remains authoritative where Hermes differs: Google Chat uses 32,000 UTF-8 bytes; GitHub and Linear control verification reports identity as `skipped`; guided setup does not collect the local listener address.

## Runtime Truth Sources

- Secret slots come exclusively from the eight `extensions/bridges/*/extension.toml` manifests.
- Provider config and behavior claims come from each provider's `provider.go`, control implementation, API client, tests, and README where present.
- CLI syntax and route parity come from `internal/cli/bridge_{mutation,setup,manifest,verify}.go` and the generated HTTP/UDS contracts.
- Bridge-author scaffolding comes from the Task 07 shared owners (`ProviderLifecycle`, `ManagedConfigReconciler`, `ProviderHTTPServer`, `RunProviderCommand`) plus the auto-discovered provider conformance suite.
- Competitor material under `.resources/hermes/` informs guide structure only; AGH runtime truth wins every claim.

## Test Placement

- Invariant: every discovered provider appears with exact required/optional slots and has setup coverage; the Slack scope list equals runtime constants. Owning layer: extension/docs contract. Canonical suite: `internal/extension/bridge_docs_conformance_test.go`; do not add prose snapshots or duplicate tests.
- Invariant: every navigation entry resolves and MDX compiles. Owning layer: Fumadocs site. Canonical gates: focused Turbo site lint/typecheck/test/build.
- Spot audit Discord Ed25519/REST, Teams Bot Framework, and GitHub/Linear auth modes manually against runtime code; record evidence instead of freezing prose.

## Completion Evidence

- `bunx oxfmt --check` passed on all 13 changed documentation and skill artifacts.
- `go test -race ./internal/extension -run '^TestBridgeProviderDocsConformance$' -count=1` passed all four conformance cases.
- `bunx turbo run typecheck test --filter=./packages/site` passed typecheck and 50 suites / 247 tests. The route-owner test caught an external Bot Framework path that looked like an undocumented AGH API route; the prose was corrected and the full lane then passed.
- `bunx turbo run build --filter=./packages/site` passed and generated 1,578 static pages. The existing Next workspace-root/multiple-lockfile warning remains unrelated.
- The task-requested combined Turbo `lint` lane cannot exist for `@agh/site` because that package defines no `lint` task. Oxfmt is the focused owner for changed MDX/Markdown/JSON; global Bun lint remains deferred with the final workflow gate by explicit user direction.
- Manual truth audit passed: Discord uses Ed25519 plus Discord REST; Teams uses Bot Framework identity/service URLs; GitHub uses PAT or GitHub App bindings; Linear separates comments/Agent Sessions from API-key/OAuth authentication.
- QA tracker: `NB-051` already covers bridge-provider setup documentation as `untested`, so no duplicate row was added.

## AGH Impact Audit

- Native tools: no descriptor/tool ID/schema change; checked bridge native-tool boundaries and documented lifecycle/setup/verify/send-test/webhook registration as CLI/HTTP/UDS management surfaces.
- Extensibility and hooks: documentation only; checked `bridge.adapter`, manifest slots, shared bridgesdk owners, extension discovery/conformance, webhook registration, delivery/progress, and config lifecycle. No runtime registry, hook, bundle, sidecar, or `config.toml` behavior changed.
- Workspace data isolation: no runtime datum changed. The guides preserve workspace-scoped bridge creation and `workspace_id` propagation; no CLI/HTTP/UDS/core/store/web/SSE/cache/event code changed.
- Official AGH skill: updated `skills/agh/references/native-tools.md` and `runtime-operations.md` with exact verbs, routes, verification semantics, progress config, and agent-manageable surface boundaries.

## Working Set

- `packages/site/content/runtime/core/bridges/`
- `extensions/bridges/discord/README.md`
- `internal/bridges/ADDING_A_BRIDGE.md`
- `skills/agh/references/{native-tools,runtime-operations}.md`
- `internal/extension/bridge_docs_conformance_test.go`
- `.compozy/tasks/hermes-bridge/{task_08.md,_techspec.md,_tests.md,memory/MEMORY.md,state.yaml}`
