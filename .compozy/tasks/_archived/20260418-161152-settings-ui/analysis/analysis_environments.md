# Analysis: Sandbox

- Veredito: NAO PRONTO

## O que a tela pede

- Catalogo de perfis de sandbox com backend, sync mode, persistence, runtime root e policy de network.
- Contagem de workspaces usando cada sandbox.
- Acao para criar novos sandboxes.
- Possivel edicao de detalhes do perfil.

## O que ja existe

- O daemon ja tem `Config.Sandboxes` com `SandboxProfile`.
- Workspaces ja podem referenciar um sandbox por `sandbox_ref`.
- O `web/` ja consome workspaces seguindo o padrao do projeto.

## Gaps para implementar a tela inteira

- Nao existe endpoint HTTP para listar sandboxes definidos no daemon.
- Nao existe endpoint HTTP para detalhar um sandbox com seus atributos.
- Nao existe endpoint HTTP para criar/editar/remover sandboxes.
- Nao existe endpoint pronto que devolva "workspaces using this sandbox"; hoje isso exigiria composicao manual no backend.

## Evidencias

- `internal/config/config.go:171-200` define `SandboxProfile`, `NetworkProfile` e `DaytonaProfile`.
- `internal/config/config.go:210-220` inclui `Sandboxes map[string]SandboxProfile` no config raiz.
- `internal/api/core/workspaces.go:39-45` aceita `SandboxRef` no create de workspace.
- `internal/api/core/workspaces.go:94-152` aceita atualizar `sandbox_ref` do workspace.
- `web/src/systems/workspace/adapters/workspace-api.ts:23-69` mostra que o `web/` ja consome o dominio de workspaces, mas nao sandboxes.
- `internal/api/httpapi/routes.go:11-27` nao registra nenhum grupo `/sandboxes`.

## Conclusao

- O dado existe no config e a referencia ja e usada por workspaces.
- A tela ainda depende de uma API propria para sandboxes antes de qualquer integracao real no `web/`.
