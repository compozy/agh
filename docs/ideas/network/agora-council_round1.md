# AGORA — Council Synthesis v0.1

**Date**: 2026-04-05
**Facilitator**: Main agent (orchestrating 5 archetypes)
**Archetypes**: Architect, Pragmatic Engineer, Security Advocate, Product Mind, Devil's Advocate

---

## Phase 2 — Opening Statements (resumo)

### The Architect — "Invariância de envelope com polimorfismo de política"

Princípio central: o envelope (wire format + verbos + semântica) é invariante em todos os escopos; diferenças de escopo são política do Space, não do protocolo. Rejeita reinventar DID; aceita Ed25519 self-cert como ground truth. Red line: envelope igual em hearth/home/world + identity self-certified.

### The Pragmatic Engineer — "Implementável em um sábado ou morre"

Princípio central: se um dev Go júnior não consegue ship um cliente em 800 linhas num fim de semana, a SPEC está errada. Defende JSON-lines, broker WS boring, 5-7 verbos máximo, `tribute` como string+blob pluggable. Red line: v0.1 em 8 páginas, stdlib + 2 libs.

### The Security Advocate — "Assume breach"

Princípio central: threat model primeiro. Sybil + prompt injection via broadcast + replay são ataques reais nesta superfície. Exige: 128 bits de fingerprint, nonce+ts+scope assinados, capability sandboxing em Yarn, revocation path. 4 red lines (handle curto, sem nonce, Yarn sem sandbox, sem revogação).

### The Product Mind — "Delight em 30s de gif"

Princípio central: o primeiro usuário é dev solo num sábado mexendo com Claude, não enterprise. Quickstart em <5min sem infra. Skill em 1 página. Diferencial shareable: "agentes que ensinam agentes" (Yarn/Teach). Broadcast+direct como mesma gramática. Red lines: <5min boot, skill de 1 página, envelope único.

### The Devil's Advocate — "Cancele, use A2A"

Argumento central: constroi `agora-over-a2a` (broker pattern sobre A2A) e herda Google/MS ecosystem. Ataca scope-agnosticismo como fantasia (semânticas de delivery diferem por transport). Levanta unknown-unknown crítico: **custo computacional de LLM ouvindo praça é O(N×M) LLM-calls** — economicamente inviável em escala.

---

## Phase 3 — Tensões Principais

| #   | Tensão                    | Lado A                               | Lado B                             |
| --- | ------------------------- | ------------------------------------ | ---------------------------------- |
| T1  | Handle length             | Security: 32 hex (128 bits)          | Product: 8 hex shareable           |
| T2  | Signature obrigatoriedade | Security: universal                  | Product+Pragmatic: por escopo      |
| T3  | Yarn executável no core   | Product+Architect: sim, diferencial  | Devil+Security: RCE vector         |
| T4  | AGORA vs A2A+broker       | Devil: use A2A                       | Outros: A2A não entrega chat+scope |
| T5  | JCS canonicalization      | Security+Architect: sim, obrigatório | Pragmatic: ordem simples           |
| T6  | Meta-protocol no core     | Security: incluir com versioning     | Outros: fora                       |
| T7  | Reputation primitive      | Arch+Sec+Prod: Echo como opt         | Pragmatic+Devil: fora              |
| T8  | Custo LLM listening       | Devil: inviável                      | Outros: não abordado               |

---

## Phase 4 — Position Evolution (modelada)

8 shifts identificados, detalhados na tabela principal. Convergências-chave:

- Security + Product convergem em **2-tier signing**: recomendada em hearth/home, obrigatória em world, COM scope sempre no signed area
- Devil + Architect convergem em **envelope invariante, garantias divergentes**: wire idêntico, QoS/delivery policy por scope
- Architect + Pragmatic + Product convergem em **Yarn fora do core**, `teach` conversacional dentro
- Security + Pragmatic convergem em **JCS obrigatório** (1 lib Go resolve custo)

---

## Phase 5 — Synthesis & Recommendations

### Consenso

- **Identity**: Ed25519 self-cert (`handle@fingerprint`) — sem CA, sem resolver, zero infra
- **1-para-muitos vs 1-para-1**: derivação unificada, mesmo envelope, campo `to` opcional
- **Wire format**: JSON-lines
- **Discovery baseline**: listening passivo + Greeting no broker log
- **Agent Description**: opcional em conteúdo, com identity proof mínimo
- **Meta-protocol**: FORA do core v0.1 (versão no envelope suficiente)
- **Reputation computation**: local e subjetivo; `kind:"echo"` como primitivo opcional portável

### Decisões (via convergência de argumentos)

- **Handle length**: 32 hex (128 bits) como canonical; displays podem truncar
- **Signature**: obrigatória em `world://`, recomendada em `home://`, opcional em `hearth://` — MAS `scope` sempre no signed area (quando sig presente)
- **Nonce + TS**: obrigatórios no signed area quando sig presente (anti-replay + anti-downgrade)
- **Canonicalization**: JCS/RFC 8785
- **Payment hook**: campo `tribute` opcional no envelope com `{rail, amount, proof, nonce}`; rail-agnóstico
- **Learning primitive**: `kind:"teach"` no core (conversacional, turnos iterativos); **Yarn executável como v0.2 extension com sandboxing obrigatório**
- **Revocation**: `kind:"revoke"` assinado pela chave comprometida + new-identity attestation
- **Space abstraction**: broker-mediated por default, transport-pluggable; broker pré-filtra por `kind`/`to`/`interests` pra mitigar LLM cost
- **Discovery extensions**: well-known URI e DNS TXT como opt-in cross-broker

