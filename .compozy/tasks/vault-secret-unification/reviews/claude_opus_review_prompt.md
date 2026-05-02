# Vault Secret Unification Implementation Review

Review the current working tree for the vault secret unification implementation.

## Context

- Plan: `.codex/plans/2026-05-01-vault-secret-unification.md`
- Session ledger: `.codex/ledger/2026-05-01-MEMORY-vault-secret-unification.md`
- Verification evidence: `make verify` passes after the implementation.
- The repository is intentionally in a dirty worktree. Ignore unrelated pre-existing edits outside this task, especially site/logo/copy changes that are not part of the secret unification surface.

## Intended Implementation

The accepted plan hard-cuts durable secrets to `internal/vault` as the canonical storage and validation authority:

- `env:NAME` remains operator-managed lookup.
- `vault:<namespace>/...` is AGH-managed encrypted storage.
- Durable public fields use `secret_ref`; ambiguous legacy names like `api_key_env`, `vault_ref`, `webhook_secret_env`, and `client_secret_env` should not remain on active surfaces.
- Public APIs return status/metadata only; plaintext secret values may only cross write-only mutation or internal launch/materialization boundaries.
- Every resolved secret value must be registered with dynamic redaction.

Implemented scope includes providers, bridges, automation webhook triggers, MCP OAuth/client/server secret env, hooks, extensions, skills, sandbox profiles, settings/API/CLI/native tools, web UI, generated contracts, and docs.

## Review Focus

Please perform a security and correctness review of the implementation. Prioritize blockers over style.

Look specifically for:

1. Any durable secret still persisted outside `internal/vault` or exposed through public API/settings responses.
2. Any lingering active public field name that should have been hard-cut to `secret_ref`.
3. Any runtime materialization path that resolves a secret but does not register dynamic redaction.
4. Any validation gap where secret-like env keys are still allowed in literal `env` maps instead of `secret_env`.
5. Any migration/schema/codegen inconsistency caused by the hard cut.
6. Any web/API mismatch where the UI sends stale fields or loses additional credential slots.

## Output Format

Return Markdown with:

- `Blockers`: concrete findings with file path and line reference, or `None`.
- `High/Medium`: concrete findings with file path and line reference, or `None`.
- `Residual Risk`: short notes on what you could not fully validate.
- `Readiness`: one of `ready`, `ready-with-nits`, or `not-ready`.

Do not modify files. Do not run destructive git commands.
