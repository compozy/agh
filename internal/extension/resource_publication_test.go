package extensionpkg

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/testutil"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

func TestResolveManifestToolResourcesMatchesDynamicSnapshotCanonicalShape(t *testing.T) {
	t.Parallel()

	manifest := &Manifest{
		Resources: ResourcesConfig{
			Tools: map[string]ToolConfig{
				" lookup ": {
					Description: " search workspace ",
					InputSchema: json.RawMessage(`{
						"properties": {"path": {"type": "string"}},
						"type": "object"
					}`),
					ReadOnly: true,
				},
			},
		},
	}

	tools := ResolveManifestToolResources(manifest)
	if got, want := len(tools), 1; got != want {
		t.Fatalf("len(ResolveManifestToolResources()) = %d, want %d", got, want)
	}

	codec, err := toolspkg.NewResourceCodec()
	if err != nil {
		t.Fatalf("toolspkg.NewResourceCodec() error = %v", err)
	}
	scope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}

	manifestCanonical := mustCanonicalToolJSON(t, codec, scope, tools[0])
	dynamicSpec, err := codec.DecodeAndValidate(testutil.Context(t), scope, []byte(`{
		"name": "lookup",
		"description": "search workspace",
		"input_schema": {
			"type": "object",
			"properties": {"path": {"type": "string"}}
		},
		"read_only": true,
		"source": "extension"
	}`))
	if err != nil {
		t.Fatalf("codec.DecodeAndValidate(dynamic) error = %v", err)
	}
	dynamicCanonical := mustCanonicalToolJSON(t, codec, scope, dynamicSpec)

	if !bytes.Equal(manifestCanonical, dynamicCanonical) {
		t.Fatalf(
			"manifest canonical tool != dynamic canonical tool\nmanifest=%s\ndynamic=%s",
			string(manifestCanonical),
			string(dynamicCanonical),
		)
	}
}

func TestResolveManifestMCPServerResourcesResolvesTemplates(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	manifest := &Manifest{
		Resources: ResourcesConfig{
			MCPServers: map[string]MCPServerConfig{
				"git": {
					Command: "./bin/mcp-git",
					Args:    []string{"--config", "{{config_dir}}/git.toml"},
					Env: map[string]string{
						"TOKEN": "{{env:GIT_TOKEN}}",
					},
				},
			},
		},
	}

	servers, err := ResolveManifestMCPServerResources(rootDir, manifest, func(key string) string {
		if key == "GIT_TOKEN" {
			return "secret-token"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("ResolveManifestMCPServerResources() error = %v", err)
	}
	if got, want := len(servers), 1; got != want {
		t.Fatalf("len(ResolveManifestMCPServerResources()) = %d, want %d", got, want)
	}
	if got, want := servers[0].Command, filepath.Join(rootDir, "bin", "mcp-git"); got != want {
		t.Fatalf("servers[0].Command = %q, want %q", got, want)
	}
	if got, want := servers[0].Args, []string{
		"--config",
		filepath.Join(rootDir, "git.toml"),
	}; !equalStrings(
		got,
		want,
	) {
		t.Fatalf("servers[0].Args = %#v, want %#v", got, want)
	}
	if got, want := servers[0].Env["TOKEN"], "secret-token"; got != want {
		t.Fatalf("servers[0].Env[TOKEN] = %q, want %q", got, want)
	}
}

func mustCanonicalToolJSON(
	t *testing.T,
	codec resources.KindCodec[toolspkg.Tool],
	scope resources.ResourceScope,
	spec toolspkg.Tool,
) []byte {
	t.Helper()

	encoded, err := codec.Encode(spec)
	if err != nil {
		t.Fatalf("codec.Encode() error = %v", err)
	}
	validated, err := codec.DecodeAndValidate(testutil.Context(t), scope, encoded)
	if err != nil {
		t.Fatalf("codec.DecodeAndValidate() error = %v", err)
	}
	canonical, err := codec.Encode(validated)
	if err != nil {
		t.Fatalf("codec.Encode(validated) error = %v", err)
	}
	return canonical
}

func equalStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for idx := range got {
		if got[idx] != want[idx] {
			return false
		}
	}
	return true
}
