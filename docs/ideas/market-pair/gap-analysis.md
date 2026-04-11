# Analise de Gaps: AGH vs. Projetos Concorrentes de Agent OS

> Pesquisa realizada em 2026-04-10 cruzando a knowledge base (ai-harness, openclaw, openfang, goclaw, hermes), os arquivos de analise .compozy/tasks/ do AGH, e pesquisa web nos repositorios concorrentes.

## Forcas Arquiteturais do AGH (Diferenciais Defensaveis)

| Forca                              | Por que importa                                                                                                                                                                     |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Modelo de subprocessos ACP**     | AGH orquestra _CLIs de agentes reais_ (Claude Code, Codex, Gemini CLI) como subprocessos via JSON-RPC/stdio — concorrentes reimplementam a logica de agente via wrappers de API LLM |
| **Binario unico em Go**            | Apenas o GoClaw compartilha isso, e o GoClaw e CC BY-NC 4.0 (nao-comercial). AGH e dono dessa faixa comercialmente                                                                  |
| **Isolamento SQLite por sessao**   | Melhor que o in-memory do OpenClaw e o DB compartilhado flat do Hermes. Recuperacao de crash e debugabilidade limpas                                                                |
| **Disciplina de composition root** | `daemon/` como raiz unica, fronteiras verificaveis por CI, sem ciclos de importacao — mais rigoroso que qualquer concorrente                                                        |
| **Design com escopo de workspace** | Alinhado com fluxos de trabalho de desenvolvedores; merge de config overlay bem implementado                                                                                        |

---

## Concorrentes Analisados

### OpenClaw

- **Linguagem**: TypeScript | **Stars**: 354k
- **Modelo**: Assistente pessoal de IA, local-first, hub-and-spoke
- **Destaques**: 22+ adaptadores de canal de mensagens, Canvas UI, ClawHub Registry (skills bundled/managed/workspace), DM pairing para seguranca, 6 modos de deploy (local, Tailscale, SSH, Docker, Fly.io, Nix)
- **Memoria**: Basica (prune/compact), sem embeddings vetoriais ou knowledge graph
- **Extensoes**: 70+ extensoes bundled, MCP, browser automation via DevTools Protocol

### OpenFang

- **Linguagem**: Rust | **Stars**: 16.5k
- **Modelo**: Agent OS autonomo, daemon always-on, microkernel (14 crates, ~137K LoC)
- **Destaques**: 53 tools embutidos, sandbox WASM, 16 camadas de seguranca, 7 "Hands" autonomos (modulos pre-construidos), 40 adaptadores de canal, protocolo P2P (OFP), Merkle hash-chain audit trail
- **Memoria**: SQLite + embeddings vetoriais para recuperacao semantica
- **Extensoes**: 25 templates MCP, cofre de credenciais AES-256-GCM, execucao WASM

### GoClaw

- **Linguagem**: Go | **Stars**: 2.4k | **Licenca**: CC BY-NC 4.0 (nao-comercial)
- **Modelo**: Gateway multi-tenant de agentes, foco enterprise
- **Destaques**: Pipeline de 8 estagios, orquestracao multi-agente com task boards compartilhados, app desktop (Wails v2), OpenTelemetry/Jaeger, pipeline de auto-evolucao
- **Memoria**: 3 camadas (working/episodic/semantic) + pgvector + wiki-links `[[wikilinks]]`, busca hibrida BM25 + semantica
- **Extensoes**: 20+ provedores LLM, domain event bus, MCP

### Hermes Agent (Nous Research)

- **Linguagem**: Python | **Stars**: 51.8k
- **Modelo**: Agente auto-aperfeicoavel, loop de aprendizado embutido
- **Destaques**: Criacao autonoma de skills, 6 backends de terminal (local/Docker/SSH/Daytona/Singularity/Modal), 8 plataformas de mensagens, compativel com agentskills.io, RL training environments
- **Memoria**: Curada pelo agente + FTS5 cross-session + modelagem de usuario via Honcho
- **Extensoes**: 40+ tools embutidos, MCP, Python RPC tool scripting, cron scheduler

