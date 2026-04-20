# Storybook `_app routes`: Tasks completo com MSW

## Summary

- Adicionar cobertura de Storybook para a família completa de rotas de `tasks` usando o roteador real da app e dados servidos por MSW.
- Tratar `web/src/systems/tasks/mocks` como a fonte canônica de fixtures e handlers de `tasks` para Storybook.
- Organizar as stories por rota, não em um único arquivo grande: `-tasks.stories.tsx`, `-tasks.$id.stories.tsx`, `-tasks.new.stories.tsx`, `-tasks.$id.edit.stories.tsx` e `-tasks.$id.runs.$runId.stories.tsx`.

## Public interfaces / internal contracts

- Estender o contrato interno do Storybook MSW em `web/src/storybook/msw.ts` com o grupo `tasks`.
- Registrar `tasks` no preview global em `web/.storybook/preview.ts`, ao lado dos demais systems já suportados.
- Estender o router stub do Storybook para incluir `/tasks`, `/tasks/new`, `/tasks/$id`, `/tasks/$id/edit` e `/tasks/$id/runs/$runId`, para manter compatibilidade com component stories que renderizam links dessas superfícies.
- Expor um barrel `web/src/systems/tasks/mocks/index.ts` com `handlers` e fixtures reutilizáveis para stories e testes.

## Implementation changes

- Criar `web/src/systems/tasks/mocks/fixtures.ts` com dados canônicos para lista, detalhe, dashboard, inbox, timeline, árvore de agentes e run detail.
- Criar `web/src/systems/tasks/mocks/handlers.ts` cobrindo os endpoints de leitura usados pelas rotas: `GET /api/tasks`, `GET /api/tasks/{id}`, `GET /api/tasks/{id}/runs`, `GET /api/tasks/{id}/timeline`, `GET /api/tasks/{id}/tree`, `GET /api/tasks/dashboard`, `GET /api/tasks/inbox` e `GET /api/task-runs/{id}`.
- Incluir handlers leves de mutação apenas quando uma `play` function depender deles; o foco da cobertura será estado visual de rota, não workflow completo de submit.
- Reaproveitar as fixtures já existentes de `tasks` movendo a origem dos dados para `systems/tasks/mocks`; `components/stories/fixtures.ts` e `components/test-fixtures.ts` devem virar adapters ou re-exports do novo ponto central, em vez de continuar como fontes paralelas.
- Adicionar `web/src/routes/_app/stories/-tasks.stories.tsx` com stories `DefaultList`, `Empty`, `Kanban`, `Dashboard`, `Inbox`, `Loading` e `Error`. Os estados de modo devem ser alcançados pelo shell real, usando `StorybookWorkspaceSetup` e `play` functions só para trocar os pills.
- Adicionar `web/src/routes/_app/stories/-tasks.$id.stories.tsx` com `Overview`, `RunsTab`, `TimelineTab`, `AgentsTab`, `ChildrenTab`, `DependenciesTab`, `Loading` e `NotFound`. A fixture base desse arquivo deve ser rica o suficiente para renderizar todos os painéis finais sem placeholders vazios.
- Adicionar `web/src/routes/_app/stories/-tasks.new.stories.tsx` com `Default`, `TemplatePreset` e `Submitting`. O preset de template deve vir do search param real da rota, não de props locais.
- Adicionar `web/src/routes/_app/stories/-tasks.$id.edit.stories.tsx` com `Default`, `Loading` e `MissingTask`, todos resolvidos por MSW sobre a rota real.
- Adicionar `web/src/routes/_app/stories/-tasks.$id.runs.$runId.stories.tsx` com `Running`, `Completed`, `Failed`, `NoSession`, `Loading` e `NotFound`, todos abastecidos por `GET /api/task-runs/{id}` e `GET /api/tasks/{id}`.
- Manter todas as route stories em `layout: "fullscreen"` e com o shell real montado, seguindo o padrão já usado em `jobs`, `knowledge`, `network` e `settings`.

## Test plan

- Atualizar `web/src/storybook/msw.test.ts` para validar o novo grupo `tasks` e a composição correta de overrides.
- Atualizar `web/src/storybook/web-storybook-msw-contract.test.ts` para exigir `tasks` no registry global e continuar garantindo ausência de duplicatas por `method + path`.
- Atualizar `web/src/storybook/web-storybook-config.test.tsx` para cobrir as novas placeholder routes do router stub.
- Adicionar regressão leve para importar os novos módulos de stories de rota de `tasks` e o novo barrel de mocks.
- Rodar os gates de verificação: `make web-lint`, `make web-typecheck`, `make web-test`, `bun run --cwd web build-storybook` e `make verify`.

## Assumptions

- “Tudo completo” significa cobrir a família inteira de rotas `tasks` e todos os estados visuais de leitura relevantes, não automatizar fluxos completos de criação/edição com submit persistente.
- O padrão de manutenção escolhido é um arquivo de stories por rota da família `tasks`, não um arquivo único monolítico.
- A consolidação recomendada de fixtures é parcial e pragmática: o novo `systems/tasks/mocks` vira a fonte canônica, e os pontos já existentes passam a reaproveitá-lo sem uma migração ampla de todo o sistema de testes além do que tocar neste trabalho.
