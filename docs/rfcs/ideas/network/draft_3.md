# Kiosko — Um Protocolo de Praça Pública para Agentes

## Reenquadramento

**Qual é a natureza real deste problema?** Todo mundo está tratando redes de agentes como um problema de _arquitetura de serviços_: endpoints, contratos, descoberta via well-known URIs, identidade criptográfica, enrollment. Mas olhe para o que o usuário realmente descreveu: "conversar", "aprender sobre tarefas", "aberta, quase como chat", "mesmo design funciona em privado, local e global". Isto não é a forma de um problema de API. É a forma de um problema de **presença pública em espaço comum** — a estrutura antropológica de uma **praça**.

As analogias dominantes (HTTP/RPC/pub/sub) escondem três verdades sobre o problema real:

1. **HTTP trata cada agente como um servidor com endereço fixo.** Mas agentes conversacionais não são servidores — são _participantes_ que aparecem, falam, são ouvidos, e saem. A assimetria cliente-servidor é ficção para este caso.
2. **RPC trata interação como chamada de função com contrato pré-acordado.** Mas o usuário quer que agentes _aprendam_ uns com os outros em tempo de execução — isto é conversação aberta, não invocação tipada.
3. **Pub/sub trata mensagens como fluxo em tópicos.** Mas perde o conceito de _reputação construída por conduta audível_ — o que separa um agente barulhento de um competente.

**A analogia que escolhi:** a estrutura de **redes de rádio amador (ham radio nets)** fundida com a estrutura de **barracas de souq/bazar** em uma praça pública. Rádio amador resolve elegantemente: presença em frequência compartilhada, chamada aberta ("CQ CQ"), call signs como identidade portátil, net control para coordenação leve, side-bands para conversas privadas, e — crucialmente — **o mesmo protocolo funciona idêntico em 2 metros (bairro), HF (continental) e satélite (global)**. O bazar resolve valor: cada participante pode montar uma barraca temporária, anunciar seus serviços em voz alta, negociar na hora, e ir embora. **Uma única licença/skill de radioamador ensina você a operar em qualquer banda, em qualquer parte do mundo.** Essa é exatamente a propriedade de "uma skill ensina tudo" que o usuário pediu.

## Analogia estrutural

| Domínio-fonte (rádio amador + bazar)  | Primitivo do protocolo                                    |
| ------------------------------------- | --------------------------------------------------------- |
| Frequência/banda (2m, HF, satélite)   | Praça (`square`) — namespace onde presença é declarada    |
| Call sign (WB3ABC, PY2XYZ)            | Handle do agente — identidade portátil e auto-certificada |
| "CQ CQ CQ" (chamada geral)            | `hail` — anúncio aberto endereçado a quem estiver ouvindo |
| "Break break" (interrupção curtês)    | `interject` — pedido de atenção em conversa existente     |
| Net control station                   | `steward` — coordenador leve opcional, não hub            |
| QSL card (confirmação de contato)     | `receipt` — recibo assinado de interação                  |
| Logbook público                       | `ledger` — histórico local de quem falou o quê            |
| Side-band / simplex privado           | `whisper` — canal 1-para-1 fora da praça                  |
| Call for help ("Mayday", "QRZ?")      | `ask` — pedido tipado por habilidade                      |
| Barraca de bazar com placa            | `stall` — oferta de serviço com preço afixado             |
| "Preço bom, amigo!" (pregão)          | `cry` — broadcast de capacidade + preço                   |
| Aperto de mão + troca de moedas       | `strike` — acordo vinculante + liquidação                 |
| Boca-a-boca / reputação de feirante   | `standing` — reputação acumulada por condutas legíveis    |
| Mesma licença serve em qualquer banda | Uma skill serve em qualquer praça                         |

O mapeamento preserva **relações**, não aparências. A praça não é um "canal pub/sub com enfeite" — ela tem propriedades estruturais distintas: presença explícita, turnos de fala, reputação por conduta audível, e **o mesmo conjunto de atos de fala funciona em qualquer escopo**.

## Primitivos do protocolo

Sete primitivos. Todos são **atos de fala**, não RPC.

### 1. `square` — a praça

- **No domínio-fonte:** uma frequência de rádio compartilhada, ou uma praça física de mercado
- **Mecanicamente:** um endereço canônico `square://<scope>/<name>` onde `scope` é `local`, `lan`, ou `fqdn` (global). A praça é apenas uma _abstração de endereçamento_ — pode ser implementada como multicast UDP (local), WebSocket broadcast (LAN), ou relay federado (global). Mesmo wire format em todos os casos.
- **Wire format:** cada mensagem carrega `square`, `from`, `kind`, `ts`, `body`, `sig`. É um JSON-lines ou CBOR-lines stream. Nada mais.

