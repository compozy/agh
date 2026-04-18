# Analysis: Network

- Veredito: PARCIAL

## O que a tela pede

- Toggle global do runtime de rede.
- `default_channel`, `port`, `max_payload`, `greet_interval`, `max_replay_age` e `max_queue_depth`.
- Link/acao para abrir a area de network.

## O que ja existe

- O daemon ja tem modelo de configuracao completo para network.
- A API HTTP ja expoe runtime status, peers, channels, mensagens, inbox e send.
- O `web/` ja tem sistema de `network` integrado no padrao atual.

## Gaps para implementar a tela inteira

- O payload de status nao expoe os knobs de configuracao da tela como `max_payload`, `greet_interval`, `max_replay_age` e `max_queue_depth`.
- Nao existe endpoint HTTP para leitura/edicao do bloco de configuracao de network.
- O que existe hoje e excelente para a tela operacional de network, mas nao para a tela de settings.

## Evidencias

- `internal/config/config.go:160-169` define `NetworkConfig`.
- `internal/api/httpapi/routes.go:185-196` registra `/api/network/status`, peers, channels, send e inbox.
- `internal/api/core/network.go:29-178` implementa os handlers runtime de network.
- `internal/api/contract/contract.go:243-267` define `NetworkStatusPayload` com metricas runtime, mas sem os knobs completos da configuracao.
- `web/src/systems/network/adapters/network-api.ts:29-168` confirma que o `web/` ja consome o dominio operacional de network.

## Conclusao

- E possivel reaproveitar bastante da infraestrutura existente para o link "Open network".
- A tela de settings ainda depende de endpoints de configuracao para ficar implementavel de ponta a ponta.
