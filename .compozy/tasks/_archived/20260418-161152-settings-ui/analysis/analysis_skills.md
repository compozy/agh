# Analysis: Skills

- Veredito: PARCIAL

## O que a tela pede

- Toggle global do engine de skills.
- Contadores de discovered/disabled.
- `poll_interval`.
- Configuracao de marketplace (`registry`, `base_url`).
- Listas de `disabled_skills`, `allowed_marketplace_mcp` e `allowed_marketplace_hooks`.

## O que ja existe

- O daemon ja tem modelo de configuracao completo para skills.
- A API HTTP ja expoe list/get/content/enable/disable de skills.
- O `web/` ja tem adaptador e hooks no padrao atual para consumir skills.

## Gaps para implementar a tela inteira

- A API atual lista skills resolvidas por workspace, mas nao expoe o bloco de configuracao de settings.
- Nao existe endpoint HTTP para `skills.enabled`, `poll_interval`, marketplace ou allow-lists.
- Nao existe endpoint HTTP especifico para sumarizar contadores globais de discovered/disabled.
- O enable/disable atual e por skill, nao pelo engine inteiro.

## Evidencias

- `internal/config/config.go:129-137` define `SkillsConfig`.
- `internal/api/httpapi/routes.go:162-169` registra o namespace `/api/skills`.
- `internal/api/core/skills.go:14-180` implementa list/get/content/enable/disable.
- `web/src/systems/skill/adapters/skill-api.ts:20-127` confirma que o `web/` ja consome esse dominio no padrao atual.

## Conclusao

- Existe boa cobertura para a parte operacional do catalogo de skills.
- A tela de settings ainda precisa de endpoints dedicados para a configuracao global de skills.
