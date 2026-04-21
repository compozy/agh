# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task_01 through task_03 now treat the session provider as durable state across runtime metadata, on-disk `session.json`, and the global session index. Observer reconcile repairs inactive legacy blank-provider metadata once before indexing, then persists the repaired provider into both storage layers.
- Task_04 now exposes that persisted provider through every explicit session create/read surface: shared API contracts, HTTP, UDS, CLI, extension Host API, and generated OpenAPI/TypeScript artifacts.
- Task_05 now exposes workspace-scoped provider picker data on `WorkspaceDetailPayload.providers`, assembled from the resolved workspace config in stable sorted order, and automatic internal session creators explicitly pass `Provider: ""` to stay on agent defaults.
- Task_06 now routes every web session-create entrypoint through a dialog that prefills agent/workspace/provider and submits the selected provider via the existing `createSession` contract; the chat header surfaces the effective provider badge and the session route renders a dedicated inline resume-failure panel that shows the session id and missing provider when the backend rejects a persisted provider.

## Shared Decisions
- Session startup and resume must resolve the effective runtime with `Config.ResolveSessionAgent(...)` after startup prompt assembly so provider-owned runtime fields and prompt mutations stay coherent together.
- Inactive legacy session metadata with a blank `provider` is repaired from the resolved agent default during metadata read/resume preparation instead of allowing silent runtime fallback later.
- Provider availability is now part of resume infrastructure validation. If a persisted provider cannot be resolved, resume fails explicitly.

## Shared Learnings
- `provider` now round-trips through `session.Info`, `Session.Meta()`, stopped-session status/list query assembly, observer reconciliation helpers, and on-disk `session.json`.
- The global SQLite `sessions` table now persists `provider`, and both in-place column migration and copy-style rebuild paths preserve it.
- Global session register/list/get/scan/reconcile paths now round-trip `store.SessionInfo.Provider`, so downstream API/CLI layers can treat the index as authoritative provider storage.
- `contract.SessionPayload.provider` is now a required generated field. Downstream typed consumers, fixtures, and tests must populate it explicitly instead of assuming session payloads omit provider.
- Workspace detail responses are now the backend source of truth for the provider picker. The web client should consume `WorkspaceDetailPayload.providers` directly instead of reconstructing provider options from scattered config assumptions.

## Open Risks
- Legacy stopped sessions only converge after a resume or observer reconcile touches them. Task_08 QA should still cover migration plus reconcile on pre-task_03 data.

## Handoffs
- Task_04 can project `session.Info.Provider` outward without adding new runtime resolution logic; create/status/list/resume and the global index now treat provider as authoritative state.
- Task_06 can consume generated session payloads with `provider` directly from the checked-in OpenAPI/TypeScript artifacts and use `WorkspaceDetailPayload.providers` for the creation dialog; no extra web-only shim or frontend-side provider discovery should be added.
- Task_07 QA planning should map every sidebar agent `+` entrypoint to the dialog flow and include a dedicated regression case for the inline resume-failure panel when the persisted provider is no longer visible in workspace config.
- Task_08 QA execution should verify provider parity across CLI, HTTP, UDS, and Host API explicit session surfaces and prove the web dialog end-to-end including the provider picker, the submit payload, and the inline resume-failure panel.
