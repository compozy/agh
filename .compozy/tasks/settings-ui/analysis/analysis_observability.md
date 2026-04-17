# Analysis: Observability

- Veredito: PARCIAL

## O que a tela pede

- Toggle de captura de eventos.
- `retention_days` e `max_global_bytes`.
- Uso atual de armazenamento.
- Toggle e limites de transcripts.
- `log.level`.
- Um live log tail.

## O que ja existe

- O daemon ja tem modelo de configuracao para observability, transcripts e log level.
- A API HTTP ja expoe health, eventos e stream SSE de eventos.
- O health atual ja entrega tamanhos de DB e counters operacionais.

## Gaps para implementar a tela inteira

- Nao existe endpoint HTTP que exponha `observability.enabled`, `retention_days`, `max_global_bytes`, `transcripts.*` e `log.level`.
- Nao existe endpoint HTTP de live daemon log tail; o stream atual e de eventos observados, nao de logs do processo.
- O health payload atual nao cobre os knobs de configuracao mostrados na tela.

## Evidencias

- `internal/config/config.go:81-99` define `ObservabilityConfig`, `ObservabilityTranscriptConfig` e `LogConfig`.
- `internal/api/httpapi/routes.go:83-87` registra apenas `/api/observe/events`, `/api/observe/events/stream` e `/api/observe/health`.
- `internal/api/core/handlers.go:563-646` implementa lista/stream de eventos observados via SSE.
- `internal/api/core/handlers.go:648-672` mostra que o health agrega apenas `health`, `memory` e `automation`.
- `internal/api/contract/contract.go:152-162` define `ObserveHealthPayload` com uptime, sessoes, agentes e tamanhos de DB, nao com os settings da tela.

## Conclusao

- A tela pode ser parcialmente abastecida com metricas runtime e usage.
- Os controles de configuracao e o live log tail ainda precisam de endpoints novos no daemon.
