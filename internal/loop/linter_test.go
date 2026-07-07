package loop_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/loop/dsl/refs"
)

func TestLinterShouldRejectStructuralAndReferenceInvalidShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*dsl.Definition)
		wantCodes []string
	}{
		{
			name: "Should accept valid finite graph",
		},
		{
			name: "Should reject graph cycles",
			mutate: func(def *dsl.Definition) {
				def.Graph.Edges = append(def.Graph.Edges, dsl.Edge{From: "agent", To: "load"})
			},
			wantCodes: []string{loop.CodeCycle},
		},
		{
			name: "Should reject unreachable nodes",
			mutate: func(def *dsl.Definition) {
				def.Graph.Nodes = append(def.Graph.Nodes, dsl.Node{
					ID:    "orphan",
					Class: dsl.NodeClassAction,
					Kind:  string(dsl.ActionTransform),
					Params: dsl.NodeParams{
						"map": map[string]any{
							"value": map[string]any{"value": "literal"},
						},
					},
				})
			},
			wantCodes: []string{loop.CodeUnreachableNode},
		},
		{
			name: "Should reject empty non terminating graph",
			mutate: func(def *dsl.Definition) {
				def.Graph.Nodes = nil
				def.Graph.Edges = nil
			},
			wantCodes: []string{loop.CodeNonTerminatingStructure},
		},
		{
			name: "Should reject unbounded fan out",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "fan")
				node.MaxFanOut = 0
			},
			wantCodes: []string{loop.CodeFanOutUnbounded},
		},
		{
			name: "Should reject fan out over ceiling",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "fan")
				node.MaxFanOut = loop.LoopMaxFanoutWidth + 1
			},
			wantCodes: []string{loop.CodeFanOutCeilingExceeded},
		},
		{
			name: "Should reject fan out collection literals without references",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "fan")
				node.Collection = "nodes.load.output.items"
			},
			wantCodes: []string{loop.CodeFanOutUnbounded},
		},
		{
			name: "Should reject gate max revisions over ceiling",
			mutate: func(def *dsl.Definition) {
				appendGate(def, dsl.Node{
					ID:            "review_gate",
					Class:         dsl.NodeClassControl,
					Kind:          string(dsl.ControlGate),
					MaxRevisions:  loop.LoopMaxGateRevisions + 1,
					VerdictPolicy: dsl.VerdictPolicyReviseUntilClean,
					Criteria: []dsl.GateCriterion{
						{
							ID:     "judge",
							Type:   dsl.CriterionAgentJudge,
							Agent:  "critic",
							Rubric: "Approve only correct work",
						},
					},
				})
			},
			wantCodes: []string{loop.CodeGateMaxRevisionsCeilingExceeded},
		},
		{
			name: "Should reject revise until clean gate without judge",
			mutate: func(def *dsl.Definition) {
				appendGate(def, dsl.Node{
					ID:            "review_gate",
					Class:         dsl.NodeClassControl,
					Kind:          string(dsl.ControlGate),
					MaxRevisions:  1,
					VerdictPolicy: dsl.VerdictPolicyReviseUntilClean,
					Criteria: []dsl.GateCriterion{
						{
							ID:     "unit",
							Type:   dsl.CriterionCommand,
							Check:  "make test",
							Expect: "exit 0",
						},
					},
				})
			},
			wantCodes: []string{loop.CodeVerdictPolicyRequiresJudge},
		},
		{
			name: "Should reject node ids outside snake case",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.ID = "Bad-ID"
			},
			wantCodes: []string{loop.CodeNodeIDInvalid},
		},
		{
			name: "Should reject unknown terminal states",
			mutate: func(def *dsl.Definition) {
				def.Contract.TerminalStates = append(def.Contract.TerminalStates, "canceled")
			},
			wantCodes: []string{loop.CodeUnknownTerminalState},
		},
		{
			name: "Should reject source input refs not declared as inputs",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "load")
				node.InputRef = "missing"
			},
			wantCodes: []string{refs.CodeUnknownReference},
		},
		{
			name: "Should reject deleted channel post action as unknown",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Kind = "channel-post"
			},
			wantCodes: []string{loop.CodeUnknownActionKind},
		},
		{
			name: "Should reject channel result harvest on reserved action kinds",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Harvest = &dsl.HarvestSpec{Kind: "channel_result", Window: "5m"}
			},
			wantCodes: []string{loop.CodeInvalidHarvest},
		},
		{
			name: "Should reject channel result harvest on non network send actions",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Kind = "agh__task_read"
				node.Harvest = &dsl.HarvestSpec{Kind: "channel_result", Window: "5m"}
			},
			wantCodes: []string{loop.CodeInvalidHarvest},
		},
		{
			name: "Should reject channel result harvest without a positive window",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Kind = "agh__network_send"
				node.Harvest = &dsl.HarvestSpec{Kind: "channel_result", Window: "0s"}
			},
			wantCodes: []string{loop.CodeInvalidHarvest},
		},
		{
			name: "Should reject unsupported channel result content rules",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Kind = "agh__network_send"
				node.Harvest = &dsl.HarvestSpec{
					Kind:        "channel_result",
					Window:      "5m",
					ContentRule: "jsonpath:decision",
				}
			},
			wantCodes: []string{loop.CodeInvalidHarvest},
		},
		{
			name: "Should reject unknown harvest kinds on open actions",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Kind = "agh__task_read"
				node.Harvest = &dsl.HarvestSpec{Kind: "event_rang"}
			},
			wantCodes: []string{loop.CodeInvalidHarvest},
		},
		{
			name: "Should reject unresolved open action ToolIDs",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Kind = "agh__missing_tool"
			},
			wantCodes: []string{loop.CodeUnknownActionKind},
		},
		{
			name: "Should reject unknown template references",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Params["prompt"] = "{{ .inputs.missing }}"
			},
			wantCodes: []string{refs.CodeUnknownReference},
		},
		{
			name: "Should reject unresolvable template paths",
			mutate: func(def *dsl.Definition) {
				node := requireNode(t, def, "agent")
				node.Params["prompt"] = "{{ .nodes.load.output.missing }}"
			},
			wantCodes: []string{refs.CodeUnresolvablePath},
		},
		{
			name: "Should reject item outside fanout",
			mutate: func(def *dsl.Definition) {
				agent := requireNode(t, def, "agent")
				agent.Params["prompt"] = "{{ .item.title }}"
				def.Graph.Nodes = []dsl.Node{*requireNode(t, def, "load"), *agent}
				def.Graph.Edges = []dsl.Edge{{From: "load", To: "agent"}}
			},
			wantCodes: []string{refs.CodeItemOutsideFanout},
		},
		{
			name: "Should reject non bool CEL conditions",
			mutate: func(def *dsl.Definition) {
				appendGate(def, dsl.Node{
					ID:        "decision",
					Class:     dsl.NodeClassControl,
					Kind:      string(dsl.ControlBranch),
					Condition: "1 + 1",
				})
			},
			wantCodes: []string{refs.CodeConditionNotBool},
		},
		{
			name: "Should accept trigger input mapping payload references for trigger starts",
			mutate: func(def *dsl.Definition) {
				def.Start = []dsl.StartBinding{{
					Kind: dsl.StartTrigger,
					InputMapping: map[string]string{
						"tasks": "{{ .trigger.payload.tasks }}",
					},
				}}
			},
		},
		{
			name: "Should reject trigger input mapping payload references for manual starts",
			mutate: func(def *dsl.Definition) {
				def.Start = []dsl.StartBinding{{
					Kind: dsl.StartManual,
					InputMapping: map[string]string{
						"tasks": "{{ .trigger.payload.tasks }}",
					},
				}}
			},
			wantCodes: []string{refs.CodeUnknownReference},
		},
		{
			name: "Should reject start input mapping keys outside declared inputs",
			mutate: func(def *dsl.Definition) {
				def.Start = []dsl.StartBinding{{
					Kind: dsl.StartTrigger,
					InputMapping: map[string]string{
						"missing": "{{ .trigger.payload.tasks }}",
					},
				}}
			},
			wantCodes: []string{refs.CodeUnknownReference},
		},
		{
			name: "Should reject contract verification template references that do not resolve",
			mutate: func(def *dsl.Definition) {
				def.Contract.Verification = []dsl.GateCriterion{{
					ID:     "dod",
					Type:   dsl.CriterionAgentJudge,
					Agent:  "critic",
					Rubric: "Judge {{ .inputs.missing }}",
				}}
			},
			wantCodes: []string{refs.CodeUnknownReference},
		},
		{
			name: "Should reject item references after collect barrier",
			mutate: func(def *dsl.Definition) {
				def.Graph.Nodes = append(def.Graph.Nodes,
					dsl.Node{
						ID:    "collect",
						Class: dsl.NodeClassControl,
						Kind:  string(dsl.ControlCollect),
					},
					dsl.Node{
						ID:    "after_collect",
						Class: dsl.NodeClassAction,
						Kind:  string(dsl.ActionRunAgent),
						Params: dsl.NodeParams{
							"agent":  "codex",
							"prompt": "Outside {{ .item.title }}",
							"output_schema": map[string]any{
								"summary": "string",
							},
						},
					},
				)
				def.Graph.Edges = append(def.Graph.Edges,
					dsl.Edge{From: "agent", To: "collect"},
					dsl.Edge{From: "collect", To: "after_collect"},
				)
			},
			wantCodes: []string{refs.CodeItemOutsideFanout},
		},
		{
			name: "Should accept CEL member functions over node outputs",
			mutate: func(def *dsl.Definition) {
				def.Contract.StopWhen = `nodes.agent.output.summary.contains("done")`
			},
		},
		{
			name: "Should accept CEL indexed node output fields",
			mutate: func(def *dsl.Definition) {
				agent := requireNode(t, def, "agent")
				agent.Params["output_schema"] = map[string]any{
					"issues": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id": map[string]any{"type": "string"},
							},
						},
					},
				}
				def.Contract.StopWhen = `nodes.agent.output.issues[0].id == "issue-1"`
			},
		},
		{
			name: "Should reject CEL indexed node output fields that do not resolve",
			mutate: func(def *dsl.Definition) {
				agent := requireNode(t, def, "agent")
				agent.Params["output_schema"] = map[string]any{
					"issues": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id": map[string]any{"type": "string"},
							},
						},
					},
				}
				def.Contract.StopWhen = `nodes.agent.output.issues[0].missing == "issue-1"`
			},
			wantCodes: []string{refs.CodeUnresolvablePath},
		},
		{
			name: "Should reject transform from paths that do not resolve",
			mutate: func(def *dsl.Definition) {
				def.Graph.Nodes = append(def.Graph.Nodes, dsl.Node{
					ID:    "shape",
					Class: dsl.NodeClassAction,
					Kind:  string(dsl.ActionTransform),
					Params: dsl.NodeParams{
						"map": map[string]any{
							"summary": map[string]any{
								"from": "nodes.agent.output.missing",
							},
						},
					},
				})
				def.Graph.Edges = append(def.Graph.Edges, dsl.Edge{From: "agent", To: "shape"})
			},
			wantCodes: []string{refs.CodeUnresolvablePath},
		},
		{
			name: "Should accept transform value literals that look like templates",
			mutate: func(def *dsl.Definition) {
				def.Graph.Nodes = append(def.Graph.Nodes, dsl.Node{
					ID:    "shape",
					Class: dsl.NodeClassAction,
					Kind:  string(dsl.ActionTransform),
					Params: dsl.NodeParams{
						"map": map[string]any{
							"literal": map[string]any{
								"value": "{{ .inputs.missing }} stays literal",
							},
						},
					},
				})
				def.Graph.Edges = append(def.Graph.Edges, dsl.Edge{From: "agent", To: "shape"})
			},
		},
		{
			name: "Should validate nested params named output schema outside declared schemas",
			mutate: func(def *dsl.Definition) {
				def.Graph.Nodes = append(def.Graph.Nodes, dsl.Node{
					ID:    "shape",
					Class: dsl.NodeClassAction,
					Kind:  string(dsl.ActionTransform),
					Params: dsl.NodeParams{
						"map": map[string]any{
							"output_schema": map[string]any{
								"template": "{{ .inputs.missing }}",
							},
						},
					},
				})
				def.Graph.Edges = append(def.Graph.Edges, dsl.Edge{From: "agent", To: "shape"})
			},
			wantCodes: []string{refs.CodeUnknownReference},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := validDefinition()
			if tt.mutate != nil {
				tt.mutate(&def)
			}

			linter := loop.NewLinter(loop.WithToolSchemaSource(fakeToolSchemas{}))
			errs := linter.Lint(def)
			requireLintCodes(t, errs, tt.wantCodes...)
		})
	}
}

