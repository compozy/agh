---
status: pending
title: Reference extensions (Go and TypeScript)
type: docs
complexity: medium
dependencies:
  - task_06
  - task_07
  - task_10
---

# Task 11: Reference extensions (Go and TypeScript)

## Overview

Build two working reference extensions that demonstrate the full extension architecture end-to-end: one in Go and one in TypeScript. The Go extension exercises the core L3 subprocess path directly. The TypeScript extension validates the `@agh/extension-sdk` package in realistic conditions. Both extensions are installed into a real AGH daemon in an integration test, which exercises the entire pipeline from manifest parsing through handshake, hook dispatch, and Host API calls.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create one Go reference extension in `sdk/examples/secret-guard/` that implements a content validator hook per `_examples.md` section 1
- MUST create one TypeScript reference extension in `sdk/examples/prompt-enhancer/` that implements a prompt enhancer hook per `_examples.md` section 4
- MUST provide a complete `extension.toml` manifest for each reference extension
- MUST include build instructions (Makefile or README) for each reference extension
- MUST write an end-to-end integration test that installs both extensions into a real AGH daemon and verifies:
  - extensions load successfully via the 6-phase pipeline
  - capability-negotiated handshake completes
  - hooks dispatch to the extensions and patches are applied
  - shutdown sequence works correctly
- MUST use the Go SDK primitives (task 02 subprocess package) or direct JSON-RPC for the Go extension
- MUST use `@agh/extension-sdk` (task 10) for the TypeScript extension
- MUST NOT introduce external dependencies beyond what the SDKs provide
</requirements>

## Subtasks
- [ ] 11.1 Create `sdk/examples/secret-guard/` Go extension with manifest, main.go, build instructions
- [ ] 11.2 Create `sdk/examples/prompt-enhancer/` TypeScript extension with manifest, index.ts, build instructions
- [ ] 11.3 Write end-to-end integration test spawning real daemon with both extensions installed
- [ ] 11.4 Verify capability enforcement end-to-end (extension with limited grants cannot call write methods)
- [ ] 11.5 Document extension author onboarding steps in each example README

## Implementation Details

New directories `sdk/examples/secret-guard/` (Go) and `sdk/examples/prompt-enhancer/` (TypeScript). New integration test in `internal/extension/integration_test.go` with build tag.

See `_examples.md` sections 1 and 4 for the full extension code. See `_protocol.md` for protocol compliance requirements.

The integration tests build the reference extension binaries as part of the test setup (`TestMain` with `go build` and `npm run build`). Use `t.TempDir()` to isolate the daemon's state.

### Relevant Files
- `internal/extension/manager.go` — Manager that loads extensions (task 06)
- `internal/extension/host_api.go` — Host API handler (task 07)
- `sdk/typescript/` — TypeScript SDK (task 10)
- `internal/subprocess/` — Go subprocess primitives (task 02)
- `_examples.md` — Reference extension code and manifests

### Dependent Files
- None. This is the validation leaf of the task graph.

### Related ADRs
- [ADR-001: Two-Tier Extension Model](adrs/adr-001.md) — Validates both Go and TypeScript subprocess paths
- [ADR-003: Capability-Scoped Security Model](adrs/adr-003.md) — Validates end-to-end capability enforcement
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — Validates resource/capability/action loading

## Deliverables
- Working Go reference extension at `sdk/examples/secret-guard/`
- Working TypeScript reference extension at `sdk/examples/prompt-enhancer/`
- End-to-end integration test in `internal/extension/integration_test.go` with build tag `//go:build integration`
- Per-extension README with build and install instructions
- Unit tests with 80%+ coverage for any shared test helpers **(REQUIRED)**
- Integration tests for end-to-end extension flow **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Shared test helpers compile and run without requiring a daemon
- Integration tests:
  - [ ] Go secret-guard extension loads, handshakes, rejects prompt containing `sk-abc123`
  - [ ] Go secret-guard extension accepts safe prompt and returns `allow: true`
  - [ ] TypeScript prompt-enhancer loads, handshakes, injects workspace context into prompt
  - [ ] Both extensions coexist in the same daemon simultaneously
  - [ ] Shutdown sequence gracefully stops both extensions
  - [ ] Extension with limited capabilities cannot call `sessions/create` via Host API (capability denied)
  - [ ] Extension subprocess crash triggers automatic restart with backoff
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Both reference extensions build and install into a real daemon
- Full end-to-end path validated from manifest parse through hook dispatch through shutdown
- `make verify` passes
- `make test-integration` passes
