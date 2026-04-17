package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type staticProvider struct {
	tools []Tool
	err   error
}

var _ ToolProvider = staticProvider{}

func (p staticProvider) Tools(_ context.Context) ([]Tool, error) {
	return p.tools, p.err
}

func assertToolEqual(t *testing.T, got, want Tool) {
	t.Helper()

	if got.Name != want.Name {
		t.Fatalf("Tool.Name = %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Fatalf("Tool.Description = %q, want %q", got.Description, want.Description)
	}
	if string(got.InputSchema) != string(want.InputSchema) {
		t.Fatalf("Tool.InputSchema = %s, want %s", string(got.InputSchema), string(want.InputSchema))
	}
	if got.ReadOnly != want.ReadOnly {
		t.Fatalf("Tool.ReadOnly = %t, want %t", got.ReadOnly, want.ReadOnly)
	}
	if got.Source != want.Source {
		t.Fatalf("Tool.Source = %v, want %v", got.Source, want.Source)
	}
}

func TestToolMarshalJSONCanonical(t *testing.T) {
	t.Parallel()

	tool := Tool{
		Name:        "pgvector_search",
		Description: "Semantic search over stored memories",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		ReadOnly:    true,
		Source:      ToolSourceExtension,
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("json.Marshal(Tool) error = %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(canonical Tool JSON) error = %v", err)
	}

	if got := string(decoded["name"]); got != `"pgvector_search"` {
		t.Fatalf("encoded name = %s, want %q", got, `"pgvector_search"`)
	}
	if _, ok := decoded["tool_name"]; ok {
		t.Fatalf("encoded JSON unexpectedly included tool_name alias: %s", string(data))
	}
	if got := string(decoded["read_only"]); got != "true" {
		t.Fatalf("encoded read_only = %s, want true", got)
	}
	if got := string(decoded["source"]); got != `"extension"` {
		t.Fatalf("encoded source = %s, want %q", got, `"extension"`)
	}
}

func TestToolUnmarshalJSONCanonicalAndHookCompatible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
		want Tool
	}{
		{
			name: "canonical tool JSON",
			data: `{"name":"builtin_ls","description":"List files","input_schema":{"type":"object"},"read_only":true,"source":"builtin"}`,
			want: Tool{
				Name:        "builtin_ls",
				Description: "List files",
				InputSchema: json.RawMessage(`{"type":"object"}`),
				ReadOnly:    true,
				Source:      ToolSourceBuiltin,
			},
		},
		{
			name: "hook-compatible tool name alias",
			data: `{"tool_name":"mcp_search","description":"Search via MCP","input_schema":{"type":"object"},"read_only":false,"source":"mcp"}`,
			want: Tool{
				Name:        "mcp_search",
				Description: "Search via MCP",
				InputSchema: json.RawMessage(`{"type":"object"}`),
				ReadOnly:    false,
				Source:      ToolSourceMCP,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got Tool
			if err := json.Unmarshal([]byte(tt.data), &got); err != nil {
				t.Fatalf("json.Unmarshal(Tool) error = %v", err)
			}

			assertToolEqual(t, got, tt.want)
		})
	}
}

func TestToolUnmarshalJSONRejectsConflictingNames(t *testing.T) {
	t.Parallel()

	var got Tool
	err := json.Unmarshal([]byte(`{"name":"builtin_ls","tool_name":"mcp_search","source":"dynamic"}`), &got)
	if err == nil {
		t.Fatal("json.Unmarshal(conflicting Tool names) error = nil, want non-nil")
	}
}

func TestToolSourceOrderingAndJSON(t *testing.T) {
	t.Parallel()

	if ToolSourceBuiltin >= ToolSourceMCP ||
		ToolSourceMCP >= ToolSourceExtension ||
		ToolSourceExtension >= ToolSourceDynamic {
		t.Fatalf("unexpected ToolSource ordering: builtin=%d mcp=%d extension=%d dynamic=%d",
			ToolSourceBuiltin, ToolSourceMCP, ToolSourceExtension, ToolSourceDynamic)
	}

	data, err := json.Marshal(ToolSourceExtension)
	if err != nil {
		t.Fatalf("json.Marshal(ToolSourceExtension) error = %v", err)
	}
	if string(data) != `"extension"` {
		t.Fatalf("json.Marshal(ToolSourceExtension) = %s, want %q", string(data), `"extension"`)
	}

	var decoded ToolSource
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(ToolSource) error = %v", err)
	}
	if decoded != ToolSourceExtension {
		t.Fatalf("decoded ToolSource = %v, want %v", decoded, ToolSourceExtension)
	}
}

func TestToolSourceInvalid(t *testing.T) {
	t.Parallel()

	if got := ToolSource(42).String(); got != "" {
		t.Fatalf("ToolSource(42).String() = %q, want empty string", got)
	}
	if err := ToolSource(42).Validate(); err == nil {
		t.Fatal("ToolSource(42).Validate() error = nil, want non-nil")
	}
	if _, err := ToolSource(42).MarshalText(); err == nil {
		t.Fatal("ToolSource(42).MarshalText() error = nil, want non-nil")
	} else if !strings.Contains(err.Error(), "marshal tool source") {
		t.Fatalf("ToolSource(42).MarshalText() error = %v, want marshal context", err)
	}

	var decoded ToolSource
	if err := json.Unmarshal([]byte(`"remote"`), &decoded); err == nil {
		t.Fatal("json.Unmarshal(invalid ToolSource) error = nil, want non-nil")
	}
}
