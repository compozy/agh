# Remediation Review Prompt: API-Key Providers via Pi Harness

You are reviewing a local AGH remediation pass. Work read-only: do not edit files, do not run destructive git commands, and do not attempt to commit. You may run non-destructive inspection commands such as `git diff`, `git status --short`, `rg`, `sed`, and targeted tests if needed.

## Objective

Review whether the previous Claude Opus findings for first-class API-key/secondary providers were fixed correctly, without introducing new correctness, security, API, web, or documentation regressions.

## Repository Context

- Repository root: `/Users/pedronauck/Dev/compozy/agh`
- Accepted plan: `.codex/plans/2026-05-01-api-key-providers.md`
- Session ledger: `.codex/ledger/2026-05-01-MEMORY-api-key-providers.md`
- Original review prompt: `.compozy/tasks/providers/reviews/claude_opus_review_prompt.md`
- Original review report: `.compozy/tasks/providers/reviews/claude_opus_review.md`
- Research artifacts: `.compozy/tasks/providers/analysis/`
- Full remediation verification has passed with `make verify`.
- The worktree contains unrelated pre-existing changes. Focus on provider-related implementation and only mention unrelated changes if they create a concrete integration risk for this feature.

## Original Findings To Recheck

Re-evaluate every original finding, especially:

1. Dynamic redaction for resolved provider secret values.
2. Vault crypto/service/store test coverage and behavior.
3. `internal/session/provider_runtime.go` direct coverage and Pi runtime materialization behavior.
4. Optional missing credentials must not write stale `apiKey` entries into Pi `models.json`.
5. Builtin catalog coverage must include all new providers and defaults.
6. Daemon provider vault wiring must not fail later in confusing ways when the registry cannot support vault storage.
7. Provider credential slot overlays must not silently force incorrect credential semantics.
8. Optional `vault:` refs must not inherit parent-shell env values.
9. Web provider editor must preserve all credential slots and render aggregate credential state correctly.
10. `vault:` refs must reject arbitrary suffixes and document the strict format.
11. Docs must not need a temporary redaction caveat if dynamic redaction is now real.

## Primary Files To Inspect

Backend/config/session/vault:

- `internal/config/provider.go`
- `internal/config/merge.go`
- `internal/config/provider_test.go`
- `internal/diagnostics/redact.go`
- `internal/diagnostics/redact_test.go`
- `internal/session/session.go`
- `internal/session/manager_start.go`
- `internal/session/manager_lifecycle.go`
- `internal/session/provider_runtime.go`
- `internal/session/provider_runtime_test.go`
- `internal/vault/`
- `internal/store/globaldb/global_db_vault.go`
- `internal/store/globaldb/global_db_vault_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/provider_vault_test.go`
- `internal/testutil/e2e/runtime_harness.go`
- `internal/testutil/e2e/runtime_harness_lifecycle_test.go`

Settings/API/contracts:

- `internal/settings/collections.go`
- `internal/settings/models.go`
- `internal/settings/service.go`
- `internal/settings/service_test.go`
- `internal/api/contract/settings.go`
- `internal/api/core/conversions.go`
- `internal/api/core/settings.go`
- `internal/api/httpapi/handlers_test.go`
- `openapi/agh.json`

Web:

- `web/src/generated/agh-openapi.d.ts`
- `web/src/hooks/routes/use-settings-providers-page.ts`
- `web/src/hooks/routes/use-settings-providers-page.test.tsx`
- `web/src/systems/settings/components/provider-card.tsx`
- `web/src/routes/_app/settings/providers.tsx`
- `web/src/routes/_app/settings/-providers.test.tsx`
- `web/src/systems/session/components/session-create-dialog.tsx`
- `web/src/systems/session/components/session-create-dialog.test.tsx`

Docs/site:

- `packages/site/content/runtime/core/agents/providers.mdx`
- `packages/site/content/runtime/core/agents/spawning.mdx`
- `packages/site/content/runtime/core/configuration/config-toml.mdx`
- `packages/site/content/runtime/core/configuration/index.mdx`
- `packages/site/content/runtime/core/getting-started/installation.mdx`
- `packages/site/content/runtime/core/operations/troubleshooting.mdx`

## Review Focus

Prioritize concrete blockers and major issues:

- Secret material must not be persisted, exposed through API responses, logged, returned in diagnostic errors, or written to Pi config when absent.
- Dynamic redaction must be lifecycle-safe, concurrency-safe, and effective for resolved `env:` and `vault:` provider secrets.
- Required versus optional credential semantics must be clear and enforced consistently across config overlays, settings, session start, and Pi materialization.
- `vault:providers/<provider>/<slot>` validation must reject traversal, whitespace, broad arbitrary paths, and ambiguous values.
- Web edits must preserve all credential slots and avoid showing unsupported controls.
- Docs must accurately describe implemented runtime behavior and must not promise unimplemented provider support.
- Tests should protect the high-risk areas rather than merely snapshotting happy paths.

## Output Format

Return a concise code-review report:

- Findings first, ordered by severity.
- Each finding must include file path and line/function reference.
- For each finding, include impact and a concrete recommended fix.
- Include `No blocking findings` only if there are genuinely no correctness/security blockers.
- List non-blocking improvements separately.
- List verification gaps or additional tests separately.
- End with an explicit readiness verdict: `Ready`, `Ready with nits`, or `Not ready`.
