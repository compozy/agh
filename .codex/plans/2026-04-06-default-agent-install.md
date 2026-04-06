# Plano: Bootstrap Inicial com `agh install` e Default Agent `general`

## Resumo

- Corrigir a inconsistência de origem: hoje existe `defaults.agent = "coder"` no core, mas nenhum bootstrap instala esse agent e as interfaces ainda exigem `agent_name` explícito.
- Adotar um fluxo explícito de primeiro setup com `agh install`, usando um wizard em Bubble Tea para gerar `~/.agh/config.toml` e garantir a existência de `~/.agh/agents/general/AGENT.md`.
- Tornar o default realmente funcional: criação de sessão sem `--agent` / `agent_name` passa a usar `defaults.agent`, e agentes podem omitir `provider` porque o runtime resolve via defaults de config.

## Abordagem Escolhida

- Abordagem recomendada: bootstrap explícito via `agh install` + resolução de defaults no runtime.
- Rejeitar hardcode de um provider fixo no agent `general`; o provider deve ser escolhido pelo usuário no wizard, não embutido magicamente no binário.
- Rejeitar heurística invisível de auto-instalação no primeiro uso; isso mascararia estado e continuaria produzindo comportamento implícito difícil de entender e testar.

## Mudanças de Interface e Comportamento

- Adicionar comando raiz `agh install`.
- `agh install` deve:
- Garantir `~/.agh` e subdirs via `EnsureHomeLayout`.
- Abrir um wizard Bubble Tea.
- Perguntar `provider` e `model`.
- Pré-carregar valores existentes quando `~/.agh/config.toml` já existir.
- Gravar um `config.toml` canônico do usuário.
- Criar `~/.agh/agents/general/AGENT.md` apenas se ele não existir.
- Mudar o default embutido de `defaults.agent` para `general`.
- Adicionar `defaults.provider` ao schema de config.
- Não adicionar `defaults.model`; o model escolhido no wizard deve ser persistido em `[providers.<provider>].default_model`, reaproveitando o mecanismo que já existe.
- Mudar a resolução de agent para esta ordem:
- `agent.name` continua obrigatório.
- `agent.provider` passa a ser opcional.
- Provider efetivo: `agent.provider` -> `config.defaults.provider` -> erro explícito com orientação para rodar `agh install`.
- Model efetivo: `agent.model` -> `provider.default_model` configurado -> erro apenas se o provider exigir model e ele continuar vazio.
- Mudar o default global de permissões para `approve-all`.
- `agh session new` deixa de marcar `--agent` como obrigatório; quando ausente, usa `config.defaults.agent`.
- `POST /api/sessions` em HTTP e UDS passa a aceitar `agent_name` ausente e usar o mesmo fallback de `config.defaults.agent`.
- Atualizar o cliente/web para refletir isso:
- Tipos/adapters aceitam `agent_name` opcional no create session.
- Estado vazio de agents no sidebar deixa de dizer só “No agents loaded” e passa a instruir o usuário a rodar `agh install`.

## Implementação

- Config:
- Estender `DefaultsConfig`, `defaultsOverlay`, validação e testes para suportar `provider`.
- Atualizar `DefaultWithHome` para `agent = "general"` e `permissions.mode = "approve-all"`.
- Criar writer canônico para `~/.agh/config.toml` a partir da config carregada/editada pelo wizard.
- Agent definition:
- Relaxar `AgentDef.Validate()` para não exigir `provider`.
- Manter `model` opcional.
- Criar template gerenciado para `general/AGENT.md` com prompt genérico e sem `provider`/`model`.
- Resolver runtime/session/API:
- Centralizar uma função de “effective agent name” para CLI/API/session manager, evitando duplicação.
- Melhorar mensagens de erro quando o agent default não existir ou o provider default não estiver configurado.
- Install wizard:
- Implementar modelo Bubble Tea simples, focado em formulário de bootstrap.
- Campo `provider`: seleção entre providers built-in, acrescida de providers já presentes no config atual.
- Campo `model`: input editável, pré-preenchido com o `default_model` atual do provider selecionado.
- Ao salvar: atualizar `defaults.agent = "general"`, `defaults.provider = <selecionado>`, `providers.<provider>.default_model = <model>`, `permissions.mode = "approve-all"`.
- Reexecução: reabrir o wizard com os valores atuais; não sobrescrever `general/AGENT.md` existente.
- Documentação:
- README e `config.toml` de exemplo passam a usar `general`.
- Quick start deve começar com `agh install` antes de `agh daemon start`.
- Exemplo de `AGENT.md` deve refletir provider names reais do sistema e explicar que `provider`/`model` podem ser omitidos.

## Testes

- Unit tests de config para:
- `defaults.provider` em load/merge/validate.
- `DefaultWithHome()` com `general` e `approve-all`.
- writer canônico de `config.toml`.
- Unit tests de agent/provider para:
- `AGENT.md` sem `provider`.
- resolução `agent.provider` -> `defaults.provider`.
- `model` vindo de `providers.<provider>.default_model`.
- Unit tests do wizard/model de install para:
- valores iniciais em fresh install.
- reconfigure com config existente.
- persistência correta no `config.toml`.
- criação idempotente de `general/AGENT.md`.
- Tests de CLI/API para:
- `agh session new` sem `--agent`.
- `POST /api/sessions` sem `agent_name`.
- erro acionável quando o bootstrap não foi feito.
- presença do comando `agh install` na árvore Cobra.
- Web tests para:
- adapter de create session aceitando payload sem `agent_name`.
- empty state instruindo `agh install`.
- Validação final:
- `make verify` obrigatório.

## Assumptions e Defaults

- `agh install` v1 será interativo via Bubble Tea; não inclui modo non-interactive neste escopo.
- O `config.toml` será reescrito em formato canônico; preservar comentários/ordem manual não faz parte deste escopo.
- O agent `general` é tratado como bootstrap gerenciado pelo AGH; se já existir, o install não o modifica.
- O default de permissão do produto passa a ser `approve-all`, coerente com o objetivo de operação autônoma contínua.
