package network

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInteractionNotFound reports that no current interaction matched the
	// lifecycle message being applied.
	ErrInteractionNotFound = errors.New("network: interaction not found")
	// ErrInteractionActorNotAllowed reports a lifecycle actor outside the
	// initiator/target pair.
	ErrInteractionActorNotAllowed = errors.New("network: interaction actor not allowed")
	// ErrInvalidStateTransition reports an impossible lifecycle transition.
	ErrInvalidStateTransition = errors.New("network: invalid interaction state transition")
	// ErrInteractionClosed reports a message for a terminal interaction that must
	// be rejected instead of reopening the interaction.
	ErrInteractionClosed = errors.New("network: interaction closed")
)

// Interaction tracks one directed interaction inside one channel.
type Interaction struct {
	ID        string
	Channel   string
	Initiator string
	Target    string
	State     InteractionState
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate reports whether the interaction carries a usable identity and state.
func (i Interaction) Validate() error {
	if i.ID == "" {
		return fmt.Errorf("%w: interaction id is required", ErrMissingField)
	}
	if i.Channel == "" {
		return fmt.Errorf("%w: interaction channel is required", ErrMissingField)
	}
	if err := ValidateChannel(i.Channel); err != nil {
		return fmt.Errorf("validate interaction channel: %w", err)
	}
	if i.Initiator == "" {
		return fmt.Errorf("%w: interaction initiator is required", ErrMissingField)
	}
	if err := ValidatePeerID(i.Initiator); err != nil {
		return fmt.Errorf("%w: interaction initiator", err)
	}
	if i.Target == "" {
		return fmt.Errorf("%w: interaction target is required", ErrMissingField)
	}
	if err := ValidatePeerID(i.Target); err != nil {
		return fmt.Errorf("%w: interaction target", err)
	}
	if err := i.State.Validate(); err != nil {
		return fmt.Errorf("validate interaction state: %w", err)
	}
	return nil
}

// IsTerminalState reports whether the state is terminal under the RFC.
func IsTerminalState(state InteractionState) bool {
	switch state {
	case StateCompleted, StateFailed, StateCanceled:
		return true
	default:
		return false
	}
}

// IsParticipant reports whether the peer owns the interaction lifecycle.
func (i Interaction) IsParticipant(peerID string) bool {
	return peerID == i.Initiator || peerID == i.Target
}

func (i Interaction) counterparty(peerID string) (string, bool) {
	switch peerID {
	case i.Initiator:
		return i.Target, true
	case i.Target:
		return i.Initiator, true
	default:
		return "", false
	}
}

// LifecycleAction explains how a lifecycle helper handled one message.
type LifecycleAction string

const (
	LifecycleActionOpened       LifecycleAction = "opened"
	LifecycleActionAdvanced     LifecycleAction = "advanced"
	LifecycleActionUnchanged    LifecycleAction = "unchanged"
	LifecycleActionIgnored      LifecycleAction = "ignored"
	LifecycleActionRejectDirect LifecycleAction = "reject_direct"
)

// LifecycleResult is the reusable lifecycle decision surface for router code.
type LifecycleResult struct {
	Interaction Interaction
	Action      LifecycleAction
	ReasonCode  *ReasonCode
}

// OpenInteraction opens a new interaction from the first directed message.
func OpenInteraction(env Envelope, at time.Time) (Interaction, error) {
	if env.Kind != KindDirect && env.Kind != KindCapability {
		return Interaction{}, fmt.Errorf("%w: opening message kind=%q", ErrInvalidField, env.Kind)
	}
	if env.To == nil {
		return Interaction{}, fmt.Errorf("%w: opening message to is required", ErrMissingField)
	}
	if env.InteractionID == nil {
		return Interaction{}, fmt.Errorf("%w: opening message interaction_id is required", ErrMissingField)
	}

	if at.IsZero() {
		at = time.Now().UTC()
	}

	interaction := Interaction{
		ID:        *env.InteractionID,
		Channel:   env.Channel,
		Initiator: env.From,
		Target:    *env.To,
		State:     StateSubmitted,
		CreatedAt: at,
		UpdatedAt: at,
	}

	if err := interaction.Validate(); err != nil {
		return Interaction{}, fmt.Errorf("validate opened interaction: %w", err)
	}

	return interaction, nil
}

// ApplyInteractionEnvelope applies one validated lifecycle envelope to the
// current interaction state and returns the router-facing decision.
func ApplyInteractionEnvelope(current *Interaction, env Envelope, at time.Time) (LifecycleResult, error) {
	at = normalizeInteractionTime(at)

	if current == nil {
		return openInteractionResult(env, at)
	}

	interaction, err := validateInteractionEnvelope(*current, env)
	if err != nil {
		return LifecycleResult{}, err
	}
	if result, terminal := terminalInteractionResult(interaction, env); terminal {
		return result, nil
	}

	return applyActiveInteractionEnvelope(interaction, env, at)
}

func normalizeInteractionTime(at time.Time) time.Time {
	if at.IsZero() {
		return time.Now().UTC()
	}
	return at
}

func openInteractionResult(env Envelope, at time.Time) (LifecycleResult, error) {
	opened, err := OpenInteraction(env, at)
	if err != nil {
		if env.Kind != KindDirect && env.Kind != KindCapability {
			return LifecycleResult{}, fmt.Errorf("%w: kind=%q", ErrInteractionNotFound, env.Kind)
		}
		return LifecycleResult{}, err
	}
	return LifecycleResult{
		Interaction: opened,
		Action:      LifecycleActionOpened,
	}, nil
}

func validateInteractionEnvelope(current Interaction, env Envelope) (Interaction, error) {
	if err := current.Validate(); err != nil {
		return Interaction{}, fmt.Errorf("validate current interaction: %w", err)
	}
	if err := validateInteractionIdentity(current, env); err != nil {
		return Interaction{}, err
	}
	if !current.IsParticipant(env.From) {
		return Interaction{}, fmt.Errorf("%w: from=%q", ErrInteractionActorNotAllowed, env.From)
	}
	if err := validateInteractionDirection(current, env); err != nil {
		return Interaction{}, err
	}
	return current, nil
}

func validateInteractionIdentity(current Interaction, env Envelope) error {
	if env.InteractionID == nil || *env.InteractionID != current.ID {
		return fmt.Errorf(
			"%w: interaction_id=%v current=%q",
			ErrInvalidField,
			env.InteractionID,
			current.ID,
		)
	}
	if env.Channel != current.Channel {
		return fmt.Errorf(
			"%w: interaction channel=%q envelope channel=%q",
			ErrInvalidField,
			current.Channel,
			env.Channel,
		)
	}
	return nil
}

func validateInteractionDirection(current Interaction, env Envelope) error {
	if env.Kind != KindDirect && env.Kind != KindCapability {
		return nil
	}
	if env.To == nil {
		return fmt.Errorf("%w: %s to is required", ErrMissingField, env.Kind)
	}

	expectedTarget, ok := current.counterparty(env.From)
	if !ok || *env.To != expectedTarget {
		return fmt.Errorf(
			"%w: from=%q to=%q expected_to=%q",
			ErrInteractionActorNotAllowed,
			env.From,
			*env.To,
			expectedTarget,
		)
	}
	return nil
}

func terminalInteractionResult(interaction Interaction, env Envelope) (LifecycleResult, bool) {
	if !IsTerminalState(interaction.State) {
		return LifecycleResult{}, false
	}

	switch env.Kind {
	case KindTrace, KindReceipt:
		return LifecycleResult{
			Interaction: interaction,
			Action:      LifecycleActionIgnored,
		}, true
	case KindDirect, KindCapability:
		reason := ReasonCodeInteractionClosed
		return LifecycleResult{
			Interaction: interaction,
			Action:      LifecycleActionRejectDirect,
			ReasonCode:  &reason,
		}, true
	default:
		return LifecycleResult{}, false
	}
}

func applyActiveInteractionEnvelope(interaction Interaction, env Envelope, at time.Time) (LifecycleResult, error) {
	switch env.Kind {
	case KindDirect, KindCapability:
		return applyDirectOrCapability(interaction, at), nil
	case KindReceipt:
		receipt, err := decodeLifecycleBody[ReceiptBody](env, "receipt")
		if err != nil {
			return LifecycleResult{}, err
		}
		return applyReceipt(interaction, receipt, at)
	case KindTrace:
		trace, err := decodeLifecycleBody[TraceBody](env, "trace")
		if err != nil {
			return LifecycleResult{}, err
		}
		return applyTrace(interaction, trace, at)
	default:
		return LifecycleResult{}, fmt.Errorf("%w: lifecycle kind=%q", ErrInvalidField, env.Kind)
	}
}

func applyDirectOrCapability(interaction Interaction, at time.Time) LifecycleResult {
	if interaction.State == StateNeedsInput {
		interaction.State = StateWorking
		interaction.UpdatedAt = at
		return LifecycleResult{Interaction: interaction, Action: LifecycleActionAdvanced}
	}
	return LifecycleResult{Interaction: interaction, Action: LifecycleActionUnchanged}
}

func decodeLifecycleBody[T any](env Envelope, label string) (T, error) {
	var zero T

	body, err := env.DecodeBody()
	if err != nil {
		return zero, fmt.Errorf("decode %s body: %w", label, err)
	}
	typed, ok := body.(T)
	if !ok {
		return zero, fmt.Errorf("%w: expected %s body", ErrInvalidBody, label)
	}
	return typed, nil
}

func applyReceipt(interaction Interaction, receipt ReceiptBody, at time.Time) (LifecycleResult, error) {
	switch receipt.Status {
	case ReceiptStatusAccepted, ReceiptStatusDuplicate, ReceiptStatusExpired, ReceiptStatusUnsupported:
		return LifecycleResult{
			Interaction: interaction,
			Action:      LifecycleActionUnchanged,
		}, nil
	case ReceiptStatusRejected:
		interaction.State = StateFailed
	case ReceiptStatusCanceled:
		interaction.State = StateCanceled
	default:
		return LifecycleResult{}, fmt.Errorf("%w: receipt status=%q", ErrInvalidStateTransition, receipt.Status)
	}

	interaction.UpdatedAt = at
	return LifecycleResult{
		Interaction: interaction,
		Action:      LifecycleActionAdvanced,
	}, nil
}

func applyTrace(interaction Interaction, trace TraceBody, at time.Time) (LifecycleResult, error) {
	if !canApplyTrace(interaction.State, trace.State) {
		return LifecycleResult{}, fmt.Errorf("%w: %s -> %s", ErrInvalidStateTransition, interaction.State, trace.State)
	}

	updated := interaction
	updated.State = trace.State
	updated.UpdatedAt = at

	return LifecycleResult{
		Interaction: updated,
		Action:      LifecycleActionAdvanced,
	}, nil
}

func canApplyTrace(current InteractionState, next InteractionState) bool {
	switch current {
	case StateSubmitted:
		return next == StateWorking || next == StateNeedsInput || next == StateCompleted || next == StateFailed ||
			next == StateCanceled
	case StateWorking:
		return next == StateWorking || next == StateNeedsInput || next == StateCompleted || next == StateFailed ||
			next == StateCanceled
	case StateNeedsInput:
		return next == StateWorking || next == StateNeedsInput || next == StateCompleted || next == StateFailed ||
			next == StateCanceled
	default:
		return false
	}
}
