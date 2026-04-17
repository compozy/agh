# AGH Tasks — Dashboard

## Veredito

Não está suficientemente coberto para começar com fidelidade ao design. O daemon já calcula métricas internas úteis, mas elas ainda não estão expostas por endpoints públicos de tasks.

## O que já existe internamente

- O observer já calcula `QueryTaskSummary`, `QueryTaskMetrics` e `TaskHealth`, exatamente o tipo de material que um dashboard precisa. Evidência: [internal/observe/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/observe/tasks.go:167), [internal/observe/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/observe/tasks.go:198), [internal/observe/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/observe/tasks.go:207)
- O domínio persiste eventos e runs com timestamps suficientes para alimentar gráficos, breakdowns e cards de saúde. Evidência: [internal/store/globaldb/global_db_task_aux.go](/Users/pedronauck/Dev/compozy/agh/internal/store/globaldb/global_db_task_aux.go:301)

## O que falta para a tela do Paper

- Não existe endpoint público de `task summary`, `task metrics` ou `task dashboard`.
- O endpoint de health público não inclui um bloco de tasks, embora o observer já tenha essa informação. Evidência: [internal/api/contract/contract.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/contract.go:153), [internal/api/core/conversions.go](/Users/pedronauck/Dev/compozy/agh/internal/api/core/conversions.go:202)
- Não existe série temporal pronta para os gráficos de atividade/status do Paper.
- Não existe endpoint público para “active runs” com progresso resumido, última tool call e duração consolidada.
- Construir tudo via `GET /api/tasks` no cliente seria incompleto e caro, porque faltam query global de eventos de task, agregações temporais e summaries operacionais.

## Conclusão prática

Antes de implementar essa tela, vale abrir um pacote de backend para expor um dashboard de tasks em cima do observer já existente. Aqui existe bastante lógica pronta no read side; o gap principal é transporte/contrato, não cálculo bruto.
