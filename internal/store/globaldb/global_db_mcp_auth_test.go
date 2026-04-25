package globaldb

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
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
