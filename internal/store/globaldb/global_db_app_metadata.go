package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

const onboardingCompletedAtKey = "onboarding.completed_at"

// GetAppMetadata returns the value stored under key and whether the key exists.
func (g *GlobalDB) GetAppMetadata(ctx context.Context, key string) (string, bool, error) {
	if err := g.checkReady(ctx, "get app metadata"); err != nil {
		return "", false, err
	}
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", false, errors.New("store: app metadata key is required")
	}

	var value string
	err := g.db.QueryRowContext(
		ctx,
		`SELECT value FROM app_metadata WHERE key = ?`,
		trimmed,
	).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("store: query app metadata %q: %w", trimmed, err)
	}
	return value, true, nil
}

// SetAppMetadata upserts the value stored under key.
func (g *GlobalDB) SetAppMetadata(ctx context.Context, key string, value string) error {
	if err := g.checkReady(ctx, "set app metadata"); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return errors.New("store: app metadata key is required")
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO app_metadata (key, value, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET
				value = excluded.value,
				updated_at = excluded.updated_at`,
		trimmed,
		value,
		store.FormatTimestamp(g.now()),
	); err != nil {
		return fmt.Errorf("store: upsert app metadata %q: %w", trimmed, err)
	}
	return nil
}

// GetOnboardingStatus returns the domain status backed by app metadata.
func (g *GlobalDB) GetOnboardingStatus(ctx context.Context) (store.OnboardingStatus, error) {
	completedAt, found, err := g.GetAppMetadata(ctx, onboardingCompletedAtKey)
	if err != nil {
		return store.OnboardingStatus{}, err
	}
	return store.OnboardingStatus{Completed: found, CompletedAt: completedAt}, nil
}

// CompleteOnboarding stores the first completion timestamp and preserves it on repeat calls.
func (g *GlobalDB) CompleteOnboarding(ctx context.Context, completedAt string) (store.OnboardingStatus, error) {
	if err := g.checkReady(ctx, "complete onboarding"); err != nil {
		return store.OnboardingStatus{}, err
	}
	trimmed := strings.TrimSpace(completedAt)
	if trimmed == "" {
		return store.OnboardingStatus{}, errors.New("store: onboarding completed_at is required")
	}

	var status store.OnboardingStatus
	err := g.withImmediateTransaction(ctx, "complete onboarding", func(exec globalSQLExecutor) error {
		if _, err := exec.ExecContext(
			ctx,
			`INSERT INTO app_metadata (key, value, updated_at)
				VALUES (?, ?, ?)
				ON CONFLICT(key) DO NOTHING`,
			onboardingCompletedAtKey,
			trimmed,
			store.FormatTimestamp(g.now()),
		); err != nil {
			return fmt.Errorf("store: complete onboarding: %w", err)
		}
		var storedAt string
		if err := exec.QueryRowContext(
			ctx,
			`SELECT value FROM app_metadata WHERE key = ?`,
			onboardingCompletedAtKey,
		).Scan(&storedAt); err != nil {
			return fmt.Errorf("store: read completed onboarding status: %w", err)
		}
		status = store.OnboardingStatus{Completed: true, CompletedAt: storedAt}
		return nil
	})
	if err != nil {
		return store.OnboardingStatus{}, err
	}
	return status, nil
}

// ResetOnboarding clears first-run completion so onboarding is shown again.
func (g *GlobalDB) ResetOnboarding(ctx context.Context) (store.OnboardingStatus, error) {
	if err := g.DeleteAppMetadata(ctx, onboardingCompletedAtKey); err != nil {
		return store.OnboardingStatus{}, err
	}
	return store.OnboardingStatus{Completed: false}, nil
}

// DeleteAppMetadata removes the value stored under key. Missing keys are a no-op.
func (g *GlobalDB) DeleteAppMetadata(ctx context.Context, key string) error {
	if err := g.checkReady(ctx, "delete app metadata"); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return errors.New("store: app metadata key is required")
	}

	if _, err := g.db.ExecContext(ctx, `DELETE FROM app_metadata WHERE key = ?`, trimmed); err != nil {
		return fmt.Errorf("store: delete app metadata %q: %w", trimmed, err)
	}
	return nil
}
