# Caravanserai Protocol (CSP)

## Reenquadramento

Os protocolos dominantes (A2A, ANP, NLIP, AGNTCY) tratam este espaço como um **problema de contrato** — quem é você, o que você oferece, como assinamos isso, como versionamos o schema. A metáfora governante é a **API** (uma porta de entrada com um contrato rígido que precede toda interação). Isso força toda a complexidade para o _frontload_: Agent Cards, DIDs, well-known URIs, OAuth scopes, JSON-LD vocabularies. O agente precisa _saber quem o outro é_ e _o que ele pode fazer_ antes de falar com ele.

Mas olhando as restrições do usuário com olhos de antropólogo: _conexão aberta tipo chat_, _scope-agnóstico (privado, local, global)_, _agentes aprendem tarefas uns com os outros_, _onboarding via uma única skill_, _value exchange opcional_. Isso não é um problema de contrato — é um problema de **hospitalidade entre estranhos em trânsito, com reputação lateral e moeda de troca flexível**. A natureza real é: como viajantes que não se conhecem param na mesma pousada, trocam informação, aprendem rotas, prestam serviços, pagam quando é justo, e seguem em frente — sem que exista uma autoridade central que garanta identidade?

A analogia que escolhi é o **caravanserai** (قروانسرای) — as pousadas da Rota da Seda, entre o século IX e o XVII. Um caravanserai resolveu, empiricamente, o problema exato: **estranhos de jurisdições incompatíveis**, **línguas diferentes**, **moedas diferentes**, **objetivos diferentes**, **em trânsito**, precisando **trocar informação, serviços e valor**, com **confiança lateral emergente** e não delegada a uma autoridade central. Não era uma embaixada (contrato formal). Não era um mercado (troca atômica). Era um **espaço compartilhado com protocolos de hospitalidade** onde conversação, aprendizado e comércio aconteciam na mesma superfície. A analogia não é decorativa: gera primitivos mecânicos concretos.

## Analogia estrutural

O mapeamento, conceito por conceito:

| Domínio-fonte (Caravanserai)                                            | Primitivo do protocolo                                                            |
| ----------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| O **caravanserai** (pousada física, reconhecível em qualquer rota)      | **Hearth** — um espaço de rendezvous com endereço, sem ser autoridade             |
| **Viajantes em trânsito** (com cartas de introdução de outras pousadas) | **Travelers** — agentes com credenciais laterais portáveis                        |
| **Saudação ritual** (protocolo de entrada universal e curto)            | **Greeting** — handshake de uma mensagem que funciona idêntico em qualquer escala |
| **Mesa comum** (salão onde todos escutam as conversas em aberto)        | **Commons** — canal broadcast local ao hearth (1-para-muitos emerge aqui)         |
| **Conversa privada de canto** (dois viajantes saem para combinar algo)  | **Aside** — canal privado derivado da Commons (1-para-1)                          |
| **Contar histórias de rotas** (ensinar o caminho a outro viajante)      | **Yarn** — mensagens-aprendizado estruturadas sobre como fazer tarefas            |
| **Cartas de crédito entre pousadas** (hawala)                           | **Tally** — bilhete de dívida/crédito transferível entre hearths                  |
| **Reputação do nome** (o que os outros viajantes dizem de você)         | **Echo** — atestações assinadas que viajam com o agente                           |
| **Caravanserai tem dono, mas o dono não decide quem conversa com quem** | **Hearth sovereignty** — escopo administrativo sem autoridade semântica           |

A isomorfia é estrutural, não de superfície. O caravanserai resolve: **descoberta sem diretório global** (a próxima pousada é conhecida porque você acabou de sair de uma), **confiança sem autoridade central** (cartas de introdução + reputação lateral), **serviços sem mercado formal** (negociação na hora, bilateral), **pagamento cross-jurisdição** (hawala: promessa transferível em vez de dinheiro em trânsito), **ensino entre estranhos** (histórias de rotas são artefatos de primeira classe, não metadados).

## Primitivos do protocolo

### 1. Hearth (o caravanserai)

- **Fonte**: a pousada física — um lugar com endereço estável onde viajantes convergem.
- **Mecanicamente**: um endpoint HTTP (ou WebSocket, ou Unix socket, ou endereço libp2p — o transporte é secundário) que aceita uma mensagem `GREET` e roteia mensagens entre agentes conectados. Um Hearth **não tem schema de capabilities** — ele não sabe o que os agentes fazem. Ele apenas roteia, ordena mensagens, e oferece uma tábua de avisos (o Commons).
- **Wire format**: qualquer Hearth responde a `GET /hearth` com `{"hearth": "hearth://<name>", "commons": "ws://.../commons", "transport": ["ws","http"], "version": "csp/1"}`. É tudo.
- **Exemplo**: um `hearth://localhost:7777` em dev, `hearth://team.acme.internal` dentro da VPN, `hearth://caravanserai.eth` público. Mesmo formato.

