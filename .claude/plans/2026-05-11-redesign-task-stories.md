# Redesign Stories — Task Overview & Task Run Detail

## Context

Um software externo (com os 3 prompts originais) produziu três HTMLs em `docs/design/new-proposal/` propondo um redesign visual para as stories:

- `systems-tasks-taskoverviewcomponents--operational-cards`
- `systems-tasks-taskoverviewcomponents--orchestration`
- `systems-tasks-taskrundetail--running`

A leitura dos HTMLs vs. o código real revela três fatos:

1. **A maior parte dos padrões propostos JÁ existe** como primitives em `@agh/ui` (`<Section>`, `<Table>`, `<Pill>`, `<MonoId>`, `<Metric>`, `<RunCard>`, `<Timeline>`, `<TimelineEvent>`, `<DetailHeader>`, `<Eyebrow>`).
2. **Cinco dos seis componentes-alvo já consomem essas primitives corretamente** — alinhados ao `DESIGN.md` (tokens, hairlines, signal-tone-only, eyebrow contract).
3. **Há UM delta estrutural real**: `tasks-timeline-panel.tsx` (timeline overview) usa um `<ol>` custom com `<Pill.Dot>` em vez de `<Timeline>` + `<TimelineEvent>`, ficando visualmente inconsistente com a timeline do run-detail (que JÁ usa as primitives corretas).

O HTML também propõe um header de página + tabs no `redesign-orchestration.html`. Esse chrome pertence à rota `routes/app/tasks/$id.tsx`, **não** aos componentes da story — fora de escopo conforme decisão do usuário.

**Objetivo do trabalho**: aplicar um pass amplo de polish visual nos seis componentes envolvidos, fazendo o delta estrutural na timeline overview e uniformizando iconografia + count + slot direito das `<Section>` em todos os cards. Nenhum dado, fixture ou feature é inventado — texto e estrutura continuam refletindo o backend real (`submit_run_review`, `latest_event_seq`, `bridge_instance`, `worker/coordinator/review/sandbox` modes).

## Arquivos críticos a modificar

### Helper compartilhado (extrair duplicação)

`web/src/systems/tasks/lib/timeline-visuals.ts` (NOVO — promove DRY)

- Exporta: `EVENT_VISUALS: Record<string, { tone: PillTone; icon: LucideIcon }>`, `visualFor(eventType)`, `FAILURE_EVENT_TYPES`, `LIVE_EVENT_TYPES`, `SUCCESS_EVENT_TYPES`, `describeEvent(item)`.
- Reutilizado por `tasks-timeline-panel.tsx` (overview) e `task-run-timeline-panel.tsx` (run detail). Hoje as funções estão duplicadas com lógica praticamente idêntica.

### Componente com delta estrutural

`web/src/systems/tasks/components/tasks-timeline-panel.tsx`

- Substituir o `<ol>` custom + `<Pill.Dot>` por `<Timeline>` + `<TimelineEvent>` do `@agh/ui` (mesmo padrão já adotado em `task-run-timeline-panel.tsx:215-267`).
- Cada `<TimelineEvent>` consome `icon` + `tone` resolvidos via `visualFor()` do novo helper.
- `meta` slot do `<TimelineEvent>` carrega `Pill mono size="xs"` para `seq N` + separadores `·` + `attempt N` + status label + origin ref (mesma estrutura que hoje renderiza inline; só muda o container).
- `time` slot recebe `<Time iso={item.timestamp} mode="relative" />`.
- `title` carrega o `event_type` em mono, com tone `danger` para falhas.
- Manter intactos:
  - `PillGroup` com modos `interleaved | by_agent | by_event_type` (essencial e ausente nos HTMLs).
  - `Load more` paginação.
  - Agrupamento por agent/event-type (cada grupo agora renderiza um `<Timeline>` próprio com header `Eyebrow + count`).
- `<Section>` ganha:
  - `icon={Activity}` (lucide).
  - `count={items.length}`.
  - `right={isLive ? <LiveIndicator /> : undefined}` — move o "Live" pulse-dot do body para o `right` slot da Section (alinhado ao HTML).
  - Mantém `aria-label="Task events"` e `data-testid="tasks-timeline-panel"`.
- `InterleavedEventList` vira função interna que retorna `<Timeline>{events.map(...)}</Timeline>`.

### Polish nos 4 cards já alinhados

Aplicar a mesma uniformidade de Section em todos:

`web/src/systems/tasks/components/tasks-reviews-card.tsx`

- `<Section>` adicionar `icon={Gavel}` + `count={reviews.length}` (estado loaded; estados loading/error/empty mantêm Section sem count para não confundir).
- Pequeno polish: `data-testid={\`${testIdPrefix}-${review.review_id}-guidance\`}`hoje usa`bg-input-fill`— manter (token correto); o redesign quer um left-rail amarelo, mas é o mesmo efeito visual via`border-l-warning` que **está banido** pela design rule "no-side-stripe-accent". Manter como está (tint sem stripe).

