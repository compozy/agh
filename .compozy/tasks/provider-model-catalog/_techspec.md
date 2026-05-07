# Provider Model Catalog and ACP Session Config Options

## MVP Boundary

The MVP implements a daemon-owned provider model catalog for pre-session model selection, while treating ACP `configOptions` as the active-session source of truth. It hard-cuts the existing flat provider model fields, persists catalog source rows/status in the global SQLite database, exposes catalog read/refresh/status through HTTP, UDS, CLI, Host API, and web, adds an HTTP-only OpenAI-compatible `/api/openai/v1/models` projection, and upgrades ACP handling to prefer `session/set_config_option` where available.

The MVP does not implement Droid discovery, does not create fake ACP sessions for discovery, does not make `models.dev` an availability authority, and does not convert `models.curated` into a permission boundary. Manual model entry remains valid.

## Problem Statement

AGH can now create sessions with explicit model and reasoning overrides, but provider model discovery still depends on static config hints:

- `internal/config/provider.go` stores `DefaultModel`, `SupportedModels`, and `SupportsReasoningEffort`.
- `internal/api/core/conversions.go` maps those static fields into `SessionProviderOptionPayload`.
- `web/src/systems/session/hooks/use-session-create-dialog.ts` builds model options directly from `supported_models`.
- `internal/acp/client.go` captures ACP `models.availableModels` after `session/new`/`session/load`, which is too late for pre-session selection.
- `internal/cli/provider.go` only exposes provider auth commands, not model catalog commands.

The correct split is:

- Pre-session catalog: "Which provider models does AGH know about before creating a session?"
- Active ACP session config: "Which controls does this ACP session expose right now?"

The catalog must be daemon-owned, persisted, refreshable, extensible, and agent-manageable.

## Inputs and Evidence

Local competitor and reference evidence:

- Zed: `.resources/zed/crates/agent_ui/src/config_options.rs`, `.resources/zed/crates/acp_thread/src/connection.rs`, `.resources/zed/crates/agent_servers/src/acp.rs`
- Harnss: `.resources/harnss/electron/src/ipc/claude-sessions.ts`, `.resources/harnss/shared/lib/codex-helpers.ts`, `.resources/harnss/src/lib/model-utils.ts`, `.resources/harnss/src/types/window.d.ts`
- Paperclip: `.resources/paperclip/adapter-plugin.md`, `.resources/paperclip/packages/adapters/opencode-local/src/server/models.ts`, `.resources/paperclip/packages/adapters/opencode-local/src/server/test.ts`, `.resources/paperclip/packages/adapters/acpx-local/src/server/execute.ts`
- `compozy-code`: `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/model-discovery-service.ts`, `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/catalog-sources/models-dev-source.ts`, `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/server/routes-models.ts`, `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/types.ts`
- Current ACP dependency: `github.com/coder/acp-go-sdk v0.6.3` in `go.mod`.
- Latest available ACP SDK version confirmed with `go list -m -versions github.com/coder/acp-go-sdk` on 2026-05-07: `v0.12.2`.

## Architectural Boundaries

1. `internal/modelcatalog` owns model catalog source execution, merge policy, refresh decisions, source status, and catalog projections.
2. `internal/store/globaldb` owns durable catalog rows and schema migrations. Other packages must not write catalog tables directly.
3. `internal/config` owns provider config parsing/validation/merge only. It does not perform network discovery and does not persist catalog rows.
4. `internal/acp` owns ACP session state and captures session-scoped `configOptions`. It may report observations to session state, but it must not rewrite global pre-session catalog truth.
5. `internal/api/core` owns shared HTTP/UDS handlers. HTTP and UDS only register transports and auth.
6. `internal/cli` consumes UDS/core client surfaces. It does not discover models by reading provider config files directly.
7. `internal/extension` may provide model source rows through capability-gated service calls. The daemon still validates, persists, and merges rows.
8. `web/` consumes generated contract types and catalog endpoints via a feature system. It must not reconstruct provider model lists from old settings payloads.
9. `packages/site` documents generated API/CLI surfaces; generated references remain source-of-truth for exact shapes.

## Delete Targets

This is a greenfield hard cut. Remove these in the implementation pass:

- `internal/config.ProviderConfig.DefaultModel`
- `internal/config.ProviderConfig.SupportedModels`
- `internal/config.ProviderConfig.SupportsReasoningEffort`
- `internal/config.ProviderConfig.EffectiveSupportedModels`
- `internal/config.ProviderConfig.EffectiveSupportsReasoningEffort`
- `internal/config.validateProviderSupportedModels`
- `internal/config.mergeProvider` and overlay fields for old model keys
- TOML keys `providers.<id>.default_model`, `providers.<id>.supported_models`, `providers.<id>.supports_reasoning_effort`
- API fields `default_model`, `supported_models`, and `supports_reasoning_effort` in provider settings and session provider option payloads
- Web settings form controls and view-model fields that edit/read the old flat keys
- New-session dialog logic that treats `supported_models` as the model source
- Test fixtures and generated OpenAPI/TypeScript fields that encode the old shape

