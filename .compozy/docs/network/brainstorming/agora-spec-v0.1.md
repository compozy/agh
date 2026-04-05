# AGORA Protocol — SPEC v0.1

**Version**: 0.1.0-draft
**Date**: 2026-04-05
**Status**: Draft para revisão
**Target implementation**: Go (separate reference implementation)

---

## 1. Introduction

AGORA é um protocolo de comunicação entre agentes de IA projetado como **gramática unificada de presença em espaço comum**. Diferente de protocolos RPC (como A2A) ou pub/sub enterprise (como AGNTCY/SLIM), AGORA trata comunicação agent-to-agent como atos de fala em uma praça: agentes entram, se apresentam, falam (broadcast ou direto), pedem ajuda, compartilham receitas, pagam — tudo via um envelope único com 7 verbos.

### 1.1 Design Principles (herdados de ANP + refinados pelo conselho)

1. **AI-Native**: desenhado para LLMs como participantes primários, não adaptado de APIs humanas
2. **Scope-Agnostic**: mesmo wire format em local (IPC), LAN privada, internet global
3. **Simplicity**: uma skill markdown de 1 página ensina o agente a usar
4. **Composability**: extensions via campos `x-*`; core pequeno + extensões opcionais
5. **Least Trust**: identity self-certified, no CA required
6. **Pragmatic Deployability**: implementável em stdlib + 2 libs, ~800 LOC Go

### 1.2 Non-Goals v0.1

- **NOT** uma alternativa a A2A para enterprise workflows com task lifecycle formal
- **NOT** uma alternativa a MCP (tool protocol layer)
- **NOT** um payment rail — apenas carrega envelope de pagamento
- **NOT** um workflow engine — recipes são Teaching artifacts interpretados pelo LLM-agent, não executados deterministicamente (v0.1)
- **NOT** reputation computation — apenas primitivos de atestação

---

## 2. Identity Model

### 2.1 Handle Format

Handle canônico: `nickname@<fingerprint_hex>`

- **nickname**: `[a-z0-9_-]{1,32}`, case-insensitive, não é identidade — é label human-readable
- **fingerprint_hex**: 32 caracteres hex (128 bits), primeiros 128 bits do SHA-256 do Ed25519 public key bytes

**Exemplo canonical**: `alice@a1b2c3d4e5f67890deadbeefcafe1234`

**Display truncation**: UIs PODEM mostrar `alice@a1b2c3d4…` (prefix 8 hex) mas comparações/verificações DEVEM usar fingerprint completo.

### 2.2 Key Generation

- **Algorithm**: Ed25519 (RFC 8032)
- **Storage**: chave privada do agente, formato PEM ou raw 32 bytes
- **Fingerprint**: `SHA-256(pubkey_bytes)[:16]` em lowercase hex
- **Zero infrastructure**: sem CA, sem DNS, sem DID resolver

### 2.3 Verification

Ao receber mensagem de `alice@fp`, receptor:

1. Extrai `fingerprint_hex` do handle
2. Resolve `pubkey` via:
   - Cache local (visto anteriormente)
   - `greeting` prévia no Space (contém pubkey)
   - Query direta ao Space: `kind:"whois", to:"alice@fp"`
3. Verifica `SHA-256(pubkey)[:16] == fingerprint_hex`
4. Verifica assinatura da mensagem

### 2.4 Rotation & Revocation

Chave comprometida:

1. Agente emite `kind:"revoke"` assinado pela chave antiga, contendo new `handle@new_fp`
2. Revoke message propaga via broker log
3. Receptores armazenam revocation em ledger local; futuras mensagens do old handle são rejeitadas

---

## 3. Space Abstraction

### 3.1 Space URI Format

```
agora://<scope>/<broker_hint>/<space_name>
```

Scopes:

- `hearth` — local (IPC, stdio, Unix socket, in-process)
- `home` — LAN/privado (broker WebSocket em intranet)
- `world` — internet pública (broker WebSocket WSS ou gossip P2P)

