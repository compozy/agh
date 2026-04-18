# AGH Tasks — List (Split View)

## Veredito

Parcial e pronto para começar a implementação da UI. A base CRUD existe, mas a tela do Paper pede um read model mais rico do que o payload atual entrega.

## O que já está disponível

- `GET /api/tasks` já lista tarefas com filtros por `scope`, `workspace`, `status`, `owner_kind`, `owner_ref`, `parent_task_id`, `network_channel` e `limit`. Isso cobre a espinha dorsal da lista e parte dos filtros do header. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1446), [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:97)
- `GET /api/tasks/{id}` já entrega um detalhe expandido com `task`, `children`, `dependencies`, `runs` e `events`, então o painel direito pode começar usando um fetch único por tarefa selecionada. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1488), [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:88), [internal/task/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:746)
- As ações principais da tela já existem: atualizar tarefa, cancelar e enfileirar uma nova run. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1506), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1526), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1626)
- Os selects auxiliares do design podem buscar agentes e workspaces via APIs já existentes. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:283), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1860)

## Lacunas para bater com o design

- O payload de lista é simples demais para os cards do split view. `TaskSummaryPayload` não traz `attempt atual / total`, contagem de filhos, tarefa bloqueadora, última atividade, duração, nem resumo da run ativa. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:11), [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:251)
- O painel de dependências da tela mostra identificador e título da tarefa dependente. Hoje `TaskDependencyPayload` traz só ids, `kind` e `created_at`; a UI teria que fazer joins extras para exibir nomes. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:45)
- Não existe busca textual por identificador ou título no backend. O design pede search explícito na lista. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:97), [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:349)
- Não existe agrupamento, ordenação customizável, paginação cursorizada ou um endpoint próprio para um read model de listagem. O store ordena por `updated_at DESC`, mas isso não resolve os agrupamentos do design. Evidência: [internal/store/globaldb/global_db_task.go](/Users/pedronauck/Dev/compozy/agh/internal/store/globaldb/global_db_task.go), [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:349)
- `priority` não é um campo de primeira classe da task. Se a tela precisa badge/filtro de prioridade, hoje isso só cabe em `metadata`. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:273), [internal/task/validate_test.go](/Users/pedronauck/Dev/compozy/agh/internal/task/validate_test.go:411)

## Conclusão prática

É viável começar a tela agora usando `GET /api/tasks` + `GET /api/tasks/{id}` e deixar alguns trechos “menos ricos” na primeira entrega. Para chegar perto do Paper sem acoplamento excessivo no frontend, o ideal é adicionar um payload de lista enriquecido ou um endpoint read-only específico para a visão split.
