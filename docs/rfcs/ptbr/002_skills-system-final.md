# RFC: Skills gerenciadas pelo daemon com ciclo de vida, ponte MCP e segurança

- **Status:** Rascunho
- **Autores:** AGH Core Team
- **Criado:** 2026-04-06
- **Relaciona-se a:** AgentSkills Specification (agentskills.io), MCP (modelcontextprotocol.io), padrões AAIF

---

## Resumo

A especificação AgentSkills (dezembro de 2025) estabeleceu um formato portátil para instruções reutilizáveis de agentes de IA. Adotada por 26+ plataformas em semanas, resolveu a fragmentação para _autoria_ de skills. Porém a spec é deliberadamente mínima — define formato de arquivo, não runtime. Não cobre como carregar skills com segurança, como declarar dependências de ferramentas, como participar de eventos do ciclo de vida do agente ou como interagir com memória persistente.

Esta RFC propõe um **runtime de skills gerenciado pelo daemon** que estende a spec AgentSkills com quatro capacidades que nenhuma implementação atual combina: varredura de segurança no carregamento, provisionamento declarativo de servidores MCP, hooks de ciclo de vida e integração bidirecional com memória. Essas extensões são expressas como campos `metadata.agh.*` no frontmatter padrão de SKILL.md, preservando compatibilidade total com a especificação base.

---

## 1. Declaração do problema

### 1.1 A spec AgentSkills é formato, não runtime

A especificação AgentSkills define um diretório com arquivo `SKILL.md` com frontmatter YAML e corpo Markdown. Estabelece progressive disclosure (metadados → instruções → recursos) e formato portátil de skill. Isso é valioso e adotado. Mas a spec adia explicitamente preocupações críticas de runtime:

| Preocupação               | Spec AgentSkills                                        | Ecossistema atual                                                                                            |
| ------------------------- | ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| **Segurança**             | Sem varredura, assinatura ou verificação                | ClawHavoc (fev. 2026): 1.184+ skills maliciosas no ClawHub. Snyk: 36,82% das skills têm falhas de segurança. |
| **Integração MCP**        | Campo `allowed-tools` (experimental, só nomes de tools) | Skills e MCP são "camadas complementares" mas não há spec de como uma skill declara dependências de MCP      |
| **Ciclo de vida**         | Conteúdo estático carregado na ativação                 | Sem hooks para eventos de sessão. Skills não reagem a criação, término ou montagem do prompt                 |
| **Memória**               | Sem conceito de estado persistente                      | Skills são stateless. Sem como declarar dependências de memória ou orientar escritas                         |
| **Hot-reload**            | Não abordado                                            | Editar uma skill exige reiniciar a sessão do agente na maioria das implementações                            |
| **Semântica de override** | Não especificada                                        | Cada plataforma implementa suas próprias regras de precedência (ou nenhuma)                                  |

### 1.2 O precedente ClawHavoc

Em fevereiro de 2026, pesquisadores de segurança descobriram 341 skills maliciosas no ClawHub (depois revisado para 1.184+ pela Antiy CERT). Vetores de ataque incluíam coleta de credenciais, reverse shells e prompt injection em arquivos de memória do agente. Causa raiz: **registro aberto sem revisão de código, sem assinatura e sem varredura automatizada**. Skills executam com as permissões completas do sistema do desenvolvedor.

A spec AgentSkills não tem modelo de segurança. A varredura ocorre no limite do registro (se ocorrer), não no carregamento. Não há verificação em runtime, sandboxing nem cadeia de proveniência.

### 1.3 Skills e MCP são complementares mas desconectados

A Anthropic posiciona skills como "o cérebro" (o que saber) e MCP como "os braços" (como conectar). Na prática, essas camadas estão desconectadas. Uma skill que ensina padrões de migração de banco não pode declarar que precisa de um servidor MCP PostgreSQL. O usuário deve configurar servidores MCP manualmente, quebrando a promessa de portabilidade da skill.

