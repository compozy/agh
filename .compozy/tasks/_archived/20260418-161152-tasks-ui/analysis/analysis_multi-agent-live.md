# AGH Tasks — Multi-Agent Live

## Veredito

Parcial, mas com lacunas maiores que as outras telas operacionais. Existe material suficiente para uma primeira versão, porém a composição hoje seria custosa e com muitos fetches/streams paralelos.

## O que já está disponível

- O domínio já suporta árvore de tasks via `parent_task_id` e criação de filhos. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:252), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1546)
- `GET /api/tasks/{id}` já devolve `children` e `runs`, o que permite descobrir parte da estrutura do “parent + child tasks”. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:88)
- Cada run pode ser ligada a uma `session` e, a partir disso, a UI consegue abrir transcript e stream ao vivo por agente. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:57), [internal/api/httpapi/routes.go](/Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go:73)

## Lacunas para bater com o design

- `GET /api/tasks/{id}` traz só `children` em nível de summary. Para chegar no layout do Paper, a UI teria que buscar detalhes e runs adicionais para cada filho e talvez para netos. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:88), [internal/task/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:761)
- Não existe endpoint agregado da árvore com `task + descendants + active runs + session ids + latest activity`.
- Não existe stream unificado de múltiplas sessões por task tree. O frontend precisaria abrir vários SSEs ou fazer polling/merge manual.
- O stream global de observe existe, mas não filtra por `task_id` nem por conjunto de sessões; ele não resolve sozinho essa tela. Evidência: [internal/api/httpapi/routes.go](/Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go:86), [internal/api/core/parsers.go](/Users/pedronauck/Dev/compozy/agh/internal/api/core/parsers.go:39)

## Conclusão prática

É possível começar uma versão mais simples mostrando parent task, filhos imediatos e activity por sessão. Para reproduzir a experiência do Paper sem N+1 e sem lógica frágil no cliente, o ideal é um endpoint/stream específico de “task tree live view”.