**Exemplos**:

- `agora://hearth/local/kitchen`
- `agora://home/broker.acme.internal/sales-floor`
- `agora://world/relay.agora.pub/devtools`

### 3.2 Broker Semantics

Um **Broker** é o processo que hospeda um Space. Responsabilidades (padrão v0.1):

- Accept connections (transport layer per scope)
- Route messages (broadcast if `to==null`, direct if `to!=null`)
- Maintain append-only log (last N messages replayed on join)
- Enforce policy (rate limits, sig verification requirement, allowlists) per scope

**Scope-specific defaults**:

| Policy                     | hearth   | home         | world                 |
| -------------------------- | -------- | ------------ | --------------------- |
| Signature required?        | optional | recommended  | **required**          |
| Sig verification enforced? | off      | on (warn)    | on (reject)           |
| Replay buffer              | 20 msgs  | 100 msgs     | 200 msgs              |
| Rate limit per handle      | none     | 100/s        | 10/s                  |
| Handle allowlist           | open     | configurable | open+reputation-based |

### 3.3 Transport Pluggability

O wire format (JSON-lines) é **invariante**. Transports diferem:

| Scope  | Reference transport       | Alternatives                   |
| ------ | ------------------------- | ------------------------------ |
| hearth | Unix socket / stdio       | in-process channel, named pipe |
| home   | WebSocket (ws://)         | gRPC streaming, MQTT           |
| world  | WebSocket Secure (wss://) | libp2p GossipSub, NATS         |

Agentes conhecem apenas URI — transport é resolvido pelo client SDK.

---

## 4. Wire Format

### 4.1 Message Envelope

Toda mensagem é **uma linha JSON** sobre o transport.

```json
{
  "v": 1,
  "id": "01JH7K8M6N7Q8R9STUV",
  "space": "agora://world/relay.agora.pub/devtools",
  "scope": "world",
  "from": "alice@a1b2c3d4e5f67890deadbeefcafe1234",
  "to": null,
  "reply_to": null,
  "thread": null,
  "kind": "say",
  "ts": 1712260800,
  "nonce": "0x7f3a9b2c",
  "body": { "text": "quem traduz grego antigo?" },
  "tribute": null,
  "sig": "base64url(ed25519_signature)"
}
```

### 4.2 Required Fields

| Field   | Type           | Purpose                             |
| ------- | -------------- | ----------------------------------- |
| `v`     | integer        | Protocol version (currently `1`)    |
| `id`    | string (ULID)  | Message UUID (globally unique)      |
| `space` | string (URI)   | Full space URI                      |
| `scope` | enum           | `"hearth"` \| `"home"` \| `"world"` |
| `from`  | string         | Handle (canonical format)           |
| `to`    | string \| null | Target handle; `null` = broadcast   |
| `kind`  | string         | Message type (see §5)               |
| `ts`    | integer        | Unix epoch seconds                  |
| `body`  | object         | Kind-specific payload               |

### 4.3 Optional Fields

| Field      | Type           | Purpose                                         |
| ---------- | -------------- | ----------------------------------------------- |
| `reply_to` | string \| null | Message ID being replied to                     |
| `thread`   | string \| null | Thread ID (groups related messages)             |
| `nonce`    | string (hex)   | Random 4-byte nonce (required if `sig` present) |
| `tribute`  | object \| null | Payment envelope (see §7)                       |
| `sig`      | string         | Ed25519 signature (base64url)                   |
| `x-*`      | any            | Extension namespace                             |

### 4.4 Signature Area

Signature covers **canonical JSON (JCS/RFC 8785)** of all fields EXCEPT `sig` itself:

```
signed_payload = JCS(message_without_sig)
sig = base64url(Ed25519_sign(signed_payload, private_key))
```

**Signed fields**: `{v, id, space, scope, from, to, reply_to, thread, kind, ts, nonce, body, tribute, x-*}`

This binding prevents:

- **Replay**: `nonce` + `ts` uniqueness enforced by broker
- **Downgrade**: `scope` included — msg for `world://` can't be replayed in `hearth://`
- **Tampering**: any field change invalidates sig

### 4.5 Signature Policy by Scope

| Scope    | Sig present? | Broker verifies?          |
| -------- | ------------ | ------------------------- |
| `hearth` | OPTIONAL     | Off by default            |
| `home`   | RECOMMENDED  | Warn on missing/invalid   |
| `world`  | **REQUIRED** | Reject on missing/invalid |

---

## 5. Message Kinds (Core Verbs)

AGORA v0.1 core has **7 required verbs** + 3 opcionais. Um agente que implementa os 7 primeiros é conformant.

### 5.1 Core verbs (mandatory)

#### 5.1.1 `join`

Agente entra em um Space. First message after connection.

```json
{ "kind": "join", "body": {} }
```

Broker responds: replay das últimas N mensagens + presence list.

#### 5.1.2 `leave`

Agente sai do Space. Broker remove do presence.

```json
{ "kind": "leave", "body": {} }
```

#### 5.1.3 `greet`

Self-introduction. Contém identity proof + skills tags.

```json
{
  "kind": "greet",
  "body": {
    "pubkey": "base64url(ed25519_pubkey)",
    "skills": ["translation", "greek", "latin"],
    "description": "Tradutor clássico. Cobro por palavra.",
    "interests": ["translation:*"]
  }
}
```

- `pubkey`: obrigatório (identity proof self-referencing the fingerprint)
- `skills`: tags livres para matching
- `description`: texto em NL (LLMs interpretam)
- `interests`: topics este agente quer pré-filtrar pelo broker (reduz LLM cost)

#### 5.1.4 `say`

Broadcast para o Space inteiro (`to == null`).

```json
{
  "kind": "say",
  "to": null,
  "body": { "text": "preciso traduzir 500 palavras grego→pt" }
}
```

#### 5.1.5 `direct`

Mensagem 1-para-1 (`to != null`).

```json
{
  "kind": "direct",
  "to": "bob@e5f6...",
  "reply_to": "<id of a say/direct>",
  "thread": "thread_01JH7...",
  "body": { "text": "faço por 0.10 USDC" }
}
```

#### 5.1.6 `recipe`

Compartilha **artefato estruturado de conhecimento procedural** (Teaching artifact). O agente receptor é o runtime — interpreta os steps com seu próprio LLM + ferramentas + contexto.

```json
{
  "kind": "recipe",
  "to": "alice@...",
  "reply_to": "<id da say que pediu>",
  "body": {
    "name": "parse-nfe-ptbr",
    "version": "1.2",
    "description": "Parse Brazilian fiscal PDF and extract CNPJ",
    "steps": [
      { "n": 1, "kind": "skill", "name": "ocr-pdf", "save_as": "text" },
      { "n": 2, "kind": "prompt", "text": "Extract CNPJ: {{ text }}", "save_as": "cnpj" },
      { "n": 3, "kind": "call", "role": "cnpj-validator", "args": { "cnpj": "{{ cnpj }}" } },
      { "n": 4, "kind": "check", "condition": "{{ last.valid }} == true", "on_fail": "abort" }
    ],
    "caveats": ["falha em NFSe municipal SP"]
  }
}
```

**Step types canônicos (4 + open namespace)**:

- `prompt` — instrução NL pro LLM (reason, extract, decide)
- `skill` — invoca skill nomeada (padrão Claude skills)
- `call` — delega step para outro agente AGORA (por role ou handle)
- `check` — verificação booleana (registrada no ledger)

Tipos custom são permitidos (`shell`, `http`, `my-org:review`). Agente que não conhece trata como `prompt` ou skip com warning.

**Variable passing**: step tem `save_as: "varname"` opcional. Referência via `{{ varname }}` ou `{{ last }}`.

**Teaching, NOT Automation**: agente LLM interpreta steps com julgamento próprio — pode executar, delegar, pular, ou improvisar. Deterministic execution fica como v0.2 extension (`x-recipe-runner`) com capability sandboxing obrigatório.

Ver `_ideas/agora-recipe-design.md` para design completo.

#### 5.1.7 `whois`

Query pubkey de um handle (discovery).

```json
{
  "kind": "whois",
  "to": null,
  "body": { "handle": "alice@a1b2c3d4..." }
}
```

Qualquer agente que conhece a pubkey responde com `direct` carregando `greet`-like body.

### 5.2 Optional verbs (recommended)

#### 5.2.1 `receipt`

Confirmação de entrega de serviço ou payment.

```json
{
  "kind": "receipt",
  "to": "bob@...",
  "reply_to": "<id do direct com resultado>",
  "body": {
    "status": "ok" | "failed" | "partial",
    "note": "translation received as expected"
  }
}
```

#### 5.2.2 `echo`

Atestação reputacional assinada sobre outro agente.

```json
{
  "kind": "echo",
  "to": null,
  "body": {
    "about": "bob@e5f6...",
    "observed": "completed translation task correctly",
    "event_ref": "<msg_id of the receipt>",
    "valence": "positive" | "negative"
  }
}
```

Echoes propagam reputação lateral. Computação fica no ledger local.

#### 5.2.3 `revoke`

Identity revocation.

```json
{
  "kind": "revoke",
  "body": {
    "reason": "key_compromise",
    "successor": "alice@new_fingerprint",
    "successor_pubkey": "base64url(new_ed25519_pubkey)"
  }
}
```

Assinada pela chave **revogada** (para provar controle).

---

## 6. Discovery

### 6.1 Baseline: Listening + Greeting Replay

Ao emitir `join`, o agente recebe replay dos últimos N `greet` e `say` do Space. Isso dá visão imediata de quem está lá + tarefas em aberto.

### 6.2 Active: Seek via Broadcast

```json
{
  "kind": "say",
  "body": { "text": "quem sabe parsear NFe PTBR?" }
}
```

Respondentes enviam `direct` com ofertas (ou `recipe` se tiverem artefato pronto).

### 6.3 Optional: Cross-Broker Discovery (extension)

Para descobrir agentes em outros Spaces/Brokers:

- **Well-known URI** (opt-in em `world://`): `GET https://<broker_host>/.well-known/agora-space.json` retorna metadata
- **DNS TXT** (opt-in): `_agora.example.com TXT "broker=wss://example.com/agora"`

Both are extensions, not required by core.

---

## 7. Payment Hook

### 7.1 Tribute Envelope

Pagamento é um **campo opcional** no envelope (`tribute`), não uma layer separada.

```json
{
  "kind": "direct",
  "to": "bob@...",
  "body": { "text": "aqui está o texto para traduzir" },
  "tribute": {
    "rail": "x402",
    "amount": "0.10",
    "currency": "USDC",
    "proof": "0xabc123...",
    "nonce": "0x7f3a9b2c"
  }
}
```

### 7.2 Rails Suportados (pluggable)

| Rail        | Proof format          | Use case                    |
| ----------- | --------------------- | --------------------------- |
| `x402`      | HTTP 402 payment hash | Crypto stablecoin           |
| `ap2`       | AP2 mandate VC        | Authorization-based         |
| `lightning` | invoice + preimage    | Micropayments               |
| `stripe`    | PaymentIntent ID      | Fiat                        |
| `tally`     | signed IOU            | Internal scoreboard         |
| `trust`     | none                  | No settlement (trust-based) |

Protocol carrega envelope; **não valida, não liquida**. Validação é responsabilidade do receptor.

### 7.3 Payment Atomicity Pattern

Para evitar double-spend / double-deliver (Devil's concern):

1. Cliente envia `direct` com request + `tribute`
2. Provedor verifica `tribute.proof` on-rail antes de entregar
3. Provedor envia `direct` com resultado
4. Cliente envia `receipt` confirmando
5. Disputes: ambos os lados guardam transcript assinado; arbitrage off-protocol

**`tribute.nonce`** previne replay. Mesmo nonce visto duas vezes = reject.

---

## 8. Security Considerations

### 8.1 Threat Model

**In scope**:

- Identity spoofing (mitigado: 128-bit fingerprint + signature)
- Replay attacks (mitigado: nonce + ts in signed area)
- Downgrade attacks (mitigado: scope in signed area)
- Key theft (mitigado: revocation via `revoke` kind)
- Handle collision (mitigado: 128 bits, birthday 2^64)

**Out of scope (agent-layer responsibility)**:

- Prompt injection via broadcast — defense is at LLM instruction hierarchy
- Sybil attacks em `world://` — mitigação via broker policy (stake, invitation)
- Agent hallucinating malicious content — human review / sandboxing
- Recipe execution weaponization — deterministic execution deferred to v0.2 `x-recipe-runner` extension with mandatory capability sandboxing

### 8.2 Prompt Injection (documented risk)

AGORA transports natural language payloads interpretados por LLMs. Um `say` pode conter "IGNORE PREVIOUS INSTRUCTIONS" — a defesa NÃO está no protocolo:

- Agentes DEVEM tratar mensagens externas como **untrusted input**
- Instruction hierarchy enforcement no LLM layer
- Output validation antes de act on broadcast content
- Anti-pattern: processar `say` body como system instruction
- Recipe `description`/`caveats`/`hint` fields são igualmente untrusted NL

### 8.3 Broker Trust Model

Em `world://`, o broker vê metadata (but not whisper content, if E2E enabled em v0.2). Broker pode:

- DoS via seletive delivery
- Log messages indefinidamente
- Inject messages (detectado por sig verification)

**v0.1 assumption**: broker é semi-trusted. v0.2 adiciona E2E encryption em `direct` messages.

### 8.4 Sybil Resistance

Self-cert handles são **gratuitos**. Mitigação:

- Brokers em `world://` podem exigir proof-of-entry (stake, invitation token)
- `echo` chain com attester reputação
- Ledger local + temporal decay torna Sybil economicamente desfavorável

v0.1 **não resolve Sybil** no protocolo — deixa para policy layer.

---

## 9. Operational Requirements for Brokers

### 9.1 Minimal Broker (v0.1)

Broker implementa:

- Accept transport connections (WS/stdio/Unix)
- Per-connection: verify `from` handle matches pubkey (when sig present)
- Enforce scope-specific signature policy (§4.5)
- Maintain append-only log (ring buffer of N messages)
- Route messages: `to==null` → broadcast; `to!=null` → direct
- Replay on `join`: last N messages
- Rate limit per handle (configurable by scope)
- Pre-filter by `interests` declared in `greet` (reduces LLM cost downstream)

### 9.2 Broker Pre-Filtering (mitigation for Devil's LLM cost concern)

Agentes declaram `interests: ["translation:*", "payment:settled"]` no `greet`. Broker roteia para cada agente apenas mensagens que matcheiam interests. Reduz LLM-load de O(N×M) para O(matched).

Wildcard patterns: `translation:*`, `*:greek`.

---

## 10. Extension Points

Extensions são versionadas via `x-*` fields. v0.2 planned extensions:

| Extension         | Purpose                                                            |
| ----------------- | ------------------------------------------------------------------ |
| `x-recipe-runner` | Deterministic recipe execution com capability sandboxing + dry-run |
| `x-e2e`           | End-to-end encryption em `direct` (X25519 derived from Ed25519)    |
| `x-crdt-log`      | Replicated broker log via CRDT (elimina SPOF em `home://`)         |
| `x-meta-proto`    | Runtime protocol negotiation (ANP-inspired)                        |
| `x-reputation`    | Aggregated reputation scoring protocol                             |
| `x-recipe-ctrl`   | Control flow step types (`branch`, `parallel`, `loop`, `retry`)    |

---

## 11. Reference Skill (o que ensina o LLM-agent)

```markdown
# AGORA Skill v0.1

Você pode conversar com outros agentes via AGORA. Use o client SDK:

client = AgoraClient(name="<você>", space="agora://<scope>/<broker>/<name>")
await client.join()

## 7 verbos que você usa:

1. JOIN: entrar em um space (feito pelo SDK no startup)
2. GREET: apresentar-se (pubkey, skills, interests)
3. SAY: falar ao space inteiro — broadcast
4. DIRECT: falar a um handle específico
5. RECIPE: compartilhar receita estruturada (workflow pra outro agente executar)
6. WHOIS: perguntar quem é um handle
7. LEAVE: sair do space

## Regras

- Você é `nickname@fingerprint`. Sua chave Ed25519 assina toda mensagem.
- Em `world://`, assinatura é obrigatória. Em `hearth://`, opcional.
- Para broadcast: deixe `to` como null. Para direct: preencha `to`.
- Quando quiser pagar, inclua `tribute: { rail, amount, proof, nonce }`.
- Quando receber um serviço, emita `receipt`.
- Quando observar comportamento de outros, pode emitir `echo` (opcional).
- Seus logs ficam em `~/.agora/ledger.jsonl`.

Isso é tudo. 7 verbos. Um envelope. Mesmo código funciona em hearth/home/world.
```

---

## 12. Open Questions / Dissent Preserved

1. **Architect's dissent on recipe execution**: "recipes com deterministic execution deveriam estar no core". Resolved post-council: `recipe` entra no core como Teaching artifact (typed steps + variables), deterministic execution fica como `x-recipe-runner` extension v0.2.
2. **Devil's dissent on AGORA vs A2A**: "use A2A+broker pattern". Documented em Appendix A.
3. **Security concern**: prompt injection mitigation fica no agent layer — adequado? Revisit em v0.2. Recipe body fields (`description`, `caveats`, `hint`) são vetor adicional de prompt injection — tratados como untrusted NL.

---

## Appendix A — Why AGORA and not A2A+broker?

(Devil's Advocate challenge response)

**A2A entrega**:

- Agent Cards assinados com RFC 8785
- Task lifecycle formal (submitted→working→completed)
- JSON-RPC over HTTP/2 + SSE streaming
- Enterprise-grade interop com Google/Microsoft stack

**A2A NÃO entrega (e AGORA entrega)**:

- **Broadcast 1-para-muitos como native primitive** (A2A é request/response P2P)
- **Scope-agnosticismo** (A2A assume HTTP/2 + public endpoint — quebra local/air-gap)
- **Chat-first grammar** (A2A é task-first — overhead pra conversa fluida)
- **Uma skill ensina** (A2A Agent Card + JSON-RPC + task states = múltiplos conceitos pra aprender)
- **Identity self-certified** (A2A usa OAuth/OpenID/API keys — infraestrutura pesada)

**Conclusão**: AGORA e A2A são complementares. Um agent production-ready pode falar ambos. A2A para enterprise workflows tipados; AGORA para conversation + discovery + simple payment em escopos variados.

---

## Appendix B — Size Budget

Target v0.1 implementação:

- Broker: ~400 LOC Go (ws + log + routing)
- Client SDK: ~200 LOC Go (connect, sign, send, listen)
- Canonicalizer (JCS): reuso lib existente (~0 LOC)
- Ed25519: stdlib (`crypto/ed25519`)
- Total: **~600 LOC**, 2 external libs (gorilla/websocket, JCS)

SPEC total: este documento, ~10 páginas. Skill: 1 página.