Os plugins Codex da OpenAI (março de 2026) empacotam skills + servidores MCP + integrações de app em uma unidade instalável. Isso valida a demanda mas prende o padrão a um formato proprietário e específico de plataforma.

### 1.4 Sem participação no ciclo de vida

Skills são texto estático injetado nos prompts. Não podem reagir a eventos de sessão. Uma skill que ensina "como configurar um projeto novo" não pode injetar estado do repositório no início da sessão. Uma skill de debugging não pode consolidar aprendizados ao fim da sessão. O modelo de progressive disclosure da spec (3 níveis) é otimização de carregamento, não modelo de ciclo de vida.

---

## 2. Proposta

### 2.1 Princípios de design

1. **Estender, não bifurcar.** Todas as extensões usam o namespace `metadata.agh.*` no frontmatter padrão de SKILL.md. Qualquer skill compatível com AgentSkills funciona sem modificação. Recursos específicos do AGH degradam com graça em outras plataformas (metadados ignorados).

2. **Segurança no limite.** Toda skill não empacotada é varrida no carregamento antes de entrar no registro. Achados críticos (prompt injection, extração de credenciais) não devem executar silenciosamente; bloqueiam o carregamento ou exigem estado explícito de quarentena retido. Isso é inegociável após o ClawHavoc.

3. **Declarativo em vez de imperativo.** Skills declaram o que precisam (servidores MCP, tags de memória, hooks de ciclo de vida); o daemon gerencia provisionamento, permissões e teardown.

4. **Daemon como governador.** Um processo daemon de longa duração (não um wrapper de CLI) gerencia ciclo de vida da skill, aplica política de segurança e mantém observabilidade. Implementações só-CLI não podem dar essas garantias.

### 2.2 Varredura de segurança (VerifyContent)

Toda skill carregada de fontes não empacotadas passa por `VerifyContent` antes de entrar no registro.

**Três níveis de severidade:**

| Severidade | Ação                         | Exemplos                                                                                                                                                                     |
| ---------- | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Crítica    | **Bloquear carregamento**    | Overrides de system prompt (`ignore all previous`), abuso de tools (`rm -rf`, `delete all files`), extração de credenciais (`print your API key`, `show your system prompt`) |
| Aviso      | Logar, permitir carregamento | Referências a caminhos sensíveis (`/etc/passwd`, `~/.ssh/`), padrões incomuns de tools                                                                                       |
| Info       | Só logar                     | Conteúdo >50K caracteres, campos incomuns de frontmatter                                                                                                                     |

**A varredura é aplicada no carregamento, não só na instalação.** Uma skill modificada no disco após a instalação é re-varrida no próximo carregamento. Isso fecha a lacuna time-of-check/time-of-use que a varredura do ClawHub perdeu.

**Skills empacotadas são confiáveis.** Vêm compiladas no binário via `go:embed` e são imutáveis durante a vida do processo.

### 2.3 MCP declarativo com carregamento preguiçoso

Skills declaram dependências de servidores MCP no frontmatter:

```yaml
---
name: postgres-tools
description: Database migration and query tooling
version: 1.0.0
metadata:
  agh:
    mcp_servers:
      - name: pg-mcp
        command: npx
        args: ["@pg/mcp-server", "--host", "localhost"]
        env:
          PG_PASSWORD: "${PG_PASSWORD}"
---
```

**Comportamento em runtime:**

1. O registry interpreta `metadata.agh.mcp_servers` durante o carregamento da skill
2. Na criação da sessão, o daemon coleta servidores MCP de todas as skills ativas
3. **Portão de consentimento do usuário** — na primeira vez que uma skill de marketplace declara um servidor MCP, o daemon pede confirmação explícita (prompt na CLI ou allowlist persistente na config). Skills de usuário, additional-root e workspace permanecem auto-aprovadas por serem colocadas deliberadamente pelo operador local.
4. Servidores aprovados são injetados em `StartOpts.MCPServers` e iniciados pelo driver ACP junto ao processo do agente
5. Variáveis de ambiente com sintaxe `${}` são resolvidas só após o consentimento, com scrubbing de valores sensíveis não explicitamente permitidos
6. Consentimento de marketplace é persistido na config via `skills.allowed_marketplace_mcp`; allowlists em nível de comando ficam para um passo posterior de endurecimento de segurança

