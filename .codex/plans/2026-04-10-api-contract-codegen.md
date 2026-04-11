# Fonte Única de Contratos para REST e Extensions

## Resumo

- O problema raiz não é “falta de geração de tipos”; é a ausência de um contrato exportável e verificável a partir do Go. Hoje o backend tem DTOs canônicos em `internal/api/contract`, mas o web ainda espelha shapes manualmente em `web/src/systems/*/types.ts` e o SDK de extensions mantém outro universo paralelo em `sdk/typescript/src/types.ts`.
- A correção v1 tem dois trilhos:
  - REST: gerar OpenAPI a partir do Go e consumir isso no TypeScript.
  - Extensions: gerar TypeScript diretamente dos contratos Go do protocolo/Host API/hooks, porque esse domínio é JSON-RPC e não REST.
- A solução não usa anotações/comentários como segunda fonte de verdade. O spec e os artefatos TS saem do mesmo registro de contratos Go.

## Mudanças principais

- Backend REST:
  - Criar `internal/api/spec` para o registro canônico das operações REST.
  - Declarar, por operação, `method`, `path`, transportes (`http`, `uds`), query params, request body, response de sucesso e erro.
  - Gerar `openapi/agh.json` com `kin-openapi`, usando `openapi3gen` para refletir schemas das structs Go.
  - Incluir endpoints REST de extensions do UDS no mesmo documento, marcados como `uds-only`.
  - Manter fora do OpenAPI os endpoints transport-local: prompt streaming, SSE e envelopes específicos do AI SDK.

- Frontend web:
  - Adicionar `mage Codegen` e `mage CodegenCheck`.
  - Gerar `web/src/generated/agh-openapi.d.ts` a partir de `openapi/agh.json` com `openapi-typescript`.
  - Adotar `openapi-fetch` como cliente REST único do web.
  - Preservar `queryKeys` e `queryOptions`, migrando apenas a camada de consumo para o client gerado.
  - Remover schemas REST duplicados de `web/src/systems/*/types.ts` quando apenas espelham o backend.
  - Não introduzir runtime validation manual nova para REST na v1.

- Extensions / SDK:
  - Extrair contratos Go hoje espalhados em:
    - `internal/subprocess/handshake.go`
    - `internal/extension/host_api.go`
    - `internal/hooks/*`
  - Mover params/resultados de Host API para um pacote de contrato exportado com registro único `method -> params/result`.
  - Criar gerador Go->TS para `sdk/typescript/src/generated/contracts.ts`.
  - Gerar:
    - handshake request/response types
    - `HostAPIMethod`
    - `HostAPIMethodMap`
    - unions de hook events/families
    - `HookPayloadByEvent`
    - `HookPatchByEvent`
  - Reduzir `sdk/typescript/src/types.ts` a utilitários não derivados do Go.
  - Preservar a API pública de `@agh/extension-sdk` via re-exports do arquivo gerado.

- Tooling:
  - Commitar os artefatos gerados.
  - Fazer `mage Verify` falhar quando `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts` ou `sdk/typescript/src/generated/contracts.ts` estiverem stale.
  - Garantir geração determinística para diffs limpos.
  - Não adotar `swaggo`/anotações nem `go-swagger`.

## Mudanças em interfaces públicas

- Novo artefato público REST: `openapi/agh.json`.
- Novo artefato gerado para o web: `web/src/generated/agh-openapi.d.ts`.
- Novo artefato gerado para o SDK: `sdk/typescript/src/generated/contracts.ts`.
- O pacote `@agh/extension-sdk` mantém a mesma superfície pública, mas passa a reexportar tipos gerados.
- O web deixa de depender de DTOs REST escritos à mão.

## Testes e critérios de aceite

- Drift canary para `SessionPayload`: o contrato gerado para o web deve refletir `workspace_id`/`workspace_path` opcionais e `stop_reason`/`stop_detail`.
- Golden test REST: o OpenAPI gerado contém todas as operações registradas com schemas corretos.
- Golden test SDK: o TS gerado bate com o registro Go de Host API e com `internal/hooks.AllHookEvents()`.
- Build do web: compila sem importar DTOs REST manuais antigos.
- Build/test do SDK: initialize/shutdown/Host API continuam verdes com tipos gerados.
- Boundary test: prompt SSE e payloads UI-local continuam fora do OpenAPI e fora do gerador REST.
- Verificação final: `make verify` cobre também a checagem de stale codegen.

## Assumptions e defaults

- A entrega é incremental; não vamos migrar o layer HTTP inteiro para Huma agora.
- OpenAPI é a fonte única apenas para REST.
- Extensions entram nesta iniciativa por um pipeline irmão Go->TS.
- Runtime Zod gerado para REST fica fora da v1; se necessário depois, a extensão correta é usar o mesmo OpenAPI com Orval, nunca voltar para schemas manuais.
