package session

import (
	"context"
	"errors"
	"fmt"
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

// RequestStopWithCause marks a session as stopping and sends the cooperative ACP
// cancel signal without forcing process termination.
func (m *Manager) RequestStopWithCause(ctx context.Context, id string, cause StopCause, detail string) error {
	if m == nil {
		return errors.New("session: manager is required")
	}
	if ctx == nil {
		return errors.New("session: request stop context is required")
	}
	if cause == CauseNone {
		cause = CauseUserRequested
	}

	session, proc, alreadyStopped, err := m.prepareStopWithCause(ctx, id, cause, detail)
	if err != nil {
		return err
	}
	if alreadyStopped {
		return nil
	}
	if proc == nil {
		return m.finalizeStopped(ctx, session, nil)
	}

	cancelErr := m.driver.Cancel(ctx, proc)
	if cancelErr != nil && !isProcessDone(proc) {
		return fmt.Errorf("session: request cooperative stop for %q: %w", id, cancelErr)
	}
	if isProcessDone(proc) {
		return errors.Join(cancelErr, m.finalizeStopped(ctx, session, nil))
	}
	return cancelErr
}

// StopWithCause stops a session while preserving the explicit stop initiator.
func (m *Manager) StopWithCause(ctx context.Context, id string, cause StopCause, detail string) error {
	if m == nil {
		return errors.New("session: manager is required")
	}
	if ctx == nil {
		return errors.New("session: stop context is required")
	}
	if cause == CauseNone {
		cause = CauseUserRequested
	}

	session, proc, alreadyStopped, err := m.prepareStopWithCause(ctx, id, cause, detail)
	if err != nil {
		return err
	}
	if alreadyStopped {
		return nil
	}
	if proc == nil {
		return m.finalizeStopped(ctx, session, nil)
	}

	stopErr := m.driver.Stop(ctx, proc)
	if stopErr == nil && !isProcessDone(proc) {
		select {
		case <-proc.Done():
		case <-ctx.Done():
			return fmt.Errorf("session: wait for process stop completion for %q: %w", id, ctx.Err())
		}
	}
	if stopErr != nil && !isProcessDone(proc) {
		return fmt.Errorf("session: stop session process for %q: %w", id, stopErr)
	}

	return errors.Join(stopErr, m.finalizeStopped(ctx, session, nil))
}

func (m *Manager) prepareStopWithCause(ctx context.Context, id string, cause StopCause, detail string) (*Session, *AgentProcess, bool, error) {
	session, err := m.lookup(id)
	if err != nil {
		return nil, nil, false, err
	}
	if session.Info().State == StateActive {
		if err := m.dispatchSessionPreStop(ctx, session); err != nil {
			return nil, nil, false, err
		}
	}

	writeMeta, promptSetupDone, err := session.prepareStop(m.now(), cause, detail)
	if err != nil {
		return nil, nil, false, err
	}
	if writeMeta {
		if err := m.writeMeta(session); err != nil {
			return nil, nil, false, err
		}
	}
	if err := waitForPromptSetup(ctx, session, promptSetupDone); err != nil {
		return nil, nil, false, err
	}

	if session.Info().State == StateStopped {
		return session, nil, true, nil
	}
	return session, session.processHandle(), false, nil
}