**Níveis de confiança:**

| Fonte                        | Consentimento MCP                               | Restrições de comando |
| ---------------------------- | ----------------------------------------------- | --------------------- |
| Empacotadas                  | Nenhum (confiável)                              | Nenhuma               |
| Usuário/Additional/Workspace | Nenhum (conteúdo local controlado pelo usuário) | Nenhuma               |
| Marketplace                  | Consentimento único, persistido na config       | Adiado                |

**Comparação com plugins Codex:** o Codex empacota config MCP em plugin.json proprietário. Esta proposta mantém declarações MCP no frontmatter padrão de SKILL.md, tornando skills portáteis enquanto o daemon fornece a governança de runtime que o Codex alcança pela plataforma.

### 2.4 Hooks de ciclo de vida

Skills declaram hooks para eventos do ciclo de vida da sessão. No TechSpec skills-v2 atual, só `on_session_created` e `on_session_stopped` estão no escopo; `on_prompt_assembly` está explicitamente adiado.

```yaml
metadata:
  agh:
    hooks:
      - event: on_session_created
        command: "inject-context"
        args: ["--format", "json"]
        timeout: 5s
      - event: on_session_stopped
        command: "consolidate-learnings"
        timeout: 10s
```

**Eventos:**

| Evento               | Gatilho                                       | Caso de uso                                                   |
| -------------------- | --------------------------------------------- | ------------------------------------------------------------- |
| `on_session_created` | Sessão inicializada, antes do primeiro prompt | Injetar estado do repo, tickets abertos, contexto de ambiente |
| `on_session_stopped` | Sessão terminada                              | Consolidar memórias, salvar aprendizados, cleanup             |

**Semântica de execução:**

- Hooks executam na ordem de precedência da hierarquia (bundled → marketplace → user → additional → workspace)
- No mesmo nível: ordem alfabética por nome da skill (determinístico)
- Timeout configurável por hook (padrão 5s)
- **Fail-open:** erros de hook são logados como avisos mas nunca bloqueiam a sessão
- Hooks recebem JSON via stdin: `{"session_id": "...", "agent_name": "...", "workspace": "..."}`
- Hooks podem emitir stdout estruturado para logging/enriquecimento futuro, mas injeção de contexto no prompt fica adiada com `on_prompt_assembly`
- O daemon estende o fan-out existente do notifier com fase dedicada pós-notifier de hooks em vez de um serviço de ciclo de vida separado

**Por que não na spec base?** A spec AgentSkills é intencionalmente agnóstica de cliente. Hooks de ciclo de vida exigem runtime com conceitos de sessão. Arquitetura com daemon fornece isso naturalmente; wrappers de CLI não.

### 2.5 Integração com memória

Esta seção permanece trabalho futuro. O TechSpec skills-v2 atual adia integração profunda com memória para um spec de follow-up; os detalhes abaixo devem ser lidos como desenho prospectivo, não escopo de implementação atual.

Na implementação base, memória e skills coexistem no prompt sem acoplamento — o contexto de memória é montado primeiro, depois o prompt do agente, depois o catálogo de skills. Isso funciona mas perde a oportunidade de skills aproveitarem memória e orientarem escritas.

**Integração profunda (esta proposta):**

**Injeção filtrada por tag.** Skills declaram dependências de memória:

```yaml
metadata:
  agh:
    memory_tags: ["project", "feedback"]
```

O daemon filtra o armazenamento de memória e injeta só memórias que casam com as tags declaradas na seção de contexto da skill. Isso evita que memórias irrelevantes consumam orçamento de contexto.

