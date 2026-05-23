package skills

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/compozy/agh/internal/filesnap"
	"github.com/compozy/agh/internal/frontmatter"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"gopkg.in/yaml.v3"
)

const (
	loaderVersionKey = "version"
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
	"name":           {},
	"description":    {},
	loaderVersionKey: {},
	"metadata":       {},
}

// ParseSkillFile reads and parses a SKILL.md file from disk.
//
// The loader fills parsed metadata plus canonical file locations. The skill
// body is intentionally not retained on the returned Skill; callers must use
// ReadSkillContent when they need the full instructions.
func ParseSkillFile(path string) (*Skill, error) {
	skill, _, err := parseSkillFileDocument(path)
	return skill, err
}

// ParseSkillFileWithSource reads and parses a skill file from disk while
// preserving the caller-selected source tier for downstream precedence and hook
// metadata handling.
func ParseSkillFileWithSource(path string, source SkillSource) (*Skill, error) {
	absPath, content, err := readSkillFile(path)
	if err != nil {
		return nil, err
	}

	skill, _, err := parseSkillDocument(absPath, filepath.Dir(absPath), content, source)
	if err != nil {
		return nil, err
	}
	if err := mergeSkillMCPSidecarFile(filepath.Dir(absPath), skill); err != nil {
		return nil, fmt.Errorf("skills: parse %q MCP JSON: %w", absPath, err)
	}
	return skill, nil
}

// ReadSkillContent reads and returns the markdown body from a SKILL.md file.
func ReadSkillContent(path string) (string, error) {
	_, body, err := parseSkillFileDocument(path)
	if err != nil {
		return "", err
	}
	return body, nil
}

func parseSkillFileDocument(path string) (*Skill, string, error) {
	absPath, content, err := readSkillFile(path)
	if err != nil {
		return nil, "", err
	}

	skill, body, err := parseSkillDocument(absPath, filepath.Dir(absPath), content, 0)
	if err != nil {
		return nil, "", err
	}
	if err := mergeSkillMCPSidecarFile(filepath.Dir(absPath), skill); err != nil {
		return nil, "", fmt.Errorf("skills: parse %q MCP JSON: %w", absPath, err)
	}

	return skill, body, nil
}

func readSkillFile(path string) (string, []byte, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", nil, fmt.Errorf("skills: resolve path %q: %w", path, err)
	}
	if err := ensurePathWithinRoot(filepath.Dir(absPath), absPath); err != nil {
		return "", nil, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("skills: read %q: %w", absPath, err)
	}

	return absPath, content, nil
}

func parseSkillDocument(filePath string, dir string, content []byte, source SkillSource) (*Skill, string, error) {
	meta, body, err := parseSkillContent(content)
	if err != nil {
		return nil, "", fmt.Errorf("skills: parse %q: %w", filePath, err)
	}
	if meta.Name == "" {
		return nil, "", fmt.Errorf("skills: parse %q: %w", filePath, errSkillNameRequired)
	}

	skill := &Skill{
		Meta:     meta,
		Source:   source,
		Dir:      dir,
		FilePath: filePath,
		Enabled:  true,
	}
	if err := parseAGHMetadata(skill); err != nil {
		return nil, "", fmt.Errorf("skills: parse %q metadata.agh: %w", filePath, err)
	}
	refreshSkillHookDecls(skill)
	if skill.Meta.Description == "" {
		slog.Warn("skills: parsed skill without description", "path", filePath, "name", skill.Meta.Name)
	}

	return skill, body, nil
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

		return appendSkillScanCandidate(absRoot, path, entry, depth, &paths, snapshots)
	})
	if walkErr != nil && !errors.Is(walkErr, errScanLimitReached) {
		return nil, nil, walkErr
	}

	slices.Sort(paths)
	return paths, snapshots, nil
}