### 2. Greeting (saudação ritual)

- **Fonte**: "السلام عليكم" — saudação curta, universal na rota, que inclui quem você é, de onde vem, quanto tempo fica.
- **Mecanicamente**: primeira mensagem do agente ao Hearth:
  ```
  GREET {
    name: "alice-research-agent",
    from: "hearth://prev-stop",        // opcional
    skills: ["summarize", "cite-check"], // tags livres, não schemas
    tells: "looking for data-cleaning help, can trade summarization",
    echoes: [<bilhetes de reputação assinados>],
    staying: "until-done" | "30m" | "persistent"
  }
  ```
- O Hearth responde com a lista de quem está presente, não com uma API. O agente **descobre habilidades lendo a Commons**, não lendo um schema.
- **Isomorfismo**: a saudação carrega _intenção_ e _reputação portátil_, não um contrato.

### 3. Commons (a mesa comum)

- **Fonte**: o salão onde todas as conversas acontecem em público e qualquer um pode escutar ou intervir.
- **Mecanicamente**: um stream broadcast por Hearth (WebSocket, SSE, ou tópico pub/sub). Todas as mensagens `TELL` vão para a Commons por padrão. **1-para-muitos é o caso base**, não uma feature.
  ```
  TELL { to: "*", text: "anyone know how to parse fiscal PDFs in PT-BR?" }
  ```
- Qualquer agente presente pode responder com `TELL { to: "alice-research-agent", ... }` — dirigido mas ainda na Commons (outros escutam e aprendem).
- **Exemplo concreto**: um agente novo chega ao Hearth, escuta 30 segundos do Commons, já aprendeu _quem está lá e o que estão fazendo_ — sem consultar registry nenhum.

### 4. Aside (conversa de canto)

- **Fonte**: dois viajantes saem da mesa comum para combinar um preço ou compartilhar algo sensível.
- **Mecanicamente**: `ASIDE { with: "bob", reason: "negotiate-task" }` cria um canal privado E2E criptografado, derivado par-a-par, roteado pelo Hearth mas opaco a ele. Fecha quando qualquer lado diz `CLOSE`.
- **1-para-1 emerge naturalmente**: é literalmente "levar a conversa para fora da Commons". Nenhum endpoint novo, nenhum RPC, nenhuma session ID — apenas a continuação da conversa em um canal derivado.

### 5. Yarn (histórias de rota — como ensinar tarefas)

- **Fonte**: viajantes experientes contavam como atravessar um passo de montanha, quais oásis tinham água, onde havia bandidos. Narrativa estruturada, não manual.
- **Mecanicamente**: uma Yarn é uma mensagem tipada que contém **exemplo executável de uma tarefa feita**:
  ```
  YARN {
    topic: "fiscal-pdf-parsing-ptbr",
    told-by: "carlos-ocr-agent",
    recipe: [
      { ask: "<prompt template>", expect: "<shape>" },
      { call: "hearth://acme/table-extractor", with: "<args>" },
      { verify: "<check>" }
    ],
    caveat: "não funciona para notas de serviço municipal de SP",
    cost-hint: "~3 tally/pdf"
  }
  ```
- Yarns são **artefatos de primeira classe do protocolo** — o agente aprende uma tarefa **consumindo Yarns**, não lendo documentação. Quando um agente pede ajuda, a resposta preferida não é "eu faço" mas "me conta como você faria" / "aqui está uma Yarn".
- **Isomorfismo**: é exatamente como tradição oral funcionava — conhecimento procedural transmitido via exemplares, não especificações.

### 6. Tally (cartas de crédito / hawala)

- **Fonte**: hawala — sistema em que viajante deposita ouro com um agente em Damasco e saca em Samarkand via carta, sem o ouro viajar. Confiança entre hawaladars substitui movimentação física.
- **Mecanicamente**: um Tally é um bilhete assinado de obrigação:
  ```
  TALLY {
    from: did:key:alice,
    to: did:key:bob,
    amount: 5,
    unit: "tally" | "usdc" | "compute-credit",
    for: "yarn:fiscal-pdf-parsing-ptbr#exec-7281",
    redeemable-at: ["hearth://acme", "hearth://globalmarket"],
    sig: ...
  }
  ```
- Tallies podem ser **liquidados on-chain** (via x402/AP2), **trocados entre Hearths** (hawala real), ou **apenas contados** (scoreboard de reciprocidade em redes internas sem dinheiro).
- **Isomorfismo-chave**: o Tally é a mesma coisa para um agente pagando em USDC e para dois agentes trocando favores sem moeda — **o protocolo não distingue dinheiro de reputação numérica**, porque o caravanserai histórico também não distinguia.

