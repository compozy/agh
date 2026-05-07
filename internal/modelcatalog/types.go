package modelcatalog

import (
	"context"
	"time"
)

// SourceKind identifies the provenance family for a catalog source row.
type SourceKind string

const (
	// SourceKindBuiltin identifies AGH's offline bootstrap catalog.
	SourceKindBuiltin SourceKind = "builtin"
	// SourceKindConfig identifies operator-authored provider model config.
	SourceKindConfig SourceKind = "config"
	// SourceKindModelsDev identifies enrichment from models.dev.
	SourceKindModelsDev SourceKind = "models_dev"
	// SourceKindProviderLive identifies live provider discovery.
	SourceKindProviderLive SourceKind = "provider_live"
	// SourceKindExtension identifies extension-provided model source rows.
	SourceKindExtension SourceKind = "extension"
	// SourceKindACPSession identifies session-scoped ACP observations.
	SourceKindACPSession SourceKind = "acp_session"
)

// ReasoningEffort identifies one normalized model reasoning level.
type ReasoningEffort string

const (
	// ReasoningEffortMinimal is the smallest supported reasoning level.
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	// ReasoningEffortLow is the low reasoning level.
	ReasoningEffortLow ReasoningEffort = "low"
	// ReasoningEffortMedium is the medium reasoning level.
	ReasoningEffortMedium ReasoningEffort = "medium"
	// ReasoningEffortHigh is the high reasoning level.
	ReasoningEffortHigh ReasoningEffort = "high"
	// ReasoningEffortXHigh is the extra-high reasoning level.
	ReasoningEffortXHigh ReasoningEffort = "xhigh"
)

// RefreshState identifies one source refresh lifecycle state.
type RefreshState string

const (
	// RefreshStateIdle indicates a source has no active refresh state.
	RefreshStateIdle RefreshState = "idle"
	// RefreshStateRefreshing indicates a source refresh is currently running.
	RefreshStateRefreshing RefreshState = "refreshing"
	// RefreshStateSucceeded indicates the last source refresh succeeded.
	RefreshStateSucceeded RefreshState = "succeeded"
	// RefreshStateFailed indicates the last source refresh failed.
	RefreshStateFailed RefreshState = "failed"
)

// ListOptions filters persisted catalog source rows.
type ListOptions struct {
	ProviderID   string
	SourceID     string
	Refresh      bool
	IncludeAll   bool
	IncludeStale bool
	Now          time.Time
}

// RefreshOptions controls a model catalog refresh request.
type RefreshOptions struct {
	ProviderID string
	SourceID   string
	Force      bool
	RequestID  string
	Now        time.Time
}

// ModelRow is one provider/model record contributed by one catalog source.
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

// SourceRef identifies one source participating in a merged catalog projection.
type SourceRef struct {
	SourceID    string
	SourceKind  SourceKind
	Priority    int
	RefreshedAt time.Time
	Stale       bool
	LastError   string
}

// Model is the deterministic merged projection for one provider/model key.
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

// SourceStatus reports provider-scoped source health and row counts.
type SourceStatus struct {
	SourceID     string
	SourceKind   SourceKind
	ProviderID   string
	Priority     int
	LastRefresh  time.Time
	NextRefresh  time.Time
	LastSuccess  time.Time
	LastError    string
	RefreshState string
	RowCount     int
	Stale        bool
}

// Source produces model rows for one catalog source.
type Source interface {
	ID() string
	Kind() SourceKind
	Priority() int
	ListModels(ctx context.Context, opts ListOptions) ([]ModelRow, error)
}

// Store persists source rows and provider-scoped source status.
type Store interface {
	ReplaceSourceRows(
		ctx context.Context,
		sourceID string,
		providerID string,
		rows []ModelRow,
		status SourceStatus,
	) error
	ListRows(ctx context.Context, opts ListOptions) ([]ModelRow, error)
	ListSourceStatus(ctx context.Context, providerID string) ([]SourceStatus, error)
}

// Service exposes merged model catalog projections.
type Service interface {
	ListModels(ctx context.Context, opts ListOptions) ([]Model, error)
	Refresh(ctx context.Context, opts RefreshOptions) ([]SourceStatus, error)
	ListSourceStatus(ctx context.Context, providerID string) ([]SourceStatus, error)
}
