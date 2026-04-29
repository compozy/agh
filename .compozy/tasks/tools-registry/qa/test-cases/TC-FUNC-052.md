# TC-FUNC-052 — OpenAPI and generated TypeScript contracts stay in sync

- **Priority:** P1
- **Type:** Functional / codegen
- **Trace:** Task 11

## Test Steps

1. `make codegen` regenerates `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` cleanly.
2. `make codegen-check` exits zero (no drift).
3. Generated TS types are consumed by `web/src/systems/tools/**` without manual DTO duplication.
4. Adding a contract field locally without regenerating fails `codegen-check`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `make codegen` / `make codegen-check` / `make bun-typecheck`
