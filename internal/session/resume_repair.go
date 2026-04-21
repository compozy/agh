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

func (m *Manager) repairInactiveMeta(
	ctx context.Context,
	metaPath string,
	meta store.SessionMeta,
) (store.SessionMeta, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	repaired, err := m.repairLegacyProvider(ctx, metaPath, meta)
	if err != nil {
		return store.SessionMeta{}, err
	}

	classified, changed := ClassifyInactiveMetaForRecovery(m.now(), repaired)
	if !changed {
		return repaired, nil
	}
	return m.persistResumeCrashClassification(metaPath, classified)
}

func (m *Manager) repairLegacyProvider(
	ctx context.Context,
	metaPath string,
	meta store.SessionMeta,
) (store.SessionMeta, error) {
	if strings.TrimSpace(meta.Provider) != "" {
		return meta, nil
	}

	resolvedWorkspace, err := m.resolveResumeWorkspace(ctx, meta)
	if err != nil {
		m.resumeLogger(meta).Warn(
			"session.resume.legacy_provider_repair_failed",
			"phase", "legacy_repair",
			"error", err,
		)
		return store.SessionMeta{}, fmt.Errorf("session: repair provider for %q: %w", strings.TrimSpace(meta.ID), err)
	}

	resolved, err := m.resolveWorkspaceSessionAgent(meta.AgentName, "", &resolvedWorkspace)
	if err != nil {
		m.resumeLogger(meta).Warn(
			"session.resume.legacy_provider_repair_failed",
			"phase", "legacy_repair",
			"error", err,
		)
		return store.SessionMeta{}, fmt.Errorf(
			"session: repair provider for %q agent %q: %w",
			strings.TrimSpace(meta.ID),
			strings.TrimSpace(meta.AgentName),
			err,
		)
	}

	repaired := meta
	repaired.Provider = strings.TrimSpace(resolved.Provider)
	repaired.UpdatedAt = m.now()
	if err := store.WriteSessionMeta(metaPath, repaired); err != nil {
		m.resumeLogger(repaired).Warn(
			"session.resume.legacy_provider_repair_persist_failed",
			"phase", "legacy_repair",
			"error", err,
		)
		return store.SessionMeta{}, fmt.Errorf(
			"session: persist repaired provider for %q: %w",
			strings.TrimSpace(meta.ID),
			err,
		)
	}

	m.resumeLogger(repaired).Info(
		"session.resume.legacy_provider_repaired",
		"phase", "legacy_repair",
	)
	return repaired, nil
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

			if agentErr := m.validateResumeAgent(meta.AgentName, meta.Provider, &resolvedWorkspace); agentErr != nil {
				errs = append(errs, resumeValidationError{
					check: resumeValidationCheckAgent,
					err: fmt.Errorf(
						"session: validate agent %q with provider %q for session %q: %w",
						strings.TrimSpace(meta.AgentName),
						strings.TrimSpace(meta.Provider),
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

func (m *Manager) validateResumeAgent(
	agentName string,
	provider string,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
) error {
	if _, err := m.resolveWorkspaceSessionAgent(agentName, provider, resolvedWorkspace); err != nil {
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
		annotated := AnnotateUnpersistedRecovery(classified, err)
		m.resumeLogger(annotated).Warn(
			"session.resume.crash_classification_persist_failed",
			"phase", "resume",
			"previous_state", strings.TrimSpace(meta.State),
			"stop_reason", sessionMetaStopReason(annotated),
			"stop_detail", strings.TrimSpace(annotated.StopDetail),
			"error", err,
		)
		return annotated, nil
	}

	reason := ""
	if classified.StopReason != nil {
		reason = string(*classified.StopReason)
	}
	m.resumeLogger(classified).Info(
		"session.resume.crash_classified",
		"phase", "resume",
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

		logger.Warn("session.resume.validation_failed", "phase", "resume", "check", check, "error", err)
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
		"provider", strings.TrimSpace(meta.Provider),
		"workspace_id", strings.TrimSpace(meta.WorkspaceID),
	)
}

func resumeStopReasonPointer(reason store.StopReason) *store.StopReason {
	value := reason
	return &value
}
