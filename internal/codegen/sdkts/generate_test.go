package sdkts

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

type EmbeddedJSONFields struct {
	Embedded string `json:"embedded,omitempty"`
	Ignored  string `json:"-"`
}

type ContainerJSONFields struct {
	EmbeddedJSONFields
	Alias    string          `json:",omitempty"`
	Optional *string         `json:"optional,omitempty"`
	Raw      json.RawMessage `json:"raw"`
}

type JSONTagFixture struct {
	DefaultName string
	Ignored     string `json:"-"`
	Explicit    string `json:"explicit,omitempty"`
	BlankName   string `json:",omitempty"`
}

func TestGenerateDeterministicAndStructured(t *testing.T) {
	t.Parallel()

	first, err := Generate()
	if err != nil {
		t.Fatalf("Generate() first call error = %v", err)
	}
	second, err := Generate()
	if err != nil {
		t.Fatalf("Generate() second call error = %v", err)
	}
	if first != second {
		t.Fatal("Generate() output changed between calls")
	}

	requiredSnippets := []string{
		generatedHeader,
		"export interface HookPayloadByEvent {\n",
		"export interface HookPatchByEvent {\n",
		"export interface HostAPIMethodMap {\n",
		"export interface HookDecl {\n",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(first, snippet) {
			t.Fatalf("Generate() output missing %q", snippet)
		}
	}
	if !strings.HasSuffix(first, "\n") {
		t.Fatal("Generate() output must end with a single trailing newline")
	}
	if strings.HasSuffix(first, "\n\n") {
		t.Fatal("Generate() output must not end with multiple trailing newlines")
	}
}

func TestNamedBaseType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		want  reflect.Type
	}{
		{name: "Nil", value: nil, want: nil},
		{name: "Struct", value: EmbeddedJSONFields{}, want: reflect.TypeFor[EmbeddedJSONFields]()},
		{name: "Pointer", value: &EmbeddedJSONFields{}, want: reflect.TypeFor[EmbeddedJSONFields]()},
		{name: "Slice", value: []EmbeddedJSONFields{}, want: reflect.TypeFor[EmbeddedJSONFields]()},
		{name: "Array", value: [1]EmbeddedJSONFields{}, want: reflect.TypeFor[EmbeddedJSONFields]()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := namedBaseType(tt.value); got != tt.want {
				t.Fatalf("namedBaseType(%T) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestJSONFieldName(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[JSONTagFixture]()
	tests := []struct {
		name          string
		field         reflect.StructField
		wantName      string
		wantOmitEmpty bool
	}{
		{name: "DefaultName", field: typ.Field(0), wantName: "DefaultName", wantOmitEmpty: false},
		{name: "Ignored", field: typ.Field(1), wantName: "", wantOmitEmpty: false},
		{name: "Explicit", field: typ.Field(2), wantName: "explicit", wantOmitEmpty: true},
		{name: "BlankName", field: typ.Field(3), wantName: "BlankName", wantOmitEmpty: true},
		{
			name: "SpacedTag",
			field: reflect.StructField{
				Name: "SpacedName",
				Tag:  reflect.StructTag(`json:" spaced_name , omitempty ,string"`),
			},
			wantName:      "spaced_name",
			wantOmitEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotName, gotOmitEmpty := jsonFieldName(tt.field)
			if gotName != tt.wantName || gotOmitEmpty != tt.wantOmitEmpty {
				t.Fatalf(
					"jsonFieldName(%s) = (%q, %t), want (%q, %t)",
					tt.field.Name,
					gotName,
					gotOmitEmpty,
					tt.wantName,
					tt.wantOmitEmpty,
				)
			}
		})
	}
}

func TestStructFieldsFlattensEmbeddedAndRespectsTags(t *testing.T) {
	t.Parallel()

	gen := newGenerator()
	gen.prepare()

	fields, err := gen.structFields(reflect.TypeFor[ContainerJSONFields]())
	if err != nil {
		t.Fatalf("structFields() error = %v", err)
	}

	want := []fieldSpec{
		{Name: "embedded", Type: "string", Optional: true},
		{Name: "Alias", Type: "string", Optional: true},
		{Name: "optional", Type: "string", Optional: true},
		{Name: "raw", Type: "JSONValue", Optional: false},
	}

	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("structFields() = %#v, want %#v", fields, want)
	}
}

func TestTSTypeCoversCompositeShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{name: "PointerString", typ: reflect.TypeFor[*string](), want: "string"},
		{name: "RawMessageSlice", typ: reflect.TypeFor[[]json.RawMessage](), want: "JSONValue[]"},
		{name: "TimeMap", typ: reflect.TypeFor[map[string]time.Time](), want: "Record<string, ISODateTime>"},
		{
			name: "InlineStruct",
			typ: reflect.TypeOf(struct {
				Name  string `json:"name"`
				Count *int   `json:"count,omitempty"`
			}{}),
			want: "{ name: string; count?: number }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := newGenerator()
			gen.prepare()

			got, err := gen.tsType(tt.typ)
			if err != nil {
				t.Fatalf("tsType() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("tsType(%v) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestResultIsList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "Nil", value: nil, want: false},
		{name: "Slice", value: []string{"a"}, want: true},
		{name: "Array", value: [1]string{"a"}, want: true},
		{name: "Struct", value: EmbeddedJSONFields{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := resultIsList(tt.value); got != tt.want {
				t.Fatalf("resultIsList(%T) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}
