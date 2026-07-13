package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

type NetworkTaskThreadOrigin struct {
	TaskID           string
	WorkspaceID      string
	Channel          string
	ThreadID         string
	OriginMessageID  string
	Digest           string
	SourceMessageIDs []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NetworkTaskThreadOriginQuery filters promoted task origin links.
type NetworkTaskThreadOriginQuery struct {
	TaskID      string
	WorkspaceID string
	Channel     string
	ThreadID    string
	Limit       int
}

// Validate ensures origin reads are either task-specific or thread-specific.
func (q NetworkTaskThreadOriginQuery) Validate() error {
	if strings.TrimSpace(q.TaskID) != "" {
		return requirePositiveLimit(q.Limit, "network task thread origin limit")
	}
	if err := (NetworkConversationRef{
		WorkspaceID: q.WorkspaceID,
		Channel:     q.Channel,
		Surface:     NetworkSurfaceThread,
		ThreadID:    q.ThreadID,
	}).Validate(); err != nil {
		return err
	}
	return requirePositiveLimit(q.Limit, "network task thread origin limit")
}

// Validate ensures the origin is thread-scoped and compact.
func (o NetworkTaskThreadOrigin) Validate() error {
	if err := requireField(o.TaskID, "network task thread origin task_id"); err != nil {
		return err
	}
	if err := (NetworkConversationRef{
		WorkspaceID: o.WorkspaceID,
		Channel:     o.Channel,
		Surface:     NetworkSurfaceThread,
		ThreadID:    o.ThreadID,
	}).Validate(); err != nil {
		return err
	}
	if err := requireField(o.OriginMessageID, "network task thread origin origin_message_id"); err != nil {
		return err
	}
	if err := requireField(o.Digest, "network task thread origin digest"); err != nil {
		return err
	}
	for _, messageID := range o.SourceMessageIDs {
		if err := requireField(messageID, "network task thread origin source_message_id"); err != nil {
			return err
		}
	}
	return nil
}

// TaskDesignationRollup stores the terminal summary for a designated run group.
type TaskDesignationRollup struct {
	DesignationGroupID string
	TaskID             string
	SummaryJSON        json.RawMessage
	CreatedAt          time.Time
}

// TaskDesignationRollupQuery filters designation rollup reads.
type TaskDesignationRollupQuery struct {
	DesignationGroupID string
	TaskID             string
	Limit              int
}

// Validate ensures rollup reads cannot become unbounded.
func (q TaskDesignationRollupQuery) Validate() error {
	if strings.TrimSpace(q.DesignationGroupID) == "" && strings.TrimSpace(q.TaskID) == "" {
		return fmt.Errorf("store: task designation rollup query requires group_id or task_id")
	}
	return requirePositiveLimit(q.Limit, "task designation rollup limit")
}

// Validate ensures the rollup can be retrieved by task and group.
func (r TaskDesignationRollup) Validate() error {
	if err := requireField(r.DesignationGroupID, "task designation rollup group_id"); err != nil {
		return err
	}
	if err := requireField(r.TaskID, "task designation rollup task_id"); err != nil {
		return err
	}
	if len(r.SummaryJSON) == 0 || !json.Valid(r.SummaryJSON) {
		return fmt.Errorf("store: task designation rollup summary_json must be valid JSON")
	}
	return nil
}

// NormalizeNetworkDirectRoomPeers validates and orders a two-party room pair.
func NormalizeNetworkDirectRoomPeers(peerA string, peerB string) (string, string, error) {
	first := strings.TrimSpace(peerA)
	second := strings.TrimSpace(peerB)
	if err := validateNetworkPeerID(first, "peer_a"); err != nil {
		return "", "", err
	}
	if err := validateNetworkPeerID(second, "peer_b"); err != nil {
		return "", "", err
	}
	if first == second {
		return "", "", fmt.Errorf("store: network direct room peers must differ")
	}
	if second < first {
		first, second = second, first
	}
	return first, second, nil
}

// NetworkDirectRoomIdentity derives the stable direct-room id for one ordered peer pair.
func NetworkDirectRoomIdentity(
	workspaceID string,
	channel string,
	peerA string,
	peerB string,
) (string, string, string, error) {
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if err := requireField(trimmedWorkspaceID, "network direct room workspace_id"); err != nil {
		return "", "", "", err
	}
	trimmedChannel := strings.TrimSpace(channel)
	if err := requireField(trimmedChannel, "network direct room channel"); err != nil {
		return "", "", "", err
	}
	normalizedA, normalizedB, err := NormalizeNetworkDirectRoomPeers(peerA, peerB)
	if err != nil {
		return "", "", "", err
	}
	sum := sha256.Sum256([]byte(
		"agh-network/direct-room/v0\x00" + trimmedWorkspaceID + "\x00" + trimmedChannel + "\x00" +
			normalizedA + "\x00" + normalizedB,
	))
	return "direct_" + hex.EncodeToString(sum[:])[:32], normalizedA, normalizedB, nil
}

func validateOptionalNetworkConversation(
	workspaceID string,
	channel string,
	surface string,
	threadID string,
	directID string,
	label string,
) error {
	if strings.TrimSpace(surface) == "" && strings.TrimSpace(threadID) == "" && strings.TrimSpace(directID) == "" {
		return nil
	}
	if err := (NetworkConversationRef{
		WorkspaceID: workspaceID,
		Channel:     channel,
		Surface:     surface,
		ThreadID:    threadID,
		DirectID:    directID,
	}).Validate(); err != nil {
		return fmt.Errorf("store: invalid %s: %w", label, err)
	}
	return nil
}

func validateNetworkDirectRoom(workspaceID string, channel string, directID string, peerA string, peerB string) error {
	ref := NetworkConversationRef{
		WorkspaceID: workspaceID,
		Channel:     channel,
		Surface:     NetworkSurfaceDirect,
		DirectID:    directID,
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	normalizedA, normalizedB, err := NormalizeNetworkDirectRoomPeers(peerA, peerB)
	if err != nil {
		return err
	}
	if normalizedA != strings.TrimSpace(peerA) || normalizedB != strings.TrimSpace(peerB) {
		return fmt.Errorf("store: network direct room peers must be stored in lexicographic order")
	}
	return nil
}

func validateNetworkMessageKind(kind string) error {
	switch strings.TrimSpace(kind) {
	case NetworkKindGreet,
		NetworkKindWhois,
		NetworkKindSay,
		NetworkKindCapability,
		NetworkKindReceipt,
		NetworkKindTrace:
		return nil
	default:
		return fmt.Errorf("store: unsupported network message kind %q", kind)
	}
}

func validateNetworkWorkState(state string) error {
	switch strings.TrimSpace(state) {
	case NetworkWorkStateSubmitted,
		NetworkWorkStateWorking,
		NetworkWorkStateNeedsInput,
		NetworkWorkStateCompleted,
		NetworkWorkStateFailed,
		NetworkWorkStateCanceled:
		return nil
	default:
		return fmt.Errorf("store: unsupported network work state %q", state)
	}
}

func networkWorkStateIsTerminal(state string) bool {
	switch strings.TrimSpace(state) {
	case NetworkWorkStateCompleted, NetworkWorkStateFailed, NetworkWorkStateCanceled:
		return true
	default:
		return false
	}
}

func validateNetworkConversationID(id string, field string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fmt.Errorf("store: network %s is required", field)
	}
	switch field {
	case typesThreadIDKey:
		if !networkThreadIDPattern.MatchString(trimmed) {
			return fmt.Errorf("store: invalid network thread_id %q", id)
		}
	case typesDirectIDKey:
		if !networkDirectIDPattern.MatchString(trimmed) {
			return fmt.Errorf("store: invalid network direct_id %q", id)
		}
	default:
		if len(trimmed) > 128 || strings.ContainsAny(trimmed, `/\`) || containsControlCharacter(trimmed) {
			return fmt.Errorf("store: invalid network %s %q", field, id)
		}
	}
	return nil
}

func validateNetworkPeerID(peerID string, field string) error {
	trimmed := strings.TrimSpace(peerID)
	if !networkPeerIDPattern.MatchString(trimmed) {
		return fmt.Errorf("store: invalid network %s %q", field, peerID)
	}
	return nil
}

// NormalizeNetworkPeerIDs trims, validates, deduplicates, and sorts peer ids.
func NormalizeNetworkPeerIDs(values []string, field string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if err := validateNetworkPeerID(trimmed, field); err != nil {
			return nil, err
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	slices.Sort(normalized)
	return normalized, nil
}

func containsControlCharacter(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func networkAuditEntryContainsRawClaimToken(entry NetworkAuditEntry) bool {
	values := []string{
		entry.ID,
		entry.SessionID,
		entry.Direction,
		entry.Kind,
		entry.Channel,
		entry.Surface,
		entry.ThreadID,
		entry.DirectID,
		entry.WorkID,
		entry.PeerFrom,
		entry.PeerTo,
		entry.MessageID,
		entry.Reason,
	}
	return slices.ContainsFunc(values, networkStringContainsRawClaimToken)
}

func networkConversationMessageContainsRawClaimToken(entry NetworkConversationMessage) bool {
	values := []string{
		entry.MessageID,
		entry.SessionID,
		entry.Channel,
		entry.Surface,
		entry.ThreadID,
		entry.DirectID,
		entry.Direction,
		entry.PeerFrom,
		entry.PeerTo,
		strings.Join(entry.Mentions, ","),
		entry.Kind,
		entry.WorkID,
		entry.ReplyTo,
		entry.TraceID,
		entry.CausationID,
		entry.Intent,
		entry.Text,
		entry.PreviewText,
	}
	return slices.ContainsFunc(values, networkStringContainsRawClaimToken)
}

func networkRawJSONContainsClaimToken(raw json.RawMessage) bool {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return networkStringContainsRawClaimToken(string(raw))
	}
	return networkValueContainsClaimToken("", value)
}

func networkValueContainsClaimToken(key string, value any) bool {
	if networkStringContainsRawClaimToken(key) || networkClaimTokenKeyHasValue(key, value) {
		return true
	}
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return networkStringContainsRawClaimToken(typed)
	case []any:
		for _, item := range typed {
			if networkValueContainsClaimToken("", item) {
				return true
			}
		}
	case map[string]any:
		for nestedKey, nestedValue := range typed {
			if networkValueContainsClaimToken(nestedKey, nestedValue) {
				return true
			}
		}
	}
	return false
}

func networkClaimTokenKeyHasValue(key string, value any) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	if normalized != "claimtoken" {
		return false
	}
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func networkStringContainsRawClaimToken(value string) bool {
	return strings.Contains(strings.TrimSpace(value), "agh_claim_")
}

// ReconcileResult reports which sessions were indexed or marked orphaned.
