# RFC: Definições de agente autocontidas com skills e memória com escopo

- **Status:** Rascunho
- **Autores:** AGH Core Team
- **Criado:** 2026-04-06
- **Relaciona-se a:** AGENTS.md (agents.md), AgentSkills Specification (agentskills.io), MCP (modelcontextprotocol.io), A2A (a2a-protocol.org)

---

## Resumo

O ecossistema de agentes de IA convergiu para padrões de instruções de projeto (AGENTS.md), instruções de fluxo reutilizáveis (AgentSkills/SKILL.md) e integração de ferramentas (MCP). O que ainda falta é **o próprio agente** — não há um padrão para definir identidade, capacidades, permissões, conjunto de skills, memória e ciclo de vida do agente como uma unidade única e portável.

Hoje a definição de um agente está espalhada: o prompt em um arquivo, as skills em um diretório global, as memórias em outro armazenamento global, os servidores MCP na configuração da plataforma. Mover um agente entre máquinas, projetos ou equipes significa remontar essas peças manualmente.

Esta RFC propõe um **formato de definição de agente autocontido** em que cada agente é um diretório contendo um arquivo AGENT.md (frontmatter YAML + prompt Markdown), um subdiretório `skills/` com skills específicas do agente e um subdiretório `memory/` com contexto persistente com escopo no agente. O diretório do agente é a unidade de portabilidade — copie o diretório e o agente funciona.

---

## 1. Declaração do problema

### 1.1 A camada ausente: definição de agente

O panorama atual de padrões cobre três camadas:

```
┌─────────────────────────────────────┐
│  Instruções de projeto              │  AGENTS.md, CLAUDE.md, .cursorrules
│  "Como trabalhar NESTE codebase"    │  (escopo de projeto, sempre carregado)
├─────────────────────────────────────┤
│  Skills reutilizáveis               │  SKILL.md (spec AgentSkills)
│  "Como fazer uma tarefa específica"  │  (portátil, carregado sob demanda)
├─────────────────────────────────────┤
│  Integração de ferramentas          │  Servidores MCP
│  "Como conectar a ferramentas externas" │  (nível de protocolo, cliente-servidor)
├─────────────────────────────────────┤
│  Definição de agente                │  ??? (sem padrão)
│  "O que É este agente"              │
└─────────────────────────────────────┘
```

O AGENTS.md informa os agentes sobre o projeto. O SKILL.md ensina como fazer coisas. O MCP dá acesso a ferramentas. Mas nada define **o próprio agente**: qual modelo usa, a quais ferramentas tem acesso, sob quais permissões opera, quais skills são exclusivas, o que lembra entre sessões.

### 1.2 Agentes não são portáveis

Imagine uma equipe com um agente especializado "code-reviewer":

- Prompt afinado por semanas
- Três skills customizadas para revisão de segurança, performance e padrões da equipe
- Memória das preferências de review ("comentários curtos", "foco em tratamento de erros")
- Servidores MCP para PRs no GitHub e contexto de tickets no Jira

Para compartilhar esse agente com outra pessoa seria preciso:

1. Copiar o arquivo de prompt
2. Copiar as skills para os diretórios global/workspace corretos
3. Exportar e importar memórias
4. Documentar quais servidores MCP configurar
5. Explicar o modelo de permissões

Isso é frágil, propenso a erro e não escala. A identidade do agente fica espalhada pelo sistema de arquivos.

### 1.3 Sem especialização por agente

Em todas as implementações atuais, skills são globais ou com escopo de projeto. Todos os agentes no workspace veem o mesmo pool de skills. Não há como dizer "este agente de debug deve ter skills de análise de log, mas o de code review não". O análogo mais próximo são as regras `.mdc` do Cursor com ativação por glob, que são por padrão de arquivo, não por agente.

Isso importa porque agentes com papéis diferentes precisam de capacidades diferentes. Um revisor focado em segurança não deveria ver skills de deploy. Um redator de documentação não deveria ver skills de migração de banco. Especialização reduz ruído de contexto e melhora a precisão da tarefa — o estudo da ETH Zürich (fevereiro de 2026) encontrou que incluir contexto irrelevante aumentou custos de inferência em mais de 20% enquanto reduzia o sucesso da tarefa.

### 1.4 Memória não é com escopo no agente

Sistemas de memória no ecossistema são:

