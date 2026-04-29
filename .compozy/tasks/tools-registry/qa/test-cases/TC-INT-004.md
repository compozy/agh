# TC-INT-004 — `make codegen` co-ships OpenAPI + web TS contracts

- **Priority:** P1
- **Type:** Integration / codegen
- **Trace:** Task 11

## Test Steps

1. Modify a contract DTO in `internal/api/contract`.
2. Run `make codegen`; re-run `make codegen-check`.
   - **Expected:** Without regeneration, `codegen-check` fails; after regeneration, both `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` reflect the change and `codegen-check` passes.
3. Web build with regenerated TS types passes.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `make codegen && make codegen-check && make bun-typecheck && make web-build`