---

## Gaps Criticos (Table-Stakes Ausentes)

Funcionalidades que **todos os concorrentes tem** e que o AGH nao tem ou tem apenas parcialmente:

| Gap                                                                    | OpenClaw               | OpenFang                                                       | GoClaw                            | Hermes                                          | Status no AGH                                                                                                                                                                                                                                                                                                                           |
| ---------------------------------------------------------------------- | ---------------------- | -------------------------------------------------------------- | --------------------------------- | ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Memoria multi-camada** (working/episodic/semantic)                   | Basica (prune/compact) | SQLite + vetores                                               | 3 camadas + pgvector + wiki-links | Curada + FTS5 + modelagem de usuario            | Escopo duplo existe, mas sem camada vetorial/semantica, sem recall cross-session FTS5                                                                                                                                                                                                                                                   |
| **Suporte MCP**                                                        | Sim                    | 25 templates                                                   | Sim                               | Sim                                             | **Coberto via ACP** — AGH declara MCP servers (config + merge por precedencia + passthrough) e agentes ACP (Claude Code, Codex, Gemini CLI) ja sao MCP hosts nativos. Decisao arquitetural correta: daemon orquestra, agente executa. MCP host nativo no daemon so faria sentido para operacoes fora de sessao (cron/workflows futuros) |
| **Roteamento multi-LLM** com fallback + rastreamento de custo          | Agnostico de modelo    | 27 provedores, 123+ modelos                                    | 20+ provedores                    | OpenRouter 200+                                 | **Delegado ao ACP** — sem abstracao de provedor embutida, sem agregacao de custo entre agentes                                                                                                                                                                                                                                          |
| **Adaptadores de canal de mensagens** (Telegram, Discord, Slack, etc.) | 22+ plataformas        | 40 canais                                                      | 5 adaptadores                     | 8 plataformas                                   | **Nenhum** — apenas HTTP/SSE + UDS                                                                                                                                                                                                                                                                                                      |
| **Operacao autonoma/agendada** (cron, agentes 24/7)                    | Cron + webhooks        | Design central (Hands rodam 24/7)                              | Cron + heartbeat                  | Cron scheduler                                  | **Ausente** — sem cron, sem execucoes agendadas                                                                                                                                                                                                                                                                                         |
| **Hardening de seguranca**                                             | DM pairing, Tailscale  | 16 sistemas, sandbox WASM, manifests assinados, taint tracking | AES-256-GCM, RBAC, RLS            | Aprovacao de comandos, isolamento por container | **Minimo** — politicas de permissao existem mas sem cofre de credenciais, sem RBAC, sem assinatura de audit log                                                                                                                                                                                                                         |
| **Auto-aperfeicoamento / loop de aprendizado**                         | Nao                    | Construcao de knowledge graph                                  | Pipeline de auto-evolucao         | Criacao autonoma de skills + auto-melhoria      | **Ausente** — consolidacao de memoria existe mas sem loop de aprendizado fechado                                                                                                                                                                                                                                                        |

---

## Gaps Importantes (Diferenciais Competitivos)

Funcionalidades que 2-3 concorrentes possuem e representam vantagem competitiva:

