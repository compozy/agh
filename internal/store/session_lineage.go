package store

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// SessionLineage is the persisted parent/root metadata used for safe spawned sessions.
type SessionLineage struct {
	ParentSessionID  string                  `json:"parent_session_id,omitempty"`
	RootSessionID    string                  `json:"root_session_id,omitempty"`
	SpawnDepth       int                     `json:"spawn_depth"`
	SpawnRole        string                  `json:"spawn_role,omitempty"`
	TTLExpiresAt     *time.Time              `json:"ttl_expires_at,omitempty"`
	AutoStopOnParent bool                    `json:"auto_stop_on_parent"`
	SpawnBudget      SessionSpawnBudget      `json:"spawn_budget"`
	PermissionPolicy SessionPermissionPolicy `json:"permission_policy"`
}

// SessionSpawnBudget captures durable spawn limits attached to a session.
type SessionSpawnBudget struct {
	MaxChildren           int   `json:"max_children"`
	MaxDepth              int   `json:"max_depth"`
	TTLSeconds            int64 `json:"ttl_seconds"`
	MaxActivePerWorkspace int   `json:"max_active_per_workspace,omitempty"`
}

// SessionPermissionPolicy captures concrete permission atoms available to a session.
type SessionPermissionPolicy struct {
	Tools           []string `json:"tools"`
	Skills          []string `json:"skills"`
	MCPServers      []string `json:"mcp_servers"`
	WorkspacePaths  []string `json:"workspace_paths"`
	NetworkChannels []string `json:"network_channels"`
	SandboxProfiles []string `json:"sandbox_profiles"`
}

// CloneSessionLineage returns a deep copy of lineage metadata.
func CloneSessionLineage(lineage *SessionLineage) *SessionLineage {
	if lineage == nil {
		return nil
	}
	cloned := *lineage
	if lineage.TTLExpiresAt != nil {
		ttl := lineage.TTLExpiresAt.UTC()
		cloned.TTLExpiresAt = &ttl
	}
	cloned.PermissionPolicy = NormalizeSessionPermissionPolicy(lineage.PermissionPolicy)
	return &cloned
}

// NormalizeSessionLineage returns lineage with trimmed identifiers and a root
// record for first-class manual sessions when no lineage was supplied.
func NormalizeSessionLineage(sessionID string, lineage *SessionLineage) *SessionLineage {
	normalized := SessionLineage{}
	if lineage != nil {
		normalized = *lineage
	}
	normalized.ParentSessionID = strings.TrimSpace(normalized.ParentSessionID)
	normalized.RootSessionID = strings.TrimSpace(normalized.RootSessionID)
	normalized.SpawnRole = strings.TrimSpace(normalized.SpawnRole)
	if normalized.TTLExpiresAt != nil {
		ttl := normalized.TTLExpiresAt.UTC()
		normalized.TTLExpiresAt = &ttl
	}
	normalized.PermissionPolicy = NormalizeSessionPermissionPolicy(normalized.PermissionPolicy)

	if normalized.ParentSessionID == "" && normalized.RootSessionID == "" {
		normalized.RootSessionID = strings.TrimSpace(sessionID)
	}
	return &normalized
}

// ValidateSessionLineage ensures lineage is structurally usable by spawn policy enforcement.
func ValidateSessionLineage(sessionID string, lineage *SessionLineage) error {
	if lineage == nil {
		return nil
	}
	normalized := NormalizeSessionLineage(sessionID, lineage)
	if normalized.SpawnDepth < 0 {
		return fmt.Errorf("store: session lineage spawn depth cannot be negative")
	}
	if err := validateSessionSpawnBudget(normalized.SpawnBudget); err != nil {
		return err
	}
	if err := validateSessionPermissionPolicy(normalized.PermissionPolicy); err != nil {
		return err
	}

	sessionID = strings.TrimSpace(sessionID)
	switch {
	case normalized.ParentSessionID == "":
		if normalized.SpawnDepth != 0 {
			return fmt.Errorf("store: root session lineage depth must be 0")
		}
		if normalized.RootSessionID != "" && sessionID != "" && normalized.RootSessionID != sessionID {
			return fmt.Errorf("store: root session lineage root must match session id")
		}
		if normalized.AutoStopOnParent {
			return fmt.Errorf("store: root session lineage cannot auto-stop on parent")
		}
	case normalized.RootSessionID == "":
		return fmt.Errorf("store: child session lineage root session id is required")
	case normalized.SpawnDepth == 0:
		return fmt.Errorf("store: child session lineage depth must be greater than 0")
	case normalized.ParentSessionID == sessionID:
		return fmt.Errorf("store: child session lineage parent cannot be the session itself")
	case normalized.RootSessionID == sessionID:
		return fmt.Errorf("store: child session lineage root cannot be the session itself")
	}
	return nil
}