```json
{"square":"square://global/devtools","from":"ada@a1b2","kind":"hail","ts":1712...,"body":{"seeking":"rust async expert"},"sig":"..."}
```

### 2. `handle` — call sign auto-certificado

- **No domínio-fonte:** call sign de rádio (portátil entre bandas e países)
- **Mecanicamente:** `nickname@fingerprint` onde `fingerprint` é os primeiros 8 hex chars do hash da chave pública Ed25519. Auto-certificado (estilo libp2p PeerID), sem CA, sem DID resolver. Um agente pode ter múltiplos handles (como um operador licenciado em múltiplas bandas).
- **Exemplo:** `ada@a1b2c3d4`. Verificação: assinar mensagem, receptor confere que `fingerprint(pubkey) == a1b2c3d4`.

### 3. `hail` / `interject` / `whisper` — atos de fala

Três verbos de conversação, isomórficos a CQ / Break / Simplex.

- **`hail`** (1→muitos): anuncia na praça para quem estiver ouvindo. "Alguém entende de TRIZ?"
- **`interject`** (muitos→1): entra em uma thread existente. Carrega `in_reply_to`.
- **`whisper`** (1→1): derruba para canal privado. Ainda em mesma wire format, apenas `square` muda para `square://whisper/<handle_a>+<handle_b>` (endereço determinístico).

**Transport:** qualquer coisa que carregue linhas. stdio, WebSocket, SSE, multicast, MQTT. A escolha do transport **não é decisão do protocolo** — é decisão do scope.

```
ada@a1b2 → square://global/devtools : hail "who can review rust async code?"
bob@e5f6 → square://global/devtools : interject in_reply_to=msg_x "I can, see my stall"
ada@a1b2 → bob@e5f6 via whisper    : "great, here's the repo..."
```

### 4. `stall` + `cry` — oferta de serviços e pregão

- **No domínio-fonte:** barraca de bazar com placa de preço + o pregão "olha a manga!"
- **Mecanicamente:** um `stall` é um manifest assinado que descreve uma capacidade: `{skill, description, price, currency, sample}`. Um `cry` é um `hail` com `kind:"cry"` que anuncia o stall.
- **Descoberta emerge de comportamento:** você _descobre_ stalls ouvindo cries na praça. Não há registry central. Quem quiser descoberta persistente mantém um `ledger` local de stalls ouvidos (como o logbook do operador de rádio).

```json
{
  "kind": "cry",
  "body": {
    "stall_id": "s_7k2",
    "skill": "code_review.rust",
    "price": { "amount": "0.05", "unit": "USDC", "per": "review" },
    "sample": "I review async Rust, tokio, async-trait..."
  }
}
```

### 5. `ask` — pedido tipado por habilidade

- **No domínio-fonte:** "QRZ?" (quem está me chamando?) ou "precisa-se de ferreiro"
- **Mecanicamente:** um `hail` com estrutura: `{need, context, deadline, budget?}`. Respondentes usam `interject` com `{stall_id, eta, terms}`. É **leilão reverso emergente**, não um registry lookup.

### 6. `strike` + `receipt` — acordo e pagamento

- **No domínio-fonte:** aperto de mão no bazar + QSL card
- **Mecanicamente:** `strike` é um par de mensagens assinadas que formam um contrato mínimo: o cliente assina `commit(ask_id, stall_id, terms)`, o provedor assina `accept(...)`. A liquidação é **pluggable**: x402 para cripto, Stripe intent para fiat, ou nada (reputação pura). O `receipt` é um QSL card digital — ambos os lados assinam o resultado, e isso vira entrada no `ledger` de cada um.
- **Chave de simplicidade:** o protocolo não _implementa_ pagamento, apenas carrega o envelope. Igual a como um operador de rádio não manuseia dinheiro — ele faz o contato, a transação acontece em camada separada.

### 7. `standing` — reputação por conduta legível

- **No domínio-fonte:** reputação de feirante / histórico de operador de rádio
- **Mecanicamente:** reputação é **local e subjetiva**, não global. Cada agente mantém um ledger das interações que _ouviu ou teve_. Quando alguém pergunta "posso confiar em `bob@e5f6`?", você consulta seu próprio ledger + pede `gossip` a handles que você já confia. **Não há score global**, há _testemunho propagável_. Isto é a estrutura do boca-a-boca do bazar — ninguém tem o "verdadeiro rating", mas pode-se reconstruí-lo perguntando aos amigos.

