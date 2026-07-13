# Workflow Memory: hermes-bridge

## Current State

- Phase D round 2 returned `SHIP`, but the mandatory post-review deslop audit found branch-added cleanup error discards, duplicated browser-side webhook security policy, duplicated provider cards, and two documentation defects. The source has changed, so round-2 SHIP is stale and round 3 is mandatory.
- Per-task and remediation evidence is focused and proportional. The full runtime/Web E2E lanes and single `make verify` remain deferred until the round reaches source freeze, per user direction.
- Cleanup now uses one bridgesdk drain/close owner and explicit commit policy/evidence. Creates require a decoded remote ID; update/delete/typing/reaction mutations commit on `2xx`; GET/auth remain error-bearing. Deterministic Discord regressions prove one attempt for both create and delete when `Close` returns a transient network error, while the cleanup remains observable through adapter markers. Linear HMAC handling fails closed, `RouteTable.Replace` no longer discards an artificial error, and GitHub test shutdown reports failure. The daemon projects canonical optional `webhook_public_url`, the Web no longer parses callback security policy, generated/API types and the official skill co-ship it, provider cards share one domain composite, and the factual docs defects are corrected. The 320px dialog min-content root is fixed; six desktop/mobile captures are visually clean with teardown confirmed. Bridge payload conversions were extracted from the over-cap legacy file into a 208-line owner.

## Shared Decisions

- Bridge progress is daemon-rendered, canonically redacted, ordered with text, and presentation-only.
- Slack/Telegram/Discord default to `new` + `accumulate` with typing/reactions; other providers default off.
- Delivery defaults remain an extensible string-valued outer object with a closed typed `progress` block. Known enums canonicalize; provider strings remain opaque.
- OpenAPI remains strict. Mixed additional-property TypeScript types use an explicit AST codegen marker/transform rather than a weaker public schema.
- Lifecycle-only final ACK classification is shared by bridgesdk and extension conformance.
- Workspace-scoped descriptor metadata and explicit workspace propagation are non-negotiable for later provider rendering.
- Provider control is an ephemeral single-instance subprocess with required `service | control` purpose, exact typed method allowlist, zero Host API grants, Manager lifecycle admission/cancellation, globally bounded concurrency, and cancelable per-instance locking.
- Bound credentials may target only operator-owned provider endpoints. Mutable provider config cannot set API/OAuth/service destinations; webhook reachability is public-HTTPS-only, DNS/IP-pinned, redirect-free, and proxy-free.
- A generated secret needed by a later manual step must be supplied or explicitly disclosed once before persistence. Hidden generation is allowed only when the daemon consumes the value directly.
- Inbound edits are a typed `edit` family; reply context is bounded and cache-only. Durable Path A recovery stores checkpoint metadata and metrics, never streamed/progress text, and universally fails open with a new visible terminal error before new registrations are admitted.
- Provider runtime scaffolding is composition-based: shared lifecycle/Host/reconcile/routes/delivery/HTTP/markers/command owners with provider-specific prepare/finalize hooks. Every provider publishes routes before final probes through the same `ManagedConfigReconciler`.
- Bridge documentation now follows an operator-first Diataxis split: one common setup/orientation hub, eight dedicated provider how-tos, one day-two operations/recovery runbook, one public in-tree author tutorial, and one concise in-repo implementation checklist.

## Shared Learnings

