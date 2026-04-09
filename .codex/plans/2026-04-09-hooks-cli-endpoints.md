# Completar `agh hooks` no CLI e no Transporte

## Summary

- A causa raiz é estrutural: a entrega anterior parou em handlers HTTP locais, com filtros parciais e sem fechar a superfície compartilhada. Hoje:
  - o catálogo só filtra `workspace` e `agent`
  - os runs não filtram `outcome`
  - a taxonomia de eventos não aceita filtros
  - o `DaemonClient` não expõe hooks
  - o CLI não tem comando `hooks`
  - o UDS não registra `/api/hooks/*`, então mesmo um client novo falharia com `404`
- A correção deve ser feita de ponta a ponta: ampliar os tipos/filtros de domínio, mover a lógica de endpoint de hooks para a camada compartilhada de API, registrar as rotas também no UDS, adicionar os 3 métodos no client e expor `agh hooks list|info|events|runs`.
- `hooks sources` fica removido do escopo. `hooks info <name>` retorna todos os hooks resolvidos com aquele nome, em ordem de catálogo.

## API & Types

- Adicionar em `contract` os DTOs de query compartilhados:
  - `HookCatalogQuery { Workspace, Agent, Event, Source, Mode string }`
  - `HookRunsQuery { Session, Event, Outcome, Since string; Last int }`
  - `HookEventsQuery { Family string; SyncOnly bool }`
- Adicionar aliases no client para esses tipos e para os records:
  - `HookCatalogRecord = contract.HookCatalogPayload`
  - `HookRunRecord = contract.HookRunPayload`
  - `HookEventRecord = contract.HookEventPayload`
- Ampliar os tipos de domínio para suportar os filtros sem hacks:
  - `hookspkg.CatalogFilter` ganha `Event HookEvent`, `Source *HookSource`, `Mode HookMode`
  - `hookspkg.EventFilter` novo com `Family HookEventFamily` e `SyncOnly bool`
  - `store.HookRunQuery` ganha `Outcome HookRunOutcome`
  - adicionar `Validate()` para `HookRunOutcome` e `HookEventFamily`
- Ampliar o catálogo para suportar `hooks info` sem endpoint dedicado:
  - `hookspkg.CatalogEntry` ganha `ExecutorKind HookExecutorKind`
  - `contract.HookCatalogPayload` ganha `ExecutorKind string`
- Manter `last` como nome público do filtro de hooks. O handler traduz `last` para `store.HookRunQuery.Limit`; não renomear o restante da API para `last`.

## Transport Changes

- Extrair a lógica de hooks de `httpapi` para `internal/api/core`, com métodos compartilhados em `BaseHandlers`:
  - `HookCatalog`
  - `HookRuns`
  - `HookEvents`
- Registrar as mesmas três rotas em ambos os transportes:
  - `GET /api/hooks/catalog`
  - `GET /api/hooks/runs`
  - `GET /api/hooks/events`
- HTTP/UDS devem usar a mesma implementação compartilhada; não duplicar parsing, validação nem payload mapping entre `httpapi` e `udsapi`.
- Regras exatas de parsing/validação:
  - `catalog`: aceitar `workspace`, `agent`, `event`, `source`, `mode`; resolver `workspace` como hoje; validar `event`, `source` e `mode` antes de consultar o observer
  - `runs`: exigir `session`; aceitar `event`, `outcome`, `since`, `last`; validar sessão, `event`, `outcome`, `last >= 0`; `since` na API continua timestamp absoluto (`RFC3339`/`RFC3339Nano`)
  - `events`: aceitar `family` e `sync_only`; validar `family`; parsear `sync_only` com `strconv.ParseBool`
- O observer/core passa a aceitar filtro de eventos:
  - `QueryHookEvents(ctx, filter hookspkg.EventFilter)`
- O store/session DB passa a filtrar `hook_runs` também por `outcome`, preservando a ordenação cronológica ascendente na resposta final mesmo quando `last` é usado.

## CLI

