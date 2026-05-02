# AGH Web and Site Provider Surface Analysis

## Sources inspected

- `web/src/routes/_app/settings/providers.tsx`
- `web/src/systems/settings/components/provider-card.tsx`
- `web/src/hooks/routes/use-settings-providers-page.ts`
- `web/src/systems/session/components/session-create-dialog.tsx`
- `web/src/systems/session/hooks/use-session-create-dialog.ts`
- `packages/site/content/runtime/core/agents/providers.mdx`
- `packages/site/content/runtime/core/configuration/config-toml.mdx`
- `packages/site/content/runtime/core/configuration/env-vars.mdx`

## Findings

- The web provider settings surface currently exposes a minimal provider record: name, command, default model, and API key env.
- The web status model is command availability plus API key env presence.
- The session create dialog receives bare provider names from workspace details and cannot show provider readiness or missing credential reasons.
- Site docs describe ACP-compatible provider commands and environment variables, but do not document encrypted provider credential storage, provider catalog fields, Pi harness routing, or verification.
- Existing docs must be corrected when the backend changes because truthful docs take precedence over aspirational UI.

## Required updates

- Web provider settings should become catalog-driven and show:
  - public provider identity
  - harness/runtime provider
  - default model
  - base URL/custom endpoint
  - credential binding status
  - command/runtime readiness
  - verify status and warnings
- Session create should show readiness and actionable missing credential information.
- Docs should explain:
  - first-class provider identity
  - Pi-backed execution for API-key providers
  - AGH vault refs vs env refs
  - bound child-process injection
  - provider verification
  - CLI/API management paths
  - known Pi MCP limitation