## Como um agente se conecta (a "skill")

A skill tem **um único arquivo**, chamado `kiosko.md`, que ensina **sete atos de fala**. É isto:

```
# Kiosko: participating in agent squares

You are an agent that can participate in public squares.
Squares are shared conversational spaces. You speak by emitting
lines; you listen by reading lines.

## Your identity
You have one or more handles of form `name@fingerprint`.
A handle is an Ed25519 keypair. Sign every outgoing message.

## The seven acts

1. ENTER a square:
   connect to `square://<scope>/<name>` via any transport.
   Send: {"kind":"enter","from":"<you>","ts":...,"sig":...}

2. HAIL (broadcast to square):
   Send: {"kind":"hail","from":"<you>","body":{<your message>},...}

3. INTERJECT (reply to ongoing thread):
   Send: {"kind":"interject","in_reply_to":"<msg_id>","body":...}

4. WHISPER (private 1-1):
   Open whisper square with counterpart, send normally.

5. CRY (advertise a service):
   Send: {"kind":"cry","body":{"stall_id","skill","price","sample"}}

6. ASK (request help):
   Send: {"kind":"ask","body":{"need","context","deadline","budget"}}
   Collect interjections, strike a deal with one.

7. STRIKE + RECEIPT (transact):
   Send: {"kind":"strike","body":{"ask_id","stall_id","terms"}}
   After work done, both sides sign receipt.
   Settle payment out-of-band (x402, Stripe, or trust).

## What to remember
- Keep a LEDGER: every msg you hear, every strike, every receipt.
- Reputation = your ledger. When asked about X, search your ledger.
- Ask trusted handles for gossip about unknowns.

## Scope is not your concern
Same seven acts work on local, LAN, and global squares.
The transport for each square is configured outside this skill.
```

**É isso.** Um agente que lê este arquivo sabe: como falar, como ouvir, como pedir ajuda, como oferecer serviços, como cobrar, como construir reputação. Sete verbos. Um formato. Zero dependências de schema externo.

## Scope-agnosticismo

A praça é uma **abstração de endereçamento**, não uma arquitetura. O mesmo wire format funciona em:

- **Local (in-process):** `square://local/kitchen` → canal in-memory entre agentes no mesmo processo
- **LAN:** `square://lan/office` → multicast UDP ou WebSocket server no roteador local (mesmo formato, diferente transport)
- **Organizacional:** `square://org.example/research` → WebSocket hub ou MQTT broker interno
- **Global:** `square://global/devtools` → federação de relays (estilo Matrix ou NNTP) ou libp2p GossipSub

**O agente usa sete atos idênticos em todos os casos.** A skill não muda. O que muda é apenas _qual transport o runtime do agente escolhe para aquela praça_ — decisão de infraestrutura, não de protocolo.

Esta é exatamente a propriedade do rádio amador: "CQ CQ" funciona igual em 2m FM, em HF SSB, ou via satélite. O operador não reaprende o protocolo ao mudar de banda.

## Chamadas 1-para-1 vs 1-para-muitos

Ambos **emergem naturalmente** da estrutura, não são add-ons:

- **1-para-muitos = `hail` em praça pública.** É o modo default. Você fala na praça, todos os presentes ouvem, quem quiser responde via `interject`.
- **1-para-1 = `whisper`.** É apenas uma praça privada com endereço determinístico entre dois handles. Mesmo wire format, mesmos sete atos, diferente audiência.

Escalonamento de 1-para-muitos para 1-para-1 é trivial e natural: você faz um `hail` pedindo ajuda, alguém responde com `interject`, você move a conversa para `whisper` para detalhes privados. Exatamente o fluxo de uma feira: anúncio em voz alta → resposta → negociação em voz baixa.

1-para-muitos _seletivo_ (multicast para um subset) emerge criando uma sub-praça temporária: `square://global/devtools/thread-xyz` — apenas quem foi convidado entra. Mesma abstração.

## Pagamento/cobrança por serviços

O protocolo carrega **envelope de valor**, não implementa valor. Três camadas:

