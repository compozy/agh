# Task Memory: task_10.md

## Objective Snapshot
- Generated contracts/CLI reference refresh + runtime docs hard-cut: providers/config docs use the nested `[providers.<id>.models]` block; new `[model_catalog.sources.models_dev]` and provider `models.discovery` config sections; native catalog endpoints, `/api/openai/v1/models` projection, and extension `model.source` contract documented; docs tests assert there are no remaining flat `default_model` / `supported_models` / `supports_reasoning_effort` claims outside the deterministic hard-cut warnings.

## Important Decisions
- Added a new dedicated runtime page `packages/site/content/runtime/core/agents/model-catalog.mdx` (registered in `packages/site/content/runtime/core/agents/meta.json`) instead of cramming the catalog/openai surface into providers.mdx. Keeps providers.mdx focused on launch/auth and gives the catalog room to document merge priorities, refresh lifecycle, and extension contract.
- Linked `[model_catalog.sources.models_dev]` from model-catalog.mdx using slug `#modelcatalogsourcesmodelsdev` because the Fumadocs slugifier in `lib/__tests__/internal-links.test.ts` strips `_` (it is not in `\p{L}\p{N}\s-`).
- Documented agent-manageability through `agh provider models …` (singular `provider` namespace) and explicitly noted `agh models …` is out of scope for the MVP, matching the techspec/CLI choice.
- Added a focused docs vitest at `packages/site/lib/__tests__/provider-model-catalog-docs.test.ts` with 12 assertions instead of expanding `landing-truth.test.tsx`. Matchers skip lines containing "no longer / hard-cut / rejected with / are rejected" so deterministic hard-cut warning copy survives the "no old field claims" check.

## Learnings
- `make codegen` + `make cli-docs` regenerate the entire `cli-reference/` tree because the cobra→MDX renderer reformats unaligned tables (`| col | desc |` instead of padded). Those reformat-only diffs ride along with new `provider/models/*` files but are intended generated output.
- `make verify` Go race step has a pre-existing failure on this branch from Task 05: `internal/daemon TestDaemonModelCatalogWiring/ShouldCancelAndJoinRefreshWorkOnShutdown` returns `context.DeadlineExceeded` instead of the expected `context.Canceled`. Confirmed by stashing all task-10 edits (`packages/site`, `openapi/`, `web/src/generated`, `sdk/`, `internal/`) and rerunning the test — still fails. Out of scope for docs Task 10; track as Task 05/11 follow-up rather than patch it from here.

## Files / Surfaces
- New: `packages/site/content/runtime/core/agents/model-catalog.mdx`, `packages/site/lib/__tests__/provider-model-catalog-docs.test.ts`, generated `packages/site/content/runtime/cli-reference/provider/models/{index,list,refresh,status}.mdx` + `meta.json`.
- Updated: `packages/site/content/runtime/core/agents/providers.mdx`, `packages/site/content/runtime/core/agents/definitions.mdx`, `packages/site/content/runtime/core/agents/meta.json`, `packages/site/content/runtime/core/configuration/config-toml.mdx`, `packages/site/content/runtime/core/configuration/agent-md.mdx`, `packages/site/content/runtime/core/extensions/develop.mdx`.
- Generated reformat ride-alongs: nearly all `packages/site/content/runtime/cli-reference/**/*.mdx` and matching `meta.json`.

## Errors / Corrections
- First version of the docs test treated lines containing "no longer accepted" as the hard-cut marker, but the warning text wraps as `…are no longer\naccepted.` so the marker fell on the next line. Adjusted the matcher to recognize "no longer | hard-cut | rejected with | are rejected" anywhere on a line.
- Initial config.toml link used `#model_catalogsourcesmodels_dev` (with underscores). The `lib/__tests__/internal-links.test.ts` slugifier strips `_`; corrected to `#modelcatalogsourcesmodelsdev`.

## Ready for Next Run
- Task 11 (cross-surface regression hardening) can build on this docs hard cut and the new docs vitest. Daemon model-catalog test failure should be triaged before claiming Task 11 done.
