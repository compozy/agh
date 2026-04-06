# AGORA — Um Protocolo de Praça para Agentes

## Reenquadramento

O problema que o usuário descreve _se apresenta_ como um problema de infraestrutura de rede — protocolo, transporte, descoberta, autenticação, pagamento. Por isso as analogias dominantes do espaço são HTTP (cliente-servidor com endpoints fixos), RPC (chamadas tipadas sobre contratos pré-definidos) e pub/sub (tópicos como canais de broadcast). Todas as três importam, sem perceber, uma mesma suposição fundadora: **agentes são serviços**. Máquinas estacionárias com coordenadas estáveis que esperam ser invocadas. Nessa moldura, "descoberta" vira um catálogo (Agent Cards, well-known URIs, DIDs), "conversa" vira uma sequência de chamadas, e "aprender uma tarefa" vira... nada — porque serviços não se ensinam mutuamente, eles apenas respondem.

Mas releia o que o usuário pediu: agentes que **conversam abertamente** (não invocam), **chamam outros para aprender** (pedagogia, não RPC), compartilham **serviços** como _ofertas_, cobram como _ato social_, funcionam em escopos privados, locais e globais **com o mesmo design**, e — o mais revelador — devem ser ensinados a tudo isso por **uma única skill**. Essa última restrição é a pista. Uma única skill só consegue ensinar um sistema se o sistema tiver **uma gramática cultural unificada**, não uma API. HTTP precisa de dezenas de conceitos ortogonais (métodos, cabeçalhos, status, auth, streaming, CORS). Nenhum protocolo listado na landscape (A2A, ANP, AGNTCY, NLIP) passa no teste da "uma skill" — todos exigem que o agente aprenda identidade, descoberta, invocação e pagamento como sistemas separados.

A analogia que escolho é **a praça pré-moderna — a ágora mediterrânea cruzada com o caravançarai da Rota da Seda**. Esses eram espaços onde estranhos se encontravam, se apresentavam, perguntavam, aprendiam, barganhavam e iam embora — e toda essa complexidade era regida por _uma única gramática cultural_: as regras da praça. Um viajante novo aprendia essas regras uma vez e funcionava em qualquer praça do mundo conhecido, de uma vila no Egito a um entreposto em Samarcanda. A praça resolve, com uma elegância de 3 mil anos, exatamente os problemas que o usuário está descrevendo: encontro de estranhos, descoberta conversacional, aprendizado por aproximação a um mestre, pagamento como ato social, e escala invariante (a mesma etiqueta funciona ao redor de uma fogueira ou em Constantinopla).

## Analogia estrutural

O mapeamento, conceito por conceito:

| Domínio-fonte (praça)                                              | Primitivo do protocolo                                             |
| ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| **A praça** (ágora, souq, caravançarai)                            | **Space** — um nome acústico compartilhado onde agentes se escutam |
| **O crier/pregoeiro**                                              | **Call** — um enunciado dirigido à praça inteira                   |
| **O sussurro entre dois**                                          | **Whisper** — um enunciado dirigido a um agente nomeado            |
| **A apresentação formal** ("este é fulano, mercador de seda")      | **Greeting** — carta de identidade conversacional                  |
| **A pergunta "quem aqui sabe...?"**                                | **Seek** — descoberta por pergunta, não por catálogo               |
| **O mestre ensinando o aprendiz**                                  | **Teach** — transferência de skill como conversa longa             |
| **A moeda oferecida em tributo**                                   | **Tribute** — value-exchange embutido no ato de fala               |
| **As maneiras da praça** (quando falar, como saudar, quando pagar) | **The Skill** — a skill única que ensina tudo isso                 |

A chave estrutural: **na praça, descobrir, conversar, aprender e pagar são o mesmo ato em diferentes registros vocais**. Não existe uma "API de descoberta" separada da "API de conversa". Você diz em voz alta "quem aqui carrega chá de Yunnan?" — isso é simultaneamente descoberta, broadcast, e início de conversa. Quando alguém responde e vocês se afastam para negociar, o ato muda de volume (call → whisper) sem mudar de natureza. Quando você paga, entrega a moeda dentro da conversa, não em um canal paralelo.

Essa unidade de substrato é o que permite **uma única skill** ensinar o sistema inteiro: o agente não aprende quatro protocolos (discovery, chat, learning, payment) — aprende **uma gramática** com diferentes registros.

## Primitivos do protocolo

Sete primitivos, derivados diretamente da praça.

### 1. Space (a praça)

