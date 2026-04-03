# TechSpec: Migrar Roles e Playbooks para Markdown com YAML Frontmatter

## Executive Summary

Migração greenfield de roles (TOML) e playbooks (Markdown raw) para o formato unificado de Markdown com YAML front matter. Um novo pacote compartilhado `internal/frontmatter` centraliza o parsing e a serialização, eliminando a dependência de `BurntSushi/toml` no carregamento de roles. Roles ganham um campo `description` e passam a usar `.md` como extensão. Playbooks ganham metadados estruturados (`name`, `description`, `domain`, `tags`) no frontmatter. Sem retrocompatibilidade — o projeto está em fase inicial e a base de usuários com artefatos custom é negligível.

Decisões-chave: formato unificado Markdown+YAML (ADR-001), parser compartilhado (ADR-002), migração greenfield sem retrocompatibilidade (ADR-003).

## System Architecture

### Component Overview

```
┌──────────────────────────────────────────────────────────┐
│                    internal/frontmatter                   │
│                                                          │
│  Parse(content, dest) → (body, error)                    │
│  Format(meta, body) → ([]byte, error)                    │
│                                                          │
│  Depende de: goccy/go-yaml                               │
└──────────┬──────────────────────┬────────────────────────┘
           │                      │
           ▼                      ▼
┌────────────────────┐  ┌────────────────────┐
│  internal/config   │  │  internal/roles    │
│                    │  │                    │
│  roles.go          │  │  bundled.go        │
│  - LoadRoles       │  │  - LoadBundled     │
│  - loadRoleFile    │  │  - Install         │
│  - RoleConfig      │  │                    │
│                    │  │  bundled/           │
│  playbooks.go      │  │  - embed.go        │
│  - LoadPlaybooks   │  │  - *.md            │
│  - loadPlaybookFile│  │                    │
│  - Playbook        │  └────────────────────┘
│                    │
│  discovery.go      │
│  - writeRoleFile   │
│  - SaveRoleDraft   │
│  - SavePlaybookDraft│
└────────────────────┘
           │
           ▼
┌────────────────────┐
│  internal/prompt   │
│                    │
│  assembler.go      │
│  - Assemble        │
│  - renderSpecial.  │
└────────────────────┘
```

**Fluxo de dados:**

1. Arquivo `.md` no disco → `frontmatter.Parse` extrai YAML + body
2. YAML unmarshalled em `RoleConfig` ou `Playbook` struct
3. Body atribuído ao campo `SystemPrompt` (roles) ou `Content` (playbooks)
4. Prompt assembler lê `role.SystemPrompt` sem mudança na interface

## Implementation Design

### Core Interfaces

**`internal/frontmatter` — parser compartilhado:**

```go
package frontmatter

// Parse extrai YAML frontmatter do content para dest e retorna o body.
// O content deve começar com "---\n". Retorna erro se frontmatter ausente.
func Parse(content []byte, dest any) (body string, err error)

// Format serializa meta como YAML frontmatter + body como Markdown.
func Format(meta any, body string) ([]byte, error)
```

**`RoleConfig` — struct atualizado:**

```go
type RoleConfig struct {
    Name         string         `yaml:"name"`
    Description  string         `yaml:"description"`
    Type         string         `yaml:"type"`
    Driver       string         `yaml:"driver"`
    Model        string         `yaml:"model"`
    SystemPrompt string         `yaml:"-"`
    Status       ArtifactStatus `yaml:"-"`
    DraftVersion int            `yaml:"-"`
    Path         string         `yaml:"-"`
}
```

**`Playbook` — struct atualizado:**

```go
type Playbook struct {
    Name         string         `yaml:"name"`
    Description  string         `yaml:"description"`
    Domain       string         `yaml:"domain"`
    Tags         []string       `yaml:"tags"`
    Content      string         `yaml:"-"`
    Status       ArtifactStatus `yaml:"-"`
    DraftVersion int            `yaml:"-"`
    Path         string         `yaml:"-"`
}
```

### Data Models

**Role file format (`*.md`):**

```markdown
---
name: architect
description: Strategic architecture advisor for code structure and design trade-offs
type: advisor
driver: claude
model: sonnet
---
You are an architecture advisor. Provide strategic, read-only analysis of code
structure, design decisions, and system trade-offs...
```