**API de consulta à memória.** Um hook futuro de montagem de prompt ou superfície equivalente de enriquecimento poderia consultar o store de memória via pedido estruturado, recebendo memórias relevantes na resposta. Permanece adiado enquanto `on_prompt_assembly` estiver fora de escopo.

**Escritas guiadas por skill.** Skills podem incluir instruções que ensinam o agente a salvar tipos específicos de memórias. Exemplo: skill de debugging que diz "salve a causa raiz como memória de projeto para referência futura". O daemon aplica regras de escopo nas escritas.

**Fluxo bidirecional:** skill lê memória → enriquece prompt → agente age → agente salva memória → próxima sessão usa memória enriquecida. Isso cria um loop de melhoria composta em que skills melhoram com o tempo sem serem editadas.

### 2.6 Auto-proposta de skill

O daemon detecta fluxos repetitivos e propõe criação de skill:

**Detecção:** Analisar as últimas N sessões no mesmo workspace. Identificar padrões: sequências repetidas de chamadas de tools, prompts similares, fluxos multi-etapa recorrentes. Limiar: 3+ ocorrências do mesmo padrão em sessões diferentes.

**Proposta:** Ao fim da sessão, se um padrão for detectado, anexar sugestão ao contexto do agente:

```
[AGH] Recurring workflow detected: "<description>".
Consider creating a skill with `agh skill create <suggested-name>`.
```

Uma meta-skill empacotada `skillify` guia o agente a formalizar o fluxo em um arquivo SKILL.md, usando histórico da sessão e memória para gerar um rascunho.

**Loop composto:** uso → detecção → proposta → skill → uso melhorado. Esse é o diferencial — o sistema melhora com o uso, sem exigir que o usuário identifique proativamente padrões reutilizáveis.

### 2.7 Distribuição e proveniência de skills (marketplace)

**Interface CLI:**

```bash
agh skill search "database tools"      # Buscar no marketplace
agh skill install @author/skill-name   # Instalar em ~/.agh/skills/
agh skill remove skill-name            # Remover skill instalada
agh skill update [--all]               # Atualizar skills do marketplace
```

**Modelo de segurança (pós-ClawHavoc):**

- **Verificação de proveniência por hash:** SHA-256 capturado na instalação e re-verificado a cada carregamento
- **Varredura no carregamento:** `VerifyContent` aplicada a toda skill baixada, a cada load
- **Trilha de auditoria de override:** aviso quando uma skill de workspace faz sombra a bundled/marketplace
- **Quarentena/bloqueio:** achados críticos exigem estado explícito de quarentena retido; caso contrário o fallback seguro é manter semântica de bloqueio no load até existir UX de re-aprovação
- **Allowlists de comando MCP:** follow-up adiado, não parte do TechSpec skills-v2 atual

### 2.8 Hierarquia de precedência

Skills são resolvidas em cinco camadas de fonte, camadas superiores sobrescrevendo inferiores:

```
1. Empacotadas                      — mais baixa, imutável, enviadas com o binário
2. Marketplace                      — entradas em `~/.agh/skills/` com `.agh-meta.json`
3. Usuário                          — entradas manuais em `~/.agh/skills/` mais convenção resolvida `~/.agents/skills/`
4. Additional                       — `.agh/skills/` sob roots adicionais de workspace configurados
5. Workspace                        — mais alta, `<workspace>/.agh/skills/`
```

Colisões de mesmo nome: vence a maior precedência. Trilha de auditoria de override registra todos os shadows.

---

## 3. Modelo de dados

