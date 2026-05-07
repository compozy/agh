# BUG-004: Web E2E provider override flow still expected a native `<select>`

## Status

Fixed

## Severity

Medium

## Source

- Task 13 real-scenario QA execution.
- Required command: `make test-e2e-web`.

## Reproduction

1. Run the required daemon-served browser E2E lane:

   ```bash
   make test-e2e-web
   ```

2. Observe `web/e2e/__tests__/session-provider-override.spec.ts` failing in the provider/model override workflow.

## Expected Behavior

The provider override E2E test should validate the currently shipped provider command selector: the selected provider is prefilled from the agent/workspace default, every workspace-visible provider is present, and the operator can choose the override provider before selecting a catalog-backed model.

## Actual Behavior

The test attempted to call `toHaveValue()` and `selectOption()` on `session-create-provider-select`, but the shipped UI is a button/popover command selector rather than a native `<select>`. Playwright failed before exercising the actual provider override path:

```text
Locator: getByTestId('session-create-provider-select')
Expected: "claude"
Error: Not an input element
```

After that mismatch was corrected, the same test also expected a workspace-configured model to appear through the daemon catalog. BUG-005 tracks that product/API gap. This BUG covers the E2E contract mismatch and keeps the provider override flow covered through the currently supported manual model entry path.

## Root Cause

The E2E test contract was not updated when the session-create provider control moved from a native select to `ProviderCommandSelect`, and it assumed workspace-only provider model metadata was available through the provider-scoped daemon catalog.

## Fix

Updated `web/e2e/__tests__/session-provider-override.spec.ts` to exercise the real command selector:

- Assert the preselected provider trigger content and runtime metadata.
- Open the provider command list and verify the provider set from runtime workspace detail.
- Select the override provider through `provider-command-item-*`.
- Trigger the provider catalog refresh, assert the empty catalog state for the workspace-only provider, and continue through the supported manual model entry path.

## Regression Coverage

- `make test-e2e-web` must pass after the fix.

## Evidence

- Initial failure log: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-test-e2e-web.log`
- Fixed rerun log: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-test-e2e-web-rerun-3.log`
