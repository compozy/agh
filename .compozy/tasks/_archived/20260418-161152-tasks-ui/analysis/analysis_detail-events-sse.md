# AGH Tasks — Detail (Events SSE)

## Veredito

Parcial e implementável, mas hoje a tela exigirá composição de múltiplas fontes no frontend. Não existe um stream unificado de eventos da task.

## O que já está disponível

- `GET /api/tasks/{id}` já retorna histórico de eventos da task no payload expandido. Isso cobre o timeline histórico base. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:88), [internal/task/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:773)
- O domínio já emite eventos relevantes para a timeline do Paper, como `task.created`, `task.dependency_added`, `task.run_enqueued`, `task.run_claimed`, `task.run_started`, `task.run_failed` e `task.run_completed`. Evidência: [internal/task/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:15)
- Para a parte “live” da execução, já existem `GET /api/sessions/{id}/events`, `GET /api/sessions/{id}/history`, `GET /api/sessions/{id}/transcript` e o stream SSE `/api/sessions/{id}/stream`. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1366), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1389), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1412), [internal/api/httpapi/routes.go](/Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go:73), [internal/api/core/handlers.go](/Users/pedronauck/Dev/compozy/agh/internal/api/core/handlers.go:361)

## Lacunas para bater com o design

- Não existe endpoint público para listar eventos da task com filtro por tipo e cursor/SSE. O único acesso público hoje é o array `events` devolvido por `GET /api/tasks/{id}`. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1488), [internal/store/globaldb/global_db_task_aux.go](/Users/pedronauck/Dev/compozy/agh/internal/store/globaldb/global_db_task_aux.go:301)
- A tela do Paper mistura eventos da task e eventos de sessão/tooling na mesma timeline. Hoje isso exige merge client-side entre `task.events` e o stream da `session` da run ativa.
- O stream de sessão existe em rota real, mas não aparece no `internal/api/spec/spec.go` nem no OpenAPI gerado. Isso dificulta integrar o frontend seguindo só o contrato codegen. Evidência: [internal/api/httpapi/routes.go](/Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go:73), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go)
- Não há stream “por task” para trocar automaticamente quando a run ativa muda.

## Conclusão prática

Dá para implementar a tela agora combinando `GET /api/tasks/{id}` com `session_id` da run ativa e SSE de sessão. Para uma integração limpa e alinhada ao design, o backend deveria expor um endpoint/stream próprio de eventos da task ou um timeline unificado task + run.
