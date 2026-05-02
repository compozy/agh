# Compozy Code and Pi Harness Analysis

## Sources inspected

- `~/dev/compozy/compozy-code/packages/electron/src/renderer/src/systems/settings/providers/provider-definitions.ts`
- `~/dev/compozy/compozy-code/packages/electron/src/shared/global-settings.ts`
- `~/dev/compozy/compozy-code/packages/electron/src/shared/buildProviderOptions.ts`
- `~/dev/compozy/compozy-code/providers/runtime/src/adapters/claude-code/index.ts`
- `~/dev/compozy/compozy-code/providers/runtime/src/adapters/*adapter.ts`
- `~/dev/compozy/compozy-code/packages/electron/src/main/utils/claude-code-env-reader.ts`
- `~/dev/compozy/compozy-code/.resources/pi/packages/coding-agent/docs/providers.md`
- `~/dev/compozy/compozy-code/.resources/pi/packages/coding-agent/docs/custom-providers.md`
- `~/dev/compozy/compozy-code/.resources/pi/packages/coding-agent/docs/rpc.md`
- npm metadata for `pi-acp`
- npm metadata for `@mariozechner/pi-ai`

## Findings

- Compozy Code exposes providers such as z.ai, OpenRouter, Vercel, Moonshot, MiniMax, Bedrock, Vertex, and Ollama as first-class UX choices while routing many through Claude Code under the hood.
- The strongest pattern is preserving provider identity while keeping harness/runtime implementation separate.
- The weaker pattern is scattering provider definitions, env builder cases, and adapter aliases across multiple files.
- `pi-acp` is an ACP adapter for Pi and spawns `pi --mode rpc`.
- Pi supports built-in API-key providers and custom providers through its own configuration.
- Pi API keys can come from CLI args, `~/.pi/agent/auth.json`, environment variables, or custom provider model definitions.
- Pi supports `PI_CODING_AGENT_DIR` to relocate its agent configuration directory.
- Pi custom providers can define base URLs, API modes, API keys, headers, auth headers, model lists, overrides, and compatibility flags.
- `pi-acp` currently accepts ACP MCP params but does not wire them through to Pi.

## Recommended AGH use

- Use `pi-acp` as the v1 harness for API-key providers.
- Generate an AGH-owned isolated Pi agent directory per session or per provider/session launch.
- Set `PI_CODING_AGENT_DIR` in the child environment.
- Generate Pi settings for provider/model selection.
- Generate Pi custom provider config only when AGH needs custom base URL or compatibility behavior.
- Inject credentials as target environment variables from AGH's bound secret resolver.
- Keep all AGH provider descriptors in one registry to avoid Compozy Code's scattered list problem.

## Limitations

- Do not claim MCP pass-through for Pi-backed providers until verified.
- Avoid shell-command API key sources from Pi config in AGH-managed paths; AGH should own secret resolution.
