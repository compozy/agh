# BUG-005: Workspace-scoped provider models are not projected into the session catalog

## Status

Open

## Severity

High

## Source

- Task 13 real-scenario QA execution.
- Required command: `make test-e2e-web`.
- Flow: `web/e2e/__tests__/session-provider-override.spec.ts`.

## Reproduction

1. Create a workspace `.agh/config.toml` with a workspace-only provider:

   ```toml
   [providers.qa-browser-override]
   command = "..."

   [providers.qa-browser-override.models]
   default = "qa-browser-model"

   [[providers.qa-browser-override.models.curated]]
   id = "qa-browser-model"
   display_name = "QA Browser Model"
   supports_reasoning = true
   reasoning_efforts = ["low", "medium", "high"]
   default_reasoning_effort = "medium"
   ```

2. Resolve the workspace and open the new-session dialog for an agent in that workspace.
3. Select `qa-browser-override`.
4. Trigger the UI catalog refresh.
5. Open the model selector.

## Expected Behavior

The session-create model selector should expose the workspace provider's configured model metadata, including `qa-browser-model` and its reasoning effort defaults, or the API should explicitly document and expose a workspace-aware catalog path.

## Actual Behavior

The provider is visible because workspace resolution includes workspace-scoped provider options, but the model selector remains empty after refresh:

```text
No catalog models — type a model name to continue.
```

The operator can continue by typing a manual model name, but configured workspace model metadata is not available to the daemon-backed catalog selection path.

## Root Cause

`internal/daemon/model_catalog.go` registers the config source from daemon startup config (`state.cfg.Providers`). The HTTP model catalog APIs are provider-scoped only (`/api/providers/{provider_id}/models`) and do not carry workspace context, while the new-session dialog's provider options are workspace-aware. That leaves workspace-only provider model metadata outside the daemon-owned model catalog projection.

## Impact

- Workspace-only provider config can provide a provider option without matching catalog model metadata.
- New-session model selection loses curated display names, availability hints, and reasoning effort defaults for workspace provider overlays.
- Operators must manually type the model name even when it is configured in the workspace.

## Current QA Handling

The provider override E2E now validates the supported fallback path by refreshing the catalog, confirming the empty state, typing `qa-browser-model` manually, and asserting the session create request persists that model without unsupported reasoning metadata.

## Suggested Fix

Design and implement a workspace-aware catalog projection, for example by adding a workspace query parameter to native model catalog list/refresh/status APIs or by including workspace provider model metadata in the session-create data path. This should be handled as a follow-up design because it affects daemon API contracts, OpenAPI/types, web query keys, and catalog source identity semantics.

## Evidence

- Browser lane rerun evidence: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/make-test-e2e-web-rerun-2.log`
