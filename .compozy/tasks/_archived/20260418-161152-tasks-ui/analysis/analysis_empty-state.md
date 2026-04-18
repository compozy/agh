# AGH Tasks — Empty State

## Veredito

Parcial, mas com baixo risco. O empty state em si pode ser implementado já; o ponto principal é que alguns templates do design ainda são conceitos de produto, não capacidades explícitas do domínio de tasks.

## O que já está disponível

- O estado vazio pode ser detectado diretamente via `GET /api/tasks` quando a lista vier vazia. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1446)
- O CTA `New task` está totalmente coberto pela API de criação. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1471)
- O CTA `Copy CLI command` faz sentido porque a CLI de tasks já existe e cobre create/list/get/update/dependencies/runs. Evidência: [internal/cli/task.go](/Users/pedronauck/Dev/compozy/agh/internal/cli/task.go:41)
- Template `Peer` conversa com o campo `network_channel`, que já é first-class na task e na run. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:252), [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:307)
- Template `Epic` pode ser montado em cima de `parent_task_id` e criação de filhos. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1546), [internal/task/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:197)

## Lacunas para bater com o design

- `Recurring` não é uma capacidade nativa da task. O encaixe mais natural hoje está em automation jobs/triggers que criam tasks, não na própria API de task. Evidência: [internal/automation/model/types.go](/Users/pedronauck/Dev/compozy/agh/internal/automation/model/types.go), [internal/api/contract/automation.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/automation.go)
- `Approval` não tem suporte como fluxo de task. O único endpoint de aprovação existente é para permissões de sessão, não para uma fila de tarefas aguardando aprovação. Evidência: [internal/api/udsapi/routes.go](/Users/pedronauck/Dev/compozy/agh/internal/api/udsapi/routes.go:72)
- O design sugere templates opinativos com semântica própria; hoje a API de task só aceita um shape genérico de create, então esses templates seriam presets da UI, não capacidades explícitas do backend. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:108)

## Conclusão prática

O empty state pode entrar já como UX. Só é importante tratar os cards de template como presets de frontend ou escopos de produto futuros, não como algo “já suportado” pelo daemon em todos os casos.