`web/src/systems/tasks/components/tasks-bridge-notifications-card.tsx`

- `<Section>` adicionar `icon={Radio}` + `count={subscriptions.length}` quando há dados.
- A `<Section right>` hoje carrega o botão "New subscription"; manter, agora ao lado direito do count chip pelo layout natural da Section.

`web/src/systems/tasks/components/tasks-execution-profile-card.tsx`

- `<Section>` adicionar `icon={Settings2}` (já importado).
- Sem `count` — execution profile é singular (presente/ausente).
- O wrapper interno `<div bg-canvas-soft rounded-lg>` já replica a aparência do "surface profile-card" do HTML. Manter.

`web/src/systems/tasks/components/tasks-stream-resume-card.tsx`

- `<Section>` adicionar `icon={Activity}` (já importado).
- Sem `count`.
- O grid `<Metric>` 3-col já casa exatamente com o HTML proposto. Nenhuma outra mudança.

`web/src/systems/tasks/components/tasks-detail-orchestration-panel.tsx`

- Ordem atual: profile → reviews → notifications → stream.
- Mesma ordem proposta no HTML do orchestration. **Nenhuma mudança.**

### Stories

`web/src/systems/tasks/components/stories/task-overview-components.stories.tsx`

- A story `OperationalCards` já renderiza timeline → reviews → stream → bridge → execution profile. Ordem alinhada com `redesign-operational-cards.html`. **Nenhuma mudança.**
- A story `Orchestration` já renderiza o painel container com a ordem correta. **Nenhuma mudança.**

`web/src/systems/tasks/components/stories/task-run-detail.stories.tsx`

- Header + timeline já alinhados via `<DetailHeader>` + `<RunCard>` + `<Timeline>` + `<TimelineEvent>`. **Nenhuma mudança.**

## O que NÃO fazer

- Não criar novos arquivos HTML — o redesign vai direto para o código React, e as stories refletem automaticamente.
- Não tocar em `routes/app/tasks/$id.tsx` (header/tabs da rota real) — escopo decidido com o usuário.
- Não trocar fixtures de dados — IDs (`review_001`, `bsub_001`, etc.) já existem em `web/src/systems/tasks/mocks/` e são os mesmos referenciados no HTML.
- Não introduzir hex literals nem importar fontes via Google/CSS direto (o HTML usa; é falha do HTML, não do código).
- Não introduzir `border-l-*` colorido em cards (banido pela design rule `no-side-stripe-accent`).
- Não criar componentes paralelos que dupliquem o que `@agh/ui` já expõe (`<Timeline>`, `<TimelineEvent>`, `<RunCard>`, `<DetailHeader>`).

## Verificação

Antes de declarar pronto:

1. **Screenshots before/after via `agh-ui-screenshot`** para os 3 stories:
   - `systems-tasks-taskoverviewcomponents--operational-cards`
   - `systems-tasks-taskoverviewcomponents--orchestration`
   - `systems-tasks-taskrundetail--running`
   - Capturar antes da mudança para baseline; depois para diff. Citar paths dos PNGs no report final.

2. **Gates de código** (não-destrutivos, executar do repo root):
   - `make web-lint` — oxfmt + oxlint zero warnings.
   - `bunx turbo run typecheck --filter=./web` — typecheck via Turbo.
   - `bunx turbo run test --filter=./web` — vitest via Turbo (verifica que os data-testid e a estrutura não quebraram nenhum teste de `tasks-timeline-panel`, `tasks-reviews-card`, etc.).

3. **Smoke storybook**:
   - `make web-dev` (ou Storybook se estiver hospedado) e verificar visualmente as 3 stories em `http://localhost:6006/`.

4. **Decisão sobre testes (consolidate-test-suites)**:
   - Invariantes da timeline overview já estão cobertos em `tasks-timeline-panel.test.tsx` (se existir) — não criar novos testes só para acolher mudança visual. O contrato testável (rendering de N items, modos de view, paginação, live indicator) permanece idêntico após o refactor.
   - Se algum teste se apoiar no `<ol>` específico ou no `Pill.Dot`, atualizar o seletor para a estrutura `<Timeline>` + `<TimelineEvent>` (ainda data-testid stable).
   - Não adicionar testes de CSS/snapshot por causa do polish — proibido pela regra de placement.

## Ordem de execução proposta

1. Criar `web/src/systems/tasks/lib/timeline-visuals.ts` consolidando `EVENT_VISUALS`, `visualFor`, sets de event types e `describeEvent`.
2. Refatorar `task-run-timeline-panel.tsx` para consumir o helper (remove duplicação local).
3. Refatorar `tasks-timeline-panel.tsx` para consumir o helper + migrar `<ol>` para `<Timeline>` + `<TimelineEvent>` + adicionar `icon`/`count`/`right` na `<Section>`.
4. Adicionar `icon` + `count` nas Sections dos demais 4 cards (reviews/bridge/execution profile/stream resume).
5. Rodar lint + typecheck + tests + capturar screenshots.
6. Reportar com paths das capturas e diffs.
