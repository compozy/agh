# Estabilizar Viewport do Dashboard

## Resumo

- Causa raiz 1: o canvas hoje executa `fitView` sempre que `shapeSignature` muda, então qualquer spawn/kill/workgroup resetta a câmera.
- Causa raiz 2: o pan/zoom vive dentro do componente do canvas; quando a sessão troca e o canvas desmonta, o viewport é perdido.
- Causa raiz 3: eventos só de status ainda disparam rebuild completo do grafo, o que aumenta churn visual e favorece glitch.

## Mudanças de implementação

- Trocar o estado canônico da câmera para um snapshot por sessão em formato `{ centerX, centerY, zoom }`, mantido fora do `TopologyCanvas` e persistido em `localStorage`.
- Restaurar esse snapshot sempre que o canvas montar ou remontar; se a sessão não tiver estado salvo, aplicar `fitView` uma única vez na carga inicial da sessão.
- Remover o auto-fit acoplado a `shapeSignature`; mudanças estruturais continuam recalculando layout, mas não podem mais alterar a câmera do usuário.
- Recalcular `x/y` do viewport a partir de `centerX/centerY/zoom` quando o tamanho do shell mudar, preservando o mesmo world center em resize, collapse da sidebar e remounts.
- Manter `fitAll`, duplo clique no background, presets de zoom e `centerOnAgent` como ações explícitas; todas elas passam a atualizar o snapshot persistido.
- No store de topologia, separar eventos que exigem relayout de eventos só de estado; `agent_state_changed` e equivalentes devem atualizar status/labels/tree sem rodar ELK nem recriar posições.

## APIs / Tipos

- Adicionar um tipo interno `PersistedViewport` com `{ centerX, centerY, zoom }`, indexado por `sessionName`.
- Nenhuma mudança de contrato backend, REST ou WebSocket.
- A interface externa do `TopologyCanvas` permanece a mesma; a persistência fica encapsulada em store/utilitário interno.

## Testes e cenários

- Atualizar o teste do canvas para exigir viewport estável também quando a topologia muda de shape; auto-fit só pode ocorrer na primeira carga sem estado salvo ou por ação explícita.
- Adicionar teste de troca de sessão: entrar na sessão A, mudar zoom/pan, ir para sessão B, voltar para A e recuperar exatamente o viewport anterior.
- Adicionar teste de restore após reload com `localStorage`, incluindo fallback seguro para payload corrompido.
- Adicionar teste de resize/collapse da sidebar preservando o centro visível.
- Adicionar teste do store garantindo que eventos só de status não chamam `computeDashboardGraph`/ELK novamente e não substituem o grafo posicional.

## Assumptions

- Persistência desejada: por sessão e também após reload.
- Auto-fit automático em updates ao vivo deixa de existir; framing automático fica restrito à primeira visita sem estado salvo.
- Se `localStorage` não estiver disponível, o sistema degrada para memória da aba sem reintroduzir reset de zoom.