No aliases, dual fields, compatibility fallback reads, redirects, or schema fallback paths are allowed.

## Proposed Design

Add a new daemon service:

```text
ProviderConfig models block
  + builtin defaults
  + models.dev catalog source
  + live provider/runtime sources
  + extension model sources
  + ACP session observations
        |
        v
internal/modelcatalog.Service
        |
        v
global SQLite source rows + status
        |
        v
merged catalog projection
        |
        v
HTTP / UDS / CLI / Host API / web / GET /api/openai/v1/models
```

The merge key is `(provider_id, model_id)`. Source rows are preserved separately. The projection is computed from rows sorted by priority, row freshness, and source identity:

- `config` priority 120 for fields explicitly defined by the operator.
- `provider_live` priority 110 for account/runtime availability and live metadata.
- `extension` priority 100 for extension-provided availability and metadata.
- `models.dev` priority 50 for broad enrichment and stale fallback.
- `builtin` priority 10 for offline bootstrap.
- `acp_session` rows are session-scoped observations and do not participate in global pre-session authority. They can be exposed with session context only.

Higher-priority sources win conflicting non-empty fields. Lower-priority sources fill missing fields. If two rows still tie after priority, the fresher `refreshed_at` wins; if freshness also ties, ascending `source_id` wins. The deterministic projection sort is `(provider_id ASC, model_id ASC)`, with each model's `sources` sorted by `(priority DESC, refreshed_at DESC, source_id ASC)`.

Availability is explicit: live provider/extension rows can mark `available=true` or `available=false`; catalog-only rows default to `available=null`, not true.

### Merged Availability

The merged projection exposes both nullable `available` and string `availability_state` so stale live truth is visible instead of collapsed:

- `available_live`: selected highest-authority live/extension row has `available=true` and `stale=false`; API `available=true`.
- `available_stale`: selected highest-authority live/extension row has `available=true` and `stale=true`; API `available=true` and `stale=true`.
- `unavailable_live`: selected highest-authority live/extension row has `available=false` and `stale=false`; API `available=false`.
- `unavailable_stale`: selected highest-authority live/extension row has `available=false` and `stale=true`; API `available=false` and `stale=true`.
- `unknown`: no live/extension row provides availability, or only catalog/builtin/config metadata exists; API `available=null`.

`models.dev` and `builtin` rows never move a model out of `unknown` availability. UI must label stale availability states and must keep manual model entry available.

## Core Interfaces

Add `internal/modelcatalog` with concrete interfaces and types shaped like this:

```go
package modelcatalog

import (
	"context"
	"time"
)

type SourceKind string

const (
	SourceKindBuiltin      SourceKind = "builtin"
	SourceKindConfig       SourceKind = "config"
	SourceKindModelsDev    SourceKind = "models_dev"
	SourceKindProviderLive SourceKind = "provider_live"
	SourceKindExtension    SourceKind = "extension"
	SourceKindACPSession   SourceKind = "acp_session"
)

type ReasoningEffort string

const (
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	ReasoningEffortLow     ReasoningEffort = "low"
	ReasoningEffortMedium  ReasoningEffort = "medium"
	ReasoningEffortHigh    ReasoningEffort = "high"
	ReasoningEffortXHigh   ReasoningEffort = "xhigh"
)

type ListOptions struct {
	ProviderID string
	SourceID   string
	Refresh    bool
	IncludeAll bool
	Now        time.Time
}

type RefreshOptions struct {
	ProviderID string
	SourceID   string
	Force      bool
	RequestID  string
	Now        time.Time
}

type ModelRow struct {
	ProviderID             string
	ModelID                string
	DisplayName            string
	SourceID               string
	SourceKind             SourceKind
	Priority               int
	Available              *bool
	Stale                  bool
	RefreshedAt            time.Time
	ExpiresAt              time.Time
	ContextWindow          *int64
	MaxInputTokens         *int64
	MaxOutputTokens        *int64
	SupportsTools          *bool
	SupportsReasoning      *bool
	ReasoningEfforts       []ReasoningEffort
	DefaultReasoningEffort *ReasoningEffort
	CostInputPerMillion    *float64
	CostOutputPerMillion   *float64
	LastError              string
}

type Model struct {
	ProviderID             string
	ModelID                string
	DisplayName            string
	Sources                []SourceRef
	Available              *bool
	AvailabilityState      string
	Stale                  bool
	RefreshedAt            time.Time
	ContextWindow          *int64
	MaxInputTokens         *int64
	MaxOutputTokens        *int64
	SupportsTools          *bool
	SupportsReasoning      *bool
	ReasoningEfforts       []ReasoningEffort
	DefaultReasoningEffort *ReasoningEffort
	CostInputPerMillion    *float64
	CostOutputPerMillion   *float64
	LastError              string
}

type SourceRef struct {
	SourceID    string
	SourceKind  SourceKind
	Priority    int
	RefreshedAt time.Time
	Stale       bool
	LastError   string
}

type SourceStatus struct {
	SourceID     string
	SourceKind   SourceKind
	ProviderID   string
	LastRefresh  time.Time
	NextRefresh  time.Time
	LastSuccess  time.Time
	LastError    string
	RefreshState string
	RowCount     int
	Stale        bool
}

type Source interface {
	ID() string
	Kind() SourceKind
	Priority() int
	ListModels(ctx context.Context, opts ListOptions) ([]ModelRow, error)
}

type Store interface {
	ReplaceSourceRows(ctx context.Context, sourceID string, providerID string, rows []ModelRow, status SourceStatus) error
	ListRows(ctx context.Context, opts ListOptions) ([]ModelRow, error)
	ListSourceStatus(ctx context.Context, providerID string) ([]SourceStatus, error)
}

type Service interface {
	ListModels(ctx context.Context, opts ListOptions) ([]Model, error)
	Refresh(ctx context.Context, opts RefreshOptions) ([]SourceStatus, error)
	ListSourceStatus(ctx context.Context, providerID string) ([]SourceStatus, error)
}
```

