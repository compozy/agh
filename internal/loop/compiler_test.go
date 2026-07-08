package loop_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/tools"
)

func TestCompilerShouldBuildResolvedFormWithParsedReferencesAndSchemaDigests(t *testing.T) {
	t.Parallel()

	t.Run("Should build resolved form with parsed references and schema digests", func(t *testing.T) {
		t.Parallel()

		def := compilerDefinition(t)
		toolID := tools.ToolIDTaskRead.String()
		source := fakeToolSchemas{
			toolID: {
				ToolID:             toolID,
				InputSchemaDigest:  "sha256:input",
				OutputSchemaDigest: "sha256:output",
				InputSchema: mustJSON(t, map[string]any{
					"type":       "object",
					"properties": map[string]any{"id": map[string]any{"type": "string"}},
				}),
				OutputSchema: mustJSON(t, map[string]any{
					"type": "object",
					"properties": map[string]any{
						"items": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":       "object",
								"properties": map[string]any{"title": map[string]any{"type": "string"}},
							},
						},
					},
				}),
			},
		}

		resolved, err := loop.NewCompiler(loop.WithCompilerToolSchemaSource(source)).Compile(def)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		if got := requireNode(t, &def, "fan").MaxParallel; got != 0 {
			t.Fatalf("Compile() mutated input fan MaxParallel = %d, want 0", got)
		}
		if _, ok := requireNode(t, &def, "child_loop").Params["mode"]; ok {
			t.Fatal("Compile() mutated input child_loop params.mode")
		}

		if resolved.Templates["start[0].input_mapping.tasks"] == nil {
			t.Fatal("resolved template for trigger input mapping is nil")
		}
		if resolved.Templates["nodes.read_task.params.id"] == nil {
			t.Fatal("resolved template for read_task params.id is nil")
		}
		if resolved.Templates["nodes.agent.params.prompt"] == nil {
			t.Fatal("resolved template for agent prompt is nil")
		}
		if resolved.Templates["contract.verification[0].prompt"] == nil {
			t.Fatal("resolved template for contract verification prompt is nil")
		}
		if resolved.Conditions["contract.stop_when"] == nil {
			t.Fatal("resolved contract.stop_when condition is nil")
		}
		if resolved.Conditions["nodes.decision.condition"] == nil {
			t.Fatal("resolved branch condition is nil")
		}
		snapshot, ok := resolved.ToolSchemas[toolID]
		if !ok {
			t.Fatalf("resolved tool schema for %s missing", toolID)
		}
		if snapshot.OutputSchemaDigest != "sha256:output" {
			t.Fatalf("OutputSchemaDigest = %q, want sha256:output", snapshot.OutputSchemaDigest)
		}
		if resolved.Defaults.FanOutBatchSize != 1 {
			t.Fatalf("FanOutBatchSize = %d, want 1", resolved.Defaults.FanOutBatchSize)
		}
		if resolved.Defaults.RunLoopMode != dsl.RunLoopAwait {
			t.Fatalf("RunLoopMode = %q, want %q", resolved.Defaults.RunLoopMode, dsl.RunLoopAwait)
		}
		if resolved.Defaults.Concurrency != dsl.ConcurrencyQueue {
			t.Fatalf("Concurrency = %q, want %q", resolved.Defaults.Concurrency, dsl.ConcurrencyQueue)
		}

		fan := requireNode(t, &resolved.Definition, "fan")
		if fan.BatchSize != 1 || fan.MaxParallel != 0 {
			t.Fatalf(
				"folded fan-out defaults = batch_size:%d max_parallel:%d, want 1/0",
				fan.BatchSize,
				fan.MaxParallel,
			)
		}
		child := requireNode(t, &resolved.Definition, "child_loop")
		if got := child.Params["mode"]; got != string(dsl.RunLoopAwait) {
			t.Fatalf("folded run-loop params.mode = %#v, want %q", got, dsl.RunLoopAwait)
		}
	})
}

func TestCompilerShouldReturnLintFailedErrorWhenDefinitionIsInvalid(t *testing.T) {
	t.Parallel()

	t.Run("Should return lint failed error when definition is invalid", func(t *testing.T) {
		t.Parallel()

		def := validDefinition()
		requireNode(t, &def, "fan").MaxFanOut = loop.LoopMaxFanoutWidth + 1

		_, err := loop.NewCompiler(loop.WithCompilerToolSchemaSource(fakeToolSchemas{})).Compile(def)
		if err == nil {
			t.Fatal("Compile() error = nil, want lint failure")
		}
		var lintErr *loop.LintFailedError
		if !errors.As(err, &lintErr) {
			t.Fatalf("Compile() error = %T %v, want *loop.LintFailedError", err, err)
		}
		if err.Error() == "" {
			t.Fatal("Compile() returned empty lint failure message")
		}
		if !strings.Contains(err.Error(), loop.CodeFanOutCeilingExceeded) {
			t.Fatalf("Compile() error = %q, want code %q", err.Error(), loop.CodeFanOutCeilingExceeded)
		}
		requireLintCodes(t, lintErr.Errors, loop.CodeFanOutCeilingExceeded)
	})
}