func TestLinterShouldAcceptCompileTimeCeilingsAtBoundary(t *testing.T) {
	t.Parallel()

	t.Run("Should accept configured ceilings exactly at the boundary", func(t *testing.T) {
		t.Parallel()

		def := validDefinition()
		requireNode(t, &def, "fan").MaxFanOut = loop.LoopMaxFanoutWidth
		requireNode(t, &def, "fan").MaxParallel = loop.LoopMaxFanoutWidth
		appendGate(&def, dsl.Node{
			ID:            "review_gate",
			Class:         dsl.NodeClassControl,
			Kind:          string(dsl.ControlGate),
			MaxRevisions:  loop.LoopMaxGateRevisions,
			VerdictPolicy: dsl.VerdictPolicyReviseUntilClean,
			Criteria: []dsl.GateCriterion{
				{
					ID:     "judge",
					Type:   dsl.CriterionAgentJudge,
					Agent:  "critic",
					Rubric: "Approve only correct work",
				},
			},
		})

		linter := loop.NewLinter(loop.WithToolSchemaSource(fakeToolSchemas{}))
		requireLintCodes(t, linter.Lint(def))
	})
}

func TestLinterShouldWarnForUnknownOutputSchemaReferences(t *testing.T) {
	t.Parallel()

	t.Run("Should warn when output schema references are unavailable", func(t *testing.T) {
		t.Parallel()

		def := validDefinition()
		agent := requireNode(t, &def, "agent")
		delete(agent.Params, "output_schema")
		def.Graph.Nodes = append(def.Graph.Nodes, dsl.Node{
			ID:    "after_agent",
			Class: dsl.NodeClassAction,
			Kind:  string(dsl.ActionRunAgent),
			Params: dsl.NodeParams{
				"agent":  "codex",
				"prompt": "Use {{ .nodes.agent.output.summary }}",
				"output_schema": map[string]any{
					"done": "boolean",
				},
			},
		})
		def.Graph.Edges = append(def.Graph.Edges, dsl.Edge{From: "agent", To: "after_agent"})

		linter := loop.NewLinter(loop.WithToolSchemaSource(fakeToolSchemas{}))
		requireLintWarning(t, linter.Lint(def), refs.CodeUnresolvablePath)
	})
}

