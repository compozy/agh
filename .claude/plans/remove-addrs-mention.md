Plano: Limpeza de menções a ADRs e specs temporárias em documentos/código permanentes

Context

O commit 64efdeb5 (refactor: redesign v2) finalizou a spec .compozy/tasks/redesign-v2/. Durante o processo, várias citações a ADRs (ADR-001 … ADR-016), a
paths como .compozy/tasks/redesign[-v2]/, e a artefatos da spec (\_techspec.md, "redesign-v2 PR-N", "task_NN") vazaram para documentos e código permanentes
do projeto.

A regra do projeto: documentos permanentes (CLAUDE.md, AGENTS.md, DESIGN.md, READMEs, skills, lessons, código-fonte) não devem referenciar documentos
temporários (ADRs e specs vivem em .compozy/tasks/<slug>/ e podem ser arquivados/movidos a qualquer momento). Regras precisam stand on their own.

Esta limpeza estende-se por todo o repositório — não apenas o último commit — porque os mesmos padrões existem em documentos anteriores (lessons que citam
ADRs de specs concluídas como autonomy/orch-improvs).

Escopo total: ~130 arquivos com ADR-[0-9] + ~12 com redesign-v2 + ~7 com .compozy/tasks/redesign.

Princípio único de transformação

Para cada match, remover a citação preservando a regra/conteúdo. A citação é um ponteiro para o porquê histórico — a regra continua válida por si só. Onde
a citação É todo o conteúdo (ex: "see ADR-001 §4"), remover a linha/parágrafo.

Exclusões (não tocar)

- .compozy/tasks/\*\* — as próprias specs temporárias
- .skeeper/\*\* — espelho do skeeper
- node_modules/, .next/, .turbo/, dist/, .git/
- ai-docs/, .tmp/, .claude/plans/, .claude/ledger/ — artefatos temporários
- web/src/generated/ — código gerado

Categoria 1 — Documentos fixos do projeto (root)

Arquivos:

- CLAUDE.md linhas 58, 73 — remover bullets que apontam para .compozy/tasks/redesign/ e redesign-v2/screenshots/proposal/. Reescrever para regra genérica
  de design (ex: "diff against a trusted prior baseline").
- AGENTS.md linhas 58, 73 — idem (espelho do CLAUDE.md).
- web/CLAUDE.md linhas 14, 66, 70 — idem.
- web/AGENTS.md linhas 14, 65, 69 — idem.
- packages/site/CLAUDE.md — só 1 menção; ler e limpar.

Manter: referências genéricas a \_techspec.md/\_prd.md como padrão de artefato dentro do workflow cy-create-\* (linhas 17, 38, 113, 238 do CLAUDE.md root).
Essas descrevem o pipeline de spec, não a redesign-v2.

Categoria 2 — Skills e agentes (.agents/, .claude/agents/)

- .agents/skills/agh-design/SKILL.md — 48 citações ADR-NNN §K. Remover toda parentética (ADR-NNN §K) ou (ADR-NNN §K + ADR-MMM §L). Manter o texto da regra.
- .agents/skills/agh-ui-screenshot/SKILL.md linha 3 — substituir "redesign-v2 audit" por "visual-regression diff against design baseline".
- .agents/skills/agh-ui-screenshot/references/cdp-flow.md linha 69 — remover path .compozy/tasks/redesign-v2/screenshots/proposal/; manter "trusted prior
  baseline".
- .agents/skills/agh-ui-screenshot/references/proposal-mock-capture.md linha 3 — neutralizar "redesign-v2 audit".
- .agents/skills/agh-ui-screenshot/references/storybook-urls.md linha 53 — renomear seção "redesign-v2 audit" → "design audit".
- .agents/skills/cy-research-competitors/SKILL.md linha 64 — manter genérico (\_techspec.md no contexto do pipeline é ok); verificar contexto.
- .claude/agents/cy-researcher.md linha 8 — substituir exemplo redesign-v2 por <slug>.

Categoria 3 — DESIGN.md (root)

101 ocorrências ADR-NNN. Padrão dominante: parentéticas tipo (ADR-001 §6), (ADR-001 §7 + ADR-010 §6), (ADR-003 + ADR-001 + ADR-011) em headings e bullets.

Estratégia: sed/regex pass para apagar a parentética, deixando a regra:

- \s*\(ADR-\d+(\s*§[\d.]+)?(\s*[+/]\s*ADR-\d+(\s*§[\d.]+)?)*\) → ""
- Headings tipo ## 2.5 Surface glaze ladder (ADR-001 §6) → ## 2.5 Surface glaze ladder
- Para textos onde a citação é o conteúdo principal (ex: "Cross-link: see ADR-001 §4") → remover linha inteira

