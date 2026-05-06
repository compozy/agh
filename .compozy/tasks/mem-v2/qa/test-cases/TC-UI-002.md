# TC-UI-002: Web Memory Settings And Config Lifecycle

**Priority:** P0
**Type:** UI
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify Memory Settings renders the backend Memory v2 config payload, permits only mutable fields, validates bounded numeric/decimal inputs, and stays aligned with config docs and runtime defaults.

## Preconditions

- [ ] Isolated daemon exposes settings memory endpoint.
- [ ] Web dev server uses the isolated API proxy target.
- [ ] Runtime docs and `internal/config/config.go` are available for truth checks.

## Test Steps

1. **Run focused settings tests**
   - Input: `cd web && bunx vitest run src/routes/_app/settings/-memory.test.tsx src/hooks/routes/use-settings-memory-page.test.tsx`
   - **Expected:** Route and hook tests pass against generated settings payloads.

2. **Open Memory Settings page**
   - Input: browser route for `/settings/memory`.
   - **Expected:** Sections cover controller, provider, recall, decisions, extractor, dream, session ledger, daily logs, file caps, and workspace identity.

3. **Validate mutable field**
   - Input: change an allowed field such as recall top-K or controller mode.
   - **Expected:** PATCH payload matches generated `OperationRequestBody` shape and backend accepts valid value.

4. **Validate readonly daemon-managed fields**
   - Input: inspect `inbox_path`, `dlq_path`, `ledger_root`, `unbound_partition`, daily archive paths, `workspace.toml_path`, and policy allow-origins.
   - **Expected:** They render read-only and are not sent as mutable changes.

5. **Validate decimal fields**
   - Input: change recall/dream weights through decimal controls.
   - **Expected:** Values preserve decimal precision and invalid out-of-range values produce UI/backend validation errors without coercion.

6. **Trigger dream action**
   - Input: click Trigger dream action when enabled.
   - **Expected:** UI calls the final dream trigger endpoint and reports `Dream triggered` or `Dream not triggered` based on daemon response.

7. **Docs/config truth check**
   - Input: run focused site tests for config docs.
   - **Expected:** Docs list the same `[memory.*]` keys and no `[memory.v2]` namespace.

## Evidence To Capture

- Focused web and site test logs.
- Browser screenshots.
- Settings GET/PATCH payloads.
- Validation error payloads.

