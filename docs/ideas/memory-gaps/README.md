# Memory Gaps: o que falta para o AGH ter memória de harness de alto nível

## Objetivo

Este documento consolida o que ainda falta para o AGH atingir um nível alto em termos de memória agentica.

O foco aqui não é apenas "ter arquivos de memória" ou "expor comandos de busca", mas responder a uma pergunta mais importante:

> o AGH já faz o agente aprender, lembrar e reutilizar contexto de forma automática, segura e relevante ao longo do tempo?

A resposta curta hoje é: **a base está melhor, mas ainda não**.

O AGH já tem uma fundação funcional:

- memória persistente em Markdown com frontmatter
- escopo global e escopo por workspace
- `MEMORY.md` como índice de entrada
- consolidação via dream runtime
- catálogo derivado em SQLite FTS5
- busca e reindex
- recall automático antes do `driver.Prompt`

Isso já é mais do que "só uma feature manual". Mas ainda fica abaixo do que harnesses mais maduros fazem, porque a automação está mais forte no **lado de recuperar memória** do que no **lado de formar memória útil de maneira contínua**.

## O padrão de um harness de alto nível em memória

Um harness realmente forte em memória tende a ter quase todos estes atributos ao mesmo tempo:

1. **Formação automática de memória**
   O sistema extrai fatos duráveis, preferências, decisões e contexto de projeto sem depender principalmente de escrita manual.

2. **Recall automático e seletivo**
   O agente recebe contexto recuperado no momento do prompt sem precisar que o usuário peça explicitamente.

3. **Memória em camadas**
   O sistema separa pelo menos memória de sessão/working, memória episódica, memória durável/semântica e, em alguns casos, memória compartilhada de equipe.

4. **Busca de alta qualidade**
   O recall combina texto completo, semântica, ranking de relevância, filtros por escopo e proteção contra contexto irrelevante.

5. **Consolidação e deduplicação**
   O sistema sabe resumir, fundir, podar, marcar staleness, remover contradições e evitar duplicação.

6. **Integração com compaction e loop do agente**
   A memória não vive ao lado do core; ela participa do ciclo de turnos, compaction e recuperação de contexto.

7. **Escopos reais de identidade**
   Além de global/workspace, existem limites mais fortes por usuário, time, tenant ou projeto.

8. **Observabilidade de qualidade**
   Operadores conseguem responder: o que foi lembrado, por que foi lembrado, o que envelheceu, o que está órfão e o que está poluindo contexto.

9. **Loop de aprendizado**
   O sistema fecha o ciclo memória -> consolidação -> skill/procedimento -> recall, em vez de apenas acumular notas.

## Onde o AGH está hoje

Hoje o AGH já cobre partes importantes da fundação:

- memória durável em arquivos Markdown
- taxonomia fechada de tipos
- índice `MEMORY.md`
- dual scope `global` + `workspace`
- catálogo derivado com FTS5
- `search` e `reindex` via daemon/API/CLI
- recall limitado e automático antes do prompt
- dream consolidation em runtime próprio
- estatísticas básicas de saúde da memória

Isso significa que o AGH **já consulta memória automaticamente**. O agente não depende apenas de `agh memory search` manual para usar memória.

Mas o AGH ainda não tem um sistema de memória forte o bastante para ser percebido como um harness de topo, porque vários componentes estruturais continuam ausentes ou incompletos.

## Gaps principais

### 1. Formação automática de memória ainda é fraca

Este é o gap mais importante.

Hoje o AGH melhorou o recall automático, mas ainda não tem um pipeline forte de extração automática por turno ou por resposta final que:

- identifique decisões importantes
- capture preferências do usuário
- detecte fatos recorrentes sobre o projeto
- separe sinal durável de ruído transitório
- escreva isso de forma consistente sem depender principalmente do operador

Na prática, isso significa que o AGH está melhor para **usar memória que já existe** do que para **criar memória boa ao longo do trabalho**.