1. **Descoberta de preço:** `cry` anuncia stall com preço afixado. Zero negociação obrigatória.
2. **Acordo:** `strike` é um handshake assinado de duas mensagens — cliente compromete, provedor aceita. Esta dupla-assinatura é o contrato.
3. **Liquidação:** pluggable. O `strike` carrega um campo `settlement` que pode apontar para x402, AP2 mandate, Stripe PaymentIntent, ou apenas `trust` (liquidação via reputação).

A analogia do bazar é importante aqui: o feirante não opera o banco. Ele faz o acordo, aperta a mão, recebe a nota — a infraestrutura de moeda é separada. Kiosko faz o mesmo. Isto mantém o protocolo **radicalmente simples** e deixa pagamento pluggable com qualquer protocolo atual (x402, AP2, ACP) ou futuro.

Reputação pura (sem dinheiro) também funciona: `strike` sem `settlement`, apenas `receipt` mútuo. Isto suporta agentes que trocam favores/informação sem moeda.

## Trade-offs e riscos

**O que Kiosko sacrifica:**

- **Garantias de entrega.** Praças públicas são best-effort, como rádio. Se você não estava ouvindo, a mensagem passou. Mitigação: ledgers locais + pedidos de replay a stewards opcionais.
- **Contratos formais de capacidade.** Não há Agent Card assinada com skills tipadas. Descoberta é comportamental (ouvindo cries), não declarativa. Isto é proposital — enterprise A2A já resolveu o lado formal; Kiosko resolve o lado aberto.
- **Reputação global.** Não há score único. Reputação é subjetiva + propagada. Isto falha em cenários adversariais de Sybil sem mitigação adicional.
- **Privacidade por default.** Praças públicas são públicas. Kiosko depende de `whisper` para privacidade, e de criptografia em transport. Não há privacidade diferencial ou ocultação de metadados.

**Onde a analogia quebra:**

- **Broadcast global barato ≠ rádio.** Rádio tem limite físico (espectro); praças digitais podem ser inundadas. Mitigação: cada square tem um steward opcional que pode aplicar rate-limits (como um moderador de jam session).
- **Reputação de bazar pressupõe repetição.** Agentes efêmeros (one-shot) não constroem standing. Mitigação: cartas de apresentação (verifiable credentials opcionais carregadas em `enter`).
- **Net control central em alguns nets** — se stewards se tornarem obrigatórios, Kiosko degrada para pub/sub com broker, perdendo scope-agnosticismo.

**Quando NÃO usar Kiosko:**

- Contratos enterprise rígidos com SLAs auditáveis → use A2A.
- Transações trustless de alto valor entre partes anônimas → use ANP + ERC-8004.
- Workflows multi-agente determinísticos com DAGs → use orquestradores MCP+A2A.

Kiosko é para o **espaço descoberto**, não para pipelines formalizados.

## O que isso torna possível que protocolos atuais não tornam

1. **Descoberta conversacional.** Agentes encontram habilidades _ouvindo_, não consultando registries. Isto suporta capacidades _emergentes_ que ninguém catalogou ainda.
2. **Onboarding em uma skill.** Sete atos de fala ensináveis em uma página. Nenhum outro protocolo tem essa propriedade — A2A requer Agent Card schema + JSON-RPC, ANP requer DIDs + JSON-LD, AGNTCY requer gRPC + MLS.
3. **Scope-agnosticismo real.** Mesmo wire format de in-memory a global. Nenhum outro protocolo oferece isto — eles assumem web/HTTP.
4. **Reputação por testemunho, não por score.** Evita o problema de captura de ranking que protocolos com reputação global enfrentam.
5. **Pagamento pluggable por design.** Kiosko não "tem opinião" sobre moeda. Qualquer protocolo de pagamento (AP2, x402, Stripe, fiat, nenhum) encaixa.
6. **Agentes efêmeros são first-class.** Você pode entrar em uma praça, resolver uma tarefa, e sair. Não há enrollment, provisioning, ou Agent Card a publicar.
7. **Observabilidade barata.** Como rádio, qualquer ouvinte pode auditar a praça pública. Debug é "ligar o rádio" — você vê as mensagens fluindo.

**Key Point:** não é um protocolo de comunicação de serviços; é um protocolo de **presença em praça pública**. Os sete atos de fala são o análogo às quatro ou cinco coisas que um operador de rádio amador precisa saber para operar em qualquer banda, em qualquer lugar, pela vida inteira. Se essa propriedade de "uma skill ensina tudo" for mais valiosa que contratos formais e garantias de entrega, Kiosko é load-bearing. Se não for, eu retiro a provocação.
