package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
)

// RegisterSessionWithCreationIdentity atomically creates or refreshes one identity-matched session.
func (g *GlobalDB) RegisterSessionWithCreationIdentity(
	ctx context.Context,
	session store.SessionInfo,
	identity store.SessionCreationIdentity,
) (store.SessionCreationRegistration, error) {
	if err := g.checkReady(ctx, "register session with creation identity"); err != nil {
		return store.SessionCreationRegistration{}, err
	}
	if err := session.Validate(); err != nil {
		return store.SessionCreationRegistration{}, err
	}
	if err := identity.Validate(); err != nil {
		return store.SessionCreationRegistration{}, err
	}
	normalized := session
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}

	registration := store.SessionCreationRegistration{
		SessionID: normalized.ID,
		Identity:  identity,
	}
	err := g.withTaskImmediateTransaction(ctx, "register session creation identity", func(exec taskSQLExecutor) error {
		storedWorkspace, storedIdentity, found, err := readSessionCreationIdentity(ctx, exec, normalized.ID)
		if err != nil {
			return err
		}
		if found {
			if strings.TrimSpace(storedWorkspace) != strings.TrimSpace(normalized.WorkspaceID) ||
				storedIdentity != identity || !storedIdentityComplete(storedIdentity) {
				return sessionCreationIdentityMismatch(normalized.ID)
			}
			if err := g.registerSession(ctx, exec, normalized); err != nil {
				return fmt.Errorf("store: refresh identity-matched session %q: %w", normalized.ID, err)
			}
			return nil
		}

		if err := g.registerSession(ctx, exec, normalized); err != nil {
			return fmt.Errorf("store: register identity-bound session %q: %w", normalized.ID, err)
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE sessions
			 SET creation_profile_ref = ?, policy_spec_digest = ?, creation_digest = ?
			 WHERE id = ?
			   AND creation_profile_ref IS NULL
			   AND policy_spec_digest IS NULL
			   AND creation_digest IS NULL`,
			identity.CreationProfileRef,
			identity.PolicySpecDigest,
			identity.CreationDigest,
			normalized.ID,
		)
		if err != nil {
			return fmt.Errorf("store: persist session creation identity %q: %w", normalized.ID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("store: read session creation identity update count %q: %w", normalized.ID, err)
		}
		if affected != 1 {
			return sessionCreationIdentityMismatch(normalized.ID)
		}
		registration.Created = true
		return nil
	})
	if err != nil {
		return store.SessionCreationRegistration{}, err
	}
	return registration, nil
}

// GetSessionCreationIdentity loads one complete immutable identity witness.
func (g *GlobalDB) GetSessionCreationIdentity(
	ctx context.Context,
	sessionID string,
) (store.SessionCreationIdentity, error) {
	if err := g.checkReady(ctx, "get session creation identity"); err != nil {
		return store.SessionCreationIdentity{}, err
	}
	_, identity, found, err := readSessionCreationIdentity(ctx, g.db, strings.TrimSpace(sessionID))
	if err != nil {
		return store.SessionCreationIdentity{}, err
	}
	if !found {
		return store.SessionCreationIdentity{}, fmt.Errorf("%w: %s", store.ErrSessionNotFound, sessionID)
	}
	if !storedIdentityComplete(identity) {
		return store.SessionCreationIdentity{}, sessionCreationIdentityMismatch(sessionID)
	}
	return identity, nil
}

func readSessionCreationIdentity(
	ctx context.Context,
	exec taskSQLExecutor,
	sessionID string,
) (string, store.SessionCreationIdentity, bool, error) {
	var (
		workspaceID        string
		creationProfileRef sql.NullString
		policySpecDigest   sql.NullString
		creationDigest     sql.NullString
	)
	err := exec.QueryRowContext(
		ctx,
		`SELECT workspace_id, creation_profile_ref, policy_spec_digest, creation_digest
		 FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&workspaceID, &creationProfileRef, &policySpecDigest, &creationDigest)
	if errors.Is(err, sql.ErrNoRows) {
		return "", store.SessionCreationIdentity{}, false, nil
	}
	if err != nil {
		return "", store.SessionCreationIdentity{}, false, fmt.Errorf(
			"store: read session creation identity %q: %w",
			sessionID,
			err,
		)
	}
	return workspaceID, store.SessionCreationIdentity{
		CreationProfileRef: strings.TrimSpace(creationProfileRef.String),
		PolicySpecDigest:   strings.TrimSpace(policySpecDigest.String),
		CreationDigest:     strings.TrimSpace(creationDigest.String),
	}, true, nil
}

func storedIdentityComplete(identity store.SessionCreationIdentity) bool {
	return identity.Validate() == nil
}

func sessionCreationIdentityMismatch(sessionID string) error {
	return &looppkg.ReasonError{
		Code: looppkg.ReasonCodeSessionCreationIdentityMismatch,
		Err:  fmt.Errorf("%w: %s", store.ErrSessionCreationIdentityMismatch, sessionID),
	}
}
