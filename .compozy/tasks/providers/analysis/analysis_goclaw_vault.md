# GoClaw Provider and Vault Analysis

## Sources inspected

- `.resources/goclaw/internal/store/provider_store.go`
- `.resources/goclaw/internal/store/sqlitestore/providers.go`
- `.resources/goclaw/internal/store/config_secrets_store.go`
- `.resources/goclaw/internal/store/sqlitestore/config-secrets.go`
- `.resources/goclaw/internal/config/config_secrets.go`
- `.resources/goclaw/internal/crypto/aes.go`
- `.resources/goclaw/internal/http/providers.go`
- `.resources/goclaw/internal/http/provider_verify.go`
- `.resources/goclaw/internal/tools/credentialed_exec.go`
- `.resources/goclaw/internal/tools/scrub.go`
- `.resources/goclaw/docs/20-api-keys-auth.md`
- QMD pages:
  - `qmd://goclaw/wiki/concepts/llm-provider-bridge.md`
  - `qmd://goclaw/wiki/concepts/security-rbac-and-crypto.md`

## Findings

- GoClaw supports many provider types, including OpenRouter, Groq, DeepSeek, Mistral, xAI, MiniMax, Cohere, Perplexity, DashScope, z.ai, Ollama, ACP, and Claude CLI.
- Provider records carry display/config data while API keys are encrypted on write and decrypted at the store seam.
- HTTP read/list responses mask API keys as `***`; update handlers strip masked input so saving a form does not erase existing credentials.
- Provider verification has local/ping and model-call modes. Cloud verification uses a small chat call with a timeout and extracts friendly provider errors.
- Secure CLI credentials are encrypted and injected into child process environment only when needed.
- Tool output scrubbing uses static token patterns plus dynamic values resolved for the current run.
- GoClaw allows optional unencrypted fallback in some paths; AGH should not copy that because AGH is greenfield alpha.

## Useful patterns for AGH

- Store secrets through a single encrypted store seam.
- Return typed masked DTOs that cannot contain raw secret values.
- Treat masked update input as "leave existing secret unchanged."
- Add provider verification as a first-class operator and agent-manageable surface.
- Seed redaction with dynamically resolved credentials for the spawned session.

## Patterns to avoid

- Do not support plaintext fallback for AGH provider secrets.
- Do not solve provider API keys and AGH external API keys in the same implementation batch.
