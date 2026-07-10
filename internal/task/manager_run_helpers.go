package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"strings"
)

func nextRunAttempt(runs []Run) int {
	maxAttempt := 0
	for _, run := range runs {
		if int(run.Attempt) > maxAttempt {
			maxAttempt = int(run.Attempt)
		}
	}
	return maxAttempt + 1
}

func (m *Service) validateNetworkChannel(path string, channel string) error {
	if m == nil || m.channelValidator == nil {
		return nil
	}

	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return nil
	}
	if err := m.channelValidator(trimmed); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrValidation, path, err)
	}
	return nil
}

func (m *Service) validateRunChannelUsable(
	ctx context.Context,
	taskRecord Task,
	run Run,
	actor ActorContext,
	operation string,
) error {
	channel := resolvedRunChannel(run.NetworkChannel, taskRecord.NetworkChannel)
	if strings.TrimSpace(channel) == "" {
		return nil
	}
	if err := m.validateNetworkChannel("task_run.network_channel", channel); err == nil {
		return nil
	}

	rejectedErr := fmt.Errorf(
		"%w: task %q run %q channel %q is no longer valid",
		ErrStaleNetworkChannel,
		taskRecord.ID,
		run.ID,
		strings.TrimSpace(channel),
	)
	if recordErr := m.recordTaskEvent(ctx, taskRecord.ID, run.ID, taskEventRunRejected, actor, rejectedRunPayload{
		Operation:      strings.TrimSpace(operation),
		Reason:         "stale_network_channel",
		NetworkChannel: strings.TrimSpace(channel),
	}); recordErr != nil {
		return errorsJoin(rejectedErr, recordErr)
	}
	return rejectedErr
}

func resolvedRunChannel(requested string, taskChannel string) string {
	if strings.TrimSpace(requested) != "" {
		return strings.TrimSpace(requested)
	}
	return strings.TrimSpace(taskChannel)
}

func errorsJoin(errs ...error) error {
	return errors.Join(errs...)
}

func runBootRecoveryError(run Run, recovery RunBootRecovery) string {
	sessionID := strings.TrimSpace(run.SessionID)
	switch {
	case sessionID != "" && recovery.Classification != "" && recovery.Detail != "":
		return fmt.Sprintf(
			"orphaned on boot: session %q classified as %s (%s)",
			sessionID,
			recovery.Classification,
			recovery.Detail,
		)
	case sessionID != "" && recovery.Classification != "":
		return fmt.Sprintf(
			"orphaned on boot: session %q classified as %s",
			sessionID,
			recovery.Classification,
		)
	case sessionID != "" && recovery.SessionState != "":
		return fmt.Sprintf("orphaned on boot: session %q is %s", sessionID, recovery.SessionState)
	case sessionID != "":
		return fmt.Sprintf("orphaned on boot: session %q is not live", sessionID)
	default:
		return "orphaned on boot: run has no live session"
	}
}

func runBootRecoveryMetadata(run Run, recovery RunBootRecovery) json.RawMessage {
	payload, err := marshalTaskEventPayload(map[string]string{
		runBootRecoveryReasonKey: normalizedBootRecoveryReason(recovery.Reason),
		"previous_status":        run.Status.Normalize().String(),
		"session_id":             strings.TrimSpace(run.SessionID),
		"session_state":          strings.TrimSpace(recovery.SessionState),
		"classification":         strings.TrimSpace(recovery.Classification),
		"detail":                 strings.TrimSpace(recovery.Detail),
	})
	if err != nil {
		return nil
	}
	return payload
}

func normalizedBootRecoveryReason(reason string) string {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return managerOrphanedOnBootKey
	}
	return trimmed
}
