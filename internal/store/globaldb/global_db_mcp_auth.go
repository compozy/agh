package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	mcpauth "github.com/compozy/agh/internal/mcp/auth"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
	"github.com/compozy/agh/internal/vault"
)

const (
	globalDBMCPAuthBearerValue = "Bearer"
)

var _ mcpauth.TokenStore = (*VaultRepo)(nil)

// SaveMCPAuthToken persists one remote MCP OAuth token record.
func (g *VaultRepo) SaveMCPAuthToken(ctx context.Context, token mcpauth.TokenRecord) error {
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

		err = sqlcgen.New(tx).UpsertMCPAuthToken(ctx, sqlcgen.UpsertMCPAuthTokenParams{
			ServerName: normalized.ServerName, Issuer: normalized.Issuer, ClientID: normalized.ClientID,
			ScopesJson: string(scopesJSON), AccessTokenRef: accessTokenRef, RefreshTokenRef: refreshTokenRef,
			TokenType: normalized.TokenType, ExpiresAt: nullableMCPTime(normalized.ExpiresAt),
			ObtainedAt: store.FormatTimestamp(normalized.ObtainedAt),
			UpdatedAt:  store.FormatTimestamp(normalized.UpdatedAt),
		})
		if err != nil {
			return fmt.Errorf("store: save MCP auth token for %q: %w", normalized.ServerName, err)
		}
		return nil
	})
}

// GetMCPAuthToken returns one persisted token record.
func (g *VaultRepo) GetMCPAuthToken(ctx context.Context, serverName string) (mcpauth.TokenRecord, error) {
	if err := g.checkReady(ctx, "get MCP auth token"); err != nil {
		return mcpauth.TokenRecord{}, err
	}
	name := strings.TrimSpace(serverName)
	if name == "" {
		return mcpauth.TokenRecord{}, errors.New("store: MCP auth token server name is required")
	}

	row, err := g.queries.GetMCPAuthToken(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mcpauth.TokenRecord{}, mcpauth.ErrTokenNotFound
		}
		return mcpauth.TokenRecord{}, fmt.Errorf("store: get MCP auth token for %q: %w", name, err)
	}
	token, err := mcpAuthTokenFromGenerated(row)
	if err != nil {
		return mcpauth.TokenRecord{}, err
	}
	if err := g.resolveMCPAuthTokenSecrets(ctx, &token); err != nil {
		return mcpauth.TokenRecord{}, err
	}
	return token, nil
}

// ListMCPAuthTokens returns all persisted token records.
func (g *VaultRepo) ListMCPAuthTokens(ctx context.Context) ([]mcpauth.TokenRecord, error) {
	if err := g.checkReady(ctx, "list MCP auth tokens"); err != nil {
		return nil, err
	}
	rows, err := g.queries.ListMCPAuthTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list MCP auth tokens: %w", err)
	}

	tokens := make([]mcpauth.TokenRecord, 0, len(rows))
	for _, row := range rows {
		token, mapErr := mcpAuthTokenFromGenerated(row)
		if mapErr != nil {
			return nil, mapErr
		}
		if err := g.resolveMCPAuthTokenSecrets(ctx, &token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// DeleteMCPAuthToken removes persisted token state for one server.
func (g *VaultRepo) DeleteMCPAuthToken(ctx context.Context, serverName string) error {
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
		if _, err := sqlcgen.New(tx).DeleteMCPAuthToken(ctx, name); err != nil {
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

func (g *VaultRepo) resolveMCPAuthTokenSecrets(ctx context.Context, token *mcpauth.TokenRecord) error {
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
	name := strings.TrimSpace(serverName)
	row, err := sqlcgen.New(exec).GetMCPAuthTokenRefs(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", mcpauth.ErrTokenNotFound
		}
		return "", "", fmt.Errorf("store: get MCP auth token refs for %q: %w", name, err)
	}
	return strings.TrimSpace(row.AccessTokenRef), strings.TrimSpace(row.RefreshTokenRef), nil
}

func mcpAuthTokenSecretRef(serverName string, fieldName string) string {
	return "vault:mcp/" + strings.TrimSpace(serverName) + "/oauth/" + strings.TrimSpace(fieldName)
}

func deleteMCPRefreshTokenSecret(ctx context.Context, service *vault.Service, serverName string) error {
	return service.DeleteSecret(ctx, mcpAuthTokenSecretRef(serverName, "refresh-token"))
}

func nullableMCPTime(value time.Time) sql.NullString {
	if value.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: store.FormatTimestamp(value), Valid: true}
}

func mcpAuthTokenFromGenerated(row sqlcgen.McpAuthToken) (mcpauth.TokenRecord, error) {
	token := mcpauth.TokenRecord{
		ServerName: row.ServerName, Issuer: row.Issuer, ClientID: row.ClientID,
		AccessToken: row.AccessTokenRef, RefreshToken: row.RefreshTokenRef, TokenType: row.TokenType,
	}
	if strings.TrimSpace(row.ScopesJson) != "" {
		if err := json.Unmarshal([]byte(row.ScopesJson), &token.Scopes); err != nil {
			return mcpauth.TokenRecord{}, fmt.Errorf("store: decode MCP auth token scopes: %w", err)
		}
	}
	if row.ExpiresAt.Valid && strings.TrimSpace(row.ExpiresAt.String) != "" {
		expiresAt, err := store.ParseTimestamp(row.ExpiresAt.String)
		if err != nil {
			return mcpauth.TokenRecord{}, fmt.Errorf("store: parse MCP auth token expires_at: %w", err)
		}
		token.ExpiresAt = expiresAt
	}
	obtainedAt, err := store.ParseTimestamp(row.ObtainedAt)
	if err != nil {
		return mcpauth.TokenRecord{}, fmt.Errorf("store: parse MCP auth token obtained_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(row.UpdatedAt)
	if err != nil {
		return mcpauth.TokenRecord{}, fmt.Errorf("store: parse MCP auth token updated_at: %w", err)
	}
	token.ObtainedAt, token.UpdatedAt = obtainedAt, updatedAt
	return token, nil
}
