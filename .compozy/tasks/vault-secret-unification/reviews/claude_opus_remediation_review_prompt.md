# Vault Secret Unification Remediation Review

Review the current working tree after remediation of the prior Opus review.

## Context

- Original plan: `.codex/plans/2026-05-01-vault-secret-unification.md`
- Session ledger: `.codex/ledger/2026-05-01-MEMORY-vault-secret-unification.md`
- Prior review prompt: `.compozy/tasks/vault-secret-unification/reviews/claude_opus_review_prompt.md`
- Fresh verification: `make verify` passes after the remediation.

## Prior Findings To Re-Check

1. `internal/extension/manager.go` hook clone must deep-copy `HookDecl.SecretEnv`.
2. MCP OAuth vault refs should use a consistent grammar.
3. Skill-projected MCP servers should reject secret-like literal `env` at resource validation/load boundaries.
4. `mcp_auth_tokens` database columns should make clear that they store vault refs, not plaintext.

## Expected Remediation

- `cloneHookDecl` deep-copies `SecretEnv`; `TestManagerCloneExtensionReturnsIsolatedSnapshot` covers it.
- MCP OAuth refs use `vault:mcp/<server>/oauth/{client-secret,access-token,refresh-token}`.
- Skill resource validation calls the same MCP server validation path and rejects `GITHUB_TOKEN` in literal `env`.
- `mcp_auth_tokens` uses `access_token_ref` and `refresh_token_ref`.
- Full `make verify` passes.

## Output Format

Return Markdown with:

- `Blockers`: concrete findings with file path and line reference, or `None`.
- `High/Medium`: concrete findings with file path and line reference, or `None`.
- `Residual Risk`: short notes.
- `Readiness`: one of `ready`, `ready-with-nits`, or `not-ready`.

Do not modify files. Do not run destructive git commands.
