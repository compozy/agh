package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/loop/dsl/refs"
	"github.com/compozy/agh/internal/task"
)

func isFileImportSourceNode(node dsl.Node) bool {
	return node.Class == dsl.NodeClassSource && node.Kind == string(dsl.SourceFileImport)
}

func evaluateFileImportNode(
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	topology controlTopology,
	output GenerationOutput,
	node dsl.Node,
	outputs []GenerationOutput,
) (GenerationOutput, *task.CoordinatorTerminal, error) {
	namespace, err := runtimeNamespace(
		run,
		generation,
		resolved.Definition.Graph,
		topology,
		outputs,
		node.ID,
		output.ItemIndex,
	)
	if err != nil {
		return GenerationOutput{}, nil, err
	}
	pattern, err := refs.RenderTemplateString(
		fmt.Sprintf("nodes.%s.pattern", node.ID),
		node.Pattern,
		namespace,
	)
	if err != nil {
		return GenerationOutput{}, nil, err
	}
	path := strings.TrimSpace(pattern)
	if path == "" {
		return GenerationOutput{}, nil, fmt.Errorf("%w: file-import pattern is required", ErrValidation)
	}
	ref, empty, err := fileImportOutputRef(node.Parse, path)
	if err != nil {
		return GenerationOutput{}, nil, err
	}
	if empty {
		output.Status = generationOutputSucceeded
		output.OutputRef = ref
		return output, &task.CoordinatorTerminal{
			Status: string(StatusNoOp),
			Cause:  string(TransitionCauseContract),
		}, nil
	}
	output.Status = generationOutputSucceeded
	output.OutputRef = ref
	return output, nil, nil
}

func fileImportOutputRef(parse dsl.FileParseKind, path string) (string, bool, error) {
	var payload any
	switch parse {
	case dsl.FileParseJSON:
		data, err := os.ReadFile(path)
		if err != nil {
			return "", false, fmt.Errorf("loop: read file-import %q: %w", path, err)
		}
		var decoded any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return "", false, fmt.Errorf("loop: decode file-import JSON: %w", err)
		}
		payload = decoded
	case dsl.FileParseText:
		data, err := os.ReadFile(path)
		if err != nil {
			return "", false, fmt.Errorf("loop: read file-import %q: %w", path, err)
		}
		payload = map[string]any{"text": string(data)}
	case dsl.FileParseMDTasks, "":
		result, err := importMarkdownTasks(path)
		if err != nil {
			return "", false, err
		}
		payload = map[string]any{"tasks": result.Tasks}
		if len(result.Tasks) == 0 {
			encoded, err := json.Marshal(payload)
			if err != nil {
				return "", false, fmt.Errorf("loop: marshal file-import output: %w", err)
			}
			return string(encoded), true, nil
		}
	default:
		return "", false, fmt.Errorf("%w: unsupported file-import parse %q", ErrValidation, parse)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", false, fmt.Errorf("loop: marshal file-import output: %w", err)
	}
	return string(encoded), false, nil
}
