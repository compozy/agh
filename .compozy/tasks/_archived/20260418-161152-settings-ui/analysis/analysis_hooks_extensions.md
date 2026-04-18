# Analysis: Hooks & Extensions

- Veredito: NAO PRONTO

## O que a tela pede

- Lista de hooks com nome, evento, modo, matcher e estado.
- Acao para criar/editar/ativar hooks.
- Configuracao de marketplace e politica de extensoes.
- Visao operacional de extensoes instaladas.

## O que ja existe

- O daemon ja tem modelo de configuracao para declaracoes de hooks.
- A API HTTP ja expoe catalogo de hooks resolvidos, historico de runs e taxonomia de eventos.
- O contrato de extensoes existe, assim como rotas UDS para listar/instalar/habilitar/desabilitar extensoes.
- O OpenAPI e os tipos gerados do `web/` tambem conhecem `/api/extensions`.

## Gaps para implementar a tela inteira

- Hooks no HTTP sao somente leitura; nao ha create/update/enable/disable para hooks.
- O servidor HTTP nao registra `registerExtensionRoutes`, entao `/api/extensions` nao esta disponivel no transporte que o `web/` usa.
- A configuracao de extensoes (`marketplace`, `allowed_kinds`, `max_scope`, rate limits) permanece somente no config, sem endpoint de settings.
- Existe um gap de paridade entre OpenAPI/types e o transporte HTTP real para extensoes, o que bloqueia uma integracao segura no `web/`.

## Evidencias

- `internal/config/hooks.go:13-30` define `HooksConfig` e os campos base de declaracao.
- `internal/config/config.go:139-157` define `ExtensionsConfig` e `ExtensionsResourcesConfig`.
- `internal/api/httpapi/routes.go:90-95` registra apenas `GET /api/hooks/catalog`, `GET /api/hooks/runs` e `GET /api/hooks/events`.
- `internal/api/core/handlers.go:493-560` implementa apenas leitura para hooks.
- `internal/api/httpapi/routes.go:11-27` nao chama `registerExtensionRoutes`.
- `internal/api/udsapi/routes.go:225-233` registra `/api/extensions` no transporte UDS.
- `internal/api/contract/contract.go:425-442` e `internal/api/contract/responses.go:204-212` definem payloads de extensoes.
- `openapi/agh.json:6876-7197` declara `/api/extensions` e `/api/extensions/{name}` no contrato.
- `web/src/generated/agh-openapi.d.ts:390-424` mostra que o cliente gerado do `web/` tambem acredita nessas rotas.

## Conclusao

- Hooks possuem observabilidade pronta, mas nao superficie de administracao.
- Extensions possuem contrato e UDS, mas nao HTTP. Para essa tela, isso e um bloqueio real e precisa ser corrigido antes da implementacao do `web/`.