- Registrar `newHooksCommand(deps)` no root command.
- Adicionar no `DaemonClient`:
  - `HookCatalog(ctx, HookCatalogQuery) ([]HookCatalogRecord, error)`
  - `HookRuns(ctx, HookRunsQuery) ([]HookRunRecord, error)`
  - `HookEvents(ctx, HookEventsQuery) ([]HookEventRecord, error)`
- Implementar builders de query string no client:
  - `event`, `source`, `mode`, `outcome`, `since`, `last`, `family`, `sync_only`
- Implementar `internal/cli/hooks.go` com 4 subcomandos:
  - `list`
    - flags: `--workspace`, `--agent`, `--event`, `--source`, `--mode`
    - tabela humana: `Order Name Event Source Mode Priority`
    - JSON: slice completo de `HookCatalogRecord`
    - Toon: `hooks[n]{order,name,event,source,skill_source,mode,required,priority}`
  - `info <name>`
    - flag: `--workspace`
    - faz `HookCatalog`, filtra por nome no cliente e retorna todos os matches
    - human: um bloco por hook com cabeçalho e seções para campos principais, `Matcher` e `Metadata`
    - JSON: slice completo de `HookCatalogRecord` já filtrado pelo nome
    - Toon: array `hooks[n]{name,event,source,skill_source,mode,required,priority,timeout_ms,executor_kind}` seguido dos blocos `matcher[...]` e `metadata[...]`
  - `events`
    - flags: `--family`, `--sync-only`
    - tabela humana: `Event Family Sync Payload Patch`
    - JSON: slice completo de `HookEventRecord`
    - Toon: `events[n]{event,family,sync_eligible,payload_schema,patch_schema}`
  - `runs`
    - flag obrigatória: `--session`
    - flags opcionais: `--event`, `--outcome`, `--since`, `--last`
    - `--since` continua aceitando RFC3339 ou duração relativa no CLI; o comando converte para timestamp absoluto antes de chamar o client
    - tabela humana: `Hook Event Outcome Duration Error`
    - JSON: slice completo de `HookRunRecord`
    - Toon: `runs[n]{hook_name,event,outcome,duration_ms,error,recorded_at}`
- Não adicionar `--agent` nem `--source` em `info`; o comportamento acordado para colisão de nome é mostrar todos.

## Test Plan

- `internal/store/sessiondb`
  - filtra `HookRunQuery` por `outcome`
  - combina `event + outcome + since + last`
  - mantém ordenação cronológica ascendente após aplicar `last`
- `internal/hooks` / `internal/observe`
  - `CatalogFilter` filtra por `event`, `source` e `mode`
  - `EventFilter` filtra por `family` e `sync_only`
  - `CatalogEntry` expõe `ExecutorKind`
- `internal/api/httpapi`
  - `catalog` propaga `event/source/mode`
  - `runs` propaga `outcome/since/last`
  - `events` propaga `family/sync_only`
  - casos inválidos retornam `400`
  - integração continua cobrindo resposta real de `/api/hooks/*`
- `internal/api/udsapi`
  - atualizar teste de rotas para incluir os 3 endpoints de hooks
  - adicionar pelo menos um smoke test de handler para garantir que hooks funcionam no transporte usado pelo CLI
- `internal/cli/client_test.go`
  - cobrir os 3 métodos novos
  - verificar encoding de `last`, `sync_only`, `source`, `mode`, `outcome`
- `internal/cli/hooks_test.go`
  - `list`, `info`, `events`, `runs` em `human`, `json` e `toon`
  - `info` retorna múltiplos matches do mesmo nome
  - `runs` falha sem `--session`
  - `runs --since 5m` vira timestamp absoluto antes da chamada
- Fechamento: rodar testes focados das áreas alteradas e depois `make verify`

## Assumptions

- `hooks sources` foi removido da spec e não será implementado agora.
- `Order` no catálogo continua sendo ordem do pipeline dentro de cada evento; como `event` sempre é exibido, não há necessidade de redefinir isso como ordem global.
- A API pública de hooks aceita `since` absoluto; duração relativa é responsabilidade do CLI.
- Não haverá endpoint `info`; `info` é uma composição de `HookCatalog` + filtro por nome no cliente.
