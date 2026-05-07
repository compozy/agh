package model

import (
	"strings"
	"testing"
)

func TestValidateTriggerPromptTemplateAcceptsSupportedReferences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prompt string
	}{
		{
			name:   "Should accept top level fields",
			prompt: `Trigger kind: {{ .Kind }}`,
		},
		{
			name:   "Should accept data field chains",
			prompt: `Session: {{ .Data.session_id }}`,
		},
		{
			name:   "Should accept data index lookups",
			prompt: `Payload: {{ index .Data "payload" }}`,
		},
		{
			name:   "Should accept data scoped with blocks",
			prompt: `{{ with .Data }}{{ index . "session_id" }}{{ end }}`,
		},
		{
			name:   "Should accept data scoped field lookups",
			prompt: `{{ with .Data }}{{ .session_id }}{{ end }}`,
		},
		{
			name:   "Should accept range variables without variable rooted field lookups",
			prompt: `{{ range $key, $value := .Data }}{{ $key }}{{ end }}`,
		},
		{
			name:   "Should accept chained data expressions",
			prompt: `{{ (.Data).session_id }}`,
		},
		{
			name:   "Should accept defined templates with root envelope invocation",
			prompt: `{{ define "body" }}{{ .Source }}{{ end }}{{ template "body" . }}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := ValidateTriggerPromptTemplate(tt.prompt); err != nil {
				t.Fatalf("ValidateTriggerPromptTemplate() error = %v", err)
			}
		})
	}
}

func TestValidateTriggerPromptTemplateRejectsUnsupportedReferences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{
			name:   "Should reject unknown top level fields",
			prompt: `{{ .EnvelopeID }}`,
			want:   []string{"EnvelopeID"},
		},
		{
			name:   "Should reject child fields on scalar values",
			prompt: `{{ .Scope.Name }}`,
			want:   []string{"Scope"},
		},
		{
			name:   "Should reject chained scalar field lookups",
			prompt: `{{ (.Source).Value }}`,
			want:   []string{"Source"},
		},
		{
			name:   "Should reject non data index targets",
			prompt: `{{ index .Kind "anything" }}`,
			want:   []string{".Kind"},
		},
		{
			name:   "Should reject root dot index targets",
			prompt: `{{ index . "payload" }}`,
			want:   []string{"only .Data"},
		},
		{
			name:   "Should reject variable rooted lookups",
			prompt: `{{ range $key, $value := .Data }}{{ $value.name }}{{ end }}`,
			want:   []string{"variable-rooted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTriggerPromptTemplate(tt.prompt)
			requireErrorContains(t, err, tt.want...)
		})
	}
}

func TestParseTriggerPromptTemplateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{
			name:   "Should reject empty prompts",
			prompt: "   ",
			want:   []string{"required"},
		},
		{
			name:   "Should reject template syntax errors",
			prompt: "{{ if .Kind }}",
			want:   []string{"parse"},
		},
		{
			name:   "Should wrap validation failures with template context",
			prompt: `{{ .EnvelopeID }}`,
			want:   []string{`validate trigger prompt template "trigger_prompt"`, "EnvelopeID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseTriggerPromptTemplate(tt.prompt)
			requireErrorContains(t, err, tt.want...)
		})
	}
}

func requireErrorContains(t *testing.T, err error, want ...string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want substrings %#v", want)
	}
	got := err.Error()
	for _, substring := range want {
		if !strings.Contains(got, substring) {
			t.Fatalf("error = %q, want substring %q", got, substring)
		}
	}
}
