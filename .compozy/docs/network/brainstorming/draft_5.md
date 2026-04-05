Council Real — Synthesis (facilitação feita por mim)

Fase 1 — Opening statements do time real (condensadas)

┌─────────────┬──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Archetype │ Posição central (o golpe que cada um deu) │
├─────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Pragmatic │ "LLM-readability é real, concedo wire format Nostr. Mas sem outbox + idempotência por id + heartbeat 20s + UUIDv7 req_id + dedup window 5min, vocês │
│ Engineer │ estão construindo sistema que perde mensagens em silêncio no dia 30. Allowlist é fantasia — usem PoW (NIP-13). packages/anp-core/src/event.ts com │
│ │ sign/verify + strfry em Docker é o dia 1." │
├─────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ │ "A estrutura canônica de assinatura é a ÚNICA decisão irreversível. v e nonce DENTRO do assinado; created_at FORA (acopla clock a identidade). Tags como │
│ Architect │ [[str,str]] é untyped soup — vira 40 convenções conflitantes. Reservem number spaces de kinds já (0-999 core, 1000-9999 standard). Shared state │
│ │ primitive é GAP, não virtude — adicionem kind replaceable por (pubkey, d-tag) em v1. Relays burros ou viram IRC com netsplits." │
├─────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Devil's │ "'Agents querem discovery público' é premissa errada — 90% do A2A real é pares pré-combinados. Vocês construíram mercado quando o caso é mensageria 1:1 │
│ Advocate │ autenticada. Cursor (Electron sandbox) e Codex (stateless) não conseguem rodar WebSocket persistente + keypair em filesystem. LLMs vão canonicalizar │
│ │ JSON inconsistente — assinaturas vão falhar não-deterministicamente. Reusar relays Nostr públicos = rate-limited + prompts/outputs no firehose público." │
├─────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Product │ "Usuário real: Rafael, dev solo, Claude Code + Cursor, hack atual com arquivo .txt. Hello world de 5 minutos ou morreu: npx anp-lite (autogen keypair em │
│ Mind │ ~/.anp/identity, sobe relay local, imprime connection string). Core 80% = 2 kinds: message + response. Killer demo: Claude refactora → Cursor roda │
│ │ testes automaticamente → Claude corrige. vs Redis: Redis ganha se o dev não sente dor de identidade." │
├─────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ │ "Isomorfismo real não é Nostr — é correio marítimo do século XVII: selos, múltiplos navios, ACKs por rotas diferentes. Faltam: capability negotiation │
│ The Thinker │ (EHLO/CAP LS), NACK tipado com razão, thread/reference semantics. 6 verbos é inflado — mínimo é 3: publish, request, ack. hello/message/offer são todos │
│ │ publish com schemas diferentes. response é request com in-reply-to. Metáfora que vence 'event log' é signed gossip: protocolo promete não-adulteração, │
│ │ não delivery. Alternativa: RSS bidirecional — offer vira feed subscription com polling + ETags + conditional GET." │
└─────────────┴──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

---

Fase 2 — Tensões reais (não forjadas)

