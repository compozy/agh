package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	// MCPJSONName is the supported JSON sidecar filename for MCP server declarations.
	MCPJSONName = "mcp.json"
)

type mcpJSONDocument struct {
	MCPServersCamel map[string]mcpJSONServer `json:"mcpServers"`
	MCPServersSnake map[string]mcpJSONServer `json:"mcp_servers"`
}

type mcpJSONServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// ParseMCPServersJSON parses an MCP JSON document into canonical MCP server values.
// The document may use either `mcpServers` or `mcp_servers` as the top-level key.
func ParseMCPServersJSON(content []byte, source string) ([]MCPServer, error) {
	sourceName := strings.TrimSpace(source)
	if sourceName == "" {
		sourceName = MCPJSONName
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()

	var document mcpJSONDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("config: decode MCP JSON %q: %w", sourceName, err)
	}
	if err := ensureJSONEOF(decoder, sourceName); err != nil {
		return nil, err
	}

	servers := sortedMCPJSONServers(document.MCPServersCamel)
	servers = OverrideMCPServers(servers, sortedMCPJSONServers(document.MCPServersSnake))
	for idx, server := range servers {
		if err := server.Validate(fmt.Sprintf("mcp.json %q[%d]", sourceName, idx)); err != nil {
			return nil, fmt.Errorf("config: validate MCP JSON %q: %w", sourceName, err)
		}
	}

	return servers, nil
}

// LoadMCPServersJSONFile parses an optional `mcp.json` file from disk.
// Missing files are treated as absent rather than as errors.
func LoadMCPServersJSONFile(path string) ([]MCPServer, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, nil
	}

	content, err := os.ReadFile(trimmed)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("config: read MCP JSON %q: %w", trimmed, err)
	}

	return ParseMCPServersJSON(content, trimmed)
}

func ensureJSONEOF(decoder *json.Decoder, source string) error {
	if decoder == nil {
		return errors.New("config: JSON decoder is required")
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("config: decode MCP JSON %q: %w", source, err)
	}

	return fmt.Errorf("config: decode MCP JSON %q: unexpected trailing JSON value", source)
}

func sortedMCPJSONServers(values map[string]mcpJSONServer) []MCPServer {
	if len(values) == 0 {
		return nil
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	servers := make([]MCPServer, 0, len(names))
	for _, name := range names {
		entry := values[name]
		servers = append(servers, MCPServer{
			Name:    strings.TrimSpace(name),
			Command: strings.TrimSpace(entry.Command),
			Args:    append([]string(nil), entry.Args...),
			Env:     mergeStringMaps(nil, entry.Env),
		})
	}

	return servers
}
