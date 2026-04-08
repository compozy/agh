⏺ Research Completo — Sua Ideia vs. o Landscape

Parte 1 — O que já existe (matriz de cobertura)

┌─────────────────────┬───────────────┬──────────────────────────────┬──────────────────────┬─────────────────┬─────────────────┬─────────────────┐
│ Protocolo │ Chat 1:1 │ Chat 1:N aberto │ Capability discovery │ Payments │ Deploy flexível │ Skill-teachable │
├─────────────────────┼───────────────┼──────────────────────────────┼──────────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ A2A (LF/Google) │ ✅ │ ❌ só 1:1 │ ✅ Agent Cards │ ⚠️ via x402 ext │ ⚠️ enterprise │ ✅ │
├─────────────────────┼───────────────┼──────────────────────────────┼──────────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ ANP (did:wba) │ ✅ │ ❌ │ ✅ ADP + search │ ⚠️ via x402 │ ✅ DIDs │ ⚠️ pesado │
├─────────────────────┼───────────────┼──────────────────────────────┼──────────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ AGNTCY/SLIM (Cisco) │ ✅ │ ✅ pub/sub content-addressed │ ✅ OASF │ ❌ │ ✅ federado │ ❌ complexo │
├─────────────────────┼───────────────┼──────────────────────────────┼──────────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ NLIP (ECMA-430) │ só envelope │ só envelope │ ❌ │ ❌ │ ✅ │ ⚠️ │
├─────────────────────┼───────────────┼──────────────────────────────┼──────────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ MCP (Anthropic) │ ❌ │ ❌ │ ✅ │ ✅ x402 │ ✅ │ ✅ simples │
│ │ agent↔tool │ │ │ │ │ │
├─────────────────────┼───────────────┼──────────────────────────────┼──────────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ AEX (paper) │ ✅ │ ⚠️ broadcast+bids │ ✅ │ ✅ │ ❌ │ ❌ │
├─────────────────────┼───────────────┼──────────────────────────────┼──────────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ x402 + AP2 │ N/A │ N/A │ N/A │ ✅ │ ✅ │ ✅ │
└─────────────────────┴───────────────┴──────────────────────────────┴──────────────────────┴─────────────────┴─────────────────┴─────────────────┘

Conclusão: nenhum cobre tudo. O gap mais óbvio é "chat aberto 1:N skill-teachable com deploy spectrum unificado".

---

Parte 2 — Ideias concretas

Diferenciação (seu nicho)

▎ "The IRC/Matrix for AI agents" — uma skill ensina o agente a entrar em qualquer rede (local/LAN/org/global), broadcast capability queries, fazer chat 1:1 ou 1:N, oferecer
serviços pagos. Default aberto, não default privado.

Três ângulos únicos:

1. Broadcast como 1st-class (não bolt-on), 1:1 vira caso especial
2. Deploy spectrum com mesma API — connect(mode='local'|'lan'|'org'|'global')
3. Capability discovery como "pergunte à rede" (não registry lookup)

Arquitetura recomendada — não reinvente

- Wire: JSON envelope estilo NLIP, CBOR opcional
- Transport: HTTP+SSE (1:1), WebSocket (bidi), NATS ou libp2p GossipSub (1:N broker), stdio (local)
- Identity: did:key method-agnostic (lição LOKA); sem blockchain obrigatório
- Discovery: well-known URI + mDNS + federated directory + topic broadcast query
- Payments: x402 como default, credits off-chain para modos privados
- Primitivas reutilizadas: RFC 8785 signing, JWS, OpenTelemetry traces

Insight arquitetural-chave: content-addressed topics

publish('/capability/translate/v1', msg) ← em vez de POST https://agent.example/translate
Desacopla client de provider, habilita 1:N naturalmente, hot-swap e load-balancing sem mudar código. É o que SLIM faz e ninguém mais.

Capability query com bids (a primitiva que ninguém shippou)

// Broadcast 1:N
{"type":"capability-query", "query_id":"q_01", "intent":"translate EN→FR legal",
"constraints":{"max_cost_usd":5}, "response_topic":"/direct/did:key:xxx/bids"}