The provider config shape becomes:

```go
package config

type ProviderConfig struct {
	Command         string               `toml:"command"`
	DisplayName     string               `toml:"display_name,omitempty"`
	Models          ProviderModelsConfig `toml:"models,omitempty"`
	Harness         ProviderHarness      `toml:"harness,omitempty"`
	RuntimeProvider string               `toml:"runtime_provider,omitempty"`
	Transport       string               `toml:"transport,omitempty"`
	BaseURL         string               `toml:"base_url,omitempty"`
	AuthMode        ProviderAuthMode     `toml:"auth_mode,omitempty"`
	EnvPolicy       ProviderEnvPolicy    `toml:"env_policy,omitempty"`
	HomePolicy      ProviderHomePolicy   `toml:"home_policy,omitempty"`
	AuthStatusCmd   string               `toml:"auth_status_command,omitempty"`
	AuthLoginCmd    string               `toml:"auth_login_command,omitempty"`
	SessionMCP      *bool                `toml:"session_mcp,omitempty"`
	Aliases         []string             `toml:"aliases,omitempty"`
	CredentialSlots []ProviderCredentialSlot `toml:"credential_slots,omitempty"`
	MCPServers      []MCPServer          `toml:"mcp_servers,omitempty"`
}

type ProviderModelsConfig struct {
	Default   string                        `toml:"default,omitempty"`
	Curated   []ProviderModelConfig         `toml:"curated,omitempty"`
	Discovery ProviderModelsDiscoveryConfig `toml:"discovery,omitempty"`
}

type ProviderModelsDiscoveryConfig struct {
	Enabled  *bool  `toml:"enabled,omitempty"`
	Command  string `toml:"command,omitempty"`
	Endpoint string `toml:"endpoint,omitempty"`
	Timeout  string `toml:"timeout,omitempty"`
}

type ProviderModelConfig struct {
	ID                     string   `toml:"id"`
	DisplayName            string   `toml:"display_name,omitempty"`
	ContextWindow          *int64   `toml:"context_window,omitempty"`
	MaxInputTokens         *int64   `toml:"max_input_tokens,omitempty"`
	MaxOutputTokens        *int64   `toml:"max_output_tokens,omitempty"`
	SupportsTools          *bool    `toml:"supports_tools,omitempty"`
	SupportsReasoning      *bool    `toml:"supports_reasoning,omitempty"`
	ReasoningEfforts       []string `toml:"reasoning_efforts,omitempty"`
	DefaultReasoningEffort string   `toml:"default_reasoning_effort,omitempty"`
	CostInputPerMillion    *float64 `toml:"cost_input_per_million,omitempty"`
	CostOutputPerMillion   *float64 `toml:"cost_output_per_million,omitempty"`
}

type ModelCatalogConfig struct {
	Sources ModelCatalogSourcesConfig `toml:"sources,omitempty"`
}

type ModelCatalogSourcesConfig struct {
	ModelsDev ModelsDevSourceConfig `toml:"models_dev,omitempty"`
}

type ModelsDevSourceConfig struct {
	Enabled  *bool  `toml:"enabled,omitempty"`
	Endpoint string `toml:"endpoint,omitempty"`
	TTL      string `toml:"ttl,omitempty"`
	Timeout  string `toml:"timeout,omitempty"`
}
```

The active ACP session state adds session-scoped config options:

```go
package acp

type ConfigOptionKind string

const (
	ConfigOptionKindEnum    ConfigOptionKind = "enum"
	ConfigOptionKindString  ConfigOptionKind = "string"
	ConfigOptionKindBoolean ConfigOptionKind = "boolean"
)

type SessionConfigOption struct {
	ID          string
	Label       string
	Description string
	Kind        ConfigOptionKind
	Current     string
	Values      []SessionConfigOptionValue
}

type SessionConfigOptionValue struct {
	Value string
	Label string
}
```