- **Globais** (MEMORY.md do Claude Code, memórias automáticas do Windsurf): todos os agentes compartilham o mesmo contexto
- **Escopo de sessão** (na maioria das implementações): o contexto morre quando a sessão termina
- **Proprietários** (Mem0, MemOS): específicos do framework, não portáveis

Nenhum suporta memória com escopo no agente — contexto que persiste entre sessões, pertence a um agente específico e viaja com o agente quando ele se move. A memória de causas raiz de um agente de debug é inútil para um agente de documentação, e vice-versa.

### 1.5 Abordagens existentes e lacunas

| Abordagem                  | Definição de agente                                  | Skills do agente        | Memória do agente            | Portabilidade                |
| -------------------------- | ---------------------------------------------------- | ----------------------- | ---------------------------- | ---------------------------- |
| **AGENTS.md**              | Não (só instruções de projeto)                       | Não                     | Não                          | Sim (formato universal)      |
| **CLAUDE.md**              | Não (só instruções de projeto)                       | Via AgentSkills         | Via MEMORY.md (global)       | Não (só Claude)              |
| **Subagentes Claude Code** | Parcial (`.claude/agents/*.md` com frontmatter)      | Só pool global          | Sem memória por agente       | Não (só Claude)              |
| **Plugins Codex**          | Não (plugins são pacotes de skill, não definição)    | Via plugins             | Não                          | Não (só Codex)               |
| **Regras Cursor**          | Não (regras são instruções, não definição de agente) | Não                     | Auto-memórias (não portátil) | Não (só Cursor)              |
| **A2A Agent Cards**        | Parcial (só descoberta de capacidades)               | Não                     | Não                          | Parcial (JSON, sem runtime)  |
| **Esta proposta**          | **Sim** (YAML completo + prompt)                     | **Sim** (escopo agente) | **Sim** (escopo agente)      | **Sim** (diretório = agente) |

---

## 2. Proposta

### 2.1 Agente como diretório

Cada agente é um diretório autocontido:

```
.agents/
  code-reviewer/
    AGENT.md                    # Definição do agente (frontmatter + prompt)
    skills/                     # Skills exclusivas do agente
      review-checklist/
        SKILL.md
      security-patterns/
        SKILL.md
    memory/                     # Memória persistente com escopo no agente
      MEMORY.md                 # Índice de memória
      feedback_style.md
      project_context.md

  debugger/
    AGENT.md
    skills/
      systematic-debug/
        SKILL.md
      log-analysis/
        SKILL.md
    memory/
      MEMORY.md
      debug_patterns.md
```

Diretórios de agente podem ficar em:

- `~/.agh/agents/<name>/` — agentes de nível de usuário (disponíveis em todo lugar)
- `<workspace>/.agents/<name>/` — agentes de nível de projeto (compartilhados via controle de versão)

### 2.2 Formato AGENT.md

```yaml
---
name: code-reviewer
description: Code review focused on security and quality
provider: claude
model: claude-sonnet-4-6
tools:
  - Read
  - Grep
  - Glob
permissions: plan
mcp_servers:
  - name: github
    command: npx
    args: ["@github/mcp-server"]
skills:
  inherit: true
  disabled:
    - agh-session-guide
  extra_sources:
    - ./shared-skills/
memory:
  inherit: true
  scope: agent
  auto_consolidate: true
---

You are a senior code reviewer. Your focus is:

1. Security (OWASP top 10)
2. Code quality (readability, maintainability)
3. Performance (hot paths, algorithmic complexity)

Use your internal skills to guide the review. Consult your memories
to understand team preferences and project context.
```

### 2.3 Esquema do frontmatter

**Campos principais** (AgentDef existente, inalterados):

| Campo         | Tipo        | Obrigatório | Descrição                                                   |
| ------------- | ----------- | ----------- | ----------------------------------------------------------- |
| `name`        | string      | Sim         | Identificador do agente (minúsculas, alfanumérico + hífens) |
| `description` | string      | Não         | Descrição em uma linha para descoberta e seleção            |
| `provider`    | string      | Sim         | Provedor de IA (claude, openai, gemini, etc.)               |
| `model`       | string      | Não         | Identificador do modelo (padrão do provedor se omitido)     |
| `command`     | string      | Não         | Comando customizado para subir o subprocesso do agente      |
| `tools`       | []string    | Não         | Lista permitida de ferramentas                              |
| `permissions` | string      | Não         | Modo de permissão (plan, auto, default)                     |
| `mcp_servers` | []MCPServer | Não         | Declarações de servidores MCP                               |

