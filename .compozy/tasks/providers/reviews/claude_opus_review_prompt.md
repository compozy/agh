# Review Prompt: API-Key Providers via Pi Harness

You are reviewing a local AGH implementation. Work read-only: do not edit files, do not run destructive git commands, and do not attempt to commit.

## Objective

Review the implementation for first-class API-key/secondary providers in AGH. The change adds a provider catalog, Pi/pi-acp runtime materialization, encrypted bound provider secret injection, backend settings/API contracts, web UI support, generated OpenAPI/types, and site documentation.

## Context

- Repository root: `/Users/pedronauck/Dev/compozy/agh`
- Relevant plan: `.codex/plans/2026-05-01-api-key-providers.md`
- Relevant memory ledger: `.codex/ledger/2026-05-01-MEMORY-api-key-providers.md`
- Research artifacts: `.compozy/tasks/providers/analysis/`
- Final verification already passed once with `make verify`.
- The worktree has unrelated pre-existing changes. Focus on provider-related implementation and only mention unrelated files if they create a real integration risk for this feature.

## Primary Files To Inspect

Backend/config/session/vault:

- `internal/config/provider.go`
- `internal/config/merge.go`
- `internal/config/provider_test.go`
- `internal/session/manager.go`
- `internal/session/manager_start.go`
- `internal/session/provider_runtime.go`
- `internal/vault/`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_vault.go`
- `internal/store/globaldb/global_db_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`

Settings/API/contracts:

- `internal/settings/collections.go`
- `internal/settings/models.go`
- `internal/settings/service.go`
- `internal/settings/service_test.go`
- `internal/api/contract/contract.go`
- `internal/api/contract/settings.go`
- `internal/api/core/conversions.go`
- `internal/api/core/settings.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/testutil/e2e/config_seed_test.go`
- `openapi/agh.json`

Web:

- `web/src/generated/agh-openapi.d.ts`
- `web/src/hooks/routes/use-settings-providers-page.ts`
- `web/src/hooks/routes/use-settings-providers-page.test.tsx`
- `web/src/routes/_app/settings/providers.tsx`
- `web/src/routes/_app/settings/-providers.test.tsx`
- `web/src/systems/settings/components/provider-card.tsx`
- `web/src/systems/settings/mocks/fixtures.ts`
- `web/src/systems/session/components/session-create-dialog.tsx`
- `web/src/systems/session/components/session-create-dialog.test.tsx`
- `web/src/systems/workspace/mocks/fixtures.ts`

Docs/site:

- `packages/site/content/runtime/core/agents/providers.mdx`
- `packages/site/content/runtime/core/agents/spawning.mdx`
- `packages/site/content/runtime/core/configuration/config-toml.mdx`
- `packages/site/content/runtime/core/configuration/env-vars.mdx`
- `packages/site/content/runtime/core/configuration/index.mdx`
- `packages/site/content/runtime/core/getting-started/installation.mdx`
- `packages/site/content/runtime/core/operations/troubleshooting.mdx`

## Review Focus

Prioritize bugs, security flaws, data leaks, broken runtime behavior, missing migration/test coverage, API contract inconsistencies, web UX correctness, and documentation claims that are false relative to the code.

Specific questions:

1. Is secret material ever persisted, exposed, logged, returned in API responses, or written to generated Pi config in plaintext?
2. Does provider credential resolution behave correctly for `env:` and `vault:` refs, required and optional slots, direct ACP providers, and `pi_acp` providers?
3. Does Pi runtime materialization match the likely `pi-acp` expectations and avoid corrupting global Pi config?
4. Are provider aliases/defaults/config overlay semantics correct and maintainable?
5. Are the settings API and web UI contract changes complete and compatible with generated types?
6. Are tests meaningful enough to catch regressions, especially around vault encryption, missing credentials, and provider catalog behavior?
7. Are docs accurate and not overstating unsupported MCP/verification behavior?

## Output Format

Return a concise code-review report:

- Findings first, ordered by severity.
- Each finding must include file path and line or function reference.
- For each finding, include impact and a concrete recommended fix.
- Include "No blocking findings" only if there are genuinely no correctness/security blockers.
- Then list non-blocking improvements separately.
- Then list verification gaps or additional tests you recommend.

Do not summarize the feature unless needed to explain a finding.