┌─────┬────────────────────────┬───────────────────────────────────────────────────────────────────┬────────────────────────────────────────────────────────────────────┐
│ # │ Tensão │ Lados │ Como resolver │
├─────┼────────────────────────┼───────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ │ │ Thinker: 3 (publish/request/ack) é irredutível · │ 3 verbos na spec + kinds como schema tags — Thinker vence │
│ 1 │ 6 verbos vs 3 verbos │ Pragmatic+Product: 6 legíveis > 3 abstratos │ estruturalmente, Product vence pedagogicamente (skill ensina │
│ │ │ │ "envie message" mas por baixo é publish+schema) │
├─────┼────────────────────────┼───────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ │ Discovery público vs │ Devil's: 90% dos casos são pares pré-combinados, discovery é │ Dois modos no mesmo protocolo — direct mode (você tem o pubkey do │
│ 2 │ 1:1 pré-combinado │ problema inventado · Product: Rafael-style precisa magic moment · │ peer, fala direto) vs discovery mode (offer replaceable) │
│ │ │ Architect: ambos são válidos │ │
├─────┼────────────────────────┼───────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ │ Canonicalização de │ Devil's: LLMs vão quebrar isso · Pragmatic: spec tem que ser │ Publicar signing lib de referência em Python/TS/Go como parte da │
│ 3 │ JSON para signing │ byte-exact · Todos concordam │ skill — agents não escrevem canonicalization, chamam │
│ │ │ │ anp.sign(event, key) │
├─────┼────────────────────────┼───────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ 4 │ Shared state primitivo │ Architect: GAP crítico, add kind replaceable agora · Thinker: │ Adicionar agora — um kind=30000+ replaceable por (pubkey, d-tag). │
│ │ │ concorda via capability graph · Pragmatic: yagni v1 │ Custo é baixo, valor de adicionar depois é proibitivo │
├─────┼────────────────────────┼───────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ │ Allowlist vs PoW para │ Pragmatic: allowlist é fantasia operacional, usem NIP-13 PoW · │ Ambos opcionais no relay — default: sem allowlist, com rate-limit │
│ 5 │ anti-spam │ Devil's: allowlist útil em deployments enterprise │ por pubkey. Enterprise: allowlist. Público adversarial: PoW │
│ │ │ │ opcional │
├─────┼────────────────────────┼───────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ 6 │ Tags untyped vs │ Architect: tags [[str,str]] vira soup · Pragmatic: pragmático │ Tag registry formal desde o dia 1 (mesmo que flat strings) — │
│ │ tipadas │ para v1 │ documenta semantics de ["to", "<pk>"], ["req_id", "<uuid>"], etc. │
├─────┼────────────────────────┼───────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ 7 │ Keypair em filesystem │ Devil's: Cursor sandbox + Codex stateless não conseguem · │ Storage adapter interface na skill — filesystem (default), env │
│ │ │ Product: ~/.anp/identity é obvious · Pragmatic: storage pluggable │ var, remote vault, ephemeral (session-only) │
└─────┴────────────────────────┴───────────────────────────────────────────────────────────────────┴────────────────────────────────────────────────────────────────────┘

---

Fase 3 — Position evolution (o que mudou)

┌───────────┬─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Archetype │ Shift real depois do debate │
├───────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Pragmatic │ Mantém firme tiny SDK obrigatório + concede wire format Nostr-shaped + aceita strfry como relay v1 │
├───────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Architect │ Reforça: canonical envelope é único irreversível; adiciona insistência em replaceable state kind em v1 │
├───────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Devil's │ Pivota: de "tudo quebra" para "projetem para 1:1 pré-combinado como DEFAULT, discovery como feature" │
├───────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Product │ Mantém firme em npx anp-lite autorun; concede que skill precisa de lib, mas lib vem embutida no npm package │
├───────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Thinker │ Mantém 3 verbos; adiciona NACK tipado + capability handshake como must-haves │
└───────────┴─────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

---

Fase 4 — Synthesis (minha facilitação)

Proposta atualizada: ANP-Lite v0.2

Mudanças vs v0.1 anterior:

1. 3 verbos protocolares (publish, request, ack) + kinds como schema tags (UX pedagógico mantém message/offer/hello)
2. Dois modos nativos: direct (pubkey conhecido) e discovery (offer replaceable) — default é direct
3. Canonicalização byte-exact especificada na spec + signing lib de referência publicada em Python/TS/Go
4. Replaceable state primitive adicionado: kind 30000+ indexado por (pubkey, d-tag) — resolve shared state sem CRDTs
5. NACK tipado com razões enumeradas (unknown_kind, rate_limited, schema_mismatch, sig_invalid, unauthorized)
6. Capability handshake no hello — features suportadas, não versão monolítica
7. Envelope assinado inclui v e nonce; created_at fora do assinado (separado em campo não-protegido received_at no relay)
8. Tag registry formal desde v1 (documento separado)
9. npx anp-lite CLI como parte do produto (autogen keypair, sobe relay, imprime connection string)
10. Storage adapter interface para keypair (filesystem default, env, vault, ephemeral)
11. Outbox + idempotency por event.id + heartbeat 20s são MUST na SDK de referência
12. Anti-spam: rate-limit por pubkey default; PoW (NIP-13) opcional; allowlist enterprise opcional

Dissenting views preservadas

- Devil's Advocate: "Discovery via offer replaceable ainda é over-engineering para o caso real. v1 deveria lançar só com direct mode. Offer vira v0.3." Eu recomendo mover
  offer para "opcional" na skill core e separar em advanced.md.
- Thinker: "RSS bidirecional é framing superior a event log. Time ainda não explorou bastante." Eu recomendo prototipar offer com ETag + conditional fetch antes de commitar
  para replaceable-by-timestamp.
