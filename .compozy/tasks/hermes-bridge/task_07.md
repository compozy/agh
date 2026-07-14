---
status: completed
title: bridgesdk scaffolding hoist and provider conformance suite
type: refactor
complexity: critical
---

# Task 7: bridgesdk scaffolding hoist and provider conformance suite

## Overview
Hoists duplicated provider scaffolding into bridgesdk and lands the auto-discovery
conformance matrix + docs↔code drift tests that prove every provider stays honest. Eight
providers carry near-1:1 lifecycle/reconcile/delivery-state copies plus byte-identical
`markers.go`/`main.go`; this slice consolidates them and gates the result so a 9th provider
without docs fails CI. Conformance proves the hoist — guideline forbids test-only tasks.

<critical>
- ALWAYS READ the PRD, the TechSpec, and their catalogs (`_user_stories.md`, `_tests.md`) before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>
- MUST hoist into bridgesdk: provider lifecycle loop (initialize/serve/health/shutdown glue),
  instance ownership sync, instance-config reconciliation + route-map swap, delivery-state
  store, config/path lookups, host-call retry — the blocks enumerated in analysis 06 §4.5.
- MUST leave per provider ONLY: signature verifier, inbound payload mappers, platform API
  client, delivery executors, progress renderer, formatter (post tasks 02–03/06 shapes).
- MUST replace the 8 copies of `markers.go` with one bridgesdk test-support package and the 8
  copies of `main.go` with a shared stub — zero-legacy deletes, no re-exports (SD-002).
- MUST preserve every provider's observable behavior: existing provider test suites pass with
  only import/wiring updates — behavior assertions unchanged (tests are the regression oracle;
  do not weaken them).
- MUST respect the 500-line cap in the new bridgesdk files (split by responsibility:
  lifecycle, reconcile, delivery-state, markers).
- MUST keep the import graph downward-only (bridgesdk must not import `internal/bridges`
  beyond the existing contract types — SD-008; `mage Boundaries` green).
- MUST extend the existing conformance matrix (canonical suite:
  `internal/extension/provider_conformance_matrix_integration_test.go`) to auto-discover every
  `extensions/bridges/*` provider (no manual enumeration — a new provider directory joins the
  suite automatically) and assert per provider: manifest declares `bridge.adapter` + `[bridge]`
  block; `secret_slots` match the config schema; binary builds; runtime serves the five
  bridgesdk methods; initialize/health round-trip on the fake transport.