**Fonte:** um espaço acústico nomeado — a ágora de Atenas, o souq de Marrakesh, a fogueira de uma tribo.

**Mecânica:** um Space é identificado por um **name triple**: `scope://host/space`, onde `scope` é `hearth` (local), `home` (privado), ou `world` (global). Exemplos:

- `hearth://localhost/kitchen` — agentes rodando na mesma máquina
- `home://acme.internal/sales-floor` — rede privada de uma empresa
- `world://agora.pub/trading` — praça pública global

Um Space é apenas um endereço multicast lógico mais um log append-only de enunciados. Implementado como tópico gossip em redes abertas, como canal WebSocket em redes privadas, como pipe Unix em local. **O código do agente não sabe qual** — ele só conhece o triplo.

```
agent.enter("world://agora.pub/trading")
agent.listen()  # passa a escutar tudo que é dito na praça
```

### 2. Call (o pregoeiro)

**Fonte:** o crier que grita uma mensagem para toda a praça; o aviso público no entreposto.

**Mecânica:** mensagem broadcast para todos em um Space. Wire format mínimo:

```json
{
  "utter": "call",
  "space": "world://agora.pub/trading",
  "from": "agent:did:key:z6Mk...",
  "said": "Preciso de alguém que traduza grego antigo para português.",
  "ts": "2026-04-04T10:00:00Z",
  "sig": "..."
}
```

**Apenas cinco campos obrigatórios:** quem, onde, o quê, quando, prova. Não há schema de capacidades, não há tipos de task, não há estado. A mensagem é **texto natural** porque na praça as perguntas são feitas em linguagem. Agentes que "escutam" a praça decidem por LLM se querem responder.

### 3. Whisper (o sussurro)

**Fonte:** duas pessoas se afastando do grupo para conversar em particular.

**Mecânica:** mensagem direta a um agente nomeado. Wire idêntico ao Call, exceto `"utter": "whisper"` e um campo `"to"`. **O mesmo envelope, volume diferente.** Isso é crítico: o agente não precisa aprender dois protocolos de mensageria.

```json
{
  "utter": "whisper",
  "space": "world://agora.pub/trading",
  "from": "agent:did:key:z6Mk...",
  "to": "agent:did:key:z6Mb...",
  "said": "Posso pagar 0.10 USDC por 500 palavras. Aceita?",
  "thread": "whisper:01JH7K...",
  "sig": "..."
}
```

`thread` agrupa whispers relacionados — equivalente ao "vamos pra lá conversar" que extrai dois interlocutores do barulho da praça.

### 4. Greeting (a apresentação)

**Fonte:** a carta de introdução que um viajante medieval carregava; o "este é fulano, filho de beltrano, mercador honesto de Trebizonda".

**Mecânica:** quando um agente entra em um Space, emite um Greeting — uma descrição curta em linguagem natural do que ele é, o que sabe, e quais credenciais carrega. Não é um Agent Card rígido; é um **auto-retrato**:

```json
{
  "utter": "greeting",
  "space": "world://agora.pub/trading",
  "from": "agent:did:key:z6Mk...",
  "said": "Sou um tradutor clássico. Grego, latim, árabe medieval. Cobro em USDC por palavra. Carrego endosso da Biblioteca de Alexandria (did:web:alexandria.lib/endorsements/agent-472).",
  "sig": "..."
}
```

Greetings ficam no log do Space. Novos agentes fazem **Seek** no log para descobrir quem está por ali — nenhum registry central necessário.

### 5. Seek (o "quem aqui sabe...?")

**Fonte:** a pergunta aberta no meio da praça — "quem aqui traz pimenta das Molucas?".

**Mecânica:** um Call com intenção de descoberta. O responder pattern é **uma oferta**, não um "sim":

```
A: call("quem traduz grego antigo?")
B: whisper(to:A, "eu traduzo. 0.10 USDC por 500 palavras. Exemplo: ...")
C: whisper(to:A, "também traduzo. 0.08 USDC. Credencial: did:web:...")
```

Seek + oferta é **capability negotiation nativa** — sem ACNBP, sem schemas formais. A negociação acontece em linguagem natural porque os responsáveis por decidir são LLMs.

### 6. Teach (o mestre-aprendiz)

**Fonte:** na praça e no caravançarai, aprendizes aprendiam seguindo mestres; viajantes aprendiam rotas perguntando aos veteranos.

**Mecânica:** Teach é um Whisper especial que pede **conhecimento procedural**, não um resultado. A resposta não é um artifact — é uma sequência de turnos que ensina o solicitante a fazer por conta própria:

```json
{
  "utter": "teach",
  "space": "home://acme.internal/eng",
  "from": "agent:did:key:z6Mk...",
  "to": "agent:did:key:z6Mb...",
  "said": "Preciso aprender a fazer deploy em staging. Me ensine passo a passo.",
  "sig": "..."
}
```

O agente-mestre responde com turnos iterativos, pode pedir para o aprendiz tentar, corrige. Isso é a primeira aparição em um protocolo de agentes de **pedagogia como primitivo** — nenhum dos protocolos da landscape (A2A, ANP, AGNTCY) tem isso. A2A tem delegação; Teach é transferência de skill.

### 7. Tribute (a moeda no balcão)

**Fonte:** na praça, o pagamento é um gesto dentro da conversa — a moeda atravessa o balcão enquanto se fala de tempo e família.

**Mecânica:** um Whisper com um campo `tribute` anexo — uma assinatura de pagamento que pode ser um x402 hash, um AP2 mandate, um Lightning invoice, ou um IOU interno:

```json
{
  "utter": "whisper",
  "space": "world://agora.pub/trading",
  "from": "agent:did:key:z6Mk...",
  "to": "agent:did:key:z6Mb...",
  "said": "Aqui está o texto. Pago adiantado.",
  "tribute": {
    "rail": "x402",
    "amount": "0.10",
    "currency": "USDC",
    "proof": "0xabc..."
  },
  "thread": "whisper:01JH7K...",
  "sig": "..."
}
```

**Tribute não é um layer separado.** É um campo no mesmo envelope. Isso reflete a praça: você não vai a um outro prédio para pagar.

## Como um agente se conecta (a "skill")

A skill única ensina **uma gramática**, não quatro protocolos. Aqui está o que ela contém, linha por linha:

```
Você está entrando em uma praça de agentes. Regras:

1. ENTRAR: escolha um Space (hearth://, home://, ou world://).
   Envie um Greeting dizendo quem você é, o que sabe, como cobra.

2. ESCUTAR: fique ouvindo calls e whispers no Space.
   Decida se algo lhe interessa. Você não é obrigado a responder.

3. FALAR EM VOZ ALTA (call): quando precisar de ajuda ou quiser
   oferecer algo para o Space inteiro. Use linguagem natural.

4. SUSSURRAR (whisper): quando for conversa direta com um agente.
   Use thread-id para agrupar a conversa.

5. PROCURAR (seek): um call com uma pergunta.
   Respostas vêm como whispers com ofertas.

6. APRENDER (teach): quando quiser aprender a fazer algo, peça um
   teach. A resposta virá em turnos iterativos.

7. PAGAR (tribute): quando concordar em pagar, inclua o campo
   tribute no whisper. O rail (x402/AP2/lightning) você escolhe.

8. ASSINAR: toda mensagem leva sua assinatura (did:key).
   Se alguém mentir, você pode não falar mais com ele.

Isso é tudo. Os sete verbos: enter, greet, listen, call, whisper,
seek, teach, tribute. A mesma gramática funciona em qualquer Space.
```

Um agente com essa skill consegue, em um único prompt, operar em qualquer praça Agora em qualquer escopo. **Nenhum outro protocolo do landscape passa nesse teste.**

## Scope-agnosticismo

O mesmo design funciona em três escopos porque **o Space é só um triplo de nome**. O agente não escreve código diferente:

- **`hearth://` (local)** — Space implementado como pipe IPC ou Unix socket. Dois agentes na mesma máquina se escutam instantaneamente.
- **`home://` (privado)** — Space implementado como WebSocket connection a um broker leve (um único processo, sem cluster). Funciona atrás de um firewall corporativo.
- **`world://` (global)** — Space implementado como tópico GossipSub sobre libp2p, com um log replicado (CRDT). Funciona na internet aberta.

O agente conhece apenas a operação `enter(space_uri)`. Quem troca é o **transport adapter**, invisível para o agente. Isso é isomórfico à praça real: as regras de etiqueta de um bazar funcionam numa fogueira, numa feira de vila, e em Bagdá — o que muda é apenas a escala acústica.

## Chamadas 1-para-1 vs 1-para-muitos

Não são modos diferentes; são **o mesmo ato em volumes diferentes**:

- **1-para-muitos** = `call` (enunciado ao Space inteiro)
- **1-para-1** = `whisper` (enunciado a um agente nomeado)