O salto qualitativo aqui seria:

- extração automática após turnos relevantes
- flush pré-compaction
- fallback extrativo quando o flush com LLM falhar
- regras de saliência para decidir o que merece ser persistido

Sem isso, memória tende a virar um repositório útil apenas para usuários disciplinados.

### 2. Falta memória em camadas

Hoje a estrutura é basicamente:

- memória global
- memória por workspace

Isso é útil, mas ainda é pouco.

Harnesses mais fortes geralmente distinguem pelo menos:

- **working/session memory**: scratchpad ou resumo incremental da conversa atual
- **episodic memory**: resumos de sessões concluídas, eventos e trajetórias
- **durable/semantic memory**: fatos e conhecimento de longo prazo
- **team/shared memory**: conhecimento compartilhado entre usuários ou agentes

No próprio plano da branch atual, `session/working`, `durable`, `episodic` e `team/shared` já ficaram explicitamente para depois.

Sem essas camadas, o sistema mistura coisas com horizontes temporais e objetivos diferentes, o que piora recall e consolidação.

### 3. Recall ainda é principalmente lexical, não semântico

O AGH agora tem uma base melhor com FTS5, o que é um avanço real.

Mas ainda faltam:

- embeddings vetoriais
- busca híbrida BM25 + vetorial
- ranking mais sofisticado
- expansão semântica de consulta
- melhor seleção quando o usuário não usa o mesmo vocabulário do arquivo salvo

Isso importa porque memória agentica falha com frequência exatamente quando o usuário faz perguntas indiretas, vagas ou conceituais.

Texto completo resolve parte do problema.
Busca semântica/híbrida resolve a parte mais difícil.

### 4. Falta recall cross-session mais amplo

O AGH melhorou recall sobre o acervo de memórias persistidas, mas ainda não opera plenamente como um sistema que vasculha o histórico rico de sessões e mensagens concluídas para recuperar contexto útil.

Um harness de alto nível geralmente consegue:

- buscar conversas antigas
- agrupar resultados por sessão
- resumir trechos relevantes antes de injetar
- diferenciar fato durável de histórico episódico

Sem isso, a memória do AGH ainda depende demais de o conhecimento já ter sido promovido para arquivos duráveis.

### 5. Consolidação ainda precisa ficar mais inteligente

O dream runtime já existe e é uma peça boa da arquitetura.

Mas ainda faltam comportamentos mais fortes de consolidação:

- deduplicar fatos equivalentes
- detectar contradições
- rebaixar ou remover memórias envelhecidas
- promover conteúdo de logs para tópicos duráveis
- gerar abstrações curtas para auto-injeção
- separar melhor o que deve continuar em log do que deve virar memória canônica

Sem isso, mesmo uma memória bem abastecida tende a acumular drift.

### 6. Falta proteção mais forte contra memória stale ou errada

Memória útil também precisa ser segura.

Hoje o AGH já ganhou warning de frescor no bloco de recall, mas ainda falta um tratamento mais sistemático para:

- envelhecimento de fatos sensíveis a código e runtime
- proveniência da memória
- score de confiança
- revalidação de memórias muito antigas
- distinção explícita entre observação histórica e verdade atual

Sem isso, o risco é transformar memória em fonte de alucinação persistente.

### 7. Falta escopo real de identidade e colaboração

`global` e `workspace` ajudam, mas ainda não cobrem bem cenários de:

- multi-user
- team/shared
- tenant isolation
- preferências pessoais separadas de contexto compartilhado
- agentes diferentes colaborando sobre o mesmo espaço de memória

Para um harness com ambição mais alta, memória precisa ser modelada com escopo mais rigoroso que apenas "global ou workspace".

### 8. Falta integração mais profunda com compaction e contexto de sessão

Hoje o recall pré-prompt é automático, o que é bom.

