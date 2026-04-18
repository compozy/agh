# Analysis: Providers

- Veredito: NAO PRONTO

## O que a tela pede

- Lista de providers disponiveis com vendor/nome.
- `command`, `default_model` e `api_key_env`.
- Estado operacional por provider, como instalado, binario ausente ou nao configurado.
- Sinalizacao clara de provider default.

## O que ja existe

- O daemon ja tem um registro de providers builtin.
- O daemon ja sabe resolver overrides de provider a partir da config.
- O daemon ja usa `defaults.provider` para resolver agentes.

## Gaps para implementar a tela inteira

- Nao existe `GET /api/providers` nem outro endpoint HTTP equivalente para listar providers ACP.
- Nao existe endpoint para verificar presenca de `APIKeyEnv` no ambiente nem se o binario/command esta instalado.
- O endpoint existente `/api/bridges/providers` nao atende essa tela; ele pertence ao dominio de bridges, nao ao catalogo de providers ACP.
- `/api/agents` lista definicoes de agentes, nao o catalogo consolidado de providers.

## Evidencias

- `internal/config/provider.go:10-16` define o shape de `ProviderConfig`.
- `internal/config/provider.go:39-70` define os providers builtin atuais.
- `internal/config/provider.go:81-109` resolve provider por nome usando builtin + overrides.
- `internal/config/provider.go:111-182` usa `defaults.provider` na resolucao de agentes.
- `internal/api/httpapi/routes.go:34-49` mostra que o unico endpoint com nome "providers" no HTTP hoje e `GET /api/bridges/providers`.
- `internal/api/httpapi/routes.go:77-80` mostra que `/api/agents` lista agentes, nao providers.
- `internal/api/httpapi/routes.go:11-27` nao registra nenhum grupo `/providers`.

## Conclusao

- O modelo interno existe, mas nao existe superficie HTTP para a UI.
- Antes de implementar essa tela no `web/`, o daemon precisa expor um catalogo de providers com status operacional consolidado.
