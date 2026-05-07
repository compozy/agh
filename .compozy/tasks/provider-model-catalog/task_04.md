---
status: completed
title: "Live Provider Discovery Sources"
type: backend
complexity: high
dependencies:
  - task_03
---

# Task 4: Live Provider Discovery Sources

## Overview
This task adds side-effect-free live model discovery sources for provider/runtime integrations. It deliberately keeps live discovery outside ACP session creation and records unavailable discovery paths as source status instead of blocking sessions.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add live provider source adapters for Codex/OpenAI, Claude/Anthropic, Gemini, OpenRouter, Vercel AI Gateway, Ollama, and OpenCode where side-effect-free discovery is available.
- MUST add explicit fail-closed source status for OpenClaw, Hermes, and Pi when no configured safe discovery command or endpoint exists.
- MUST consume `providers.<id>.models.discovery` command/endpoint/enabled/timeout config for OpenClaw, Hermes, Pi, and any other provider that needs explicit safe discovery wiring.
- MUST use provider effective auth/home/env policy for live discovery.
- MUST serialize/coalesce refresh subprocess/provider-home work per `provider_id` before touching operator `HOME`.
- MUST apply explicit timeouts to all subprocess and HTTP discovery calls.
- MUST NOT create, load, mutate, or stop ACP sessions for model discovery.
- MUST return catalog rows/status that integrate with `internal/modelcatalog` merge semantics.
</requirements>

## Subtasks
- [x] 4.1 Add live source registration and provider mapping for the core providers named in the TechSpec.
- [x] 4.2 Implement safe HTTP/subprocess discovery wrappers with explicit timeout and env/home policy.
- [x] 4.3 Add configured discovery and fail-closed status behavior for OpenClaw, Hermes, and Pi.
- [x] 4.4 Normalize live provider rows into catalog source rows without inventing unknown reasoning levels.
- [x] 4.5 Add fake HTTP/subprocess tests for success, timeout, auth/env policy, unavailable discovery, per-provider refresh coalescing, and no-ACP-session behavior.

## Implementation Details
Follow `_techspec.md` sections `Live Provider Sources` and `Safety Invariants`. Use Harnss and Paperclip references for provider-specific discovery behavior, but keep AGH's provider contracts and env/home policy.

### Relevant Files
- `internal/modelcatalog/` - live source types and registration.
- `internal/config/provider.go` - provider runtime/auth/home/env config.
- `internal/providerenv` - provider environment/home policy helpers.
- `internal/session/provider_runtime.go` - existing provider env handling patterns.
- `.resources/harnss/shared/lib/codex-helpers.ts` - Codex model list selection reference.
- `.resources/harnss/electron/src/ipc/claude-sessions.ts` - Claude supported models cache/reference behavior.
- `.resources/harnss/src/lib/model-utils.ts` - Harnss model utility behavior used by the TechSpec evidence.
- `.resources/harnss/src/types/window.d.ts` - IPC/config type reference for provider discovery and session controls.
- `.resources/paperclip/packages/adapters/opencode-local/src/server/models.ts` - OpenCode discovery/validation reference.
- `.resources/paperclip/packages/adapters/opencode-local/src/server/test.ts` - provider model validation status reference.
- `.resources/paperclip/packages/adapters/acpx-local/src/server/execute.ts` - ACPX adapter execution behavior and no-fake-session boundary reference.

### Dependent Files
- `internal/daemon/` - Task 05 wires sources into the daemon.
- `internal/api/core` - Task 07 exposes live source status.
- `internal/testutil` - may host fake subprocess/http helpers if package-local helpers are insufficient.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - live sources feed daemon-owned catalog rows.

### Web/Docs Impact
- `web/`: none directly in this task - status is exposed through Task 07 and displayed in Task 09.
- `packages/site`: none directly in this task - live discovery behavior is documented in Task 10.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: establishes built-in source behavior that extension sources must match later.
- Agent manageability: no direct surface yet; status rows must be suitable for CLI/HTTP/UDS source-status output.
- Config lifecycle: consumes `providers.<id>.models.discovery.enabled`, `.command`, `.endpoint`, and `.timeout`; no old model keys may be used.

## Deliverables
- Live provider source adapters and safe source registration.
- Fail-closed status for providers without safe discovery.
- Fake HTTP/subprocess tests and no-ACP-session assertions **(REQUIRED)**.
- Unit tests with 80%+ coverage for provider source behavior **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] Codex/OpenAI source maps successful list output into catalog rows.
  - [x] Claude/Anthropic source handles successful supported-models response and unavailable runtime.
  - [x] OpenRouter and Vercel gateway sources preserve provider/model IDs.
  - [x] Ollama/OpenCode source parses model list output and records unavailable command status.
  - [x] OpenClaw/Hermes/Pi return clear source status when no discovery path is configured.
  - [x] configured OpenClaw/Hermes/Pi discovery command or endpoint is used only when enabled.
  - [x] concurrent refreshes for the same provider coalesce and do not double-fork against the same operator `HOME`.
  - [x] discovery timeout records failure status without blocking indefinitely.
  - [x] secret-shaped env values are redacted from source errors.
- Integration tests:
  - [x] live source refresh uses effective provider env/home policy.
  - [x] no live discovery path calls ACP `session/new`, `session/load`, or `session/set_*`.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `go test ./internal/modelcatalog/...` passes.
- Session creation remains independent from live discovery success or failure.
