# TC-UI-006: Network Settings Page Validation

**Priority:** P1
**Type:** UI
**Module:** Web Settings
**Requirement:** Network settings should validate numeric fields and prevent invalid saves.

## Objective

Verify the settings network page renders runtime status, edits draft config, tracks invalid numeric inputs, and gates save actions.

## Preconditions

- Settings network route can load section data.
- Runtime status includes enabled, listener, peers, channels, queue, and worker metrics.
- Settings mutation can be mocked for success and failure.

## Test Steps

1. Open `/settings/network`.
   **Expected:** Runtime stat grid and operational link to `/network` render.
2. Toggle embedded network.
   **Expected:** Draft becomes dirty and save bar appears.
3. Enter invalid listener port or delivery field value.
   **Expected:** Field-level error appears and save is disabled.
4. Correct the value.
   **Expected:** Error clears and save can proceed.
5. Save valid changes.
   **Expected:** Mutation runs, warnings and restart banner display according to returned result.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Loading state | pending query | accessible loading status |
| Query error | settings API error | error state with retry |
| Runtime unavailable | `runtime.available=false` | status line reflects unavailable state |

## Related

- TC-UI-101