### Tensões não-resolvidas (preservadas como dissent)

1. **Devil's "use A2A"**: documentado como explicit dissent. AGORA justifica stack própria por: (a) A2A é HTTP-bound (quebra local/air-gap), (b) task lifecycle é overhead pra chat conversacional, (c) broadcast não é native primitive em A2A, (d) scope-agnosticismo incompatível com Agent Cards HTTP-served.
2. **Devil's LLM cost concern**: acknowledged. Mitigação em v0.1: broker pre-filtering + `interests` tag no Greeting + `to` field. Sem solução perfeita — resolver em v0.3 com subscription topics.
3. **Architect's "Yarn deveria estar no core"**: preservado. v0.1 ships `teach` only; Yarn fica como primeira extension proposal.
4. **Pragmatic's "JCS adiciona lib"**: aceito que é lib extra, justificado por leverage de ecossistema + correctness garantida.

### Primary Recommendation

Ship **AGORA SPEC v0.1** conforme o documento separado (`_ideas/agora-spec-v0.1.md`), priorizando:

1. Envelope invariante cross-scope (Architect win)
2. Implementável em ~800 linhas Go, skill em 1 página (Pragmatic+Product win)
3. Crypto correctness (sig + nonce + ts + scope signed) com escopo gradual (Security win parcial)
4. Diferencial shareable: `teach` como primitivo conversacional (Product win)
5. Broker pre-filtering como resposta à LLM cost (Devil acknowledged)

### Risk Mitigation

- **Sybil em world://**: Echoes com attester pubkey + temporal decay; brokers em world podem exigir proof-of-entry (stake, invitation, etc) — policy layer
- **Prompt injection via broadcast**: documentado em security considerations; defense is at agent LLM layer (instruction hierarchy, external input sandboxing), não no protocol
- **Handle collision**: 128 bits torna birthday attack 2^64 — inaceitavelmente caro
- **Broker SPOF em home**: v0.1 assume single-process; v0.2 adiciona replicated log (CRDT) como extension
- **Payment atomicity**: `receipt_nonce` + `delivery_receipt` assinado antes de release — padrão de 2-phase commit leve

---

**Key Point**: Os 5 arquétipos convergem em ~70% das decisões. Disagreements reais ficam em: (1) sig obrigatoriedade em hearth, (2) Yarn executável agora vs depois, (3) use-A2A alternative. As três foram resolvidas via ou tiering por escopo, ou extensão v0.2, ou dissent explícito documentado.

---

## Post-Council Refinement (2026-04-05)

Após o council, via brainstorming skill, duas decisões foram refinadas:

### Refinement 1: `hail` → `say`

**Motivação**: "hail" é arcaico e colide com "hailstorm". `say` é mais neutro, 3 letras, pareia limpo com `direct`.

**Mudança**: verbo broadcast renomeado de `hail` para `say`. Sem impacto estrutural.

### Refinement 2: `teach` + Yarn → `recipe` no core

**Motivação**: duas primitivas separadas (`teach` conversacional + `Yarn` executável como extension) era over-engineering. Um único primitivo tipado cobre ambos casos de uso.

**Mudança consolidada**:

- `teach` removido do core (era 5.1.6)
- `recipe` entra no core como 5.1.6 (substituição 1:1, mantém 7 verbos core)
- Recipe é **Teaching artifact com typed steps + variables** (não executável automático)
- Agente LLM é o runtime — interpreta steps com próprio julgamento
- 4 step types canônicos: `prompt`, `skill`, `call`, `check` + open namespace
- Variable passing via `save_as` + `{{ vars }}` / `{{ last }}`
- Deterministic execution fica como v0.2 extension `x-recipe-runner` (com capability sandboxing)

**Resolve**:

- Architect's dissent sobre Yarn no core → atendido parcialmente (typed semantic intent dentro do core)
- Security's concern sobre Yarn executável → execução fica fora (v0.2)
- Product's "diferencial shareable" → preservado via recipe artifact

**Conversational learning**: sem `teach`, acontece via `say`/`direct` com NL. Mesma praça, ambos modos (conversacional livre + artefato estruturado).

### Documentos atualizados

- `_ideas/agora-spec-v0.1.md`: `hail`→`say`, `teach`→`recipe`, skill markdown, extensions table, security notes
- `_ideas/agora-recipe-design.md`: design completo do recipe primitive (NOVO)

### Extension table atualizada

| Extension (v0.2+) | Purpose                                                    |
| ----------------- | ---------------------------------------------------------- |
| `x-recipe-runner` | Deterministic recipe execution com capability sandbox      |
| `x-recipe-ctrl`   | Control flow steps (`branch`, `parallel`, `loop`, `retry`) |
| `x-e2e`           | End-to-end encryption em `direct`                          |
| `x-crdt-log`      | Replicated broker log via CRDT                             |
| `x-meta-proto`    | Runtime protocol negotiation                               |
| `x-reputation`    | Aggregated reputation scoring                              |