func TestLinterShouldRejectClosedEnumAndReservedSchemaViolations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		def       dsl.Definition
		wantCodes []string
	}{
		{
			name: "Should reject unknown node classes",
			def: singleNodeDefinition(dsl.Node{
				ID:    "mystery",
				Class: "custom",
				Kind:  "anything",
			}),
			wantCodes: []string{refs.CodeUnknownReference},
		},
		{
			name: "Should reject unknown control kinds",
			def: singleNodeDefinition(dsl.Node{
				ID:    "control",
				Class: dsl.NodeClassControl,
				Kind:  "parallel",
			}),
			wantCodes: []string{loop.CodeUnknownControlKind},
		},
		{
			name: "Should reject synthetic budget gate node id",
			def: singleNodeDefinition(dsl.Node{
				ID:    loop.BudgetGateID,
				Class: dsl.NodeClassControl,
				Kind:  string(dsl.ControlGate),
			}),
			wantCodes: []string{loop.CodeNodeIDInvalid},
		},
		{
			name: "Should reject unknown source kinds",
			def: singleNodeDefinition(dsl.Node{
				ID:    "source",
				Class: dsl.NodeClassSource,
				Kind:  "mailbox",
			}),
			wantCodes: []string{loop.CodeUnknownSourceKind},
		},
		{
			name: "Should reject unresolved open action ToolIDs",
			def: singleNodeDefinition(dsl.Node{
				ID:    "bad_tool",
				Class: dsl.NodeClassAction,
				Kind:  "BadTool",
			}),
			wantCodes: []string{loop.CodeUnknownActionKind},
		},
		{
			name: "Should reject run agent missing required params",
			def: singleNodeDefinition(dsl.Node{
				ID:     "agent",
				Class:  dsl.NodeClassAction,
				Kind:   string(dsl.ActionRunAgent),
				Params: dsl.NodeParams{},
			}),
			wantCodes: []string{refs.CodeUnresolvablePath},
		},
		{
			name: "Should reject run loop missing loop and invalid mode",
			def: singleNodeDefinition(dsl.Node{
				ID:    "child",
				Class: dsl.NodeClassAction,
				Kind:  string(dsl.ActionRunLoop),
				Params: dsl.NodeParams{
					"mode": "later",
				},
			}),
			wantCodes: []string{refs.CodeUnresolvablePath},
		},
		{
			name: "Should reject transform without map",
			def: singleNodeDefinition(dsl.Node{
				ID:     "transform",
				Class:  dsl.NodeClassAction,
				Kind:   string(dsl.ActionTransform),
				Params: dsl.NodeParams{},
			}),
			wantCodes: []string{refs.CodeUnresolvablePath},
		},
		{
			name: "Should reject fan out max parallel over ceiling",
			def: singleNodeDefinition(dsl.Node{
				ID:          "fan",
				Class:       dsl.NodeClassControl,
				Kind:        string(dsl.ControlFanOut),
				Collection:  "{{ .inputs.items }}",
				MaxFanOut:   1,
				MaxParallel: loop.LoopMaxFanoutWidth + 1,
			}),
			wantCodes: []string{loop.CodeFanOutCeilingExceeded},
		},
		{
			name: "Should reject empty sub loops",
			def: singleNodeDefinition(dsl.Node{
				ID:    "sub",
				Class: dsl.NodeClassControl,
				Kind:  string(dsl.ControlSubLoop),
				Body:  &dsl.Graph{},
			}),
			wantCodes: []string{loop.CodeNonTerminatingStructure},
		},
		{
			name: "Should reject invalid sub loop body structures",
			def: singleNodeDefinition(dsl.Node{
				ID:    "sub",
				Class: dsl.NodeClassControl,
				Kind:  string(dsl.ControlSubLoop),
				Body: &dsl.Graph{
					Nodes: []dsl.Node{
						{
							ID:       "nested_load",
							Class:    dsl.NodeClassSource,
							Kind:     string(dsl.SourceInput),
							InputRef: "items",
						},
						{
							ID:    "nested_agent",
							Class: dsl.NodeClassAction,
							Kind:  string(dsl.ActionRunAgent),
							Params: dsl.NodeParams{
								"agent":  "codex",
								"prompt": "Summarize {{ .nodes.nested_load.output.type }}",
								"output_schema": map[string]any{
									"summary": "string",
								},
							},
						},
					},
					Edges: []dsl.Edge{
						{From: "nested_load", To: "nested_agent"},
						{From: "nested_agent", To: "nested_load"},
					},
				},
			}),
			wantCodes: []string{loop.CodeCycle},
		},
		{
			name: "Should reject invalid file import source schema",
			def: singleNodeDefinition(dsl.Node{
				ID:    "files",
				Class: dsl.NodeClassSource,
				Kind:  string(dsl.SourceFileImport),
				Parse: "yaml",
			}),
			wantCodes: []string{refs.CodeUnresolvablePath},
		},
		{
			name: "Should accept watch source closed enum",
			def: singleNodeDefinition(dsl.Node{
				ID:        "watch",
				Class:     dsl.NodeClassSource,
				Kind:      string(dsl.SourceWatchSource),
				WatchSpec: map[string]any{"kind": "reviews"},
			}),
		},
		{
			name: "Should reject watch source without kind",
			def: singleNodeDefinition(dsl.Node{
				ID:    "watch",
				Class: dsl.NodeClassSource,
				Kind:  string(dsl.SourceWatchSource),
			}),
			wantCodes: []string{loop.CodeWatchKindRequired},
		},
		{
			name: "Should accept file import source closed enum",
			def: singleNodeDefinition(dsl.Node{
				ID:      "files",
				Class:   dsl.NodeClassSource,
				Kind:    string(dsl.SourceFileImport),
				Pattern: "docs/*.md",
				Parse:   dsl.FileParseMDTasks,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			linter := loop.NewLinter(loop.WithToolSchemaSource(fakeToolSchemas{}))
			requireLintCodes(t, linter.Lint(tt.def), tt.wantCodes...)
		})
	}
}

