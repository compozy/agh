# Integrar a SPA `web/` ao daemon na mesma porta HTTP

## Resumo

- Servir a aplicação React diretamente pelo daemon em `http://<host>:<port>/`, reutilizando o `http.port` já existente no `config.toml`.
- Manter `/api/*` exatamente como está hoje; a mudança é de serving da SPA e fallback de roteamento, não de contrato de API.
- Adotar bundle embedado no binário como caminho oficial. Sem fallback para filesystem, sem placeholder e sem build automático no startup.

## Interfaces públicas e comportamento

- Sem nova configuração: continuam valendo apenas `[http].host` e `[http].port`.
- Nova superfície HTTP visível ao usuário:
  - `GET /` retorna o `index.html` da SPA.
  - `GET /session/:id`, `GET /design-system` e qualquer deep link SPA retornam o mesmo `index.html`.
  - `GET /assets/*` e demais arquivos reais do bundle retornam o asset correspondente.
  - `/api/*` permanece inalterado.
- Nova superfície interna:
  - adicionar `web/embed.go` com `package webassets` exportando `DistFS embed.FS` via `//go:embed all:dist`, preservando `web/dist` como saída canônica do Vite.

## Mudanças de implementação

- Em `internal/httpapi`, adicionar um módulo de static serving que:
  - cria `fs.Sub(webassets.DistFS, "dist")`;
  - registra as rotas `/api/*` primeiro;
  - usa `NoRoute` apenas para requests não-API;
  - para `GET` e `HEAD`, serve asset real quando o arquivo existir;
  - para caminhos sem extensão que não sejam API, devolve `index.html`;
  - para caminhos com extensão que não existirem, devolve `404` em vez de HTML;
  - para métodos diferentes de `GET` e `HEAD`, mantém `404`.
- Servir arquivos com `fs.ReadFile` + `http.ServeContent`, não com `http.FileServer` na raiz, para evitar redirect/clean-path incorreto no shell da SPA.
- Ajustar o pipeline de build:
  - `Build` passa a gerar `web/dist` antes do `go build`;
  - `Verify` passa a rodar `web-lint`, `web-typecheck`, `web-test` e `web-build` antes dos gates Go;
  - se o build web falhar, o build do daemon falha. Sem fallback silencioso.
- Atualizar a documentação para deixar explícito:
  - `agh daemon start` expõe a UI no mesmo host/porta HTTP do daemon;
  - `make web-dev` continua sendo o fluxo de desenvolvimento da UI em `:3000`, separado do modo normal de execução.
- Corrigir o shell público do frontend adicionando um favicon real em `web/public/` e apontando `web/index.html` para ele, para não deixar um `404` garantido na carga inicial.

## Testes e cenários

- Unit tests em `internal/httpapi`:
  - `GET /` responde `200` com `index.html`;
  - `GET /session/teste` responde `200` com `index.html`;
  - `GET /assets/...` responde o asset correto com conteúdo não-HTML;
  - asset inexistente com extensão responde `404`;
  - `GET /api/daemon/status` continua `200` JSON e não cai no fallback SPA;
  - rota `/api/...` inexistente continua `404`, sem retornar `index.html`.
- Integration test do servidor HTTP:
  - subir o server real e validar UI + API no mesmo `http.port`;
  - validar refresh direto em deep link da SPA.
- Checks do frontend:
  - `make web-lint`
  - `make web-typecheck`
  - `make web-test`
  - `make web-build`
- Gate final:
  - `make verify`

## Assunções e defaults

- Estratégia escolhida: bundle embedado no daemon apenas; sem modo de proxy para Vite dentro do daemon.
- O fluxo suportado para gerar o binário passa a ser `make build` e `make verify`; não vamos mascarar ausência do bundle com UI vazia, leitura do disco ou auto-build no startup.
- O frontend já está preparado para same-origin em produção, então não há mudança de contrato em `/api/*`; o trabalho é integrar serving, fallback e pipeline.