Mas ainda falta integração mais profunda entre:

- session summaries
- compaction intermediária
- flush de memória antes da compactação
- recuperação após compactação
- continuidade entre sessões reabertas

Em um harness maduro, memória de sessão e memória durável trabalham juntas; não são dois sistemas quase independentes.

### 9. Falta loop fechado de aprendizado

Este é o gap mais estratégico.

Harnesses mais fortes não apenas lembram fatos. Eles aprendem padrões.

Exemplos do que falta nesse eixo:

- transformar trajetórias repetidas em skills/procedimentos
- promover práticas recorrentes para memória procedural
- detectar correções frequentes do usuário e convertê-las em guidance estável
- gerar insights sobre o que o sistema está aprendendo

Sem esse loop, a memória do AGH continua mais próxima de um bom sistema de recall do que de um sistema de aprendizado agentico.

### 10. Falta observabilidade orientada à qualidade da memória

Hoje já existem estatísticas e eventos básicos, o que é útil.

Mas um sistema de memória de alto nível precisa responder perguntas operacionais mais fortes:

- quais memórias são mais recuperadas
- quais memórias são ignoradas sempre
- quais consultas falham em achar contexto útil
- qual foi a taxa de recall útil vs recall poluente
- quais memórias estão órfãs, redundantes ou contraditórias
- quanto contexto está sendo gasto com memória por turno

Sem essas métricas, a evolução do sistema vira percepção subjetiva.

## O que mais importa corrigir primeiro

Se o objetivo for sair de "memória boa" para "memória de harness de alto nível", a ordem mais racional é:

### Prioridade 1: fortalecer escrita automática

Antes de ampliar busca, é preciso garantir que o sistema produz memória boa de forma contínua.

Isso inclui:

- extração automática por turno ou por parada
- flush pré-compaction
- fallback extrativo
- regras claras de saliência e dedup

### Prioridade 2: introduzir camadas de memória

Separar explicitamente:

- sessão/working
- episódica
- durável/semântica
- compartilhada

Sem essa separação, o resto da evolução fica torto.

### Prioridade 3: melhorar recall

Depois disso, faz sentido subir a qualidade do recall com:

- cross-session real
- busca híbrida
- seleção semântica
- ranking por frescor e relevância

### Prioridade 4: consolidar aprendizado

Só então vale atacar o ciclo mais ambicioso:

- consolidar padrões recorrentes
- gerar memória procedural
- promover para skills
- fechar o loop memória -> skill -> recall

## O que seria um bom critério de "alto nível"

O AGH pode ser considerado realmente forte em memória quando estas afirmações forem verdadeiras:

- o usuário não precisa lembrar de salvar manualmente o que importa na maior parte do tempo
- o agente recebe automaticamente memória relevante antes do prompt com baixa taxa de ruído
- o sistema diferencia bem contexto de sessão, episódios passados e fatos duráveis
- o recall funciona mesmo quando a pergunta não bate lexicalmente com o texto salvo
- memórias antigas não são tratadas como verdade atual sem caveat
- o sistema sabe consolidar, deduplicar e podar sozinho
- existe ao menos um começo de memória compartilhada e escopos fortes de identidade
- o sistema aprende padrões de trabalho, não só fatos isolados

## Resumo executivo

O AGH já saiu do estágio de "memória só manual" e entrou no estágio de **memória durável com recall automático básico**.

O que ainda falta para virar um harness de alto nível em memória não é principalmente mais CLI ou mais endpoints.

O que falta é:

- **escrita automática forte**
- **camadas de memória**
- **recall semântico/híbrido**
- **cross-session de verdade**
- **consolidação inteligente**
- **escopos compartilhados e identidade**
- **proteção forte contra staleness**
- **loop de aprendizado**

Enquanto esses pontos não existirem, o AGH terá uma boa fundação de memória operacional, mas ainda não uma memória agentica de primeira linha.
