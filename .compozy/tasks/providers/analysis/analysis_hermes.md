# Hermes Provider and Credential Analysis

## Sources inspected

- `.resources/hermes/hermes_cli/providers.py`
- `.resources/hermes/hermes_cli/auth.py`
- `.resources/hermes/hermes_cli/auth_commands.py`
- `.resources/hermes/agent/runtime_provider.py`
- `.resources/hermes/agent/credential_pool.py`
- `.resources/hermes/agent/auxiliary_client.py`
- `.resources/hermes/agent/redact.py`
- `.resources/hermes/hermes_cli/web_server.py`
- QMD page: `qmd://hermes/wiki/concepts/authentication-and-provider-system.md`

## Findings

- Hermes has a provider registry that merges upstream `models.dev` data, Hermes overlays, aliases, and user configuration.
- The overlay model captures transport, aggregator status, auth type, extra env vars, base URL overrides, and base URL env vars.
- Hermes supports secondary providers such as OpenRouter, z.ai, Kimi/Moonshot, Vercel AI Gateway, xAI, and others.
- Provider aliases normalize user-facing names such as `moonshot`, `z.ai`, `z-ai`, `glm`, and `ai-gateway`.
- Runtime resolution separates provider, model, API mode, base URL, and credential source.
- Hermes supports credential pools with priority, status, last error, cooldown, and rotation strategies.
- Hermes stores API keys in `~/.hermes/.env` and OAuth data in `~/.hermes/auth.json`; it relies on file permissions and process/container isolation rather than encryption at rest.
- Redaction exists but is not the same as a daemon-owned encrypted vault.

## Useful patterns for AGH

- Copy the registry/overlay/alias idea.
- Preserve visible provider identity while routing through a lower-level harness.
- Design the schema so credential pooling can be added later, even if v1 ships with one credential per provider slot.
- Keep provider/model/base URL/API mode resolution centralized to avoid stale or leaked routing state.

## Patterns to avoid

- Do not use plaintext provider secrets as the primary AGH storage model.
- Do not rely on process-wide environment passthrough as the authoritative credential binding.
- Do not expose a global auth file that every child process can read.
