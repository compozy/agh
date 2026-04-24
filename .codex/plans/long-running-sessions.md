# Long-Running Sessions No Harness AGH

## Summary

Implementar um supervisor de atividade por prompt/sessao inspirado no Hermes: a sessao pode rodar por horas, mas deve manter `last_activity_at` vivo por eventos reais, heartbeats de espera e progresso periodico. Timeouts passam a ser por inatividade, nao por duracao total.

A primeira versao foca em sessoes ACP, task-backed sessions, SSE/API e bridge progress. Nao portar o registry Python/global do Hermes nem dual-write JSONL; a AGH continua usando daemon + SQLite/event store como fonte canonica.

## Key Changes

- Criar um modelo interno `RuntimeActivity`/`SessionActivityMeta` ligado a `SessionLivenessMeta`, com campos: `turn_id`, `turn_source`, `turn_started_at`, `last_activity_at`, `last_activity_kind`, `last_activity_detail`, `current_tool`, `tool_call_id`, `last_progress_at`, `iteration_current`, `iteration_max`, `idle_seconds`.
- Estender persistencia de sessao para carregar/salvar a activity nos metadados e no indice global; corrigir a reconciliacao em `internal/observe/reconcile.go` para nao perder `Liveness`.
- Adicionar configuracao:
  - `[session.supervision] activity_heartbeat_interval = "30s"`
  - `progress_notify_interval = "10m"`
  - `inactivity_warning_after = "15m"`
  - `inactivity_timeout = "30m"`
  - `timeout_cancel_grace = "30s"`
  - `0` desabilita warning/timeout/progress; heartbeat deve ser positivo.
- Adicionar eventos ACP internos:
  - `runtime_progress`: evento persistido/streamado a cada `progress_notify_interval`, com texto equivalente a "Still working..." e payload estruturado de activity.
  - `runtime_warning`: emitido uma vez quando a sessao passa de `inactivity_warning_after`.
  - Timeout por inatividade cancela o prompt cooperativamente; se nao terminar dentro de `timeout_cancel_grace`, para a sessao com `StopTimeout`.
- Nao enviar heartbeats pelo canal ACP de eventos para evitar backpressure. Heartbeats curtos atualizam apenas metadata/liveness; eventos de progresso, warning e timeout sao persistidos e notificados em cadencia baixa.
- Alterar o fluxo de prompt para criar um handle ativo por turno com `turn_id`, cancel func e supervisor. `CancelPrompt`, stop de sessao e timeout usam o mesmo caminho idempotente de cancelamento.
- Atualizar `pumpPrompt` para selecionar entre eventos ACP reais e eventos do supervisor. Eventos reais atualizam `current_tool`/activity; eventos de progresso tambem sao entregues ao prompt stream aberto, ao `/sessions/:id/stream`, ao observer e ao event store.
- Atualizar `acp.PromptRequest` para aceitar um reporter/callback de activity. O driver ACP chama esse reporter enquanto `session/prompt` esta bloqueado esperando provider, sem criar eventos persistidos a cada tick.
- Para tool calls, usar os eventos existentes `tool_call`/`tool_result` para manter `current_tool`; se o provider nao expuser iteracao, deixar `iteration_current/max` ausentes em vez de inventar valores.
- Integrar task runtime: quando `PromptMeta.Synthetic.TaskRunID` ou detached task metadata existir, o supervisor registra heartbeat/progress do run e a recuperacao de boot usa `last_activity_at` em vez de apenas `LastUpdateAt`.
- Integrar bridges sem poluir a resposta final do agente: `runtime_progress` vira uma projecao de progresso separada do conteudo `agent_message`; adapters que nao suportarem progress ignoram o evento de forma explicita, e API/SSE continuam sendo a fonte canonica.
- Expor activity em `SessionPayload` e health/observe: status deve mostrar atividade atual, idade desde ultima atividade, stall/warning state e current tool quando houver.
- Ajustar o harness E2E para observacao long-running: adicionar helper que observa SSE ate predicado/evento e retorna sem exigir EOF ou `done`, mantendo fechamento explicito do body/reader.

## Test Plan

- Unitarios de `SessionActivityMeta`: validacao, clone, persistencia em meta/global DB, reconciliacao preservando `Liveness`, e calculo de idle/progress.
- Unitarios de supervisor: heartbeat atualiza metadata sem gravar evento; progress grava `runtime_progress`; warning emite uma vez; timeout cancela prompt; grace expirada para sessao com `StopTimeout`.
- Unitarios de concorrencia: `CancelPrompt` idempotente, timeout vs stop simultaneo, nenhum duplo `dispatchTurnEnd`, prompt sintetico continua usando fila exclusiva.
- ACP/acpmock: fixture `block_until_cancel` deve simular prompt silencioso; driver reporter deve tocar activity enquanto `SendRequest` esta bloqueado; cancel real deve desbloquear sem orfao.
- API/SSE: `/sessions/:id/stream` recebe `runtime_progress`; reconnect com `Last-Event-ID` nao duplica eventos; prompt HTTP/UDS aberto tambem recebe progress; desconectar cliente nao cancela producao.
- Bridge: progress nao e anexado ao texto final do agente; start/delta/final/error existentes continuam iguais; adapters sem progress nao quebram.
- Task runtime: run em execucao permanece saudavel com heartbeat; boot recovery diferencia running, stalled e orphaned usando `last_activity_at`; cancel/force-stop propaga para sessao.
- Harness E2E: adicionar helper "observe until predicate"; converter o teste de blocked cancel para nao depender de goroutine manual; validar daemon real + acpmock + transcript/event capture.
- Verificacao final obrigatoria:
  - `go test ./internal/session ./internal/acp ./internal/store/... ./internal/api/core`
  - `go test ./internal/testutil/e2e ./internal/testutil/acpmock`
  - `go test -tags integration ./internal/daemon -run 'TestDaemonE2EACPmock.*|TestBootRecovers.*TaskRun.*' -count=1`
  - `make verify`

## Assumptions

- Subagents continuam restritos a analise; implementacao sera feita apenas no agente principal.
- Este corte nao porta o `ProcessRegistry` global do Hermes. O equivalente AGH sera feito via session/task supervision e eventos tipados; um registry completo de processos background pode vir depois com techspec proprio.
- Inactivity timeout nao e wall-clock timeout: uma tarefa de horas e saudavel se continua emitindo activity real ou heartbeat de espera controlado.
- Progress events sao parte do contrato publico novo; clientes antigos podem ignorar tipos desconhecidos, mas os contratos/testes da AGH serao atualizados para renderizar/validar `runtime_progress`.
