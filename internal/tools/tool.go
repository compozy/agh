// Package tools defines the minimal tool registration types shared by the
// extension architecture.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolSource identifies where a tool definition originated.
type ToolSource int

const (
	// ToolSourceBuiltin marks daemon-defined tools.
	ToolSourceBuiltin ToolSource = iota
	// ToolSourceMCP marks tools discovered from MCP servers.
	ToolSourceMCP
	// ToolSourceExtension marks tools provided by extensions.
	ToolSourceExtension
	// ToolSourceDynamic marks tools assembled at runtime.
	ToolSourceDynamic
)

var toolSourceNames = map[ToolSource]string{
	ToolSourceBuiltin:   "builtin",
	ToolSourceMCP:       "mcp",
	ToolSourceExtension: "extension",
	ToolSourceDynamic:   "dynamic",
}

// String returns the stable text form for the source.
func (s ToolSource) String() string {
	name, ok := toolSourceNames[s]
	if !ok {
		return ""
	}
	return name
}

// MarshalText encodes the source as a string.
func (s ToolSource) MarshalText() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return []byte(s.String()), nil
}

// UnmarshalText decodes the source from a string value.
func (s *ToolSource) UnmarshalText(text []byte) error {
	value := strings.TrimSpace(string(text))
	for source, name := range toolSourceNames {
		if value == name {
			*s = source
			return nil
		}
	}
	return fmt.Errorf("tools: invalid tool source %q", value)
}

// Validate ensures the source is one of the documented values.
func (s ToolSource) Validate() error {
	if _, ok := toolSourceNames[s]; !ok {
		return fmt.Errorf("tools: invalid tool source %d", s)
	}
	return nil
}

// Tool describes one registered tool.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	ReadOnly    bool            `json:"read_only"`
	Source      ToolSource      `json:"source"`
}

// UnmarshalJSON accepts the canonical tool shape and hook-compatible tool name
// payloads without depending on the hooks package.
func (t *Tool) UnmarshalJSON(data []byte) error {
	type rawTool struct {
		Name        string          `json:"name"`
		ToolName    string          `json:"tool_name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"input_schema"`
		ReadOnly    *bool           `json:"read_only"`
		Source      ToolSource      `json:"source"`
	}

	var raw rawTool
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Name != "" && raw.ToolName != "" && raw.Name != raw.ToolName {
		return fmt.Errorf("tools: conflicting tool names %q and %q", raw.Name, raw.ToolName)
	}

	name := raw.Name
	if name == "" {
		name = raw.ToolName
	}

	t.Name = name
	t.Description = raw.Description
	t.InputSchema = raw.InputSchema
	t.ReadOnly = raw.ReadOnly != nil && *raw.ReadOnly
	t.Source = raw.Source

	return nil
}

// ToolProvider lists tools available from one source.
type ToolProvider interface {
	Tools(ctx context.Context) ([]Tool, error)
}
