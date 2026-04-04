# PRD: AGH Web Client V1

## Overview

O AGH Web Client V1 é uma SPA (Single-Page Application) web que fornece uma interface visual rica para interação com agentes de coding AI gerenciados pelo daemon AGH. Resolve o problema de que o terminal é limitado para visualizar tool calls, diffs, markdown renderizado, e gerenciar múltiplas sessões simultâneas. O client consome a HTTP/SSE API do daemon e replica o core de conversa do projeto de referência `.resources/harnss`, adaptado para browser.

- **O que resolve**: Experiência visual rica para interagir com agentes AI (Claude, Codex, Gemini, etc.) — streaming de respostas, tool cards colapsáveis, permission management, e navegação entre sessões.
- **Para quem**: Desenvolvedores individuais que rodam o AGH daemon localmente e tech leads de equipes pequenas que precisam interagir com e monitorar agentes de coding.
- **Por que é valioso**: O terminal não consegue renderizar diffs visuais, tool cards interativos, markdown formatado, ou navegar entre múltiplas sessões de forma fluida. Uma interface web moderna aumenta a produtividade e reduz o atrito na interação com agentes.

## Goals

- **Loop de conversa completo**: Usuário consegue criar sessão, enviar mensagem, ver resposta streaming com tool cards, e navegar entre sessões em menos de 3 cliques.
- **Fidelidade visual ao harnss**: Tool cards, streaming text, e permission prompts devem ter qualidade visual e funcional equivalente ao projeto de referência.
- **Latência imperceptível**: Mensagens do usuário devem aparecer instantaneamente; streaming do agente deve iniciar em < 500ms após envio.
- **Zero configuration**: O client conecta automaticamente ao daemon local (localhost:2123) sem configuração manual.
- **Milestone**: V1 funcional end-to-end entregue como uma fase completa antes de iniciar observabilidade ou features avançadas.

## User Stories

### Dev Individual

- Como dev, quero ver todos os meus agentes configurados no sidebar para saber quais ferramentas tenho disponíveis.
- Como dev, quero expandir um agente no sidebar e ver suas sessões ativas para retomar uma conversa anterior.
- Como dev, quero criar uma nova sessão com um agente para iniciar uma nova conversa de coding.
- Como dev, quero enviar uma mensagem e ver a resposta do agente aparecer em tempo real (streaming) para ter feedback imediato.
- Como dev, quero ver tool calls (Read, Write, Edit, Bash) como cards colapsáveis para entender exatamente o que o agente está fazendo no meu código.
- Como dev, quero expandir um tool card para ver o input completo e o resultado da execução.
- Como dev, quero aprovar ou rejeitar pedidos de permissão do agente (allow once/always, reject once/always) para manter controle sobre o que o agente faz.
- Como dev, quero ver quando o agente está "pensando" (reasoning/thinking blocks) para entender seu processo de raciocínio.
- Como dev, quero navegar entre sessões diferentes sem perder o estado da conversa atual.
- Como dev, quero ver o estado de cada sessão (active, stopped, processing) no sidebar para saber o que está rodando.

### Tech Lead

- Como tech lead, quero ver sessões de diferentes agentes organizadas por agente no sidebar para monitorar o que cada agente está fazendo.
- Como tech lead, quero ver o histórico de uma sessão com todas as tool calls e respostas para auditar o trabalho do agente.
- Como tech lead, quero ver indicadores visuais de sessões que precisam de atenção (pending permission) para agir rapidamente.
- Como tech lead, quero retomar sessões paradas para continuar trabalho interrompido.

## Core Features

### F1: Sidebar de Agentes e Sessões

**O que faz**: Painel lateral esquerdo que lista todos os agentes disponíveis (vindos da config do daemon) com suas sessões agrupadas hierarquicamente.

**Comportamento**:
- Cada agente aparece como um item colapsável com ícone e nome
- Ao expandir, mostra lista de sessões do agente ordenadas por última atividade (mais recente primeiro)
- Cada sessão mostra: título (ou ID se sem título), estado visual (badge de cor: active=verde, stopped=cinza, processing=animação), indicador de pending permission (pulsing amber dot)
- Botão "Nova Sessão" por agente para criar sessão
- Clicar em uma sessão abre a conversa na área principal
- Sessão ativa fica destacada visualmente no sidebar

