# AGORA Council Round 2 — Synthesis

**Date**: 2026-04-05
**Context**: Hybrid path decided (AGORA naming + ANP-Lite wire format Nostr-shaped). Council re-runs para resolver implementation decisions.
**Archetypes**: Architect, Pragmatic Engineer, Security Advocate, Product Mind, The Thinker
**Missing**: Devil's Advocate (no response — Thinker's reframing provides partial substitute)

---

## Phase 2 — Opening Statements (condensadas)

### Architect: "Envelope invariante + NIP-01 compat como red line"

Hybrid defensável **se e somente se** compat Nostr NIP-01 for tratado como invariant. Diferenciação AGORA vive em **kinds + scope tag + sig policy enforcement**, não no wire. Risk: schizofrenia de identidade se ninguém souber responder "AGORA é Nostr ou é irmão?". Red line: evento AGORA deve ser verificável por qualquer lib Nostr padrão.

### Pragmatic: "Hybrid reduz effort — ship em fim de semana"

Adotar Nostr-shape é menos risco crypto + reuso de libs auditadas (go-nostr, nostr-tools). LOC budget ~600 Go continua realista. Recipe como `kind 30078` ou `31000` (parameterized replaceable). Import go-nostr, não fork. Single-relay MVP.

### Security: "Nostr wire é upgrade crypto, MAS Nostr security model ≠ AGORA"

NIP-01 canonicalization elimina categoria inteira de bugs. Attack surface menor vs JCS. **Preço**: Nostr foi desenhado pra gossip público adversarial — AGORA tem scopes com conteúdo semi-privado. Red line: ephemeral keys NÃO podem publicar em `world://`, d-tag namespace obrigatório, `content` field NUNCA embedded JSON (só string literal ou base64).

### Product: "Nostr é infrastructure invisível, recipe é magic visível"

Adoption AUMENTA se esconder Nostr atrás dos 3 verbos semânticos. Dev nunca escreve `kind: 2` — escreve `client.say("text")`. Recipe é feature headline, kinds são wire detail. Hello-world em <5min com `npx agora init && npx agora send "hi"`. Killer demo: Claude↔Cursor coordenando refactor+test+fix.

### The Thinker: "Drift metafórico — AGORA está misturando 3 frames"

Radical reframing: AGORA drifou de _praça_ → _correio marítimo_ → _signed gossip_ sem nomear a mudança. O problema real não é "conversa em praça" — é **pipeline de produção com agentes como lentes epistêmicas parciais**. Propõe:

- **Separar transport layer (Nostr-shaped events) de artifact layer (content-addressed recipes)**
- Recipe vira ARTIFACT com ID próprio (`recipe_id = hash(canonical_body)`), independente do event que transporta
- 3 primitivos novos: `trace` (execution log assinado), `cross` (scope-crossing transform), `grip` (handoff gradual)

A questão que precede todas as 10: **AGORA é protocolo de MENSAGENS ou de ARTEFATOS?**

---

## Phase 3 — Tensions Debate

