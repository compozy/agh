# Task Memory: task_05

## Objective Snapshot

- Turn the Bridges Web surface into a truthful setup orchestrator: post-create Slack manifest handoff, state-derived setup checklist, inline provider verification, Telegram webhook registration, real send-test alongside dry-run target resolution, and typed progress defaults in create/edit.

## Preflight Decisions

- The fixed Slack endpoint requires `instance=<persisted-id>` and builds the manifest from persisted `provider_config.webhook.public_url`. The pre-create provider stage may advertise the handoff, but it must not fetch, fabricate, reconstruct, or silently persist a manifest. The explicit create response supplies the only valid ID; the still-open wizard then fetches and displays the daemon JSON.
- Web setup creation changes from `enabled: true` to `enabled: false`. This makes the required sequence real (`create -> bind/configure -> register/verify -> enable -> healthy`) and is necessary because Telegram registration rejects enabled instances.
- A successful create is a committed boundary. During the post-create handoff, Back/Cancel cannot return to stale draft fields or create again. Manifest retry reuses the same persisted ID; Close/Open bridge navigates to that instance even if manifest fetch or clipboard copy fails.
- Provider setup profiles are a small typed presentation catalog grounded in the four fixed Task 04 routes and known platform dashboards: Slack exposes manifest handoff; Telegram exposes registration; webhook-backed providers expose callback configuration. The catalog supplies UI routing/check-to-slot relations only and never becomes a second source of runtime status.
- `verified` and Telegram `registered` are not persisted bridge fields. They are current-session API evidence with explicit unknown/not-run states. Never infer either from `enabled`, `status`, or `health=ready`; clear the evidence after bridge ID, provider config, or secret-binding changes.
- Verify records do not identify a secret slot. Inline results use exact typed platform/check-to-slot mappings. Never parse remediation prose, list order, or check-name substrings. Unmapped checks remain in the general verification summary.
- Required-secret readiness is the required `provider.secret_slots` subset against daemon binding metadata; missing optional bindings never block setup. Webhook readiness uses canonical `provider_config.webhook.public_url`, not an internal path or a guessed host.
- Existing `test-delivery` stays a dry-run target resolver and remains available while disabled. The real `send-test` is distinctly labeled, requires a non-empty message, uses the provider delivery path, and is disabled until the bridge is enabled.
- Progress fields share one editor. Omitting progress means provider default; selecting a mode initializes a valid grouping. Returning to provider default removes the entire progress object instead of retaining stale grouping/typing/reaction overrides.
- User override remains active: Task 05 runs only focused/proportional Web, Playwright, type, lint, and screenshot checks. Global Bun/Web/E2E gates and the single `make verify` remain reserved for the completed workflow tail.

## Design Frame

- Scene: a bridge operator under setup pressure should complete one coherent activation path without bouncing among AGH, terminal commands, documentation, and provider dashboards.
- Register: Product. Dials: `VISUAL_VARIANCE=3`, `MOTION_INTENSITY=2`, `INFORMATION_DENSITY=7`.
- Use a dense flat operational hierarchy with one emphasized next action. Signal colors describe real pass/warn/fail state only. Reuse `@agh/ui` primitives and canonical tokens; no decorative cards, raw values, nested surfaces, or invented controls.
- Setup sits immediately under the detail header so the first unresolved step precedes generic metrics. Manifest and detail states must cover loading, failure/retry, copy failure, not-run, pass, warn, fail, skipped, disabled, degraded, and healthy.

## Architecture Council Decision

- The daemon owns facts and side effects; the Web owns orchestration and presentation of evidence. One pure setup projection consumes provider discovery, persisted bridge state, bindings, health, and current verify/register responses.
- Create uses one explicit persistence transition. A Slack result advances the same wizard to a post-create handoff and starts the manifest query by returned ID; non-Slack creation navigates directly to the detail setup path. Duplicate submit and cross-instance manifest reuse are invalid states.
- Structural owners stay separated as `adapters -> lib -> hooks/view-models -> components`. Add a dedicated Task 04 control adapter, manifest query, setup mutations, pure profile/state projection, and route orchestration hooks rather than growing the 429-line adapter or 478-line detail hook.
- `bridge-create-dialog.tsx` and `bridge-detail-panel.tsx` become sub-500-line composition shells. Extract provider/runtime steps, shared delivery/progress fields, manifest handoff, setup checklist, provider secret runtime, target directory, event stream, header/metrics, and delivery-test actions into single-purpose files.
- Reuse one delivery draft form for two explicit intents where practical, but keep dry-run and real-send response types, side effects, result labels, and endpoint counters distinct.

