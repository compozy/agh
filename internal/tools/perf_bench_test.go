package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pedronauck/agh/internal/resources"
)

var (
	benchmarkCanonicalToolJSON = []byte(`{
		"name":"builtin_ls",
		"description":"List files",
		"input_schema":{"type":"object","properties":{"path":{"type":"string"}}},
		"read_only":true,
		"source":"builtin"
	}`)
	benchmarkHookToolJSON = []byte(`{
		"tool_name":"mcp_search",
		"description":"Search via MCP",
		"input_schema":{"type":"object","properties":{"query":{"type":"string"}}},
		"read_only":false,
		"source":"mcp"
	}`)
	benchmarkToolScope = resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
	benchmarkToolSpec  = Tool{
		Name:        "lookup",
		Description: "Search files",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`),
		ReadOnly:    true,
		Source:      ToolSourceExtension,
	}
)

func BenchmarkToolSourceString(b *testing.B) {
	b.ReportAllocs()

	source := ToolSourceExtension
	for b.Loop() {
		if source.String() == "" {
			b.Fatal("ToolSource.String() returned empty string")
		}
	}
}

func BenchmarkToolSourceMarshalText(b *testing.B) {
	b.ReportAllocs()

	source := ToolSourceExtension
	for b.Loop() {
		data, err := source.MarshalText()
		if err != nil {
			b.Fatalf("ToolSource.MarshalText() error = %v", err)
		}
		if len(data) == 0 {
			b.Fatal("ToolSource.MarshalText() returned empty data")
		}
	}
}

func BenchmarkToolSourceUnmarshalText(b *testing.B) {
	b.ReportAllocs()

	text := []byte("extension")
	for b.Loop() {
		var source ToolSource
		if err := (&source).UnmarshalText(text); err != nil {
			b.Fatalf("ToolSource.UnmarshalText() error = %v", err)
		}
	}
}

func BenchmarkToolUnmarshalJSON(b *testing.B) {
	b.ReportAllocs()

	benchmarks := []struct {
		name string
		data []byte
	}{
		{name: "canonical", data: benchmarkCanonicalToolJSON},
		{name: "hook_alias", data: benchmarkHookToolJSON},
	}

	for _, bench := range benchmarks {
		b.Run(bench.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var tool Tool
				if err := json.Unmarshal(bench.data, &tool); err != nil {
					b.Fatalf("json.Unmarshal(Tool) error = %v", err)
				}
			}
		})
	}
}

func BenchmarkValidateToolSpec(b *testing.B) {
	b.ReportAllocs()

	ctx := context.Background()
	for b.Loop() {
		if _, err := validateToolSpec(ctx, benchmarkToolScope, benchmarkToolSpec); err != nil {
			b.Fatalf("validateToolSpec() error = %v", err)
		}
	}
}