**Novos campos — configuração de skills:**

| Campo                  | Tipo     | Padrão | Descrição                                     |
| ---------------------- | -------- | ------ | --------------------------------------------- |
| `skills.inherit`       | bool     | true   | Herdar skills dos pools global e de workspace |
| `skills.disabled`      | []string | []     | Skills nomeadas dos pools herdados a excluir  |
| `skills.extra_sources` | []string | []     | Diretórios adicionais para varrer skills      |

**Novos campos — configuração de memória:**

| Campo                     | Tipo   | Padrão  | Descrição                                                                      |
| ------------------------- | ------ | ------- | ------------------------------------------------------------------------------ |
| `memory.inherit`          | bool   | true    | Herdar memórias das stores global e de workspace                               |
| `memory.scope`            | string | "agent" | Escopo de escrita padrão para novas memórias: `agent`, `workspace` ou `global` |
| `memory.auto_consolidate` | bool   | true    | Consolidar automaticamente memórias do agente ao fim da sessão                 |

### 2.4 Hierarquia de resolução de skills

Quando o daemon monta o prompt para uma sessão com um agente específico, as skills são resolvidas em cinco camadas:

```
1. Skills empacotadas (go:embed)                         — base, imutável
2. Skills globais (~/.agh/skills/ + ~/.agents/skills/)   — nível de usuário
3. Skills de workspace (.agents/skills/ + .agh/skills/)  — nível de projeto
4. Fontes extras (caminhos skills.extra_sources)         — declaradas pelo agente
5. Skills do agente (.agents/<name>/skills/)             — específicas do agente, maior precedência
```

**Regras de sobrescrita:**

- Colisão de mesmo nome: vence a maior precedência (skill do agente sobrescreve a global)
- `skills.disabled` remove skills nomeadas das camadas herdadas antes do merge
- `skills.inherit: false` pula as camadas 1–3, usa só específicas do agente e fontes extras
- Trilha de auditoria de sobrescritas registra todos os shadows para debug

**Exemplo:** O agente `code-reviewer` tem `skills.disabled: [agh-session-guide]` e uma skill local `review-checklist`. O conjunto efetivo é: todas as skills global/workspace menos `agh-session-guide`, mais `review-checklist` do agente (que sobrescreveria qualquer skill global com o mesmo nome).

### 2.5 Hierarquia de resolução de memória

```
1. Memória global (~/.agh/memory/)                 — contexto do usuário
2. Memória de workspace (.agh/memory/)             — contexto do projeto
3. Memória do agente (.agents/<name>/memory/)      — específica do agente, maior precedência
```

**Regras de merge:**

- Todos os níveis são carregados e concatenados no prompt (o mais específico por último)
- Conflito de nome: memória do agente faz sombra à memória de workspace/global com o mesmo nome de arquivo
- `memory.inherit: false` pula as camadas 1–2, carrega só memórias com escopo no agente
- Escritas de memória usam por padrão o escopo declarado em `memory.scope`

### 2.6 Ciclo de vida da memória do agente

**Escritas.** Quando o agente é instruído a salvar uma memória:

1. O agente gera o conteúdo (via chamada de ferramenta ou instrução embutida no prompt)
2. O daemon recebe o pedido de escrita com `scope` alvo (agent | workspace | global)
3. O escopo padrão vem de `memory.scope` no AGENT.md
4. Arquivo escrito em `.agents/<name>/memory/<file>.md` (para escopo agent)
5. Arquivo índice `.agents/<name>/memory/MEMORY.md` atualizado automaticamente

**Consolidação automática.** Se `memory.auto_consolidate: true`:

1. Ao fim da sessão, o daemon analisa memórias acumuladas do agente
2. Identifica redundâncias, informação desatualizada, contradições
3. Gera versão consolidada (merge de duplicatas, remoção de entradas obsoletas)
4. Atualiza arquivos de memória e índice
5. Registra evento de consolidação na observabilidade para auditoria

**Por que consolidação automática?** Sem isso, memórias do agente crescem sem limite. O estudo da ETH Zürich mostrou que inchaço de arquivos de contexto aumenta custos de inferência em mais de 20% enquanto piora o sucesso da tarefa. Uma passagem dedicada de consolidação mantém o armazenamento enxuto e relevante.

### 2.7 Portabilidade

O diretório do agente é a unidade atômica de portabilidade:

