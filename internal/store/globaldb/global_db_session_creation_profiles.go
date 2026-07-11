package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

var _ store.SessionCreationStore = (*GlobalDB)(nil)

// PutSessionCreationProfile persists one immutable content-addressed profile.
func (g *GlobalDB) PutSessionCreationProfile(
	ctx context.Context,
	profile store.SessionCreationProfile,
) (string, error) {
	if err := g.checkReady(ctx, "put session creation profile"); err != nil {
		return "", err
	}
	profile = store.NormalizeSessionCreationProfile(profile)
	profileRef, err := profile.Ref()
	if err != nil {
		return "", err
	}
	payload, err := profile.CanonicalJSON()
	if err != nil {
		return "", err
	}
	err = g.withTaskImmediateTransaction(ctx, "put session creation profile", func(exec taskSQLExecutor) error {
		if _, err := exec.ExecContext(
			ctx,
			`INSERT INTO session_creation_profiles (profile_ref, profile_json, created_at)
			 VALUES (?, ?, ?) ON CONFLICT(profile_ref) DO NOTHING`,
			profileRef,
			string(payload),
			store.FormatTimestamp(g.now()),
		); err != nil {
			return fmt.Errorf("store: insert session creation profile %q: %w", profileRef, err)
		}
		var storedJSON string
		if err := exec.QueryRowContext(
			ctx,
			`SELECT profile_json FROM session_creation_profiles WHERE profile_ref = ?`,
			profileRef,
		).Scan(&storedJSON); err != nil {
			return fmt.Errorf("store: read persisted session creation profile %q: %w", profileRef, err)
		}
		if strings.TrimSpace(storedJSON) != string(payload) {
			return fmt.Errorf("store: session creation profile ref collision: %s", profileRef)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return profileRef, nil
}

// GetSessionCreationProfile loads and verifies one immutable profile.
func (g *GlobalDB) GetSessionCreationProfile(
	ctx context.Context,
	profileRef string,
) (store.SessionCreationProfile, error) {
	if err := g.checkReady(ctx, "get session creation profile"); err != nil {
		return store.SessionCreationProfile{}, err
	}
	profileRef = strings.TrimSpace(profileRef)
	if profileRef == "" {
		return store.SessionCreationProfile{}, fmt.Errorf("store: session creation profile ref is required")
	}
	var payload string
	err := g.db.QueryRowContext(
		ctx,
		`SELECT profile_json FROM session_creation_profiles WHERE profile_ref = ?`,
		profileRef,
	).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return store.SessionCreationProfile{}, fmt.Errorf("%w: %s", store.ErrSessionNotFound, profileRef)
	}
	if err != nil {
		return store.SessionCreationProfile{}, fmt.Errorf("store: get session creation profile %q: %w", profileRef, err)
	}
	var profile store.SessionCreationProfile
	if err := json.Unmarshal([]byte(payload), &profile); err != nil {
		return store.SessionCreationProfile{}, fmt.Errorf(
			"store: decode session creation profile %q: %w",
			profileRef,
			err,
		)
	}
	profile = store.NormalizeSessionCreationProfile(profile)
	computedRef, err := profile.Ref()
	if err != nil {
		return store.SessionCreationProfile{}, err
	}
	if computedRef != profileRef {
		return store.SessionCreationProfile{}, fmt.Errorf(
			"store: session creation profile %q failed content-address verification",
			profileRef,
		)
	}
	return profile, nil
}
