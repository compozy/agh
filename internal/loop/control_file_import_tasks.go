package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/compozy/agh/internal/frontmatter"
	"gopkg.in/yaml.v3"
)

const (
	compozyTaskManifestFileName = "_tasks.md"
	compozyTaskManifestVersion  = "compozy.tasks/v2"
)

var taskFileNamePattern = regexp.MustCompile(`^task_(\d+)\.md$`)

type markdownTasksImportResult struct {
	Tasks []markdownTaskPayload
}

type markdownTaskPayload struct {
	ID      string   `json:"id"`
	Number  int      `json:"number"`
	Title   string   `json:"title"`
	Path    string   `json:"path"`
	Body    string   `json:"body"`
	BodyRef string   `json:"body_ref"`
	Blocks  []string `json:"blocks"`
}

type compozyTaskManifest struct {
	SchemaVersion string               `yaml:"schema_version"`
	Workflow      string               `yaml:"workflow"`
	Graph         compozyTaskGraphSpec `yaml:"graph"`
}

type compozyTaskGraphSpec struct {
	Nodes []compozyTaskGraphNode `yaml:"nodes"`
	Edges []compozyTaskGraphEdge `yaml:"edges"`
}

type compozyTaskGraphNode struct {
	ID   string `yaml:"id"`
	File string `yaml:"file"`
}

type compozyTaskGraphEdge struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type compozyTaskFile struct {
	ID     string
	Number int
	Path   string
	Meta   compozyTaskFrontmatter
	Body   string
}

type compozyTaskFrontmatter struct {
	Status       string   `yaml:"status"`
	Title        string   `yaml:"title"`
	Dependencies []string `yaml:"dependencies"`
}

func importMarkdownTasks(pattern string) (markdownTasksImportResult, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return markdownTasksImportResult{}, fmt.Errorf("%w: invalid md_tasks glob %q: %w", ErrValidation, pattern, err)
	}
	slices.Sort(matches)
	tasksDir, err := tasksDirFromPattern(pattern, matches)
	if err != nil {
		return markdownTasksImportResult{}, err
	}
	manifest, err := readCompozyTaskManifest(tasksDir)
	if err != nil {
		return markdownTasksImportResult{}, err
	}
	if err := validateCompozyTaskManifest(tasksDir, manifest, matches); err != nil {
		return markdownTasksImportResult{}, err
	}
	files, err := readCompozyTaskFiles(tasksDir, manifest.Graph.Nodes)
	if err != nil {
		return markdownTasksImportResult{}, err
	}
	ordered, err := orderCompozyTasks(manifest.Graph.Nodes, manifest.Graph.Edges, files)
	if err != nil {
		return markdownTasksImportResult{}, err
	}
	payloads := make([]markdownTaskPayload, 0, len(ordered))
	blocksByTarget := compozyTaskBlocksByTarget(manifest.Graph.Edges)
	for _, taskFile := range ordered {
		if compozyTaskStatusCompleted(taskFile.Meta.Status) {
			continue
		}
		payloads = append(payloads, markdownTaskPayload{
			ID:      taskFile.ID,
			Number:  taskFile.Number,
			Title:   taskFile.Meta.Title,
			Path:    taskFile.Path,
			Body:    taskFile.Body,
			BodyRef: OutputRefForPayload([]byte(taskFile.Body)),
			Blocks:  blocksByTarget[taskFile.ID],
		})
	}
	return markdownTasksImportResult{Tasks: payloads}, nil
}

func tasksDirFromPattern(pattern string, matches []string) (string, error) {
	cleanPattern := filepath.Clean(strings.TrimSpace(pattern))
	if len(matches) > 0 {
		return filepath.Abs(filepath.Dir(matches[0]))
	}
	if strings.ContainsAny(cleanPattern, "*?[") {
		base := cleanPattern
		for strings.ContainsAny(filepath.Base(base), "*?[") {
			next := filepath.Dir(base)
			if next == base {
				break
			}
			base = next
		}
		return filepath.Abs(base)
	}
	return filepath.Abs(filepath.Dir(cleanPattern))
}

func readCompozyTaskManifest(tasksDir string) (compozyTaskManifest, error) {
	path := filepath.Join(tasksDir, compozyTaskManifestFileName)
	content, err := os.ReadFile(path)
	if err != nil {
		return compozyTaskManifest{}, fmt.Errorf("%w: read md_tasks manifest %q: %w", ErrValidation, path, err)
	}
	parts, err := frontmatter.Split(content)
	if err != nil {
		return compozyTaskManifest{}, fmt.Errorf("%w: parse md_tasks manifest %q: %w", ErrValidation, path, err)
	}
	var manifest compozyTaskManifest
	if err := yaml.Unmarshal(parts.Metadata, &manifest); err != nil {
		return compozyTaskManifest{}, fmt.Errorf("%w: decode md_tasks manifest %q: %w", ErrValidation, path, err)
	}
	manifest.SchemaVersion = strings.TrimSpace(manifest.SchemaVersion)
	manifest.Workflow = strings.TrimSpace(manifest.Workflow)
	for idx := range manifest.Graph.Nodes {
		manifest.Graph.Nodes[idx].ID = strings.TrimSpace(manifest.Graph.Nodes[idx].ID)
		manifest.Graph.Nodes[idx].File = filepath.ToSlash(strings.TrimSpace(manifest.Graph.Nodes[idx].File))
	}
	for idx := range manifest.Graph.Edges {
		manifest.Graph.Edges[idx].From = strings.TrimSpace(manifest.Graph.Edges[idx].From)
		manifest.Graph.Edges[idx].To = strings.TrimSpace(manifest.Graph.Edges[idx].To)
	}
	return manifest, nil
}