```bash
# Copiar o agente completo (definição + skills + memória)
cp -r .agents/code-reviewer/ /other/project/.agents/

# Compartilhar via controle de versão
git add .agents/code-reviewer/
git commit -m "feat: add code-reviewer agent"

# Exportar/importar (integração futura com marketplace)
agh agent export code-reviewer > code-reviewer.tar.gz
agh agent import code-reviewer.tar.gz
```

Sem dependências externas para perseguir. Sem estado global a replicar. O diretório contém tudo o que o agente precisa. Skills dentro do diretório do agente seguem o formato SKILL.md padrão AgentSkills — funcionam em qualquer plataforma compatível se extraídas.

### 2.8 CLI

```bash
# Gestão de agentes
agh agent list                            # Listar agentes disponíveis (usuário + workspace)
agh agent info <name>                     # Mostrar AGENT.md + skills + memórias
agh agent create <name>                   # Gerar .agents/<name>/ com estrutura completa

# Skills com escopo no agente
agh agent skills <name>                   # Listar skills efetivas (merged)
agh agent skills <name> --local-only      # Listar só skills internas ao agente

# Memória com escopo no agente
agh agent memory <name>                   # Listar memórias do agente
agh agent memory <name> --consolidate     # Forçar consolidação manual
```

---

## 3. Comparação com abordagens existentes

### 3.1 vs. AGENTS.md

O AGENTS.md define instruções de projeto — "como trabalhar neste codebase". É amplamente adotado (60.000+ projetos) e agnóstico de ferramenta. Mas descreve o _projeto_, não o _agente_. Dois agentes no mesmo projeto leem o mesmo AGENTS.md, mesmo com papéis completamente diferentes.

Esta proposta é complementar: AGENTS.md fornece contexto de projeto que todos os agentes herdam. AGENT.md define o próprio agente — capacidades, especialização e estado.

### 3.2 vs. subagentes Claude Code

O Claude Code suporta subagentes customizados em `.claude/agents/*.md` com frontmatter YAML (ferramentas, modelo, permissões). É o precedente existente mais próximo. Porém:

- Skills de subagentes vêm do pool global — sem skills com escopo no agente
- Sem memória com escopo no agente — todos os subagentes compartilham o mesmo MEMORY.md
- Sem controle de herança de skills (não dá para desabilitar skills específicas por agente)
- Sem consolidação de memória
- Específico do Claude — não funciona com Codex, Gemini CLI, etc.

Esta proposta generaliza o padrão de subagente do Claude Code com recursos com escopo no agente e formato independente de provedor.

### 3.3 vs. A2A Agent Cards

O protocolo A2A define Agent Cards — documentos JSON descrevendo capacidades do agente para descoberta. Os Agent Cards são para comunicação entre agentes ("o que você pode fazer?"), não para configuração do agente ("como você deve se comportar?"). Não há conceito de skills, memória ou conteúdo de prompt.

Esta proposta trata de outra camada: Agent Cards A2A poderiam ser _gerados a partir_ de uma definição AGENT.md, fornecendo metadados de descoberta enquanto o AGENT.md fornece a configuração de runtime.

### 3.4 vs. plugins Codex

Plugins Codex empacotam skills + servidores MCP + integrações de app em unidades instaláveis. Mas plugins são _pacotes de capacidade_, não definições de agente. Um plugin diz "aqui estão ferramentas para trabalho com banco". Um AGENT.md diz "aqui está um agente especialista em banco com estas ferramentas, estas skills, estas memórias e este prompt".

---

## 4. Integração na arquitetura

### 4.1 Mudanças nos componentes existentes

| Componente                              | Mudança                                                                                     |
| --------------------------------------- | ------------------------------------------------------------------------------------------- |
| `internal/config/agent.go`              | Estender `AgentDef` com structs `SkillsConfig` e `MemoryConfig`                             |
| `internal/skills/registry.go`           | Adicionar método `ForAgent()` — resolve conjunto merged de skills para um agente específico |
| `internal/memory/assembler.go`          | Adicionar método `ForAgent()` — monta contexto de memória com escopo no agente              |
| `internal/daemon/composed_assembler.go` | Usar `ForAgent()` em vez de `ForWorkspace()` quando o agente tiver recursos com escopo      |
| `internal/cli/agent.go`                 | Novos comandos: `agent list`, `agent info`, `agent create`, `agent skills`, `agent memory`  |
| `internal/daemon/daemon.go`             | Sequência de boot carrega diretórios de agente e registra file watchers                     |

