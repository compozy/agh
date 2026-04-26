## TC-AUTO-002: Agent Contracts, OpenAPI, And Token Redaction Parity

**Priority:** P1 (High)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify autonomy DTOs, OpenAPI registration, generated web TypeScript contracts, and read-model
conversion preserve safe channel and lease metadata while keeping raw claim tokens confined to the
synchronous claim response.

### Traceability

- Task: task_02, Agent Contract DTOs And OpenAPI Parity.
- TechSpec: API Endpoints, Task-Channel Coordination Contract, Impact Analysis generated contracts.
- ADR: ADR-002, ADR-003, ADR-006, ADR-011, ADR-012.
- Resource lesson: Multica typed client references require generated contract parity rather than local type guesses.
- Surfaces: `internal/api/contract`, `internal/api/spec`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, web task/session types.

### Preconditions

- Generated artifacts are present from task_02 and current with the repository source.
- Test fixtures include a claimed run, a read-model run, a session lineage row, and a channel-bound run.

### Test Steps

1. Run contract/OpenAPI tests for autonomy agent endpoints and schemas.
   - **Expected:** `/agent/me`, `/agent/context`, `/agent/channels`, `/agent/tasks/*`, `/agent/spawn`, coordinator config, lineage, and channel metadata schemas are registered with required fields.

2. Convert task run/session models into public read DTOs.
   - **Expected:** Read models include safe lease state or `claim_token_hash` when needed, never raw `claim_token`.

3. Convert a successful claim response.
   - **Expected:** The synchronous claim response contains raw `claim_token` once, plus safe coordination channel display metadata for channel-bound runs.

4. Inspect channel DTOs and metadata schema.
   - **Expected:** Task/run/workflow/channel correlation fields are typed, MVP `message_kind` values are constrained, and raw `claim_token` metadata is rejected.

5. Run generated contract checks and web typecheck.
   - **Expected:** `openapi/agh.json`, generated TypeScript, task types, session types, and fixtures compile without `any`, non-null assertions, or local duplicate DTOs.

### Evidence To Capture

- `qa/logs/TC-AUTO-002/go-test-contract.log`
- `qa/logs/TC-AUTO-002/openapi-redaction-check.log`
- `qa/logs/TC-AUTO-002/codegen-check.log`
- `qa/logs/TC-AUTO-002/web-typecheck.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Unclaimed run | no token hash | DTO omits raw token and represents no active lease |
| Channel-bound claim | run has channel ID | Claim response includes channel metadata |
| Non-channel run | no channel ID | Claim response omits channel cleanly |
| Malicious metadata | `{"claim_token":"secret"}` | Contract validation rejects or sanitizes |

### Related Test Cases

- TC-AUTO-006: Store redaction and schema.
- TC-AUTO-015: Web consumers render generated fields.
