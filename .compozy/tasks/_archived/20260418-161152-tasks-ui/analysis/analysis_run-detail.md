# AGH Tasks — Run Detail

## Veredito

Parcial e startável, mas a lateral de métricas da tela depende hoje de derivação no cliente. Não existe um endpoint de detalhe de run com dados operacionais agregados.

## O que já está disponível

- A run em si já está representada por `TaskRunPayload`, com `attempt`, `session_id`, `claimed_by`, timestamps, erro e resultado. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:57)
- A task expandida já devolve todas as runs da tarefa, o que permite localizar a run selecionada. Evidência: [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:88)
- A transcrição e o histórico da sessão já existem para montar a coluna principal da tela. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1389), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1412), [internal/transcript/transcript.go](/Users/pedronauck/Dev/compozy/agh/internal/transcript/transcript.go:39)
- `Kill run` é componível com `POST /api/task-runs/{id}/cancel` e parada da `session` associada via `DELETE /api/sessions/{id}`. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1746), [internal/api/httpapi/routes.go](/Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go:67)

## Lacunas para bater com o design

- Não existe um endpoint `GET /api/task-runs/{id}` com detalhe consolidado da run. A UI teria que buscar a task, encontrar a run e depois ir na sessão vinculada. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1605)
- O botão `Pause` não tem suporte no domínio nem na API de tasks/runs. Evidência: [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1646), [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go:1746)
- Métricas como `tool calls`, `input tokens`, `output tokens`, `step 4 of ~6` e `% progress` não são devolvidas em um payload pronto. Parte disso pode ser inferida dos eventos de sessão, mas não existe contrato específico para essa lateral. Evidência: [internal/transcript/transcript.go](/Users/pedronauck/Dev/compozy/agh/internal/transcript/transcript.go:39), [internal/api/core/parsers.go](/Users/pedronauck/Dev/compozy/agh/internal/api/core/parsers.go:15)
- `Skill used` não é dado de primeira classe da run. Isso possivelmente pode ser inferido dos eventos/transcript, mas não há campo dedicado.

## Conclusão prática

A tela pode nascer já com transcript, status e ações básicas. Para ficar fiel ao Paper, vale criar um read model de `task run detail` com métricas agregadas e um comando explícito para interrupção/pausa de run.
