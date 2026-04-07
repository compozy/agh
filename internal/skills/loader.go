package skills

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/filesnap"
	"github.com/pedronauck/agh/internal/frontmatter"
	"gopkg.in/yaml.v3"
)

const (
	skillFileName     = "SKILL.md"
	maxScanDepth      = 4
	maxScanCandidates = 300
)

var (
	errSkillNameRequired = errors.New("skills: skill name is required")
	errScanLimitReached  = errors.New("skills: scan candidate limit reached")
)

var allowedFrontmatterFields = map[string]struct{}{
	"name":        {},
	"description": {},
	"version":     {},
	"metadata":    {},
}

// ParseSkillFile reads and parses a SKILL.md file from disk.
//
// The loader fills the parsed metadata, markdown body, and canonical file
// locations. Callers that know the registry provenance should assign Source
// after parsing.
func ParseSkillFile(path string) (*Skill, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("skills: resolve path %q: %w", path, err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("skills: read %q: %w", absPath, err)
	}

	meta, body, err := parseSkillContent(content)
	if err != nil {
		return nil, fmt.Errorf("skills: parse %q: %w", absPath, err)
	}
	if meta.Name == "" {
		return nil, fmt.Errorf("skills: parse %q: %w", absPath, errSkillNameRequired)
	}

	skill := &Skill{
		Meta:     meta,
		Content:  body,
		Dir:      filepath.Dir(absPath),
		FilePath: absPath,
		Enabled:  true,
	}
	parseAGHMetadata(skill)
	if skill.Meta.Description == "" {
		slog.Warn("skills: parsed skill without description", "path", absPath, "name", skill.Meta.Name)
	}

	return skill, nil
}

// scanDirectory returns every SKILL.md file discovered under dir.
func scanDirectory(dir string) ([]string, error) {
	paths, _, err := scanDirectoryWithSnapshots(dir)
	return paths, err
}

func scanDirectoryWithSnapshots(dir string) ([]string, map[string]filesnap.Snapshot, error) {
	root := strings.TrimSpace(dir)
	if root == "" {
		return nil, nil, errors.New("skills: scan directory root is required")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("skills: resolve scan root %q: %w", dir, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, map[string]filesnap.Snapshot{}, nil
		}
		return nil, nil, fmt.Errorf("skills: stat scan root %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("skills: scan root %q is not a directory", absRoot)
	}

	paths := make([]string, 0, maxScanCandidates)
	snapshots := make(map[string]filesnap.Snapshot, maxScanCandidates)
	walkErr := filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			slog.Warn("skills: skipping unreadable path during scan", "path", path, "error", walkErr)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		depth, err := scanDepth(absRoot, path, entry.IsDir())
		if err != nil {
			return fmt.Errorf("skills: determine scan depth for %q: %w", path, err)
		}

		if entry.IsDir() {
			if path != absRoot && shouldSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			if depth > maxScanDepth {
				return filepath.SkipDir
			}
			return nil
		}

		if depth > maxScanDepth || entry.Name() != skillFileName {
			return nil
		}

		snapshot, err := filesnap.FromPath(path)
		if err != nil {
			slog.Warn("skills: skipping unreadable skill file during scan", "path", path, "error", err)
			return nil
		}

		paths = append(paths, path)
		snapshots[path] = snapshot
		if len(paths) >= maxScanCandidates {
			slog.Warn("skills: scan candidate limit reached", "root", absRoot, "limit", maxScanCandidates)
			return errScanLimitReached
		}

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errScanLimitReached) {
		return nil, nil, walkErr
	}

	slices.Sort(paths)
	return paths, snapshots, nil
}

func decodeSkillMeta(frontmatter string) (SkillMeta, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return SkillMeta{}, err
	}

	warnUnknownFields(&document)

	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return SkillMeta{}, err
	}

	meta.Name = strings.TrimSpace(meta.Name)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.Version = strings.TrimSpace(meta.Version)

	return meta, nil
}

func parseSkillContent(content []byte) (SkillMeta, string, error) {
	parts, err := frontmatter.Split(content)
	if err != nil {
		return SkillMeta{}, "", err
	}

	meta, err := decodeSkillMeta(string(parts.Metadata))
	if err != nil {
		return SkillMeta{}, "", fmt.Errorf("decode YAML frontmatter: %w", err)
	}

	return meta, parts.Body, nil
}

func parseAGHMetadata(skill *Skill) {
	if skill == nil || skill.Meta.Metadata == nil {
		return
	}

	rawAGH, ok := skill.Meta.Metadata["agh"]
	if !ok || rawAGH == nil {
		return
	}

	agh, ok := rawAGH.(map[string]any)
	if !ok {
		warnAGHMetadata(skill, "skills: malformed metadata.agh block", "type", fmt.Sprintf("%T", rawAGH))
		return
	}

	if rawMCPServers, ok := agh["mcp_servers"]; ok {
		skill.MCPServers = parseMCPServerDecls(skill, rawMCPServers)
	}
	if rawHooks, ok := agh["hooks"]; ok {
		skill.Hooks = parseHookDecls(skill, rawHooks)
	}
}

