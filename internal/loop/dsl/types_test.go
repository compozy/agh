package dsl_test

import (
	"testing"

	"github.com/compozy/agh/internal/loop/dsl"
)

func TestDefinitionShouldNormalizeAndValidateHeader(t *testing.T) {
	t.Parallel()

	def := dsl.Definition{}
	def.Normalize()
	if def.APIVersion != dsl.APIVersion {
		t.Fatalf("APIVersion = %q, want %q", def.APIVersion, dsl.APIVersion)
	}
	if def.Kind != dsl.KindLoop {
		t.Fatalf("Kind = %q, want %q", def.Kind, dsl.KindLoop)
	}
	if def.Concurrency != dsl.ConcurrencyForbid {
		t.Fatalf("Concurrency = %q, want %q", def.Concurrency, dsl.ConcurrencyForbid)
	}
	if def.Inputs == nil || def.Start == nil || def.Graph.Nodes == nil || def.Graph.Edges == nil {
		t.Fatal("Normalize() left nil authoring collections")
	}
	if err := def.ValidateHeader(); err != nil {
		t.Fatalf("ValidateHeader() error = %v", err)
	}

	def.APIVersion = "agh.loop/v2"
	if err := def.ValidateHeader(); err == nil {
		t.Fatal("ValidateHeader() error = nil for wrong apiVersion")
	}
	def.APIVersion = dsl.APIVersion
	def.Kind = "Workflow"
	if err := def.ValidateHeader(); err == nil {
		t.Fatal("ValidateHeader() error = nil for wrong kind")
	}
}

func TestEnumsShouldClassifyLoopKinds(t *testing.T) {
	t.Parallel()

	reserved := dsl.ReservedActionKinds()
	if len(reserved) != 3 {
		t.Fatalf("ReservedActionKinds() len = %d, want 3", len(reserved))
	}
	if !dsl.IsReservedActionKind(string(dsl.ActionRunAgent)) {
		t.Fatalf("IsReservedActionKind(%q) = false, want true", dsl.ActionRunAgent)
	}
	if dsl.IsReservedActionKind("agh__tool_info") {
		t.Fatal("IsReservedActionKind(open ToolID) = true, want false")
	}

	for _, kind := range []dsl.ControlKind{
		dsl.ControlFanOut,
		dsl.ControlCollect,
		dsl.ControlBranch,
		dsl.ControlGate,
		dsl.ControlSubLoop,
	} {
		if !dsl.IsKnownControlKind(string(kind)) {
			t.Fatalf("IsKnownControlKind(%q) = false, want true", kind)
		}
	}
	if dsl.IsKnownControlKind("parallel") {
		t.Fatal("IsKnownControlKind(unknown) = true, want false")
	}

	for _, kind := range []dsl.SourceKind{
		dsl.SourceInput,
		dsl.SourceFileImport,
		dsl.SourceWatchSource,
	} {
		if !dsl.IsKnownSourceKind(string(kind)) {
			t.Fatalf("IsKnownSourceKind(%q) = false, want true", kind)
		}
	}
	if dsl.IsKnownSourceKind("mailbox") {
		t.Fatal("IsKnownSourceKind(unknown) = true, want false")
	}
}

func TestNodeParamsShouldDecodePerKindSchemas(t *testing.T) {
	t.Parallel()

	runAgent := dsl.NodeParams{
		"agent":         "codex",
		"prompt":        "Ship it",
		"output_schema": map[string]any{"summary": "string"},
		"cwd":           "/repo",
		"model":         "gpt-5",
		"allowed_tools": []string{"agh__task_read"},
		"max_turns":     3,
	}
	var agentParams dsl.RunAgentParams
	if err := runAgent.Decode(&agentParams); err != nil {
		t.Fatalf("Decode(RunAgentParams) error = %v", err)
	}
	if agentParams.Agent != "codex" || agentParams.Prompt != "Ship it" {
		t.Fatalf("RunAgentParams = %#v, want decoded agent/prompt", agentParams)
	}
	if got := agentParams.OutputSchema["summary"]; got != "string" {
		t.Fatalf("OutputSchema[summary] = %#v, want string", got)
	}
	if len(agentParams.AllowedTools) != 1 || agentParams.AllowedTools[0] != "agh__task_read" {
		t.Fatalf("AllowedTools = %#v, want agh__task_read", agentParams.AllowedTools)
	}

	transform := dsl.NodeParams{
		"map": map[string]any{
			"summary": map[string]any{"template": "{{ .nodes.agent.output.summary }}"},
		},
	}
	var transformParams dsl.TransformParams
	if err := transform.Decode(&transformParams); err != nil {
		t.Fatalf("Decode(TransformParams) error = %v", err)
	}
	if transformParams.Map["summary"].Template == "" {
		t.Fatalf("TransformParams.Map = %#v, want summary template", transformParams.Map)
	}
}

func TestParseSerializeShouldRejectInvalidDocuments(t *testing.T) {
	t.Parallel()

	if _, err := dsl.Parse([]byte(" \n\t")); err == nil {
		t.Fatal("Parse(empty) error = nil")
	}
	if _, err := dsl.Parse([]byte("apiVersion: agh.loop/v2\nkind: Loop\n")); err == nil {
		t.Fatal("Parse(wrong apiVersion) error = nil")
	}
	if _, err := dsl.Parse([]byte("apiVersion: agh.loop/v1\nkind: [")); err == nil {
		t.Fatal("Parse(invalid YAML) error = nil")
	}
	if _, err := dsl.Serialize(dsl.Definition{APIVersion: dsl.APIVersion, Kind: "Workflow"}); err == nil {
		t.Fatal("Serialize(wrong kind) error = nil")
	}
}
