# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 02 foundation now exists in code: config/home-path surfaces, embedded transport primitives, and dual-path network audit persistence are implemented. Daemon boot wiring still belongs to the later network-manager/boot tasks.
- Task 03 now exists in code: `internal/network` owns runtime peer presence and message routing foundations, including greet/leave/heartbeat/whois handling, sender-side directed-send preflight, and replay-aware inbound routing.
- Task 04 now exists in code: optional session `Space` metadata round-trips through session create/resume/runtime state, persisted `meta.json`, the global session index, daemon session contracts, and CLI `session new/list/stop/resume` flows.
- Task 05 now exists in code: `internal/session` carries daemon-local `TurnSource` metadata through `PromptWithOpts` / `PromptNetwork`, and `internal/acp` enforces network-turn write/terminal guardrails with allowlisted `agh network` subcommands plus `network_owned` terminal tracking.
- Task 06 now exists in code: `internal/network/delivery.go` owns bounded per-session inbound queues, per-session workers, safe inbound prompt rendering, and turn-end-triggered draining via `session.Manager.SetTurnEndNotifier` plus `Manager.IsPrompting`.
- Task 07 now exists in code: `internal/network/manager.go` owns transport/router/presence/delivery/audit composition, daemon boot now inserts `bootNetwork` between `bootRuntime` and `bootHooks`, shutdown tears network down after sessions and before hooks, and daemon info/status surfaces expose safe network listener/status diagnostics without broker credentials.
- Task 08 now exists in code: shared network DTOs, `/api/network/*` daemon endpoints, and `agh network {status,peers,spaces,send,inbox}` now expose the booted network runtime through daemon-owned contracts, while preserving optional correlation and AGH workflow/handoff `ext` metadata.
- Task 09 now exists in code: bundled `internal/skills/bundled/skills/agh-network/SKILL.md` content is embedded, `internal/session/startSession` appends that bundled guidance after prompt assembly and before ACP start only when `Session.Space` is non-empty, and resume paths re-inject the same guidance from persisted `meta.Space`.

## Shared Decisions
- Network audit persistence follows the existing `internal/store` pattern with a shared `store.NetworkAuditEntry` type and `globaldb` write/list methods, while `internal/network/audit.go` owns sink normalization and file mirroring.
- Embedded transport is exposed as `internal/network.NewTransport(ctx, config.NetworkConfig, ...)`, keeping the broker token inside the transport runtime only and publishing no credential path through `HomePaths`.
- Network-delivered prompts should enter sessions through `PromptNetwork()` / `PromptWithOpts(..., TurnSourceNetwork)` instead of overloading the existing user-turn `Prompt()` path.
- ACP network-turn enforcement depends only on session/ACP runtime metadata, not on `internal/network` imports: file writes are blocked, terminal creation is structurally allowlisted to `agh network {send,peers,spaces,status,inbox}`, and only `network_owned` terminals remain accessible during network turns.
- Turn-end delivery wakeups must happen after prompt teardown clears the active turn source; later tasks should reuse `session.Manager.SetTurnEndNotifier` rather than re-hooking the earlier `dispatchTurnEnd()` path.
- Daemon/runtime consumers should depend on the boot-owned `core.NetworkService` surface plus the late-bound `session.Manager.SetNetworkPeerLifecycle` / `SetTurnEndNotifier` seams, not on direct constructor coupling between `internal/session` and `internal/network`.
- Network control-plane callers should keep using the contract/core conversion layer on top of `core.NetworkService`; later tasks should extend that surface instead of reaching around it to `internal/network` internals.

## Shared Learnings
- The touched-package unit coverage currently meets the task target: `internal/config` 84.0%, `internal/store/globaldb` 80.5%, `internal/network` 80.0%.
- Later network tasks can rely on `PeerRegistry` and `Router` as the canonical runtime surfaces for presence and routing; those tasks should compose these types instead of duplicating peer freshness, whois resolution, or direct-send rejection rules.
- Later daemon, prompt, and network-manager tasks can consume `Session.Space` directly from canonical session metadata/state; no separate transport-specific opt-in store is needed.
- The touched-package unit coverage currently meets the task target for prompt/ACP guardrails too: `internal/session` 82.7% and `internal/acp` 82.7%.
- The touched-package unit coverage currently meets the task target for delivery too: `internal/session` 81.9% and `internal/network` 81.2% after task 06.
- Task 07 verification finished cleanly with touched-package coverage at `internal/session` 81.9%, `internal/network` 80.4%, `internal/daemon` 80.5%, and `internal/api/core` 80.1%, followed by a passing `make verify`.
- Contract changes on the daemon API surface must be followed by `make codegen` before the final `make verify`, because the repo treats stale `openapi/agh.json` and generated SDK types as a hard verification failure.
- Task 09 verification finished with touched-package unit coverage at `internal/skills/bundled` 86.7% and `internal/session` 81.9%, plus a passing targeted resume integration test and a passing `make verify`.
- Task 10 hardened session recovery semantics: crash-classified resumes may intentionally preserve repaired `agent_crashed` stop metadata while active, and `StopWithCause()` now waits for `proc.Done()` before finalizing stop metadata so resume/cleanup flows do not race the watcher path.
- Task 10 hardened network shutdown accounting: interrupted in-flight prompt drains now log `network.message.delivery_interrupted`, do not increment delivered metrics, and shutdown diagnostics report queued plus in-flight pending work.

## Open Risks
- No currently tracked open risks for the AGH Network task chain after task 10 hardening.

## Handoffs
- Task 07 or the boot-integration task should consume `internal/network/transport.go` and `internal/network/audit.go` instead of re-creating transport/audit foundations.
- Task 06 and Task 07 should consume `internal/network/peer.go` and `internal/network/router.go` as the runtime source of truth for peer visibility, direct routing, and inbound replay handling.
- Task 05, Task 06, and Task 07 should read `Session.Space` from the session manager/global session index rather than re-parsing metadata files or introducing duplicate opt-in state.
- Task 07 and later delivery workers should call `session.Manager.PromptNetwork()` and rely on `TurnSourceNetwork` instead of formatting network turns as ordinary user prompts.
- Task 07 should compose `internal/network/delivery.go` rather than replacing it; reuse `session.Manager.SetTurnEndNotifier`, `Manager.IsPrompting`, and the queue/worker behavior already covered by task 06 tests.
- Task 08 should use the already-wired daemon/API contract surfaces (`RuntimeDeps.Network`, `core.NetworkService`, daemon status payloads, and persisted `daemon.json` network info) instead of inventing a second diagnostics or service-discovery path.
- Task 09 and later network UX tasks should build on the verified `agh network` command tree, the matching daemon endpoints, and the optional workflow/handoff metadata propagation that task 08 already exposes.