func TestCompilerShouldCompileWatchEventsFiltersWithEventEnv(t *testing.T) {
	t.Parallel()

	t.Run("Should compile watch-events filters with event inputs and node references", func(t *testing.T) {
		t.Parallel()

		def := watchEventsCompilerDefinition()
		resolved, err := loop.NewCompiler().Compile(def)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		if resolved.Conditions["nodes.task_activity.events.0.filter"] == nil {
			t.Fatal("resolved first watch-events filter condition is nil")
		}
		if resolved.Conditions["nodes.task_activity.events.1.filter"] == nil {
			t.Fatal("resolved second watch-events filter condition is nil")
		}
		if resolved.Templates["nodes.summarize.params.prompt"] == nil {
			t.Fatal("resolved summarize prompt template is nil")
		}
		if got := requireNode(t, &def, "task_activity").Events[0].Filter; got == "" {
			t.Fatal("Compile() mutated input watch-events filter")
		}
	})
}

func TestCompilerShouldRejectInvalidWatchEventsFilterAtPublish(t *testing.T) {
	t.Parallel()

	t.Run("Should return lint failed error for unknown watch-events filter variables", func(t *testing.T) {
		t.Parallel()

		def := watchEventsCompilerDefinition()
		requireNode(t, &def, "task_activity").Events[0].Filter = `missing.task_id == inputs.parent_task_id`
		_, err := loop.NewCompiler().Compile(def)
		if err == nil {
			t.Fatal("Compile() error = nil, want lint failure")
		}
		var lintErr *loop.LintFailedError
		if !errors.As(err, &lintErr) {
			t.Fatalf("Compile() error = %T %v, want *loop.LintFailedError", err, err)
		}
		requireLintCodes(t, lintErr.Errors, loop.CodeWatchEventsFilterInvalid)
	})
}

func TestCompilerShouldCompileEpicWatchdogFixture(t *testing.T) {
	t.Parallel()

	t.Run("Should compile the phase-A epic-watchdog fixture", func(t *testing.T) {
		t.Parallel()

		def, err := dsl.Parse([]byte(epicWatchdogFixture))
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		resolved, err := loop.NewCompiler().Compile(def)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		if resolved.Conditions["nodes.task_activity.events.0.filter"] == nil {
			t.Fatal("resolved first epic-watchdog filter condition is nil")
		}
		if resolved.Conditions["nodes.task_activity.events.1.filter"] == nil {
			t.Fatal("resolved second epic-watchdog filter condition is nil")
		}
		if resolved.Templates["nodes.summarize.params.prompt"] == nil {
			t.Fatal("resolved epic-watchdog summarize prompt template is nil")
		}
	})
}

func TestCompilerShouldCompileSubLoopBodyWithQualifiedKeys(t *testing.T) {
	t.Parallel()

	t.Run("Should compile sub-loop body with qualified keys", func(t *testing.T) {
		t.Parallel()

		def := validDefinition()
		def.Graph.Nodes = []dsl.Node{
			{
				ID:       "load",
				Class:    dsl.NodeClassSource,
				Kind:     string(dsl.SourceInput),
				InputRef: "tasks",
			},
			{
				ID:    "nested",
				Class: dsl.NodeClassControl,
				Kind:  string(dsl.ControlSubLoop),
				Body: &dsl.Graph{
					Nodes: []dsl.Node{
						{
							ID:        "decision",
							Class:     dsl.NodeClassControl,
							Kind:      string(dsl.ControlBranch),
							Condition: "inputs.tasks != null",
						},
						{
							ID:    "worker",
							Class: dsl.NodeClassAction,
							Kind:  string(dsl.ActionRunAgent),
							Params: dsl.NodeParams{
								"agent":  "codex",
								"prompt": "Handle {{ .inputs.tasks }}",
							},
						},
					},
					Edges: []dsl.Edge{{From: "decision", To: "worker"}},
				},
			},
		}
		def.Graph.Edges = []dsl.Edge{{From: "load", To: "nested"}}

		resolved, err := loop.NewCompiler().Compile(def)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		if resolved.Conditions["nodes.nested__decision.condition"] == nil {
			t.Fatal("resolved nested branch condition is nil")
		}
		if resolved.Templates["nodes.nested__worker.params.prompt"] == nil {
			t.Fatal("resolved nested worker prompt is nil")
		}
	})
}