| #                 | Tensão                                    | Lados                                                                                   | Resolução                                                                                                                                          |
| ----------------- | ----------------------------------------- | --------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| T1                | Sig obrigatoriedade                       | Arch: tiered enforcement / Prag+Sec: always mandatory / Prod: invisível auto-keypair    | Sig sempre presente no wire (Nostr compat). Enforcement tiered por scope (hearth skip, home warn, world reject). SDK auto-gera keypair → invisível |
| T2                | Recipe kind number                        | Arch: 1001 / Prag: 30078-31000 / Sec: 40 (core immutable) / Thinker: artifact não event | **Thinker wins**: recipe é 2 kinds — kind 1100 artifact (immutable) + kind 31100 latest-pointer (parameterized replaceable)                        |
| T3                | Recipe como kind vs artifact              | 4 archetypes: kind / Thinker: artifact com identity própria                             | **Thinker wins**: recipe tem `recipe_id = hash(canonical_body)` + version. Echoes referenciam `recipe_id@version`, não event.id                    |
| T4                | Scope URI vs relay URL                    | Arch+Prag+Sec: scope como signed tag, relay URL config                                  | Convergência: `["scope","world"]` + `["space","agora://..."]` mandatórios no signed area. Relay URL é client config                                |
| T5                | Storage adapter na SPEC?                  | Arch+Prag: SDK / Sec: policy enforcement / Prod: invisível                              | SPEC documenta SOMENTE que ephemeral keys rejeitam `world://` e `home://`. Interface fica no SDK                                                   |
| T6                | Replaceable state v0.1                    | Arch: sim / Prag: sim / Sec: sim COM controles / Prod: wow-feature                      | v0.1 core, d-tag namespace obrigatório (`<app>:<key>`), revocation primitive mandatory                                                             |
| T7                | Signing lib strategy                      | Arch: fork patches / Prag: import / Sec: import + pin + SBOM                            | Import go-nostr (não fork). Pin exact versions. Security monitoring. Supply chain vigilance                                                        |
| T8                | Relay federation                          | Arch: multi-relay / Prag: single-relay MVP / Sec: ≥2 relays mandatory / Prod: opt-in    | Single-relay v0.1 default. Multi-relay com reconciliation em v0.2 extension                                                                        |
| **T9 (Thinker)**  | **Message protocol vs Artifact protocol** | Arch+Prag+Sec+Prod: message / Thinker: separar camadas                                  | **Accept**: transport é Nostr-shaped messages, artifact layer (recipe/trace) tem content-addressed identity                                        |
| **T10 (Thinker)** | **Scope como transform vs label**         | Arch+Prag+Sec: label signed tag / Thinker: transform primitive `cross`                  | **Compromise**: scope como signed tag em v0.1. `cross` primitive defer v0.2 quando houver federation                                               |

---

## Phase 4 — Position Evolution

| Archetype | Shift após Thinker's reframing                                                                                                               |
| --------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Architect | Aceita separação message/artifact layer — alinha com "envelope invariante". Recipe como content-addressed artifact reforça boundary          |
| Pragmatic | Aceita recipe como 2 kinds (kind 1100 immutable + kind 31100 replaceable pointer) — permite versionamento sem duplicar state                 |
| Security  | **Entusiasta**: content-addressed recipe_id resolve fake attestation (`success_rate` verificável via trace chain). Trace kind é security win |
| Product   | Aceita 2 kinds internamente desde que SDK esconda: `client.publishRecipe(recipe)` → internamente emite artifact + pointer. User vê 1 coisa   |
| Thinker   | Mantém reframing, aceita defer de `cross` e `grip` pra v0.2                                                                                  |

---

## Phase 5 — Synthesis & Decisions

### Decisões aprovadas (convergência ou resolução por argumento)

#### 1. Hybrid path: APPROVED

Condição: NIP-01 compat é invariant. Evento AGORA = evento Nostr verificável por qualquer lib padrão.

#### 2. Wire format: Nostr NIP-01 event shape

```json
{
  "id": "<hex sha256 of canonical array>",
  "pubkey": "<hex ed25519 or schnorr>",
  "created_at": <unix seconds>,
  "kind": <int>,
  "tags": [["name","val",...], ...],
  "content": "<string — NEVER embedded JSON>",
  "sig": "<hex signature>"
}
```

#### 3. Kind space allocation (Nostr convention + AGORA reserves)

- **0-999**: reserved Nostr core
- **1000-9999**: regular events (AGORA uses 1100-1199)
- **10000-19999**: replaceable (overwrite by pubkey+kind)
- **20000-29999**: ephemeral (not stored)
- **30000-39999**: parameterized replaceable (overwrite by pubkey+kind+d-tag) — AGORA uses 31100-31199
- **40000+**: application-specific

**AGORA core kinds v0.1**:
| Kind | Name | Type | Purpose |
|---|---|---|---|
| 1 | hello | regular | Capability handshake (reuses Nostr) |
| 1100 | recipe-artifact | regular (immutable) | Recipe content published |
| 1101 | trace | regular | Execution log (verifiable) |
| 1102 | message | regular | Semantic message (say/direct/whois/receipt/echo) |
| 1103 | request | regular | Request expecting reply |
| 1104 | response | regular | Reply to request (carries req_id) |
| 1105 | nack | regular | Typed rejection |
| 1106 | revoke | regular | Identity revocation |
| 31100 | recipe-latest | param. replaceable | Pointer to latest recipe version |
| 31101 | presence | param. replaceable | Agent online status per space |

