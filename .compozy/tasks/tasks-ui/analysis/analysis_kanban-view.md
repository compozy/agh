# AGH Tasks — Kanban View

## Veredito

Parcial e implementável para uma primeira versão, mas o design do Paper assume estados e resumos que o domínio atual ainda não expõe direito.

## O que já está disponível

- `GET /api/tasks` já devolve `status`, `owner`, `scope`, `workspace`, `parent_task_id` e `network_channel`, o suficiente para montar colunas básicas no cliente. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:11), [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:97)
- Os status canônicos já incluem `ready`, `in_progress`, `blocked`, `completed`, `failed` e `canceled`, então a maior parte das colunas do kanban está coberta. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go)
- Retry/enqueue já é possível por `POST /api/tasks/{id}/runs`, e cancelamento também existe. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1526), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1626)
- O filtro por owner do design pode usar `owner_kind` e `owner_ref`, e os candidatos podem vir de `GET /api/agents`. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:97), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:283)

## Lacunas para bater com o design

- A coluna `Pending` do Paper não está realmente ancorada no ciclo normal do manager. `TaskStatusPending` existe no tipo, mas `CreateTask` cria tarefas em `ready`, não em `pending`. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:22), [internal/task/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:158)
- Os cards do kanban mostram informações operacionais como `attempt`, duração ao vivo, quantidade de tools e erro resumido. Nada disso sai de `TaskSummaryPayload`. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:11)
- O estado “failed com Retry” é construível, mas exige derivar a última run e seu erro; não existe um summary pronto por tarefa para isso. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:57), [internal/task/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:769)
- Não existe ordenação/agrupamento server-side para kanban nem uma consulta voltada a board view. O frontend teria que carregar e agrupar tudo sozinho. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:349)

## Conclusão prática

Dá para subir um kanban funcional com agrupamento client-side por `status`, filtro de owner/scope e ação de retry. Para ficar fiel ao design, o backend precisa fornecer um summary de board por tarefa e decidir se `pending` continuará existindo só como conceito visual ou como estado de domínio real.