**Playbook file format (`*.md`):**

```markdown
---
name: software-dev
description: Software development strategy focused on quality and best practices
domain: software-dev
tags:
  - development
  - quality
---
# Software Development

When approaching a software development task...
```

**Naming conventions (unchanged pattern, new extension for roles):**

| Artifact | Approved | Draft | Versioned Draft |
|----------|----------|-------|-----------------|
| Role | `{name}.md` | `{name}.draft.md` | `{name}.draft_v{N}.md` |
| Playbook | `{name}.md` | `{name}.draft.md` | `{name}.draft_v{N}.md` |

### API Endpoints

Não há endpoints HTTP afetados. As mudanças são internas ao carregamento de artefatos e CLI.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/frontmatter/` | new | Novo pacote parser genérico. Risco: baixo | Criar `frontmatter.go` e `frontmatter_test.go` |
| `internal/config/roles.go` | modified | `RoleConfig` ganha `Description`, tags mudam de `toml:` para `yaml:`, `loadRoleFile` usa `frontmatter.Parse`, extensões mudam de `.toml` para `.md`. Risco: médio — muitos testes dependem | Reescrever `loadRoleFile`, atualizar `parseCatalogFileName` calls, adicionar `Description` |
| `internal/config/playbooks.go` | modified | `Playbook` ganha `Description`, `Domain`, `Tags`, `loadPlaybookFile` usa `frontmatter.Parse`. Risco: médio | Reescrever `loadPlaybookFile`, atualizar struct |
| `internal/config/discovery.go` | modified | `writeRoleFile` gera Markdown+YAML em vez de TOML. `SavePlaybookDraft`/`saveApprovedPlaybook` serializam frontmatter. Remove import `BurntSushi/toml`. Risco: médio | Reescrever `writeRoleFile`, atualizar saves de playbook para usar `frontmatter.Format` |
| `internal/roles/bundled.go` | modified | `LoadBundled` usa `frontmatter.Parse` em vez de `toml.Decode`. Remove import `BurntSushi/toml`. Risco: baixo | Reescrever parsing |
| `internal/roles/bundled/embed.go` | modified | Directive muda de `*.toml` para `*.md`. Risco: baixo | Trocar directive |
| `internal/roles/bundled/*.toml` | deprecated | 6 arquivos TOML removidos e substituídos por 6 `.md`. Risco: baixo | Reescrever cada arquivo |
| `internal/config/config_test.go` | modified | Todos os testes que criam role/playbook fixtures precisam do novo formato. Risco: alto — arquivo grande (~1670 linhas), muitos fixtures TOML | Atualizar todos os `writeFile` de roles para Markdown+YAML, playbooks para incluir frontmatter |
| `internal/roles/bundled_test.go` | modified | Testes de bundled roles precisam refletir novo formato. Risco: baixo | Atualizar assertions |
| `internal/prompt/assembler.go` | unchanged | Já lê `role.SystemPrompt` — campo continua existindo. Risco: nenhum | Nenhuma ação necessária |

## Testing Approach

### Unit Tests

**`internal/frontmatter/frontmatter_test.go`:**
- Parse com frontmatter válido e body
- Parse com frontmatter válido e body vazio
- Parse sem delimitadores `---` (deve falhar)
- Parse com YAML inválido no frontmatter (deve falhar)
- Parse com apenas frontmatter, sem body
- Format round-trip: `Format(meta, body)` → `Parse(result, &meta)` → body igual
- Format com campos vazios

**`internal/config/config_test.go` (roles):**
- Testes existentes de `LoadRoles`, `FindRole`, `SaveRoleDraft`, `ApproveRole` mantêm a mesma semântica, fixtures mudam de TOML para Markdown
- Novo teste: valida que `Description` é carregado corretamente
- Novo teste: rejeita arquivo sem frontmatter

**`internal/config/config_test.go` (playbooks):**
- Testes existentes mantêm a mesma semântica, fixtures ganham frontmatter
- Novo teste: valida `Description`, `Domain`, `Tags` carregados
- Novo teste: rejeita playbook sem frontmatter

**`internal/roles/bundled_test.go`:**
- Testes existentes validam parsing dos novos arquivos `.md`
- Valida que `Description` está presente em todos os bundled roles

### Integration Tests

- `TestLoadFromRootRoundTrip` e `TestLoadFromRootMergesGlobalAndWorkspaceArtifacts` exercitam o pipeline completo (disco → parsing → merge → lookup) com os novos formatos.
- `TestSaveRoleDraftAndApproveRole` / `TestSavePlaybookDraftAndApprovePlaybook` validam o ciclo draft → approve com serialização Markdown.

## Development Sequencing

### Build Order

1. **`internal/frontmatter`** — parser e formatter genéricos. Sem dependências internas. Testes unitários isolados.
2. **`internal/config/roles.go`** — migra `RoleConfig` struct (tags `yaml:`, campo `Description`), reescreve `loadRoleFile` para usar `frontmatter.Parse`, atualiza extensões de `.toml`/`.draft.toml` para `.md`/`.draft.md`.
3. **`internal/roles/bundled/`** — reescreve 6 arquivos `.toml` → `.md`, atualiza `embed.go` para `go:embed *.md`.
4. **`internal/roles/bundled.go`** — migra `LoadBundled` para usar `frontmatter.Parse`, remove `BurntSushi/toml` import.
5. **`internal/config/playbooks.go`** — migra `Playbook` struct (adiciona `Description`, `Domain`, `Tags`), reescreve `loadPlaybookFile` para usar `frontmatter.Parse`.
6. **`internal/config/discovery.go`** — migra `writeRoleFile` para gerar Markdown+YAML via `frontmatter.Format`. Atualiza `SavePlaybookDraft`/`saveApprovedPlaybook` para serializar frontmatter+body. Remove import `BurntSushi/toml`.
7. **`internal/config/config_test.go`** — atualiza todos os fixtures de roles (TOML → Markdown) e playbooks (raw → frontmatter). Adiciona testes para novos campos.
8. **`internal/roles/bundled_test.go`** — atualiza assertions para novo formato.

### Technical Dependencies

- `goccy/go-yaml` já está no `go.mod` — nenhuma nova dependência necessária.
- Nenhuma dependência de infraestrutura ou serviço externo.

## Monitoring and Observability

Não se aplica. A mudança é de formato de serialização de artefatos locais. Erros de parsing já são surfaced via retorno de erro no boot do kernel e na CLI.

## Technical Considerations

### Key Decisions

- **Frontmatter obrigatório**: Arquivos sem `---` delimitadores são rejeitados. Não há fallback para formato antigo.
- **`SystemPrompt` vem do body, não do frontmatter**: Evita problemas de escaping YAML com textos longos multi-linha. O campo `SystemPrompt` tem tag `yaml:"-"` e é populado pelo retorno de `frontmatter.Parse`.
- **`Content` dos playbooks vem do body**: Mesma abordagem — metadados no frontmatter, conteúdo no body.
- **Validação de `name` mantém `validateCatalogName`**: A regra de que `name` no frontmatter deve coincidir com o nome extraído do filename é preservada.
- **`Description` obrigatório para roles**: Roles sem `description` falham na validação (`Validate()`). Para playbooks, `description` é opcional para não quebrar o fluxo de criação rápida.
- **`formatUndecodedKeys` não existe em YAML**: O YAML parser do `goccy/go-yaml` usa `KnownFields(true)` ou validação manual para rejeitar campos desconhecidos. O behavior de strict parsing é preservado para roles.

### Known Risks

- **Edge cases em YAML frontmatter**: Valores com `:` ou caracteres especiais no `description` podem precisar de quoting. Mitigação: YAML suporta quoting nativo; documentar no formato.
- **Playbooks sem frontmatter no disco de usuários existentes**: Serão rejeitados pelo loader após a migração. Mitigação: migração greenfield aceita (ADR-003); early-stage project.

## Architecture Decision Records

- [ADR-001: Markdown with YAML Frontmatter as Unified Artifact Format](adrs/adr-001.md) — Adoção de Markdown+YAML como formato unificado para roles, playbooks e skills.
- [ADR-002: Shared Frontmatter Parser Package](adrs/adr-002.md) — Criação de `internal/frontmatter` como parser/formatter compartilhado.
- [ADR-003: Greenfield Migration Without Backward Compatibility](adrs/adr-003.md) — Migração limpa sem suporte ao formato antigo.
