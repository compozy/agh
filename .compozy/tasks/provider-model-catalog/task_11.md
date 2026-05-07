---
status: completed
title: "Cross-Surface Regression Hardening"
type: backend
complexity: critical
dependencies:
  - task_06
  - task_07
  - task_08
  - task_09
  - task_10
---

# Task 11: Cross-Surface Regression Hardening

## Overview
This task closes the gaps that appear only after all surfaces exist together. It verifies the hard cut, cross-surface parity, runtime fixtures, redaction, and E2E contract drift before the formal QA tail starts.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST verify no non-test old-field references for `default_model`, `supported_models`, and `supports_reasoning_effort` survived earlier hard-cut tasks.
- MUST verify config, catalog, HTTP, UDS, CLI, Host API, and web surfaces agree on the same seeded catalog state.
- MUST update deterministic E2E/runtime fixtures affected by model catalog and ACP config option changes.
- MUST add redaction assertions for source errors across logs/status/API/web-visible payloads.
- MUST add concurrency/timeout coverage for refresh paths and detached request lifetime behavior.
- MUST ensure generated docs/types and web fixtures match the final contract.
- MUST run focused gates before handing to QA planning.
</requirements>

## Subtasks
- [x] 11.1 Add residue guard assertions proving earlier tasks removed old provider model field references instead of cleaning them here.
- [x] 11.2 Add seeded end-to-end runtime fixture for catalog list/refresh/status parity.
- [x] 11.3 Add cross-surface assertions comparing CLI, HTTP, UDS, Host API, and web-consumed payloads.
- [x] 11.4 Add redaction and deterministic error tests for source failures.
- [x] 11.5 Add refresh timeout/concurrency, SQLite write-contention, and detached request-lifetime tests.
- [x] 11.6 Run focused backend/web/docs gates and fix drift before QA tail.

## Implementation Details
Follow `_techspec.md` sections `Safety Invariants`, `Testing Approach`, and `Observability`. Activate `systematic-debugging`, `no-workarounds`, `agh-test-conventions`, `testing-anti-patterns`, `deadlock-finder-and-fixer`, and `cy-final-verify` when completing.

### Relevant Files
- `internal/testutil/e2e/` - runtime harness fixtures and mock agents.
- `web/e2e/fixtures/runtime-seed.ts` - web runtime seed data.
- `web/e2e/__tests__/session-provider-override.spec.ts` - high-risk session model override flow.
- `internal/api/core/*_test.go` - cross-surface handler assertions.
- `internal/cli/*_test.go` - CLI structured output assertions.
- `internal/extension/*_test.go` - Host API/capability assertions.
- `packages/site/content/runtime/core/agents/providers.mdx` - old field text should be gone after Task 10.
- `openapi/agh.json` - generated API should be final.

### Dependent Files
- `.compozy/tasks/provider-model-catalog/qa/` - Task 12/13 will consume findings and verification evidence.
- `docs/_memory/lessons/` - only if a confirmed new durable lesson emerges; otherwise no lesson write.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - cross-surface daemon catalog authority.
- [ADR-002: Provider Model Config Hard Cut](adrs/adr-002-provider-model-config-hard-cut.md) - no old-field residue.
- [ADR-003: Extension Model Source Contract](adrs/adr-003-extension-model-source-contract.md) - extension source status and capability behavior.

### Web/Docs Impact
- `web/`: validates `web/src/systems/session`, `web/src/systems/settings`, generated types, mocks, and E2E fixtures.
- `packages/site`: validates provider/config/API/CLI/extension docs and generated CLI docs have no stale contract claims.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: verifies extension model source rows/status behave through Host API and native catalog projections.
- Agent manageability: verifies CLI/HTTP/UDS structured outputs and deterministic errors for list/refresh/status.
- Config lifecycle: verifies old keys are absent/rejected and new nested config is represented consistently in docs, settings, API, and runtime.

## Deliverables
- Old-field residue guard assertions; no planned cleanup belongs in this task.
- Cross-surface parity regression tests.
- E2E/runtime fixture updates.
- Redaction/concurrency/timeout regression tests with 80%+ relevant coverage **(REQUIRED)**.
- Focused verification evidence before QA planning **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] repository scan or targeted tests prove no production code reads old provider model fields.
  - [x] redaction helper removes API keys, OAuth tokens, and secret-shaped env values from source errors.
  - [x] refresh concurrency does not corrupt source rows/status.
  - [x] concurrent refreshes for one provider coalesce and avoid SQLite `BUSY` failures.
  - [x] concurrent refreshes across providers preserve lock fairness and deterministic row/status replacement.
  - [x] request cancellation does not cancel detached refresh work prematurely.
- Integration tests:
  - [x] seeded catalog state matches across HTTP, UDS, CLI, Host API, and web-generated types.
  - [x] old TOML keys fail while new nested config drives catalog config source rows.
  - [x] `/api/openai/v1/models` projection matches native catalog rows for the same provider filter.
  - [x] native HTTP/UDS canonical JSON bytes match for a deterministic catalog payload.
  - [x] E2E fixture for session provider/model override uses catalog + ACP config option semantics.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Focused gates pass: `go test ./internal/modelcatalog/... ./internal/store/globaldb ./internal/acp ./internal/api/... ./internal/cli ./internal/extension/...`, `make bun-typecheck`, `make bun-test`, `make web-build`, `make codegen-check`.
- Feature is ready for QA plan generation.