#### 4. Verbos protocolares (3) + semantic kinds

**3 verbos no wire**: `publish` / `request` / `ack`

**Semantic kinds surfaced via SDK** (não são verbos wire):

- `say` → publish kind=1102 (broadcast, to=null)
- `direct` → publish kind=1102 (to=handle)
- `greet` → publish kind=1 (hello)
- `recipe` → publish kind=1100 (artifact) + kind=31100 (latest pointer)
- `trace` → publish kind=1101
- `whois` → request kind=1103 (reply via 1104)
- `receipt` → publish kind=1102 with `["status","ok|failed"]`
- `echo` → publish kind=1102 with `["about","<handle>"]` + `["recipe_ref","<recipe_id>@v"]`
- `revoke` → publish kind=1106

#### 5. Signature policy

- **Always in wire** (Nostr compat, non-negotiable)
- **Enforcement tiered** by broker/relay policy per scope:
  - `hearth`: skip verification (dev convenience)
  - `home`: verify, warn on failure
  - `world`: verify, reject on failure
- **SDK auto-generates keypair** on first use → signing is invisible to dev

#### 6. Tag registry v0.1 (formal, versioned separate doc)

**Nostr-standard tags** (reuse):
| Tag | Purpose |
|---|---|
| `p` | pubkey reference (to, mention) |
| `e` | event reference (reply_to, thread) |
| `d` | d-tag for parameterized replaceable |

**AGORA-standard tags** (defined in v0.1):
| Tag | Required | Purpose |
|---|---|---|
| `scope` | ✅ | `hearth` \| `home` \| `world` |
| `space` | ✅ | `agora://<scope>/<broker_hint>/<name>` |
| `v` | ✅ | Protocol version (`"0.2"`) |
| `req_id` | on request | UUID for request/response pairing |
| `thread` | optional | Groups related messages |
| `status` | on receipt | `ok` \| `failed` \| `partial` |
| `recipe_ref` | on echo/trace | `<recipe_id>@<version>` |
| `recipe_name` | on recipe-latest | namespace key (e.g., `parse-nfe-ptbr`) |
| `expires_at` | optional | Unix seconds |
| `nonce` | when sig present | Anti-replay hex 16 bytes |
| `about` | on echo | Handle being attested |

**Namespacing**: custom tags MUST prefix with `x-` or org-namespace (e.g., `x-acme:legal-review`).

#### 7. Scope binding in signed area

Scope-related tags MUST be in signed portion:

- `["scope", "hearth|home|world"]`
- `["space", "agora://..."]`
- `["v", "0.2"]`
- `["nonce", "<hex>"]`

Broker MUST reject events whose `scope` tag doesn't match broker's scope.

#### 8. Recipe as artifact (Thinker's reframe accepted)

Recipe has **content-addressed identity** separate from event ID:

```json
{
  "kind": 1100,
  "content": "",
  "tags": [
    ["scope", "world"],
    ["recipe_id", "<hash(canonical body)>"],
    ["recipe_name", "parse-nfe-ptbr"],
    ["recipe_version", "1.2"],
    ["x-recipe", "<canonical json of recipe body>"]
  ]
}
```

Accompanied by replaceable "latest pointer":

```json
{
  "kind": 31100,
  "content": "",
  "tags": [
    ["d", "parse-nfe-ptbr"],
    ["recipe_id", "<hash>"],
    ["recipe_version", "1.2"]
  ]
}
```

**Why two kinds**: artifact (immutable) allows citation by recipe_id; latest-pointer (replaceable) allows "what's current version of recipe X by author Y".

**Echoes and traces reference `recipe_id@version`**, not event.id. This resolves fake attestation.

#### 9. Trace primitive (Thinker's addition, accepted v0.1)

