package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/fileutil"
	"github.com/pedronauck/agh/internal/store"
)

type sessionDBBackup struct {
	original string
	backup   string
}

// ClearConversation resets the persisted transcript and ACP conversation context
// for an existing session while preserving the same session id. The session is
// restarted on a fresh event store so subsequent prompts start from a clean
// conversation.
func (m *Manager) ClearConversation(ctx context.Context, id string) (_ *Session, err error) {
	if m == nil {
		return nil, errors.New("session: manager is required")
	}
	if ctx == nil {
		return nil, errors.New("session: clear conversation context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return nil, errors.New("session: session id is required")
	}
	if m.isPending(target) {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotActive, target)
	}

	active, activeBefore := m.Get(target)
	if activeBefore {
		if active.IsPrompting() {
			return nil, fmt.Errorf("%w: %s", ErrPromptInProgress, target)
		}
		if stopErr := m.Stop(ctx, target); stopErr != nil {
			return nil, stopErr
		}
	}

	meta, err := m.readMetaWithContext(ctx, target)
	if err != nil {
		return nil, err
	}

	sanitized := clearedConversationMeta(meta, m.now())
	spec, err := m.prepareResumeStart(ctx, sanitized)
	if err != nil {
		return nil, err
	}

	dbPath := store.SessionDBFile(filepath.Join(m.homePaths.SessionsDir, target))
	metaPath := store.SessionMetaFile(filepath.Join(m.homePaths.SessionsDir, target))
	backups, err := backupSessionDB(dbPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			if cleanupErr := discardSessionDBBackup(backups); cleanupErr != nil {
				err = cleanupErr
			}
			return
		}
		if restoreErr := restoreClearedConversationFailure(backups, dbPath, metaPath, meta); restoreErr != nil {
			err = errors.Join(err, restoreErr)
		}
	}()

	if writeErr := store.WriteSessionMeta(metaPath, sanitized); writeErr != nil {
		return nil, fmt.Errorf("session: persist cleared metadata for %q: %w", target, writeErr)
	}

	session, err := m.startSession(ctx, &spec)
	if err != nil {
		return nil, err
	}
	if commitErr := commitSessionDBClear(dbPath); commitErr != nil {
		return nil, fmt.Errorf("session: commit cleared event store for %q: %w", target, commitErr)
	}

	return session, nil
}

func clearedConversationMeta(meta store.SessionMeta, now time.Time) store.SessionMeta {
	cleared := meta
	cleared.State = string(StateStopped)
	cleared.StopReason = nil
	cleared.StopDetail = ""
	cleared.ACPSessionID = nil
	cleared.UpdatedAt = now
	return cleared
}

func backupSessionDB(path string) ([]sessionDBBackup, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, errors.New("session: session database path is required")
	}
	if err := recoverSessionDBClear(cleanPath); err != nil {
		return nil, fmt.Errorf("session: recover previous clear state before backup: %w", err)
	}

	paths := []string{cleanPath, cleanPath + "-wal", cleanPath + "-shm"}
	backups := make([]sessionDBBackup, 0, len(paths))

	for _, original := range paths {
		if _, err := os.Stat(original); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, rollbackSessionDBBackup(
				backups,
				fmt.Errorf("session: stat event store artifact %q: %w", original, err),
			)
		}

		backup := original + ".clear-backup"
		if err := removeSessionDBArtifact(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, rollbackSessionDBBackup(
				backups,
				fmt.Errorf("session: remove stale clear backup %q: %w", backup, err),
			)
		}
		if err := os.Rename(original, backup); err != nil {
			return nil, rollbackSessionDBBackup(
				backups,
				fmt.Errorf("session: backup event store artifact %q: %w", original, err),
			)
		}
		backups = append(backups, sessionDBBackup{original: original, backup: backup})
		if err := syncSessionDBDir(original); err != nil {
			return nil, rollbackSessionDBBackup(
				backups,
				fmt.Errorf("session: sync clear backup directory for %q: %w", original, err),
			)
		}
	}

	return backups, nil
}

func rollbackSessionDBBackup(backups []sessionDBBackup, primary error) error {
	if len(backups) == 0 {
		return primary
	}
	if rollbackErr := restoreSessionDBArtifacts(backups); rollbackErr != nil {
		return errors.Join(primary, fmt.Errorf("session: rollback clear backup: %w", rollbackErr))
	}
	return primary
}