| Gap                                                                                   | Quem tem                                                                      | Status no AGH                                                     |
| ------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| **Engine de knowledge graph** (entidades, relacoes, score de confianca)               | OpenFang, GoClaw                                                              | Ausente — sem camada de grafo                                     |
| **Busca hibrida** (BM25 + vetores semanticos)                                         | GoClaw, OpenFang                                                              | Ausente — sem infra de busca alem de queries SQLite               |
| **Apps nativos desktop/mobile**                                                       | OpenClaw (macOS/iOS/Android), OpenFang (Tauri), GoClaw (Wails)                | Ausente — apenas SPA web                                          |
| **Orquestracao multi-agente** (delegacao de times, quadros de tarefas compartilhados) | GoClaw (coordenacao de times), OpenFang (Hands), Hermes (spawn de subagentes) | Agente unico por sessao — network da Fase 3 endereca parcialmente |
| **Engine de workflow** (passos sequenciais/paralelos/condicionais)                    | OpenFang (completo), GoClaw (domain events)                                   | Ausente — sem primitivas de workflow                              |
| **Execucao de extensoes em sandbox WASM**                                             | OpenFang (sandbox com dual-metering)                                          | Planejado na arquitetura de extensoes P1, nao implementado        |
| **OpenTelemetry / tracing distribuido**                                               | GoClaw (OTLP/Jaeger), OpenFang (audit Merkle)                                 | Ausente — pacote observe rastreia eventos mas sem export OTel     |
| **Suporte ao protocolo A2A** (Google Agent-to-Agent)                                  | OpenFang                                                                      | Fase 3 planejada (baseada em NATS, nao A2A)                       |
| **Hints de prompt caching** para secoes estaveis de contexto                          | Hermes                                                                        | Ausente                                                           |
| **Maquina de estados de aprovacao de tools** (escopos once/session/permanent)         | Hermes, OpenClaw, OpenFang                                                    | Apenas politicas basicas de permissao                             |

---

## O Que Ja Esta Planejado mas Ainda Nao Construido

Da analise em `.compozy/tasks/`:

| Funcionalidade Planejada                                    | Prioridade | Status da Spec                                                   |
| ----------------------------------------------------------- | ---------- | ---------------------------------------------------------------- |
| **Taxonomia de stop reason** (P3)                           | Alta       | Design completo, nao implementado                                |
| **Reparo de sessao no load** (P4)                           | Alta       | Checklist de validacao desenhado                                 |
| **Guard de loop/recursao** (P5)                             | Media      | Budget de iteracao + deteccao de ciclo desenhados                |
| **Runtime Wasm para extensoes** (P1 parcial)                | Media      | Integracao Extism/wazero pendente                                |
| **Comandos CLI de extensao** (`agh extension list/install`) | Media      | Pendente                                                         |
| **SDK TypeScript para extensoes** (`@agh/extension-sdk`)    | Media      | Contrato definido, pacote npm pendente                           |
| **Network de Agentes v0** (Fase 3)                          | Menor      | Techspec completa de 10 tarefas com 760+ linhas, baseada em NATS |

---

## Matriz Comparativa Completa

| Capacidade               | AGH                           | OpenClaw                  | OpenFang                    | GoClaw                                | Hermes                                 |
| ------------------------ | ----------------------------- | ------------------------- | --------------------------- | ------------------------------------- | -------------------------------------- |
| **Linguagem**            | Go                            | TypeScript                | Rust                        | Go                                    | Python                                 |
| **Tamanho do binario**   | Binario unico                 | App Node.js               | ~32MB                       | ~25MB                                 | Pacote Python                          |
| **Protocolo de agente**  | ACP (JSON-RPC/stdio)          | JSON-RPC/WS               | OFP P2P + REST/WS/SSE       | REST + WS                             | Gateway + TUI                          |
| **Gestao de sessao**     | SQLite por sessao             | In-memory + pruning       | SQLite + compaction         | PostgreSQL/SQLite                     | SQLite                                 |
| **Camadas de memoria**   | Fase 2 (parcial)              | Basica (prune/compact)    | SQLite + vetores            | 3 camadas (working/episodic/semantic) | Curada + FTS5 + modelagem de usuario   |
| **Sistema de skills**    | Fase 2 (parcial)              | ClawHub registry          | 60 bundled + SKILL.md       | BM25 + descoberta semantica           | Auto-aperfeicoamento + agentskills.io  |
| **Suporte MCP**          | Nao nativo                    | Sim                       | 25 templates                | Sim                                   | Sim                                    |
| **Suporte A2A**          | Fase 3 (planejado)            | Nao                       | Sim                         | Nao                                   | Nao                                    |
| **Multi-LLM**            | Via agentes ACP               | Agnostico de modelo       | 27 provedores, 123+ modelos | 20+ provedores                        | OpenRouter (200+)                      |
| **Adaptadores de canal** | HTTP/SSE + UDS                | 22+ mensageiros           | 40 canais                   | Telegram/Discord/Slack/etc.           | Telegram/Discord/Slack/WhatsApp/Signal |
| **Web UI**               | React SPA (Vite)              | Control UI + WebChat      | Dashboard                   | React embutido                        | Nao (foco em TUI)                      |
| **CLI**                  | Cobra                         | CLI rico                  | TUI dashboard               | Wizard de onboard                     | TUI com slash commands                 |
| **Observabilidade**      | Gravacao de eventos, metricas | Rastreamento de uso/custo | Merkle audit trail          | OpenTelemetry/Jaeger                  | Modo debug                             |
| **Seguranca**            | Minima (early)                | DM pairing                | 16 sistemas, sandbox WASM   | AES-256-GCM, RBAC                     | Aprovacao de comandos, container       |
| **Multi-agente**         | Agente unico por sessao       | Nos multi-dispositivo     | Hands autonomos             | Orquestracao de times + task boards   | Spawn de subagentes                    |
| **Autonomo/agendado**    | Nao                           | Cron + webhooks           | Design central (24/7 Hands) | Cron + heartbeat                      | Cron scheduler                         |
| **Knowledge graph**      | Nao                           | Nao                       | Nao                         | Sim (pgvector)                        | Nao                                    |
| **Auto-aperfeicoamento** | Nao                           | Nao                       | Nao                         | Pipeline de auto-evolucao             | Auto-melhoria de skills                |
| **App desktop**          | Nao                           | macOS/iOS/Android         | Tauri desktop               | Wails desktop (Lite)                  | Nao                                    |

