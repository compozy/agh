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

// Interaction tracks one directed interaction inside one space.
type Interaction struct {
	ID        string
	Space     string
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
	if i.Space == "" {
		return fmt.Errorf("%w: interaction space is required", ErrMissingField)
	}
	if err := ValidateSpace(i.Space); err != nil {
		return err
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
		return err
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
	if env.Kind != KindDirect {
		return Interaction{}, fmt.Errorf("%w: opening message kind=%q", ErrInvalidField, env.Kind)
	}
	if env.To == nil {
		return Interaction{}, fmt.Errorf("%w: direct to is required", ErrMissingField)
	}
	if env.InteractionID == nil {
		return Interaction{}, fmt.Errorf("%w: direct interaction_id is required", ErrMissingField)
	}

	if at.IsZero() {
		at = time.Now().UTC()
	}

	interaction := Interaction{
		ID:        *env.InteractionID,
		Space:     env.Space,
		Initiator: env.From,
		Target:    *env.To,
		State:     StateSubmitted,
		CreatedAt: at,
		UpdatedAt: at,
	}

	if err := interaction.Validate(); err != nil {
		return Interaction{}, err
	}

	return interaction, nil
}

// ApplyInteractionEnvelope applies one validated lifecycle envelope to the
// current interaction state and returns the router-facing decision.
func ApplyInteractionEnvelope(current *Interaction, env Envelope, at time.Time) (LifecycleResult, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}

	if current == nil {
		opened, err := OpenInteraction(env, at)
		if err != nil {
			if env.Kind != KindDirect {
				return LifecycleResult{}, fmt.Errorf("%w: kind=%q", ErrInteractionNotFound, env.Kind)
			}
			return LifecycleResult{}, err
		}
		return LifecycleResult{
			Interaction: opened,
			Action:      LifecycleActionOpened,
		}, nil
	}

	interaction := *current
	if err := interaction.Validate(); err != nil {
		return LifecycleResult{}, err
	}
	if env.InteractionID == nil || *env.InteractionID != interaction.ID {
		return LifecycleResult{}, fmt.Errorf("%w: interaction_id=%v current=%q", ErrInvalidField, env.InteractionID, interaction.ID)
	}
	if env.Space != interaction.Space {
		return LifecycleResult{}, fmt.Errorf("%w: interaction space=%q envelope space=%q", ErrInvalidField, interaction.Space, env.Space)
	}
	if !interaction.IsParticipant(env.From) {
		return LifecycleResult{}, fmt.Errorf("%w: from=%q", ErrInteractionActorNotAllowed, env.From)
	}

	if IsTerminalState(interaction.State) {
		switch env.Kind {
		case KindTrace:
			return LifecycleResult{
				Interaction: interaction,
				Action:      LifecycleActionIgnored,
			}, nil
		case KindDirect:
			reason := ReasonCodeInteractionClosed
			return LifecycleResult{
				Interaction: interaction,
				Action:      LifecycleActionRejectDirect,
				ReasonCode:  &reason,
			}, nil
		case KindReceipt:
			return LifecycleResult{
				Interaction: interaction,
				Action:      LifecycleActionIgnored,
			}, nil
		}
	}

	switch env.Kind {
	case KindDirect:
		if interaction.State == StateNeedsInput {
			interaction.State = StateWorking
			interaction.UpdatedAt = at
			return LifecycleResult{Interaction: interaction, Action: LifecycleActionAdvanced}, nil
		}
		return LifecycleResult{Interaction: interaction, Action: LifecycleActionUnchanged}, nil
	case KindReceipt:
		body, err := env.DecodeBody()
		if err != nil {
			return LifecycleResult{}, err
		}
		receipt, ok := body.(ReceiptBody)
		if !ok {
			return LifecycleResult{}, fmt.Errorf("%w: expected receipt body", ErrInvalidBody)
		}
		return applyReceipt(interaction, receipt, at)
	case KindTrace:
		body, err := env.DecodeBody()
		if err != nil {
			return LifecycleResult{}, err
		}
		trace, ok := body.(TraceBody)
		if !ok {
			return LifecycleResult{}, fmt.Errorf("%w: expected trace body", ErrInvalidBody)
		}
		return applyTrace(interaction, trace, at)
	default:
		return LifecycleResult{}, fmt.Errorf("%w: lifecycle kind=%q", ErrInvalidField, env.Kind)
	}
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
		return next == StateWorking || next == StateNeedsInput || next == StateCompleted || next == StateFailed || next == StateCanceled
	case StateWorking:
		return next == StateWorking || next == StateNeedsInput || next == StateCompleted || next == StateFailed || next == StateCanceled
	case StateNeedsInput:
		return next == StateWorking || next == StateNeedsInput || next == StateCompleted || next == StateFailed || next == StateCanceled
	default:
		return false
	}
}
