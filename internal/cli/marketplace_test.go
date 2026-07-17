package cli

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/api/contract"
)

func TestMarketplaceCommands(t *testing.T) {
	t.Parallel()

	t.Run("Should render grouped search as the shared JSON contract", func(t *testing.T) {
		t.Parallel()

		want := MarketplaceSearchRecord{Kinds: []contract.MarketplaceKindResult{{
			Kind: "skill",
			Items: []contract.MarketplaceListingPayload{{
				Kind: "skill", EntryID: "skill-entry", Name: "Review", Version: "1.2.0",
				Installed: true, Source: "curated",
			}},
		}}}
		deps := newTestDeps(t, &stubClient{
			searchMarketplaceFn: func(
				_ context.Context,
				query string,
				limit int,
				scope MarketplaceReadScope,
			) (MarketplaceSearchRecord, error) {
				if query != "review" {
					t.Fatalf("query = %q, want review", query)
				}
				if limit != 7 {
					t.Fatalf("limit = %d, want 7", limit)
				}
				if scope.Scope != contract.SettingsWorkspaceScopeWorkspace || scope.WorkspaceID != "ws-alpha" {
					t.Fatalf("read scope = %#v, want workspace ws-alpha", scope)
				}
				return want, nil
			},
		})

		stdout, _, err := executeRootCommand(
			t,
			deps,
			"marketplace",
			"search",
			"review",
			"--limit",
			"7",
			"--scope",
			"workspace",
			"--workspace",
			"ws-alpha",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("marketplace search command error = %v", err)
		}
		var got MarketplaceSearchRecord
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(marketplace search) error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("marketplace search = %#v, want %#v", got, want)
		}
	})

	t.Run("Should render one kind without changing the daemon payload", func(t *testing.T) {
		t.Parallel()

		want := MarketplaceKindRecord{
			Kind: "extension",
			Items: []MarketplaceListingRecord{{
				Kind: "extension", EntryID: "extension-entry", Name: "Bridge", Source: "curated",
			}},
		}
		deps := newTestDeps(t, &stubClient{
			browseMarketplaceFn: func(
				_ context.Context,
				kind string,
				query string,
				limit int,
				scope MarketplaceReadScope,
			) (MarketplaceKindRecord, error) {
				if kind != "extension" {
					t.Fatalf("kind = %q, want extension", kind)
				}
				if query != "" {
					t.Fatalf("query = %q, want empty", query)
				}
				if limit != marketplaceDefaultLimit {
					t.Fatalf("limit = %d, want %d", limit, marketplaceDefaultLimit)
				}
				if scope.Scope != contract.SettingsWorkspaceScopeGlobal || scope.WorkspaceID != "" {
					t.Fatalf("read scope = %#v, want global", scope)
				}
				return want, nil
			},
		})

		stdout, _, err := executeRootCommand(
			t, deps, "marketplace", "search", "--kind", "extension", "-o", "json",
		)
		if err != nil {
			t.Fatalf("marketplace kind search command error = %v", err)
		}
		var got MarketplaceKindRecord
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(marketplace kind search) error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("marketplace kind search = %#v, want %#v", got, want)
		}
	})

	t.Run("Should resolve detail by kind and stable entry id", func(t *testing.T) {
		t.Parallel()

		want := MarketplaceEntryRecord{Entry: MarketplaceListingRecord{
			Kind: "mcp", EntryID: "github-mcp", Name: "GitHub", Source: "curated",
		}}
		deps := newTestDeps(t, &stubClient{
			marketplaceInfoFn: func(
				_ context.Context,
				kind string,
				entryID string,
				scope MarketplaceReadScope,
			) (MarketplaceEntryRecord, error) {
				if kind != "mcp" {
					t.Fatalf("kind = %q, want mcp", kind)
				}
				if entryID != "github-mcp" {
					t.Fatalf("entryID = %q, want github-mcp", entryID)
				}
				if scope.Scope != contract.SettingsWorkspaceScopeWorkspace || scope.WorkspaceID != "ws-alpha" {
					t.Fatalf("read scope = %#v, want workspace ws-alpha", scope)
				}
				return want, nil
			},
		})

		stdout, _, err := executeRootCommand(
			t,
			deps,
			"marketplace",
			"info",
			"mcp",
			"github-mcp",
			"--scope",
			"workspace",
			"--workspace",
			"ws-alpha",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("marketplace info command error = %v", err)
		}
		var got MarketplaceEntryRecord
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(marketplace info) error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("marketplace info = %#v, want %#v", got, want)
		}
	})

	t.Run("Should refresh the selected feed-backed kind", func(t *testing.T) {
		t.Parallel()

		want := MarketplaceRefreshRecord{Kinds: []contract.MarketplaceRefreshKindPayload{{
			Kind: "skill", Outcome: "updated", EntryCount: 3,
		}}}
		deps := newTestDeps(t, &stubClient{
			refreshMarketplaceFn: func(_ context.Context, kind string) (MarketplaceRefreshRecord, error) {
				if kind != "skill" {
					t.Fatalf("kind = %q, want skill", kind)
				}
				return want, nil
			},
		})

		stdout, _, err := executeRootCommand(
			t, deps, "marketplace", "refresh", "--kind", "skill", "-o", "json",
		)
		if err != nil {
			t.Fatalf("marketplace refresh command error = %v", err)
		}
		var got MarketplaceRefreshRecord
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(marketplace refresh) error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("marketplace refresh = %#v, want %#v", got, want)
		}
	})

	t.Run("Should reject incomplete installed-state scope flags before transport", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{})
		cases := []struct {
			name    string
			args    []string
			wantErr string
		}{
			{
				name:    "Should require a workspace ID for workspace scope",
				args:    []string{"marketplace", "search", "--scope", "workspace"},
				wantErr: "--scope workspace requires --workspace",
			},
			{
				name: "Should reject a workspace ID for global scope",
				args: []string{
					"marketplace", "info", "mcp", "github-mcp", "--scope", "global", "--workspace", "ws-alpha",
				},
				wantErr: "--workspace requires --scope workspace",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				_, _, err := executeRootCommand(t, deps, tc.args...)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("executeRootCommand() error = %v, want containing %q", err, tc.wantErr)
				}
			})
		}
	})
}
