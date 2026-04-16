# Reorganizar a navegação de `runtime` com root folders do Fumadocs

## Summary

- Usar o Fumadocs do jeito oficial dentro de `/runtime`: as organizações maiores ficam em root folders com `"root": true`, e o `DocsLayout` com `tabMode="navbar"` passa a ser a fonte única da top nav interna.
- Manter a navegação global do site e todo o `/protocol` como estão hoje. A mudança é só na árvore de documentação de `runtime`.
- Não criar lógica customizada para esconder itens da sidebar. A separação entre navbar e sidebar deve vir da própria page tree do Fumadocs.

## Key Changes

- Reestruturar `packages/site/content/runtime`:
  - `meta.json` da raiz passa a listar somente `core`, `cli-reference` e `api-reference`.
  - `core/meta.json` passa a ter o título `Core Concepts` e esta ordem de grupos: `index`, `overview`, `getting-started`, `sessions`, `agents`, `memory`, `skills`, `workspaces`, `automation`, `bridges`, `hooks`, `extensions`, `operations`, `configuration`.
  - Mover fisicamente os blocos conceituais para `content/runtime/core/*`, espelhando o padrão da docs do Compozy e garantindo que o sidebar de Core mostre só conteúdo de Core.
  - Unificar `reference/` dentro de `core/configuration/`; a seção `Configuration` passa a conter `index`, `config-toml`, `agent-md`, `skill-md`, `mcp-json`, `env-vars` e `file-locations`.
  - Criar `content/runtime/core/index.mdx` como landing page de `Core Concepts`.
- Atualizar a renderização do runtime:
  - `packages/site/app/runtime/[[...slug]]/page.tsx` deixa de redirecionar `/runtime` para `/runtime/core/` e passa a renderizar a landing existente de `runtime`.
  - `runtime/index.mdx` atualiza os links e rótulos para `Core Concepts`, `CLI Reference` e `API Reference`; qualquer link para páginas conceituais passa a usar a nova árvore em `/runtime/core/...`.
  - `packages/site/components/docs/doc-page-masthead.tsx` passa a derivar a seção real a partir da slug aninhada (`core/<section>/...`, `cli-reference/<group>/...`, `api-reference/...`), para não mostrar `Core` em todas as páginas conceituais.
- Normalizar ícones e títulos:
  - `packages/site/lib/source.ts` adiciona os ícones já usados nos `meta.json` e hoje ausentes no resolver (`Brain`, `Waypoints`, `Plug`) e inclui `FolderTree` para `Workspaces`.
  - `workspaces/meta.json` ganha `title: "Workspaces"` e `icon: "FolderTree"`.

## Test Plan

- Validar a topologia da docs do runtime:
  - a navegação maior tem exatamente 3 tabs: `Core Concepts`, `CLI Reference`, `API Reference`
  - uma página em Core mostra só grupos de Core na sidebar
  - uma página em CLI mostra só grupos de CLI na sidebar
- Validar rotas:
  - `/runtime` renderiza a landing do runtime
  - `/runtime/core` renderiza a landing de Core Concepts
  - `/runtime/cli-reference` e `/runtime/api-reference` continuam válidas
  - `/protocol` permanece inalterado
- Validar apresentação:
  - `Memory`, `Hooks`, `Extensions` e `Workspaces` exibem ícones no sidebar
  - o masthead mostra a seção correta nas páginas aninhadas
- Rodar a verificação do pacote do site e a verificação exigida pelo workspace antes de concluir.

## Assumptions

- `protocol` continua como documentação high level separada do runtime e não entra nesta reorganização.
- `Memory`, `Skills`, `Workspaces` e `Automation` permanecem como grupos separados.
- `API Reference` continua com a estrutura atual de uma página por enquanto; a mudança agora é de organização e navegação.
- Não haverá camada de compatibilidade para as URLs antigas das páginas conceituais; a implementação deve atualizar todos os links internos do repositório para a nova árvore.
