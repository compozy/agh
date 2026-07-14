package globaldb

import (
	"context"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

var _ store.NetworkAcceptanceStore = (*NetworkRepo)(nil)

// AcceptNetworkMessage commits conversation evidence, recipient decisions, and wake work atomically.
func (g *NetworkRepo) AcceptNetworkMessage(
	ctx context.Context,
	req store.AcceptNetworkMessageRequest,
) (result store.AcceptNetworkMessageResult, err error) {
	if err := g.checkReady(ctx, "accept network message"); err != nil {
		return store.AcceptNetworkMessageResult{}, err
	}
	normalized, err := g.normalizeNetworkAcceptance(req)
	if err != nil {
		return store.AcceptNetworkMessageResult{}, err
	}
	now := normalized.Message.Timestamp

	err = g.withNetworkImmediateTransaction(ctx, "accept network message", func(exec networkSQLExecutor) error {
		write, acceptanceSeq, persistErr := persistNetworkConversationMessageWithExecutor(
			ctx,
			exec,
			normalized.Message,
		)
		if persistErr != nil {
			return persistErr
		}
		result.AcceptanceSeq = acceptanceSeq
		result.Duplicate = write.Duplicate
		result.Conversation = write
		if write.Duplicate {
			result.Dispositions, persistErr = listNetworkMessageDispositions(
				ctx,
				exec,
				normalized.Message.MessageID,
			)
			return persistErr
		}

		if persistErr := insertNetworkMessageDispositions(
			ctx,
			exec,
			normalized.Message,
			normalized.Dispositions,
			acceptanceSeq,
			now,
		); persistErr != nil {
			return persistErr
		}
		result.Dispositions = append([]store.NetworkMessageDisposition(nil), normalized.Dispositions...)
		availability, availabilityErr := getNetworkAvailabilityWithExecutor(ctx, exec)
		if availabilityErr != nil {
			return availabilityErr
		}
		result.AvailabilityEpoch = availability.Epoch
		for _, input := range normalized.Admissions {
			decision, admissionErr := g.admitNetworkWake(
				ctx,
				exec,
				normalized.Message,
				input,
				acceptanceSeq,
				now,
				availability.Enabled,
			)
			if admissionErr != nil {
				return admissionErr
			}
			if decision.reservation != nil {
				result.Admitted = append(result.Admitted, *decision.reservation)
				result.Notify = append(result.Notify, store.CommittedNetworkNotification{
					RecipientSessionID: input.RecipientSessionID,
					TaskRunID:          decision.reservation.TaskRunID,
					AcceptanceSeq:      acceptanceSeq,
				})
			}
			if decision.skip != nil {
				result.Skipped = append(result.Skipped, *decision.skip)
			}
		}
		return nil
	})
	if err != nil {
		return store.AcceptNetworkMessageResult{}, err
	}
	return result, nil
}

func (g *NetworkRepo) normalizeNetworkAcceptance(
	req store.AcceptNetworkMessageRequest,
) (store.AcceptNetworkMessageRequest, error) {
	message, err := g.normalizeConversationMessage(req.Message)
	if err != nil {
		return store.AcceptNetworkMessageRequest{}, err
	}
	normalized := store.AcceptNetworkMessageRequest{
		Message:      message,
		Dispositions: make([]store.NetworkMessageDisposition, 0, len(req.Dispositions)),
		Admissions:   make([]store.NetworkWakeAdmissionInput, 0, len(req.Admissions)),
	}
	decisions := make(map[string]string, len(req.Dispositions))
	for _, disposition := range req.Dispositions {
		disposition.RecipientSessionID = strings.TrimSpace(disposition.RecipientSessionID)
		disposition.Decision = strings.TrimSpace(disposition.Decision)
		if err := disposition.Validate(); err != nil {
			return store.AcceptNetworkMessageRequest{}, err
		}
		if _, exists := decisions[disposition.RecipientSessionID]; exists {
			return store.AcceptNetworkMessageRequest{}, fmt.Errorf(
				"store: duplicate network message disposition recipient %q",
				disposition.RecipientSessionID,
			)
		}
		decisions[disposition.RecipientSessionID] = disposition.Decision
		normalized.Dispositions = append(normalized.Dispositions, disposition)
	}
	admissionRecipients := make(map[string]struct{}, len(req.Admissions))
	for _, input := range req.Admissions {
		input = normalizeNetworkWakeAdmissionInput(input)
		if err := input.Validate(); err != nil {
			return store.AcceptNetworkMessageRequest{}, err
		}
		if decisions[input.RecipientSessionID] != store.NetworkDispositionDeliver {
			return store.AcceptNetworkMessageRequest{}, fmt.Errorf(
				"store: network wake recipient %q must have a deliver disposition",
				input.RecipientSessionID,
			)
		}
		if _, exists := admissionRecipients[input.RecipientSessionID]; exists {
			return store.AcceptNetworkMessageRequest{}, fmt.Errorf(
				"store: duplicate network wake admission recipient %q",
				input.RecipientSessionID,
			)
		}
		admissionRecipients[input.RecipientSessionID] = struct{}{}
		if input.Spec.Mode == "live" &&
			(input.Spec.WorkspaceID != message.WorkspaceID || input.Spec.ChannelID != message.Channel) {
			return store.AcceptNetworkMessageRequest{}, fmt.Errorf(
				"store: network wake participation scope does not match accepted message",
			)
		}
		normalized.Admissions = append(normalized.Admissions, input)
	}
	return normalized, nil
}

func normalizeNetworkWakeAdmissionInput(input store.NetworkWakeAdmissionInput) store.NetworkWakeAdmissionInput {
	input.RecipientSessionID = strings.TrimSpace(input.RecipientSessionID)
	input.OwnerKey = strings.TrimSpace(input.OwnerKey)
	input.Trigger = strings.TrimSpace(input.Trigger)
	input.RootID = strings.TrimSpace(input.RootID)
	input.WakeID = strings.TrimSpace(input.WakeID)
	input.TaskRunID = strings.TrimSpace(input.TaskRunID)
	return input
}
