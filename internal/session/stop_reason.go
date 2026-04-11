package session

import (
	"context"
	"errors"
	"strings"

	"github.com/pedronauck/agh/internal/store"
)

func classifyStopReason(cause StopCause, waitErr error, detail string) (store.StopReason, string) {
	trimmedDetail := strings.TrimSpace(detail)

	switch cause {
	case CauseShutdown:
		if trimmedDetail == "" {
			trimmedDetail = "daemon shutdown"
		}
		return store.StopShutdown, trimmedDetail
	case CauseHookDenied:
		return store.StopHookStopped, trimmedDetail
	case CauseUserRequested:
		lowerDetail := strings.ToLower(trimmedDetail)
		switch {
		case strings.Contains(lowerDetail, "max_iterations"):
			return store.StopMaxIterations, trimmedDetail
		case strings.Contains(lowerDetail, "loop_detected"):
			return store.StopLoopDetected, trimmedDetail
		case strings.Contains(lowerDetail, "budget_exceeded"):
			return store.StopBudgetExceeded, trimmedDetail
		default:
			return store.StopUserCanceled, trimmedDetail
		}
	case CauseProcessExited:
		if waitErr != nil {
			return store.StopAgentCrashed, waitErr.Error()
		}
		return store.StopError, "process exited unexpectedly"
	case CauseCompleted:
		return store.StopCompleted, ""
	case CauseFailed:
		return store.StopError, trimmedDetail
	default:
		if waitErr != nil {
			return store.StopError, waitErr.Error()
		}
		return store.StopCompleted, ""
	}
}

// StopWithCause stops a session while preserving the explicit stop initiator.
func (m *Manager) StopWithCause(ctx context.Context, id string, cause StopCause, detail string) error {
	if ctx == nil {
		return errors.New("session: stop context is required")
	}
	if cause == CauseNone {
		cause = CauseUserRequested
	}

	session, err := m.lookup(id)
	if err != nil {
		return err
	}
	if err := m.dispatchSessionPreStop(ctx, session); err != nil {
		return err
	}

	writeMeta, promptSetupDone, err := session.prepareStop(m.now(), cause, detail)
	if err != nil {
		return err
	}
	if writeMeta {
		if err := m.writeMeta(session); err != nil {
			return err
		}
	}
	if err := waitForPromptSetup(ctx, session, promptSetupDone); err != nil {
		return err
	}

	state := session.Info().State
	if state == StateStopped {
		return nil
	}

	proc := session.processHandle()
	if proc == nil {
		return m.finalizeStopped(ctx, session, nil)
	}

	stopErr := m.driver.Stop(ctx, proc)
	if !isProcessDone(proc) {
		return stopErr
	}

	return errors.Join(stopErr, m.finalizeStopped(ctx, session, nil))
}