---

## Recomendacoes Priorizadas

### Tier 1 — Fechar gaps table-stakes (bloqueia credibilidade)

1. **Memoria estruturada com busca semantica** — no minimo recall cross-session via FTS5 (padrao Hermes), idealmente embeddings vetoriais
2. **Execucao agendada/cron de agentes** — operacao autonoma e uma expectativa crescente
3. **Fundamentos de seguranca** — cofre de credenciais (AES-256-GCM), chaves de API com escopo, assinatura de audit log

### Tier 2 — Construir diferenciacao

4. **Mensageria multi-canal** — comecar com 3-5 (Telegram, Discord, Slack) usando uma interface de adaptador
5. **Abstracao de provedor LLM** — agregar rastreamento de custo entre agentes ACP, roteamento com fallback de provedor
6. **Maquina de estados de aprovacao de tools** — escopos once/session/permanent
7. **Export OpenTelemetry** — dos eventos do observe existentes para OTLP/Jaeger

### Tier 3 — Fossos estrategicos

8. **Camada de knowledge graph** — entidades + relacoes sobre a fundacao SQLite existente
9. **Primitivas de workflow** — execucao de passos sequenciais/paralelos/condicionais
10. **Shell de app desktop** — Wails v2 envolvendo a SPA React existente (padrao GoClaw)
11. **Loop de auto-aperfeicoamento** — fechar o ciclo memoria -> skills -> recall

---

## Posicionamento Estrategico

A posicao mais defensavel do AGH e como um **agent OS centrado em desenvolvedores que orquestra CLIs de agentes reais** — nao um wrapper de API LLM. O modelo de subprocessos ACP e genuinamente unico. Os concorrentes todos reconstroem a logica do agente internamente; o AGH compoe agentes existentes (Claude Code, Codex, Gemini CLI) como drivers plugaveis.

O risco principal e que sem memoria estruturada e execucao agendada, o AGH parece incompleto comparado ate com o menor concorrente. MCP ja esta coberto via delegacao ACP (decisao arquitetural correta — daemon orquestra, agente executa). O roadmap da Fase 2 esta corretamente sequenciado — fechar os gaps restantes e as vantagens arquiteturais ficam evidentes.

---

## Padroes Validados (Consenso entre 4+ frameworks)

Padroes que o AGH ja implementa e que sao confirmados como corretos pela analise cruzada:

