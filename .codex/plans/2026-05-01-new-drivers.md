# Add Latest ACP Drivers and Refresh Provider Defaults

## Summary

- Add built-in ACP providers for blackbox, cline, goose, hermes, junie, kimi-cli, openclaw, openhands, qoder, and qwen-code.
- Remove pinned driver package versions from built-ins and docs, using explicit @latest for npm/npx drivers and pi-acp@latest for Pi-backed providers.
- Keep kimi mapped to the existing moonshot Pi/API provider, per user choice; add the direct Kimi CLI driver as kimi-cli.
- Do not invent model IDs for provider-managed CLIs. Providers with confirmed concrete defaults get default_model; providers whose model is selected in
  their own config/login flow keep an empty default_model.

## Public Interfaces And Provider Matrix

- Update internal/config/provider.go built-ins and aliases. No OpenAPI shape change is expected because provider IDs are strings, not enums.
- Update agh install bootstrap behavior so an empty model is valid for direct ACP providers with provider-managed model selection; keep model required
  for Pi-backed providers because pi_acp runtime materialization requires it.
- New/updated provider commands:
  - claude: npx -y @agentclientprotocol/claude-agent-acp@latest, model claude-sonnet-4-6, env ANTHROPIC_API_KEY.
  - codex: npx -y @zed-industries/codex-acp@latest, model gpt-5.4, env OPENAI_API_KEY.
  - gemini: keep gemini --acp, model gemini-3.1-pro-preview, env GEMINI_API_KEY.
  - opencode: npx -y opencode-ai@latest acp.
  - all Pi-backed providers: replace the previous pinned Pi ACP adapter with npx -y pi-acp@latest; update pi to claude-opus-4-7,
    openrouter to openai/gpt-5.4, and vercel-ai-gateway to anthropic/claude-opus-4-7.
  - blackbox: blackbox --experimental-acp, env BLACKBOX_API_KEY, provider-managed model.
  - cline: npx -y cline@latest --acp, provider-managed model.
  - goose: goose acp, provider-managed model.
  - hermes: hermes acp, provider-managed model.
  - junie: junie --acp true, provider-managed model.
  - kimi-cli: kimi acp, provider-managed model, env KIMI_API_KEY, aliases kimi-cli, kimi cli, kimi-code.
  - openclaw: openclaw acp, provider-managed gateway model.
  - openhands: openhands acp, provider-managed model.
  - qoder: npx -y @qoder-ai/qodercli@latest --acp, provider-managed model, env QODER_PERSONAL_ACCESS_TOKEN.
  - qwen-code: npx -y @qwen-code/qwen-code@latest --acp --experimental-skills, model qwen3.6-plus, aliases qwen, qwen code.

## Implementation Changes

- Backend/config:
  - Add a table-driven registry test covering every built-in provider, command, harness, runtime provider, credential env, and model default.
  - Add a guard test rejecting pinned npm driver package versions for built-in ACP commands, while allowing normal model IDs that contain numbers.
  - Update alias tests to cover kimi-cli, qwen-code, and the preserved kimi -> moonshot behavior.
  - Update install/bootstrap tests for optional model behavior on provider-managed direct ACP providers and required model behavior on Pi-backed
    providers.
- Web:
  - Keep production provider listing dynamic from /api/settings/providers.
  - Update settings/workspace/session fixtures, route tests, and provider/agent icon maps so the new providers render intentionally instead of falling
    through everywhere.
  - Do not mix ACP agent providers with bridge providers.
- Site/docs:
  - Update packages/site provider tables, config examples, env-var docs, agent definition examples, spawning docs, landing supported-agent strip, and
    related tests.
  - Regenerate CLI reference only if the agh install help/example text changes.
  - Document provider-managed defaults clearly: AGH exposes the driver, while model/login selection may remain inside the vendor CLI.

## Test Plan

- Run targeted backend tests first: go test ./internal/config ./internal/cli ./internal/settings ./internal/api/core ./internal/session -count=1.
- Run web checks touching changed surfaces: relevant Vitest files for settings/session/fixtures, then make web-test, make web-typecheck, and make web-
  lint.
- Run site checks: cd packages/site && bun run test, cd packages/site && bun run typecheck, and cd packages/site && bun run build.
- Run make cli-docs if Cobra install examples/help change.
- Run final monorepo gate: make verify.

## Assumptions And Sources

- Work with the current dirty worktree; do not revert or clean unrelated modified/untracked files.
- No schema migration or OpenAPI regeneration is planned unless implementation reveals a contract shape change.
- After this plan is accepted and execution is allowed, persist it under .codex/plans/ before editing code.
- Primary research sources used: ACP Agents (https://agentclientprotocol.com/get-started/agents), ACP Registry
  (https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json), Blackbox CLI
  (https://docs.blackbox.ai/features/blackbox-cli/introduction), Cline (https://cline.bot/), Goose ACP
  (https://block.github.io/goose/docs/guides/acp-clients/), Hermes ACP
  (https://hermes-agent.nousresearch.com/docs/user-guide/features/acp), Junie (https://junie.jetbrains.com/), Kimi CLI
  (https://github.com/MoonshotAI/kimi-cli), OpenClaw ACP (https://docs.openclaw.ai/cli/acp), OpenHands ACP
  (https://docs.openhands.dev/openhands/usage/cli/ide/overview), Qoder ACP (https://docs.qoder.com/cli/acp), Qwen Code
  (https://github.com/QwenLM/qwen-code), OpenAI GPT-5.4 (https://platform.openai.com/docs/models/gpt-5.4),
  Anthropic models (https://platform.claude.com/docs/en/about-claude/models/overview), and Gemini models
  (https://ai.google.dev/gemini-api/docs/models).