func appendSkillScanCandidate(
	absRoot string,
	path string,
	entry fs.DirEntry,
	depth int,
	paths *[]string,
	snapshots map[string]filesnap.Snapshot,
) error {
	if depth > maxScanDepth || entry.Name() != skillFileName {
		return nil
	}
	if err := ensurePathWithinRoot(absRoot, path); err != nil {
		slog.Warn("skills: skipping skill file that escapes scan root", "path", path, "error", err)
		return nil
	}

	snapshot, err := filesnap.FromPath(path)
	if err != nil {
		slog.Warn("skills: skipping unreadable skill file during scan", "path", path, "error", err)
		return nil
	}

	*paths = append(*paths, path)
	snapshots[path] = snapshot
	if len(*paths) >= maxScanCandidates {
		slog.Warn("skills: scan candidate limit reached", "root", absRoot, "limit", maxScanCandidates)
		return errScanLimitReached
	}

	return nil
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

func parseAGHMetadata(skill *Skill) error {
	if skill == nil || skill.Meta.Metadata == nil {
		return nil
	}

	rawAGH, ok := skill.Meta.Metadata["agh"]
	if !ok || rawAGH == nil {
		return nil
	}

	agh, ok := rawAGH.(map[string]any)
	if !ok {
		warnAGHMetadata(skill, "skills: malformed metadata.agh block", "type", fmt.Sprintf("%T", rawAGH))
		return nil
	}

	if rawMCPServers, ok := agh["mcp_servers"]; ok {
		skill.MCPServers = parseMCPServerDecls(skill, rawMCPServers)
	}
	if rawHooks, ok := agh["hooks"]; ok {
		hooks, err := parseHookDecls(skill, rawHooks)
		if err != nil {
			return err
		}
		skill.Hooks = hooks
	}

	return nil
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
			warnAGHMetadata(
				skill,
				"skills: malformed metadata.agh.mcp_servers entry",
				"index",
				idx,
				"type",
				fmt.Sprintf("%T", item),
			)
			continue
		}

		server := MCPServerDecl{
			Name:      strings.TrimSpace(stringValue(entry["name"])),
			Command:   strings.TrimSpace(stringValue(entry["command"])),
			Args:      stringSliceValue(skill, "metadata.agh.mcp_servers", idx, "args", entry["args"]),
			Env:       stringMapValue(skill, "metadata.agh.mcp_servers", idx, "env", entry["env"]),
			SecretEnv: stringMapValue(skill, "metadata.agh.mcp_servers", idx, "secret_env", entry["secret_env"]),
		}
		if server.Name == "" {
			warnAGHMetadata(
				skill,
				"skills: invalid metadata.agh.mcp_servers entry",
				"index",
				idx,
				"reason",
				"missing name",
			)
			continue
		}
		if server.Command == "" {
			warnAGHMetadata(
				skill,
				"skills: invalid metadata.agh.mcp_servers entry",
				"index",
				idx,
				"reason",
				"missing command",
			)
			continue
		}

		servers = append(servers, server)
	}

	if len(servers) == 0 {
		return nil
	}

	return slices.Clip(servers)
}

type parsedSkillHookDecl struct {
	Event     string               `yaml:"event"`
	Command   string               `yaml:"command"`
	Args      []string             `yaml:"args,omitempty"`
	Timeout   time.Duration        `yaml:"timeout,omitempty"`
	Env       map[string]string    `yaml:"env,omitempty"`
	SecretEnv map[string]string    `yaml:"secret_env,omitempty"`
	Mode      hookspkg.HookMode    `yaml:"mode,omitempty"`
	Priority  *int                 `yaml:"priority,omitempty"`
	Matcher   hookspkg.HookMatcher `yaml:"matcher,omitempty"`
}

func parseHookDecls(skill *Skill, raw any) ([]hookspkg.HookDecl, error) {
	items, ok := raw.([]any)
	if !ok {
		warnAGHMetadata(skill, "skills: malformed metadata.agh.hooks field", "type", fmt.Sprintf("%T", raw))
		return nil, nil
	}

	hooks := make([]hookspkg.HookDecl, 0, len(items))
	for idx, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			warnAGHMetadata(
				skill,
				"skills: malformed metadata.agh.hooks entry",
				"index",
				idx,
				"type",
				fmt.Sprintf("%T", item),
			)
			continue
		}

		decoded, err := decodeSkillHookDecl(entry)
		if err != nil {
			return nil, fmt.Errorf(
				"skills: invalid metadata.agh.hooks entry for %q at index %d: %w",
				skillIdentifier(skill),
				idx,
				err,
			)
		}

		hook, err := buildSkillHookDecl(skill, decoded, idx, len(items))
		if err != nil {
			return nil, err
		}
		if err := hookspkg.ValidateHookDecl(hook); err != nil {
			return nil, fmt.Errorf(
				"skills: invalid metadata.agh.hooks entry for %q at index %d: %w",
				skillIdentifier(skill),
				idx,
				err,
			)
		}

		hooks = append(hooks, hook)
	}

	if len(hooks) == 0 {
		return nil, nil
	}

	return slices.Clip(hooks), nil
}

