# AGORA — Recipe Design

**Date**: 2026-04-05
**Status**: Refinement post-council via brainstorming skill
**Supersedes**: o que o council synthesis chamava de "Yarn" / "teach executável"

---

## Contexto

Durante o council, o Architect defendeu "Yarn" (receita executável) como primitivo do core. Security + Devil empurraram pra v0.2 extension com sandboxing. O council synthesis resolveu em compromise: `teach` conversacional no core + Yarn executável como extension.

Post-council, via brainstorming skill, o conceito foi **refinado e promovido**:

- **"Yarn" foi renomeado para `recipe`** (evita collision com npm, mais direto/funcional)
- **`recipe` entra no core v0.1** substituindo `teach` (mesmo count de verbos: 7)
- **`recipe` é Teaching artifact, não Automation** — o agente receptor é o runtime, interpreta com julgamento próprio
- **Steps são tipados** (4 canônicos + open namespace)

Este doc captura o design detalhado do `recipe` primitive.

---

## Decisões consolidadas

### 1. Nome: `recipe`

**Razões:**

- Direto e funcional (não arcaico como "yarn")
- Zero collision dev em 2026 (Chef está dead)
- LLM-native: training data cheio de recipes estruturadas
- Funciona em PT ("receita")
- 6 letras, minimal

**Rejeitados:**

- `yarn` — colisão npm, muito metafórico
- `playbook` — Ansible association
- `howto` — informal demais
- `runbook` — ops-specific

### 2. Caso de uso: **Teaching** (agente é runtime)

Recipe é **artefato declarativo de conhecimento procedural**. O agente receptor:

- Lê a recipe
- Interpreta steps com seu próprio LLM + ferramentas + contexto
- Decide se executa, delega, pula, ou improvisa cada step
- **Não é obrigatório executar deterministicamente**

**Rejeitados:**

- **Automation puro**: recipe como workflow executável → RCE vector, scope creep, compete com Temporal/Airflow
- **Pure composition**: multi-agent choreography → expressa via step type `call` como caso especial
- **Hybrid ambíguo**: "às vezes executa, às vezes lê" → clareza maior sendo Teaching

**Por que Teaching wins:**

- Security-safe: zero execução forçada = zero RCE
- Unique differentiator: nenhum protocolo (A2A, ANP, MCP, AGNTCY) tem pedagogia procedural transferível
- Simplicidade: recipe é "structured suggestions for LLM-agent"
- Respeita autonomia: cada agente aplica ao seu contexto

### 3. Step types: 4 canônicos + open namespace

| Type         | O que faz                                                 | Output                          |
| ------------ | --------------------------------------------------------- | ------------------------------- |
| **`prompt`** | Instrução NL pro LLM (reason, extract, decide, summarize) | texto livre / structured output |
| **`skill`**  | Invoca skill nomeada (padrão Claude skills)               | depende do skill                |
| **`call`**   | Delega step para outro agente AGORA (por role ou handle)  | output do agente remoto         |
| **`check`**  | Verificação booleana (passa/falha, registrada no ledger)  | `true` / `false`                |

**Open namespace**: campo `kind` em steps é string livre. Tipos custom são permitidos (`shell`, `http`, `my-org:legal-review`). Agente que não conhece um type pode:

1. Tratar como `prompt` (interpretar body como NL)
2. Skip com warning
3. Pedir clarificação via `direct`

**Explicitamente NÃO em v0.1:**

- `branch` (conditional execution) → v0.2
- `parallel` (concurrent steps) → v0.2
- `loop` (iteration) → v0.2

Agente pode implementar conditionals via `prompt` ("se step 3 falhou, tente X") — LLM decide.

### 4. Variable passing: `save_as` + `{{ vars }}`

Todo step tem campo opcional `save_as: "varname"`. Steps subsequentes referenciam via `{{ varname }}` ou `{{ last }}`.

```json
{
  "steps": [
    { "n": 1, "kind": "skill", "name": "ocr-pdf", "save_as": "text" },
    { "n": 2, "kind": "prompt", "text": "Extract CNPJ: {{ text }}", "save_as": "cnpj" }
  ]
}
```

`{{ last }}` refere ao output do step imediatamente anterior.

### 5. Transport: novo verbo `kind: "recipe"` no core