**Por que é importante**: Ponto de entrada principal do usuário — precisa dar overview imediato de todos os agentes e sessões disponíveis.

### F2: Chat View com Streaming

**O que faz**: Área principal de conversa que mostra o histórico de mensagens com streaming em tempo real.

**Comportamento**:
- Mensagens do usuário e do agente exibidas em formato de chat
- Streaming de texto do agente em tempo real (character-by-character ou chunk-by-chunk)
- Markdown renderizado (headings, code blocks com syntax highlighting, listas, links, etc.)
- Thinking/reasoning blocks do agente exibidos de forma distinta (colapsável, visual diferenciado)
- Scroll automático para o fim durante streaming (bottom-lock)
- Scroll manual desativa bottom-lock; botão para voltar ao fim
- Virtualização de mensagens para performance em conversas longas
- Input de mensagem na parte inferior com envio por Enter (Shift+Enter para nova linha)

**Por que é importante**: Core da experiência — onde o usuário passa 90% do tempo.

### F3: Tool Cards Colapsáveis

**O que faz**: Renderização rica de tool calls do agente como cards interativos dentro do fluxo de chat.

**Comportamento** (copiado do harnss):
- Cada tool call aparece como um card com: ícone da tool, nome da tool, descrição resumida em uma linha, estado (executing/success/error)
- Card é colapsável — expandir mostra: input da tool (parâmetros formatados), output/resultado (texto, diff, stdout/stderr)
- Tool cards de arquivo (Read/Write/Edit) mostram: path do arquivo, preview do conteúdo, diff visual para edits
- Tool card de Bash mostra: comando executado, stdout/stderr com formatação
- Auto-expand quando resultado chega; auto-collapse após breve delay (configurável)
- Agrupamento de tool calls consecutivas em um grupo visual
- Indicador visual de tool em execução (spinner)

**Por que é importante**: Diferencial principal sobre o terminal — visualização rica do que o agente está fazendo.

### F4: Permission Prompt

**O que faz**: Interface para o usuário responder a pedidos de permissão do agente quando ele quer executar tools em modos restritivos.

**Comportamento** (copiado do harnss):
- Quando o agente solicita permissão, aparece um prompt inline no chat ou como overlay
- Mostra: nome da tool, input/parâmetros que o agente quer executar
- Opções de resposta: Allow Once, Allow Always (para esta tool), Reject Once, Reject Always
- Indicador visual no sidebar (pulsing amber dot) quando uma sessão tem permission pendente
- Sessão bloqueia novas mensagens até que a permissão seja resolvida
- Timeout visual se a permissão ficar muito tempo sem resposta

**Por que é importante**: Segurança e controle — essencial para agentes em modo deny-all ou approve-reads.

### F5: Gerenciamento de Sessões

**O que faz**: Capacidade de criar, navegar, parar, e retomar sessões.

**Comportamento**:
- Criar nova sessão: seleciona agente, opcionalmente define nome/workspace, sessão inicia automaticamente
- Navegar entre sessões: clicar no sidebar carrega o histórico da sessão e reconecta ao stream se ativa
- Parar sessão: botão de stop que encerra o processo do agente
- Retomar sessão: botão de resume para sessões paradas (se o agente suportar `session/load`)
- Estado da sessão refletido visualmente em tempo real (badge de cor, indicadores)
- Sessão ativa preserva estado ao navegar para outra e vice-versa

**Por que é importante**: Permite trabalho em múltiplos contextos simultaneamente sem perder progresso.

### F6: Status e Saúde do Daemon

**O que faz**: Indicação visual básica de que o daemon está rodando e conectado.

**Comportamento**:
- Indicador de conexão no header ou footer (connected/disconnected/reconnecting)
- Se o daemon estiver offline, mostrar mensagem clara com instrução de como iniciar
- Reconexão automática quando o daemon volta
- Informações básicas: versão do daemon, uptime (opcional, se disponível via health endpoint)

**Por que é importante**: Sem conexão com o daemon, nada funciona — o usuário precisa feedback imediato.

## User Experience

