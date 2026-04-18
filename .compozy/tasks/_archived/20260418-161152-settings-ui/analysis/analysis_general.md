# Analysis: General

- Veredito: NAO PRONTO

## O que a tela pede

- Estado do daemon com PID, uptime e bind de socket/HTTP.
- Contadores operacionais como sessoes ativas e concorrencia de agentes.
- Defaults globais de `agent`, `provider` e `environment`.
- Politica global de permissoes e timeout de sessao.
- Acoes de "Open Config" e "Restart Daemon".

## O que ja existe

- O daemon ja expoe um snapshot runtime em `GET /api/daemon/status`.
- O daemon ja expoe health em `GET /api/observe/health`.
- O modelo de configuracao ja possui `defaults`, `limits`, `session.limits.timeout` e `permissions.mode`.
- O `web/` ja tem adaptador seguindo o padrao do projeto para consumir status e health do daemon.

## Gaps para implementar a tela inteira

- `GET /api/daemon/status` nao expoe `defaults.agent`, `defaults.provider`, `defaults.environment`, `permissions.mode` nem `session.limits.timeout`.
- Nao existe endpoint HTTP para leitura/edicao da configuracao global de settings.
- Nao existe endpoint de restart do daemon no namespace HTTP atual.
- "Open Config" hoje depende de acesso direto ao arquivo, nao de endpoint.

## Evidencias

- `internal/api/httpapi/routes.go:180-183` registra apenas `GET /api/daemon/status` no grupo `daemon`.
- `internal/api/core/handlers.go:675-714` mostra que `DaemonStatus` retorna apenas status runtime, bind, sessoes, versao e network.
- `internal/api/contract/responses.go:127-137` define `HealthResponse` e `DaemonStatusResponse` sem payload de configuracao de settings.
- `internal/config/config.go:42-79` define `DefaultsConfig`, `LimitsConfig`, `SessionLimitsConfig` e `PermissionsConfig`.
- `web/src/systems/daemon/adapters/daemon-api.ts:10-24` confirma que o `web/` ja consome `/api/observe/health` e `/api/daemon/status` no padrao atual.

## Conclusao

- Da para montar um card de status parcial do daemon.
- Nao da para implementar a tela desenhada de forma fiel sem criar uma superficie HTTP para settings globais do daemon e uma acao de restart.
