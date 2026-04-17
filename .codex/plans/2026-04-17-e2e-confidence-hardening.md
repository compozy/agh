# Plano: Hardening determinístico da suite E2E

## Resumo

- O follow-up vai manter o foco em confiança determinística, não em provider real nesta rodada.
- O objetivo é remover as duas principais fontes de falsa confiança já confirmadas no repo: roteamento de fixture por substring em prompt renderizado e cobertura limitada a agentes cooperativos.
- O pacote entregável será um hardening único que alinha documentação, contrato de fixture, injeção de falhas e estratégia de build do harness para que `test-e2e-runtime`, `test-e2e-web` e `test-e2e` passem a medir o que a suite realmente promete.

## Mudanças de interface

- `internal/acp.PromptRequest` passa a carregar `Meta` estruturado, com pelo menos `turn_source` e um bloco opcional `network` contendo `message_id`, `kind`, `channel`, `from`, `to`, `interaction_id`, `reply_to`, `trace_id` e `causation_id`.
- `internal/session.PromptOpts` passa a aceitar esse metadata ACP; `Prompt()` continua populando apenas `turn_source=user`, e `PromptNetwork()` passa a preencher `turn_source=network` mais os campos estruturados do envelope.
- O fixture JSON do `internal/testutil/acpmock` sobe para `version: 2` e quebra compatibilidade com v1.
- `TurnMatch` deixa de aceitar `equals` e `contains`; v2 passa a aceitar apenas match exato por `turn_source`, `user_text`, `occurrence` e campos estruturados de `network`.
- `DiagnosticsRecord` do `acpmock` passa a incluir snapshot do metadata recebido e o nome do matcher/turn selecionado.
- `AGH_TEST_DAEMON_BIN` e `AGH_TEST_ACPMOCK_DRIVER_BIN` passam a ser os overrides oficiais do harness para Go e Playwright.

## Mudanças de implementação

- Propagar metadata estruturado até o ACP real: `network/delivery` deixa de depender do XML renderizado para seleção de turn e passa a enviar o envelope normalizado como metadata do prompt; o texto renderizado continua existindo apenas como contexto para o agente.
- Reescrever todos os fixtures existentes para v2, substituindo substring por match exato. Nos cenários de rede, a chave principal passa a ser `message_id`/`kind`/`reply_to` em vez de `contains: kind="..."`.
- Congelar os `StepKind` sem espalhar novos kinds específicos; para falhas de transporte, adicionar um único `driver_control` com payload tipado e ações iniciais `disconnect`, `write_raw_jsonrpc` e `block_until_cancel`.
- Adicionar três cenários negativos obrigatórios no runtime lane: crash no meio do stream, frame ACP inválido durante update e permission request seguida de disconnect. Cada cenário deve validar HTTP, UDS, transcript/events e artifacts, não só o transcript final.
- Tirar o custo de build do caminho crítico da lane: `mage runE2ELane` deve prebuildar `./cmd/agh` e `./internal/testutil/acpmock/cmd/acpmock-driver` uma vez por execução, exportar os paths para subprocessos Go e Playwright, e deixar os builders locais como fallback apenas para execuções avulsas de pacote.
- Manter `MockAgentSpec` como narrow waist do fluxo cooperativo e documentá-lo explicitamente; os cenários patológicos ficam encapsulados no `driver_control` do acpmock, não espalhados em helpers ad hoc.
- Corrigir a verdade documental do task pack: criar `ADR-006` para formalizar o mock em Go e superseder `adr-001`; criar `ADR-007` declarando que nenhuma lane atual cobre provider real; atualizar `_techspec.md`, `task_02.md` e os review notes que ainda citam `aimock` ou `driver/dist/index.js`.

## Testes e validação

- Adicionar testes unitários para serialização do novo `PromptRequest.Meta`, matcher v2, leitura de diagnostics com metadata e `driver_control`.
- Adicionar testes de integração para os três cenários negativos no `internal/daemon` e, quando fizer sentido, assertions de projeção em `internal/api/httpapi` e `internal/api/udsapi`.
- Executar stress run do recorte de transport parity em loop para eliminar a flake observada sob concorrência antes de considerar o hardening concluído.
- Validar as lanes oficiais com `make test-e2e-runtime`, `make test-e2e-web` e `make test-e2e`; manter `make verify` como gate separado nesta rodada, assumindo que a CI continua chamando as lanes E2E explicitamente.
- Confirmar no browser lane que o seed do Playwright usa os mesmos bins prebuildados e que nenhuma spec ainda depende de build local redundante do `acpmock-driver`.

## Assumptions e defaults

- Não entra smoke com provider real agora; isso fica explicitamente documentado como trabalho separado.
- A quebra de fixture v1 é intencional e aceitável, desde que todos os fixtures e testes in-repo sejam migrados no mesmo change.
- O texto renderizado do prompt continua existindo para comportamento do agente e legibilidade de artifacts, mas deixa de ser contrato de roteamento do mock.
- O foco desta rodada é confiança do harness e honestidade do claim E2E; não inclui ampliar escopo funcional do produto nem adicionar novas jornadas de UI fora das já existentes.
