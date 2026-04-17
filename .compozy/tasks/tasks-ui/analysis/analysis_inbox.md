# AGH Tasks — Inbox

## Veredito

Não está coberto hoje. A tela do Paper pressupõe uma caixa de entrada de trabalho com estados de leitura, arquivamento e aprovação que não existem no domínio atual de tasks.

## O que já existe parcialmente

- É possível identificar tarefas `blocked` e tarefas/runs com falha usando os status atuais de task e run. Evidência: [internal/task/types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go), [internal/api/contract/tasks.go](/Users/pedronauck/Dev/compozy/agh/internal/api/contract/tasks.go:57)
- `Retry` pode ser modelado com `POST /api/tasks/{id}/runs`.
- `View logs`/abrir detalhe pode ser construído com o detalhe da task e, quando houver `session_id`, com as APIs de sessão.

## Lacunas críticas

- Não existe um read model de inbox com agrupamentos como `My work`, `Approvals`, `Failed runs`, `Blocked tasks` e `Archived`.
- Não existe estado persistido de `read/unread`, `archived`, `dismissed` ou `mark read` para itens de task.
- Não existe workflow de aprovação de task/run. O único `approve` atual é para prompts/permissões de sessão, que é outro domínio. Evidência: [internal/api/udsapi/routes.go](/Users/pedronauck/Dev/compozy/agh/internal/api/udsapi/routes.go:72)
- Não existe `archive all`, `dismiss`, `mark read` ou qualquer mutação equivalente na API de tasks.
- Não existe modelagem de “notificação por unblocking” ou feed de eventos pessoais por owner.

## Conclusão prática

Essa tela precisa de backend novo, não só de composição no frontend. O caminho mais limpo é definir um domínio/read model explícito de inbox de tasks e então expor endpoints para itens, ações de triagem e agrupamentos por tipo.
