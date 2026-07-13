# Workflow Memory: hermes-bridge

## Current State

- Phase B, Tasks 01–06 are checkpointed at `66880e5`, `b103482`, `2be5ddf`, `d362403`, `c8ebbcaf`, and `2b00ab3`. Task 07 implementation, focused verification, contract audit, deslop audit, and QA impact tracking are complete; its atomic checkpoint is the next loop action.
- Per-task evidence is focused and proportional. Global tests and the single `make verify` are deferred until all ten tasks, QA, and review remediation are complete per user direction.

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

## Shared Learnings

- Provider-specific delivery defaults must survive every normalize/edit/compact round-trip and must never leak into test-delivery target overrides.
- Typed tool-result state must outrank stale legacy error fields; tool failures must not be modeled as prompt failures.
- Fake providers, subprocess integration, direct HTTP/UDS contracts, and exact daemon E2E provide faithful evidence without live bridge accounts.
- Independent deslop review found and drove fixes for lifecycle ACKs, outer-default extensibility, enum canonicalization, generated TS usability, and Web provider-default preservation.
- Google Chat's current official message cap is 32,000 bytes; the 4,096-character value in the vendored Hermes reference is stale. Task 02 delivery and Task 03 progress measurement must use the current byte cap.
- Bridge text snapshots are cumulative. Edit-capable adapters keep overflow previews to one mutable chunk and materialize ordered continuations only on terminal delivery; provider formatting must be measured before chunking.
- Explicit provider edit operations and references outrank state-based create/update inference. A provider success response without the remote ID required for an ACK is transient and must not advance state.
- Task 03 must reuse Task 02's exact wire units and formatter seams: Slack/Discord/Teams/WhatsApp code points, Telegram UTF-16 units, and Google Chat UTF-8 bytes.
- Scheduled progress must retain generation identity from timer creation through callback completion. Shutdown cancels every active run before join, and scheduled edits revalidate the throttle under the accumulator mutex rather than forcing a stale timer write.
- `progress=off` is authoritative before operational provider-config errors: an opt-out must detach pending state and ACK without platform calls even when the provider is degraded.
- Runtime E2E low-tier coverage belongs to the `^TestDaemonE2E` contract-mock lane; provider-specific API bodies remain in provider fake-server suites. One Teams-declared temporary reference adapter can prove both ordered progress and per-instance opt-in without a second daemon harness.
- Modular HTTP route files must declare their route group locally so the repository's manual-doc route inventory can prove the registered path.
- Static contract tests caught two genuine Task 04 co-ship gaps: the TypeScript runtime fixture needed the required purpose, and modular bridge subroutes needed inventory-visible group ownership.
- A shutdown admission must reject new service calls immediately. Otherwise health probes can remain green while shutdown waits for in-flight initialization, creating a deterministic coordination deadlock in subprocess conformance.

## Open Risks

- Unchanged `internal/api/httpapi/static.go` currently reports three `gosec G703` findings during an unfiltered scoped lint. Preserve it as unrelated until its owning workstream resolves it; the final global gate will expose any remaining blocker.
- Modified Daytona sidecar assets are unrelated worktree changes and must not be included in any workflow checkpoint.
- Task 09 QA seeds must be reminted above `NB-054` before tracker insertion.
- No live Slack/Telegram/etc. accounts are available; later QA must continue using isolated fake-provider and public-surface scenarios.
- The checkpoint-only ledger intentionally does not replay multichunk text after restart. Universal visible fail-open avoids prefix duplication and stale-anchor dependence; future exact multichunk resume would require a separate explicit ordered-handle/content contract.

## Handoffs

- Task 02 provides the shared chunker, six provider delivery loops, and Slack/Telegram formatting seams.
- Task 03 provides visible progress rendering across all six chat providers, issue-provider no-op ACKs, shared scheduled lifecycle discipline, exact fake-platform contracts, and the low-tier daemon contract E2E.
- Task 04 provides Slack manifest generation, guided/headless setup, typed provider verification, doctor aggregation, Telegram registration, real send-test, hardened control isolation, public API parity, and generated contracts.
- Task 05 provides the disabled-first Web setup orchestrator, daemon-owned Slack manifest handoff, state-derived setup checklist, typed verify/register/send-test flows, progress editors, and route-local evidence isolation. `NB-039` and `NB-052` are `untested` inputs to the QA tail.
- Task 06 provides typed Slack/Telegram edits, bounded Slack/Telegram/GChat reply context, a workspace-isolated checkpoint/metric ledger, boot admission fencing, and universal visible fail-open recovery. `NB-053` and `NB-054` are `untested` inputs to the QA tail.
- Task 07 provides shared provider lifecycle/Host/reconcile/routes/delivery/HTTP/marker/command owners, eight migrated providers, auto-discovered manifest/schema/binary/runtime conformance, and docs↔code invariants. Task 08 owns the currently red setup/slots parity. NB-029 and NB-031 are reset to `untested` for the discovered lifecycle fixes.
- Task 09/10 own living QA planning/execution and tracker retests.