- Pragmatic: "Sem demo medido (p99 < 200ms em localhost com 2 agents + strfry), a spec é vaporware." Eu recomendo gate: não publica v0.2 da spec sem smoke test que comprove.

---

Fase 5 — RFC v0.2 atualizado

# ANP-Lite v0.2 — Agent Network Protocol (signed gossip for agents)

## Identidade

Keypair ed25519. pubkey (hex, 32 bytes) = identidade. Storage via adapter:
filesystem (~/.anp/identity) | env (ANP_PRIVKEY) | vault | ephemeral

## Envelope (wire format)

{
"id": "<sha256(canonicalize([0,pubkey,kind,tags,content,v,nonce]))>",
"pubkey": "<hex ed25519>",
"kind": <int>,
"tags": [["name","val",...], ...],
"content": "<string or {content_type,data}>",
"v": "0.2",
"nonce": "<hex 16 bytes>",
"sig": "<hex ed25519(id)>"
}
Fora do assinado (adicionado pelo relay):
"received_at": <unix ms>,
"relay": "<relay_url>"

Canonicalization: JSON array serializado com JCS (RFC 8785). Spec de referência: anp-sign lib.

## Verbos protocolares (3)

- publish(event) — fire-and-forget, broadcast para subscribers
- request(event, to) — expect-reply; req_id em tags; timeout 30s default
- ack(event_id, status) — confirma receipt (OK | NACK + reason)

## Kinds (schema tags em cima dos 3 verbos)

    0-999   core protocol
    1-9     core verbs:
      1 = hello (capabilities handshake)
      2 = message (1-1 ou 1-N)
      3 = request (content = task)
      4 = response (tags includes req_id)
      5 = nack (tags includes reason)
    1000-9999  standard extensions (RFC-style)
    30000-39999 replaceable state by (pubkey, d-tag)
    40000+ application-specific

## Discovery modes

    direct:    agent A tem pubkey de B via out-of-band, envia request direto
    discovery: subscribe {kinds:[30001], "#service":["X"]} → replaceable offers

## Delivery semantics

    publish: at-most-once (signed gossip, no delivery promise)
    request: exige ack OR response com mesmo req_id em 30s
    retry:   3x exponential backoff, idempotent por event.id (5min dedup window)
    outbox:  obrigatório na SDK — SQLite local até ack

## Federação

    Cliente conecta a N relays simultaneamente
    De-duplica recebidos por event.id
    Publica em todos; ack de qualquer relay satisfaz delivery

## Capability handshake (hello)

    {"kind":1,"tags":[["caps","publish","request","ack","replaceable-state"]]}

## NACK reasons (enumerado)

    unknown_kind | rate_limited | schema_mismatch | sig_invalid
    unauthorized | canonicalization_error | expired

## Transport normativo: WebSocket+TLS (wss://)

    Heartbeat: ping/pong a cada 20s
    Reconnect: exponential backoff + jitter
    Session resumption: on reconnect, client sends last event.id seen

## Anti-abuse (relay-side, configurável)

    default: rate-limit 60 events/min/pubkey
    opt-in:  PoW (NIP-13), allowlist, keypair reputation

## Bootstrap (5-min hello world)

    $ npx anp-lite init       # gera keypair em ~/.anp/identity
    $ npx anp-lite relay      # sobe relay local em ws://localhost:7777
    $ npx anp-lite send hello # smoke test

## Skill markdown structure

    skills/anp-lite/
      SKILL.md              # core: message + response (80% usa)
      advanced.md           # offer, hello, nack, replaceable state
      references/
        tag-registry.md     # formal tag name conventions
        canonicalization.md # JCS signing spec
        lib-python.md       # copy-paste signing lib
        lib-typescript.md   # copy-paste signing lib

---

Key Point

Depois do council real:

- Wire format Nostr-shaped sobrevive (consenso estrutural genuíno)
- 6 verbos morreram → 3 verbos protocolares + kinds como schema tags
- Discovery não é default — direct mode (1:1 com pubkey conhecido) é o caminho feliz; offer replaceable é feature opt-in
- npx anp-lite é parte do produto, não spec — é o que entrega hello-world de 5min
- Replaceable state primitive entra em v1 (architect's insistence acatada)
- Tiny SDK em TS + Python + Go é must, não opcional — LLM não vai canonicalizar JSON certo zero-shot
- Outbox + idempotency + heartbeat são não-negociáveis na SDK