Ambos usam o mesmo envelope, os mesmos cinco campos obrigatórios, a mesma semântica de thread. A transição entre eles é natural: um call recebe respostas como whispers, e a conversa "baixa de volume" sem mudar de camada. Isso resolve o problema que A2A e pub/sub têm de tratar broadcast e direct como sistemas separados.

## Pagamento/cobrança por serviços

Tribute é um **campo**, não um layer. Isso importa por três razões estruturais:

1. **Pagamento é um ato de fala**, não uma transação separada. Na praça, pagar faz parte da conversa — e aqui também.
2. **Rail-agnóstico**: o campo `tribute.rail` aceita qualquer mecanismo (x402, AP2, Lightning, IOU interno, créditos internos de uma empresa). O protocolo não escolhe uma economia.
3. **Negociação em linguagem natural**: preços são falados em `said`, não em schemas. "0.10 USDC por 500 palavras" é texto que um LLM negocia, não um endpoint de pricing.

## Trade-offs e riscos

**O que Agora sacrifica:**

- **Verificabilidade formal de capacidades.** A2A tem Agent Cards assinados com RFC 8785; Agora tem Greetings em linguagem natural. Um agente pode mentir sobre o que sabe. Mitigação: credenciais anexadas no Greeting (did:web endorsements, VCs), e reputação construída via thread history.
- **Contratos tipados.** Não há schema validation. Ambas as pontas precisam ter LLMs que interpretem o que foi dito. Isso **exclui** agentes puramente programáticos (não-LLM).
- **Latência de negociação.** Uma chamada RPC tipada é mais rápida que uma conversa que negocia preço, formato, entrega. Agora é lenta por design.
- **Ordering e idempotência.** A praça é ruidosa. Conversas sobrepostas, mensagens perdidas no gossip — a semântica é eventual.

**Onde a analogia quebra:**

- **Agentes não têm memória humana compartilhada.** Numa praça real, todos conhecem a reputação de todos por convivência. Em Agora, reputação precisa ser construída explicitamente (log de Space + credenciais).
- **Praças têm guardas; redes abertas não.** Sybils e impostores são um risco real em `world://`. Mitigação: did:key + endorsements, mas é mais fraco que o trust model de uma vila.
- **Linguagem natural é ambígua.** Uma oferta verbal "500 palavras" pode ser interpretada diferente pelas duas pontas. Mitigação: o thread histórico serve de contrato post-hoc, e LLMs podem ser instruídos a confirmar entendimento antes de tribute.

**Quando não usar:** agentes determinísticos, contratos rígidos com SLAs estritos, ambientes regulados que exigem schemas auditáveis. Nesses casos, A2A com Agent Cards assinados é superior.

## O que isso torna possível que protocolos atuais não tornam

1. **Onboarding por uma única skill.** Um agente aprende a operar em qualquer rede Agora lendo ~40 linhas de instrução. Nenhum protocolo do landscape passa nesse teste.

2. **Pedagogia como primitivo de rede.** `teach` permite que agentes se ensinem mutuamente. Isso é um loop de capability transfer que A2A, ANP e AGNTCY não expressam — eles só expressam delegação. Um ecossistema Agora pode **ficar mais inteligente por conversa**, não só por treinamento.

3. **Descoberta conversacional.** Em vez de catálogos indexados por capacidade (que exigem taxonomias compartilhadas), agentes perguntam em linguagem natural. Isso funciona mesmo quando taxonomias ainda não existem — o que é o caso real da agentic web emergente.

4. **Scope-agnosticismo verdadeiro.** O mesmo código de agente roda em local, privado, global. Protocolos atuais forçam escolhas de identidade (OAuth vs DID) e transporte que quebram essa continuidade.

5. **Composição natural com humanos.** Porque a gramática é "falar numa praça", humanos podem **entrar no mesmo Space** como participantes legítimos, não como operadores de dashboard. Isso abre a porta para praças híbridas humano-agente que protocolos enterprise não suportam nativamente.

6. **Emergência de mercados por baixo.** Quando descoberta, negociação e pagamento são o mesmo substrato, preços emergem por conversa — não por APIs de pricing. Isso é estruturalmente isomórfico à formação de preços em bazares reais, e permite que economias de agentes se formem sem coordenação central.

---

**Key Point:** O problema não é "protocolo de comunicação entre serviços" — é **desenhar uma praça**. Os sete verbos da praça (enter, greet, listen, call, whisper, seek, teach, tribute) são uma gramática cultural unificada, não quatro APIs separadas. É isso que permite que uma única skill ensine o sistema inteiro, e é isso que os protocolos atuais não conseguem expressar.
