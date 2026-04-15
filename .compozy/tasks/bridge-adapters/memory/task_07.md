# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Update the web bridge management flows so create/detail screens expose provider-owned config, DM policy, and provider metadata while keeping delivery defaults limited to outbound target resolution.

## Important Decisions
- Use the generated OpenAPI bridge/provider types from task 06 as the contract source for task 07 web models.
- Treat the PRD, tech spec, and ADRs as the already-approved design for this implementation run instead of pausing for separate brainstorming approval.
- Model `provider_config` in the create flow as validated JSON text that is parsed into an object on submit; this keeps the form progressive without inventing fake provider field schemas.

## Learnings
- `web/src/generated/agh-openapi.d.ts` already includes `provider_config`, `dm_policy`, `config_schema`, and `secret_slots`; the current web bridge code simply does not project those fields into its draft helpers or UI.
- Existing test-delivery flows already read only `delivery_defaults`, which should remain unchanged while create/detail expand around provider-owned fields.
- Task-scoped coverage is best measured by explicitly including the bridge route, adapter, helper, and component files; shared UI primitives and unrelated bridge screens otherwise dilute the summary below the task target.

## Files / Surfaces
- `web/src/systems/bridges/types.ts`
- `web/src/systems/bridges/lib/bridge-drafts.ts`
- `web/src/systems/bridges/lib/bridge-drafts.test.ts`
- `web/src/systems/bridges/lib/bridge-formatters.ts`
- `web/src/systems/bridges/lib/bridge-formatters.test.ts`
- `web/src/systems/bridges/adapters/bridges-api.ts`
- `web/src/systems/bridges/components/bridge-create-dialog.tsx`
- `web/src/systems/bridges/components/bridge-detail-panel.tsx`
- `web/src/systems/bridges/components/bridge-detail-panel.test.tsx`
- `web/src/routes/_app/bridges.tsx`
- `web/src/systems/bridges/hooks/use-bridge-actions.ts`
- `web/src/systems/bridges/components/bridge-create-dialog.test.tsx`
- `web/src/systems/bridges/hooks/use-bridge-actions.test.tsx`
- `web/src/routes/_app/-bridges.test.tsx`
- `web/src/systems/bridges/adapters/bridges-api.test.ts`

## Errors / Corrections
- Pre-change gap confirmed: the create mutation currently only submits `delivery_defaults`, and the detail panel only renders delivery defaults plus generic configuration facts.
- Typecheck correction: provider requirement badges had to use the bridge `Pill` tone set (`amber`/`neutral`), not a raw `"warning"` tone.

## Ready for Next Run
- Implementation, task-scoped coverage, web gates, and full `make verify` are complete; next step is task tracking plus the final local commit.
