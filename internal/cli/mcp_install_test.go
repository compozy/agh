package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/api/contract"
)

func TestMCPInstallCommandMapsFlagsAndPreservesStructuredResponse(t *testing.T) {
	t.Run("Should map flags and preserve the structured response", func(t *testing.T) {
		t.Parallel()

		want := InstallSettingsMCPServerRecord{
			MCPServer: contract.SettingsMCPServerItemPayload{
				Name:      "github-workspace",
				Transport: "stdio",
				Command:   "npx",
				SecretEnv: map[string]string{
					"GITHUB_TOKEN": "vault:mcp/ws/ws-1/github-workspace/env/GITHUB_TOKEN",
				},
				Scope:          contract.SettingsScopeWorkspace,
				WorkspaceID:    "ws-1",
				CatalogEntry:   "github",
				CatalogVersion: "1.2.3",
			},
			Apply: mcpInstallApplyFixture(
				contract.SettingsScopeWorkspace,
				contract.SettingsWriteTargetWorkspaceMCPSidecar,
				"ws-1",
			),
			NextStep: "none",
		}
		client := &stubClient{
			installSettingsMCPServerFn: func(
				_ context.Context,
				request InstallSettingsMCPServerRequest,
			) (InstallSettingsMCPServerRecord, error) {
				if got, want := request.EntryID, "github"; got != want {
					t.Fatalf("EntryID = %q, want %q", got, want)
				}
				if got, want := request.Name, "github-workspace"; got != want {
					t.Fatalf("Name = %q, want %q", got, want)
				}
				if got, want := request.Scope, contract.SettingsWorkspaceScopeWorkspace; got != want {
					t.Fatalf("Scope = %q, want %q", got, want)
				}
				if got, want := request.WorkspaceID, "ws-1"; got != want {
					t.Fatalf("WorkspaceID = %q, want %q", got, want)
				}
				if request.Values == nil {
					t.Fatal("request.Values = nil")
				}
				if got, want := request.Values.Env["GITHUB_TOKEN"].Value, "write-only-secret"; got != want {
					t.Fatalf("GITHUB_TOKEN value = %q, want mapped secret", got)
				}
				if got, want := request.Values.Env["CONFIG"].VaultRef, "vault:mcp/shared/config"; got != want {
					t.Fatalf("CONFIG VaultRef = %q, want %q", got, want)
				}
				if request.Values.OAuthClientSecret == nil {
					t.Fatal("OAuthClientSecret = nil")
				}
				if got, want := request.Values.OAuthClientSecret.Value, "oauth-write-only-secret"; got != want {
					t.Fatalf("OAuthClientSecret.Value = %q, want mapped secret", got)
				}
				return want, nil
			},
		}
		deps := newTestDeps(t, client)
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"mcp",
			"install",
			"github",
			"--name",
			"github-workspace",
			"--scope",
			"workspace",
			"--workspace",
			"ws-1",
			"--set",
			"GITHUB_TOKEN=write-only-secret",
			"--vault-ref",
			"CONFIG=vault:mcp/shared/config",
			"--oauth-client-secret-value",
			"oauth-write-only-secret",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("mcp install command error = %v", err)
		}
		if strings.Contains(stdout, "write-only-secret") {
			t.Fatalf("mcp install output leaked write-only secret: %s", stdout)
		}
		var got InstallSettingsMCPServerRecord
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(mcp install) error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("mcp install response = %#v, want %#v", got, want)
		}
	})

	t.Run("Should render the config apply truth in human output", func(t *testing.T) {
		t.Parallel()

		client := &stubClient{installSettingsMCPServerFn: func(
			context.Context,
			InstallSettingsMCPServerRequest,
		) (InstallSettingsMCPServerRecord, error) {
			return InstallSettingsMCPServerRecord{
				MCPServer: contract.SettingsMCPServerItemPayload{
					Name: "github", Transport: "stdio", Scope: contract.SettingsScopeGlobal,
					CatalogEntry: "github", CatalogVersion: "1.2.3",
				},
				Apply: contract.SettingsApplyResponse{
					Applied:          false,
					Lifecycle:        contract.SettingsApplyLifecycleLiveAdd,
					ApplyRecordID:    "cfgapp-failed",
					ActiveGeneration: 2,
					NextAction:       contract.SettingsApplyNextActionRetry,
				},
				NextStep: contract.SettingsMCPInstallNextStepNone,
			}, nil
		}}
		stdout, _, err := executeRootCommand(t, newTestDeps(t, client), "mcp", "install", "github")
		if err != nil {
			t.Fatalf("mcp install command error = %v", err)
		}
		for _, detail := range []string{"Applied", "false", "live-add", "cfgapp-failed", "retry"} {
			if !strings.Contains(stdout, detail) {
				t.Fatalf("human output missing %q:\n%s", detail, stdout)
			}
		}
	})

	t.Run("Should map an OAuth client secret Vault ref", func(t *testing.T) {
		t.Parallel()

		client := &stubClient{
			installSettingsMCPServerFn: func(
				_ context.Context,
				request InstallSettingsMCPServerRequest,
			) (InstallSettingsMCPServerRecord, error) {
				if request.Values == nil || request.Values.OAuthClientSecret == nil {
					t.Fatalf("request.Values = %#v, want OAuth client secret", request.Values)
				}
				if got, want := request.Values.OAuthClientSecret.VaultRef, "vault:mcp/shared/oauth"; got != want {
					t.Fatalf("OAuthClientSecret.VaultRef = %q, want %q", got, want)
				}
				return InstallSettingsMCPServerRecord{
					MCPServer: contract.SettingsMCPServerItemPayload{
						Name: "github", Scope: contract.SettingsScopeGlobal,
					},
					NextStep: contract.SettingsMCPInstallNextStepAuthorize,
				}, nil
			},
		}
		stdout, _, err := executeRootCommand(
			t,
			newTestDeps(t, client),
			"mcp",
			"install",
			"github",
			"--oauth-client-secret-vault-ref",
			"vault:mcp/shared/oauth",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("mcp install command error = %v", err)
		}
		if strings.Contains(stdout, "vault:mcp/shared/oauth") {
			t.Fatalf("mcp install output leaked OAuth client secret ref: %s", stdout)
		}
	})
}

