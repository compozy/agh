package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	eventspkg "github.com/compozy/agh/internal/events"
)

type EventSummary struct {
	ID          string
	SessionID   string
	WorkspaceID string
	Sequence    int64
	Type        string
	AgentName   string
	Provider    string
	Outcome     string
	Content     json.RawMessage
	EventCorrelation
	ParentSessionID string
	RootSessionID   string
	SpawnDepth      int
	Summary         string
	Timestamp       time.Time
}

// Validate ensures the summary contains the required identifying fields.
func (s EventSummary) Validate() error {
	eventType := strings.TrimSpace(s.Type)
	if err := requireField(eventType, "event summary type"); err != nil {
		return err
	}
	if err := eventspkg.ValidatePublicName(eventType); err != nil {
		return fmt.Errorf("store: invalid event summary type: %w", err)
	}
	if !eventspkg.ValidOutcome(s.Outcome) {
		return fmt.Errorf("store: invalid event summary outcome %q", s.Outcome)
	}
	if eventSummaryAllowsGlobalScope(eventType) {
		return nil
	}
	if err := requireField(s.WorkspaceID, "event summary workspace_id"); err != nil {
		return err
	}
	if err := requireField(s.SessionID, "event summary session id"); err != nil {
		return err
	}
	if err := requireField(s.AgentName, "event summary agent name"); err != nil {
		return err
	}
	return nil
}

func cloneNormalizedTimestamp(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func eventSummaryAllowsGlobalScope(eventType string) bool {
	return eventspkg.AllowsGlobalScope(eventType)
}

// EventSummaryQuery filters global event summary queries.
type EventSummaryQuery struct {
	SessionID     string
	WorkspaceID   string
	AgentName     string
	Type          string
	TaskID        string
	RunID         string
	ActorKind     string
	ActorID       string
	Provider      string
	Outcome       string
	Component     string
	ErrorOnly     bool
	AfterSequence int64
	Since         time.Time
	Limit         int
}

// Validate ensures the query uses sane bounds.
func (q EventSummaryQuery) Validate() error {
	if err := requirePositiveLimit(q.Limit, "event summary limit"); err != nil {
		return err
	}
	if q.AfterSequence < 0 {
		return fmt.Errorf("store: invalid event summary after sequence %d", q.AfterSequence)
	}
	if !eventspkg.ValidOutcome(q.Outcome) {
		return fmt.Errorf("store: invalid event summary outcome %q", q.Outcome)
	}
	if !eventspkg.ValidComponent(q.Component) {
		return fmt.Errorf("store: invalid event summary component %q", q.Component)
	}
	return nil
}

// ObservabilityRetentionSweepResult reports how many global observability rows
// were deleted by one retention sweep.
type ObservabilityRetentionSweepResult struct {
	CutoffAt              time.Time
	DeletedEventSummaries int64
	DeletedTokenStats     int64
	DeletedPermissionLogs int64
}

// TokenStats is the aggregated usage record for a session in the global database.
type TokenStats struct {
	ID           string
	SessionID    string
	AgentName    string
	InputTokens  *int64
	OutputTokens *int64
	TotalTokens  *int64
	TotalCost    *float64
	CostCurrency *string
	CostStatus   string
	CostSource   string
	TurnCount    int64
	UpdatedAt    time.Time
}

// TokenStatsUpdate adds one or more turns of usage into a session aggregate.
type TokenStatsUpdate struct {
	SessionID    string
	AgentName    string
	InputTokens  *int64
	OutputTokens *int64
	TotalTokens  *int64
	CostAmount   *float64
	CostCurrency *string
	CostStatus   string
	CostSource   string
	Turns        int64
	UpdatedAt    time.Time
}

// Validate ensures the aggregate update contains the required identifying fields.
func (u TokenStatsUpdate) Validate() error {
	if err := requireField(u.SessionID, "token stats session id"); err != nil {
		return err
	}
	if err := requireField(u.AgentName, "token stats agent name"); err != nil {
		return err
	}
	return validateTokenStatsCost(u)
}

func validateTokenStatsCost(update TokenStatsUpdate) error {
	status := strings.TrimSpace(update.CostStatus)
	source := strings.TrimSpace(update.CostSource)
	switch status {
	case tokenCostStatusActual:
		if source != tokenCostSourceAgentReported ||
			!validTokenStatsMoney(update.CostAmount, update.CostCurrency) {
			return errors.New(
				"store: actual token cost requires agent_reported amount and currency; " +
					"amount must be finite and non-negative",
			)
		}
	case tokenCostStatusEstimated:
		if source != tokenCostSourceCatalogConfig &&
			source != tokenCostSourceModelsDev && source != tokenCostSourceBuiltin {
			return errors.New("store: estimated token cost requires a catalog source")
		}
		if !validTokenStatsMoney(update.CostAmount, update.CostCurrency) {
			return errors.New(
				"store: estimated token cost requires amount and currency; " +
					"amount must be finite and non-negative",
			)
		}
	case tokenCostStatusIncluded:
		if source != tokenCostSourceNone || update.CostAmount != nil || update.CostCurrency != nil {
			return errors.New("store: included token cost cannot carry amount or currency")
		}
	case tokenCostStatusUnknown:
		if source != tokenCostSourceNone || update.CostAmount != nil || update.CostCurrency != nil {
			return errors.New("store: unknown token cost cannot carry amount or currency")
		}
	default:
		return fmt.Errorf("store: invalid token cost status %q", update.CostStatus)
	}
	return nil
}

func validTokenStatsMoney(amount *float64, currency *string) bool {
	return amount != nil && *amount >= 0 && !math.IsNaN(*amount) && !math.IsInf(*amount, 0) &&
		currency != nil && strings.TrimSpace(*currency) != ""
}

// TokenStatsQuery filters token aggregation lookups.
type TokenStatsQuery struct {
	SessionID string
	AgentName string
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q TokenStatsQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "token stats limit")
}

// PermissionLogEntry is an audit log entry for a daemon permission decision.
type PermissionLogEntry struct {
	ID         string
	SessionID  string
	AgentName  string
	Action     string
	Resource   string
	Decision   string
	PolicyUsed string
	Timestamp  time.Time
}

// Validate ensures the permission audit entry is complete.
func (e PermissionLogEntry) Validate() error {
	if err := requireField(e.SessionID, "permission log session id"); err != nil {
		return err
	}
	if err := requireField(e.AgentName, "permission log agent name"); err != nil {
		return err
	}
	if err := requireField(e.Action, "permission log action"); err != nil {
		return err
	}
	if err := requireField(e.Resource, "permission log resource"); err != nil {
		return err
	}
	if err := requireField(e.Decision, "permission log decision"); err != nil {
		return err
	}
	if err := requireField(e.PolicyUsed, "permission log policy"); err != nil {
		return err
	}
	return nil
}

// PermissionLogQuery filters permission audit queries.
type PermissionLogQuery struct {
	SessionID string
	AgentName string
	Decision  string
	Since     time.Time
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q PermissionLogQuery) Validate() error {
	return requirePositiveLimit(q.Limit, "permission log limit")
}

// NetworkAuditEntry is an audit row for one network message event.
