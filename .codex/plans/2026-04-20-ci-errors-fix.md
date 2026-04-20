# Remover Infra Visual do CI e Corrigir Lint

## Summary

- Corrigir o failure de `Verify` em Go no ponto exato reportado pelo CI: substituir a repetição de `"warn"` em `internal/observe/tasks.go` por uma constante local/nomeada que satisfaça `goconst` sem alterar comportamento.
- Remover a suíte de snapshot visual de `packages/ui` e `web` por completo, em vez de só desligar os jobs do CI. A causa raiz dos dois failures visuais é estrutural: o repositório versiona apenas baselines `darwin`, enquanto o CI Linux exige `linux`, e o time decidiu que não quer mais manter essa categoria de teste nesta fase.
- Limpar toda a superfície associada para não deixar código morto, exports quebrados, scripts órfãos, docs inconsistentes ou filtros de CI apontando para jobs removidos.

## Key Changes

- Workflow CI:
  - Remover os jobs `ui-visual` e `web-visual` de `.github/workflows/ci.yml`.
  - Remover os outputs e filtros `ui-visual` / `web-visual` do job `changes`.
  - Preservar `verify` e a lógica de detecção `backend` / `web` sem depender dos gates visuais.
- `packages/ui`:
  - Remover `playwright.config.ts`, `tests/visual/*`, `src/components/stories/__snapshots__/`, `scripts/serve-storybook.ts` e qualquer helper exclusivo da suíte visual.
  - Remover os scripts `build:visual`, `test:visual`, `test:visual:update`, `test:visual:install` de `package.json`.
  - Remover o export público `@agh/ui/testing/visual` e o arquivo `src/testing/visual-story-index.ts`.
  - Atualizar a README para apagar a seção de visual regression, comandos correspondentes e referências ao gate `ui-visual` / `web-visual`.
- `web`:
  - Remover `playwright.visual.config.ts`, `tests/visual/*`, `tests/visual/__snapshots__/` e `scripts/serve-storybook.ts`.
  - Remover os scripts `build:visual`, `test:visual`, `test:visual:update`, `test:visual:install` de `web/package.json`.
  - Remover imports e dependências internas que existam só para a suíte visual, incluindo o uso de `@agh/ui/testing/visual`.
- Limpeza de referências:
  - Atualizar testes/documentação que hoje afirmam a existência da configuração visual ou dos snapshots.
  - Ajustar comentários/copy que mencionem “visual snapshot suite” como contrato ativo.
- Lint fix:
  - Introduzir a constante para o status `warn` no mesmo escopo sem refatorações paralelas.

## Public APIs / Interfaces

- Remover o subpath export `@agh/ui/testing/visual`.
- Remover os comandos npm/bun públicos de visual testing em `packages/ui` e `web`.
- Remover do CI os jobs nomeados `UI visual snapshots (@agh/ui)` e `Web visual snapshots (web/)`.

## Test Plan

- Rodar buscas de sanidade para garantir que não restem referências a:
  - `test:visual`, `build:visual`, `playwright.visual.config`, `@agh/ui/testing/visual`, `ui-visual`, `web-visual`.
- Executar:
  - `bun run --cwd packages/ui test`
  - `bun run --cwd web test`
  - `make verify`
- Validar que:
  - `make verify` passa sem lint errors.
  - Storybook normal continua disponível via `storybook` / `build-storybook`.
  - O workflow CI não define nem referencia mais os jobs visuais removidos.
  - Nenhum arquivo órfão da suíte visual permanece versionado.

## Assumptions

- A remoção é intencional e completa: snapshot visual deixa de ser parte do contrato local e do CI nesta fase do produto.
- Storybook continua em uso para desenvolvimento/documentação; só a camada de Playwright visual regression sai.
- Não haverá substituição imediata por outro tipo de teste visual.
- Os PNGs versionados de snapshot devem ser removidos do repositório junto com a infra, não apenas ignorados.
