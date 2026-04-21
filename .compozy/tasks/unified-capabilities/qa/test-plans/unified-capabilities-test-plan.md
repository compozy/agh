# Unified Capabilities QA Test Plan

- Feature: Unified capabilities across runtime, network, API, `web/`, and `packages/site`
- Planned by: task_09
- Created: 2026-04-20
- Execution owner: task_10 using `qa-output-path=.compozy/tasks/unified-capabilities`

## Executive Summary

This QA plan defines the execution matrix that proves AGH now exposes one unified capability model instead of a `capability + recipe` split. The plan focuses on the real seams changed by tasks 01-08: canonical capability authoring and digesting, `kind:"capability"` transfer and lifecycle behavior, discovery/API contract coherence, frontend peer-detail rendering, and public protocol/runtime documentation consistency.

The plan intentionally avoids generic smoke coverage. Every P0 and P1 case in this package is tied to a concrete seam changed by the unification and maps back to the TechSpec, ADRs, or implementation tasks that introduced the behavior.

## Objectives

1. Prove authored capability catalogs still normalize correctly across supported layouts while the runtime owns `digest`.
2. Prove `kind:"capability"` replaced `recipe` on the wire without regressing validation, delivery, or lifecycle semantics.
3. Prove brief discovery, rich discovery, peer details, and daemon API payloads expose one coherent typed capability model.
4. Prove the `web/` network surface renders unified capabilities from the typed backend contract without recipe-era assumptions.
5. Prove `packages/site` protocol and runtime docs teach the same single-concept model documented in RFC 003 and `docs/agents/capabilities.md`.
6. Leave task_10 with stable artifact paths for screenshots, issues, and final verification reporting.

## Scope

### In Scope

- Backend capability schema, normalization, validation, and digest invariants from task_01.
- Capability wire kind replacement and validation behavior from task_02.
- Capability delivery, interaction opening, and terminal lifecycle behavior from task_03.
- Discovery, peer detail, HTTP/UDS contract, and filtering/size behavior from task_04.
- `web/` typed client, route-level view model, and peer-detail UI behavior from task_06.
- `packages/site` protocol reference, examples, runtime capability docs, and nav metadata from tasks_07-08.
- QA artifact layout under `.compozy/tasks/unified-capabilities/qa/`.

### Out of Scope

- Implementing new runtime or UI behavior outside the seams already changed by tasks 01-08.
- Non-capability network features unrelated to the unification.
- Visual redesign work not needed to verify capability-related network UX.
- Live bug fixing in this task; execution and remediation belong to task_10.

## Strategy and Approach

- Run task_10 against this artifact set without changing `qa-output-path`.
- Use the P0/P1 manual cases under `qa/test-cases/` as the authoritative execution seed.
- Start with backend invariants, then discovery/API coherence, then operator-facing UI, then public docs.
- Capture evidence only under this artifact root:
  - `qa/issues/BUG-*.md`
  - `qa/screenshots/<TC-ID>-<slug>.<png|jpg>`
  - `qa/verification-report.md`
- Treat any surviving steady-state `recipe` behavior, payload key, label, or documentation claim as a regression unless explicitly historical.

## Coverage Matrix

| Seam to Prove | Priority | Evidence Type | Task / Rule Traceability | Planned Cases |
| --- | --- | --- | --- | --- |
| Canonical schema, digest stability, and optional no-catalog behavior | P0 | Backend tests, fixture inspection, runtime load output | task_01, ADR-002, TechSpec: Data Models + Testing Approach | `TC-INT-001` |
| `kind:"capability"` validation and hard rejection of legacy `recipe` | P0 | Envelope validation, decode/encode tests, router rejection evidence | task_02, ADR-001, ADR-003, TechSpec: Core Interfaces + Data Models | `TC-INT-002` |
| Capability transfer delivery and lifecycle continuity | P0 | Router/delivery/lifecycle integration evidence | task_03, ADR-003, TechSpec: Testing Approach | `TC-INT-003` |
| Brief/rich discovery, peer details, and typed daemon API alignment | P0 | HTTP/UDS payload inspection, whois evidence, size/filter checks | task_04, ADR-001/002/003, TechSpec: API Endpoints + Testing Approach | `TC-INT-004` |
| `web/` typed-client and peer-detail UX alignment | P1 | Route/component regression tests, browser/manual screenshots | task_06, task_04, ADR-001/002/003 | `TC-UI-001` |
| Public protocol reference and example coherence | P1 | Content review, nav validation, `site-build` output | task_07, task_05, ADR-001, ADR-003 | `TC-REG-001` |
| Runtime capability docs and repo-guide coherence | P1 | Content review, nav validation, `site-build` output | task_08, task_05, ADR-001, ADR-002 | `TC-REG-002` |

## Environment Matrix

