# Analysis: MCP Servers

- Veredito: NAO PRONTO

## O que a tela pede

- Lista de MCP servers globais e/ou sobrepostos por workspace.
- Campos visiveis como `name`, `command`, quantidade de `args` e `env`.
- Troca de escopo global vs workspace.
- Acao para criar/editar/remover servidores.

## O que ja existe

- O modelo de configuracao ja tem `Config.MCPServers`.
- Providers e agentes conseguem compor MCP servers por precedencia.
- O runtime ja recebe MCP servers resolvidos ao iniciar sessoes/agentes.

## Gaps para implementar a tela inteira

- Nao existe endpoint HTTP para listar MCP servers globais.
- Nao existe endpoint HTTP para listar overlay de MCP servers por workspace.
- Nao existe endpoint HTTP para criar/editar/remover MCP servers.
- Nao existe endpoint de introspeccao que explique a precedencia efetiva global -> provider -> agent/workspace para a UI.

## Evidencias

- `internal/config/provider.go:18-24` define `MCPServer`.
- `internal/config/config.go:203-220` inclui `MCPServers []MCPServer` no config raiz.
- `internal/config/provider.go:111-170` resolve agentes usando `Config.MCPServers`, `provider.MCPServers` e `agent.MCPServers`.
- `internal/config/provider.go:201-220` implementa `MergeMCPServers`.
- `internal/api/httpapi/routes.go:11-27` nao registra nenhum grupo `/mcp`, `/mcp_servers` ou equivalente.

## Conclusao

- O daemon tem modelo e runtime para MCP servers, mas nao tem API de settings para essa tela.
- Essa tela depende primeiro de endpoints dedicados para catalogo e mutacoes de MCP servers.