func watchEventsCompilerDefinition() dsl.Definition {
	def := validDefinition()
	def.Inputs = map[string]dsl.Input{
		"parent_task_id": {Type: dsl.InputTypeString, Required: true},
		"runner":         {Type: dsl.InputTypeAgent, Default: "code_implementer"},
	}
	def.Contract.Goal = "React to task events"
	def.Contract.DefinitionOfDone = "The watched task activity was summarized"
	def.Contract.IterationCap = 0
	def.Graph.Nodes = []dsl.Node{
		{
			ID:    "task_activity",
			Class: dsl.NodeClassSource,
			Kind:  string(dsl.SourceWatchEvents),
			Events: []dsl.EventSubscription{
				{
					Kind:   "task.status_changed",
					Filter: `event.task_id == inputs.parent_task_id && event.payload.to_status in ["blocked", "completed", "failed"]`,
				},
				{
					Kind:   "task.run.completed",
					Filter: `event.task_id == inputs.parent_task_id`,
				},
			},
		},
		{
			ID:    "summarize",
			Class: dsl.NodeClassAction,
			Kind:  string(dsl.ActionRunAgent),
			Params: dsl.NodeParams{
				"agent":  "{{ .inputs.runner }}",
				"prompt": "Summarize {{ .nodes.task_activity.output.events }}",
			},
		},
	}
	def.Graph.Edges = []dsl.Edge{{From: "task_activity", To: "summarize"}}
	return def
}

const epicWatchdogFixture = `apiVersion: agh.loop/v1
kind: Loop
meta:
  name: epic-watchdog
  description: Watch a tracked parent task and react when its child work changes status.

inputs:
  parent_task_id:
    type: string
    required: true
  runner:
    type: agent
    default: code_implementer

contract:
  goal: "React to task lifecycle changes under {{ .inputs.parent_task_id }}."
  definition_of_done: "The watched task activity was summarized."
  iteration_cap: 0
  terminal_states: [done, blocked, failed, stalled]

graph:
  nodes:
    - id: task_activity
      class: source
      kind: watch-events
      events:
        - kind: task.status_changed
          filter: "event.task_id == inputs.parent_task_id && event.payload.to_status in ['blocked', 'completed', 'failed']"
        - kind: task.run.completed
          filter: "event.task_id == inputs.parent_task_id"

    - id: summarize
      class: action
      kind: run-agent
      params:
        agent: "{{ .inputs.runner }}"
        prompt: "Summarize the watched task activity: {{ toJson .nodes.task_activity.output.events }}"
`

func compilerDefinition(t *testing.T) dsl.Definition {
	t.Helper()

	def := validDefinition()
	def.Concurrency = dsl.ConcurrencyQueue
	def.Contract.StopWhen = "generation > 10"
	def.Contract.Verification = []dsl.GateCriterion{{
		ID:     "dod",
		Type:   dsl.CriterionAgentJudge,
		Agent:  "critic",
		Prompt: "Judge {{ .inputs.tasks }}",
		Rubric: "Require complete summaries",
	}}
	def.Start = []dsl.StartBinding{
		{
			Kind: dsl.StartTrigger,
			InputMapping: map[string]string{
				"tasks": "{{ .trigger.payload.tasks }}",
			},
		},
	}

	toolID := tools.ToolIDTaskRead.String()
	def.Graph.Nodes = []dsl.Node{
		{
			ID:       "load",
			Class:    dsl.NodeClassSource,
			Kind:     string(dsl.SourceInput),
			InputRef: "tasks",
			Produces: dsl.Schema{
				"task_id": "string",
			},
		},
		{
			ID:    "read_task",
			Class: dsl.NodeClassAction,
			Kind:  toolID,
			Params: dsl.NodeParams{
				"id": "{{ .nodes.load.output.task_id }}",
			},
		},
		{
			ID:         "fan",
			Class:      dsl.NodeClassControl,
			Kind:       string(dsl.ControlFanOut),
			Collection: "{{ .nodes.read_task.output.items }}",
			MaxFanOut:  4,
		},
		{
			ID:    "agent",
			Class: dsl.NodeClassAction,
			Kind:  string(dsl.ActionRunAgent),
			Params: dsl.NodeParams{
				"agent":  "codex",
				"prompt": "Summarize {{ .item.title }}",
				"output_schema": map[string]any{
					"summary": "string",
				},
			},
		},
		{
			ID:        "decision",
			Class:     dsl.NodeClassControl,
			Kind:      string(dsl.ControlBranch),
			Condition: `nodes.agent.output.summary != ""`,
		},
		{
			ID:    "child_loop",
			Class: dsl.NodeClassAction,
			Kind:  string(dsl.ActionRunLoop),
			Params: dsl.NodeParams{
				"loop": "child-loop",
				"inputs": map[string]any{
					"summary": "{{ .nodes.agent.output.summary }}",
				},
			},
		},
	}
	def.Graph.Edges = []dsl.Edge{
		{From: "load", To: "read_task"},
		{From: "read_task", To: "fan"},
		{From: "fan", To: "agent"},
		{From: "agent", To: "decision"},
		{From: "decision", To: "child_loop"},
	}
	return def
}
