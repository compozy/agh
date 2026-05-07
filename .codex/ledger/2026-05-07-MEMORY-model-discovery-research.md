Ledger Snapshot: Goal = fechar a estratégia de discovery de modelos por provider/ACP usando Zed, Harnss, Paperclip e `compozy-code`; Now/Next = síntese final, sem mutar arquivos; Open Questions = none.

Sobre o `.codex/ledger`: eu **não gravei o arquivo** porque ainda estamos em **Plan Mode**, e a regra ativa proíbe mutação de arquivos. A mensagem exata que você pediu para persistir está preservada no contexto desta sessão; quando o Plan Mode for encerrado, a primeira ação correta é criar algo como `.codex/ledger/2026-05-07-MEMORY-model-discovery-research.md` e colar o conteúdo exatamente como foi emitido.

**O Que O `compozy-code` Muda Na Minha Recomendação**

O `compozy-code` confirma que o melhor desenho não é “ACP lista tudo”, e sim um **Model Discovery Service com sources plugáveis**.

O SDK de lá tem o padrão que eu copiaria para AGH:

- `ProviderDriver.listModels()` como source runtime/autenticada, com prioridade alta.
- `ModelsDevCatalogSource` como source externa de catálogo, usando `https://models.dev/api.json`, cache de 24h e stale fallback.
- `ModelDiscoveryService` que consulta sources em paralelo, aceita sucesso parcial, falha só quando todas falham, e faz merge por prioridade.
- Source de prioridade alta ganha campos conflitantes; source de prioridade baixa preenche lacunas como contexto, custo e display name.
- API exposta em formato parecido com OpenAI `/v1/models`, mas com metadados próprios em `compozy`.

Referências locais principais:

- [models-dev-source.ts](/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/catalog-sources/models-dev-source.ts:65)
- [model-discovery-service.ts](/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/model-discovery-service.ts:78)
- [routes-models.ts](/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/server/routes-models.ts:43)
- [adapters/types.ts](/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/adapters/types.ts:30)

Eu também chequei o `models.dev` ao vivo hoje, **2026-05-07**: o JSON está grande o bastante para ser útil como catálogo inicial. A amostra atual tem 118 providers, OpenAI com 52 modelos, e campos como `reasoning`, `tool_call`, `limit.context`, `limit.input`, `limit.output` e `cost`. Isso é importante porque o parser do `compozy-code` aceita `supportsReasoning/supports_tools`, mas o schema atual do `models.dev` expõe `reasoning/tool_call`; para AGH eu faria o parser tolerante aos dois formatos.

**Minha Ideia Final Para AGH**

Eu faria um **AGH Model Catalog Service** daemon-owned, agent-manageable e separado do estado ACP de sessão.

A arquitetura final:

1. **Pre-session model catalog**
   Usado antes de criar sessão. Alimenta o dropdown de modelo, CLI, UDS e HTTP. Ele responde: “quais modelos AGH conhece para este provider?”

2. **ACP session config options**
   Usado depois que a sessão existe. Ele responde: “quais opções esta sessão ACP realmente permite agora?”

Essas duas coisas não devem ser misturadas. Zed e Harnss mostram que ACP é forte em `configOptions` session-scoped. Paperclip e `compozy-code` mostram que catálogo pré-sessão precisa ser provider/adapter-owned.

**Sources Que Eu Usaria**

Prioridades recomendadas:

- `provider_live` / `driver`, prioridade 100: official API, CLI runtime, Codex `model/list`, OpenRouter API, Ollama tags, Vercel Gateway, extensões AGH.
- `config`, prioridade política: o `supported_models` atual entra como seed/override/allowlist, não como “verdade do mundo”.
- `catalog:models-dev`, prioridade 50: enriquecimento amplo e fallback, com display name, context window, output limit, tool calling, reasoning boolean e custo.
- `builtin`, prioridade 10: defaults mínimos para offline fresh install.
- `acp_session`, separado do catálogo global: só vale para sessão ativa; se ACP expõe `configOptions`, isso vence a UI da sessão.

Merge: chave canônica `provider_id + model_id`; source de maior prioridade vence valores conflitantes; source de menor prioridade só preenche campos ausentes. Isso é exatamente a parte mais boa do `compozy-code`.