`internal/acp.Driver.applySessionModel` must first use `session/set_config_option` when a config option with model semantics exists; it falls back to `session/set_model` only when config options are absent. Reasoning effort follows the same rule and is never sent when the active session exposes no matching control.

## Data Model

Append a new global SQLite migration at the tail of `internal/store/globaldb.globalSchemaMigrations`.

```sql
CREATE TABLE model_catalog_sources (
  source_id TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  source_kind TEXT NOT NULL,
  priority INTEGER NOT NULL,
  refresh_state TEXT NOT NULL,
  last_refresh_at TEXT NOT NULL DEFAULT '',
  next_refresh_at TEXT NOT NULL DEFAULT '',
  last_success_at TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  row_count INTEGER NOT NULL DEFAULT 0,
  stale INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (source_id, provider_id)
);

CREATE TABLE model_catalog_rows (
  source_id TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  model_id TEXT NOT NULL,
  source_kind TEXT NOT NULL,
  priority INTEGER NOT NULL,
  available INTEGER,
  stale INTEGER NOT NULL DEFAULT 0,
  refreshed_at TEXT NOT NULL DEFAULT '',
  expires_at TEXT NOT NULL DEFAULT '',
  display_name TEXT NOT NULL DEFAULT '',
  context_window INTEGER,
  max_input_tokens INTEGER,
  max_output_tokens INTEGER,
  supports_tools INTEGER,
  supports_reasoning INTEGER,
  default_reasoning_effort TEXT,
  cost_input_per_million REAL,
  cost_output_per_million REAL,
  last_error TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (source_id, provider_id, model_id)
);

CREATE TABLE model_catalog_reasoning_efforts (
  source_id TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  model_id TEXT NOT NULL,
  effort TEXT NOT NULL,
  rank INTEGER NOT NULL,
  PRIMARY KEY (source_id, provider_id, model_id, effort),
  FOREIGN KEY (source_id, provider_id, model_id)
    REFERENCES model_catalog_rows(source_id, provider_id, model_id)
    ON DELETE CASCADE
);

CREATE INDEX idx_model_catalog_rows_provider_model
  ON model_catalog_rows(provider_id, model_id, priority DESC, refreshed_at DESC, source_id ASC);

CREATE INDEX idx_model_catalog_rows_source_provider
  ON model_catalog_rows(source_id, provider_id);

CREATE INDEX idx_model_catalog_sources_provider
  ON model_catalog_sources(provider_id, refresh_state, stale);
```

### Data-Model Field Rationale

- `source_id`: stable source identity such as `config`, `builtin`, `models_dev`, `provider_live:codex`, or `extension:<slug>`. Dynamic IDs use `<kind>:<slug>` where `<slug>` must match `^[a-z0-9][a-z0-9_-]*$`; extension names are normalized to that slug shape during manifest validation and rejected if they cannot be normalized deterministically.
- `provider_id`: AGH provider ID, not upstream vendor ID. `model_catalog_sources` is always per-provider; cross-provider sources such as `models.dev` synthesize one status row per provider refreshed or reported, so status listing never has to merge global and per-provider source rows.
- `source_kind`: enum for merge policy, status grouping, and UI labels.
- `priority`: persisted with rows so historical projections remain inspectable even if source registration changes.
- `refresh_state`: `idle`, `refreshing`, `succeeded`, or `failed`; used by CLI/web/Host API.
- `last_refresh_at`, `next_refresh_at`, `last_success_at`: source freshness and refresh scheduling.
- `last_error`: redacted operator-facing error summary; never stores secret-bearing command lines or response bodies.
- `row_count`: quick status display and regression assertions.
- `stale`: stale rows are usable fallback but must be labeled.
- `model_id`: exact provider model identifier sent to the runtime.
- `available`: tri-state. `true` means live source confirmed; `false` means live source denied; `null` means catalog metadata only.
- `refreshed_at`, `expires_at`: row-level freshness independent of source-level status.
- `display_name`: optional human label; empty means use `model_id`.
- `context_window`, `max_input_tokens`, `max_output_tokens`: separate columns for filtering/sorting and OpenAI projection.
- `supports_tools`, `supports_reasoning`: nullable booleans so unknown is not collapsed into false.
- `default_reasoning_effort`: nullable per-model default when explicitly known; `NULL` means no source knows a default and UI/runtime defaults apply.
- `cost_input_per_million`, `cost_output_per_million`: numeric cost columns for sorting/display; no currency conversion in MVP.
- `model_catalog_reasoning_efforts.effort`: normalized effort value.
- `model_catalog_reasoning_efforts.rank`: stable order for UI and deterministic payloads.

### Side-Table-vs-JSON Decisions