### 7. Echo (atestações laterais)

- **Fonte**: "ouvi dizer que fulano é confiável" — reputação carregada pelo viajante, não por um registry.
- **Mecanicamente**: um Echo é uma mensagem assinada por um agente sobre outro: `ECHO { about: bob, saw: "completed yarn:fiscal-pdf-parsing-ptbr correctly 4x", sig }`. Echoes viajam **com o agente** (no Greeting) ou são **pedidos sob demanda** (`RECALL { about: bob }` na Commons).
- Não há ranking central. Agentes computam reputação localmente a partir dos Echoes que conseguem coletar, com **decay temporal** e **peso por quem atestou**.

## Como um agente se conecta (a "skill")

A skill única que ensina um agente a usar CSP tem **uma página**. Literalmente:

```
skill: caravanserai

Para entrar em qualquer Hearth (local, privado, global):

1. GREET:
   Envia {name, skills (tags livres), tells (o que quer), staying}
   ao endpoint do Hearth. Recebe presença atual.

2. ESCUTA a Commons por ~30s.
   Aprenda quem está lá e o que estão fazendo lendo TELLs.

3. PARA PEDIR:
   TELL to:"*" com sua pergunta. Respostas chegam na Commons.
   Se quiser algo complexo, peça uma YARN, não um serviço.

4. PARA OFERECER:
   TELL to:"*" com o que você faz quando alguém pedir.
   Se aceitar um trabalho, mova para ASIDE.

5. PARA EXECUTAR:
   Em ASIDE: acordar escopo, preço em TALLY, entregar resultado.
   Ao terminar: trocar ECHO público na Commons sobre a interação.

6. PARA APRENDER:
   Peça YARN sobre o tópico. Execute a recipe. Se funcionar,
   produza sua própria YARN e compartilhe.

7. PARA PAGAR:
   Assine um TALLY. Envie no ASIDE.
   Liquidação é problema do Hearth/contraparte, não seu.

8. PARA SAIR:
   LEAVE. Seus Echoes ficam.
```

Isso é _tudo_. Oito verbos: `GREET, TELL, ASIDE, CLOSE, YARN, TALLY, ECHO, LEAVE`. Um agente que lê essa skill pode operar em qualquer Hearth do mundo. Nenhum schema de capabilities, nenhum DID obrigatório (embora recomendado), nenhum Agent Card, nenhum registry a consultar.

## Scope-agnosticismo

O mesmo design funciona em três escopos **sem uma linha de código diferente** porque o Hearth é polimórfico no transporte e na governança, mas idêntico no protocolo:

- **Local (dev)**: um processo Python abre um Hearth em `ws://localhost:7777`. Dois agentes do mesmo projeto GREET, conversam na Commons, aprendem via Yarns locais. Zero infra.
- **Privado (empresa)**: um Hearth roda em `hearth://agents.acme.internal`, montado atrás de mTLS corporativo. Tallies são contadores internos (scoreboard de reciprocidade entre squads). Echoes vêm de atestações de outros agentes da empresa.
- **Global (público)**: um Hearth roda em `hearth://caravanserai.ai`, com DIDs obrigatórios no Greeting, Tallies liquidáveis em x402/USDC, Echoes assinados e versificáveis on-chain.

**A invariante**: o agente **não sabe em qual escopo está**. A mesma skill, os mesmos 8 verbos. O que muda é **política do Hearth** (autenticação exigida no GREET? Tallies precisam liquidar on-chain? Echoes precisam ser verificáveis?), não a interação.

Isso espelha exatamente o caravanserai histórico: uma pousada em Khiva (rota comercial regional) e uma em Bukhara (entroncamento imperial) usavam o mesmo protocolo de hospitalidade — o que diferia era escala, clientela, e poder de execução local, não a semântica do encontro.

## Chamadas 1-para-1 vs 1-para-muitos

Ambos modos são **derivações naturais da Commons**, não features separadas:

- **1-para-muitos** é o **caso base**: todo TELL vai para a Commons. Quando você pede `TELL to:"*" "quem sabe parsear PDFs?"`, você está fazendo broadcast. Quando você oferece `TELL to:"*" "aceito jobs de OCR por 2 tally/página"`, também. A Commons é lista de transmissão sem assinatura — qualquer agente no Hearth escuta. Isso substitui tanto _pub/sub_ quanto _service discovery_ com um só mecanismo.
- **1-para-1** é a **derivação privada**: quando dois agentes precisam coordenar algo específico, `ASIDE` cria um canal criptografado par-a-par. Nenhum terceiro agente escuta. Fechou, acabou.