// NormalizeSessionPermissionPolicy returns a policy with stable, trimmed atom lists.
func NormalizeSessionPermissionPolicy(policy SessionPermissionPolicy) SessionPermissionPolicy {
	return SessionPermissionPolicy{
		Tools:           normalizePolicyAtoms(policy.Tools),
		Skills:          normalizePolicyAtoms(policy.Skills),
		MCPServers:      normalizePolicyAtoms(policy.MCPServers),
		WorkspacePaths:  normalizePolicyAtoms(policy.WorkspacePaths),
		NetworkChannels: normalizePolicyAtoms(policy.NetworkChannels),
		SandboxProfiles: normalizePolicyAtoms(policy.SandboxProfiles),
	}
}

func validateSessionSpawnBudget(budget SessionSpawnBudget) error {
	switch {
	case budget.MaxChildren < 0:
		return fmt.Errorf("store: session spawn budget max_children cannot be negative")
	case budget.MaxDepth < 0:
		return fmt.Errorf("store: session spawn budget max_depth cannot be negative")
	case budget.TTLSeconds < 0:
		return fmt.Errorf("store: session spawn budget ttl_seconds cannot be negative")
	case budget.MaxActivePerWorkspace < 0:
		return fmt.Errorf("store: session spawn budget max_active_per_workspace cannot be negative")
	default:
		return nil
	}
}

func validateSessionPermissionPolicy(policy SessionPermissionPolicy) error {
	checks := []struct {
		name   string
		values []string
	}{
		{name: "tools", values: policy.Tools},
		{name: "skills", values: policy.Skills},
		{name: "mcp_servers", values: policy.MCPServers},
		{name: "workspace_paths", values: policy.WorkspacePaths},
		{name: "network_channels", values: policy.NetworkChannels},
		{name: "sandbox_profiles", values: policy.SandboxProfiles},
	}
	for _, check := range checks {
		for _, value := range check.values {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("store: session permission policy %s contains an empty atom", check.name)
			}
		}
	}
	return nil
}

func normalizePolicyAtoms(values []string) []string {
	if values == nil {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, strings.TrimSpace(value))
	}
	slices.Sort(normalized)
	return slices.Compact(normalized)
}

// EncodeSessionSpawnBudget marshals budget metadata for the global session catalog.
func EncodeSessionSpawnBudget(budget SessionSpawnBudget) (string, error) {
	if err := validateSessionSpawnBudget(budget); err != nil {
		return "", err
	}
	data, err := json.Marshal(budget)
	if err != nil {
		return "", fmt.Errorf("store: marshal session spawn budget: %w", err)
	}
	return string(data), nil
}

// DecodeSessionSpawnBudget unmarshals budget metadata from the global session catalog.
func DecodeSessionSpawnBudget(raw string) (SessionSpawnBudget, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return SessionSpawnBudget{}, nil
	}
	var budget SessionSpawnBudget
	if err := json.Unmarshal([]byte(trimmed), &budget); err != nil {
		return SessionSpawnBudget{}, fmt.Errorf("store: parse session spawn budget json: %w", err)
	}
	if err := validateSessionSpawnBudget(budget); err != nil {
		return SessionSpawnBudget{}, err
	}
	return budget, nil
}

// EncodeSessionPermissionPolicy marshals normalized permission policy metadata.
func EncodeSessionPermissionPolicy(policy SessionPermissionPolicy) (string, error) {
	normalized := NormalizeSessionPermissionPolicy(policy)
	if err := validateSessionPermissionPolicy(normalized); err != nil {
		return "", err
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("store: marshal session permission policy: %w", err)
	}
	return string(data), nil
}

// DecodeSessionPermissionPolicy unmarshals permission policy metadata from the global session catalog.
func DecodeSessionPermissionPolicy(raw string) (SessionPermissionPolicy, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return NormalizeSessionPermissionPolicy(SessionPermissionPolicy{}), nil
	}
	var policy SessionPermissionPolicy
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return SessionPermissionPolicy{}, fmt.Errorf("store: parse session permission policy json: %w", err)
	}
	normalized := NormalizeSessionPermissionPolicy(policy)
	if err := validateSessionPermissionPolicy(normalized); err != nil {
		return SessionPermissionPolicy{}, err
	}
	return normalized, nil
}
