# Storybook: Correcao de raiz para rotas quebradas por overrides de MSW

## Summary

- Corrigir a causa raiz no contrato do Storybook/MSW, nao as stories quebradas individualmente.
- Escopo escolhido: repo-wide. A correcao cobre as route stories com override local e tambem as demais stories do repositorio que usam o mesmo padrao, porque o bug e do harness, nao de uma rota especifica.
- Falhas reproduzidas com `agent-browser` na auditoria atual:
  - `routes-app-stories-bridges--empty`
  - `routes-app-stories-knowledge--content-loading`
  - `routes-app-stories-network--empty-channels`
  - `routes-app-stories-network--loading`
- Causa raiz confirmada:
  - `msw-storybook-addon` faz `resetHandlers()` e reaplica apenas `context.parameters.msw`.
  - O preview atual publica os handlers padrao como array flat.
  - Quando uma story define `parameters.msw.handlers`, ela substitui o conjunto global em vez de preservar os handlers padrao e sobrescrever so o dominio necessario.

## Changes

- Em `web/.storybook/preview.ts`, trocar o contrato de MSW do preview de um array flat para um registry agrupado por dominio:
  - `agent`
  - `automation`
  - `bridges`
  - `daemon`
  - `knowledge`
  - `network`
  - `session`
  - `settings`
  - `skill`
  - `workspace`
- Manter um export derivado flat apenas para testes e diagnostico; o preview em si deve usar o objeto agrupado.
- Em `web/src/storybook/`, adicionar um helper dedicado para composicao de overrides de MSW:
  - tipos explicitos para os grupos de handlers
  - helper unico para criar `parameters.msw` com override por grupo
  - `appRouteParameters()` continua responsavel so pelo router; composicao de MSW fica nesse helper
- Migrar todas as stories com override local para o novo formato:
  - `web/src/routes/_app/stories/**`
  - `web/src/routes/_app/settings/stories/**`
  - `web/src/systems/**/components/stories/**` que hoje usam `msw.handlers`
- Regra da migracao:
  - cada story sobrescreve apenas o grupo do sistema que ela quer alterar
  - nenhum arquivo continua usando `msw: { handlers: [ ... ] }` como array cru
- Ajustes especificos esperados nas stories que hoje falham:
  - `bridges`: `Empty` e `Loading` sobrescrevem o grupo `bridges`, preservando providers e demais dependencias
  - `knowledge`: `ContentLoading` sobrescreve o grupo `knowledge`, preservando a lista base de memories
  - `network`: `EmptyChannels` e `Loading` sobrescrevem o grupo `network`, preservando status, peers e detalhes dependentes
- Ajustar tambem as stories nao visivelmente quebradas hoje, mas com o mesmo risco, para remover a classe de bug inteira.

## Interfaces

- Mudanca explicita no contrato interno de Storybook:
  - antes: `parameters.msw.handlers = HttpHandler[]`
  - depois: `parameters.msw.handlers = { [groupName]: HttpHandler[] }`
- Novo helper interno de story authoring em `web/src/storybook/`:
  - entrada: overrides parciais por grupo
  - saida: objeto `msw` pronto para `parameters`
- Convencao nova para o repositorio:
  - preview define defaults por grupo
  - stories locais sobrescrevem grupos, nunca substituem o conjunto inteiro via array

## Test Plan

- Atualizar `web/src/storybook/web-storybook-config.test.tsx` para validar:
  - `preview.parameters.msw.handlers` como objeto agrupado
  - decorators/loaders permanecem inalterados
  - contrato do router real continua funcionando
- Atualizar `web/src/storybook/web-storybook-msw-contract.test.ts` para validar:
  - o registry agrupado inclui todos os mock barrels dos systems
  - o flatten derivado nao possui pares duplicados `method + path`
  - um override de um grupo nao exige redefinir os outros grupos
- Adicionar um teste de regressao focado no helper novo em `web/src/storybook/`:
  - override parcial de um grupo preserva os defaults dos grupos nao tocados
  - stories que precisam alterar um endpoint usam apenas o grupo correspondente
- Verificacao manual obrigatoria com `agent-browser` apos a mudanca:
  - enumerar todas as route stories via `index.json`
  - abrir cada `iframe.html?id=...&viewMode=story`
  - confirmar que nenhuma story fora das variantes `Error` renderiza `data-testid$=\"-error\"` com `Not Found`
  - conferir explicitamente as quatro stories reproduzidas na auditoria
- Gate final esperado:
  - `make web-lint`
  - `make web-typecheck`
  - `bun run --cwd web build-storybook`

## Assumptions

- A correcao deve ser repo-wide, porque o bug esta no contrato de Storybook/MSW e ja afeta superficies alem de `routes/`.
- Nao havera workaround de story individual, fallback silencioso, ou supressao de erro; a solucao e mudar o contrato de composicao dos handlers.
- O comportamento desejado das stories `Error` continua o mesmo; elas seguem podendo sobrescrever o grupo do sistema para forcar falhas reais.