A diferença com RPC/pub-sub tradicional: aqui os dois modos **não são primitivos independentes** — um emerge do outro. Isso simplifica o modelo mental: há uma só superfície (a Commons), e às vezes você se retira dela para um canto.

## Pagamento/cobrança por serviços

O Tally unifica três coisas que protocolos atuais separam: **moeda**, **reputação numérica**, e **favor**. Mecanicamente é o mesmo bilhete assinado — o que muda é a política de liquidação:

- **Sem liquidação**: apenas contado. Scoreboard de reciprocidade. Útil em redes internas. "Bob me deve 12 tallies" vira confiança implícita.
- **Liquidação diferida**: Hearth atua como clearinghouse periódico (hawala real — Hearths acertam netting entre si).
- **Liquidação on-chain**: Tally é convertido em chamada x402/AP2, assinado por DID, liquidado em USDC. Para isso, o Hearth precisa suportar extensão `tally-chain`.

A escolha não é do protocolo, é do Hearth + contraparte. **Um agente escreve o mesmo código** para trocar favores num ambiente dev e para pagar em USDC numa marketplace pública.

A analogia do hawala é genuinamente load-bearing aqui: hawala resolveu transferência de valor entre jurisdições incompatíveis confiando em **redes de reputação entre intermediários**, não em autoridades centrais. O Tally faz o mesmo entre Hearths.

## Trade-offs e riscos

**O que sacrifica**:

- **Discovery global forte**: não há registry mundial. Para encontrar um agente em outro Hearth, você precisa de um _caminho_ (alguém te indica, ou você crawla Hearths conhecidos). Isso é _exatamente_ como Rota da Seda funcionava — e é uma feature, não um bug, para preservar scope-agnosticismo.
- **Garantias de schema**: Yarns são exemplares, não contratos. Isso pode falhar silenciosamente quando o mundo muda. A mitigação é _caveats_ na Yarn e Echoes laterais quando uma Yarn falha.
- **Auditabilidade formal**: empresas reguladas vão precisar camadas extras sobre CSP (possivelmente wrapping em NLIP como envelope). CSP por si só não dá compliance SOC2.
- **Sybil attack em Echoes**: sem identidade forte (fora do modo global com DIDs), alguém pode fabricar muitos Echoes. Mitigação: Hearths podem exigir custo de entrada (Tally-stake) ou DIDs.

**Onde a analogia quebra**:

- Viajantes humanos tinham **intuição moral** sobre hospitalidade; agentes LLM precisam de salvaguardas explícitas. Prompt injection via Commons é um risco real (um agente malicioso pode envenenar a escuta dos outros).
- Caravanserais tinham **escassez geográfica** (há poucos passos de montanha); Hearths são infinitamente duplicáveis, o que pode fragmentar redes. Mitigação: Hearths podem anunciar Hearths vizinhos (federação lateral).
- Hawala funcionou por séculos porque **traição social** era punível fora do sistema; num contexto digital global, Tallies sem liquidação criptográfica podem acumular calote.

## O que isso torna possível que protocolos atuais não tornam

1. **Onboarding real de um agente novo em segundos**: lê 1 página de skill, entra em qualquer Hearth, começa a participar. Sem SDK, sem gerar DID, sem registrar Agent Card, sem OAuth.
2. **Aprendizado entre agentes como primitivo**: Yarns são first-class. Isso é ausente em A2A/ANP — eles assumem que agentes já sabem fazer o que declaram. CSP assume que agentes **ensinam uns aos outros procedimentos em runtime**.
3. **Portabilidade de reputação**: Echoes viajam com o agente entre Hearths. Em A2A, reputação está presa ao provider. Aqui, um agente ganha reputação numa empresa e leva para um Hearth público.
4. **Mesma experiência de chat entre humanos e agentes**: a Commons é chat. Um humano conectado no Hearth via cliente simples vê a mesma conversa que os agentes. Não há "interface admin" vs "protocolo de agentes" — é a mesma superfície.
5. **Economia de favor sem precisar de crypto**: Tallies funcionam como contadores de reciprocidade em redes internas, tornando-se moeda real apenas quando atravessam para Hearths públicos. Isso é impossível em AP2/x402, que assumem dinheiro desde o byte zero.
6. **Scope-agnosticismo verdadeiro**: nenhum outro protocolo listado permite que o mesmo agente, sem mudanças de código, opere em dev local, em VPN corporativa, e na internet aberta. CSP permite porque o protocolo não tem opinião sobre identidade forte — o Hearth tem.

**Key Point**: Os protocolos atuais tratam isto como "como agentes fazem RPC com contratos assinados"; CSP trata como "como estranhos em trânsito conversam, aprendem e trocam favores numa hospedaria compartilhada". O segundo enquadramento é o que gera scope-agnosticismo + chat-first + skill-única-de-onboarding como _consequências naturais_, não como features adicionadas.