func validDefinition() dsl.Definition {
	return dsl.Definition{
		APIVersion:  dsl.APIVersion,
		Kind:        dsl.KindLoop,
		Concurrency: dsl.ConcurrencyForbid,
		Meta: dsl.Meta{
			Name:    "valid-loop",
			Version: 1,
		},
		Inputs: map[string]dsl.Input{
			"tasks": {Type: dsl.InputTypeRef, Required: true},
		},
		Contract: dsl.Contract{
			Goal:             "Complete task list",
			DefinitionOfDone: "All tasks have summaries",
			Verification:     []dsl.GateCriterion{},
			TerminalStates:   []dsl.TerminalState{dsl.TerminalDone, dsl.TerminalFailed},
			IterationCap:     3,
			NoProgress: dsl.NoProgress{
				Window:     2,
				HashFields: []string{"nodes.agent.output.summary"},
			},
			Budget: dsl.Budget{
				Tokens:       1000,
				WallClockSec: 60,
				OnExceeded:   dsl.BudgetExceededHalt,
			},
		},
		Graph: dsl.Graph{
			Nodes: []dsl.Node{
				{
					ID:       "load",
					Class:    dsl.NodeClassSource,
					Kind:     string(dsl.SourceInput),
					InputRef: "tasks",
					Produces: dsl.Schema{
						"items": []any{map[string]any{"title": "string"}},
					},
				},
				{
					ID:          "fan",
					Class:       dsl.NodeClassControl,
					Kind:        string(dsl.ControlFanOut),
					Collection:  "{{ .nodes.load.output.items }}",
					MaxFanOut:   4,
					MaxParallel: 2,
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
			},
			Edges: []dsl.Edge{
				{From: "load", To: "fan"},
				{From: "fan", To: "agent"},
			},
		},
		Start: []dsl.StartBinding{{Kind: dsl.StartManual}},
	}
}

func singleNodeDefinition(node dsl.Node) dsl.Definition {
	return dsl.Definition{
		APIVersion: dsl.APIVersion,
		Kind:       dsl.KindLoop,
		Meta:       dsl.Meta{Name: "single-node"},
		Inputs: map[string]dsl.Input{
			"items": {Type: dsl.InputTypeRef},
		},
		Contract: dsl.Contract{
			Goal:             "Validate",
			DefinitionOfDone: "No lint errors",
			TerminalStates:   []dsl.TerminalState{dsl.TerminalDone, dsl.TerminalFailed},
			IterationCap:     1,
			NoProgress:       dsl.NoProgress{Window: 1},
			Budget: dsl.Budget{
				Tokens:       100,
				WallClockSec: 10,
				OnExceeded:   dsl.BudgetExceededHalt,
			},
		},
		Graph: dsl.Graph{
			Nodes: []dsl.Node{node},
			Edges: []dsl.Edge{},
		},
		Start: []dsl.StartBinding{{Kind: dsl.StartManual}},
	}
}

func appendGate(def *dsl.Definition, node dsl.Node) {
	def.Graph.Nodes = append(def.Graph.Nodes, node)
	def.Graph.Edges = append(def.Graph.Edges, dsl.Edge{From: "agent", To: node.ID})
}

func requireNode(t *testing.T, def *dsl.Definition, id dsl.NodeID) *dsl.Node {
	t.Helper()
	for idx := range def.Graph.Nodes {
		if def.Graph.Nodes[idx].ID == id {
			return &def.Graph.Nodes[idx]
		}
	}
	t.Fatalf("node %q not found", id)
	return nil
}

func requireLintCodes(t *testing.T, errs []loop.LintError, wantCodes ...string) {
	t.Helper()
	if len(wantCodes) == 0 {
		if len(errs) > 0 {
			t.Fatalf("Lint() errors = %#v, want none", errs)
		}
		return
	}

	got := make([]string, 0, len(errs))
	for _, err := range errs {
		if err.Message == "" {
			t.Fatalf("Lint() emitted empty message for code %q", err.Code)
		}
		if err.Severity != loop.SeverityError {
			t.Fatalf(
				"Lint() severity for code %q = %q, want %q",
				err.Code,
				err.Severity,
				loop.SeverityError,
			)
		}
		got = append(got, err.Code)
	}
	for _, want := range wantCodes {
		if !slices.Contains(got, want) {
			t.Fatalf("Lint() codes = %#v, want code %q; errors=%#v", got, want, errs)
		}
	}
}

func requireLintWarning(t *testing.T, errs []loop.LintError, code string) {
	t.Helper()
	if len(errs) == 0 {
		t.Fatalf("Lint() errors = nil, want warning code %s", code)
	}
	for _, err := range errs {
		if err.Code == code && err.Severity == loop.SeverityWarning {
			return
		}
		if err.Severity == loop.SeverityError {
			t.Fatalf("Lint() emitted blocking error while looking for warning: %#v", errs)
		}
	}
	t.Fatalf("Lint() diagnostics = %#v, want warning code %s", errs, code)
}

type fakeToolSchemas map[string]loop.ToolSchemaSnapshot

func (s fakeToolSchemas) Snapshot(toolID string) (loop.ToolSchemaSnapshot, bool) {
	snapshot, ok := s[toolID]
	return snapshot, ok
}

func mustJSON(t *testing.T, value map[string]any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}
