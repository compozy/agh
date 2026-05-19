package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/vault"
)

const (
	globalDBMCPAuthBearerValue = "Bearer"
)

var _ mcpauth.TokenStore = (*GlobalDB)(nil)

// SaveMCPAuthToken persists one remote MCP OAuth token record.
func (g *GlobalDB) SaveMCPAuthToken(ctx context.Context, token mcpauth.TokenRecord) error {
	if err := g.checkReady(ctx, "save MCP auth token"); err != nil {
		return err
	}
	normalized, err := normalizeMCPAuthToken(token, g.now())
	if err != nil {
		return err
	}
	scopesJSON, err := json.Marshal(normalized.Scopes)
	if err != nil {
		return fmt.Errorf("store: marshal MCP auth token scopes: %w", err)
	}
	accessTokenRef := mcpAuthTokenSecretRef(normalized.ServerName, "access-token")
	refreshTokenRef := ""
	if strings.TrimSpace(normalized.RefreshToken) != "" {
		refreshTokenRef = mcpAuthTokenSecretRef(normalized.ServerName, "refresh-token")
	}

	return store.ExecuteWrite(ctx, g.db, func(ctx context.Context, tx *store.WriteTx) error {
		service, err := g.vaultServiceForStore(transactionVaultStore{owner: g, exec: tx})
		if err != nil {
			return err
		}
		if _, err := service.PutSecret(
			ctx,
			accessTokenRef,
			"mcp_oauth_access_token",
			normalized.AccessToken,
		); err != nil {
			return fmt.Errorf("store: save MCP auth access token for %q: %w", normalized.ServerName, err)
		}
		if refreshTokenRef != "" {
			if _, err := service.PutSecret(
				ctx,
				refreshTokenRef,
				"mcp_oauth_refresh_token",
				normalized.RefreshToken,
			); err != nil {
				return fmt.Errorf("store: save MCP auth refresh token for %q: %w", normalized.ServerName, err)
			}
		} else if err := deleteMCPRefreshTokenSecret(ctx, service, normalized.ServerName); err != nil &&
			!errors.Is(err, vault.ErrSecretNotFound) {
			return fmt.Errorf("store: clear MCP auth refresh token for %q: %w", normalized.ServerName, err)
		}

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO mcp_auth_tokens (
				server_name, issuer, client_id, scopes_json, access_token_ref, refresh_token_ref,
				token_type, expires_at, obtained_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(server_name) DO UPDATE SET
				issuer = excluded.issuer,
				client_id = excluded.client_id,
				scopes_json = excluded.scopes_json,
				access_token_ref = excluded.access_token_ref,
				refresh_token_ref = excluded.refresh_token_ref,
				token_type = excluded.token_type,
				expires_at = excluded.expires_at,
				obtained_at = excluded.obtained_at,
				updated_at = excluded.updated_at`,
			normalized.ServerName,
			normalized.Issuer,
			normalized.ClientID,
			string(scopesJSON),
			accessTokenRef,
			refreshTokenRef,
			normalized.TokenType,
			nullableTime(normalized.ExpiresAt),
			store.FormatTimestamp(normalized.ObtainedAt),
			store.FormatTimestamp(normalized.UpdatedAt),
		)
		if err != nil {
			return fmt.Errorf("store: save MCP auth token for %q: %w", normalized.ServerName, err)
		}
		return nil
	})
}

// GetMCPAuthToken returns one persisted token record.
func (g *GlobalDB) GetMCPAuthToken(ctx context.Context, serverName string) (mcpauth.TokenRecord, error) {
	if err := g.checkReady(ctx, "get MCP auth token"); err != nil {
		return mcpauth.TokenRecord{}, err
	}
	name := strings.TrimSpace(serverName)
	if name == "" {
		return mcpauth.TokenRecord{}, errors.New("store: MCP auth token server name is required")
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT server_name, issuer, client_id, scopes_json, access_token_ref, refresh_token_ref,
			token_type, expires_at, obtained_at, updated_at
		FROM mcp_auth_tokens
		WHERE server_name = ?`,
		name,
	)
	token, err := scanMCPAuthToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mcpauth.TokenRecord{}, mcpauth.ErrTokenNotFound
		}
		return mcpauth.TokenRecord{}, err
	}
	if err := g.resolveMCPAuthTokenSecrets(ctx, &token); err != nil {
		return mcpauth.TokenRecord{}, err
	}
	return token, nil
}