1. Model source rows use a table instead of a JSON blob because provider/model/source lookup, priority merge, stale filtering, and status joins are core runtime queries.
2. Reasoning efforts use a side table instead of JSON because they need deterministic ordering, exact matching, validation, and partial source merge.
3. Cost uses scalar columns instead of JSON because the MVP only supports input/output per-million pricing and needs direct sorting/display.
4. Source status uses a table instead of embedding status in each row because a source can fail while stale rows remain usable.
5. Raw upstream payloads are not stored in MVP. They create redaction risk, schema drift risk, and no required runtime query.

## Source Implementations

### Builtin Source

`builtin` converts built-in provider model defaults into rows with priority 10. It enables offline fresh installs and documents AGH's first-run recommendations.

### Config Source

`config` converts `providers.<id>.models.default` and `models.curated` into rows. Explicit config metadata wins for the fields it defines. Empty config fields do not erase lower-priority enrichment.

### Models.dev Source

`models_dev` fetches `https://models.dev/api.json` using an explicit-timeout HTTP client. It has a 24h TTL and returns stale cached rows when refresh fails after a prior success. Operators can disable or override this source through `[model_catalog.sources.models_dev]`; disabled sources return status and do not perform outbound network calls.

The parser must accept current and legacy schema variants:

- `reasoning`, `supportsReasoning`, `supports_reasoning`
- `tool_call`, `supportsTools`, `supports_tools`
- `limit.context`, `limit.input`, `limit.output`
- `contextWindow`, `maxInputTokens`, `maxOutputTokens`
- `cost.input`, `cost.output`, `pricing.input`, `pricing.output`

Provider mapping is AGH-specific. Do not copy `compozy-code` mappings literally. A single `models.dev` refresh writes provider-scoped source status rows for each AGH provider it maps, never a global empty-provider status row.

### Live Provider Sources

Implement side-effect-free, timeout-bound sources for:

- OpenAI/Codex
- Anthropic/Claude
- Gemini
- OpenRouter
- Vercel AI Gateway
- Ollama
- OpenCode

OpenClaw, Hermes, and Pi get explicit `providers.<id>.models.discovery` adapter/config support that fails closed with source status when no side-effect-free command or endpoint is configured. Discovery config accepts `enabled`, `command`, `endpoint`, and `timeout`; at least one of `command` or `endpoint` is required when enabled for these providers. Discovery must use the provider's effective auth/home/env policy and must not create ACP sessions.

Refresh execution is serialized per `provider_id` before any live discovery subprocess or provider-home operation starts. Concurrent refresh requests for the same provider coalesce behind the in-flight refresh and return the same source statuses when it finishes; refreshes for different providers may proceed concurrently. The service mints or accepts a `refresh_request_id` at the handler/service boundary and carries it through logs, source status events, and source-specific errors.

### Extension Sources

Extensions declaring `model.source` can answer `models/list`. AGH validates rows, records status, and applies normal merge policy.

## Public Interfaces

### Native HTTP and UDS

Add shared handlers in `internal/api/core` and register them in HTTP/UDS:

- `GET /api/providers/models`
- `GET /api/providers/{provider_id}/models`
- `POST /api/providers/models/refresh`
- `POST /api/providers/{provider_id}/models/refresh`
- `GET /api/providers/models/status`
- `GET /api/providers/{provider_id}/models/status`

Representative payload:

```go
type ProviderModelPayload struct {
	ProviderID             string                  `json:"provider_id"`
	ModelID                string                  `json:"model_id"`
	DisplayName            string                  `json:"display_name,omitempty"`
	Sources                []ModelCatalogSourceRef `json:"sources,omitempty"`
	Available              *bool                   `json:"available,omitempty"`
	AvailabilityState      string                  `json:"availability_state"`
	Stale                  bool                    `json:"stale,omitempty"`
	RefreshedAt            string                  `json:"refreshed_at,omitempty"`
	ContextWindow          *int64                  `json:"context_window,omitempty"`
	MaxInputTokens         *int64                  `json:"max_input_tokens,omitempty"`
	MaxOutputTokens        *int64                  `json:"max_output_tokens,omitempty"`
	SupportsTools          *bool                   `json:"supports_tools,omitempty"`
	SupportsReasoning      *bool                   `json:"supports_reasoning,omitempty"`
	ReasoningEfforts       []string                `json:"reasoning_efforts,omitempty"`
	DefaultReasoningEffort *string                 `json:"default_reasoning_effort,omitempty"`
	Cost                   *ModelCostPayload       `json:"cost,omitempty"`
	LastError              string                  `json:"last_error,omitempty"`
}
```

### OpenAI-Compatible Projection

Add HTTP-only `GET /api/openai/v1/models` with optional `provider_id` query. Do not register this route on UDS. It uses the same bearer-auth and middleware contract as `/api/*`, including CORS and rate-limit behavior when those are enabled for HTTP. Unauthenticated or unauthorized requests return an OpenAI-shaped error envelope with AGH's normal status code semantics.