- MUST add docs↔code drift tests as INVARIANTS (relations), not snapshots: the `index.mdx`
  provider-slots table covers every discovered provider; the Slack scope list in docs matches
  the task_04 scope constants (not old task_08); each provider has a setup section (guards
  task_08's parity from regressing). Adding provider+docs together stays green; adding only
  one side fails.
- MUST wire the suite under the integration tag in `make test-integration`.
</requirements>

## Subtasks
- [x] 7.1 bridgesdk provider-runtime design landed as small files (lifecycle, reconcile,
      delivery-state, markers test-support, main stub) — each under the 500-line cap
- [x] 7.2 Migrate slack + telegram (the two most divergent shapes) first; provider suites green
- [x] 7.3 Migrate discord, whatsapp, teams, gchat, github, linear
- [x] 7.4 Delete all duplicated scaffolding blocks + the ×8 `markers.go`/`main.go`; hard
      deletes only — no compat shims
- [x] 7.5 Auto-discovery + per-provider contract assertions in the conformance matrix suite
- [x] 7.6 Docs drift tests (slots-table completeness, Slack scope-list match vs task_04
      constants, setup-section presence) as invariants
- [x] 7.7 CI wiring: suite runs under the integration tag in `make test-integration`
- [x] 7.8 Port strongest per-provider reconcile cases into bridgesdk as the canonical home
- [x] 7.9 `mage Boundaries` green; aggregate `provider.go` line reduction ≥40%

## Implementation Details
Reference `_techspec.md` §3.6 and §4 delete targets, plus §9 verification. Runs after
provider-touching slices (tasks 01–06) so it consolidates their final shapes — including
progress renderers and edit/reply routing — instead of churning against them. Composition
(embedded runtime struct + interfaces for the platform-specific surface), not inheritance —
Hermes's god `base.py` is the negative example. Drift tests live beside the conformance suite
(`bridge_docs_conformance_test.go`); they gate a product contract (docs ARE the setup surface).
Skills: `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`,
`consolidate-test-suites`, `architectural-analysis` (optional for the hoist boundary).

### Relevant Files
- `internal/bridgesdk/{provider_runtime,reconcile,delivery_state,markers}.go` (new; names
  indicative) — hoisted scaffolding
- `extensions/bridges/*/provider.go` — reduced to platform-specific code
- `extensions/bridges/*/{markers.go,main.go}` — deleted
- `internal/extension/provider_conformance_matrix_integration_test.go` — auto-discovery
  extension
- `internal/extension/bridge_docs_conformance_test.go` — docs↔code drift invariants

### Dependent Files
- All 8 provider test suites (import/wiring updates only — behavior assertions untouched)
- `packages/site/content/runtime/core/bridges/` — read by drift tests (task_08 makes them
  green for setup sections / slots table)
- `mage Boundaries` / package import graph

### Competitor References
- `.resources/hermes/tests/gateway/test_plugin_platform_interface.py:24-214` — auto-discovery
  + parametrized contract shape
- `.resources/hermes/tests/gateway/relay/test_contract_doc_conformance.py:1-60` —
  invariant-not-snapshot doctrine
- `.resources/hermes/gateway/platforms/whatsapp_common.py:44` — shared-behavior extraction
  idea (reject the mixin/god-base mechanism)
- Negative: `.resources/hermes/gateway/platforms/base.py` (5623-line god base)

## Deliverables
- bridgesdk provider runtime; 8 slimmed providers; duplicated `markers.go`/`main.go` deleted
- Auto-discovering conformance suite over all bridge providers
- Docs↔code drift guards (invariants)
- Unit, integration, and E2E cases assigned below implemented and passing **(REQUIRED)**
- `mage Boundaries` green

## Tests

Cases assigned from `_tests.md` (remap: old task_16+14 → this task). Read full definitions
there before writing tests.

- Unit tests (suite: `internal/bridgesdk/` new runtime files — `_tests.md` §2 cases 12–13):
  - [x] Lifecycle: initialize→ready reporting, ownership-sync retry, shutdown drains cleanly
        (D7)
  - [x] Reconcile: config add/remove/path-conflict + route-map swap produce the SAME state
        transitions the retired per-provider copies produced (strongest cases ported here as
        the canonical home)
- Unit tests (suite: `extensions/bridges/*/provider_test.go` — existing, regression oracle —
  `_tests.md` §5 case 20):
  - [x] All 8 suites pass with behavior assertions UNTOUCHED (wiring-only diffs)
- Unit tests (suite: `internal/extension/bridge_docs_conformance_test.go` — `_tests.md` §5
  case 17):
  - [x] Every discovered provider appears in the `index.mdx` slots table (may stay red until
        task_08 lands setup/slots parity — red-first proof against current docs is acceptable
        until task_08; this task owns the invariant harness)
  - [x] Slack docs scope list ⊆/= the task_04 scope constants (not old task_08)
  - [x] Every discovered provider has a setup section (or per-provider page) — same red-first
        / task_08-green contract
- Integration tests (suite:
  `internal/extension/provider_conformance_matrix_integration_test.go` — `_tests.md` §5
  cases 16, 20):
  - [x] A synthetic new provider directory with a valid manifest is discovered and passes;
        removing a required secret slot from its schema fails the matching assertion
  - [x] All 8 real (migrated) providers pass initialize/health round-trip on the fake
        transport
  - [x] Conformance matrix green post-hoist; `mage Boundaries` green
- E2E tests (lane: `make test-e2e-runtime` bridge lane — `_tests.md` §6 case 7):
  - [x] Contract lane green after the bridgesdk provider-runtime migration
- Test coverage target: >=80% for touched packages
- All tests must pass under the repo gates (`-race` for Go, integration tag)

## Completion Evidence

- Shared runtime/reconcile owners are used by all eight providers; `provider.go` aggregate is
  10,521 lines versus the 17,688-line baseline (40.5% reduction), and no provider-local
  `markers.go` remains.
- Focused race coverage is 80.0% or higher for bridgesdk and every provider. Auto-discovered
  conformance passes 11/11, the focused daemon bridge E2E passes 4/4, scoped lint has zero
  issues, and package boundaries pass.
- The docs invariant harness is intentionally red-first under this task's explicit allowance:
  the Slack scope relation is green; Task 08 must add the missing slots/setup documentation
  and make the complete invariant green before Phase B ends.
- Per user direction, global test targets and `make verify` remain deferred until all tasks and
  review remediation are complete.

## Success Criteria
- Every assigned test case implemented and passing
- `find extensions/bridges -name markers.go | wc -l` = 0
- Aggregate `provider.go` line count reduced by ≥40%; a scaffolding fix now lands in exactly
  one bridgesdk file
- Adding a 9th provider directory with no docs section fails CI with a message naming the
  missing surface
- `mage Boundaries` green; coverage >=80% on touched packages