**Sem novos pacotes.** Todas as mudanças são extensões aos pacotes existentes, mantendo a arquitetura plana do projeto. Os métodos `ForAgent()` seguem o padrão estabelecido `ForWorkspace()`.

### 4.2 Registry ForAgent

```go
func (r *Registry) ForAgent(
    ctx context.Context,
    workspace string,
    agentDef *config.AgentDef,
) ([]*Skill, error) {
    // 1. Se inherit=true: coletar skills globais (bundled + user + marketplace)
    // 2. Se inherit=true: coletar skills de workspace
    // 3. Coletar skills de extra_sources
    // 4. Coletar .agents/<name>/skills/
    // 5. Aplicar agentDef.Skills.Disabled (remover skills nomeadas)
    // 6. Resolver sobrescritas por precedência (agent > extra > workspace > global)
    // 7. Retornar lista final merged
}
```

### 4.3 Memory ForAgent

```go
func (a *Assembler) ForAgent(
    ctx context.Context,
    workspace string,
    agentName string,
    inherit bool,
) (string, error) {
    // 1. Se inherit=true: carregar índice de memória global
    // 2. Se inherit=true: carregar índice de memória de workspace
    // 3. Carregar .agents/<name>/memory/MEMORY.md
    // 4. Aplicar regras de shadow (agent > workspace > global)
    // 5. Retornar contexto concatenado
}
```

---

## 5. Exemplo completo

### Estrutura de diretórios

```
.agents/code-reviewer/
  AGENT.md
  skills/
    review-checklist/
      SKILL.md
    security-patterns/
      SKILL.md
  memory/
    MEMORY.md
    feedback_prefer_short.md
    project_auth_context.md
```

### skills/review-checklist/SKILL.md

Formato AgentSkills padrão — funciona em qualquer plataforma compatível:

```yaml
---
name: review-checklist
description: Team's standard code review checklist. Use on every PR review.
version: 1.0.0
---

## Review Checklist

- [ ] Error handling: all errors handled with wrapped context
- [ ] Tests: >=80% coverage for new code
- [ ] Security: no SQL injection, XSS, command injection
- [ ] Performance: no N+1 queries, no unnecessary loops
- [ ] Concurrency: correct mutexes, no race conditions
```

### memory/feedback_prefer_short.md

Formato de memória padrão com frontmatter:

```yaml
---
name: prefer-short-comments
description: Team prefers short, direct review comments
type: feedback
---
Review comments should be short and direct (1-2 lines).
Don't explain the problem in detail — just point it out and suggest a fix.

**Why:** Team reported that verbose reviews get ignored.
**How to apply:** In every review comment, limit to 2 lines max.
```

---

## 6. Questões em aberto

1. **Convergência de formato.** O AGENT.md deveria alinhar-se mais ao AGENTS.md (Markdown puro, sem frontmatter) por compatibilidade universal? Ou frontmatter estruturado é essencial para configuração parseável por máquina? Posição atual: frontmatter estruturado é necessário — os campos (provider, model, tools, permissions) são dados estruturados, não prosa.

2. **Portabilidade entre plataformas.** Se outra plataforma adotar o formato AGENT.md, como tratar campos específicos de provedor (ex.: `permissions: plan` do Claude)? Namespace (`claude.permissions`)? Ignorar campos desconhecidos? Usar modelo de capacidades em vez de campos por provedor?

3. **Identidade do agente entre projetos.** Quando o mesmo diretório de agente é copiado para vários projetos, memórias de projetos diferentes deveriam ser merged, mantidas separadas ou namespaced explicitamente? Posição atual: memórias são por diretório, então cópias divergem naturalmente.

4. **Estratégia de consolidação de memória.** Consolidação automática exige heurísticas para redundância e obsolescência. O daemon deveria usar regra simples (dedup, expiração por idade) ou delegar ao LLM do agente para consolidação semântica? Esta é mais precisa mas tem implicação de custo.

5. **Profundidade de herança de skills.** Se o agente A declara `extra_sources: ["../shared-skills/"]` e outro agente B tem skill com o mesmo nome, qual a precedência? Posição atual: local ao agente sempre vence fontes extras, que vencem workspace/global.

6. **Caminho de padronização.** Este formato deveria ser proposto como extensão ao AGENTS.md sob governança AAIF? Ou como spec standalone? A convenção AGENTS.md é intencionalmente mínima — adicionar definições estruturadas de agente pode conflitar com a filosofia de "Markdown simples".
