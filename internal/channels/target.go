package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// DeliveryMode identifies the daemon-owned outbound delivery behavior requested
// for one canonical channel target.
type DeliveryMode string

const (
	// DeliveryModeDirectSend sends a fresh outbound message into the target conversation.
	DeliveryModeDirectSend DeliveryMode = "direct-send"
	// DeliveryModeReply sends an outbound reply within the resolved conversation context.
	DeliveryModeReply DeliveryMode = "reply"
)

// Normalize returns the canonical delivery mode representation.
func (m DeliveryMode) Normalize() DeliveryMode {
	switch normalized := strings.ToLower(strings.TrimSpace(string(m))); normalized {
	case "direct", "direct-send", "direct_send", "send":
		return DeliveryModeDirectSend
	case "reply", "reply-send", "reply_send":
		return DeliveryModeReply
	default:
		return DeliveryMode(normalized)
	}
}

// Validate reports whether the delivery mode belongs to the supported mode set.
func (m DeliveryMode) Validate() error {
	switch normalized := m.Normalize(); normalized {
	case DeliveryModeDirectSend, DeliveryModeReply:
		return nil
	case "":
		return errors.New("channels: delivery target mode is required")
	default:
		return fmt.Errorf("channels: unsupported delivery target mode %q", strings.TrimSpace(string(m)))
	}
}

// ResolveDeliveryTargetRequest captures one outbound target request before
// channel-instance defaults have been merged in.
type ResolveDeliveryTargetRequest struct {
	ChannelInstanceID string       `json:"channel_instance_id"`
	PeerID            string       `json:"peer_id,omitempty"`
	ThreadID          string       `json:"thread_id,omitempty"`
	GroupID           string       `json:"group_id,omitempty"`
	Mode              DeliveryMode `json:"mode,omitempty"`
}

// Validate reports whether the request identifies the owning channel instance.
func (r ResolveDeliveryTargetRequest) Validate() error {
	return requireField(strings.TrimSpace(r.ChannelInstanceID), "delivery target request channel instance id")
}

// TargetResolver resolves one canonical outbound delivery target from channel
// instance metadata plus explicit destination overrides.
type TargetResolver interface {
	ResolveDeliveryTarget(ctx context.Context, req ResolveDeliveryTargetRequest) (*DeliveryTarget, error)
}

var _ TargetResolver = (*Service)(nil)

type deliveryTargetDefaults struct {
	PeerID   string       `json:"peer_id,omitempty"`
	ThreadID string       `json:"thread_id,omitempty"`
	GroupID  string       `json:"group_id,omitempty"`
	Mode     DeliveryMode `json:"mode,omitempty"`
}

// BuildDeliveryTarget merges channel-instance delivery defaults with explicit
// request overrides and returns one canonical outbound target.
func BuildDeliveryTarget(instance ChannelInstance, req ResolveDeliveryTargetRequest) (DeliveryTarget, error) {
	normalizedInstance := instance.normalize()
	if err := normalizedInstance.Validate(); err != nil {
		return DeliveryTarget{}, err
	}

	normalizedReq := req.normalize()
	if err := normalizedReq.Validate(); err != nil {
		return DeliveryTarget{}, err
	}
	if normalizedReq.ChannelInstanceID != normalizedInstance.ID {
		return DeliveryTarget{}, fmt.Errorf(
			"channels: delivery target request channel instance id %q does not match instance %q",
			normalizedReq.ChannelInstanceID,
			normalizedInstance.ID,
		)
	}

	defaults, err := decodeDeliveryTargetDefaults(normalizedInstance.DeliveryDefaults)
	if err != nil {
		return DeliveryTarget{}, err
	}

	target := DeliveryTarget{
		ChannelInstanceID: normalizedInstance.ID,
		PeerID:            firstNonEmpty(normalizedReq.PeerID, defaults.PeerID),
		ThreadID:          firstNonEmpty(normalizedReq.ThreadID, defaults.ThreadID),
		GroupID:           firstNonEmpty(normalizedReq.GroupID, defaults.GroupID),
		Mode:              normalizedReq.Mode,
	}
	if target.Mode == "" {
		target.Mode = defaults.Mode
	}
	if target.Mode == "" {
		target.Mode = DeliveryModeDirectSend
	}

	canonical := target.normalize()
	if err := canonical.Validate(); err != nil {
		return DeliveryTarget{}, err
	}
	return canonical, nil
}

// ResolveDeliveryTarget loads the owning channel instance and resolves the
// canonical outbound target under that instance's delivery defaults.
func (s *Service) ResolveDeliveryTarget(ctx context.Context, req ResolveDeliveryTargetRequest) (*DeliveryTarget, error) {
	if err := s.checkReady(ctx, "resolve delivery target"); err != nil {
		return nil, err
	}

	normalizedReq := req.normalize()
	if err := normalizedReq.Validate(); err != nil {
		return nil, err
	}

	instance, err := s.loadRoutableInstance(ctx, normalizedReq.ChannelInstanceID)
	if err != nil {
		return nil, err
	}

	target, err := BuildDeliveryTarget(instance, normalizedReq)
	if err != nil {
		return nil, err
	}
	return cloneDeliveryTarget(target), nil
}

func (r ResolveDeliveryTargetRequest) normalize() ResolveDeliveryTargetRequest {
	normalized := r
	normalized.ChannelInstanceID = strings.TrimSpace(normalized.ChannelInstanceID)
	normalized.PeerID = strings.TrimSpace(normalized.PeerID)
	normalized.ThreadID = strings.TrimSpace(normalized.ThreadID)
	normalized.GroupID = strings.TrimSpace(normalized.GroupID)
	normalized.Mode = normalized.Mode.Normalize()
	return normalized
}

func (d deliveryTargetDefaults) normalize() deliveryTargetDefaults {
	return deliveryTargetDefaults{
		PeerID:   strings.TrimSpace(d.PeerID),
		ThreadID: strings.TrimSpace(d.ThreadID),
		GroupID:  strings.TrimSpace(d.GroupID),
		Mode:     d.Mode.Normalize(),
	}
}

func decodeDeliveryTargetDefaults(raw json.RawMessage) (deliveryTargetDefaults, error) {
	normalized, err := normalizeRawJSON(raw, "channel instance delivery defaults")
	if err != nil {
		return deliveryTargetDefaults{}, err
	}
	if len(normalized) == 0 {
		return deliveryTargetDefaults{}, nil
	}

	var defaults deliveryTargetDefaults
	if err := json.Unmarshal(normalized, &defaults); err != nil {
		return deliveryTargetDefaults{}, fmt.Errorf("channels: decode channel instance delivery defaults: %w", err)
	}

	defaults = defaults.normalize()
	if defaults.Mode != "" {
		if err := defaults.Mode.Validate(); err != nil {
			return deliveryTargetDefaults{}, err
		}
	}
	return defaults, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneDeliveryTarget(target DeliveryTarget) *DeliveryTarget {
	cloned := target.normalize()
	return &cloned
}
