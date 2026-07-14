package store

import (
	"fmt"
	"strings"
	"time"
)

const NetworkWakeUsageReserved = "reserved"

// NetworkUsageQuery scopes wake accounting to one authenticated workspace.
type NetworkUsageQuery struct {
	WorkspaceID string
	Channel     string
	RunID       string
	OwnerKey    string
}

// Validate rejects ambiguous or unscoped usage reads.
func (q NetworkUsageQuery) Validate() error {
	if err := requireField(q.WorkspaceID, "network usage workspace_id"); err != nil {
		return err
	}
	if strings.TrimSpace(q.RunID) != "" && strings.TrimSpace(q.OwnerKey) != "" {
		return fmt.Errorf("store: network usage run_id and owner_key are mutually exclusive")
	}
	return nil
}

// NetworkWakeUsageDetail is the charged usage truth for one admitted wake.
type NetworkWakeUsageDetail struct {
	WakeID          string
	TaskRunID       string
	OwnerKey        string
	WorkspaceID     string
	Channel         string
	RootID          string
	Depth           int
	State           string
	UsageState      string
	ChargedWallTime time.Duration
	InputTokens     int64
	OutputTokens    int64
	ReservedAt      time.Time
	SettledAt       *time.Time
	Reason          string
}

// NetworkUsageSummary aggregates exactly the returned wake details.
type NetworkUsageSummary struct {
	WakeCount            int
	ReservedWakeCount    int
	ActualWakeCount      int
	UnavailableWakeCount int
	ChargedWallTime      time.Duration
	InputTokens          int64
	OutputTokens         int64
}

// NetworkUsageReport pairs per-wake evidence with its exact aggregate.
type NetworkUsageReport struct {
	Details []NetworkWakeUsageDetail
	Total   NetworkUsageSummary
}
