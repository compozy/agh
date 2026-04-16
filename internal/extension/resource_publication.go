package extensionpkg

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

// ResolveManifestToolResources converts manifest tool declarations into tool specs.
func ResolveManifestToolResources(manifest *Manifest) []toolspkg.Tool {
	if manifest == nil || len(manifest.Resources.Tools) == 0 {
		return nil
	}

	names := make([]string, 0, len(manifest.Resources.Tools))
	for name := range manifest.Resources.Tools {
		names = append(names, name)
	}
	slices.Sort(names)

	tools := make([]toolspkg.Tool, 0, len(names))
	for _, name := range names {
		cfg := manifest.Resources.Tools[name]
		tools = append(tools, toolspkg.Tool{
			Name:        strings.TrimSpace(name),
			Description: strings.TrimSpace(cfg.Description),
			InputSchema: cloneRawMessage(cfg.InputSchema),
			ReadOnly:    cfg.ReadOnly,
			Source:      toolspkg.ToolSourceExtension,
		})
	}
	return tools
}

// ResolveManifestMCPServerResources converts manifest MCP declarations into MCP server specs.
func ResolveManifestMCPServerResources(
	rootDir string,
	manifest *Manifest,
	getenv func(string) string,
) ([]aghconfig.MCPServer, error) {
	if manifest == nil || len(manifest.Resources.MCPServers) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(manifest.Resources.MCPServers))
	for name := range manifest.Resources.MCPServers {
		names = append(names, name)
	}
	slices.Sort(names)

	servers := make([]aghconfig.MCPServer, 0, len(names))
	for _, name := range names {
		decl := manifest.Resources.MCPServers[name]
		command, err := resolveManifestCommand(rootDir, decl.Command, getenv)
		if err != nil {
			return nil, err
		}
		args, err := resolveManifestStringSlice(rootDir, decl.Args, getenv)
		if err != nil {
			return nil, err
		}
		env, err := resolveManifestStringMap(rootDir, decl.Env, getenv)
		if err != nil {
			return nil, err
		}
		server := aghconfig.MCPServer{
			Name:    strings.TrimSpace(name),
			Command: command,
			Args:    args,
			Env:     env,
		}
		if err := server.Validate("extension.resources.mcp_servers[" + name + "]"); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func resolveManifestCommand(rootDir string, value string, getenv func(string) string) (string, error) {
	resolved, err := resolveManifestString(rootDir, value, getenv)
	if err != nil {
		return "", err
	}
	if resolved == "" {
		return "", nil
	}
	if filepath.IsAbs(resolved) {
		return filepath.Clean(resolved), nil
	}
	if strings.ContainsRune(resolved, filepath.Separator) || strings.HasPrefix(resolved, ".") {
		return resolvePathWithinRoot(rootDir, resolved)
	}
	return resolved, nil
}

func resolveManifestStringSlice(rootDir string, values []string, getenv func(string) string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	resolved := make([]string, 0, len(values))
	for _, value := range values {
		item, err := resolveManifestString(rootDir, value, getenv)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, item)
	}
	return resolved, nil
}

func resolveManifestStringMap(
	rootDir string,
	env map[string]string,
	getenv func(string) string,
) (map[string]string, error) {
	if len(env) == 0 {
		return nil, nil
	}

	resolved := make(map[string]string, len(env))
	for key, value := range env {
		item, err := resolveManifestString(rootDir, value, getenv)
		if err != nil {
			return nil, err
		}
		resolved[key] = item
	}
	return resolved, nil
}

func resolveManifestString(rootDir string, value string, getenv func(string) string) (string, error) {
	resolved := strings.TrimSpace(value)
	if resolved == "" {
		return "", nil
	}

	resolved = strings.ReplaceAll(resolved, "{{config_dir}}", rootDir)
	for {
		start := strings.Index(resolved, "{{env:")
		if start < 0 {
			break
		}
		end := strings.Index(resolved[start:], "}}")
		if end < 0 {
			return "", fmt.Errorf("invalid env template %q", value)
		}
		end += start
		key := strings.TrimSpace(strings.TrimPrefix(resolved[start:end], "{{env:"))
		resolved = resolved[:start] + getenvValue(getenv, key) + resolved[end+2:]
	}
	return resolved, nil
}

func getenvValue(getenv func(string) string, key string) string {
	if getenv == nil {
		return ""
	}
	return getenv(key)
}