Response shape:

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-5.4",
      "object": "model",
      "created": 0,
      "owned_by": "codex",
      "agh": {
        "provider_id": "codex",
        "display_name": "GPT-5.4",
        "supports_tools": true,
        "supports_reasoning": true,
        "availability_state": "available_live",
        "reasoning_efforts": ["minimal", "low", "medium", "high", "xhigh"],
        "context_window": 256000,
        "max_output_tokens": 32000,
        "sources": ["config", "models_dev"]
      }
    }
  ]
}
```

Use `agh` metadata, not `compozy`, for AGH-specific fields. The OpenAI-compatible route is list-only; refresh remains available only through the native catalog refresh endpoints and CLI/UDS/Host API surfaces.

### CLI

Add commands under the existing singular `provider` namespace:

```bash
agh provider models list [provider] -o json
agh provider models refresh [provider] -o json
agh provider models status [provider] -o json
```

The list command supports `--source`, `--refresh`, and `--include-stale`. The refresh command returns source statuses, not only success text.

The MVP intentionally keeps commands under `agh provider models ...` because the catalog is provider-scoped and already neighbors `agh provider auth ...`. The site docs must explain this namespace choice. A top-level `agh models` alias is out of scope for the MVP to avoid adding a second command contract before the first one is stable.

### Extension Protocol

Add:

- manifest provide capability: `model.source`
- AGH -> extension method: `models/list`
- Host API methods:
  - `models/list`
  - `models/refresh`
  - `models/status`

Host API methods require capability checks and return daemon-owned projections/status, not raw extension payloads.

### Web

Add or update a model catalog system under `web/src/systems/`:

- query keys and query options for provider model lists/status/refresh;
- abort-signal aware adapter methods;
- loading, stale, error, empty, manual-entry, and refresh states;
- new-session dialog model picker uses catalog rows for the selected provider;
- Settings > Providers edits `models.default` and `models.curated`, and displays source status.

After session creation, active session controls switch to ACP `configOptions` if present. Catalog assumptions never override active session `configOptions`.

## Config Lifecycle

New TOML shape:

```toml
[providers.codex]
command = "npx -y @zed-industries/codex-acp@latest"
display_name = "Codex"
harness = "acp"

[providers.codex.models]
default = "gpt-5.4"

[[providers.codex.models.curated]]
id = "gpt-5.4"
display_name = "GPT-5.4"
supports_tools = true
supports_reasoning = true
reasoning_efforts = ["minimal", "low", "medium", "high", "xhigh"]
default_reasoning_effort = "medium"

[model_catalog.sources.models_dev]
enabled = true
endpoint = "https://models.dev/api.json"
ttl = "24h"
timeout = "10s"

[providers.openclaw.models.discovery]
enabled = true
command = "openclaw models --json"
timeout = "10s"
```

Validation rules:

- `models.default` may be any non-blank model ID. It does not need to appear in `models.curated`.
- each curated model `id` is required and unique per provider;
- blank reasoning efforts are rejected;
- `default_reasoning_effort` must be present in `reasoning_efforts` when both are set;
- old flat model keys are rejected with exact error paths;
- `[model_catalog.sources.models_dev].enabled` defaults to true when omitted;
- `[model_catalog.sources.models_dev].endpoint` defaults to `https://models.dev/api.json` and must be an absolute HTTP(S) URL when set;
- `[model_catalog.sources.models_dev].ttl` defaults to `24h` and must parse as a positive duration;
- `[model_catalog.sources.models_dev].timeout` defaults to `10s` and must parse as a positive duration;
- `providers.<id>.models.discovery.enabled` defaults to false unless the built-in provider declares a side-effect-free discovery path;
- `providers.<id>.models.discovery.command` and `.endpoint` are mutually exclusive per provider instance unless a provider-specific adapter documents both;
- `providers.<id>.models.discovery.timeout` defaults to the model catalog source timeout and must parse as a positive duration;
- rendered config writes only the new shape.

No compatibility aliases or fallback readers are allowed.

## ACP Session Config Options

Upgrade `github.com/coder/acp-go-sdk` to `v0.12.2` using `go get`.

Because this is a six-minor-version jump from `v0.6.3`, the implementation must first produce `.compozy/tasks/provider-model-catalog/analysis/acp-sdk-breaking-changes.md`. The audit must enumerate every changed ACP symbol used by AGH before code migration starts, including `NewSessionResponse`, model/mode fields, `SetSessionModelRequest`, `SessionId`, config option wire types, captureCaps behavior, resume/load paths, and any renamed cancellation/error fields. The task is not complete until tests prove existing create/load/resume/mode behavior still passes on the upgraded SDK.

Capture config options from:

- `session/new`
- `session/load`
- `config_option_update`

When starting a session:

1. Create or load the ACP session.
2. Capture modes, legacy model state, and `configOptions`.
3. Apply mode with existing ACP behavior.
4. Apply model via `session/set_config_option` when a model config option exists.
5. Apply reasoning effort via `session/set_config_option` when a reasoning config option exists.
6. Fall back to `session/set_model` only when config options are absent and legacy model state supports it.

Config option matching must be conservative:

- model option IDs: exact `model` first, then known ACP model option IDs discovered in tests;
- reasoning option IDs: exact `reasoning_effort`, `effort`, or provider-specific known IDs only when documented by fixtures;
- never invent reasoning levels from `supports_reasoning=true`.

## Extensibility Integration Plan

- Extension capability: add `model.source` to the manifest provide surface.
- Extension service: add `models/list` to AGH -> extension service methods.
- Host API: add `models/list`, `models/refresh`, and `models/status`.
- SDK/types: regenerate extension SDK and TypeScript types so extension authors can implement model sources.
- Hooks: no hook is added in MVP. Model discovery is an explicit source/service contract, not a hook event.
- Skills/capabilities: no reusable AGH capability artifact is introduced. This uses extension capabilities, not AGH Network capabilities.
- Bundles/registries: extension registry stores grants as it already does; model source status is daemon catalog state.
- Bridge SDK: no bridge SDK impact.

## Agent Manageability Plan

Agents can manage the catalog without web UI through:

- CLI `agh provider models list|refresh|status -o json`
- UDS parity for local agents
- HTTP parity for remote/runtime clients
- Host API methods for extensions
- HTTP-only OpenAI-compatible `GET /api/openai/v1/models` for clients that already know that shape

All status payloads include source IDs, stale markers, last refresh, next refresh, row count, and redacted last error.

## Web/Docs Impact

Web impact:

- `web/src/systems/session/hooks/use-session-create-dialog.ts`
- `web/src/systems/session/components/session-create-dialog.tsx`
- `web/src/routes/_app/settings/providers.tsx`
- `web/src/hooks/routes/use-settings-providers-page.ts`
- `web/src/systems/settings/*` fixtures, adapters, schemas, and tests
- generated `web/src/generated/agh-openapi.d.ts`
- E2E fixture data that still uses `default_model`, `supported_models`, or `supports_reasoning_effort`

Docs/site impact:

- regenerate `openapi/agh.json`;
- regenerate web TypeScript contract types;
- regenerate CLI docs with `make cli-docs`;
- update runtime provider configuration docs in `packages/site/content/runtime/`;
- document `/api/openai/v1/models` projection and native model catalog endpoints;
- document extension `model.source` contract in extension authoring docs.

## Safety Invariants

1. Session creation never depends on successful network model discovery.
2. Discovery must not create, load, mutate, or stop ACP sessions.
3. Live discovery uses the provider's effective auth/home/env policy and explicit timeouts.
4. Source refresh failure records source status and preserves prior stale rows when available.
5. `models.dev` rows never prove account-level availability.
6. `models.curated` is never an allowlist; manual model IDs remain valid.
7. Active ACP `configOptions` override catalog metadata for that session only.
8. Global catalog rows are only written through `internal/modelcatalog.Store`.
9. Raw secrets, API keys, OAuth data, and provider credential material never appear in source errors, logs, status payloads, SSE, web UI, or Host API responses.
10. SQLite schema changes append a new migration at the registry tail and include fresh DB plus reopen-after-restart tests.
11. HTTP/UDS request lifetime does not own background refresh lifetime; refresh work uses `context.WithoutCancel(ctx)` and re-attaches an explicit deadline via `context.WithDeadline`. Deadlines must not be inherited from the request context by accident.
12. Live refresh work is serialized/coalesced per `provider_id` before touching operator `HOME`, native CLI auth state, provider cache files, or SQLite source replacement.
13. Partial-source success is success; the service fails list requests only when every usable source fails and no stale cache exists.

## Implementation Plan

1. Config hard cut:
   - replace flat provider model fields with `ProviderModelsConfig`;
   - add `ModelCatalogConfig` and `ProviderModelsDiscoveryConfig`;
   - update built-in providers;
   - update merge/clone/validate/render logic;
   - add hard-cut errors for old keys;
   - co-ship settings/API contract removals, generated OpenAPI/TypeScript drift, and web settings consumers that would otherwise keep old-field references alive.

2. Catalog persistence:
   - append global DB migration;
   - implement store methods;
   - add fresh DB and reopen-after-restart tests.

3. Catalog service:
   - implement source interface, merge policy, status model, TTL/stale handling;
   - implement deterministic priority/freshness/source-id tie-breaks;
   - implement merged availability states;
   - implement per-provider refresh serialization/coalescing;
   - wire service in `internal/daemon` composition root.

4. Sources:
   - implement builtin/config sources;
   - implement `models.dev` source with parser aliases and HTTP timeout;
   - honor `[model_catalog.sources.models_dev]` enabled/endpoint/TTL/timeout config;
   - implement live provider sources and adapter-config paths for OpenClaw/Hermes/Pi;
   - implement extension source adapter.

