# TC-UI-004 — Generated TS types feed adapters/hooks without manual DTO duplication

- **Priority:** P2
- **Type:** UI / contract parity
- **Trace:** Task 13

## Test Steps

1. Inspect `web/src/systems/tools/**` adapters/hooks.
   - **Expected:** Imports types from `web/src/generated/agh-openapi.d.ts`; no parallel manual interfaces for tool list/info/invoke/projection.
2. Add a contract field; regenerate; web typecheck still passes once adapters consume the new field.
3. Remove a contract field; web typecheck fails until adapters are updated → behavior is desirable, not a defect.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `make bun-typecheck`
