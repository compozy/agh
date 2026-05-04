# Native ACP Provider Authentication and Isolation

## Summary

Fix AGH so native ACP providers are not preflight-blocked by missing API keys. AGH will model provider authentication explicitly instead of inferring it from `credential_slots`.

Research basis:

- ACP defines agent-owned auth as the default auth method, with env-var auth as a separate explicit method: https://agentclientprotocol.com/rfds/auth-methods
- Claude Code supports native login, API-key auth, and `claude auth status`; API keys can override subscription login: https://code.claude.com/docs/en/authentication and https://code.claude.com/docs/en/cli-reference
- Codex CLI supports ChatGPT login and API-key auth, with ChatGPT login as the default CLI path when no session exists: https://developers.openai.com/codex/auth#openai-authentication
- OpenCode ACP runs `opencode acp`; OpenCode credentials are managed through its own `opencode auth` commands: https://opencode.ai/docs/acp/ and https://opencode.ai/docs/providers
- Local competitor research from `.resources/hermes`, `.resources/openclaw`, `.resources/openfang`, and `.resources/goclaw` all points to the same split: API-key providers are AGH/runtime-managed, native subprocess providers own their own auth.

## Key Changes

- Add explicit provider auth and isolation fields to provider config and resolved agent state:
  - `auth_mode`: `native_cli`, `bound_secret`, or `none`.
  - `env_policy`: `filtered` or `isolated`.
  - `home_policy`: `operator` or `isolated`.
  - Optional diagnostic command fields: `auth_status_command` and `auth_login_command`.
- Default built-ins:
  - Direct ACP providers such as `claude`, `codex`, `gemini`, `opencode`, `hermes`, `openclaw`, `openfang`-style custom ACP providers, `goclaw`-style custom ACP providers, `goose`, `cline`, `qwen-code`, `copilot`, `cursor`, and similar launchers resolve to `auth_mode = "native_cli"`, `env_policy = "filtered"`, `home_policy = "operator"`.
  - Pi/API-key providers such as `pi`, `openrouter`, `zai`, `moonshot`, `vercel-ai-gateway`, `xai`, `minimax`, `mistral`, and `groq` resolve to `auth_mode = "bound_secret"` and keep required credential slots.
  - Local/no-auth custom providers may set `auth_mode = "none"`.
- Hard-cut the bad shape:
  - Native providers must not have required `credential_slots`. If a native provider overlay defines `required = true`, config validation fails with a targeted error telling the operator to either remove the slot or set `auth_mode = "bound_secret"`.
  - Remove built-in optional API-key slots from native ACP providers so AGH does not silently re-inject `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, or similar shell secrets into CLIs that already have native login.
  - Remove the repository root `config.toml` required key overlays for `claude`, `codex`, and `gemini`; built-ins should own those defaults.
- Keep `credential_slots` as bound secret injection only:
  - `bound_secret` providers fail before launch when required secrets are missing.
  - `native_cli` providers launch without keys; optional credential injection is allowed only when explicitly configured with `required = false`.
  - Missing native CLI auth is reported by the provider process or by diagnostic commands, not by AGH credential-slot preflight.
- Implement provider env/home policy:
  - `filtered` keeps the existing safe daemon environment behavior: operational variables stay, secret-shaped variables are stripped, AGH session/provider variables are added.
  - `isolated` creates a minimal child env plus AGH session/provider variables and explicitly injected secrets only.
  - `operator` home policy preserves the operator's native CLI auth stores by default.
  - `isolated` home policy creates `${AGH_HOME}/providers/<provider>` with `0700`, sets `PROVIDER_HOME`, sets provider-specific config vars where known (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `OPENCODE_CONFIG_DIR`), and sets XDG dirs for generic CLIs. It never copies credentials from the operator home.
- Add native auth diagnostics:
  - CLI and HTTP/UDS settings surfaces expose redacted auth metadata and status for every provider.
  - `agh provider auth status <name> -o json` reports auth mode, env/home policy, command availability, required secret status for `bound_secret`, and native status when a configured status command exists.
  - `agh provider auth login <name>` runs the provider's native login command under the same env/home policy when configured; otherwise it returns actionable manual guidance.
- Update web Settings providers:
  - Provider cards distinguish `native_cli`, `bound_secret`, and `none`.
  - Native providers no longer show as `unconfigured` just because an API-key env var is absent.
  - The provider editor defaults primary credential slots to `required = false` for `native_cli`, `required = true` for `bound_secret`, and disallows required slots for `native_cli`.
- Update docs and copy:
  - Installation and quick-start docs stop saying provider API keys are universally required.
  - Provider docs explain native CLI auth vs AGH-managed bound secrets, with examples for Claude Code, Codex, OpenCode, Hermes/OpenClaw-style ACP providers, and Pi/API-key providers.
  - Configuration docs document `auth_mode`, `env_policy`, `home_policy`, native diagnostics, and the hard validation rule.

## Public Interfaces

- `config.toml` provider fields gain `auth_mode`, `env_policy`, `home_policy`, `auth_status_command`, and `auth_login_command`.
- Settings API/UDS provider payloads include auth mode, env policy, home policy, and redacted auth status.
- Workspace provider option payloads include auth mode so session creation surfaces can represent native providers truthfully.
- OpenAPI and generated web TypeScript types are regenerated in the same change.

## Test Plan

- Config tests:
  - Built-in native ACP providers resolve with `auth_mode = native_cli`, no required credential slots, and no default API-key injection.
  - Pi/API-key providers resolve with `auth_mode = bound_secret` and required slots.
  - Native provider overlays with required slots fail validation unless they explicitly switch to `bound_secret`.
  - Repo root `config.toml` no longer turns native built-ins into required-key providers.
- Session/runtime tests:
  - Starting `claude`, `codex`, `opencode`, `hermes`, and `openclaw` with no provider API env vars does not fail in `prepareProviderForStart`.
  - Required missing secrets still fail for `openrouter`/Pi-style providers.
  - Filtered env strips daemon API keys/tokens; isolated env contains only the minimal allowlist, AGH variables, provider home variables, and explicit bound secrets.
  - `home_policy = isolated` creates provider home dirs with `0700` and sets known provider-specific home env vars.
- API/settings tests:
  - Provider list/detail payloads include auth metadata and never leak secrets.
  - Auth status endpoints return machine-readable `native_cli`, `bound_secret`, `none`, `missing_required`, `present`, and `unknown` states as applicable.
- Web tests:
  - Native provider cards render as installed/native-auth when no key is present.
  - Bound-secret providers render missing/unconfigured when required secrets are absent.
  - Provider editor cannot save a required slot under `native_cli`.
- Docs/codegen/gates:
  - Run `make codegen`, `make codegen-check`, targeted Go tests, targeted web tests, `make bun-lint`, `make bun-typecheck`, `make lint`, `make test`, `make web-build`, and final `make verify`.

## Assumptions and Defaults

- Default native ACP behavior uses `home_policy = operator`, so existing `claude auth login`, `codex login`, `opencode auth login`, Hermes, OpenClaw, and similar native auth stores keep working.
- Provider-home isolation ships in this change but is opt-in through `home_policy = isolated`.
- AGH will not import, copy, or infer native provider credentials from another tool's store.
- API-key CI or gateway workflows remain supported by explicitly setting `auth_mode = bound_secret` plus required `credential_slots`.
- No database schema migration is required; this is config, runtime launch, API contract, UI, and docs work.
