# Memory v2 Slice 1 Traceability Matrix

**qa-output-path:** `.compozy/tasks/mem-v2`
**Status:** Planning complete, not executed
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

This matrix maps tasks 01-24 to QA cases and regression hot spots. Every completed implementation task maps to at least one release scenario, every public surface is represented, and the open search-visibility risk is a P0 scenario.

## Task To Test Cases

| Task | Implemented Slice | Test Cases | Regression Hot Spots |
|---|---|---|---|
| task_01 | Memory contract extraction and hard cut | TC-INT-001, TC-INT-002, TC-INT-003, TC-SEC-001 | Contract DTO drift, legacy field/route reintroduction, raw/internal replay fields in public payloads |
| task_02 | Schema and workspace DB identity | TC-INT-001, TC-INT-002, TC-SCEN-002 | `workspace_id` stability, per-workspace DB migrations, path-move orphaning |
| task_03 | Atomic store, workspacedb, replay core | TC-INT-001, TC-SCEN-001 | pending decisions replay, atomic writes, reindex/search visibility |
| task_04 | Scan policy and prompt assets | TC-INT-001, TC-SEC-001 | prompt-injection rejects, WHAT_NOT_TO_SAVE policy, prompt version drift |
| task_05 | Write controller and decisions WAL | TC-SCEN-001, TC-INT-001, TC-INT-003 | single write path, idempotency keys, WAL-before-mutation, revert |
| task_06 | Deterministic recall, signals, shadow rules | TC-SCEN-001, TC-INT-002, TC-SEC-001 | FTS/trigram recall, signal recorder drops/failures, scope shadowing, `_system` exclusion |
| task_07 | Local provider and registry surface | TC-INT-004, TC-INT-003 | provider registry collision, bundled local provider fallback, provider-first recall |
| task_08 | Frozen snapshot and prompt assembly | TC-SCEN-002, TC-SEC-001 | no mid-session mutation, reload next boot only, sub-agent read-only inherited snapshot |
| task_09 | Memory observability and SSE hygiene | TC-SEC-001, TC-INT-003 | `memory_events` fan-in, `<memory-context>` scrub, durable append before broadcast |
| task_10 | Extractor hook, inbox, runtime queue | TC-INT-005, TC-SEC-001 | persisted-message hook, bounded queue, `_inbox`, DLQ replay, sub-agent skip |
| task_11 | Dreaming runtime and promotion gates | TC-INT-005, TC-INT-002 | signal gate, `promoted_at`, `_system/dreaming`, dream DLQ retry |
| task_12 | Session lineage and ledger materialization | TC-SCEN-002, TC-UI-003 | `ledger.jsonl`, parent/root session metadata, unbound partition, forensic-only semantics |
| task_13 | Config and settings backend | TC-UI-002, TC-REG-001 | defaults, validation, overlay merge, settings DTO parity, readonly daemon paths |
| task_14 | Public memory contract surface | TC-INT-003, TC-UI-001, TC-UI-002, TC-UI-003 | generated operation IDs, redaction-safe payloads, hard-cut old route shapes |
| task_15 | Codegen and generated consumer refresh | TC-REG-001, TC-UI-001, TC-UI-002, TC-UI-003 | OpenAPI/TS drift, handwritten DTO mirrors |
| task_16 | HTTP and UDS route parity | TC-INT-003, TC-SCEN-001 | transport route drift, unsupported route envelopes, status/body assertions |
| task_17 | CLI memory hard cut | TC-SCEN-001, TC-INT-003, TC-REG-001 | `show`/`dream trigger` hard cut, structured output, old verb absence |
| task_18 | Native tools and extension host memory surfaces | TC-INT-003, TC-INT-004, TC-SEC-001 | final `agh__memory_*` IDs, policy-gated writes, Host API recall fallback |
| task_19 | Daemon wiring and boundary registration | TC-SCEN-001, TC-SCEN-002, TC-INT-004, TC-INT-005 | composition-root ownership, lifecycle shutdown, boundaries |
| task_20 | Web Knowledge surface | TC-UI-001, TC-REG-001 | generated selector use, server-backed search, controller-backed edit/delete, decision history |
| task_21 | Web Memory Settings surface | TC-UI-002, TC-REG-001 | generated settings payload, mutable vs readonly fields, decimal controls, dream trigger |
| task_22 | Web Session Inspector memory surface | TC-SCEN-002, TC-UI-003 | session ledger query gate, read-only forensic fields, 404 unavailable state |
| task_23 | Runtime memory and configuration docs | TC-REG-001, TC-SEC-001 | narrative docs truth, old surface absence, config/doc vocabulary |
| task_24 | CLI/API reference and discoverability co-ship | TC-REG-001, TC-INT-003 | generated CLI/API reference truth, docs discovery, manual shell examples |