### Jornada do Usuário

1. **Primeiro acesso**: Usuário abre `localhost:3000` no browser. Se o daemon estiver rodando, vê o sidebar com agentes disponíveis. Se não, vê mensagem de "Daemon não encontrado" com instrução para rodar `agh daemon start`.

2. **Criar sessão**: Clica em "Nova Sessão" no agente desejado (ex: Claude). Sessão é criada e abre automaticamente na área de chat. Input de mensagem fica focado.

3. **Conversar**: Digita mensagem e envia. Vê a resposta do agente aparecer em streaming. Tool calls aparecem como cards colapsáveis. Pode expandir para ver detalhes.

4. **Permissão**: Se o agente pede permissão para executar uma tool, o permission prompt aparece. Usuário aprova ou rejeita. Agente continua.

5. **Navegar**: Clica em outra sessão no sidebar. Conversa anterior preserva estado. Nova sessão carrega com histórico. Pode voltar e continuar de onde parou.

6. **Retomar**: Sessão parada aparece com badge cinza. Clica em "Resume" para reconectar e continuar trabalhando.

### Layout Principal

```
┌──────────────────────────────────────────────────┐
│  Header: AGH logo + status do daemon             │
├──────────┬───────────────────────────────────────┤
│ Sidebar  │                                       │
│          │   Chat Area                           │
│ Agentes  │                                       │
│  └ Agent │   ┌───────────────────────────────┐   │
│    └ Ses │   │ Chat Header (session info)    │   │
│    └ Ses │   ├───────────────────────────────┤   │
│  └ Agent │   │                               │   │
│    └ Ses │   │ Messages (virtualized)        │   │
│          │   │  - User message               │   │
│ [+ Nova] │   │  - Agent streaming text       │   │
│          │   │  - Tool cards (colapsáveis)   │   │
│          │   │  - Thinking blocks            │   │
│          │   │  - Permission prompt          │   │
│          │   │                               │   │
│          │   ├───────────────────────────────┤   │
│          │   │ Input composer + Send button  │   │
│          │   └───────────────────────────────┘   │
├──────────┴───────────────────────────────────────┤
│  Footer (opcional): daemon info, conexão         │
└──────────────────────────────────────────────────┘
```

### Considerações de UI/UX

- **Responsividade**: Sidebar colapsável em telas menores (< 768px)
- **Acessibilidade**: Navegação por teclado, labels ARIA, contraste adequado
- **Dark/Light mode**: Suportar ambos temas (OKLCH color system já configurado)
- **Performance**: Virtualização de mensagens para conversas com 1000+ mensagens
- **Feedback visual**: Indicadores de loading, streaming, erro em todos os estados

## High-Level Technical Constraints

- O web client é uma SPA que roda no browser e se conecta ao daemon AGH via HTTP/SSE em `localhost:2123`
- Deve funcionar com a API existente do daemon. Uma modificação no backend é necessária: `POST /api/sessions/:id/approve` precisa ser implementado (atualmente retorna 501) para suportar o fluxo de permissões interativas
- O protocolo de streaming SSE segue o formato `x-vercel-ai-ui-message-stream: v1` já implementado no daemon
- Patterns de UI devem ser adaptados do `.resources/harnss/` (Electron → browser)
- Stack frontend já definida e configurada: React 19, Vite 8, TanStack Router/Query, shadcn/ui (base-nova), Tailwind v4
- Precisa funcionar localmente (local-first) — sem dependência de serviços cloud

## Non-Goals (Out of Scope)

- **Terminal embutido**: Não teremos xterm.js ou terminal no browser na V1
- **File explorer / browser panel**: Sem painel de arquivos ou browser embutido
- **Spaces / Projects**: Sem conceito de workspaces agrupados (feature do harnss Electron)
- **Dashboard de observabilidade**: Sem painéis de métricas, custos, ou eventos globais
- **Edição de configuração**: Sem UI para editar config.toml ou agent definitions
- **Multi-user / auth**: Sem autenticação — o daemon é local-first
- **MCP server management**: Sem UI para gerenciar MCP servers
- **Subagent visualization**: Sem renderização de árvores de subagentes na V1
- **Busca em sessões**: Sem busca full-text em mensagens/sessões
- **Export de sessões**: Sem exportar conversas como markdown/JSON

