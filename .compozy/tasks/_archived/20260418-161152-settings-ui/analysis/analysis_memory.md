# Analysis: Memory

- Veredito: PARCIAL

## O que a tela pede

- Toggle global de memoria.
- `global_dir`.
- Status e configuracao de dream consolidation.
- Campos como `agent`, `min_hours`, `min_sessions`, `check_interval`.
- Acao manual de "Trigger now".

## O que ja existe

- O daemon ja tem modelo de configuracao completo para memoria e dream.
- A API HTTP ja expoe leitura/escrita de documentos de memoria.
- A API HTTP ja expoe `POST /api/memory/consolidate`.
- O health do daemon ja inclui contagem de arquivos e ultimo consolidation timestamp.

## Gaps para implementar a tela inteira

- Nao existe endpoint HTTP que exponha `memory.enabled`.
- Nao existe endpoint HTTP que exponha `memory.global_dir`.
- Nao existe endpoint HTTP que exponha `dream.agent`, `dream.min_hours`, `dream.min_sessions` e `dream.check_interval`.
- Nao existe endpoint HTTP de update para esses campos; hoje so existe o trigger e o CRUD dos arquivos de memoria.

## Evidencias

- `internal/config/config.go:101-115` define `MemoryConfig` e `DreamConfig`.
- `internal/api/httpapi/routes.go:171-177` registra `GET/PUT/DELETE /api/memory/*` e `POST /api/memory/consolidate`.
- `internal/api/core/memory.go:25-143` implementa list/read/write/delete/consolidate.
- `internal/api/core/memory.go:145-181` mostra que o health de memoria expoe apenas contagem de arquivos, `dream_enabled` e `last_consolidation`.
- `internal/api/contract/contract.go:485-491` define `MemoryHealthPayload` sem os campos de configuracao da tela.

## Conclusao

- Esta e uma das telas mais proximas de poder ser implementada.
- Ainda falta uma superficie de settings para ler e atualizar a configuracao de memoria; sem isso a UI ficaria reduzida a health + arquivos + trigger manual.
