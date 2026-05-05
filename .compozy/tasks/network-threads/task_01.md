---
status: completed
title: RFC, Glossary, and Protocol Hard Cut
type: docs
complexity: high
dependencies: []
---

# Task 01: RFC, Glossary, and Protocol Hard Cut

## Overview

Rewrite the active AGH Network protocol vocabulary before implementation begins so every later task works from the same conversation model. This task makes `public_thread`, `direct_room`, `surface`, and `work_id` normative and deletes active `interaction_id` and `kind:"direct"` semantics from supported protocol docs.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-002, ADR-003, `docs/_memory/spec-authoring-playbook.md`, and `docs/_memory/glossary.md` before editing.
- REFERENCE TECHSPEC for protocol rules; do not invent compatibility behavior.
- FOCUS ON WHAT must be true across active docs: channel-scoped threads, two-party direct rooms, work lifecycle, reply edges, trace correlation, and trust signing.
- TESTS REQUIRED for docs/examples scans and RFC 004 signed-field examples.
- NO WORKAROUNDS: active docs must not keep `interaction_id`, `kind:"direct"`, or "direct as message kind" language.
</critical>

<requirements>
- MUST activate `documentation-writer`, `copywriting`, and `cy-spec-preflight` before public docs edits.
- MUST update RFC 003 to define `surface:"thread"|"direct"`, `thread_id`, `direct_id`, and `work_id`.
- MUST update RFC 004 to include `surface`, `thread_id`, `direct_id`, and `work_id` in verified signed content when present.
- MUST update the glossary with `public_thread`, `direct_room`, `work_id`, and the reduced message-kind set.
- MUST clearly distinguish `direct_room` restricted visibility from cryptographic privacy.
- MUST preserve NATS transport terminology only where it is explicitly transport-scoped and not confused with `surface:"direct"`.
</requirements>

## Subtasks

- [x] 1.1 Rewrite RFC 003 envelope, validation, routing, lifecycle, and examples.
- [x] 1.2 Rewrite RFC 004 trust, signed-field, canonicalization, and NATS examples.
- [x] 1.3 Update `docs/_memory/glossary.md` with canonical terms and deleted terminology.
- [x] 1.4 Add docs checks that active protocol examples no longer teach `interaction_id` or `kind:"direct"`.
- [x] 1.5 Record any remaining archived references as historical only, not supported behavior.

## Implementation Details

This is a docs-first hard cut. It gives implementation agents the authoritative vocabulary before code changes fan out across runtime, store, web, tools, and docs.

### Relevant Files

- `docs/rfcs/003_agh-network-v0.md` - active protocol definition.
- `docs/rfcs/004_agh-network-v1.md` - verified/trust extension over protocol fields.
- `docs/_memory/glossary.md` - canonical repo vocabulary.
- `.compozy/tasks/network-threads/_techspec.md` - normative implementation design.
- `.compozy/tasks/network-threads/adrs/adr-001.md` - public threads vs direct rooms.
- `.compozy/tasks/network-threads/adrs/adr-002.md` - `interaction_id` to `work_id`.
- `.compozy/tasks/network-threads/adrs/adr-003.md` - direct as surface, not kind.

### Dependent Files

- `packages/site/content/protocol/*` - later site docs must mirror this vocabulary in task_16.
- `internal/network/envelope.go` - task_02 implements these fields in runtime.
- `internal/api/contract/contract.go` - task_08 exposes these fields publicly.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - defines conversation containers.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - defines lifecycle naming.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - defines wire vocabulary.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: document the protocol as implementable outside AGH; no AGH-only requirement may be introduced.
- Agent manageability: define fields that CLI, HTTP, UDS, native tools, and Host API must expose in later tasks.
- Config lifecycle: no new config keys; document that `[network]` enablement remains the gate.

### Web/Docs Impact

- Web impact: no UI code changes in this task; later generated web contracts must match this vocabulary.
- Docs impact: this task owns RFC and glossary changes; task_16 owns site/runtime/API/CLI docs.

## Deliverables

- Updated RFC 003 and RFC 004.
- Updated glossary entries and message-kind list.
- Active examples using `surface`, `thread_id`, `direct_id`, and `work_id`.
- Docs scan evidence proving active docs no longer advertise `interaction_id` or `kind:"direct"`.

## Tests

- Unit tests:
  - [x] Active RFC examples use `surface:"thread"` or `surface:"direct"` for conversation-bearing messages.
  - [x] Active RFC examples use `work_id` only for lifecycle-bearing work.
  - [x] Active RFC examples do not use `interaction_id` or `kind:"direct"`.
  - [x] RFC 004 examples include new signed fields when present.
- Integration tests:
  - [x] Docs validation command used by the repo passes for changed RFC/glossary files.
  - [x] `rg` scans confirm stale terms exist only in archived/historical artifacts or peer-review inputs.
- Test coverage target: docs validation coverage for every changed active doc.
- All tests must pass.

## Verification Evidence

- `bunx vitest run packages/site/lib/protocol-rfc-hard-cut.test.ts` passed 1 file / 4 tests.
- `bunx oxfmt --check docs/rfcs/003_agh-network-v0.md docs/rfcs/004_agh-network-v1.md docs/_memory/glossary.md packages/site/lib/protocol-rfc-hard-cut.test.ts` passed.
- `bun run typecheck` in `packages/site` passed.
- Active RFC/glossary stale-term scan returned no matches:
  `rg -n 'interaction_id|kind:"direct"|...` over `docs/rfcs/003_agh-network-v0.md`, `docs/rfcs/004_agh-network-v1.md`, and `docs/_memory/glossary.md`.
- Full `make verify` passed after one unrelated transient `internal/session` retry; final run ended with `DONE 8066 tests` and `OK: all package boundaries respected`.

## Success Criteria

- Active protocol docs define public threads, direct rooms, and work lifecycle without ambiguity.
- No supported active doc teaches `interaction_id`, `kind:"direct"`, or direct rooms as cryptographic privacy.
- Later tasks can cite the RFC and ADRs without re-litigating vocabulary.