| Surface | Required Environment | Primary Checks | Execution Notes | Output Artifacts |
| --- | --- | --- | --- | --- |
| Backend runtime + network | Local repo checkout with Go toolchain and temp agent fixtures | `make verify`, targeted Go package/test runs around `internal/config`, `internal/session`, `internal/network`, `internal/api/*` | Prefer existing integration tests and real temp directories over ad-hoc scripts | `qa/verification-report.md`, `qa/issues/BUG-*.md` |
| Daemon API / UDS | Local daemon or test harness able to exercise peer list/detail and `whois` flows | HTTP/UDS peer payload inspection plus targeted tests | Compare brief and rich capability shapes against typed payloads, not raw API `ext` blobs | `qa/verification-report.md`, request/response snippets in issue files |
| Web network UX | Local daemon + web dev server or existing route/component regressions; browsers at 1280 / 768 / 375 widths | `make web-lint`, `make web-typecheck`, relevant web tests, browser/manual verification | Capture peer-detail screenshots only for capability-related states | `qa/screenshots/TC-UI-001-*.png`, `qa/issues/BUG-*.md` |
| Site docs | `packages/site` buildable with Bun | `make site-build` plus manual source review | Validate nav, broken links, and recipe-free steady-state wording | `qa/verification-report.md`, optional screenshots for broken rendering |

## Entry Criteria

- Tasks 01-08 are implemented on the branch and their relevant files are present.
- The shared QA artifact root exists at `.compozy/tasks/unified-capabilities/qa/`.
- Task_10 can write to `qa/issues/`, `qa/screenshots/`, and `qa/verification-report.md`.
- Repository verification commands are available: `make verify`, `make web-lint`, `make web-typecheck`, `make web-test`, `make site-build`.
- The executor has read this plan, the regression suite, the manual cases, `_techspec.md`, and ADRs 001-003.

## Exit Criteria

- All P0 cases pass.
- At least 90% of P1 cases pass, with any failure documented in `qa/issues/BUG-*.md` and referenced from `qa/verification-report.md`.
- No critical or high-severity open bug remains against the unified-capabilities seams.
- Fresh evidence exists for backend, API, web, and docs surfaces; parser-only proof is insufficient.
- `qa/verification-report.md` summarizes the executed lanes, case outcomes, issues, screenshots, and rerun verification commands.
- Final verification reruns after the last fix include `make verify`, `make web-lint`, `make web-typecheck`, and the relevant site/web test/build lanes touched by fixes.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Canonical digest mismatches only appear after transfer or cross-format loading | Medium | High | Make schema/digest coverage a P0 gate and compare equivalent TOML/JSON fixtures plus mutated fields. |
| `recipe` survives in a narrow validation or helper path even though main flows use `capability` | Medium | High | Use P0 transfer-validation coverage that explicitly injects legacy `recipe` envelopes and malformed capability bodies. |
| Rich discovery and API payloads drift apart, especially when filtered or partially known | Medium | High | Compare `greet`, `whois`, peer list, and peer detail evidence in one P0 case and verify typed payloads on HTTP/UDS. |
| `web/` still assumes recipe-era fields or reads raw `ext` data indirectly | Medium | Medium | Use a peer-detail P1 case with brief-only and rich-catalog data, plus loading/error/empty states and responsive screenshots. |
| Site docs stay internally inconsistent even if single pages were updated | Medium | Medium | Cross-check source pages against RFC 003 and runtime guide, and require `make site-build` in task_10. |
| Task_10 writes evidence outside the agreed artifact root | Low | Medium | Keep all path expectations explicit in this plan and the regression suite; reserve `issues/`, `screenshots/`, and `verification-report.md`. |

## Artifact Layout and Naming

All planning and execution artifacts for this feature live under:

```text
.compozy/tasks/unified-capabilities/qa/
├── test-plans/
│   ├── unified-capabilities-test-plan.md
│   └── unified-capabilities-regression.md
├── test-cases/
│   ├── TC-INT-001.md
│   ├── TC-INT-002.md
│   ├── TC-INT-003.md
│   ├── TC-INT-004.md
│   ├── TC-UI-001.md
│   ├── TC-REG-001.md
│   └── TC-REG-002.md
├── issues/
│   └── BUG-*.md
├── screenshots/
│   └── <TC-ID>-<slug>.<png|jpg>
└── verification-report.md
```

Naming rules for task_10:

- Screenshots: prefix with the case ID, for example `TC-UI-001-peer-detail-desktop.png`.
- Bugs: use `BUG-###.md` and link the originating case ID in the report.
- Verification report: one file at `qa/verification-report.md` for the full execution summary.

## Timeline and Deliverables

1. Pre-flight: read this plan, the regression suite, and all referenced cases.
2. Baseline: run the repo-level verification gate and capture the initial health state.
3. Smoke lane: execute the highest-risk unified-capability seams first and stop on any P0 failure.
4. Targeted lane: complete the remaining P1 coverage across UI and docs.
5. Full lane: rerun the full verification/build/test gates after the final fix set and publish the verification report.

Deliverables handed to task_10:

- This plan: `qa/test-plans/unified-capabilities-test-plan.md`
- Regression execution order: `qa/test-plans/unified-capabilities-regression.md`
- Manual cases: `qa/test-cases/TC-*.md`
- Reserved artifact paths: `qa/issues/`, `qa/screenshots/`, `qa/verification-report.md`