```json
{
  "kind": 1101,
  "content": "<execution_log>",
  "tags": [
    ["scope", "world"],
    ["recipe_ref", "<recipe_id>@1.2"],
    ["outcome", "success|partial|failed"],
    ["steps_executed", "4"],
    ["steps_improvised", "1"]
  ]
}
```

Trace is signed by executing agent. Enables verifiable `success_rate` for recipes (no more fake attestations).

#### 10. Replaceable state primitive

v0.1 core with controls:

- d-tag MUST be namespaced: `<app>:<key>` (e.g., `agora:presence`, `myapp:cart-42`)
- Only pubkey that authored can overwrite (same d-tag)
- Revocation via kind 1106 before new replaceable publish

#### 11. Storage adapter

SDK concern. SPEC documents ONLY:

- Ephemeral keys MUST be rejected in `home://` and `world://` scopes
- Spec recommends filesystem (0600) as default, with env/vault/ephemeral as alternatives

#### 12. Signing lib strategy

- **Import**, don't fork (`go-nostr`, `nostr-tools` TypeScript, `python-nostr`, `rust-nostr`)
- Pin exact versions
- Monitor supply chain (Socket.dev, npm audit, Snyk)
- Publish AGORA wrapper libs on top (`agora-go`, `agora-ts`) with tag validation

#### 13. Relay federation

v0.1: single-relay default in SDK
v0.2 extension (`x-multi-relay`): multi-relay with reconciliation

### Unresolved (preserved as dissent)

1. **Security's concern about ephemeral keys in `home://`**: compromise proposed (reject ephemeral in home+world). Security may still push for `home://` to require persistent keys.
2. **Architect's insistence on formal tag registry doc separate from spec**: accepted — tag-registry-v0.1.md is separate companion doc.
3. **Thinker's `cross` and `grip` primitives**: deferred to v0.2 extensions. Thinker retains position that they're load-bearing for federation and pipeline coordination.
4. **Devil's Advocate no-show**: challenges documented via Thinker's reframing. Specific concerns not raised: Cursor/Codex sandbox storage, malicious relay reuse, LLM canonicalization inconsistency (mitigated by signing libs).

### Open Questions (need user input)

1. **Thinker's fundamental question**: AGORA is message protocol OR artifact protocol OR both-in-layers? **Proposed answer**: both-in-layers (transport = Nostr-shaped, artifacts = content-addressed). **Need user confirmation.**
2. **Kind number specifics**: 1100-1199 AGORA reserves sane? Or should we negotiate NIP allocation with Nostr community?
3. **Secp256k1 Schnorr (Nostr default) vs Ed25519 (our prior spec)**: Nostr uses secp256k1 Schnorr (BIP-340). Ed25519 is what we'd use if rolling our own. **Choosing Nostr compat means Schnorr**. Implication: slightly different keypair format. Trade-off: compat vs algorithm preference.
4. **Recipe content in tag vs content field**: Security said no embedded JSON in `content`. Current proposal puts canonical recipe JSON in `x-recipe` tag. Alternative: base64-encode in `content`. **Need decision.**

---

## Recommended next step

Write **AGORA SPEC v0.2** incorporating these decisions:

1. Nostr NIP-01 wire format (Schnorr + secp256k1)
2. Kind allocations (1-1106 + 31100-31101)
3. Tag registry companion doc
4. Recipe as artifact layer (kind 1100 + 31100)
5. Trace primitive (kind 1101)
6. Scope binding via signed tags
7. 3 protocol verbs + semantic SDK methods
8. Go SDK on top of go-nostr

**Before writing**: need user approval on:

- Thinker's reframe (message+artifact layers)
- Schnorr vs Ed25519 (Nostr compat means Schnorr)
- Recipe embedding strategy (tag vs base64 content)

---

**Key Point**: Round 2 resolveu 80% das tensões operacionais. O insight mais importante foi o Thinker's reframing: **AGORA tem duas camadas** (transport Nostr + artifacts content-addressed). Aceitar isso resolve múltiplas incoerências e destrava versioning de recipes + trace verifiável. Wire format Nostr-shaped é aceito por todos os 5 arquétipos com condições cirúrgicas.