5. ACP SDK/config options:
   - audit `v0.6.3` to `v0.12.2` SDK breaking changes in `analysis/acp-sdk-breaking-changes.md`;
   - upgrade ACP SDK via `go get github.com/coder/acp-go-sdk@v0.12.2`;
   - capture config options;
   - prefer `session/set_config_option`;
   - keep legacy model fallback covered.

6. Public surfaces:
   - add contract payloads;
   - add HTTP/UDS handlers;
   - add HTTP-only OpenAI-compatible `/api/openai/v1/models` with API auth and OpenAI-shaped errors;
   - add CLI commands;
   - add Host API and extension service contracts;
   - run codegen.

7. Web:
   - add model catalog API/query system;
   - update new session dialog;
   - update provider settings editor;
   - update fixtures and tests.

8. Docs and verification:
   - update site docs and generated CLI docs;
   - run focused tests;
   - run `make verify`.

## Testing Approach

Config:

- `go test ./internal/config`
- Assert old flat keys fail validation.
- Assert new nested fields parse, merge, clone, validate, and render.
- Assert manual default model outside curated list is accepted.

Persistence:

- `go test ./internal/store/globaldb`
- Assert migration creates the three model catalog tables and indexes.
- Assert fresh DB and reopen-after-restart paths.
- Assert migration registry append-only contract still passes.

Catalog service:

- `go test ./internal/modelcatalog`
- Assert priority merge, freshness/source-id tie-breaks, lower-priority enrichment, merged availability states, partial success, all-source failure, stale fallback, TTL refresh, per-provider refresh coalescing, and source status.
- Assert redaction of source errors.

Sources:

- `go test ./internal/modelcatalog/...`
- Use `httptest` for `models.dev`.
- Use fake subprocesses or fake HTTP servers for live provider sources.
- Assert current `models.dev` fields and legacy aliases both parse.
- Assert no discovery source calls ACP session creation.

ACP:

- `go test ./internal/acp`
- Assert upgraded SDK compiles and existing create/load behavior stays covered.
- Assert `configOptions` captured on new/load/update fixtures.
- Assert `session/set_config_option` is preferred for model/reasoning.
- Assert legacy `session/set_model` fallback remains only when config options are absent.

API/CLI/extension:

- `go test ./internal/api/...`
- `go test ./internal/cli`
- `go test ./internal/extension/...`
- Assert HTTP and UDS parity.
- Assert CLI JSON output.
- Assert Host API capability checks and extension service method mapping.
- Assert `/api/openai/v1/models` projection shape, auth, HTTP-only registration, OpenAI-shaped errors, and provider filter.
- Assert HTTP and UDS canonical JSON byte equality for at least one native catalog payload after deterministic sorting.

Web:

- `make bun-typecheck`
- `make bun-test`
- targeted tests for new session dialog, provider settings, query hooks, stale/error/manual states.

Codegen and final gate:

- `make codegen`
- `make codegen-check`
- `make verify`

Real provider discovery tests are opt-in with explicit env/tags and are not part of `make verify`.

## Observability

Emit structured logs and events for:

- catalog refresh started/succeeded/failed;
- source row count changes;
- stale fallback usage;
- all-source failure;
- extension source denied/unavailable;
- ACP config option captured/updated.

Required correlation keys:

- `refresh_request_id` for refresh-lifecycle events
- `provider_id`
- `source_id`
- `source_kind`
- `model_id` when row-scoped
- `session_id` only for ACP session config observations
- `extension_name` for extension sources

## Risks and Mitigations

- Provider APIs differ widely. Mitigation: source interface with source status, timeouts, and fake-source tests.
- `models.dev` schema can drift. Mitigation: tolerant parser and tests for current plus legacy field names.
- UI may imply unavailable models are usable. Mitigation: tri-state `available` and explicit stale/source labels.
- ACP config option shapes may differ by agent. Mitigation: conservative exact matching and fixtures from upgraded SDK behavior.
- Source refresh can become slow. Mitigation: lazy/manual refresh, detached context with deadline, stale fallback.
- Concurrent refresh can touch the same native provider home. Mitigation: per-provider serialization/coalescing before subprocess or provider-home work.

## Assumptions and Defaults

- The implementation is a hard cut because AGH is greenfield alpha.
- `models.curated` is metadata and selection UX, not authorization.
- Manual model entry is always valid.
- Default source TTL for `models.dev` is 24h.
- Default `models.dev` source endpoint is `https://models.dev/api.json`; default timeout is 10s; operators can disable or override the source.
- Catalog list calls return cached rows immediately when present.
- Refresh calls return source status, even on partial failure.
- Droid is out of scope for v1.

## Architecture Decision Records

- `adrs/adr-001-daemon-owned-provider-model-catalog.md`
- `adrs/adr-002-provider-model-config-hard-cut.md`
- `adrs/adr-003-extension-model-source-contract.md`
