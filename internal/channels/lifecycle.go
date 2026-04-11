package channels

import (
	"fmt"
)

// ValidateInstanceStateTransition reports whether the next enabled/status pair is
// a valid lifecycle transition from the current instance state.
func ValidateInstanceStateTransition(current ChannelInstance, nextEnabled bool, nextStatus ChannelStatus) error {
	normalizedCurrent := current.normalize()
	if err := normalizedCurrent.Validate(); err != nil {
		return err
	}

	normalizedNextStatus := nextStatus.Normalize()
	if err := validateInstanceLifecycle(nextEnabled, normalizedNextStatus); err != nil {
		return err
	}

	if normalizedCurrent.Enabled == nextEnabled && normalizedCurrent.Status == normalizedNextStatus {
		return nil
	}

	if !canTransitionInstanceState(normalizedCurrent.Enabled, normalizedCurrent.Status, nextEnabled, normalizedNextStatus) {
		return fmt.Errorf(
			"%w: enabled=%t,status=%s -> enabled=%t,status=%s",
			ErrInvalidChannelStateTransition,
			normalizedCurrent.Enabled,
			normalizedCurrent.Status,
			nextEnabled,
			normalizedNextStatus,
		)
	}

	return nil
}

func validateInstanceLifecycle(enabled bool, status ChannelStatus) error {
	normalizedStatus := status.Normalize()
	if err := normalizedStatus.Validate(); err != nil {
		return err
	}

	if !enabled && normalizedStatus != ChannelStatusDisabled {
		return fmt.Errorf("channels: disabled channel instance must report status %q", ChannelStatusDisabled)
	}
	if enabled && normalizedStatus == ChannelStatusDisabled {
		return fmt.Errorf("channels: enabled channel instance cannot report status %q", ChannelStatusDisabled)
	}

	return nil
}

func canTransitionInstanceState(currentEnabled bool, currentStatus ChannelStatus, nextEnabled bool, nextStatus ChannelStatus) bool {
	normalizedCurrent := currentStatus.Normalize()
	normalizedNext := nextStatus.Normalize()

	switch normalizedCurrent {
	case ChannelStatusDisabled:
		return !currentEnabled && nextEnabled && normalizedNext == ChannelStatusStarting
	case ChannelStatusStarting:
		if !currentEnabled {
			return false
		}
		return transitionFromStarting(nextEnabled, normalizedNext)
	case ChannelStatusReady:
		if !currentEnabled {
			return false
		}
		return transitionFromReady(nextEnabled, normalizedNext)
	case ChannelStatusDegraded:
		if !currentEnabled {
			return false
		}
		return transitionFromDegraded(nextEnabled, normalizedNext)
	case ChannelStatusAuthRequired:
		if !currentEnabled {
			return false
		}
		return transitionFromAuthRequired(nextEnabled, normalizedNext)
	case ChannelStatusError:
		if !currentEnabled {
			return false
		}
		return transitionFromError(nextEnabled, normalizedNext)
	default:
		return false
	}
}

func transitionFromStarting(nextEnabled bool, nextStatus ChannelStatus) bool {
	if !nextEnabled {
		return nextStatus == ChannelStatusDisabled
	}

	switch nextStatus {
	case ChannelStatusStarting, ChannelStatusReady, ChannelStatusDegraded, ChannelStatusAuthRequired, ChannelStatusError:
		return true
	default:
		return false
	}
}

func transitionFromReady(nextEnabled bool, nextStatus ChannelStatus) bool {
	if !nextEnabled {
		return nextStatus == ChannelStatusDisabled
	}

	switch nextStatus {
	case ChannelStatusReady, ChannelStatusStarting, ChannelStatusDegraded, ChannelStatusAuthRequired, ChannelStatusError:
		return true
	default:
		return false
	}
}

func transitionFromDegraded(nextEnabled bool, nextStatus ChannelStatus) bool {
	if !nextEnabled {
		return nextStatus == ChannelStatusDisabled
	}

	switch nextStatus {
	case ChannelStatusDegraded, ChannelStatusStarting, ChannelStatusReady, ChannelStatusAuthRequired, ChannelStatusError:
		return true
	default:
		return false
	}
}

func transitionFromAuthRequired(nextEnabled bool, nextStatus ChannelStatus) bool {
	if !nextEnabled {
		return nextStatus == ChannelStatusDisabled
	}

	switch nextStatus {
	case ChannelStatusAuthRequired, ChannelStatusStarting, ChannelStatusError:
		return true
	default:
		return false
	}
}

func transitionFromError(nextEnabled bool, nextStatus ChannelStatus) bool {
	if !nextEnabled {
		return nextStatus == ChannelStatusDisabled
	}

	switch nextStatus {
	case ChannelStatusError, ChannelStatusStarting:
		return true
	default:
		return false
	}
}