**Como Isso Encaixa No AGH Atual**

Hoje AGH já tem os pedaços iniciais, mas eles estão no lugar errado para discovery completo:

- `ProviderConfig` já tem `default_model`, `supported_models` e `supports_reasoning_effort` em [provider.go](/Users/pedronauck/Dev/compozy/agh/internal/config/provider.go:66).
- A API já expõe providers de sessão com esses campos em [contract.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/contract.go:915) e [conversions.go](/Users/pedronauck/Dev/compozy/agh/internal/api/core/conversions.go:988).
- O ACP driver já captura `SessionModelState.AvailableModels` depois de `session/new`/`session/load` em [client.go](/Users/pedronauck/Dev/compozy/agh/internal/acp/client.go:1210).
- A UI de nova sessão hoje só monta opções a partir de `supported_models` em [use-session-create-dialog.ts](/Users/pedronauck/Dev/compozy/agh/web/src/systems/session/hooks/use-session-create-dialog.ts:145).

Então o próximo passo não é jogar fora isso. É inserir uma camada nova entre config/provider e UI/API:

`ProviderConfig` + `models.dev` + provider live sources + builtin defaults → `ModelCatalogService` → HTTP/UDS/CLI/Web.

**API Que Eu Exporia**

Para ser agent-manageable:

- HTTP: listar modelos por provider, listar todos, forçar refresh, ver source/staleness/error.
- UDS: mesma capacidade para agentes locais.
- CLI: `agh providers models list --provider codex`, `--refresh`, `--source`.
- Web: o modal de nova sessão consome essa API, não `supported_models` direto.

Resposta mínima por modelo:

- `provider_id`
- `model_id`
- `display_name`
- `source`
- `sources`
- `stale`
- `refreshed_at`
- `last_error`
- `context_window`
- `max_input_tokens`
- `max_output_tokens`
- `supports_tools`
- `supports_reasoning`
- `supported_reasoning_efforts`
- `default_reasoning_effort`
- `cost`

**Reasoning**

Eu não manteria `supports_reasoning_effort` como boolean no longo prazo. Ele foi bom para o primeiro corte da branch, mas o modelo correto é per-model/per-provider.

`models.dev` ajuda a dizer “esse modelo tem reasoning?”, mas não é suficiente para níveis como `minimal`, `low`, `medium`, `high`, `xhigh`. Esses níveis devem vir de:

- provider live source, quando souber;
- config explícita;
- ACP `configOptions` da sessão, quando existir;
- fallback conservador.

Ou seja: `models.dev.reasoning = true` habilita UX informativa, mas não deve inventar levels.

**O Que Eu Não Copiaria Literalmente**

Eu não copiaria o `compozy-code` 1:1 por três motivos:

- O mapping de provider é diferente. Lá `anthropic -> claude-code`; em AGH provavelmente é `anthropic -> claude`, `openai -> codex`, `vercel -> vercel-ai-gateway`, etc.
- O parser precisa aceitar o schema atual do `models.dev`: `reasoning`, `tool_call`, `limit.input`, além dos aliases antigos.
- `models.dev` não prova disponibilidade real na conta do usuário. Ele é catálogo e enriquecimento, não autoridade de autorização.

**Conclusão**

Minha ideia final fica assim:

**AGH deve ter um Model Catalog Service próprio, com `models.dev` como source de catálogo ampla e cacheada, provider/runtime sources como autoridade de disponibilidade, `supported_models` como config/policy local, e ACP `configOptions` como verdade apenas da sessão ativa.**

Isso combina o melhor dos quatro lugares:

- Zed: sessão ACP controlada por `configOptions`.
- Harnss: caminhos específicos para providers conhecidos e fallback ACP pragmático.
- Paperclip: adapter registry + refresh/manual model + agent-manageability.
- `compozy-code`: source interface, priorities, cache, stale fallback e merge por enriquecimento.

Fontes externas usadas:

- Models.dev: https://models.dev/
- Context7 Models.dev summary: https://context7.com/anomalyco/models.dev
- ACP session config options: https://agentclientprotocol.com/protocol/session-config-options

Nenhum arquivo foi alterado.
