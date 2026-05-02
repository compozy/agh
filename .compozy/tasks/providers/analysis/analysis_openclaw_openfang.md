# OpenClaw and OpenFang Provider/Vault Analysis

## Sources inspected

- `.resources/openclaw`
- `.resources/openfang`
- QMD page: `qmd://openfang/wiki/concepts/credential-vault-internals.md`

## Findings

- OpenClaw separates public provider/model identity from agent runtime or harness selection.
- OpenClaw uses provider catalog/plugin concepts, `provider/model` references, `SecretRef`, auth profile storage, and runtime auth plans.
- OpenClaw treats Pi as an embeddable harness option instead of forcing every provider into a native adapter.
- OpenFang stores API keys, bot tokens, OAuth refresh tokens, MCP secrets, and similar sensitive values in an encrypted credential vault.
- OpenFang's credential vault uses AES-256-GCM at rest and resolves credentials through a chain that can include vault, dotenv, environment, and prompt.
- OpenFang wraps returned secret strings to reduce accidental leakage and keeps the credential vault separate from unrelated knowledge storage.

## Useful patterns for AGH

- Separate provider identity from execution harness.
- Use explicit secret references and runtime auth plans instead of broad env inheritance.
- Keep credential storage separate from provider display/config records.
- Prefer encrypted daemon-owned credential storage for API-key providers.

## Patterns to avoid

- Do not conflate provider secrets with knowledge/memory vault concepts.
- Do not expose raw provider secrets to extensions or child runtimes except through explicit bound injection.