func validateCompozyTaskManifest(
	tasksDir string,
	manifest compozyTaskManifest,
	matches []string,
) error {
	if manifest.SchemaVersion != compozyTaskManifestVersion {
		return fmt.Errorf(
			"%w: md_tasks manifest schema_version must be %q",
			ErrValidation,
			compozyTaskManifestVersion,
		)
	}
	if len(manifest.Graph.Nodes) == 0 {
		return fmt.Errorf("%w: md_tasks manifest graph.nodes must not be empty", ErrValidation)
	}
	if err := validateCompozyTaskNodes(tasksDir, manifest.Graph.Nodes, matches); err != nil {
		return err
	}
	if err := validateCompozyTaskEdges(manifest.Graph.Nodes, manifest.Graph.Edges); err != nil {
		return err
	}
	if _, err := compozyTaskTopologicalIDs(manifest.Graph.Nodes, manifest.Graph.Edges); err != nil {
		return err
	}
	return nil
}

func validateCompozyTaskNodes(tasksDir string, nodes []compozyTaskGraphNode, matches []string) error {
	seenIDs := make(map[string]struct{}, len(nodes))
	seenFiles := make(map[string]string, len(nodes))
	matchedFiles := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		rel, err := filepath.Rel(tasksDir, match)
		if err == nil && !strings.HasPrefix(rel, "..") {
			matchedFiles[filepath.ToSlash(rel)] = struct{}{}
		}
	}
	for _, node := range nodes {
		if node.ID == "" || node.File == "" {
			return fmt.Errorf("%w: md_tasks manifest nodes require id and file", ErrValidation)
		}
		if idFromTaskFile(node.File) != node.ID {
			return fmt.Errorf("%w: md_tasks node file %q must match id %q", ErrValidation, node.File, node.ID)
		}
		if _, exists := seenIDs[node.ID]; exists {
			return fmt.Errorf("%w: duplicate md_tasks node id %q", ErrValidation, node.ID)
		}
		seenIDs[node.ID] = struct{}{}
		if previous, exists := seenFiles[node.File]; exists {
			return fmt.Errorf(
				"%w: duplicate md_tasks node file %q already assigned to %q",
				ErrValidation,
				node.File,
				previous,
			)
		}
		seenFiles[node.File] = node.ID
		if len(matchedFiles) > 0 {
			if _, ok := matchedFiles[node.File]; !ok {
				return fmt.Errorf("%w: md_tasks manifest file %q is outside glob", ErrValidation, node.File)
			}
		}
	}
	return nil
}

func validateCompozyTaskEdges(nodes []compozyTaskGraphNode, edges []compozyTaskGraphEdge) error {
	nodeSet := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		nodeSet[node.ID] = struct{}{}
	}
	seenEdges := make(map[string]struct{}, len(edges))
	for _, edge := range edges {
		if edge.From == "" || edge.To == "" {
			return fmt.Errorf("%w: md_tasks graph edges require from and to", ErrValidation)
		}
		if edge.From == edge.To {
			return fmt.Errorf("%w: md_tasks self-edge %q -> %q is not allowed", ErrValidation, edge.From, edge.To)
		}
		if _, ok := nodeSet[edge.From]; !ok {
			return fmt.Errorf("%w: md_tasks edge source %q is not a graph node", ErrValidation, edge.From)
		}
		if _, ok := nodeSet[edge.To]; !ok {
			return fmt.Errorf("%w: md_tasks edge target %q is not a graph node", ErrValidation, edge.To)
		}
		key := edge.From + "\x00" + edge.To
		if _, exists := seenEdges[key]; exists {
			return fmt.Errorf("%w: duplicate md_tasks edge %q -> %q", ErrValidation, edge.From, edge.To)
		}
		seenEdges[key] = struct{}{}
	}
	return nil
}

func readCompozyTaskFiles(tasksDir string, nodes []compozyTaskGraphNode) (map[string]compozyTaskFile, error) {
	files := make(map[string]compozyTaskFile, len(nodes))
	for _, node := range nodes {
		path := filepath.Join(tasksDir, filepath.FromSlash(node.File))
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%w: read md_tasks task file %q: %w", ErrValidation, path, err)
		}
		meta, body, err := parseCompozyTaskFile(content)
		if err != nil {
			return nil, fmt.Errorf("%w: parse md_tasks task file %q: %w", ErrValidation, path, err)
		}
		if len(meta.Dependencies) > 0 || frontmatterHasKey(content, "dependencies") {
			return nil, fmt.Errorf(
				"%w: md_tasks dependencies must live in _tasks.md graph.edges for schema %s",
				ErrValidation,
				compozyTaskManifestVersion,
			)
		}
		if strings.TrimSpace(meta.Status) == "" {
			return nil, fmt.Errorf("%w: md_tasks task file %q missing status", ErrValidation, path)
		}
		files[node.ID] = compozyTaskFile{
			ID:     node.ID,
			Number: taskNumberFromTaskFile(node.File),
			Path:   filepath.ToSlash(path),
			Meta:   meta,
			Body:   body,
		}
	}
	return files, nil
}

func parseCompozyTaskFile(content []byte) (compozyTaskFrontmatter, string, error) {
	parts, err := frontmatter.Split(content)
	if err != nil {
		return compozyTaskFrontmatter{}, "", err
	}
	var meta compozyTaskFrontmatter
	if err := yaml.Unmarshal(parts.Metadata, &meta); err != nil {
		return compozyTaskFrontmatter{}, "", err
	}
	meta.Status = strings.TrimSpace(meta.Status)
	meta.Title = strings.TrimSpace(meta.Title)
	return meta, parts.Body, nil
}