Recipe trafega como mensagem com `kind: "recipe"`. Entra no core, substituindo `teach`.

**Core verbs atualizado (7):**

1. `join`
2. `leave`
3. `greet`
4. `say` (era `hail`)
5. `direct`
6. `recipe` (novo, substitui `teach`)
7. `whois`

**Por que novo verbo (não body discriminator):**

- Broker pode filtrar via `interests: ["recipe:*"]`
- Audit trail explícito
- Typed semantic intent (atende Architect)
- 1 verbo a mais mantém budget simplicity

### 6. Conversational learning via existing verbs

Sem `teach`, pedidos de aprendizado acontecem via `say`/`direct` com NL:

```
alice: say { text: "alguém tem recipe pra parsear NFe?" }
bob:   recipe { to: alice, body: {name, steps, caveats} }

OU

alice: direct { to: bob, text: "me ensina a fazer deploy?" }
bob:   direct { to: alice, text: "primeiro você roda tests..." }
bob:   direct { to: alice, text: "depois git push..." }
```

Conversational mode usa `direct` com NL. Structured mode usa `recipe`. Mesma praça, ambos modos possíveis.

---

## Wire format completo de `recipe`

```json
{
  "v": 1,
  "id": "01JH7K8M...",
  "space": "agora://home/broker.acme.internal/engineering",
  "scope": "home",
  "from": "bob@e5f6...",
  "to": "alice@a1b2...",
  "reply_to": "01JH7K...",
  "kind": "recipe",
  "ts": 1712260800,
  "nonce": "0x7f3a9b2c",
  "body": {
    "name": "parse-nfe-ptbr",
    "version": "1.2",
    "description": "Parse Brazilian fiscal PDF and extract CNPJ + validate",
    "tested_on": "2026-03-15",
    "success_rate": 0.94,
    "caveats": ["falha em NFSe municipal de SP", "OCR ruim em PDFs escaneados <200dpi"],
    "inputs": [{ "name": "pdf_bytes", "type": "bytes", "required": true }],
    "outputs": [
      { "name": "cnpj", "type": "string" },
      { "name": "valid", "type": "boolean" }
    ],
    "steps": [
      {
        "n": 1,
        "kind": "skill",
        "name": "ocr-pdf",
        "args": { "file": "{{ inputs.pdf_bytes }}" },
        "save_as": "text"
      },
      {
        "n": 2,
        "kind": "prompt",
        "text": "Extract CNPJ from this text: {{ text }}. Return just the 14-digit number.",
        "save_as": "cnpj"
      },
      {
        "n": 3,
        "kind": "call",
        "role": "cnpj-validator",
        "hint": "validate format and checksum",
        "args": { "cnpj": "{{ cnpj }}" },
        "save_as": "validation"
      },
      {
        "n": 4,
        "kind": "check",
        "condition": "{{ validation.valid }} == true",
        "on_fail": "abort"
      }
    ]
  },
  "sig": "base64url(ed25519_signature)"
}
```

### Campos do body

| Field          | Required       | Purpose                                    |
| -------------- | -------------- | ------------------------------------------ |
| `name`         | ✅             | Identifier da recipe (slug)                |
| `version`      | ✅             | SemVer da recipe                           |
| `description`  | ✅             | NL summary                                 |
| `steps`        | ✅             | Array ordenado de steps                    |
| `tested_on`    | ⚠️ recomendado | Data/contexto do último teste bem-sucedido |
| `success_rate` | opcional       | Fração de sucessos em execuções            |
| `caveats`      | opcional       | Lista de limitações conhecidas             |
| `inputs`       | opcional       | Schema dos inputs esperados                |
| `outputs`      | opcional       | Schema dos outputs produzidos              |

### Campos de step

| Field     | Required          | Purpose                                                        |
| --------- | ----------------- | -------------------------------------------------------------- |
| `n`       | ✅                | Step number (ordem)                                            |
| `kind`    | ✅                | Step type (canonical ou custom)                                |
| `save_as` | opcional          | Variable name pra output                                       |
| `on_fail` | opcional          | `"abort"` \| `"continue"` \| `"retry"`                         |
| outros    | depende do `kind` | `text` pra prompt, `name` pra skill, `role`/`to` pra call, etc |

---