func TestMCPInstallCommandRejectsAmbiguousInputsBeforeCallingDaemon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		detail string
	}{
		{
			name: "Should reject duplicate typed and Vault modes",
			args: []string{
				"mcp", "install", "github",
				"--set", "TOKEN=secret",
				"--vault-ref", "TOKEN=vault:mcp/shared/token",
			},
			detail: "assigned more than once",
		},
		{
			name: "Should reject duplicate OAuth client secret modes",
			args: []string{
				"mcp", "install", "github",
				"--oauth-client-secret-value", "secret",
				"--oauth-client-secret-vault-ref", "vault:mcp/shared/oauth",
			},
			detail: "OAuth client secret is assigned more than once",
		},
		{
			name:   "Should require a workspace ID for workspace scope",
			args:   []string{"mcp", "install", "github", "--scope", "workspace"},
			detail: "requires --workspace",
		},
		{
			name:   "Should reject workspace ID in global scope",
			args:   []string{"mcp", "install", "github", "--workspace", "ws-1"},
			detail: "requires --scope workspace",
		},
		{
			name:   "Should reject malformed assignments",
			args:   []string{"mcp", "install", "github", "--set", "TOKEN"},
			detail: "requires KEY=VALUE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := newTestDeps(t, &stubClient{})
			_, _, err := executeRootCommand(t, deps, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.detail) {
				t.Fatalf("executeRootCommand() error = %v, want detail %q", err, tc.detail)
			}
		})
	}
}

func TestUnixSocketClientInstallSettingsMCPServerUsesCanonicalEndpoint(t *testing.T) {
	t.Run("Should use the canonical MCP install endpoint", func(t *testing.T) {
		t.Parallel()

		want := InstallSettingsMCPServerRecord{
			MCPServer: contract.SettingsMCPServerItemPayload{
				Name: "github", Transport: "stdio", Scope: contract.SettingsScopeGlobal,
				CatalogEntry: "github", CatalogVersion: "1.2.3",
			},
			Apply: mcpInstallApplyFixture(
				contract.SettingsScopeGlobal,
				contract.SettingsWriteTargetGlobalMCPSidecar,
				"",
			),
			NextStep: "none",
		}
		request := InstallSettingsMCPServerRequest{
			EntryID: "github",
			Scope:   contract.SettingsWorkspaceScopeGlobal,
			Values:  &contract.SettingsMCPCatalogInstallValuesPayload{},
		}
		transport := roundTripperFunc(func(httpRequest *http.Request) (*http.Response, error) {
			if got, want := httpRequest.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			if got, want := httpRequest.URL.Path, "/api/settings/mcp-servers/install"; got != want {
				t.Fatalf("path = %q, want %q", got, want)
			}
			var gotRequest InstallSettingsMCPServerRequest
			if err := json.NewDecoder(httpRequest.Body).Decode(&gotRequest); err != nil {
				t.Fatalf("Decode(request) error = %v", err)
			}
			if !reflect.DeepEqual(gotRequest, request) {
				t.Fatalf("request = %#v, want %#v", gotRequest, request)
			}
			payload, err := json.Marshal(want)
			if err != nil {
				t.Fatalf("json.Marshal(response) error = %v", err)
			}
			return newHTTPResponse(http.StatusOK, string(payload)), nil
		})
		client := &unixSocketClient{
			socketPath: "/tmp/agh.sock",
			httpClient: &http.Client{Transport: transport},
		}
		client.streamClient = client.httpClient

		got, err := client.InstallSettingsMCPServer(t.Context(), request)
		if err != nil {
			t.Fatalf("InstallSettingsMCPServer() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("InstallSettingsMCPServer() = %#v, want %#v", got, want)
		}
	})
}

func mcpInstallApplyFixture(
	scope contract.SettingsScopeKind,
	writeTarget contract.SettingsWriteTargetKind,
	workspaceID string,
) contract.SettingsApplyResponse {
	return contract.SettingsApplyResponse{
		Section:          contract.SettingsApplyTargetMCPServers,
		Scope:            scope,
		WriteTarget:      writeTarget,
		WorkspaceID:      workspaceID,
		Applied:          true,
		Lifecycle:        contract.SettingsApplyLifecycleLiveAdd,
		ApplyRecordID:    "cfgapp-mcp-install",
		ActiveGeneration: 3,
		ActiveConfigHash: "sha256:mcp-install",
		NextAction:       contract.SettingsApplyNextActionNone,
	}
}

func TestMCPInstallClientPropagatesTransportErrors(t *testing.T) {
	t.Run("Should preserve the transport error detail", func(t *testing.T) {
		t.Parallel()

		client := &unixSocketClient{
			socketPath: "/tmp/agh.sock",
			httpClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("transport unavailable")
			})},
		}
		client.streamClient = client.httpClient
		_, err := client.InstallSettingsMCPServer(t.Context(), InstallSettingsMCPServerRequest{})
		if err == nil || !strings.Contains(err.Error(), "transport unavailable") {
			t.Fatalf("InstallSettingsMCPServer() error = %v, want transport unavailable", err)
		}
	})
}