## Phased Rollout Plan

### MVP (Phase 1) — Web Client V1

**Core features**:
- F1: Sidebar de Agentes e Sessões
- F2: Chat View com Streaming
- F3: Tool Cards Colapsáveis
- F4: Permission Prompt
- F5: Gerenciamento de Sessões (criar, navegar, parar, retomar)
- F6: Status do Daemon

**Critério de sucesso para Phase 2**:
- Usuário consegue completar um loop de conversa end-to-end (criar sessão → enviar mensagem → ver resposta com tools → aprovar permissão → navegar entre sessões)
- Streaming funciona sem lag perceptível
- Tool cards renderizam corretamente para todas as tools principais (Read, Write, Edit, Bash)
- Funciona em Chrome, Firefox, e Safari

### Phase 2 — Observabilidade e Polish

**Features adicionais**:
- Dashboard de saúde do daemon (sessões ativas, uptime, métricas)
- Visualização de custos por sessão (tokens, custo estimado)
- Busca em sessões e mensagens
- Subagent visualization (nested tool calls)
- Notificações de eventos importantes
- Atalhos de teclado avançados

**Critério de sucesso para Phase 3**:
- Tech leads conseguem monitorar agentes sem usar o CLI
- Custos e usage são rastreáveis por sessão

### Phase 3 — Features Avançadas

**Features adicionais**:
- Workspaces / Projects (agrupar sessões por projeto)
- Configuração de agentes via UI
- MCP server management
- Export de sessões
- Terminal embutido (xterm.js)
- Busca full-text avançada

## Success Metrics

- **Conversa end-to-end funcional**: 100% dos fluxos core (criar, conversar, permissão, navegar) funcionam sem erros
- **Tempo até primeira resposta**: < 500ms entre enviar mensagem e iniciar streaming da resposta
- **Performance de renderização**: Chat com 500+ mensagens mantém 60fps de scroll (via virtualização)
- **Tool card accuracy**: Todas as tools do ACP (Read, Write, Edit, Bash, e genéricas) renderizam corretamente
- **Permission response time**: Permissão resolvida em < 2 cliques
- **Reconexão**: Client reconecta ao daemon em < 3s após desconexão

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Diferenças entre SSE do daemon e IPC do harnss podem causar bugs de streaming | Alto | Mapear todos os event types do daemon para o modelo UIMessage antes de implementar; criar testes de integração com o daemon real |
| Conversas longas podem degradar performance no browser | Médio | Virtualização de mensagens desde o início (TanStack Virtual); lazy loading de tool card content |
| Permission flow via HTTP pode ter latência maior que local IPC | Baixo | Daemon roda localmente (localhost), latência deve ser < 10ms; implementar otimistic UI update |
| Agentes podem emitir events que o harnss trata mas o daemon não expõe | Médio | Auditar todos os event types do daemon vs. harnss antes de implementar; degradar gracefully para tipos desconhecidos |
| Usuário pode não ter o daemon rodando ao abrir o client | Baixo | Tela de "daemon não encontrado" com instruções claras; polling de reconexão automático |

## Architecture Decision Records

- [ADR-001: Harness Web Lite](adrs/adr-001.md) — SPA web focada no loop de conversa, copiando patterns do harnss, sem features Electron-specific

## Open Questions

- **Títulos de sessão**: O daemon gera títulos automaticamente (como o harnss faz com uma query Haiku) ou o usuário define manualmente? Verificar se o endpoint de prompt retorna metadata de título.
- **Workspace selection**: Na criação de sessão, o usuário precisa selecionar um workspace/diretório? V1 pode usar o CWD do daemon como default e aceitar um text input simples para override.
- **Event replay**: Ao abrir uma sessão existente, carregamos todo o histórico de events e reconstruímos a UI, ou o daemon tem um endpoint de "snapshot" do estado atual?
- **Theme persistence**: Onde salvar a preferência de tema (light/dark) — localStorage é suficiente?
- **Handling de reconnect mid-stream**: Se o browser desconecta durante streaming, como retomamos? O Last-Event-ID do SSE é suficiente?