func discardSessionDBBackup(backups []sessionDBBackup) error {
	var errs []error
	for _, item := range backups {
		target := strings.TrimSpace(item.backup)
		if target == "" {
			continue
		}
		if err := removeSessionDBArtifact(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("session: remove clear backup %q: %w", target, err))
		}
	}
	return errors.Join(errs...)
}

func restoreClearedConversationFailure(
	backups []sessionDBBackup,
	dbPath string,
	metaPath string,
	meta store.SessionMeta,
) error {
	var errs []error

	errs = append(errs, restoreSessionDBArtifacts(backups))
	if err := discardSessionDBClearCommit(dbPath); err != nil {
		errs = append(errs, err)
	}

	if err := store.WriteSessionMeta(metaPath, meta); err != nil {
		errs = append(errs, fmt.Errorf("session: restore cleared metadata for %q: %w", meta.ID, err))
	}

	return errors.Join(errs...)
}

func restoreSessionDBArtifacts(backups []sessionDBBackup) error {
	var errs []error

	for i := len(backups) - 1; i >= 0; i-- {
		item := backups[i]
		if err := removeSessionDBArtifact(item.original); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("session: remove failed clear artifact %q: %w", item.original, err))
		}
		if err := os.Rename(item.backup, item.original); err != nil {
			errs = append(errs, fmt.Errorf("session: restore cleared artifact %q: %w", item.original, err))
			continue
		}
		if err := syncSessionDBDir(item.original); err != nil {
			errs = append(errs, fmt.Errorf("session: sync restored clear artifact %q: %w", item.original, err))
		}
	}

	return errors.Join(errs...)
}

func recoverSessionDBClear(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return errors.New("session: session database path is required")
	}

	backups, err := existingSessionDBClearBackups(cleanPath)
	if err != nil {
		return err
	}
	committed, err := hasSessionDBClearCommit(cleanPath)
	if err != nil {
		return err
	}
	if len(backups) == 0 {
		if committed {
			return discardSessionDBClearCommit(cleanPath)
		}
		return nil
	}
	if committed {
		if err := discardSessionDBBackup(backups); err != nil {
			return fmt.Errorf("session: discard committed clear backups: %w", err)
		}
		return discardSessionDBClearCommit(cleanPath)
	}
	if err := restoreSessionDBArtifacts(backups); err != nil {
		return fmt.Errorf("session: restore interrupted clear backups: %w", err)
	}
	return discardSessionDBClearCommit(cleanPath)
}

func existingSessionDBClearBackups(path string) ([]sessionDBBackup, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, errors.New("session: session database path is required")
	}

	paths := []string{cleanPath, cleanPath + "-wal", cleanPath + "-shm"}
	backups := make([]sessionDBBackup, 0, len(paths))
	for _, original := range paths {
		backup := original + ".clear-backup"
		info, err := os.Lstat(backup)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("session: stat clear backup %q: %w", backup, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		backups = append(backups, sessionDBBackup{original: original, backup: backup})
	}
	return backups, nil
}

func commitSessionDBClear(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return errors.New("session: session database path is required")
	}
	if err := fileutil.AtomicWriteFile(sessionDBClearCommitPath(cleanPath), []byte("committed\n"), 0o600); err != nil {
		return fmt.Errorf("session: write clear commit marker: %w", err)
	}
	return nil
}

func hasSessionDBClearCommit(path string) (bool, error) {
	marker := sessionDBClearCommitPath(path)
	if _, err := os.Stat(marker); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("session: stat clear commit marker %q: %w", marker, err)
	}
	return true, nil
}

func discardSessionDBClearCommit(path string) error {
	marker := sessionDBClearCommitPath(path)
	if err := fileutil.AtomicRemoveFile(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("session: remove clear commit marker %q: %w", marker, err)
	}
	return nil
}

func sessionDBClearCommitPath(path string) string {
	return strings.TrimSpace(path) + ".clear-committed"
}

func removeSessionDBArtifact(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return errors.New("session: session database artifact path is required")
	}
	if err := fileutil.AtomicRemoveFile(cleanPath); err != nil {
		return err
	}
	return nil
}

func syncSessionDBDir(path string) error {
	return fileutil.SyncDir(filepath.Dir(path))
}
