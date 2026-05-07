package modelcatalog_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/modelcatalog"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
	_ "modernc.org/sqlite"
)

func TestCatalogServiceGlobalDBIntegration(t *testing.T) {
	t.Parallel()

	t.Run("Should refresh and list rows by provider with global DB store", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		store, _ := openCatalogGlobalDB(t)
		source := modelcatalog.NewConfigSource(map[string]aghconfig.ProviderConfig{
			"codex": {
				Models: aghconfig.ProviderModelsConfig{
					Default: "manual-model",
					Curated: []aghconfig.ProviderModelConfig{
						{ID: "gpt-5.4", DisplayName: "GPT-5.4"},
					},
				},
			},
			"claude": {
				Models: aghconfig.ProviderModelsConfig{
					Default: "claude-sonnet-4-6",
				},
			},
		})
		service, err := modelcatalog.NewService(store, []modelcatalog.Source{source})
		if err != nil {
			t.Fatalf("NewService() error = %v", err)
		}

		if _, err := service.Refresh(
			ctx,
			modelcatalog.RefreshOptions{Force: true, Now: integrationTime(0)},
		); err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}
		models, err := service.ListModels(ctx, modelcatalog.ListOptions{ProviderID: "codex", Now: integrationTime(1)})
		if err != nil {
			t.Fatalf("ListModels(codex) error = %v", err)
		}
		if got, want := len(models), 2; got != want {
			t.Fatalf("len(models) = %d, want %d: %#v", got, want, models)
		}
		for _, model := range models {
			if model.ProviderID != "codex" {
				t.Fatalf("model.ProviderID = %q, want codex", model.ProviderID)
			}
		}
	})

	t.Run("Should not persist raw models dev upstream payload", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		store, path := openCatalogGlobalDB(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := fmt.Fprint(w, `{
				"openai": {
					"models": {
						"gpt-5.4": {
							"name": "GPT-5.4",
							"unused_raw_marker": "raw_secret_marker_should_not_persist"
						}
					}
				}
			}`)
			if err != nil {
				t.Errorf("Fprint(response) error = %v", err)
			}
		}))
		t.Cleanup(server.Close)
		enabled := true
		source, err := modelcatalog.NewModelsDevSource(nil, aghconfig.ModelsDevSourceConfig{
			Enabled:  &enabled,
			Endpoint: server.URL,
			TTL:      "1h",
			Timeout:  "1s",
		})
		if err != nil {
			t.Fatalf("NewModelsDevSource() error = %v", err)
		}
		service, err := modelcatalog.NewService(store, []modelcatalog.Source{source})
		if err != nil {
			t.Fatalf("NewService() error = %v", err)
		}
		if _, err := service.Refresh(
			ctx,
			modelcatalog.RefreshOptions{ProviderID: "codex", Force: true, Now: integrationTime(0)},
		); err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}

		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatalf("sql.Open() error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := db.Close(); closeErr != nil {
				t.Errorf("db.Close() error = %v", closeErr)
			}
		})
		var matches int
		if err := db.QueryRowContext(
			ctx,
			`SELECT
				(SELECT COUNT(*) FROM model_catalog_rows
				 WHERE source_id LIKE ? OR provider_id LIKE ? OR model_id LIKE ? OR display_name LIKE ? OR last_error LIKE ?)
				+
				(SELECT COUNT(*) FROM model_catalog_sources
				 WHERE source_id LIKE ? OR provider_id LIKE ? OR last_error LIKE ?)`,
			"%raw_secret_marker%",
			"%raw_secret_marker%",
			"%raw_secret_marker%",
			"%raw_secret_marker%",
			"%raw_secret_marker%",
			"%raw_secret_marker%",
			"%raw_secret_marker%",
			"%raw_secret_marker%",
		).Scan(&matches); err != nil {
			t.Fatalf("QueryRowContext(raw marker) error = %v", err)
		}
		if matches != 0 {
			t.Fatalf("raw marker persisted in %d catalog fields, want 0", matches)
		}
	})
}

func openCatalogGlobalDB(t *testing.T) (*globaldb.GlobalDB, string) {
	t.Helper()

	ctx := testutil.Context(t)
	path := filepath.Join(t.TempDir(), store.GlobalDatabaseName)
	store, err := globaldb.OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(testutil.Context(t)); closeErr != nil {
			t.Errorf("GlobalDB.Close() error = %v", closeErr)
		}
	})
	return store, path
}

func integrationTime(offset int) time.Time {
	return time.Date(2026, 5, 7, 13, offset, 0, 0, time.UTC)
}
