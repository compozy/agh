package store

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	typesSentKey = "sent"
)

const (
	typesDirectIDKey = "direct_id"
	typesRejectedKey = "rejected"
	typesThreadIDKey = "thread_id"
)

const (
	// NetworkSurfaceThread stores a public thread conversation container.
	NetworkSurfaceThread = "thread"
	// NetworkSurfaceDirect stores a two-party direct-room conversation container.
	NetworkSurfaceDirect = "direct"

	// NetworkFanoutPolicyCapabilityMatch activates peers by declared capability.
	NetworkFanoutPolicyCapabilityMatch = "capability_match"
	// NetworkFanoutPolicyCoordinator activates the configured channel coordinator.
	NetworkFanoutPolicyCoordinator = "coordinator"
	// NetworkFanoutPolicyAllMembers activates every eligible local channel member.
	NetworkFanoutPolicyAllMembers = "all_members"

	// NetworkSubscriptionModeMute suppresses unmentioned matching traffic.
	NetworkSubscriptionModeMute = "mute"
	// NetworkSubscriptionModeFull preserves full prompt injection.
	NetworkSubscriptionModeFull = "full"

	// NetworkKindGreet stores a presence announcement.
	NetworkKindGreet = "greet"
	// NetworkKindWhois stores a peer identity request or response.
	NetworkKindWhois = "whois"
	// NetworkKindSay stores a text conversation message.
	NetworkKindSay = "say"
	// NetworkKindCapability stores a capability transfer message.
	NetworkKindCapability = "capability"
	// NetworkKindReceipt stores an admission receipt message.
	NetworkKindReceipt = "receipt"
	// NetworkKindTrace stores a work lifecycle trace message.
	NetworkKindTrace = "trace"

	// NetworkWorkStateSubmitted is the initial work state.
	NetworkWorkStateSubmitted = "submitted"
	// NetworkWorkStateWorking marks active work.
	NetworkWorkStateWorking = "working"
	// NetworkWorkStateNeedsInput marks blocked work awaiting input.
	NetworkWorkStateNeedsInput = "needs_input"
	// NetworkWorkStateCompleted marks successful terminal work.
	NetworkWorkStateCompleted = "completed"
	// NetworkWorkStateFailed marks failed terminal work.
	NetworkWorkStateFailed = "failed"
	// NetworkWorkStateCanceled marks canceled terminal work.
	NetworkWorkStateCanceled = "canceled"
)

var (
	networkThreadIDPattern = regexp.MustCompile(`^thread_[a-z0-9][a-z0-9_-]{2,95}$`)
	networkDirectIDPattern = regexp.MustCompile(`^direct_[a-f0-9]{32}$`)
	networkPeerIDPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

type NetworkAuditEntry struct {
	ID          string
	SessionID   string
	Direction   string
	Kind        string
	WorkspaceID string
	Channel     string
	Surface     string
	ThreadID    string
	DirectID    string
	WorkID      string
	PeerFrom    string
	PeerTo      string
	MessageID   string
	Reason      string
	Size        int
	Timestamp   time.Time
}

// Validate ensures the network audit entry is complete and internally consistent.
func (e NetworkAuditEntry) Validate() error {
	if err := requireField(e.SessionID, "network audit session id"); err != nil {
		return err
	}
	if err := requireField(e.Direction, "network audit direction"); err != nil {
		return err
	}
	direction := strings.TrimSpace(e.Direction)
	switch direction {
	case typesSentKey, "received", typesRejectedKey, "delivered":
	default:
		return fmt.Errorf(
			"store: network audit direction must be one of %q, %q, %q, %q: %q",
			typesSentKey,
			"received",
			typesRejectedKey,
			"delivered",
			e.Direction,
		)
	}
	if direction != e.Direction {
		return fmt.Errorf("store: network audit direction must not contain surrounding whitespace: %q", e.Direction)
	}
	if err := requireField(e.Kind, "network audit kind"); err != nil {
		return err
	}
	if err := (NetworkChannelRef{WorkspaceID: e.WorkspaceID, Channel: e.Channel}).Validate(); err != nil {
		return err
	}
	if err := requireField(e.PeerFrom, "network audit peer_from"); err != nil {
		return err
	}
	if err := validateOptionalNetworkConversation(
		e.WorkspaceID,
		e.Channel,
		e.Surface,
		e.ThreadID,
		e.DirectID,
		"network audit conversation",
	); err != nil {
		return err
	}
	if strings.TrimSpace(e.WorkID) != "" {
		if err := validateNetworkConversationID(e.WorkID, "work_id"); err != nil {
			return err
		}
	}
	if err := requireField(e.MessageID, "network audit message id"); err != nil {
		return err
	}
	if e.Size < 0 {
		return fmt.Errorf("store: network audit size must be zero or positive: %d", e.Size)
	}
	if direction == typesRejectedKey && strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("store: network audit reason is required when direction is %q", e.Direction)
	}
	if networkAuditEntryContainsRawClaimToken(e) {
		return fmt.Errorf("store: network audit entry contains raw claim_token material")
	}
	return nil
}

