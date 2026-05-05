# TC-FUNC-058 — Generated CLI/API references regenerate cleanly

- **Priority:** P1
- **Type:** Documentation / generated references
- **Trace:** Task 14

## Test Steps

1. `make cli-docs`.
   - **Expected:** Regenerates `packages/site/content/runtime/cli-reference/**` with `agh tool list/search/info/invoke`, `agh toolsets list/info`, `agh tool mcp` (operator-internal command), and existing `agh mcp auth login/status/logout`.
2. `cd packages/site && bun run source:generate`.
   - **Expected:** Passes.
3. `bun run typecheck` and `bun run build` pass.
4. Re-run `make cli-docs`; diff is empty.
5. API reference entry pages reference the regenerated `openapi/agh.json` and link to canonical endpoint groups.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `make cli-docs && cd packages/site && bun run source:generate && bun run typecheck && bun run build`
