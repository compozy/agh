# Extension Bundles and Activation Runtime

## Summary

- Implementar um modelo de `extension bundles` como recurso estático novo da extensão, separado de `skills`, `agents`, `hooks` e `mcp_servers`.
- O bundle vira um catálogo declarativo de perfis de produto/equipa; o operador ativa um perfil explicitamente por escopo/workspace, e o daemon reconcilia os recursos geridos.
- A solução recomendada é um subsistema novo de ativações de bundle, daemon-owned, em vez de fazer a extensão se auto-materializar via Host API dinâmica.
- `skills/agents/hooks/MCP` continuam com o comportamento atual. O novo modelo é aditivo e não quebra o pipeline de registro já existente.
- Disable/uninstall de extensão com bundle ativado deve ser bloqueado até que as ativações sejam removidas/desativadas explicitamente.

## Key Changes

- Estender `internal/extension` para aceitar `resources.bundles []string` no manifesto e carregar arquivos de bundle sob a raiz da extensão, com a mesma disciplina de path resolution usada para os outros recursos.
- Definir um `BundleSpec` com perfis múltiplos por extensão, cobrindo:
  - catálogo de canais declarados com um canal primário
  - jobs/triggers de automação reaproveitando o shape atual de automations, mas permitindo referências a canais declarados do bundle
  - presets de bridge reaproveitando o shape atual de criação de instância, mais metadata de slots de segredo/bindings exigidos
- Adicionar um runtime novo, bootado em `daemon/` depois de extensões e automations, para persistir `BundleActivation` e reconciliar recursos em boot, activate, update, deactivate e reload de extensão.
- Persistir ownership/materialization por ativação: ids determinísticos por `activation_id + resource_key`, inventário de recursos materializados e overlays operacionais separados do spec-base.
- Introduzir catálogo de canais declarados, separado dos canais ativos do runtime de network, para que instaladores vejam “o que o pacote traz” mesmo sem sessões ativas.
- Adicionar um resolvedor de `effective default channel` com base em config + override opcional de ativação. O flag de ativação `bind_primary_channel_as_daemon_default` defaulta para `false`.
- Não mutar `Config.Network.DefaultChannel` em disco. O bind ao default do daemon deve ser persistido como estado operacional do bundle/runtime e exposto por API.
- Em automations, adicionar uma fonte explícita `package` para jobs/triggers geridos por bundle. Não tratá-los como `dynamic`.
- Bloquear mutações diretas do spec-base de recursos `package` nas APIs atuais de automation/bridge; permitir apenas overlays aprovados, como `enabled`, bindings/segredos e seleção de workspace feita na ativação.
- Em bridges, a ativação cria instâncias geridas em estado inicial não operacional (`disabled` por default, com possibilidade de evoluir para `auth_required` quando faltarem bindings/autenticação), sem tentar subir o adapter imediatamente.
- Expor contratos públicos para bridge secret bindings, preferencialmente genéricos para qualquer bridge instance, não só para bundles.
- Não adicionar Host API de escrita para ativar bundles. A ativação permanece operator-driven por CLI/HTTP/UDS; Host API, se tocada, fica no máximo read-only para introspecção futura.

## Public Interfaces / Types

- `internal/extension.Manifest.Resources` ganha `Bundles []string`.
- Novos tipos internos: `BundleSpec`, `BundleProfile`, `DeclaredChannel`, `BridgePreset`, `BundleActivation`, `BundleResourceInventory`.
- `automation.JobSource` ganha `package`.
- Network/API ganha payload de settings efetivos, incluindo `configured_default_channel`, `effective_default_channel` e `effective_default_source`.
- Novos endpoints/contratos HTTP+UDS para:
  - listar catálogo de bundles por extensão
  - pré-visualizar uma ativação
  - ativar, listar, detalhar, atualizar overlays e desativar ativações
  - listar canais declarados por bundle
  - criar/atualizar/remover bridge secret bindings
- Erros públicos novos para:
  - tentativa de editar spec-base de recurso `package`
  - disable/uninstall bloqueado por ativações existentes

## Test Plan

- Testes de manifesto/loader cobrindo `resources.bundles`, arquivos inválidos, escape de path, perfis duplicados, canal primário ausente e slots de segredo inválidos.
- Testes do runtime de ativação cobrindo activate idempotente, reconcile em reload/upgrade, deactivate limpando recursos geridos e bloqueio de disable/uninstall com ativações ativas.
- Testes de network cobrindo catálogo de canais declarados, bind opcional do canal primário como default efetivo e limpeza do override ao desativar a ativação.
- Testes de automations cobrindo `source=package`, ids determinísticos, proteção do spec-base e overlays permitidos.
- Testes de bridges cobrindo criação de instância pendente, bindings de segredo, compatibilidade com o runtime existente de adapter e ausência de autostart indevido.
- Testes de API/CLI cobrindo catálogo, preview, activate/deactivate, mensagens de ownership e erro de lifecycle.
- Verificação final obrigatória: `make verify`.

## Assumptions and Defaults

- Nomear o novo recurso como `bundle` para evitar confusão entre “extensão” e “pacote ativável”.
- Bundle activation é única por `(extension, profile, scope, workspace_id)`; múltiplas ativações só são permitidas quando o tuple muda.
- O operador ativa bundles explicitamente; a instalação/habilitação da extensão não materializa recursos por si só.
- Ownership segue o modelo `owner + overlay`: o bundle continua dono do spec-base e o operador só altera overlays autorizados.
- O bind do canal primário ao default do daemon é opt-in por ativação, nunca automático.
- Disable/uninstall de extensão com bundles ativos é bloqueado, sem cascade implícita.
- `skills`, `agents`, `hooks` e `mcp_servers` permanecem com semântica atual, sem migração forçada para o modelo de bundle.
