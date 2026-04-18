# AGH Tasks — Create Modal

## Veredito

Parcial e pronto para começar, mas a modal do Paper está mais avançada que o contrato atual da task.

## O que já está disponível

- `POST /api/tasks` cobre criação com `title`, `description`, `scope`, `workspace`, `owner`, `network_channel`, `identifier` e `metadata`. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1471), [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:108)
- `Save draft` no sentido de “criar sem enfileirar run” é tecnicamente possível com `POST /api/tasks`.
- `Create & enqueue` é componível agora: primeiro `POST /api/tasks`, depois `POST /api/tasks/{id}/runs`. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1626)
- `Parent task` já é suportado no modelo de criação, então a modal pode criar tarefas-filhas ou relacionar uma task a um pai. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:273)
- `Metadata`, `idempotency key` e `network channel` têm encaixe claro nos contratos atuais. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:57), [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:108)

## Lacunas para bater com o design

- O design mostra `Priority` como campo de primeira classe. Hoje prioridade só cabe em `metadata`; não existe campo estruturado para isso. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:108), [internal/task/validate_test.go](/Users/pedronauck/Dev/compozy/agh/internal/task/validate_test.go:411)
- O design mostra `Attempts` como parte da criação. O domínio atual não tem `max_attempts` na task; existe apenas `attempt` por run já criada. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:57), [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:307)
- O campo `Parent task` no Paper sugere busca por identificador ou título. Não existe busca textual server-side em tasks; a UI teria que fazer lookup parcial no cliente sobre `GET /api/tasks`. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:97)
- `Save draft` não mapeia para um estado “draft” real. `CreateTask` cria a task em `ready`, então um draft salvo apareceria como trabalho pronto ou bloqueado, não como rascunho. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:22), [internal/task/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:158)
- O dropdown de owner pode listar agentes via API, mas o contrato da task aceita `owner kind/ref` genérico; não há validação explícita de “owner disponível”. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:283), [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:273)

## Conclusão prática

É seguro iniciar a modal com os campos estruturados que já existem e empurrar `priority` para `metadata` no curto prazo. Para alcançar o design com semântica limpa, o backend precisa decidir se `priority`, `max_attempts` e `draft` vão virar capacidades explícitas do domínio.
