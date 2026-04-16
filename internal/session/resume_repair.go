package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	resumeValidationCheckMetaFields = "meta_fields"
	resumeValidationCheckWorkspace  = "workspace_dir"
	resumeValidationCheckAgent      = "agent"
	resumeValidationCheckEventStore = "event_store"
	resumeStopDetailAgentCrashed    = "daemon crashed while session active"
	resumeStopDetailStartIncomplete = "start did not complete"
)

type resumeValidationError struct {
	check string
	err   error
}

func (e resumeValidationError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e resumeValidationError) Unwrap() error {
	return e.err
}

func (e resumeValidationError) Check() string {
	return e.check
}

func classifyPreviousStop(meta store.SessionMeta) (store.SessionMeta, bool) {
	next := meta

	switch strings.TrimSpace(meta.State) {
	case string(StateActive):
		next.State = string(StateStopped)
		next.StopReason = resumeStopReasonPointer(store.StopAgentCrashed)
		next.StopDetail = resumeStopDetailAgentCrashed
		return next, true
	case string(StateStopping):
		next.State = string(StateStopped)
		next.StopReason = resumeStopReasonPointer(store.StopAgentCrashed)
		next.StopDetail = "stop did not complete"
		return next, true
	case string(StateStarting):
		next.State = string(StateStopped)
		next.StopReason = resumeStopReasonPointer(store.StopError)
		next.StopDetail = resumeStopDetailStartIncomplete
		next.ACPSessionID = nil
		return next, true
	case string(StateStopped):
		if strings.TrimSpace(meta.StopDetail) == resumeStopDetailStartIncomplete && meta.ACPSessionID != nil {
			next.ACPSessionID = nil
			return next, true
		}
		return next, false
	default:
		return next, false
	}
}

func (m *Manager) repairInactiveMeta(metaPath string, meta store.SessionMeta) (store.SessionMeta, error) {
	classified, changed := classifyPreviousStop(meta)
	if !changed {
		return meta, nil
	}
	return m.persistResumeCrashClassification(metaPath, classified)
}

func (m *Manager) restoreFailedResumeStart(
	metaPath string,
	meta store.SessionMeta,
	clearACP bool,
) (store.SessionMeta, error) {
	restored := meta
	restored.State = string(StateStopped)
	if clearACP {
		restored.StopReason = resumeStopReasonPointer(store.StopError)
		restored.StopDetail = resumeStopDetailStartIncomplete
		restored.ACPSessionID = nil
	}
	restored.UpdatedAt = m.now()

	if err := store.WriteSessionMeta(metaPath, restored); err != nil {
		return store.SessionMeta{}, fmt.Errorf(
			"session: restore stopped metadata after failed resume for %q: %w",
			strings.TrimSpace(meta.ID),
			err,
		)
	}

	return restored, nil
}

func (m *Manager) validateInfrastructure(ctx context.Context, meta store.SessionMeta) []error {
	var errs []error

	if err := meta.Validate(); err != nil {
		errs = append(errs, resumeValidationError{
			check: resumeValidationCheckMetaFields,
			err:   fmt.Errorf("session: validate session metadata for %q: %w", strings.TrimSpace(meta.ID), err),
		})
	}

	resolver, resolverErr := m.requireWorkspaceResolver()
	if resolverErr != nil {
		errs = append(errs, resumeValidationError{
			check: resumeValidationCheckWorkspace,
			err:   resolverErr,
		})
	} else {
		resolvedWorkspace, err := resolver.Resolve(ctx, strings.TrimSpace(meta.WorkspaceID))
		if err != nil {
			errs = append(errs, resumeValidationError{
				check: resumeValidationCheckWorkspace,
				err: fmt.Errorf(
					"session: resolve workspace %q for session %q: %w",
					strings.TrimSpace(meta.WorkspaceID),
					strings.TrimSpace(meta.ID),
					err,
				),
			})
		} else {
			if statErr := validateWorkspaceRoot(resolvedWorkspace.RootDir); statErr != nil {
				errs = append(errs, resumeValidationError{
					check: resumeValidationCheckWorkspace,
					err: fmt.Errorf(
						"session: validate workspace root %q for session %q: %w",
						strings.TrimSpace(resolvedWorkspace.RootDir),
						strings.TrimSpace(meta.ID),
						statErr,
					),
				})
			}

			if agentErr := m.validateResumeAgent(meta.AgentName, &resolvedWorkspace); agentErr != nil {
				errs = append(errs, resumeValidationError{
					check: resumeValidationCheckAgent,
					err: fmt.Errorf(
						"session: validate agent %q for session %q: %w",
						strings.TrimSpace(meta.AgentName),
						strings.TrimSpace(meta.ID),
						agentErr,
					),
				})
			}
		}
	}

	if eventStoreErr := m.validateEventStore(meta); eventStoreErr != nil {
		errs = append(errs, resumeValidationError{
			check: resumeValidationCheckEventStore,
			err:   eventStoreErr,
		})
	}

	return errs
}