- Provider-specific delivery defaults must survive every normalize/edit/compact round-trip and must never leak into test-delivery target overrides.
- Typed tool-result state must outrank stale legacy error fields; tool failures must not be modeled as prompt failures.
- Fake providers, subprocess integration, direct HTTP/UDS contracts, and exact daemon E2E provide faithful evidence without live bridge accounts.
- Independent deslop review found and drove fixes for lifecycle ACKs, outer-default extensibility, enum canonicalization, generated TS usability, and Web provider-default preservation.
- Google Chat's current official message cap is 32,000 bytes; the 4,096-character value in the vendored Hermes reference is stale. Task 02 delivery and Task 03 progress measurement must use the current byte cap.
- Bridge text snapshots are cumulative. Edit-capable adapters keep overflow previews to one mutable chunk and materialize ordered continuations only on terminal delivery; provider formatting must be measured before chunking.
- Explicit provider edit operations and references outrank state-based create/update inference. A provider success response without the remote ID required for an ACK is transient and must not advance state.
- Task 03 must reuse Task 02's exact wire units and formatter seams: Slack/Telegram UTF-16 units, Discord/Teams/WhatsApp Unicode code points, and Google Chat UTF-8 bytes.
- Scheduled progress must retain generation identity from timer creation through callback completion. Shutdown cancels every active run before join, and scheduled edits revalidate the throttle under the accumulator mutex rather than forcing a stale timer write.
- `progress=off` is authoritative before operational provider-config errors: an opt-out must detach pending state and ACK without platform calls even when the provider is degraded.
- Runtime E2E low-tier coverage belongs to the `^TestDaemonE2E` contract-mock lane; provider-specific API bodies remain in provider fake-server suites. One Teams-declared temporary reference adapter can prove both ordered progress and per-instance opt-in without a second daemon harness.
- Modular HTTP route files must declare their route group locally so the repository's manual-doc route inventory can prove the registered path.
- Static contract tests caught two genuine Task 04 co-ship gaps: the TypeScript runtime fixture needed the required purpose, and modular bridge subroutes needed inventory-visible group ownership.
- A shutdown admission must reject new service calls immediately. Otherwise health probes can remain green while shutdown waits for in-flight initialization, creating a deterministic coordination deadlock in subprocess conformance.
- Nominal provider coverage is not documentation parity. The vendored Hermes guides are more usable when they supply a single entry point, provider-specific journeys, observable success criteria, and recovery operations. AGH keeps its stronger typed verification/runtime truth while adopting that information architecture.
- A response-body cleanup failure is secondary only after explicit remote commit evidence. Successful status is sufficient for mutations whose remote identity is already known; creates that must return a new handle require the decoded non-empty identity. Read/auth/status/decode failures remain retryable errors rather than being hidden by cleanup reporting.

## Open Risks

- Unchanged `internal/api/httpapi/static.go` currently reports three `gosec G703` findings during an unfiltered scoped lint. Preserve it as unrelated until its owning workstream resolves it; the final global gate will expose any remaining blocker.
- Modified Daytona sidecar assets are unrelated worktree changes and must not be included in any workflow checkpoint.
- Task 09 consciously merged all 27 provisional QA seeds into the seven content-addressed Hermes scenarios and reset the affected `NB-024..NB-039` scenarios; duplicate tracker rows are no longer an open risk.
- No live Slack/Telegram/etc. accounts are available; later QA must continue using isolated fake-provider and public-surface scenarios.
- The checkpoint-only ledger intentionally does not replay multichunk text after restart. Universal visible fail-open avoids prefix duplication and stale-anchor dependence; future exact multichunk resume would require a separate explicit ordered-handle/content contract.
- `BUG-20260712-reasoning-evidence-attribution`/R-017 is fixed inside the test harness without widening production. Writer anti-forgery, subprocess propagation, exact fail-closed owner selection, concurrent shared-JSONL daemon E2E, ten stress runs, scoped vet, and an independent patch review pass are recorded; the complete runtime lane remains pending final source freeze.
- Phase D B-005 makes Telegram guided and strict-JSON setup select one closed private, ordinary-group, or forum shape per bridge. `BUG-20260713-telegram-route-shapes` remains open only for the broader one-instance alternative-shape routing contract; `NB-bridge-provider-setup` is reset to `untested` without rewriting historical QA evidence.
- The final B-002 Storybook capture proved the six daemon-owned metrics and exposed a separate 320px overflow below them. The designer-owned fix now reflows complete secret refs and provider/config metadata with zero horizontal scrolling (`318/318`), preserves desktop (`1438/1438`), passes focused Web gates, and has inspected captures plus clean teardown; `NB-web-bridge-setup` is reset to `untested` without erasing historical QA.
- R-006 terminal chrome is command-name-only by design. Canonical semantic `ToolInput` remains redacted for live/replay fingerprinting; public heuristic previews omit every argument instead of depending on an enumerable redaction taxonomy.

