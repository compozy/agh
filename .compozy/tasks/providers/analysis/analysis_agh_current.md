# AGH Current Provider Architecture

## Sources inspected

- `internal/config/provider.go`
- `internal/session/manager_start.go`
- `internal/acp/types.go`
- `internal/acp/client.go`
- `internal/acp/launcher.go`
- `internal/config/config.go`
- `internal/api/contract/settings.go`
- `internal/settings/collections.go`
- `internal/api/contract/contract.go`
- `web/src/routes/_app/settings/providers.tsx`
- `web/src/systems/settings/components/provider-card.tsx`
- `web/src/systems/session/components/session-create-dialog.tsx`
- `packages/site/content/runtime/core/agents/providers.mdx`
- `packages/site/content/runtime/core/configuration/config-toml.mdx`
- `packages/site/content/runtime/core/configuration/env-vars.mdx`

## Findings

- AGH currently models providers as ACP-compatible commands with optional `default_model`, explicit `credential_slots`, and MCP server metadata.
- Built-ins include `claude`, `codex`, `gemini`, `opencode`, `copilot`, `cursor`, `kiro`, and `pi`.
- The built-in `pi` command currently uses `npx pi-acp@0.0.22`.
- `ResolvedAgent.Model` and credential slots are mostly metadata/status today. The session start path does not pass selected model/provider routing into ACP or Pi.
- `.env` loading is used as an in-memory lookup for validation and settings presence, but `.env` values are not automatically injected into spawned provider processes.
- The settings contract and web UI expose name, command, default model, and credential slot refs.
- The current web copy implies default model and API key env are enforced by the daemon, which overstates current runtime behavior.
- Workspace/session provider options are bare provider names, so the session picker cannot explain readiness or credential state.

## Implications for implementation

- Provider identity, runtime harness, runtime provider ID, selected model, and credential bindings need to become explicit runtime data, not UI-only metadata.
- `default_model` must either affect the child runtime or be documented as metadata-only. For Pi-backed providers it can be made real by generating Pi settings.
- Provider readiness needs richer status than command availability and env-var presence.
- API, UDS, CLI, web, and docs must co-ship because provider settings cross the daemon boundary.
