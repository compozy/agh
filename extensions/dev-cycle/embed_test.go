package devcycle

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	toolspkg "github.com/compozy/agh/internal/tools"
)

func TestEmbeddedLoopsShouldCompileWithDevCycleToolSchemas(t *testing.T) {
	t.Parallel()

	source := devCycleToolSchemaSource(t)
	compiler := loop.NewCompiler(loop.WithCompilerToolSchemaSource(source))
	linter := loop.NewLinter(loop.WithToolSchemaSource(source))
	loopFiles := embeddedLoopFiles(t)
	if len(loopFiles) == 0 {
		t.Fatal("embedded loop files = 0, want at least one dev-cycle loop")
	}

	for _, path := range loopFiles {
		t.Run("Should compile "+strings.TrimSuffix(filepath.Base(filepath.Dir(path)), ".yaml"), func(t *testing.T) {
			t.Parallel()

			data, err := fs.ReadFile(FS(), path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", path, err)
			}
			spec, def, err := loop.ParseResource(data, loop.ResourceParseOptions{
				Source:   loop.SourceMarketplace,
				Dir:      filepath.ToSlash(filepath.Dir(path)),
				FilePath: filepath.ToSlash(path),
				Linter:   linter,
			})
			if err != nil {
				t.Fatalf("ParseResource(%q) error = %v", path, err)
			}
			if spec.Name == "" {
				t.Fatalf("ParseResource(%q) returned empty loop name", path)
			}
			if _, err := compiler.Compile(def); err != nil {
				t.Fatalf("Compile(%q) error = %v", path, err)
			}
		})
	}
}

func TestEmbeddedLoopsShouldKeepDevCycleRuntimeContracts(t *testing.T) {
	t.Run("Should keep reviews-watch trigger start and clean-tick stop_when", func(t *testing.T) {
		t.Parallel()

		def := parseEmbeddedLoopForTest(t, "loops/reviews-watch/loop.yaml")
		if got, want := def.Contract.StopWhen,
			"nodes.fetch_issues.status == 'succeeded' && size(nodes.fetch_issues.output.issues) == 0"; got != want {
			t.Fatalf("reviews-watch stop_when = %q, want %q", got, want)
		}
		if got, want := def.Contract.IterationCap, 0; got != want {
			t.Fatalf("reviews-watch iteration_cap = %d, want %d", got, want)
		}
		if !hasStartKind(def, dsl.StartTrigger) {
			t.Fatalf("reviews-watch start = %#v, want trigger start binding", def.Start)
		}
		newReview := requireDevCycleNode(t, def, "new_review")
		reviewSchema := requireSchemaObject(t, newReview.Produces, "review")
		properties := requireSchemaObject(t, reviewSchema, "properties")
		for _, field := range []string{"head_sha", "review_id", "submitted_at"} {
			if _, ok := properties[field]; !ok {
				t.Fatalf("new_review produces.review.properties missing %q: %#v", field, properties)
			}
		}
	})

	t.Run("Should keep software-delivery verification opt-in", func(t *testing.T) {
		t.Parallel()

		def := parseEmbeddedLoopForTest(t, "loops/software-delivery/loop.yaml")
		verifyCommand, ok := def.Inputs["verify_command"]
		if !ok {
			t.Fatal("software-delivery input verify_command missing")
		}
		if got, want := verifyCommand.Default, ""; got != want {
			t.Fatalf("verify_command default = %#v, want %q", got, want)
		}
		verify := requireDevCycleNode(t, def, "verify")
		if got, want := len(verify.Criteria), 1; got != want {
			t.Fatalf("verify criteria = %d, want %d", got, want)
		}
		if got, want := verify.Criteria[0].Type, dsl.CriterionCommand; got != want {
			t.Fatalf("verify criterion type = %q, want %q", got, want)
		}
		if got, want := verify.Criteria[0].Check, "{{ .inputs.verify_command }}"; got != want {
			t.Fatalf("verify criterion check = %q, want %q", got, want)
		}
	})
}

func TestEmbeddedAgentsShouldParseWithRuntimeSchema(t *testing.T) {
	t.Parallel()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	root, err := materializeEmbeddedExtension(homePaths)
	if err != nil {
		t.Fatalf("materializeEmbeddedExtension() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Fatalf("RemoveAll(%q) error = %v", root, err)
		}
	})
	agentFiles := embeddedAgentFiles(t, root)
	if len(agentFiles) == 0 {
		t.Fatal("embedded agent files = 0, want at least one dev-cycle agent")
	}
	for _, path := range agentFiles {
		t.Run("Should parse "+filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			t.Parallel()

			agent, err := aghconfig.LoadAgentDefFile(path)
			if err != nil {
				t.Fatalf("LoadAgentDefFile(%q) error = %v", path, err)
			}
			if strings.TrimSpace(agent.Name) == "" {
				t.Fatalf("LoadAgentDefFile(%q).Name is empty", path)
			}
		})
	}
}

