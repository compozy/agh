package globaldb

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptoRand "crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	"github.com/pedronauck/agh/internal/store"
)

var _ mcpauth.TokenStore = (*GlobalDB)(nil)

const (
	mcpAuthSecretKeyBytes      = 32
	mcpAuthSecretEncodingV1    = "v1:"
	mcpAuthSecretKeyFileSuffix = ".mcp-auth.key" // #nosec G101 -- file suffix, not secret material.
	mcpAuthSecretKeyFileMode   = 0o600
)

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
	accessToken, err := g.encryptMCPAuthTokenSecret(normalized.ServerName, "access_token", normalized.AccessToken)
	if err != nil {
		return err
	}
	refreshToken, err := g.encryptMCPAuthTokenSecret(normalized.ServerName, "refresh_token", normalized.RefreshToken)
	if err != nil {
		return err
	}

	_, err = g.db.ExecContext(
		ctx,
		`INSERT INTO mcp_auth_tokens (
			server_name, issuer, client_id, scopes_json, access_token, refresh_token,
			token_type, expires_at, obtained_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(server_name) DO UPDATE SET
			issuer = excluded.issuer,
			client_id = excluded.client_id,
			scopes_json = excluded.scopes_json,
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_type = excluded.token_type,
			expires_at = excluded.expires_at,
			obtained_at = excluded.obtained_at,
			updated_at = excluded.updated_at`,
		normalized.ServerName,
		normalized.Issuer,
		normalized.ClientID,
		string(scopesJSON),
		accessToken,
		refreshToken,
		normalized.TokenType,
		nullableTime(normalized.ExpiresAt),
		store.FormatTimestamp(normalized.ObtainedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("store: save MCP auth token for %q: %w", normalized.ServerName, err)
	}
	return nil
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
		`SELECT server_name, issuer, client_id, scopes_json, access_token, refresh_token,
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
	if err := g.decryptMCPAuthTokenSecrets(&token); err != nil {
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
		`SELECT server_name, issuer, client_id, scopes_json, access_token, refresh_token,
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
		if err := g.decryptMCPAuthTokenSecrets(&token); err != nil {
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
	if _, err := g.db.ExecContext(ctx, `DELETE FROM mcp_auth_tokens WHERE server_name = ?`, name); err != nil {
		return fmt.Errorf("store: delete MCP auth token for %q: %w", name, err)
	}
	return nil
}

func normalizeMCPAuthToken(token mcpauth.TokenRecord, now time.Time) (mcpauth.TokenRecord, error) {
	token.ServerName = strings.TrimSpace(token.ServerName)
	token.Issuer = strings.TrimSpace(token.Issuer)
	token.ClientID = strings.TrimSpace(token.ClientID)
	token.AccessToken = strings.TrimSpace(token.AccessToken)
	token.RefreshToken = strings.TrimSpace(token.RefreshToken)
	token.TokenType = strings.TrimSpace(token.TokenType)
	if token.TokenType == "" {
		token.TokenType = "Bearer"
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

func (g *GlobalDB) encryptMCPAuthTokenSecret(serverName string, fieldName string, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", nil
	}
	aead, err := g.mcpAuthSecretCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := cryptoRand.Read(nonce); err != nil {
		return "", fmt.Errorf("store: generate MCP auth token nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, []byte(secret), mcpAuthSecretAAD(serverName, fieldName))
	payload := make([]byte, 0, len(nonce)+len(ciphertext))
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)
	return mcpAuthSecretEncodingV1 + base64.RawStdEncoding.EncodeToString(payload), nil
}

func (g *GlobalDB) decryptMCPAuthTokenSecrets(token *mcpauth.TokenRecord) error {
	if token == nil {
		return nil
	}
	accessToken, err := g.decryptMCPAuthTokenSecret(token.ServerName, "access_token", token.AccessToken)
	if err != nil {
		return err
	}
	refreshToken, err := g.decryptMCPAuthTokenSecret(token.ServerName, "refresh_token", token.RefreshToken)
	if err != nil {
		return err
	}
	token.AccessToken = accessToken
	token.RefreshToken = refreshToken
	return nil
}

func (g *GlobalDB) decryptMCPAuthTokenSecret(serverName string, fieldName string, encoded string) (string, error) {
	if strings.TrimSpace(encoded) == "" {
		return "", nil
	}
	if !strings.HasPrefix(encoded, mcpAuthSecretEncodingV1) {
		return "", fmt.Errorf(
			"store: MCP auth token %s for %q is not encrypted",
			fieldName,
			strings.TrimSpace(serverName),
		)
	}
	aead, err := g.mcpAuthSecretCipher()
	if err != nil {
		return "", err
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(encoded, mcpAuthSecretEncodingV1))
	if err != nil {
		return "", fmt.Errorf(
			"store: decode MCP auth token %s for %q: %w",
			fieldName,
			strings.TrimSpace(serverName),
			err,
		)
	}
	nonceSize := aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf(
			"store: MCP auth token %s for %q ciphertext is truncated",
			fieldName,
			strings.TrimSpace(serverName),
		)
	}
	plaintext, err := aead.Open(
		nil,
		payload[:nonceSize],
		payload[nonceSize:],
		mcpAuthSecretAAD(serverName, fieldName),
	)
	if err != nil {
		return "", fmt.Errorf(
			"store: decrypt MCP auth token %s for %q: %w",
			fieldName,
			strings.TrimSpace(serverName),
			err,
		)
	}
	return string(plaintext), nil
}

func (g *GlobalDB) mcpAuthSecretCipher() (cipher.AEAD, error) {
	key, err := g.mcpAuthSecretKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("store: create MCP auth token cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("store: create MCP auth token AEAD: %w", err)
	}
	return aead, nil
}

func (g *GlobalDB) mcpAuthSecretKey() ([]byte, error) {
	keyPath := g.mcpAuthSecretKeyPath()
	key, err := os.ReadFile(keyPath)
	if err == nil {
		if len(key) != mcpAuthSecretKeyBytes {
			return nil, fmt.Errorf(
				"store: MCP auth token key %q has %d bytes, want %d",
				keyPath,
				len(key),
				mcpAuthSecretKeyBytes,
			)
		}
		if chmodErr := os.Chmod(keyPath, mcpAuthSecretKeyFileMode); chmodErr != nil {
			return nil, fmt.Errorf("store: restrict MCP auth token key %q: %w", keyPath, chmodErr)
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("store: read MCP auth token key %q: %w", keyPath, err)
	}
	return createMCPAuthSecretKey(keyPath)
}

func (g *GlobalDB) mcpAuthSecretKeyPath() string {
	return strings.TrimSpace(g.path) + mcpAuthSecretKeyFileSuffix
}

func createMCPAuthSecretKey(path string) ([]byte, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("store: create MCP auth token key directory %q: %w", filepath.Dir(path), err)
	}
	key := make([]byte, mcpAuthSecretKeyBytes)
	if _, err := cryptoRand.Read(key); err != nil {
		return nil, fmt.Errorf("store: generate MCP auth token key: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mcpAuthSecretKeyFileMode)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			key, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil, fmt.Errorf("store: read MCP auth token key %q: %w", path, readErr)
			}
			if len(key) != mcpAuthSecretKeyBytes {
				return nil, fmt.Errorf(
					"store: MCP auth token key %q has %d bytes, want %d",
					path,
					len(key),
					mcpAuthSecretKeyBytes,
				)
			}
			return key, nil
		}
		return nil, fmt.Errorf("store: create MCP auth token key %q: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()
	written, err := file.Write(key)
	if err != nil {
		return nil, fmt.Errorf("store: write MCP auth token key %q: %w", path, err)
	}
	if written != len(key) {
		return nil, fmt.Errorf("store: write MCP auth token key %q: wrote %d bytes, want %d", path, written, len(key))
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("store: close MCP auth token key %q: %w", path, err)
	}
	return key, nil
}

func mcpAuthSecretAAD(serverName string, fieldName string) []byte {
	return []byte(strings.TrimSpace(serverName) + "\x00" + strings.TrimSpace(fieldName))
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
