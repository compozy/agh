# Session Chat Production Hardening

## Summary

- Corrigir o chat de sessão separando histórico durável do transcript e a cauda live da rodada atual. O transcript vira a fonte de verdade do histórico; o AI SDK fica responsável só pelo fluxo live da rodada em andamento.
- Remover o seletor de `skills`, adicionar feedback consistente de `thinking/loading`, corrigir a renderização live de tool calls, e introduzir um `Clear conversation` real que zera contexto no backend sem trocar o `session_id`.
- Persistir as 6 análises paralelas coletadas em `~/.codex/analysis/session-chat-ux/analysis_<name>.md`.

## Implementation Changes

- Frontend session state (`web/src/hooks/routes/use-session-page.ts`, `web/src/systems/session/hooks/use-session-chat.ts`, `web/src/systems/session/components/*`):
- Parar de tratar `messages` como um snapshot único substituível.
- Manter `transcriptMessages` como histórico durável e uma cauda `liveMessages` separada para a rodada atual; a UI renderiza `history + liveTail`.
- No `onFinish`, invalidar o transcript, promover o transcript fresco para `historyMessages` e limpar a live tail.
- Materializar os `message.parts` do AI SDK em linhas AGH ordenadas: segmentos de assistant text/reasoning, `tool_call`, `tool_result` e estados pendentes de ferramenta.
- Continuar usando `data-agh-permission` apenas para approval UI; tool rendering não pode depender de refresh nem de replay tardio.
- Feedback visual e controles:
- Inserir o prompt do usuário imediatamente na live tail.
- Mostrar um estado explícito de `Thinking...` no gap entre `submitted` e o primeiro token/primeira tool, mantendo o `ThinkingBlock` para reasoning parcial.
- Exibir tool cards assim que `tool-input-available` aparecer, com estado ativo até `output/error`.
- Passar estado pendente de stop/resume/clear para o header; os botões mostram spinner, ficam desabilitados durante a mutação e não aceitam double-click.
- Tornar o composer `inert` enquanto existir permission prompt pendente, em vez de só “disabled”.
- Remover o pill de `skills` ponta a ponta: UI, draft store, payload types, query de skills no page hook e testes relacionados. O seletor de channel permanece.
- Adicionar `Clear conversation` no header com confirmação, optimistic clear local, rollback em erro e disabled state quando não houver mensagens, durante streaming ou enquanto a limpeza estiver pendente.
- Backend reset flow (`internal/api/*`, `internal/session/*`, `internal/store/sessiondb/*`):
- Adicionar `POST /api/sessions/{id}/clear` e a mutation correspondente no web.
- Implementar o clear como hard reset in-place: mesmo `session_id`, mesma rota, contexto novo do agent/runtime, transcript vazio e sem reaproveitar contexto anterior do provider.
- Rejeitar clear com `409` se ainda houver prompt em voo; o frontend já mantém o botão desabilitado nesse estado.
- Reinitializar o runtime da sessão após o clear para que ela volte imediatamente utilizável e `active`, evitando o caminho atual onde event store vazio quebra resume.

## Public APIs / Types

- Adicionar `SessionManager.ClearConversation(ctx, id)` e handlers HTTP/UDS para `POST /api/sessions/{id}/clear`.
- Adicionar adapter/mutation `clearSessionConversation(id)` no web, retornando o estado atualizado da sessão.
- Remover `skillId` de `MessageComposerPayload`, `ComposerDraft` e dos chamadores do composer.
- Introduzir um shape explícito de estado frontend para `historyMessages`, `liveMessages` e `pending controls`, no lugar do atual `messages` único substituído por `setState`.

## Test Plan

- Frontend unit/integration:
- Tool calls aparecem durante o stream sem F5 e continuam corretos depois que o transcript é promovido.
- O histórico continua visível após `stop -> resume -> send new prompt`.
- O gap pós-send mostra `Thinking...`/processing imediatamente.
- Stop/resume/clear mostram loading state e bloqueiam reentrada.
- Permission prompt deixa o composer inert.
- O seletor de `skills` não renderiza mais e nenhum `skillId` é enviado ou persistido em draft.
- O clear abre diálogo, limpa otimisticamente, faz rollback em erro e mantém a mesma sessão/rota após sucesso.
- Backend/integration:
- `POST /api/sessions/{id}/clear` zera transcript/histórico e reseta o contexto do runtime mantendo o mesmo `session_id`.
- O próximo prompt depois do clear não enxerga contexto pré-clear.
- Clear com prompt ativo retorna `409`.
- O SSE atual de prompt continua emitindo text/reasoning/tool lifecycle com a mesma semântica.

## Assumptions / Defaults

- “Clear” significa reset real de histórico persistido e contexto do agent, não wipe só de UI e não criação de nova sessão.
- Depois do clear, a sessão volta `active` e pronta para uso imediato, mesmo que antes estivesse `stopped`.
- Streams resumíveis do AI SDK continuam desativados; AGH mantém cancelamento explícito.
- A correção fica concentrada no pipeline de sessão e no novo endpoint de clear; não haverá mudança de roteamento global nem hack de refresh para mascarar estado.