type devCycleToolSchemas map[string]loop.ToolSchemaSnapshot

func (s devCycleToolSchemas) Snapshot(toolID string) (loop.ToolSchemaSnapshot, bool) {
	snapshot, ok := s[toolID]
	return snapshot, ok
}

func embeddedLoopFiles(t *testing.T) []string {
	t.Helper()
	files := []string{}
	if err := fs.WalkDir(FS(), "loops", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Base(path) != "loop.yaml" {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		t.Fatalf("WalkDir(loops) error = %v", err)
	}
	return files
}

func embeddedAgentFiles(t *testing.T, root string) []string {
	t.Helper()
	files := []string{}
	agentsRoot := filepath.Join(root, "agents")
	if err := filepath.WalkDir(agentsRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != aghconfig.AgentDefinitionFileName {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		t.Fatalf("WalkDir(%q) error = %v", agentsRoot, err)
	}
	return files
}

func parseEmbeddedLoopForTest(t *testing.T, path string) dsl.Definition {
	t.Helper()
	data, err := fs.ReadFile(FS(), path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	source := devCycleToolSchemaSource(t)
	linter := loop.NewLinter(loop.WithToolSchemaSource(source))
	_, def, err := loop.ParseResource(data, loop.ResourceParseOptions{
		Source:   loop.SourceMarketplace,
		Dir:      filepath.ToSlash(filepath.Dir(path)),
		FilePath: filepath.ToSlash(path),
		Linter:   linter,
	})
	if err != nil {
		t.Fatalf("ParseResource(%q) error = %v", path, err)
	}
	return def
}

func hasStartKind(def dsl.Definition, kind dsl.StartKind) bool {
	for _, binding := range def.Start {
		if binding.Kind == kind {
			return true
		}
	}
	return false
}

func requireDevCycleNode(t *testing.T, def dsl.Definition, id dsl.NodeID) dsl.Node {
	t.Helper()
	for _, node := range def.Graph.Nodes {
		if node.ID == id {
			return node
		}
	}
	t.Fatalf("node %q missing", id)
	return dsl.Node{}
}

func requireSchemaObject(t *testing.T, schema map[string]any, key string) map[string]any {
	t.Helper()
	raw, ok := schema[key]
	if !ok {
		t.Fatalf("schema key %q missing from %#v", key, schema)
	}
	object, ok := raw.(map[string]any)
	if ok {
		return object
	}
	typed, ok := raw.(dsl.Schema)
	if ok {
		return map[string]any(typed)
	}
	t.Fatalf("schema key %q type = %T, want map[string]any", key, raw)
	return nil
}

func devCycleToolSchemaSource(t *testing.T) devCycleToolSchemas {
	t.Helper()

	return devCycleToolSchemas{
		devCycleToolID(t, toolFetchUnresolved): toolSnapshot(
			t,
			toolFetchUnresolved,
			fetchInputSchema,
			fetchOutputSchema,
		),
		devCycleToolID(t, toolResolveThreads): toolSnapshot(
			t,
			toolResolveThreads,
			resolveInputSchema,
			resolveOutputSchema,
		),
		devCycleToolID(t, toolGitPush): toolSnapshot(
			t,
			toolGitPush,
			pushInputSchema,
			pushOutputSchema,
		),
	}
}

func devCycleToolID(t *testing.T, handler string) string {
	t.Helper()
	id, err := runtimeToolID(handler)
	if err != nil {
		t.Fatalf("runtimeToolID(%q) error = %v", handler, err)
	}
	return id.String()
}

func toolSnapshot(
	t *testing.T,
	handler string,
	inputSchema json.RawMessage,
	outputSchema json.RawMessage,
) loop.ToolSchemaSnapshot {
	t.Helper()
	id := devCycleToolID(t, handler)
	inputDigest := schemaDigest(t, handler+" input", inputSchema)
	outputDigest := schemaDigest(t, handler+" output", outputSchema)
	return loop.ToolSchemaSnapshot{
		ToolID:             id,
		InputSchema:        cloneRawMessage(inputSchema),
		OutputSchema:       cloneRawMessage(outputSchema),
		InputSchemaDigest:  inputDigest,
		OutputSchemaDigest: outputDigest,
	}
}

func schemaDigest(t *testing.T, name string, schema json.RawMessage) string {
	t.Helper()
	digest, err := toolspkg.SchemaDigest(schema)
	if err != nil {
		t.Fatalf("%s schema digest error = %v", name, err)
	}
	return digest
}