Manter: todas as regras de design, exemplos, e referências a DESIGN.md §N.M (auto-referências permanecem válidas).

Categoria 4 — Lessons e memory (docs/\_memory/)

- docs/\_memory/lessons/L-022-eyebrow-canonical-source.md linhas 6, 40 — remover redesign-v2/task_06, Redesign-v2 superseded, e ADR-002 §1. Manter regra
  ("single Inter UC contract, prop-less primitive") e rationale.
- docs/\_memory/lessons/L-003-task-runs-single-queue.md linha 4 — remover (autonomy techspec, ADR-003), manter data e conteúdo.
- docs/\_memory/lessons/L-004-manual-equals-peer.md linhas 4, 5, 34 — remover ADR-010 referenced 7+ times across \_techspec.md, manter o lesson content.
- docs/\_memory/lessons/L-005-authoritative-primitive-exclusivity.md linhas 4, 5 — remover autonomy ADR-004, task_11, ADR-004 + task_11 memory. Substituir
  por descrição neutra.
- docs/\_memory/lessons/L-019-diagnostic-data-outlives-primary-record.md linhas 4, 5 — remover orch-improvs task 25 / ADR-003, ADR-003 + bridge subscription
  store tests. Manter conteúdo.
- docs/\_memory/lessons/L-020-dense-typed-records-need-pointer-boundaries.md linhas 12, 66 — remover (ADR-002, ADR-009, ADR-010) parentéticas; manter regra.
- docs/\_memory/spec-authoring-playbook.md — manter referências genéricas a \_techspec.md/\_prd.md/\_idea.md (descrevem o pipeline). Ajustar linha 142 (every
  ADR) para "every per-task ADR file if present" (já é genérico, só clarificar).
- docs/\_memory/standing_directives.md — manter como está (referências hipotéticas, sem leak).
- docs/\_memory/\_synthesis.md e docs/\_memory/analysis/\*.md — OUT OF SCOPE. Esses são documentos de análise forense que catalogam evidência por design (citar
  ADRs específicos é o ponto). São o registro de pesquisa, não regra de projeto. Deixar intactos.

Categoria 5 — Lint plugin (regras permanentes do design system)

- lint-plugins/compozy-design-system.mjs:
  - Linha 2: "redesign-v2 contracts" → "design-system contracts"
  - Linha 7: remover redesign-v2 PR-4 closeout (task_29)
  - Linha 14: remover adjective redesign-v2 da descrição da rule
  - Linhas 196, 249, 287, 325: remover "See ADR-NNN §K" das mensagens de erro; trocar por referência a DESIGN.md ou pelo nome semântico da regra
  - Linha 121: remover task_06 / PR-2 do comentário
  - Linha 150–152: remover comentário sobre redesign-v2 PR-4 closeout
- lint-plugins/**tests**/compozy-design-system.test.mjs — verificar se algum teste asserta nas strings de mensagem que foram alteradas; ajustar conforme
  necessário.

Categoria 6 — Tokens CSS

- packages/ui/src/tokens.css — 16 ocorrências de ADR-NNN §K em comentários de seção. Estratégia: remover citações parentéticas; manter títulos descritivos.
  Exemplo: /_ Surface glaze ladder — ADR-001 §6. _/ → /_ Surface glaze ladder. _/. Remover também redesign-v2 task_01 na linha 31.

Categoria 7 — Código-fonte (comentários JSDoc)

Per direção do usuário: remover o comentário inteiro quando só serve para citar ADR. Manter JSDoc quando ele tem conteúdo descritivo além da citação.

packages/ui/src/components/custom/\*.tsx (~17 arquivos):