func buildSkillHookDecl(
	skill *Skill,
	decoded parsedSkillHookDecl,
	index int,
	total int,
) (hookspkg.HookDecl, error) {
	event, err := skillHookEvent(skill, strings.TrimSpace(decoded.Event), index)
	if err != nil {
		return hookspkg.HookDecl{}, err
	}

	mode := decoded.Mode
	if mode == "" {
		mode = hookspkg.HookModeAsync
	}

	hook := normalizeSkillHookDecl(skill, hookspkg.HookDecl{
		Name:        skillHookName(skill, index, total),
		Event:       event,
		Mode:        mode,
		Priority:    0,
		Timeout:     decoded.Timeout,
		Matcher:     decoded.Matcher,
		Command:     strings.TrimSpace(decoded.Command),
		Args:        append([]string(nil), decoded.Args...),
		Env:         cloneStringMap(decoded.Env),
		SecretEnv:   cloneStringMap(decoded.SecretEnv),
		PrioritySet: decoded.Priority != nil,
	}, index, total)
	if decoded.Priority != nil {
		priority, err := hookspkg.PriorityFromInt(*decoded.Priority)
		if err != nil {
			return hookspkg.HookDecl{}, err
		}
		hook.Priority = priority
	}
	if hook.Command == "" {
		return hookspkg.HookDecl{}, fmt.Errorf(
			"skills: invalid metadata.agh.hooks entry for %q at index %d: command is required",
			skillIdentifier(skill),
			index,
		)
	}

	return hook, nil
}

func skillHookEvent(skill *Skill, rawEvent string, index int) (hookspkg.HookEvent, error) {
	event := hookspkg.HookEvent(rawEvent)
	if replacement, ok := legacyHookEventReplacement(rawEvent); ok {
		return "", fmt.Errorf(
			"skills: invalid metadata.agh.hooks entry for %q at index %d: hook event %q was removed; use %q",
			skillIdentifier(skill),
			index,
			rawEvent,
			replacement,
		)
	}
	if !validHookEvent(event) {
		return "", fmt.Errorf(
			"skills: invalid metadata.agh.hooks entry for %q at index %d: unknown hook event %q",
			skillIdentifier(skill),
			index,
			rawEvent,
		)
	}
	return event, nil
}

func decodeSkillHookDecl(entry map[string]any) (parsedSkillHookDecl, error) {
	var decoded parsedSkillHookDecl

	payload, err := yaml.Marshal(entry)
	if err != nil {
		return parsedSkillHookDecl{}, fmt.Errorf("encode hook declaration: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(payload))
	decoder.KnownFields(true)
	if err := decoder.Decode(&decoded); err != nil {
		return parsedSkillHookDecl{}, fmt.Errorf("decode hook declaration: %w", err)
	}

	return decoded, nil
}

func validHookEvent(event hookspkg.HookEvent) bool {
	return event.Validate() == nil
}

func legacyHookEventReplacement(event string) (hookspkg.HookEvent, bool) {
	switch strings.TrimSpace(event) {
	case "on_session_created":
		return hookspkg.HookSessionPostCreate, true
	case "on_session_stopped":
		return hookspkg.HookSessionPostStop, true
	default:
		return "", false
	}
}

func stringValue(value any) string {
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}

	return stringValue
}

func skillIdentifier(skill *Skill) string {
	if skill == nil {
		return "unknown skill"
	}

	name := strings.TrimSpace(skill.Meta.Name)
	if name != "" {
		return name
	}

	path := strings.TrimSpace(skill.FilePath)
	if path != "" {
		return path
	}

	return "unknown skill"
}

func stringSliceValue(skill *Skill, scope string, index int, field string, raw any) []string {
	if raw == nil {
		return nil
	}

	items, ok := raw.([]any)
	if !ok {
		warnAGHMetadata(
			skill,
			"skills: malformed metadata list field",
			"scope",
			scope,
			"index",
			index,
			"field",
			field,
			"type",
			fmt.Sprintf("%T", raw),
		)
		return nil
	}

	values := make([]string, 0, len(items))
	for itemIndex, item := range items {
		value, ok := item.(string)
		if !ok {
			warnAGHMetadata(
				skill,
				"skills: malformed metadata list item",
				"scope",
				scope,
				"index",
				index,
				"field",
				field,
				"item_index",
				itemIndex,
				"type",
				fmt.Sprintf("%T", item),
			)
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
		warnAGHMetadata(
			skill,
			"skills: malformed metadata map field",
			"scope",
			scope,
			"index",
			index,
			"field",
			field,
			"type",
			fmt.Sprintf("%T", raw),
		)
		return nil
	}

	values := make(map[string]string, len(input))
	for key, rawValue := range input {
		value, ok := rawValue.(string)
		if !ok {
			warnAGHMetadata(
				skill,
				"skills: malformed metadata map entry",
				"scope",
				scope,
				"index",
				index,
				"field",
				field,
				"key",
				key,
				"type",
				fmt.Sprintf("%T", rawValue),
			)
			continue
		}

		values[key] = value
	}

	if len(values) == 0 {
		return nil
	}

	return values
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
	case ".agh":
		return false
	}

	return strings.HasPrefix(name, ".")
}