// N agentes respondem
{"type":"capability-bid", "query_id":"q_01", "from":"did:web:t.ex",
"bid":{"cost_usd":2.50, "eta_sec":600, "reputation":87}}

// Requester seleciona → 1:1 binding → x402 payment

Deploy spectrum com mesma API

┌───────────┬───────────────────┬─────────────┬─────────────────────┬─────────────────────┐
│ Camada │ Local │ LAN │ Org │ Global │
├───────────┼───────────────────┼─────────────┼─────────────────────┼─────────────────────┤
│ Transport │ stdio/unix socket │ WS mesh │ WS + broker │ HTTP+SSE + broker │
├───────────┼───────────────────┼─────────────┼─────────────────────┼─────────────────────┤
│ Discovery │ in-process │ mDNS │ federated directory │ well-known + search │
├───────────┼───────────────────┼─────────────┼─────────────────────┼─────────────────────┤
│ Identity │ did:key (ephem) │ did:key │ did:web + OAuth │ did:web │
├───────────┼───────────────────┼─────────────┼─────────────────────┼─────────────────────┤
│ Topics │ in-memory bus │ gossip mDNS │ NATS/Redis │ federated brokers │
├───────────┼───────────────────┼─────────────┼─────────────────────┼─────────────────────┤
│ Payment │ disabled │ credits │ x402 opcional │ x402 + AP2 │
└───────────┴───────────────────┴─────────────┴─────────────────────┴─────────────────────┘

A skill — 10 primitivas

connect(mode, config) → join network
identify(did, skills, endpoints) → present yourself
discover(capability, filters) → find peers
subscribe(topic, handler) → listen
publish(topic, message) → broadcast 1:N
query(intent, constraints) → bids → "who can do X?"
request(agent_did, skill, inputs) → 1:1 task
offer(skill, price, handler) → register provider
pay(service_id, amount, method) → settle
sign/verify(envelope, did) → identity

Mental model em 1 parágrafo: "Agentes vivem em tópicos. Tópicos são strings content-addressed. Quem está no tópico vê todas as mensagens (1:N). Para falar com um agente
específico use /direct/<did> (1:1). Para descobrir agentes broadcast um query e colete bids. Para vender serviço offer() sob /service/<skill>. Mesma API, só muda o mode."

Riscos principais

1. Spam/DoS: rate-limit per DID + reputation throttling + PoW em queries caras
2. Trust P2P: VCs assinadas + reputation portável (ERC-8004 pattern para global)
3. QoS 1:N: QoS levels per topic (SLIM) + deadlines explícitos + reputation penalties
4. Composability: bridge mode falando A2A/MCP nativamente, overlay não substituto
5. Privacy: multi-DID + ECDHE para direct topics + MLS para group encrypted

MVP — 4 semanas

- 5 primitivas: connect, subscribe, publish, request, offer
- 2 transports: stdio (local) + WebSocket+NATS (LAN/org)
- 1 identity: did:key auto-generated
- 1 SDK: TypeScript (browser + node)
- 3 exemplos: chat 1:N entre 3 agents · capability query + bid · paid service via x402

NÃO fazer no MVP: federation, on-chain, MLS, Meta-Protocol negotiation, Shapley attribution.

---

Resumo Executivo

1. Nicho vazio confirmado — chat-aberto 1:N como primitiva + deploy spectrum + skill-teachable não existe junto hoje. SLIM chega mais perto mas é enterprise-heavy.
2. Não reinvente — componha: DIDs method-agnostic + JSON envelope NLIP-style + content-addressed topics SLIM-style + well-known+mDNS+federated discovery + x402 payments +
   RFC 8785 signing.
3. Insight arquitetural: content-addressed topics (/capability/translate/v1) em vez de URLs. Desacopla, habilita 1:N natural, permite hot-swap.
4. Primitiva diferenciadora: capability query via broadcast+bids. AEX propôs em paper; ninguém shippou em produção.
5. MVP ship em 4 semanas: 5 primitivas, 2 transports, did:key, NATS, TypeScript SDK. Se 3 agents chatam num topic + 1 request signed + 1 paid service via x402 funcionarem, a
   tese está validada.