## Council Synthesis

- Advisors: Product Mind, Architect Advisor, and Devil's Advocate. Product centered the activation outcome and argued against a speculative provider-capability framework. Architect required explicit evidence/state and composition boundaries. Devil's Advocate accepted the direction conditionally but preserved failure-transition and stale-evidence risk.
- Consensus: keep the operator in one activation thread; never generate a pre-create manifest; create disabled; treat verify/register as transient evidence; use a small typed setup profile until the daemon publishes an authoritative capability/check-slot contract; split the current god files.
- Resolved tension: the profile map is intentionally narrow discovery/presentation data, not a general state-machine or provider framework. Architect partially conceded the generic framework; the frontend map must be deleted if a backend descriptor later owns these relations.
- Surviving dissent: a post-create handoff can still orphan or duplicate instances if close/retry semantics are careless, and transient evidence can survive a mutation and show false green. Devil's Advocate withholds unconditional approval until the composed invariant is automated.
- Required mitigation/evidence: manifest GET count is zero before create; create POST count is exactly one and persists `enabled:false`; every fetch/copy retry uses the returned ID; closing after the commit routes to that instance; ID/config/secret changes clear verify/register evidence; profile check mappings are validated against real descriptor slots; unknown checks remain global.
- Position evolution: Product held the minimal product-safe path; Architect partially moved from possible framework expansion to the fixed profile while retaining five load-bearing boundaries; Devil's Advocate partially conceded the architecture but retained test-gated dissent.

## Test Placement

- Adapter request/status/body/error invariants: existing `adapters/__tests__/bridges-api.test.ts`.
- Manifest key/options/signal/enablement: existing query-key, query-options, and `use-bridges` suites.
- Verify/register/send mutations and invalidation: existing `hooks/__tests__/use-bridge-actions.test.tsx`.
- New pure checklist/profile/check-to-slot truth projection: new canonical `lib/__tests__/bridge-setup.test.ts`; no existing suite owns this invariant.
- Progress create/update serialization and provider-default removal: existing `lib/__tests__/bridge-drafts.test.ts`; normalization remains in the formatter suite only if behavior changes.
- Manifest/progress/detail rendering and interaction: existing create/edit/detail component suites plus a colocated real-send dialog suite if a dedicated component is introduced.
- Shallow navigation/toast orchestration: existing route suites. The full stateful create HTTP sequence needs one new real-hook/MSW route integration owner because the existing bridges route suite mocks the whole system layer.
- Mocked-daemon public journeys belong in a new focused `web/e2e/__tests__/bridges-setup.spec.ts`; the existing 689-line real-daemon bridge journey remains untouched.
- Required negative companions include zero manifest GET before create, one POST, returned-ID GET, POST/GET/copy failure, non-Slack absence, stale evidence invalidation, unknown-check fallback, optional-secret non-blocking, disabled send-test, and endpoint separation.

## AGH Impact Audit (Preflight)

- Native tools: no `agh__*` IDs, descriptors, schemas, digests, risk flags, or capability gates change; Task 04 HTTP behavior is consumed through typed Web adapters.
- Extensibility and hooks: no extension handshake, hook event, skill/capability, bundle/resource, bridge SDK, MCP sidecar, or `config.toml` change. The Web profile catalog exposes only the platform-specific public routes that already exist and never fabricates provider capability or health.
- Workspace data isolation: no new persisted datum. Manifest, verify, register, send-test, bindings, and checklist evidence remain keyed to one bridge instance whose existing scope/workspace ownership is enforced by shared handlers; client evidence is replaced/cleared on instance change and never crosses query keys.
- Official AGH skill: no new public verb/route/tool contract; inspect only for copy that becomes false when Web-created bridges now start disabled. Task 04/08 own the already-public bridge operations documentation.

## Open Risks