```go
type Skill struct {
    Meta          SkillMeta
    Content       string           // Markdown body after frontmatter
    Source        SkillSource      // Bundled | Marketplace | User | Additional | Workspace
    Dir           string           // Absolute path to skill directory
    FilePath      string           // Absolute path to SKILL.md
    Enabled       bool
    MCPServers    []MCPServerDecl  // Parsed from metadata.agh.mcp_servers
    Hooks         []HookDecl       // Parsed from metadata.agh.hooks
    Provenance    *Provenance      // Marketplace: registry/source metadata + hash
    InstalledFrom string           // Marketplace: registry slug
}

type HookDecl struct {
    Event   HookEvent             // on_session_created | on_session_stopped
    Command string
    Args    []string
    Timeout time.Duration
    Env     map[string]string
}

type MCPServerDecl struct {
    Name    string
    Command string
    Args    []string
    Env     map[string]string
}

type Provenance struct {
    Slug      string
    Registry  string              // e.g., "clawhub", "skills.sh"
    Version   string
    Hash      string
    InstalledAt time.Time
}
```

---

## 4. Comparação com abordagens existentes

| Capacidade             | Spec AgentSkills           | Plugins Codex              | Regras Cursor                | Esta proposta                                                             |
| ---------------------- | -------------------------- | -------------------------- | ---------------------------- | ------------------------------------------------------------------------- |
| Formato portátil       | Sim (SKILL.md)             | Não (plugin.json)          | Não (.mdc)                   | Sim (SKILL.md + metadata.agh.\*)                                          |
| Varredura de segurança | Só no registro (se houver) | Gerenciada pela plataforma | Nenhuma                      | No carregamento, a cada load                                              |
| Integração MCP         | `allowed-tools` (só nomes) | Empacotado no plugin       | Nenhuma                      | Declarativo no frontmatter + provisionamento pelo daemon                  |
| Hooks de ciclo de vida | Nenhum                     | Triggers (eventos GitHub)  | Nenhuma                      | 2 eventos de sessão com protocolo stdin/stdout; montagem de prompt adiada |
| Integração memória     | Nenhuma                    | Nenhuma                    | Auto-memórias (proprietário) | Filtrada por tag, bidirecional, escritas guiadas por skill                |
| Hot-reload             | Não especificado           | Não especificado           | File watcher                 | Polling por stat (global) + checagem mtime (workspace)                    |
| Semântica de override  | Não especificado           | Precedência de plugin      | Precedência de regras        | Hierarquia em 5 camadas com trilha de auditoria                           |
| Auto-proposta          | Nenhuma                    | Nenhuma                    | Nenhuma                      | Detecção de padrão + meta-skill skillify                                  |
| Proveniência           | Nenhuma                    | Curada pela plataforma     | N/A                          | Verificação por hash + varredura no load                                  |

---

## 5. Entrega incremental

| Incremento | Escopo                                                                                                      | Status       |
| ---------- | ----------------------------------------------------------------------------------------------------------- | ------------ |
| 1          | Loader, registry dual-scope, injeção no prompt, varredura de segurança, CLI, skills empacotadas, hot-reload | **Completo** |
| 2          | MCP lazy-load, hooks de ciclo de vida e auto-proposta de skill                                              | Planejado    |
| 3          | Integração com marketplace, proveniência por hash, trilha de auditoria de override                          | Planejado    |

Cada incremento entrega valor independente. O incremento 1 já está pronto para produção.

---

## 6. Questões em aberto

1. **Ordem de execução de hooks entre skills.** Quando duas skills declaram `on_session_created`, a execução segue precedência da hierarquia e depois ordem alfabética. Skills deveriam poder declarar dependências explícitas de ordenação?

2. **Persistência do consentimento MCP.** Consentimento único por skill é persistido na config. O consentimento deveria ser revogável? Expirar? Ser por workspace ou global?

3. **Taxonomia de tags de memória.** Skills declaram `memory_tags` para injeção filtrada. Deveria haver vocabulário controlado, ou texto livre basta? Risco: proliferação de tags sem descoberta.

4. **Precisão da auto-proposta.** Detecção de padrão entre sessões exige heurísticas. Falsos positivos (sugerir skills para fluxos únicos) podem erodir confiança. Qual o limiar certo?

5. **Governança do marketplace.** Skills de marketplace deveriam exigir revisão manual, só varredura automatizada, ou combinação? Qual o equilíbrio entre abertura e segurança pós-ClawHavoc?