## Handoffs

- Task 02 provides the shared chunker, six provider delivery loops, and Slack/Telegram formatting seams.
- Task 03 provides visible progress rendering across all six chat providers, issue-provider no-op ACKs, shared scheduled lifecycle discipline, exact fake-platform contracts, and the low-tier daemon contract E2E.
- Task 04 provides Slack manifest generation, guided/headless setup, typed provider verification, doctor aggregation, Telegram registration, real send-test, hardened control isolation, public API parity, and generated contracts.
- Task 05 provides the disabled-first Web setup orchestrator, daemon-owned Slack manifest handoff, state-derived setup checklist, typed verify/register/send-test flows, progress editors, and route-local evidence isolation. `NB-039` and `NB-web-bridge-setup` are `untested` inputs to the next QA cycle after the Phase D changes.
- Task 06 provides typed Slack/Telegram edits, bounded Slack/Telegram/GChat reply context, a workspace-isolated checkpoint/metric ledger, boot admission fencing, and universal visible fail-open recovery. `NB-bridge-edit-reply` passed the Task 10 cycle; `NB-bridge-restart-recovery` is `untested` after the Phase D metrics/reconciliation changes.
- Task 07 provides shared provider lifecycle/Host/reconcile/routes/delivery/HTTP/marker/command owners, eight migrated providers, auto-discovered manifest/schema/binary/runtime conformance, and docs↔code invariants. Task 08 turned its former setup/slots red lane green. NB-029 and NB-031 are reset to `untested` for the discovered lifecycle fixes.
- Task 08 now provides exact 8/8 dedicated provider guides, an operator-first shared setup hub, a bridge operations/recovery runbook, a provider comparison, GitHub/Linear colocated READMEs, and executable author guidance from scaffold through route/send-test. The author tutorial's missing `bin/` build prerequisite and invalid empty check example were corrected. Focused evidence is green: docs conformance 4/4 under `-race`, site MDX generation/typecheck through Turbo, Oxfmt on 18 files, test-shape checker, and `git diff --check`; no global suite or `make verify` ran.
- Task 09 provides bridge personas, seven content-addressed journeys, nine immutable content-addressed charters, living-scenario mapping/reset for 17 changed/canonical behaviors, explicit reconciliation of all 27 provisional seeds, TTFM/risk/taxonomy ownership, four per-file automation candidates, and the executable Task 10 contract in `docs/qa/reports/2026-07-12-hermes-bridge-plan.md`.
- Task 10 executed the fresh isolated QA cycle, wrote evidence-backed living-scenario verdicts and a dated report, and completed clean teardown. Seven charters passed; the two Telegram setup charters are blocked by `BUG-20260713-telegram-route-shapes`.
- Task 10 preconditions found `BUG-20260712-goal-judge-fixture-model`, `BUG-20260712-reasoning-evidence-attribution`, re-found canonical `BUG-0037`, and found `BUG-20260712-bridge-e2e-retired-route`. Goal fixture negotiation, ACP diagnostics attribution, and the current Web asset/route owners are source-fixed; the fresh complete runtime/Web lanes remain final source-freeze evidence.
- Task 10 independently reproduced open `BUG-0028`, completed guided provider/CLI/HTTP/UDS, Slack progress/edit, and browser-first Slack setup journeys, and recorded provider-fake qualifications. The dated report owns all run observations; charters remain immutable.
- The final Hermes comparison returned SHIP after confirming all eight provider journeys, runnable lifecycle checkpoints, routing/progress truth, day-two recovery/rotation/rollback, and the in-tree author tutorial. The examples import the leaf bridge contract, secret input documents only stdin/file, and Discord server permissions are truthful.