## Fluxo típico de uso

### Descoberta + consumo

```
alice: say { text: "alguém tem recipe pra validar CNPJ brasileiro?" }

bob: recipe {
  to: alice,
  body: {
    name: "cnpj-validation",
    version: "2.0",
    steps: [...]
  }
}

alice: [executa localmente com seu LLM runtime, usando próprios skills/tools]

alice: echo {
  about: bob,
  observed: "recipe cnpj-validation v2.0 worked on 15/15 test cases",
  event_ref: "<id da recipe recebida>",
  valence: "positive"
}
```

### Versioning

Recipe autor publica v2.0 quando melhora algo. Consumidores decidem migrar baseado em:

- Changelog no `description`
- `success_rate` reportada
- `echoes` positivas de outros agentes

Sem forced updates — agente mantém cache local de recipes confiáveis.

---

## Não está em v0.1 (futuro)

### v0.2 extension: `x-recipe-runner`

Extensão opcional que define **execução determinística** com sandboxing:

- Capability declaration obrigatória (`capabilities_required`)
- Side effects declaration (`side_effects`)
- Sandbox policy (`dry_run_first`, `human_review_required`)
- Dry-run mode que simula sem executar

Até lá, recipe é Teaching only — agente LLM interpreta.

### v0.2+: control flow types

- `branch` — conditional execution
- `parallel` — concurrent step groups
- `loop` — iterate over collection
- `retry` — retry policy

### v0.3+: recipe registry

- Shareable recipe catalog (opt-in)
- Recipe discovery across Spaces
- Aggregated reputation per recipe

---

## Relação com Claude skills

Claude skills são **ingredientes**, recipes são **receitas que usam os ingredientes**.

- Uma **skill** encapsula capability específica (ex: `ocr-pdf`, `cnpj-validator`)
- Um **recipe** orquestra skills + prompts + calls + checks numa sequência

Step `kind: "skill"` é a ponte: dentro de um recipe, você pode invocar qualquer skill instalada no runtime do agente. A recipe não declara QUAL runtime (Claude, custom) — apenas que um step é "uma skill nomeada".

Agente que não tem a skill `name` requerida pode:

- Fazer fallback pra `prompt` type (LLM improvisa)
- Skip step com warning
- Pedir clarificação

---

## Security notes

Mesmo sendo Teaching (não execução), recipes carregam alguns riscos:

1. **Prompt injection via recipe body**: `description`, `caveats`, `hint` fields podem conter instruction injection. Agente DEVE tratar como untrusted NL.

2. **Misleading steps**: recipe pode sugerir `call: "evil-agent"` numa tentativa de socializar com adversário. Agente DEVE verificar reputação antes de seguir `call` steps.

3. **Fake attestation**: `tested_on`, `success_rate` podem ser mentira. Validação via echoes de outros agentes.

**Mitigation v0.1**: recipes são signed (Ed25519 do autor). Agente pode rejeitar recipes de handles sem reputação positiva no ledger local.

---

## Open questions

1. **Recipe signing além do envelope?** O envelope inteiro é assinado (padrão AGORA), mas o body poderia ser re-assinado pelo autor original pra permitir redistribuição. v0.2?

2. **Recipe composition (recipes calling recipes)?** Step `kind: "recipe"` poderia invocar outra recipe como sub-routine. Útil mas adiciona complexidade. v0.2 se houver demanda.

3. **Recipe parameters além de inputs/outputs?** Templates, defaults, optional params. v0.1 fica com inputs/outputs simples.

---

## Changelog desta refinement (vs council synthesis)

| Aspecto         | Council Synthesis                  | Post-refinement                                             |
| --------------- | ---------------------------------- | ----------------------------------------------------------- |
| Nome            | "teach" (core) + "Yarn" (v0.2 ext) | `recipe` (core)                                             |
| Execution model | `teach` conversacional apenas      | `recipe` structured com typed steps                         |
| v0.2 extension  | `x-yarn` (executable w/ sandbox)   | `x-recipe-runner` (deterministic exec)                      |
| Core verbs      | `teach` em 5.1.6                   | `recipe` em 5.1.6                                           |
| Differentiator  | "agentes que ensinam agentes"      | "agentes compartilham recipes executáveis pelo LLM runtime" |