// NetworkAuditQuery filters network audit lookups.
type NetworkAuditQuery struct {
	SessionID   string
	WorkspaceID string
	// Global explicitly allows daemon-admin aggregate callers to scan audit rows
	// across workspaces. Workspace-scoped API surfaces must leave this false.
	Global    bool
	Direction string
	Kind      string
	Channel   string
	Surface   string
	ThreadID  string
	DirectID  string
	WorkID    string
	MessageID string
	Since     time.Time
	Limit     int
}

// Validate ensures the query uses sane bounds.
func (q NetworkAuditQuery) Validate() error {
	workspaceID := strings.TrimSpace(q.WorkspaceID)
	if q.Global && workspaceID != "" {
		return errors.New("store: network audit query cannot combine global scan with workspace_id")
	}
	if !q.Global {
		if err := requireField(workspaceID, "network audit query workspace_id"); err != nil {
			return err
		}
	}
	return requirePositiveLimit(q.Limit, "network audit limit")
}

// NetworkChannelEntry stores durable channel metadata for the operator-facing
// network workspace.
type NetworkChannelEntry struct {
	Channel           string
	WorkspaceID       string
	Purpose           string
	FanoutPolicy      string
	CoordinatorPeerID string
	CreatedBy         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NetworkChannelPatch describes a partial channel metadata update. Nil fields
// are left unchanged by store implementations.
type NetworkChannelPatch struct {
	Purpose           *string
	FanoutPolicy      *string
	CoordinatorPeerID *string
	UpdatedAt         time.Time
}

// Apply returns the entry that would result from applying the patch.
func (p NetworkChannelPatch) Apply(entry NetworkChannelEntry) NetworkChannelEntry {
	if p.Purpose != nil {
		entry.Purpose = strings.TrimSpace(*p.Purpose)
	}
	if p.FanoutPolicy != nil {
		entry.FanoutPolicy = NormalizeNetworkFanoutPolicy(*p.FanoutPolicy)
	}
	if p.CoordinatorPeerID != nil {
		entry.CoordinatorPeerID = strings.TrimSpace(*p.CoordinatorPeerID)
	}
	if !p.UpdatedAt.IsZero() {
		entry.UpdatedAt = p.UpdatedAt.UTC()
	}
	return entry
}

// HasChanges reports whether the patch includes at least one mutable field.
func (p NetworkChannelPatch) HasChanges() bool {
	return p.Purpose != nil || p.FanoutPolicy != nil || p.CoordinatorPeerID != nil
}

// NetworkChannelRef identifies one workspace-qualified network channel.
type NetworkChannelRef struct {
	WorkspaceID string
	Channel     string
}

// Validate ensures the channel reference is workspace-qualified.
func (r NetworkChannelRef) Validate() error {
	if err := requireField(r.WorkspaceID, "network channel workspace_id"); err != nil {
		return err
	}
	if err := requireField(r.Channel, "network channel channel"); err != nil {
		return err
	}
	return nil
}

// Validate ensures the persisted channel metadata is complete.
func (e NetworkChannelEntry) Validate() error {
	if err := (NetworkChannelRef{WorkspaceID: e.WorkspaceID, Channel: e.Channel}).Validate(); err != nil {
		return err
	}
	if err := requireField(e.Purpose, "network channel purpose"); err != nil {
		return err
	}
	if err := ValidateNetworkChannelFanoutConfiguration(e.FanoutPolicy, e.CoordinatorPeerID); err != nil {
		return err
	}
	if strings.TrimSpace(e.CoordinatorPeerID) != "" {
		if err := validateNetworkPeerID(e.CoordinatorPeerID, "network channel coordinator_peer_id"); err != nil {
			return err
		}
	}
	return nil
}

// ValidateNetworkChannelFanoutConfiguration checks policy and coordinator
// coupling for channel metadata.
func ValidateNetworkChannelFanoutConfiguration(policy string, coordinatorPeerID string) error {
	normalized := NormalizeNetworkFanoutPolicy(policy)
	if err := ValidateNetworkFanoutPolicy(normalized); err != nil {
		return err
	}
	coordinator := strings.TrimSpace(coordinatorPeerID)
	if normalized == NetworkFanoutPolicyCoordinator {
		if coordinator == "" {
			return errors.New("store: network channel coordinator_peer_id is required for coordinator fanout policy")
		}
		return nil
	}
	if coordinator != "" {
		return errors.New("store: network channel coordinator_peer_id requires coordinator fanout policy")
	}
	return nil
}

// NormalizeNetworkFanoutPolicy applies the default channel activation policy.
func NormalizeNetworkFanoutPolicy(policy string) string {
	trimmed := strings.TrimSpace(policy)
	if trimmed == "" {
		return NetworkFanoutPolicyCapabilityMatch
	}
	return trimmed
}

// ValidateNetworkFanoutPolicy checks one channel activation policy value.
func ValidateNetworkFanoutPolicy(policy string) error {
	switch NormalizeNetworkFanoutPolicy(policy) {
	case NetworkFanoutPolicyCapabilityMatch, NetworkFanoutPolicyCoordinator, NetworkFanoutPolicyAllMembers:
		return nil
	default:
		return fmt.Errorf("store: unsupported network fanout policy %q", policy)
	}
}

// NetworkChannelQuery filters persisted network channel metadata lookups.
type NetworkChannelQuery struct {
	Channel     string
	WorkspaceID string
	Limit       int
}

// Validate ensures the query uses sane bounds.
func (q NetworkChannelQuery) Validate() error {
	if err := requireField(q.WorkspaceID, "network channel query workspace_id"); err != nil {
		return err
	}
	return requirePositiveLimit(q.Limit, "network channel limit")
}

// NetworkSubscriptionRef identifies one session's channel or thread delivery mode.
type NetworkSubscriptionRef struct {
	WorkspaceID string
	Channel     string
	ThreadID    string
	SessionID   string
}

// Validate ensures the subscription target is workspace-qualified.
func (r NetworkSubscriptionRef) Validate() error {
	if err := (NetworkChannelRef{WorkspaceID: r.WorkspaceID, Channel: r.Channel}).Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.ThreadID) != "" {
		if err := validateNetworkConversationID(r.ThreadID, typesThreadIDKey); err != nil {
			return err
		}
	}
	return requireField(r.SessionID, "network subscription session_id")
}

