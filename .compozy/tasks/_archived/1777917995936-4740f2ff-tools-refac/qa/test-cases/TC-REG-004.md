# TC-REG-004: Web `tasks` System Aligned With Autonomy Hard Cut

**Priority:** P0 (Critical)
**Type:** Regression / Web
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the regenerated `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, and `web/src/systems/tasks/mocks/fixtures.ts` no longer reference raw `claim_token`. Confirm `make bun-typecheck` and the focused Vitest lanes for `web/src/systems/{tasks,automation,settings}` pass against the new types.

## Traceability

- Tasks: task_09 (autonomy hard cut), task_11 (docs/codegen alignment).
- TechSpec: "Web/Docs Impact", "Post-Implementation Residual Checks".
- ADR: ADR-005.
- Surfaces: `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/{types.ts,mocks/fixtures.ts,components,hooks}`.

## Preconditions

- TC-REG-001 passed (codegen clean).
- Bun deps installed.

## Test Steps

1. Regenerate web types:
   ```bash
   make codegen | tee qa/logs/TC-REG-004/make-codegen.log
   ```

2. Grep for legacy fields:
   ```bash
   grep -RIn "claim_token" web/src/generated/agh-openapi.d.ts | tee qa/logs/TC-REG-004/openapi-d-ts-grep.txt
   grep -RIn "claim_token" web/src/systems/tasks/types.ts | tee qa/logs/TC-REG-004/tasks-types-grep.txt
   grep -RIn "claim_token" web/src/systems/tasks/mocks/fixtures.ts | tee qa/logs/TC-REG-004/tasks-fixtures-grep.txt
   grep -RIn "ClaimToken" web/src | tee qa/logs/TC-REG-004/claim-token-symbol-grep.txt
   ```
   - **Expected:** Zero matches except `claim_token_hash`.

3. Run monorepo typecheck (this gates the entire monorepo and is the canonical gate):
   ```bash
   make bun-typecheck | tee qa/logs/TC-REG-004/bun-typecheck.log
   ```

4. Run focused Vitest lanes:
   ```bash
   bunx vitest run web/src/systems/tasks --reporter=basic | tee qa/logs/TC-REG-004/vitest-tasks.log
   bunx vitest run web/src/systems/automation --reporter=basic | tee qa/logs/TC-REG-004/vitest-automation.log
   bunx vitest run web/src/systems/settings --reporter=basic | tee qa/logs/TC-REG-004/vitest-settings.log
   ```

5. Optional: `make bun-test` for the full monorepo run.

## Evidence To Capture

- All grep output files.
- Typecheck and Vitest logs.
- If a test fails, capture the failing assertion output verbatim and link to the underlying bug.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Mock fixture referencing legacy DTO | leftover from before hard cut | Update fixture; re-run TC-REG-004 |
| Component imports `ClaimToken` symbol | leftover | Replace with the new `run_id`-keyed shape |
| Storybook story uses old shape | old story file | Update story; re-run typecheck |

## Channels Exercised

- Generated TypeScript artifacts.
- Web Vitest lanes for the affected systems.

## Related Test Cases

- TC-SEC-001 (cross-channel claim_token sweep).
- TC-REG-001 (codegen drift).
- TC-UI-001 (UI spot-check).
