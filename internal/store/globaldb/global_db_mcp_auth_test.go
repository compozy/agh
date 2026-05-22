package globaldb

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpauth "github.com/compozy/agh/internal/mcp/auth"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestMCPAuthTokenStorePersistsAcrossReopenWithPrivatePermissions(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	path := filepath.Join(t.TempDir(), store.GlobalDatabaseName)
	db, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm() & 0o077; got != 0 {
		t.Fatalf("global db permissions = %#o, want no group/other bits", info.Mode().Perm())
	}

	expiresAt := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	if err := db.SaveMCPAuthToken(ctx, mcpauth.TokenRecord{
		ServerName:   "linear",
		Issuer:       "https://issuer.example",
		ClientID:     "client",
		Scopes:       []string{"read", "write"},
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		ObtainedAt:   expiresAt.Add(-time.Hour),
		UpdatedAt:    expiresAt.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("SaveMCPAuthToken() error = %v", err)
	}
	var rawAccessToken, rawRefreshToken string
	if err := db.db.QueryRowContext(
		ctx,
		`SELECT access_token_ref, refresh_token_ref FROM mcp_auth_tokens WHERE server_name = ?`,
		"linear",
	).Scan(&rawAccessToken, &rawRefreshToken); err != nil {
		t.Fatalf("query raw token row error = %v", err)
	}
	for label, raw := range map[string]string{"access": rawAccessToken, "refresh": rawRefreshToken} {
		plaintext := label + "-token"
		if raw == plaintext {
			t.Fatalf("raw %s token = %q, want vault ref without plaintext token material", label, raw)
		}
		if !strings.HasPrefix(raw, "vault:mcp/") {
			t.Fatalf("raw %s token = %q, want vault:mcp ref", label, raw)
		}
	}
	if got, want := rawAccessToken, "vault:mcp/linear/oauth/access-token"; got != want {
		t.Fatalf("raw access token ref = %q, want %q", got, want)
	}
	if got, want := rawRefreshToken, "vault:mcp/linear/oauth/refresh-token"; got != want {
		t.Fatalf("raw refresh token ref = %q, want %q", got, want)
	}
	var encryptedValue string
	if err := db.db.QueryRowContext(
		ctx,
		`SELECT encrypted_value FROM vault_secrets WHERE ref = ?`,
		rawAccessToken,
	).Scan(&encryptedValue); err != nil {
		t.Fatalf("query vault access token error = %v", err)
	}
	if strings.Contains(encryptedValue, "access-token") {
		t.Fatalf("vault encrypted access token = %q, want no plaintext token material", encryptedValue)
	}
	if _, err := os.Stat(path + ".mcp-auth.key"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(legacy MCP auth key) error = %v, want not exists", err)
	}
	if err := db.Close(ctx); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	reopened, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB(reopen) error = %v", err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(ctx); err != nil {
			t.Fatalf("Close(reopened) error = %v", err)
		}
	})

	token, err := reopened.GetMCPAuthToken(ctx, "linear")
	if err != nil {
		t.Fatalf("GetMCPAuthToken() error = %v", err)
	}
	if token.AccessToken != "access-token" || token.RefreshToken != "refresh-token" {
		t.Fatalf("token = %#v, want persisted token material", token)
	}
	if len(token.Scopes) != 2 || token.Scopes[0] != "read" || token.Scopes[1] != "write" {
		t.Fatalf("token scopes = %#v", token.Scopes)
	}
	if !token.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("token expires_at = %s, want %s", token.ExpiresAt, expiresAt)
	}

	if err := reopened.DeleteMCPAuthToken(ctx, "linear"); err != nil {
		t.Fatalf("DeleteMCPAuthToken() error = %v", err)
	}
	if _, err := reopened.GetMCPAuthToken(ctx, "linear"); !errors.Is(err, mcpauth.ErrTokenNotFound) {
		t.Fatalf("GetMCPAuthToken(deleted) error = %v, want ErrTokenNotFound", err)
	}
}

func TestMCPAuthTokenStoreRejectsIncompleteToken(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db, err := OpenGlobalDB(ctx, filepath.Join(t.TempDir(), store.GlobalDatabaseName))
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(ctx); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	err = db.SaveMCPAuthToken(ctx, mcpauth.TokenRecord{
		ServerName: "linear",
		ClientID:   "client",
	})
	if err == nil {
		t.Fatal("SaveMCPAuthToken() error = nil, want missing access token failure")
	}
}
