# Analysis: Environments

- Veredito: NAO PRONTO

## O que a tela pede

- Catalogo de perfis de ambiente com backend, sync mode, persistence, runtime root e policy de network.
- Contagem de workspaces usando cada ambiente.
- Acao para criar novos ambientes.
- Possivel edicao de detalhes do perfil.

## O que ja existe

- O daemon ja tem `Config.Environments` com `EnvironmentProfile`.
- Workspaces ja podem referenciar um ambiente por `environment_ref`.
- O `web/` ja consome workspaces seguindo o padrao do projeto.

## Gaps para implementar a tela inteira

- Nao existe endpoint HTTP para listar ambientes definidos no daemon.
- Nao existe endpoint HTTP para detalhar um ambiente com seus atributos.
- Nao existe endpoint HTTP para criar/editar/remover ambientes.
- Nao existe endpoint pronto que devolva "workspaces using this environment"; hoje isso exigiria composicao manual no backend.

## Evidencias

- `internal/config/config.go:171-200` define `EnvironmentProfile`, `NetworkProfile` e `DaytonaProfile`.
- `internal/config/config.go:210-220` inclui `Environments map[string]EnvironmentProfile` no config raiz.
- `internal/api/core/workspaces.go:39-45` aceita `EnvironmentRef` no create de workspace.
- `internal/api/core/workspaces.go:94-152` aceita atualizar `environment_ref` do workspace.
- `web/src/systems/workspace/adapters/workspace-api.ts:23-69` mostra que o `web/` ja consome o dominio de workspaces, mas nao ambientes.
- `internal/api/httpapi/routes.go:11-27` nao registra nenhum grupo `/environments`.

## Conclusao

- O dado existe no config e a referencia ja e usada por workspaces.
- A tela ainda depende de uma API propria para ambientes antes de qualquer integracao real no `web/`.
