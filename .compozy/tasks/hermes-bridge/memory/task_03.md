# Task Memory: task_03

## Objective Snapshot

- Render daemon-projected `ToolProgress` through `ProgressAccumulator` on the six chat providers, preserve provider routing/format/length contracts, keep the two issue providers side-effect-free, and prove exact payloads with focused fake transports.

## Preflight Decisions

- Slack, Telegram, and Discord use resolved default-on `new + accumulate` progress with typing and reactions; Teams, Google Chat, and WhatsApp remain default-off and become active solely through the instance-owned `delivery_defaults.progress` block.
- Providers consume the already rendered, canonically redacted, workspace-scoped `DeliveryEvent.Progress`; they must not inspect raw tool input or rebuild labels/previews.
- `ProgressAccumulator` remains the authority for accumulate/separate grouping, repeat collapse, overflow, content reset, typing, and reactions. Provider code supplies platform sinks and lifecycle wiring.
- Task 02's wire contracts are reused unchanged: Slack/Discord/Teams/WhatsApp count Unicode code points, Telegram counts UTF-16 code units, and Google Chat counts UTF-8 bytes. Slack and Telegram progress text uses their existing outbound formatter seams before measurement and API delivery.
- Progress and final text share the provider delivery state keyed by delivery ID. Normal content calls `OnContent` before final delivery so pending progress flushes and typing clears without allowing a progress remote ID to become the textual ACK handle.
- Google Chat v1 uses text message create/patch, not CardV2. WhatsApp is append-only sparse/separate. GitHub and Linear return lifecycle ACKs without platform calls.
- Per-task verification stays focused. Global suites and the single `make verify` remain deferred to the workflow tail by user direction.

## Test Placement

- Provider API payload/routing/mode/affordance invariants belong to each provider's existing `provider_test.go` or `provider_delivery_test.go`; no duplicate standalone suite.
- Shared accumulator scheduling or repeat-collapse behavior belongs to the existing `internal/bridgesdk/progress_test.go` only if the shared API must change.
- Bridge contract ordering belongs to the existing runtime/provider delivery contract lane; no new parallel E2E harness.
- Invariant: a provider receives one ordered progress lifecycle and maps it to the exact platform APIs without changing transcript or daemon contracts.

## Current State

- Complete. Slack, Telegram, and Discord render default-on accumulated progress with provider formatting, typing/reaction affordances, rate-limit-aware edits, text-boundary cleanup, and terminal/shutdown close outside provider mutexes.
- Complete. Teams, Google Chat, and WhatsApp remain default-off before API creation and render only after instance opt-in. Teams/GChat update one status bubble; WhatsApp defaults to sparse `new + separate` posts. Mode-off transitions cancel pending dispatchers even when GChat/WhatsApp provider configuration is invalid.
- Complete. GitHub and Linear retain no production progress handler; exact runtime-peer tests prove empty lifecycle ACKs and zero platform client calls.
- The shared dispatcher identity-tracks every scheduled run until cancellation/completion, prevents stale generations from clearing newer state, cancels all timers/active contexts before shutdown join, and revalidates throttle timing under the accumulator mutex. Deterministic RED/GREEN regressions cover active cancellation, `Stop=false` generation gaps, and stale-timer edits.
- `OnContent` closes a successfully flushed bubble even if typing cleanup fails, preserving the cleanup error/state for retry without reusing the old progress bubble.
- Google Chat message edits use the official `PATCH` transport with `updateMask=text`; a canonical transport test failed red under PUT and passes green with PATCH.
- The daemon E2E is now `TestDaemonE2EBridgeIngressCreatesAndReusesRouteThroughOptedInLowTierContractMock`. A temporary Teams-declared contract mock proves low-tier default-off, per-instance `all + accumulate` opt-in, six ordered/redacted progress events, monotonic indexes, empty progress ACK handles, separate final delivery, and route reuse.
- Public behavior is documented in all six chat-provider READMEs and the lean official AGH runtime reference. QA impact is flagged as `NB-provider-progress-rendering` (`untested`) in the living scenario tree.
- Fresh focused `-race` coverage: bridgesdk 81.3744%, Slack 81.3187%, Telegram 80.2074%, Discord 82.5439%, Teams 80.0138%, GChat 80.8033%, WhatsApp 80.7136%. Exact GitHub/Linear no-op tests, the low-tier extension integration, and the daemon E2E pass.
- Scoped lint reports zero new issues (the four daemon gosec findings are unchanged baseline outside this diff); scoped vet, gofmt, `git diff --check`, file caps, QA CSV validation, cross-provider review, and deslop audit all pass. Both read-only audits returned SHIP.
- Safe scoped checkpoint `2be5ddf` contains the 44 Task 03 files and excludes both unrelated Daytona sidecar assets.

## Open Risks

- Unrelated Daytona sidecar assets remain modified and must never be staged in this checkpoint.
- Global suites and the single `make verify` remain deferred until all workflow tasks, QA, and review remediation are complete per user direction.

## Next

- Emit the exact continuous-loop iteration summary, immediately detect Task 04, and continue without pausing.