func validateWorkspaceRoot(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return errors.New("workspace root path is required")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("workspace root is not a directory")
	}
	return nil
}

func (m *Manager) validateResumeAgent(agentName string, resolvedWorkspace *workspacepkg.ResolvedWorkspace) error {
	agentDef, err := m.resolveWorkspaceAgent(agentName, resolvedWorkspace)
	if err != nil {
		return err
	}
	if _, err := resolvedWorkspace.Config.ResolveAgent(agentDef); err != nil {
		return err
	}
	return nil
}

func (m *Manager) validateEventStore(meta store.SessionMeta) error {
	sessionID := strings.TrimSpace(meta.ID)
	if sessionID == "" {
		return nil
	}

	dbPath := store.SessionDBFile(filepath.Join(m.homePaths.SessionsDir, sessionID))
	info, err := os.Stat(dbPath)
	if err != nil {
		return fmt.Errorf("session: validate event store %q for session %q: %w", dbPath, sessionID, err)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("session: validate event store %q for session %q: file is empty", dbPath, sessionID)
	}
	return nil
}

func (m *Manager) persistResumeCrashClassification(metaPath string, meta store.SessionMeta) (store.SessionMeta, error) {
	classified := meta
	classified.UpdatedAt = m.now()
	if err := store.WriteSessionMeta(metaPath, classified); err != nil {
		return store.SessionMeta{}, fmt.Errorf(
			"session: persist crash classification for %q: %w",
			strings.TrimSpace(meta.ID),
			err,
		)
	}

	reason := ""
	if classified.StopReason != nil {
		reason = string(*classified.StopReason)
	}
	m.resumeLogger(classified).Info(
		"session.resume.crash_classified",
		"previous_state", strings.TrimSpace(meta.State),
		"stop_reason", reason,
		"stop_detail", strings.TrimSpace(classified.StopDetail),
	)
	return classified, nil
}

func (m *Manager) logResumeValidationFailures(meta store.SessionMeta, errs []error) {
	logger := m.resumeLogger(meta)
	for _, err := range errs {
		if err == nil {
			continue
		}

		check := ""
		var validationErr resumeValidationError
		if errors.As(err, &validationErr) {
			check = validationErr.Check()
		}

		logger.Warn("session.resume.validation_failed", "check", check, "error", err)
	}
}

func (m *Manager) resumeLogger(meta store.SessionMeta) *slog.Logger {
	logger := m.logger
	if logger == nil {
		logger = slog.Default()
	}

	return logger.With(
		"session_id", strings.TrimSpace(meta.ID),
		"agent_name", strings.TrimSpace(meta.AgentName),
		"workspace_id", strings.TrimSpace(meta.WorkspaceID),
	)
}

func resumeStopReasonPointer(reason store.StopReason) *store.StopReason {
	value := reason
	return &value
}