// NetworkSubscriptionEntry stores one session delivery preference.
type NetworkSubscriptionEntry struct {
	WorkspaceID string
	Channel     string
	ThreadID    string
	SessionID   string
	Mode        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate ensures one subscription row is usable by zero-token routing.
func (e NetworkSubscriptionEntry) Validate() error {
	if err := (NetworkSubscriptionRef{
		WorkspaceID: e.WorkspaceID,
		Channel:     e.Channel,
		ThreadID:    e.ThreadID,
		SessionID:   e.SessionID,
	}).Validate(); err != nil {
		return err
	}
	if err := ValidateNetworkSubscriptionMode(e.Mode); err != nil {
		return err
	}
	return nil
}

// NetworkSubscriptionQuery filters session delivery preferences.
type NetworkSubscriptionQuery struct {
	WorkspaceID string
	Channel     string
	ThreadID    string
	SessionID   string
	ExactThread bool
	Limit       int
}

// Validate ensures subscription reads remain workspace-scoped.
func (q NetworkSubscriptionQuery) Validate() error {
	if err := (NetworkChannelRef{WorkspaceID: q.WorkspaceID, Channel: q.Channel}).Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(q.ThreadID) != "" {
		if err := validateNetworkConversationID(q.ThreadID, typesThreadIDKey); err != nil {
			return err
		}
	}
	if strings.TrimSpace(q.SessionID) != "" {
		if err := requireField(q.SessionID, "network subscription session_id"); err != nil {
			return err
		}
	}
	return requirePositiveLimit(q.Limit, "network subscription limit")
}

// ValidateNetworkSubscriptionMode checks one delivery preference mode.
func ValidateNetworkSubscriptionMode(mode string) error {
	switch strings.TrimSpace(mode) {
	case NetworkSubscriptionModeMute, NetworkSubscriptionModeFull:
		return nil
	default:
		return fmt.Errorf("store: unsupported network subscription mode %q", mode)
	}
}