func parseMCPServerDecls(skill *Skill, raw any) []MCPServerDecl {
	items, ok := raw.([]any)
	if !ok {
		warnAGHMetadata(skill, "skills: malformed metadata.agh.mcp_servers field", "type", fmt.Sprintf("%T", raw))
		return nil
	}

	servers := make([]MCPServerDecl, 0, len(items))
	for idx, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			warnAGHMetadata(skill, "skills: malformed metadata.agh.mcp_servers entry", "index", idx, "type", fmt.Sprintf("%T", item))
			continue
		}

		server := MCPServerDecl{
			Name:    strings.TrimSpace(stringValue(entry["name"])),
			Command: strings.TrimSpace(stringValue(entry["command"])),
			Args:    stringSliceValue(skill, "metadata.agh.mcp_servers", idx, "args", entry["args"]),
			Env:     stringMapValue(skill, "metadata.agh.mcp_servers", idx, "env", entry["env"]),
		}
		if server.Name == "" {
			warnAGHMetadata(skill, "skills: invalid metadata.agh.mcp_servers entry", "index", idx, "reason", "missing name")
			continue
		}
		if server.Command == "" {
			warnAGHMetadata(skill, "skills: invalid metadata.agh.mcp_servers entry", "index", idx, "reason", "missing command")
			continue
		}

		servers = append(servers, server)
	}

	if len(servers) == 0 {
		return nil
	}

	return slices.Clip(servers)
}

func parseHookDecls(skill *Skill, raw any) []HookDecl {
	items, ok := raw.([]any)
	if !ok {
		warnAGHMetadata(skill, "skills: malformed metadata.agh.hooks field", "type", fmt.Sprintf("%T", raw))
		return nil
	}

	hooks := make([]HookDecl, 0, len(items))
	for idx, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			warnAGHMetadata(skill, "skills: malformed metadata.agh.hooks entry", "index", idx, "type", fmt.Sprintf("%T", item))
			continue
		}

		event := HookEvent(strings.TrimSpace(stringValue(entry["event"])))
		if !validHookEvent(event) {
			warnAGHMetadata(skill, "skills: invalid metadata.agh.hooks entry", "index", idx, "reason", "unknown event", "event", event)
			continue
		}

		hook := HookDecl{
			Event:   event,
			Command: strings.TrimSpace(stringValue(entry["command"])),
			Args:    stringSliceValue(skill, "metadata.agh.hooks", idx, "args", entry["args"]),
			Env:     stringMapValue(skill, "metadata.agh.hooks", idx, "env", entry["env"]),
			Timeout: durationValue(skill, "metadata.agh.hooks", idx, "timeout", entry["timeout"]),
		}

		hooks = append(hooks, hook)
	}

	if len(hooks) == 0 {
		return nil
	}

	return slices.Clip(hooks)
}

func validHookEvent(event HookEvent) bool {
	switch event {
	case HookSessionCreated, HookSessionStopped:
		return true
	default:
		return false
	}
}

func stringValue(value any) string {
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}

	return stringValue
}

func stringSliceValue(skill *Skill, scope string, index int, field string, raw any) []string {
	if raw == nil {
		return nil
	}

	items, ok := raw.([]any)
	if !ok {
		warnAGHMetadata(skill, "skills: malformed metadata list field", "scope", scope, "index", index, "field", field, "type", fmt.Sprintf("%T", raw))
		return nil
	}

	values := make([]string, 0, len(items))
	for itemIndex, item := range items {
		value, ok := item.(string)
		if !ok {
			warnAGHMetadata(skill, "skills: malformed metadata list item", "scope", scope, "index", index, "field", field, "item_index", itemIndex, "type", fmt.Sprintf("%T", item))
			continue
		}

		values = append(values, value)
	}

	if len(values) == 0 {
		return nil
	}

	return slices.Clip(values)
}

func stringMapValue(skill *Skill, scope string, index int, field string, raw any) map[string]string {
	if raw == nil {
		return nil
	}

	input, ok := raw.(map[string]any)
	if !ok {
		warnAGHMetadata(skill, "skills: malformed metadata map field", "scope", scope, "index", index, "field", field, "type", fmt.Sprintf("%T", raw))
		return nil
	}

	values := make(map[string]string, len(input))
	for key, rawValue := range input {
		value, ok := rawValue.(string)
		if !ok {
			warnAGHMetadata(skill, "skills: malformed metadata map entry", "scope", scope, "index", index, "field", field, "key", key, "type", fmt.Sprintf("%T", rawValue))
			continue
		}

		values[key] = value
	}

	if len(values) == 0 {
		return nil
	}

	return values
}

func durationValue(skill *Skill, scope string, index int, field string, raw any) time.Duration {
	if raw == nil {
		return 0
	}

	value, ok := raw.(string)
	if !ok {
		warnAGHMetadata(skill, "skills: malformed metadata duration field", "scope", scope, "index", index, "field", field, "type", fmt.Sprintf("%T", raw))
		return 0
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		warnAGHMetadata(skill, "skills: invalid metadata duration value", "scope", scope, "index", index, "field", field, "value", value, "error", err)
		return 0
	}

	return parsed
}

func warnAGHMetadata(skill *Skill, message string, args ...any) {
	attrs := make([]any, 0, len(args)+4)
	if skill != nil {
		if skill.FilePath != "" {
			attrs = append(attrs, "path", skill.FilePath)
		}
		if skill.Meta.Name != "" {
			attrs = append(attrs, "name", skill.Meta.Name)
		}
	}
	attrs = append(attrs, args...)

	slog.Warn(message, attrs...)
}

func warnUnknownFields(document *yaml.Node) {
	if document == nil || len(document.Content) == 0 {
		return
	}

	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := strings.TrimSpace(root.Content[i].Value)
		if _, ok := allowedFrontmatterFields[key]; ok {
			continue
		}

		slog.Warn("skills: unknown frontmatter field", "field", key)
	}
}

func scanDepth(root, current string, isDir bool) (int, error) {
	rel, err := filepath.Rel(root, current)
	if err != nil {
		return 0, err
	}
	if rel == "." {
		return 0, nil
	}

	parts := strings.Split(rel, string(filepath.Separator))
	if !isDir {
		return max(len(parts)-1, 0), nil
	}

	return len(parts), nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules":
		return true
	case ".agh", ".agents":
		return false
	}

	return strings.HasPrefix(name, ".")
}