- Live provider accounts are unavailable. Contract-faithful MSW/Playwright responses, exact typed adapters, Storybook captures, and later Tasks 09–10 scenario QA provide the available evidence.
- The provider discovery payload has no generic setup-capability bit. Task 05 must keep its small profile catalog presentation-only and explicitly omit unsupported controls; a future generic provider setup contract would require its own approved design.
- Unrelated Daytona sidecar assets remain modified and must never be staged.

## Next

- Task 05 implementation and focused verification are complete. Update task tracking/state, create the atomic checkpoint, and continue directly into Task 06.

## Implementation and Review State

- Typed Task 04 adapters, manifest query, verify/register/send-test mutations, fixtures, and public exports are implemented with focused unit coverage.
- Create now persists `enabled:false`; Slack advances to a committed post-create manifest handoff keyed only by the returned bridge ID. Pending creation cannot be dismissed, retried, or navigated backward.
- The detail surface is split into single-purpose sub-500-line files. Setup checklist, exact inline check records, Telegram registration, dry-run, and real send-test are integrated. The former 1,105-line detail owner is now a composition shell.
- The pure setup projection has 53 focused passing cases. It validates exact check-to-slot mappings, keeps collective Teams identity global, treats required metadata omission as required, and rejects literal private/loopback/link-local/reserved IPv4/IPv6 webhook destinations consistently with the daemon's public-routability policy.
- Review found and fixed stale-evidence resurrection: evidence fingerprints bridge revision/config/lifecycle, bindings, and provider descriptor; operation epochs discard late verify/register responses after mutation, lifecycle action, instance change, or external refetch.
- Verify/register toasts now distinguish pass, warn, fail, skipped, and empty results. Telegram registration is disabled while enabled and during lifecycle work; Enable remains a daemon lifecycle action rather than being authorized by transient evidence.
- Dynamic bridge-route content is keyed by instance ID, so edit/target/delivery local state cannot leak from bridge A to bridge B. Focused route coverage proves external revision invalidation, late-response discard, truthful tones, instance remount, and Telegram's disabled registration precondition (17/17 passing).
- Existing real-daemon E2E expectations now reflect disabled creation and enable only after the required secret binding. A new three-journey mocked-daemon setup spec is written; its exact execution is pending final source/type freeze.
- Registration and verification now use separate semantic fingerprints. Registration ignores lifecycle/provider-health and survives name/routing/progress-only edits, while provider-config or binding changes clear all evidence. The canonical route regression covers `register -> enable -> verify -> edit name/progress -> refetch` with reordered JSON keys and remains complete.
- The final Slack fixtures are coherent: an enabled/ready bridge reports webhook reachability as `pass`, and the manifest handoff story renders the shared daemon-shaped `slackBridgeManifestFixture` instead of reconstructing a partial manifest.

## Final Evidence

- Consolidated focused Turbo/Vitest run: 14 Task 05 owner suites, 192/192 tests passing after the final patch.
- Focused Web typecheck passed; `oxfmt --check` and `oxlint --deny-warnings` passed for the seven final-patch files.
- Exact mocked-daemon Playwright file passed 3/3 journeys: committed Slack create/manifest handoff, exact verify cards, and dry-run/send-test endpoint separation.
- React Doctor scored 100/100 for the Task 05 React surface before the final narrow evidence patch; the final hook change is covered by the 19/19 route suite and package typecheck.
- Visual evidence was recaptured after source freeze and inspected: `/tmp/agh-ui-screenshot/task05-final/bridge-detail-configured-isolated.png` and `/tmp/agh-ui-screenshot/task05-final/bridge-slack-manifest-isolated.png`. Existing inspected failure/loading states remain under the same directory.
- Contract-parity audit returned PASS. Final deslop re-audit returned SHIP with no P1, P2, or residual slop.
- QA impact is flagged, not retested: `NB-039` was reset to `untested` for the changed dry-run presentation and `NB-052` records the new Web setup orchestrator as `untested`. The tracker validates at 434 unique 16-column rows.
- Per the user's explicit cost constraint, global Web suites and the single monorepo `make verify` remain deferred until every task, QA action, and review remediation is complete.

## Ready for Next Run

- Task 06 inherits a disabled-first setup UI and no new backend contract. It should preserve Task 05's exact bridge-instance scoping when adding durable delivery state.
- The two unrelated Daytona sidecar assets remain modified and must stay outside every workflow checkpoint.