| Padrao                                      | Status no AGH                                                                | Evidencia                                       |
| ------------------------------------------- | ---------------------------------------------------------------------------- | ----------------------------------------------- |
| Interface uniforme de tools com JSON Schema | Implementado (P2)                                                            | Todos os 6 frameworks                           |
| Skills como markdown com YAML frontmatter   | Existe                                                                       | Claude Code, GoClaw, Hermes, Pi-Mono, OpenClaw  |
| Precedencia de skills em 5 camadas          | Existe                                                                       | Claude Code, GoClaw, OpenClaw, Pi-Mono          |
| Hooks de lifecycle em pontos nomeados       | Implementado (P0)                                                            | Todos os 6 frameworks                           |
| Hooks podem bloquear/modificar              | Implementado (P0)                                                            | Claude Code, Pi-Mono, OpenClaw, Hermes          |
| Fluxo de aprovacao para ops perigosas       | Via passthrough ACP                                                          | Claude Code, Hermes, OpenClaw, OpenFang         |
| Integracao MCP de tools                     | Coberto — config + merge + passthrough ACP; agentes ja sao MCP hosts nativos | Claude Code, GoClaw, Hermes, OpenFang, OpenClaw |
| Descoberta de plugins via manifesto         | Implementado (P1 parcial)                                                    | OpenClaw, Pi-Mono, OpenFang                     |
| Fan-out nao-bloqueante para eventos         | Via padrao Notifier                                                          | Claude Code, GoClaw, OpenClaw                   |
| Namespacing de tools contra colisoes        | Implementado (P2)                                                            | Claude Code, GoClaw, OpenFang                   |
| Progressive disclosure (lazy skill loading) | Implementado (P6)                                                            | Claude Code, Pi-Mono, GoClaw                    |

## Anti-Padroes a Evitar (Aprendidos da Analise)

| Anti-Padrao                                                 | Fonte            | Por que o AGH deve evitar                                                           |
| ----------------------------------------------------------- | ---------------- | ----------------------------------------------------------------------------------- |
| WebSocket RPC entre daemon e agente                         | OpenClaw         | JSON-RPC sobre stdio e mais simples, padrao de subprocesso local e apropriado       |
| Adaptadores de canal in-process em escala                   | OpenClaw         | Complexidade de lifecycle de goroutines Go; isolamento por subprocesso e preferivel |
| Persistencia de sessao em JSONL                             | OpenClaw         | SQLite e estritamente melhor para queries, concorrencia, recuperacao de crash       |
| 70+ extensoes bundled no binario                            | OpenClaw         | Filosofia e core minimo robusto + extensoes instalaveis                             |
| Globais mutaveis no nivel de modulo para registro           | Hermes           | Padrao fragil causando bugs em subagentes; usar contexto de thread                  |
| Core sincrono com bridging async                            | Hermes           | Modelo de concorrencia Go (goroutines) evita impedance mismatch                     |
| DB de sessao flat compartilhado com retries de write jitter | Hermes           | Split do AGH (catalogo global + por-sessao) e arquiteturalmente mais limpo          |
| Efeitos colaterais de import-time via `init()`              | Hermes           | Padrao Go: registro explicito no composition root do daemon                         |
| Monolito kitchen-sink com tudo compilado                    | Hermes, OpenFang | Sistema de extensoes permite separacao de responsabilidades                         |
| Protocolo wire customizado (OFP)                            | OpenFang         | Usar A2A/HTTP padrao para interoperabilidade, nao protocolos proprietarios          |

---

## Fontes

- Knowledge base: `/Users/pedronauck/dev/knowledge/` (topicos: ai-harness, openclaw, openfang, goclaw, hermes, ai-memory, agent-networks, claude-code)
- Analise de extensibilidade: `.compozy/tasks/extensability/analysis.md`
- Analises por projeto: `.compozy/tasks/ext-architecture/analysis_{openclaw,goclaw,openfang,hermes}.md`
- TechSpec Network v0: `.compozy/tasks/agh-network/_techspec.md`
- Repositorios: github.com/openclaw/openclaw, github.com/RightNow-AI/openfang, github.com/nextlevelbuilder/goclaw, github.com/NousResearch/hermes-agent