// ListMCPAuthTokens returns all persisted token records.
func (g *GlobalDB) ListMCPAuthTokens(ctx context.Context) ([]mcpauth.TokenRecord, error) {
	if err := g.checkReady(ctx, "list MCP auth tokens"); err != nil {
		return nil, err
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT server_name, issuer, client_id, scopes_json, access_token_ref, refresh_token_ref,
			token_type, expires_at, obtained_at, updated_at
		FROM mcp_auth_tokens
		ORDER BY server_name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list MCP auth tokens: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	tokens := make([]mcpauth.TokenRecord, 0)
	for rows.Next() {
		token, scanErr := scanMCPAuthToken(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if err := g.resolveMCPAuthTokenSecrets(ctx, &token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate MCP auth tokens: %w", err)
	}
	return tokens, nil
}

// DeleteMCPAuthToken removes persisted token state for one server.
func (g *GlobalDB) DeleteMCPAuthToken(ctx context.Context, serverName string) error {
	if err := g.checkReady(ctx, "delete MCP auth token"); err != nil {
		return err
	}
	name := strings.TrimSpace(serverName)
	if name == "" {
		return errors.New("store: MCP auth token server name is required")
	}
	return store.ExecuteWrite(ctx, g.db, func(ctx context.Context, tx *store.WriteTx) error {
		accessTokenRef, refreshTokenRef, err := getMCPAuthTokenRefsWithExecutor(ctx, tx, name)
		if err != nil && !errors.Is(err, mcpauth.ErrTokenNotFound) {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM mcp_auth_tokens WHERE server_name = ?`, name); err != nil {
			return fmt.Errorf("store: delete MCP auth token for %q: %w", name, err)
		}
		service, err := g.vaultServiceForStore(transactionVaultStore{owner: g, exec: tx})
		if err != nil {
			return err
		}
		for _, ref := range []string{accessTokenRef, refreshTokenRef} {
			if strings.TrimSpace(ref) == "" {
				continue
			}
			if err := service.DeleteSecret(ctx, ref); err != nil && !errors.Is(err, vault.ErrSecretNotFound) {
				return fmt.Errorf("store: delete MCP auth token secret for %q: %w", name, err)
			}
		}
		return nil
	})
}

func normalizeMCPAuthToken(token mcpauth.TokenRecord, now time.Time) (mcpauth.TokenRecord, error) {
	token.ServerName = strings.TrimSpace(token.ServerName)
	token.Issuer = strings.TrimSpace(token.Issuer)
	token.ClientID = strings.TrimSpace(token.ClientID)
	token.AccessToken = strings.TrimSpace(token.AccessToken)
	token.RefreshToken = strings.TrimSpace(token.RefreshToken)
	token.TokenType = strings.TrimSpace(token.TokenType)
	if token.TokenType == "" {
		token.TokenType = globalDBMCPAuthBearerValue
	}
	token.Scopes = trimTokenScopes(token.Scopes)
	if token.ObtainedAt.IsZero() {
		token.ObtainedAt = now.UTC()
	}
	if token.UpdatedAt.IsZero() {
		token.UpdatedAt = now.UTC()
	}
	switch {
	case token.ServerName == "":
		return mcpauth.TokenRecord{}, errors.New("store: MCP auth token server name is required")
	case token.ClientID == "":
		return mcpauth.TokenRecord{}, errors.New("store: MCP auth token client id is required")
	case token.AccessToken == "":
		return mcpauth.TokenRecord{}, errors.New("store: MCP auth token access token is required")
	default:
		return token, nil
	}
}

func trimTokenScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		if value := strings.TrimSpace(scope); value != "" {
			trimmed = append(trimmed, value)
		}
	}
	return trimmed
}

func (g *GlobalDB) resolveMCPAuthTokenSecrets(ctx context.Context, token *mcpauth.TokenRecord) error {
	if token == nil {
		return nil
	}
	service, err := g.vaultService()
	if err != nil {
		return err
	}
	accessToken, err := resolveMCPAuthTokenRef(ctx, service, token.ServerName, "access_token", token.AccessToken)
	if err != nil {
		return err
	}
	refreshToken, err := resolveMCPAuthTokenRef(ctx, service, token.ServerName, "refresh_token", token.RefreshToken)
	if err != nil {
		return err
	}
	token.AccessToken = accessToken
	token.RefreshToken = refreshToken
	return nil
}

func resolveMCPAuthTokenRef(
	ctx context.Context,
	service *vault.Service,
	serverName string,
	fieldName string,
	ref string,
) (string, error) {
	trimmedRef := strings.TrimSpace(ref)
	if trimmedRef == "" {
		return "", nil
	}
	if err := vault.ValidateSecretRefNamespace(trimmedRef, "mcp"); err != nil {
		return "", fmt.Errorf(
			"store: MCP auth token %s for %q is not a vault ref: %w",
			fieldName,
			strings.TrimSpace(serverName),
			err,
		)
	}
	value, err := service.ResolveRef(ctx, trimmedRef)
	if err != nil {
		return "", fmt.Errorf(
			"store: resolve MCP auth token %s for %q: %w",
			fieldName,
			strings.TrimSpace(serverName),
			err,
		)
	}
	return value, nil
}

func getMCPAuthTokenRefsWithExecutor(
	ctx context.Context,
	exec globalSQLExecutor,
	serverName string,
) (string, string, error) {
	var accessTokenRef string
	var refreshTokenRef string
	name := strings.TrimSpace(serverName)
	err := exec.QueryRowContext(
		ctx,
		`SELECT access_token_ref, refresh_token_ref
		FROM mcp_auth_tokens
		WHERE server_name = ?`,
		name,
	).Scan(&accessTokenRef, &refreshTokenRef)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", mcpauth.ErrTokenNotFound
		}
		return "", "", fmt.Errorf("store: get MCP auth token refs for %q: %w", name, err)
	}
	return strings.TrimSpace(accessTokenRef), strings.TrimSpace(refreshTokenRef), nil
}

func mcpAuthTokenSecretRef(serverName string, fieldName string) string {
	return "vault:mcp/" + strings.TrimSpace(serverName) + "/oauth/" + strings.TrimSpace(fieldName)
}

func deleteMCPRefreshTokenSecret(ctx context.Context, service *vault.Service, serverName string) error {
	return service.DeleteSecret(ctx, mcpAuthTokenSecretRef(serverName, "refresh-token"))
}

func scanMCPAuthToken(scanner rowScanner) (mcpauth.TokenRecord, error) {
	var (
		token         mcpauth.TokenRecord
		scopesRaw     string
		expiresAtRaw  sql.NullString
		obtainedAtRaw string
		updatedAtRaw  string
	)
	if err := scanner.Scan(
		&token.ServerName,
		&token.Issuer,
		&token.ClientID,
		&scopesRaw,
		&token.AccessToken,
		&token.RefreshToken,
		&token.TokenType,
		&expiresAtRaw,
		&obtainedAtRaw,
		&updatedAtRaw,
	); err != nil {
		return mcpauth.TokenRecord{}, fmt.Errorf("store: scan MCP auth token: %w", err)
	}
	if strings.TrimSpace(scopesRaw) != "" {
		if err := json.Unmarshal([]byte(scopesRaw), &token.Scopes); err != nil {
			return mcpauth.TokenRecord{}, fmt.Errorf("store: decode MCP auth token scopes: %w", err)
		}
	}
	if expiresAtRaw.Valid && strings.TrimSpace(expiresAtRaw.String) != "" {
		expiresAt, err := store.ParseTimestamp(expiresAtRaw.String)
		if err != nil {
			return mcpauth.TokenRecord{}, fmt.Errorf("store: parse MCP auth token expires_at: %w", err)
		}
		token.ExpiresAt = expiresAt
	}
	obtainedAt, err := store.ParseTimestamp(obtainedAtRaw)
	if err != nil {
		return mcpauth.TokenRecord{}, fmt.Errorf("store: parse MCP auth token obtained_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return mcpauth.TokenRecord{}, fmt.Errorf("store: parse MCP auth token updated_at: %w", err)
	}
	token.ObtainedAt = obtainedAt
	token.UpdatedAt = updatedAt
	return token, nil
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return store.FormatTimestamp(value)
}
