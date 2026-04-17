# Analysis: Automation

- Veredito: PARCIAL

## O que a tela pede

- Toggle global do engine.
- `timezone`.
- `max_concurrent_jobs`.
- Resumo de `default_fire_limit`.
- Link/acao para abrir a area operacional de automation.

## O que ja existe

- O daemon ja tem configuracao para engine, timezone e limites.
- A API HTTP ja expoe um dominio completo para jobs, triggers e runs.
- O `web/` ja tem sistema de `automation` pronto no padrao do projeto.

## Gaps para implementar a tela inteira

- Nao existe endpoint HTTP que exponha o bloco `automation.enabled`, `timezone`, `max_concurrent_jobs` e `default_fire_limit`.
- A API atual e focada em entidades operacionais de automacao, nao em settings do engine.
- A tela desenhada pode abrir a area existente de automacao, mas nao pode ainda editar os settings do topo.

## Evidencias

- `internal/config/automation.go:13-21` define `AutomationConfig`.
- `internal/api/httpapi/routes.go:115-138` registra jobs, triggers e runs sob `/api/automation`.
- `internal/api/core/automation.go:35-256` mostra CRUD e historico de jobs.
- `web/src/systems/automation/adapters/automation-api.ts:50-257` confirma integracao pronta para o dominio operacional.
- `web/src/systems/automation/index.ts:27-106` mostra o sistema web ja estruturado para esse dominio.

## Conclusao

- A parte "Open automation" e viavel hoje.
- A tela de settings propriamente dita ainda precisa de endpoints para configuracao global do engine.
