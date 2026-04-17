package tools

import (
	"testing"

	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/testutil"
)

func mustToolResourceCodec(t *testing.T) resources.KindCodec[Tool] {
	t.Helper()

	codec, err := NewResourceCodec()
	if err != nil {
		t.Fatalf("NewResourceCodec() error = %v", err)
	}
	return codec
}

func TestToolResourceCodecCanonicalizesInputSchema(t *testing.T) {
	t.Parallel()

	codec := mustToolResourceCodec(t)

	scope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
	spec, err := codec.DecodeAndValidate(testutil.Context(t), scope, []byte(`{
		"name": " lookup ",
		"description": " search files ",
		"input_schema": {
			"properties": {"path": {"type": "string"}},
			"type": "object"
		},
		"read_only": true,
		"source": "extension"
	}`))
	if err != nil {
		t.Fatalf("codec.DecodeAndValidate() error = %v", err)
	}

	if got, want := spec.Name, "lookup"; got != want {
		t.Fatalf("spec.Name = %q, want %q", got, want)
	}
	if got, want := spec.Description, "search files"; got != want {
		t.Fatalf("spec.Description = %q, want %q", got, want)
	}
	if got, want := string(spec.InputSchema), `{"properties":{"path":{"type":"string"}},"type":"object"}`; got != want {
		t.Fatalf("spec.InputSchema = %s, want %s", got, want)
	}
}

func TestToolResourceCodecRejectsInvalidSchema(t *testing.T) {
	t.Parallel()

	codec := mustToolResourceCodec(t)

	_, err := codec.DecodeAndValidate(
		testutil.Context(t),
		resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		[]byte(`{
		"name": "lookup",
		"input_schema": "{not-json",
		"source": "extension"
	}`),
	)
	if err == nil {
		t.Fatal("codec.DecodeAndValidate() error = nil, want invalid input_schema failure")
	}
}