## Test Case To Source Authorities

| Case | Priority | Source Authorities | Surfaces |
|---|---|---|---|
| TC-SCEN-001 | P0 | TechSpec Safety Invariants 1-4, ADR-001, ADR-009, tasks 03/05/06/16/17/19 | CLI, HTTP, UDS, DB, search, events |
| TC-SCEN-002 | P0 | TechSpec Safety Invariants 5-8, ADR-002, ADR-004, ADR-006, tasks 08/12/19/22 | sessions, prompt snapshot, sub-agent, ledger, web inspector |
| TC-INT-001 | P0 | ADR-001, ADR-003, ADR-009, tasks 03/04/05 | controller, WAL, replay, revert, atomic file writes |
| TC-INT-002 | P0 | ADR-002, ADR-011, tasks 06/11 | recall, signals, shadowing, freshness, dreaming promotions |
| TC-INT-003 | P0 | Agent Manageability Plan, tasks 14/16/17/18/24 | CLI, HTTP, UDS, native tools, generated references |
| TC-INT-004 | P1 | ADR-008, tasks 07/18/19 | provider registry, local provider, extension host |
| TC-INT-005 | P1 | ADR-005, ADR-010, tasks 10/11/19 | extractor, dreaming, `_inbox`, DLQs, shutdown |
| TC-UI-001 | P0 | Web/Docs Impact, tasks 15/20/24 | Knowledge UI, adapters, hooks, MSW fixtures, decision panel |
| TC-UI-002 | P0 | Config Lifecycle, tasks 13/21/23/24 | settings backend, settings UI, readonly fields, config docs |
| TC-UI-003 | P1 | Session ledger, tasks 12/22 | session route, inspector, ledger adapter/hook |
| TC-SEC-001 | P0 | Safety Invariants 6-9, Security Invariants, ADR-005, tasks 04/09/10/14/23 | redaction, `_system`, SSE, public payloads, docs |
| TC-REG-001 | P0 | Test Plan verification commands, tasks 15/23/24/25 | codegen, CLI docs, API docs, site tests, QA artifacts |

## Surface Coverage Audit

| Surface Family | Required By | Verified By |
|---|---|---|
| Controller writes | tasks 04-05, 17-18 | TC-SCEN-001, TC-INT-001, TC-INT-003 |
| Decision WAL and replay | tasks 03, 05 | TC-INT-001 |
| Recall/search | task_06 | TC-SCEN-001, TC-INT-002 |
| Recall signals and dreaming promotion | tasks 06, 11 | TC-INT-002, TC-INT-005 |
| Workspace identity | tasks 02-03 | TC-SCEN-002, TC-INT-002 |
| Frozen prompt snapshot | task_08 | TC-SCEN-002 |
| Extractor and inbox | task_10 | TC-INT-005 |
| Session ledger | task_12 | TC-SCEN-002, TC-UI-003 |
| Config lifecycle | tasks 13, 21, 23 | TC-UI-002, TC-REG-001 |
| HTTP/UDS parity | task_16 | TC-INT-003, TC-SCEN-001 |
| CLI hard cut | tasks 17, 24 | TC-INT-003, TC-REG-001 |
| Native tools | task_18 | TC-INT-003, TC-INT-004 |
| Extension host/provider | tasks 07, 18, 19 | TC-INT-004 |
| Web Knowledge | task_20 | TC-UI-001 |
| Web Settings | task_21 | TC-UI-002 |
| Web Session Inspector | task_22 | TC-UI-003 |
| Runtime docs | tasks 23-24 | TC-REG-001 |
| Redaction and prompt-safety | tasks 04, 09, 14, 23 | TC-SEC-001 |
| Concurrency/shutdown | tasks 03, 10, 11, 19 | TC-INT-001, TC-INT-005 |

## Required Negative Assertions

- No `agh memory read` command or generated reference.
- No `agh memory consolidate` command or generated reference.
- No `GET /api/memory/search` route; search is `POST /api/memory/search`.
- No `PUT /api/memory/{filename}` current route; create is `POST /api/memory`, edit is `PATCH /api/memory/{filename}`.
- No old builtin IDs `memory_read`, `memory_history`, `agh__memory_read`, or `agh__memory_history`.
- No `[memory.v2]` config namespace.
- No raw decision replay content, raw LLM traces, tokens, or prompt-only memory in public payloads.
- No `_system/` file returned by recall unless an explicitly supported include-system path is under test.
- No UI editor/promote/replay controls in the Session Inspector memory panel.