- catalog-card.tsx, detail-header.tsx, detail-inspector.tsx, form-section.tsx, kpi-card.tsx, priority-bars.tsx, queue-health-sparkline.tsx, radio-card.tsx,
  section.tsx, status-dot.tsx, status-line-topbar-slot.tsx, time.tsx, topbar.tsx, index.ts (linhas 234, 260 — comentários "Redesign-v2 PR-2 content
  primitives")

packages/ui/src/components/: sidebar.tsx linha 40, textarea.tsx linha 8, icon.tsx.

packages/ui/src/lib/owner-palette.ts linha 3 — "thin re-export per ADR-001 §7" → remover citação, manter "thin re-export".

web/src/components/design-system-showcase.tsx — 6 cites em descrições de tokens. Remover parentéticas (ADR-NNN §K).

web/src/lib/status-tone.ts linha 6 — substituir "redesign-v2 contract (TechSpec §...)" por "core status-tone vocabulary".

web/src/lib/owner-palette.ts — verificar e limpar.

web/src/hooks/routes/use-session-topbar-slot.tsx — verificar e limpar.

web/src/systems/\*_/_.tsx e .ts (~30 arquivos): cada um tem 2–6 cites em JSDoc. Padrão dominante: \* Per-agent card per ADR-007 §9 — .... Estratégia: remover
apenas per ADR-NNN §K mantendo o resto da frase. Onde a frase fica vazia, remover o JSDoc.

Arquivos:

- web/src/systems/agent/components/agent-info-inspector.tsx
- web/src/systems/knowledge/components/knowledge-pill-tone.ts, knowledge-type-tone.ts
- web/src/systems/network/components/timeline/message-avatar.tsx
- web/src/systems/network/components/work/work-banner.tsx
- web/src/systems/runtime/components/connection-indicator.tsx, hooks/nav-counts-store.ts, use-nav-counts.ts
- web/src/systems/session/components/permission-prompt.tsx, session-inspector.tsx, tool-call-card.tsx
- web/src/systems/tasks/components/agent-card.tsx, task-group.tsx, task-kanban-card.tsx, task-run-timeline-panel.tsx, tasks-dashboard-cards.tsx,
  tasks-dashboard-queue-health.tsx, tasks-dashboard-view.tsx, tasks-detail-children-panel.tsx, tasks-detail-header.tsx, tasks-detail-overview-panel.tsx,
  tasks-empty-state.tsx, tasks-inbox-row.tsx, tasks-kanban-board.tsx
- web/src/systems/tasks/lib/inbox-grouping.ts, task-formatters.ts, task-grouping.ts, task-templates.ts, types.ts

Backend:

- internal/memory/catalog.go linha 575 — remover Task 06 / ADR-011 do comentário; manter rationale.
- internal/sandbox/daytona/VALIDATION.md — verificar contexto.

Categoria 8 — Storybook stories (\*.stories.tsx)

~22 arquivos packages/ui/src/components/custom/stories/\*.stories.tsx + alguns em web/src/. Padrão: descrição de story cita ADR como contexto. Estratégia
idêntica à categoria 7 — remover só a parte ADR.

Arquivos principais:

- Stories em packages/ui/src/components/custom/stories/ (~20)
- Stories em web/src/systems/\*\*/stories/ (~5)
- web/src/components/stories/topbar-shell.stories.tsx

Categoria 9 — Testes

Por direção do usuário: deletar testes que apenas validam citações ADR. Para testes com asserções mistas (regra + cite), manter a checagem da regra e
remover a do ADR.

Ações por arquivo:

- packages/ui/src/**tests**/design-md.test.ts (10 cites) — remover it(...) blocks tipo it("Should cross-link to ADR-001 §4", ...). Manter blocks de seção
  heading e regra. Os snapshots associados (**snapshots**/design-md.test.ts.snap) serão regenerados.
- packages/ui/src/**tests**/agh-design-skill.test.ts (50 cites) — remover a tabela PR1_RULES cujo único valor é checar citation regex; manter checagem
  needle (regra). Reduzir a suite à validação das regras (sem o citation:).
- packages/ui/src/**tests**/tokens.test.ts (13 cites) — remover asserts tipo expect(...).toMatch(/ADR-/). Manter asserts de valores de tokens.
- web/src/**tests**/styles.test.ts linhas 48, 97, 102, 105 — remover comentários e renomear it("Should pin --shadow-overlay (ADR-003)", ...) removendo a
  parte ADR do nome.
- web/src/**tests**/lint-config.test.ts linhas 60, 63, 68, 81 — renomear describe blocks removendo redesign-v2 PR-4 closeout e task_29 / PR-4.
- Demais \*.test.ts(x) (~30 arquivos): geralmente o ADR aparece em comentário ou em describe label. Aplicar política — limpar.

Snapshots:

- packages/ui/src/**tests**/**snapshots**/agh-design-skill.test.ts.snap (48 hits)
- packages/ui/src/**tests**/**snapshots**/design-md.test.ts.snap (22 hits)
- Estes serão regenerados automaticamente ao atualizar os testes/fontes (vitest -u).

Categoria 10 — packages/ui/README.md

- Linha 17 — remover frase The original redesign ADR files are not checked into this repo.
- Linha 73 — remover (ADR-002 §1, lesson L-022); manter referência a L-022 apenas se permanente (lesson L-022 é OK, é institucional).
- Linhas 212–215 — bloco inteiro "Redesign-v2 PR-2 chrome primitives (task_10)…" precisa ser reescrito: deletar o changelog ou reescrever como "Primitives
  shipped in this release" sem citar PR/task/ADR.

Categoria 11 — Site e content

- packages/site/content/runtime/core/configuration/config-toml.mdx — verificar e limpar se houver leak.
- packages/site/lib/**tests**/site-design-token-contract.test.ts — limpar refs a .compozy/tasks/redesign.

Categoria 12 — E2E tests

- web/e2e/**tests**/tasks-coordinator-handoff.spec.ts — verificar e limpar.
- web/src/routes/\_app/**tests**/-agents.$name.sessions.$id.test.tsx, -tasks.test.tsx, settings/**tests**/-vault.test.tsx — limpar.

Ordem de execução

1.  Documentos-mestre primeiro (CLAUDE.md/AGENTS.md/DESIGN.md/skills) — assim o "source of truth" fica limpo.
2.  Lessons e memory.
3.  Lint plugin + tokens.css (esses são "permanentes ativos" — afetam todo o resto).
4.  Código-fonte (JSDoc/comments).
5.  Stories.
6.  Testes — por último, porque vamos ter que rodar vitest -u para regenerar snapshots e validar que as assertions removidas estão coerentes.
7.  make verify no final.

Verificação

# 1) Confirma que não sobrou ADR-N em arquivos permanentes:

rg -l 'ADR-[0-9]' \
 -g '!.compozy/**' -g '!.skeeper/**' -g '!node_modules/**' \
 -g '!ai-docs/**' -g '!.tmp/**' -g '!.claude/plans/**' \
 -g '!.claude/ledger/**' -g '!web/src/generated/**' \
 -g '!docs/\_memory/\_synthesis.md' -g '!docs/\_memory/analysis/\*\*' .

# Esperado: 0 arquivos (ou apenas docs/\_memory/analysis/\* + \_synthesis.md, que ficam de fora por design).

# 2) Confirma que não sobrou redesign-v2:

     # 2) Confirma que não sobrou redesign-v2:
     rg -l 'redesign-v2|redesign v2' \
       [mesmos excludes] .
     # Esperado: 0

     # 3) Confirma que não sobrou .compozy/tasks/redesign em paths:
     rg -l '\.compozy/tasks/redesign' [mesmos excludes] .
     # Esperado: 0

     # 4) Gate completo:
     make verify

     make verify deve passar (Bun lint/typecheck/test em todos os workspaces + Go lint/test/build). Os snapshots precisam ser regenerados (bunx turbo run
     test --filter=./packages/ui -- -u e similares para web). Sem warnings.

     Riscos

     - Snapshots desatualizados depois do edit massivo — regenerar com vitest -u por workspace.
     - Testes que dependem da estrutura textual de DESIGN.md/SKILL.md vão quebrar — vamos ter que ajustar caso a caso (Categoria 9).
     - Lint plugin com mensagens de erro mudadas — se algum teste asserta no texto da mensagem, precisa update.
     - Lessons perderem traceability histórica — aceita-se: a regra/lição é o que importa, não o ponteiro pro ADR temporário. Quem quiser forensics vai no
     git blame.
     - Volume: ~130 arquivos. Execução manual file-a-file. Estratégia: agrupar por padrão (uma regex pass por categoria onde possível) + spot-check.

     Arquivos críticos a modificar (lista enxuta para execução)

     Top-priority (mais cites ou docs canônicos):
     1. DESIGN.md
     2. .agents/skills/agh-design/SKILL.md
     3. packages/ui/src/tokens.css
     4. packages/ui/src/__tests__/agh-design-skill.test.ts + snapshot
     5. packages/ui/src/__tests__/design-md.test.ts + snapshot
     6. packages/ui/src/__tests__/tokens.test.ts
     7. lint-plugins/compozy-design-system.mjs
     8. CLAUDE.md, AGENTS.md, web/CLAUDE.md, web/AGENTS.md, packages/site/CLAUDE.md
     9. packages/ui/README.md
     10. packages/ui/src/components/custom/*.tsx (17 arquivos) + stories (22)
     11. web/src/systems/**/*.tsx,*.ts (30 arquivos)
     12. web/src/components/design-system-showcase.tsx
     13. web/src/__tests__/styles.test.ts, lint-config.test.ts
     14. web/src/lib/status-tone.ts, owner-palette.ts
     15. docs/_memory/lessons/L-003, L-004, L-005, L-019, L-020, L-022
     16. .agents/skills/agh-ui-screenshot/SKILL.md + 3 references
     17. .agents/skills/cy-research-competitors/SKILL.md
     18. .claude/agents/cy-researcher.md
     19. internal/memory/catalog.go
     20. internal/sandbox/daytona/VALIDATION.md
     21. Demais cites pontuais em web/src/hooks, web/e2e, packages/site

     Total estimado: ~130 arquivos, ~400+ edits.
