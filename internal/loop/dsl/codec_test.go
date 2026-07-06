package dsl_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/loop/dsl"
)

func TestCodecShouldRoundTripContractOptionalFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		body             string
		wantConstraints  []string
		wantBoundaries   []string
		wantStopWhen     string
		wantSerializedIn []string
	}{
		{
			name: "Should default absent optional contract fields to empty",
			body: minimalDefinition(`
contract:
  goal: Ship safely
  definition_of_done: Tests pass
  verification: []
  terminal_states: [done, failed]
  iteration_cap: 3
  no_progress: { window: 2, hash_fields: [] }
  budget: { tokens: 0, wall_clock_sec: 0, on_exceeded: halt }
`),
			wantConstraints: []string{},
			wantBoundaries:  []string{},
		},
		{
			name: "Should preserve authored optional contract fields",
			body: minimalDefinition(`
contract:
  goal: Ship safely
  definition_of_done: Tests pass
  constraints: [keep scope]
  boundaries: [do not deploy]
  stop_when: "inputs.done == true"
  verification: []
  terminal_states: [done, failed]
  iteration_cap: 3
  no_progress: { window: 2, hash_fields: [] }
  budget: { tokens: 0, wall_clock_sec: 0, on_exceeded: halt }
`),
			wantConstraints:  []string{"keep scope"},
			wantBoundaries:   []string{"do not deploy"},
			wantStopWhen:     "inputs.done == true",
			wantSerializedIn: []string{"constraints:", "boundaries:", "stop_when:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def, err := dsl.Parse([]byte(tt.body))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(def.Contract.Constraints, tt.wantConstraints) {
				t.Fatalf(
					"Constraints = %#v, want %#v",
					def.Contract.Constraints,
					tt.wantConstraints,
				)
			}
			if !reflect.DeepEqual(def.Contract.Boundaries, tt.wantBoundaries) {
				t.Fatalf("Boundaries = %#v, want %#v", def.Contract.Boundaries, tt.wantBoundaries)
			}
			if def.Contract.StopWhen != tt.wantStopWhen {
				t.Fatalf("StopWhen = %q, want %q", def.Contract.StopWhen, tt.wantStopWhen)
			}

			serialized, err := dsl.Serialize(def)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}
			reparsed, err := dsl.Parse(serialized)
			if err != nil {
				t.Fatalf("Parse(serialized) error = %v; yaml=%s", err, string(serialized))
			}
			if !reflect.DeepEqual(reparsed.Contract.Constraints, tt.wantConstraints) {
				t.Fatalf(
					"round-trip Constraints = %#v, want %#v",
					reparsed.Contract.Constraints,
					tt.wantConstraints,
				)
			}
			for _, fragment := range tt.wantSerializedIn {
				if !containsString(string(serialized), fragment) {
					t.Fatalf("serialized YAML missing %q:\n%s", fragment, string(serialized))
				}
			}
		})
	}
}

func TestCodecShouldStructurallyMergeGraphUnknownFields(t *testing.T) {
	t.Parallel()

	def, err := dsl.Parse([]byte(`apiVersion: agh.loop/v1
kind: Loop
meta: { name: test-loop }
concurrency: forbid
inputs:
  done: { type: boolean, required: true }
x_root: keep
contract:
  goal: Ship safely
  definition_of_done: Tests pass
  constraints: [keep scope]
  boundaries: [do not deploy]
  stop_when: "inputs.done == true"
  verification: []
  terminal_states: [done, failed]
  iteration_cap: 3
  no_progress: { window: 2, hash_fields: [] }
  budget: { tokens: 0, wall_clock_sec: 0, on_exceeded: halt }
graph:
  nodes:
    - id: source
      class: source
      kind: input
      input_ref: done
      x_agent_authored: preserve
  edges:
    - from: source
      to: source
      label: preserve-label
      when: preserve-when
      x_edge_authored: preserve
start: [{ kind: manual }]
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	graph := dsl.DefinitionToGraph(def)
	merged := dsl.GraphToDefinition(graph)
	if !reflect.DeepEqual(merged.Graph, def.Graph) {
		t.Fatalf(
			"GraphToDefinition(DefinitionToGraph(def)).Graph = %#v, want %#v",
			merged.Graph,
			def.Graph,
		)
	}
	if merged.Extra["x_root"] != "keep" {
		t.Fatalf("merged root extra = %#v, want keep", merged.Extra["x_root"])
	}
	if got := merged.Graph.Nodes[0].Extra["x_agent_authored"]; got != "preserve" {
		t.Fatalf("merged node extra = %#v, want preserve", got)
	}
	if got := merged.Graph.Edges[0].Extra["x_edge_authored"]; got != "preserve" {
		t.Fatalf("merged edge extra = %#v, want preserve", got)
	}
	if got := merged.Graph.Edges[0].Extra["label"]; got != "preserve-label" {
		t.Fatalf("merged edge label extra = %#v, want preserve-label", got)
	}
	if got := merged.Graph.Edges[0].Extra["when"]; got != "preserve-when" {
		t.Fatalf("merged edge when extra = %#v, want preserve-when", got)
	}
}

func minimalDefinition(contract string) string {
	return `apiVersion: agh.loop/v1
kind: Loop
meta: { name: test-loop }
concurrency: forbid
inputs:
  done: { type: boolean, required: true }
` + contract + `
graph:
  nodes:
    - id: source
      class: source
      kind: input
      input_ref: done
      produces: { value: boolean }
  edges: []
start: [{ kind: manual }]
`
}

func containsString(value string, fragment string) bool {
	return strings.Contains(value, fragment)
}
