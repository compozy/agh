package config

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tomltree "github.com/pelletier/go-toml"
	tomlast "github.com/pelletier/go-toml/v2/unstable"
)

var (
	// ErrUnsupportedTOMLMutation reports a mutation that would require rewriting
	// unrelated TOML structure instead of editing the targeted document fragment.
	ErrUnsupportedTOMLMutation = errors.New("config: unsupported TOML mutation")
)

// WriteScope identifies the config scope a write should target.
type WriteScope string

const (
	// WriteScopeGlobal targets the global AGH home config.
	WriteScopeGlobal WriteScope = "global"
	// WriteScopeWorkspace targets a workspace-local AGH overlay.
	WriteScopeWorkspace WriteScope = "workspace"
)

// Validate ensures the write scope is supported.
func (s WriteScope) Validate() error {
	switch s {
	case WriteScopeGlobal, WriteScopeWorkspace:
		return nil
	default:
		return fmt.Errorf("config: invalid write scope %q", s)
	}
}

// WriteTargetKind describes the canonical persistence destination without
// exposing filesystem paths to higher layers.
type WriteTargetKind string

const (
	// WriteTargetGlobalConfig writes `~/.agh/config.toml`.
	WriteTargetGlobalConfig WriteTargetKind = "global-config"
	// WriteTargetWorkspaceConfig writes `<workspace>/.agh/config.toml`.
	WriteTargetWorkspaceConfig WriteTargetKind = "workspace-config"
	// WriteTargetGlobalMCPSidecar writes `~/.agh/mcp.json`.
	WriteTargetGlobalMCPSidecar WriteTargetKind = "global-mcp-sidecar"
	// WriteTargetWorkspaceMCPSidecar writes `<workspace>/.agh/mcp.json`.
	WriteTargetWorkspaceMCPSidecar WriteTargetKind = "workspace-mcp-sidecar"
)

// WriteTarget captures a semantic destination while keeping the on-disk path
// internal to the config package.
type WriteTarget struct {
	kind          WriteTargetKind
	scope         WriteScope
	path          string
	workspaceRoot string
}

// Kind returns the semantic destination identifier for the write target.
func (t WriteTarget) Kind() WriteTargetKind {
	return t.kind
}

// Scope returns the write scope for the target.
func (t WriteTarget) Scope() WriteScope {
	return t.scope
}

// Path returns the resolved filesystem path for operator-facing diagnostics and tools.
func (t WriteTarget) Path() string {
	return t.path
}

func (t WriteTarget) isConfigTarget() bool {
	return t.kind == WriteTargetGlobalConfig || t.kind == WriteTargetWorkspaceConfig
}

func (t WriteTarget) isMCPSidecarTarget() bool {
	return t.kind == WriteTargetGlobalMCPSidecar || t.kind == WriteTargetWorkspaceMCPSidecar
}

// ResolveConfigWriteTarget resolves the canonical config overlay destination for
// the requested scope.
func ResolveConfigWriteTarget(homePaths HomePaths, workspaceRoot string, scope WriteScope) (WriteTarget, error) {
	return resolveWriteTarget(homePaths, workspaceRoot, scope, false)
}

// ResolveMCPSidecarWriteTarget resolves the canonical MCP sidecar destination
// for the requested scope.
func ResolveMCPSidecarWriteTarget(homePaths HomePaths, workspaceRoot string, scope WriteScope) (WriteTarget, error) {
	return resolveWriteTarget(homePaths, workspaceRoot, scope, true)
}

func resolveWriteTarget(
	homePaths HomePaths,
	workspaceRoot string,
	scope WriteScope,
	sidecar bool,
) (WriteTarget, error) {
	if err := scope.Validate(); err != nil {
		return WriteTarget{}, err
	}

	switch scope {
	case WriteScopeGlobal:
		if sidecar {
			return WriteTarget{
				kind:  WriteTargetGlobalMCPSidecar,
				scope: scope,
				path:  globalMCPJSONFile(homePaths),
			}, nil
		}
		return WriteTarget{
			kind:  WriteTargetGlobalConfig,
			scope: scope,
			path:  homePaths.ConfigFile,
		}, nil
	case WriteScopeWorkspace:
		resolvedRoot, err := resolveWorkspaceRoot(workspaceRoot)
		if err != nil {
			return WriteTarget{}, err
		}
		if strings.TrimSpace(resolvedRoot) == "" {
			return WriteTarget{}, errors.New("config: workspace write target requires a workspace root")
		}
		if sidecar {
			return WriteTarget{
				kind:          WriteTargetWorkspaceMCPSidecar,
				scope:         scope,
				path:          workspaceMCPJSONFile(resolvedRoot),
				workspaceRoot: resolvedRoot,
			}, nil
		}
		return WriteTarget{
			kind:          WriteTargetWorkspaceConfig,
			scope:         scope,
			path:          workspaceConfigFile(resolvedRoot),
			workspaceRoot: resolvedRoot,
		}, nil
	default:
		return WriteTarget{}, fmt.Errorf("config: invalid write scope %q", scope)
	}
}

// OverlayEditor applies safe, comment-preserving mutations to one TOML overlay
// document.
type OverlayEditor struct {
	content []byte
	source  string
}

// SetValue updates or creates one scalar or array value at the provided path.
func (e *OverlayEditor) SetValue(path []string, value any) error {
	cleanPath, err := normalizeMutationPath(path)
	if err != nil {
		return err
	}

	normalized, err := normalizeTOMLValue(value)
	if err != nil {
		return fmt.Errorf("config: set TOML value %q: %w", strings.Join(cleanPath, "."), err)
	}

	updated, err := setValueInOverlayDocument(e.content, cleanPath, normalized)
	if err != nil {
		return err
	}
	e.content = updated
	return nil
}

// SetTable replaces or creates a TOML table at the provided path.
func (e *OverlayEditor) SetTable(path []string, values map[string]any) error {
	cleanPath, err := normalizeMutationPath(path)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return unsupportedTOMLMutation(cleanPath, "table replacement requires at least one key")
	}

	updated, err := setTableInOverlayDocument(e.content, cleanPath, values)
	if err != nil {
		return fmt.Errorf("config: set TOML table %q: %w", strings.Join(cleanPath, "."), err)
	}
	e.content = updated
	return nil
}

// UpsertArrayTableItem replaces or appends one named entry in an array-of-tables.
func (e *OverlayEditor) UpsertArrayTableItem(
	path []string,
	nameField string,
	name string,
	values map[string]any,
) error {
	cleanPath, err := normalizeMutationPath(path)
	if err != nil {
		return err
	}
	field := strings.TrimSpace(nameField)
	if field == "" {
		return errors.New("config: array-table name field is required")
	}
	key := strings.TrimSpace(name)
	if key == "" {
		return errors.New("config: array-table item name is required")
	}

	itemValues := cloneStringAnyMap(values)
	itemValues[field] = key

	updated, err := upsertArrayTableItemInOverlayDocument(e.content, cleanPath, field, key, itemValues)
	if err != nil {
		return fmt.Errorf("config: set TOML array-table item %q: %w", strings.Join(cleanPath, "."), err)
	}
	e.content = updated
	return nil
}

// Delete removes one TOML key path when present.
func (e *OverlayEditor) Delete(path []string) error {
	cleanPath, err := normalizeMutationPath(path)
	if err != nil {
		return err
	}
	updated, _, err := deletePathInOverlayDocument(e.content, cleanPath)
	if err != nil {
		return err
	}
	e.content = updated
	return nil
}

// DeleteArrayTableItem removes one named entry from an array-of-tables.
func (e *OverlayEditor) DeleteArrayTableItem(path []string, nameField string, name string) (bool, error) {
	cleanPath, err := normalizeMutationPath(path)
	if err != nil {
		return false, err
	}
	field := strings.TrimSpace(nameField)
	if field == "" {
		return false, errors.New("config: array-table name field is required")
	}
	key := strings.TrimSpace(name)
	if key == "" {
		return false, errors.New("config: array-table item name is required")
	}

	updated, deleted, err := deleteArrayTableItemInOverlayDocument(e.content, cleanPath, field, key)
	if err != nil {
		return false, err
	}
	e.content = updated
	return deleted, nil
}

// HasPath reports whether the current document already contains the given path.
func (e *OverlayEditor) HasPath(path []string) bool {
	cleanPath, err := normalizeMutationPath(path)
	if err != nil {
		return false
	}
	document, err := parseOverlayDocument(e.content)
	if err != nil {
		return false
	}
	return document.findKeyValue(cleanPath) != nil ||
		document.findTable(cleanPath) != nil ||
		len(document.arrayTableBlocks(cleanPath)) > 0
}

func (e *OverlayEditor) Bytes() ([]byte, error) {
	return append([]byte(nil), e.content...), nil
}

func newOverlayEditor(path string, contents []byte) (*OverlayEditor, error) {
	source := strings.TrimSpace(path)
	if source == "" {
		source = ConfigName
	}
	if _, err := parseOverlayDocument(contents); err != nil {
		return nil, fmt.Errorf("config: parse config overlay %q: %w", source, err)
	}
	return &OverlayEditor{content: append([]byte(nil), contents...), source: source}, nil
}

// EditConfigOverlay applies one validated mutation to a canonical TOML overlay
// target and returns the merged effective config after the write.
func EditConfigOverlay(
	homePaths HomePaths,
	workspaceRoot string,
	target WriteTarget,
	mutate func(*OverlayEditor) error,
) (Config, error) {
	if !target.isConfigTarget() {
		return Config{}, fmt.Errorf("config: write target %q is not a config overlay", target.Kind())
	}
	if mutate == nil {
		return Config{}, errors.New("config: config overlay mutation is required")
	}

	contents, err := readOptionalFile(target.path)
	if err != nil {
		return Config{}, err
	}

	editor, err := newOverlayEditor(target.path, contents)
	if err != nil {
		return Config{}, err
	}
	if err := mutate(editor); err != nil {
		return Config{}, err
	}

	rendered, err := editor.Bytes()
	if err != nil {
		return Config{}, err
	}

	finalCfg, err := validateEffectiveConfigWrite(homePaths, workspaceRoot, target, rendered)
	if err != nil {
		return Config{}, err
	}
	if err := writePersistedFile(target.path, rendered); err != nil {
		return Config{}, err
	}
	return finalCfg, nil
}

func validateEffectiveConfigWrite(
	homePaths HomePaths,
	workspaceRoot string,
	target WriteTarget,
	rendered []byte,
) (Config, error) {
	dotenvLookup, err := loadDotEnvLookup(workspaceRoot)
	if err != nil {
		return Config{}, err
	}
	lookup := layeredEnvLookup(processEnvLookup, dotenvLookup)

	cfg := DefaultWithHome(homePaths)

	globalOverlay, err := loadConfigOverlayForWrite(homePaths.ConfigFile, target, rendered)
	if err != nil {
		return Config{}, err
	}
	if err := globalOverlay.Apply(&cfg); err != nil {
		return Config{}, fmt.Errorf("apply global config overlay: %w", err)
	}
	if err := applyConfigMCPSidecarContent(globalMCPJSONFile(homePaths), target, rendered, &cfg); err != nil {
		return Config{}, fmt.Errorf("load global MCP JSON: %w", err)
	}

	resolvedWorkspaceRoot, err := resolveWorkspaceRoot(workspaceRoot)
	if err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(resolvedWorkspaceRoot) != "" {
		workspaceOverlay, err := loadConfigOverlayForWrite(workspaceConfigFile(resolvedWorkspaceRoot), target, rendered)
		if err != nil {
			return Config{}, err
		}
		if err := workspaceOverlay.Apply(&cfg); err != nil {
			return Config{}, fmt.Errorf("apply workspace config overlay: %w", err)
		}
		if err := applyConfigMCPSidecarContent(
			workspaceMCPJSONFile(resolvedWorkspaceRoot),
			target,
			rendered,
			&cfg,
		); err != nil {
			return Config{}, fmt.Errorf("load workspace MCP JSON: %w", err)
		}
	}

	if err := normalizeConfigPaths(&cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.validateWithEnv(lookup); err != nil {
		return Config{}, fmt.Errorf("validate config write for %q: %w", target.Kind(), err)
	}
	return cfg, nil
}

func loadConfigOverlayForWrite(path string, target WriteTarget, rendered []byte) (configOverlay, error) {
	if target.isConfigTarget() && samePath(target.path, path) {
		return loadConfigOverlayBytes(rendered, path)
	}
	return loadConfigOverlayFile(path)
}

func applyConfigMCPSidecarContent(path string, target WriteTarget, rendered []byte, cfg *Config) error {
	if cfg == nil {
		return errors.New("config: config is required")
	}

	var (
		servers []MCPServer
		err     error
	)
	if target.isMCPSidecarTarget() && samePath(target.path, path) {
		servers, err = parseOptionalMCPServersJSON(rendered, path)
	} else {
		servers, err = LoadMCPServersJSONFile(path)
	}
	if err != nil {
		return err
	}
	if len(servers) == 0 {
		return nil
	}

	cfg.MCPServers = OverrideMCPServers(cfg.MCPServers, servers)
	return nil
}

func parseOptionalMCPServersJSON(content []byte, source string) ([]MCPServer, error) {
	if strings.TrimSpace(string(content)) == "" {
		return nil, nil
	}
	return ParseMCPServersJSON(content, source)
}

func newEmptyTree() (*tomltree.Tree, error) {
	tree, err := tomltree.TreeFromMap(map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("config: create empty TOML tree: %w", err)
	}
	return tree, nil
}

func newTreeFromMap(values map[string]any) (*tomltree.Tree, error) {
	normalized, err := normalizeTreeValue(values)
	if err != nil {
		return nil, err
	}

	tree, err := tomltree.TreeFromMap(normalized)
	if err != nil {
		return nil, fmt.Errorf("config: create TOML tree: %w", err)
	}
	return tree, nil
}

func normalizeTreeValue(values map[string]any) (map[string]any, error) {
	normalized := make(map[string]any, len(values))
	for _, key := range sortedStringKeys(values) {
		value, err := normalizeTreeEntry(key, values[key])
		if err != nil {
			return nil, err
		}
		normalized[key] = value
	}
	return normalized, nil
}

func normalizeTreeEntry(key string, value any) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		return normalizeTreeValue(typed)
	case map[string]string:
		return normalizeTreeValue(stringMapToAny(typed))
	case []map[string]any:
		items := make([]map[string]any, 0, len(typed))
		for idx := range typed {
			normalized, err := normalizeTreeValue(typed[idx])
			if err != nil {
				return nil, fmt.Errorf("key %q array-table item %d: %w", key, idx, err)
			}
			items = append(items, normalized)
		}
		return items, nil
	case []map[string]string:
		items := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, stringMapToAny(item))
		}
		return normalizeTreeEntry(key, items)
	default:
		normalized, err := normalizeTOMLValue(typed)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", key, err)
		}
		return normalized, nil
	}
}

func normalizeMutationPath(path []string) ([]string, error) {
	if len(path) == 0 {
		return nil, errors.New("config: TOML mutation path is required")
	}

	clean := make([]string, 0, len(path))
	for _, element := range path {
		trimmed := strings.TrimSpace(element)
		if trimmed == "" {
			return nil, errors.New("config: TOML mutation path contains an empty segment")
		}
		clean = append(clean, trimmed)
	}
	return clean, nil
}

func normalizeTOMLValue(value any) (any, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case bool:
		return typed, nil
	case nil:
		return nil, errors.New("nil TOML values are not supported")
	case map[string]any, map[string]string, []map[string]any, []map[string]string:
		return nil, errors.New("tables and array-of-tables must use table helpers")
	default:
		if normalized, ok := normalizeIntegerValue(typed); ok {
			return normalized, nil
		}
		if normalized, ok := normalizeUnsignedValue(typed); ok {
			return normalized, nil
		}
		if normalized, ok := normalizeFloatValue(typed); ok {
			return normalized, nil
		}
		if normalized, ok, err := normalizeSliceValue(typed); ok || err != nil {
			return normalized, err
		}
		return nil, fmt.Errorf("unsupported TOML value type %T", value)
	}
}

func normalizeIntegerValue(value any) (any, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	default:
		return nil, false
	}
}

func normalizeUnsignedValue(value any) (any, bool) {
	switch typed := value.(type) {
	case uint:
		return uint64(typed), true
	case uint8:
		return uint64(typed), true
	case uint16:
		return uint64(typed), true
	case uint32:
		return uint64(typed), true
	case uint64:
		return typed, true
	default:
		return nil, false
	}
}

func normalizeFloatValue(value any) (any, bool) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return nil, false
	}
}

func normalizeSliceValue(value any) (any, bool, error) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true, nil
	case []bool:
		return append([]bool(nil), typed...), true, nil
	case []int:
		values := make([]int64, 0, len(typed))
		for _, item := range typed {
			values = append(values, int64(item))
		}
		return values, true, nil
	case []int64:
		return append([]int64(nil), typed...), true, nil
	case []uint64:
		return append([]uint64(nil), typed...), true, nil
	case []float64:
		return append([]float64(nil), typed...), true, nil
	case []any:
		values := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized, err := normalizeTOMLValue(item)
			if err != nil {
				return nil, true, err
			}
			switch normalized.(type) {
			case map[string]any, []map[string]any, []map[string]string:
				return nil, true, errors.New("inline tables and array-of-tables must use table helpers")
			}
			values = append(values, normalized)
		}
		return values, true, nil
	default:
		return nil, false, nil
	}
}

type overlayDocument struct {
	source      []byte
	expressions []overlayExpression
}

type overlayExpression struct {
	kind          tomlast.Kind
	path          []string
	containerPath []string
	raw           tomlast.Range
	value         tomlast.Range
}

type overlayBlock struct {
	path     []string
	startIdx int
	endIdx   int
	start    int
	end      int
}

func setValueInOverlayDocument(source []byte, path []string, value any) ([]byte, error) {
	document, err := parseOverlayDocument(source)
	if err != nil {
		return nil, err
	}

	if existing := document.findKeyValue(path); existing != nil {
		if isUsableValueRange(existing.raw, existing.value) {
			rendered, err := renderBareValue(value)
			if err != nil {
				return nil, err
			}
			return replaceRange(source, existing.value, []byte(rendered)), nil
		}

		relativePath := path
		if len(existing.containerPath) > 0 && len(path) >= len(existing.containerPath) {
			relativePath = path[len(existing.containerPath):]
		}
		line, err := renderKeyValueLine(relativePath, value)
		if err != nil {
			return nil, err
		}
		start, end := lineReplacementOffsets(source, existing.raw)
		return replaceOffsets(source, start, end, []byte(line)), nil
	}

	tablePath := path[:len(path)-1]
	if err := document.ensureNoScalarPrefix(tablePath); err != nil {
		return nil, err
	}
	if document.findTable(path) != nil || len(document.arrayTableBlocks(path)) > 0 {
		return nil, unsupportedTOMLMutation(path, "value path collides with a table")
	}
	if len(document.arrayTableBlocks(tablePath)) > 0 {
		return nil, unsupportedTOMLMutation(path, "array-of-tables require item-level edits")
	}

	if document.findTable(tablePath) != nil {
		insertAt, err := document.tableInsertOffset(tablePath)
		if err != nil {
			return nil, err
		}
		line, err := renderKeyValueLine([]string{path[len(path)-1]}, value)
		if err != nil {
			return nil, err
		}
		return insertAtOffset(source, insertAt, line), nil
	}

	fragment, err := renderTableFragment(tablePath, map[string]any{path[len(path)-1]: value})
	if err != nil {
		return nil, err
	}
	return appendFragment(source, fragment), nil
}

func setTableInOverlayDocument(source []byte, path []string, values map[string]any) ([]byte, error) {
	document, err := parseOverlayDocument(source)
	if err != nil {
		return nil, err
	}
	if err := document.ensureNoScalarPrefix(path); err != nil {
		return nil, err
	}
	if document.findKeyValue(path) != nil || len(document.arrayTableBlocks(path)) > 0 {
		return nil, unsupportedTOMLMutation(path, "table path collides with a scalar or array-of-tables")
	}

	fragment, err := renderTableFragment(path, values)
	if err != nil {
		return nil, err
	}

	block, ok, err := document.tableBlock(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return appendFragment(source, fragment), nil
	}
	return replaceOffsets(source, block.start, block.end, []byte(fragment)), nil
}

func upsertArrayTableItemInOverlayDocument(
	source []byte,
	path []string,
	nameField string,
	name string,
	values map[string]any,
) ([]byte, error) {
	document, err := parseOverlayDocument(source)
	if err != nil {
		return nil, err
	}
	if err := document.ensureNoScalarPrefix(path); err != nil {
		return nil, err
	}
	if document.findKeyValue(path) != nil || document.findTable(path) != nil {
		return nil, unsupportedTOMLMutation(path, "array-of-tables path collides with a scalar or table")
	}

	fragment, err := renderArrayTableFragment(path, values)
	if err != nil {
		return nil, err
	}

	blocks := document.arrayTableBlocks(path)
	for _, block := range blocks {
		currentName, ok := document.blockStringField(block, nameField)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(currentName), name) {
			return replaceOffsets(source, block.start, block.end, []byte(fragment)), nil
		}
	}

	if len(blocks) > 0 {
		last := blocks[len(blocks)-1]
		return insertAtOffset(source, last.end, fragment), nil
	}
	return appendFragment(source, fragment), nil
}

func deletePathInOverlayDocument(source []byte, path []string) ([]byte, bool, error) {
	document, err := parseOverlayDocument(source)
	if err != nil {
		return nil, false, err
	}

	if existing := document.findKeyValue(path); existing != nil {
		return replaceOffsets(source, rangeStart(existing.raw), rangeEnd(existing.raw), nil), true, nil
	}
	block, ok, err := document.tableBlock(path)
	if err != nil {
		return nil, false, err
	}
	if ok {
		return replaceOffsets(source, block.start, block.end, nil), true, nil
	}

	return append([]byte(nil), source...), false, nil
}

func deleteArrayTableItemInOverlayDocument(
	source []byte,
	path []string,
	nameField string,
	name string,
) ([]byte, bool, error) {
	document, err := parseOverlayDocument(source)
	if err != nil {
		return nil, false, err
	}
	blocks := document.arrayTableBlocks(path)
	for _, block := range blocks {
		currentName, ok := document.blockStringField(block, nameField)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(currentName), name) {
			return replaceOffsets(source, block.start, block.end, nil), true, nil
		}
	}
	return append([]byte(nil), source...), false, nil
}

func parseOverlayDocument(source []byte) (*overlayDocument, error) {
	document := &overlayDocument{
		source:      append([]byte(nil), source...),
		expressions: make([]overlayExpression, 0),
	}
	if len(bytes.TrimSpace(source)) == 0 {
		return document, nil
	}

	parser := tomlast.Parser{KeepComments: true}
	parser.Reset(source)

	currentTable := []string{}
	for parser.NextExpression() {
		node := parser.Expression()
		expr := overlayExpression{
			kind:          node.Kind,
			raw:           expressionRange(node),
			containerPath: clonePath(currentTable),
		}

		switch node.Kind {
		case tomlast.Table, tomlast.ArrayTable:
			path, err := nodePath(node)
			if err != nil {
				return nil, err
			}
			expr.path = path
			expr.containerPath = clonePath(path)
			expr.raw = tableHeaderRange(source, node)
			currentTable = clonePath(path)
		case tomlast.KeyValue:
			path, err := nodePath(node)
			if err != nil {
				return nil, err
			}
			expr.path = append(clonePath(currentTable), path...)
			expr.value = node.Value().Raw
		case tomlast.Comment:
			// Comments remain scoped to the current table.
		default:
			continue
		}

		document.expressions = append(document.expressions, expr)
	}

	if err := parser.Error(); err != nil {
		return nil, err
	}
	return document, nil
}

func tableHeaderRange(source []byte, node *tomlast.Node) tomlast.Range {
	if len(source) == 0 {
		return tomlast.Range{}
	}

	start := rangeStart(node.Raw)
	keyIter := node.Key()
	if keyIter.Next() {
		start = rangeStart(keyIter.Node().Raw)
	}
	if start < 0 {
		start = 0
	}
	if start > len(source) {
		start = len(source)
	}

	lineStart := 0
	if start > 0 {
		lineStart = bytes.LastIndexByte(source[:start], '\n') + 1
	}
	if idx := bytes.IndexByte(source[lineStart:], '['); idx >= 0 {
		start = lineStart + idx
	} else {
		start = lineStart
	}

	end := start
	for end < len(source) && source[end] != '\n' {
		end++
	}
	if end < len(source) {
		end++
	}

	return tomlast.Range{Offset: uint32(start), Length: uint32(end - start)}
}

func nodePath(node *tomlast.Node) ([]string, error) {
	keys := make([]string, 0)
	it := node.Key()
	for it.Next() {
		key := strings.TrimSpace(string(it.Node().Data))
		if key == "" {
			return nil, errors.New("config: TOML key path contains an empty segment")
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func expressionRange(node *tomlast.Node) tomlast.Range {
	minStart := -1
	maxEnd := 0
	var walk func(*tomlast.Node)
	walk = func(current *tomlast.Node) {
		if !current.Valid() {
			return
		}
		if current.Raw.Length > 0 {
			start := rangeStart(current.Raw)
			end := rangeEnd(current.Raw)
			if minStart == -1 || start < minStart {
				minStart = start
			}
			if end > maxEnd {
				maxEnd = end
			}
		}
		children := current.Children()
		for children.Next() {
			walk(children.Node())
		}
	}
	walk(node)
	if minStart == -1 {
		return node.Raw
	}
	length := maxEnd - minStart
	if minStart < 0 || length < 0 || uint64(minStart) > math.MaxUint32 || uint64(length) > math.MaxUint32 {
		return node.Raw
	}
	return tomlast.Range{Offset: uint32(minStart), Length: uint32(length)}
}

func (d *overlayDocument) findKeyValue(path []string) *overlayExpression {
	for idx := range d.expressions {
		expr := &d.expressions[idx]
		if expr.kind == tomlast.KeyValue && pathsEqual(expr.path, path) {
			return expr
		}
	}
	return nil
}

func (d *overlayDocument) findTable(path []string) *overlayExpression {
	for idx := range d.expressions {
		expr := &d.expressions[idx]
		if expr.kind == tomlast.Table && pathsEqual(expr.path, path) {
			return expr
		}
	}
	return nil
}

func (d *overlayDocument) tableInsertOffset(path []string) (int, error) {
	for idx := range d.expressions {
		expr := d.expressions[idx]
		if expr.kind != tomlast.Table || !pathsEqual(expr.path, path) {
			continue
		}

		offset := rangeEnd(expr.raw)
		for nextIdx := idx + 1; nextIdx < len(d.expressions); nextIdx++ {
			next := d.expressions[nextIdx]
			switch next.kind {
			case tomlast.KeyValue, tomlast.Comment:
				if pathsEqual(next.containerPath, path) {
					offset = lineEndOffset(d.source, next.raw)
					continue
				}
				return offset, nil
			case tomlast.Table, tomlast.ArrayTable:
				return offset, nil
			}
		}
		return offset, nil
	}

	return 0, unsupportedTOMLMutation(path, "table does not exist")
}

func (d *overlayDocument) tableBlock(path []string) (overlayBlock, bool, error) {
	for idx := range d.expressions {
		expr := d.expressions[idx]
		if expr.kind != tomlast.Table || !pathsEqual(expr.path, path) {
			continue
		}

		block := overlayBlock{
			path:     clonePath(path),
			startIdx: idx,
			endIdx:   idx,
			start:    rangeStart(expr.raw),
			end:      rangeEnd(expr.raw),
		}

		for nextIdx := idx + 1; nextIdx < len(d.expressions); nextIdx++ {
			next := d.expressions[nextIdx]
			switch next.kind {
			case tomlast.KeyValue, tomlast.Comment:
				if pathsEqual(next.containerPath, path) {
					block.endIdx = nextIdx
					block.end = lineEndOffset(d.source, next.raw)
					continue
				}
				return block, true, nil
			case tomlast.Table, tomlast.ArrayTable:
				if pathHasPrefix(next.path, path) && len(next.path) > len(path) {
					return overlayBlock{}, false, unsupportedTOMLMutation(
						path,
						"nested subtables are not supported for table replacement",
					)
				}
				return block, true, nil
			}
		}
		return block, true, nil
	}

	return overlayBlock{}, false, nil
}

func (d *overlayDocument) arrayTableBlocks(path []string) []overlayBlock {
	blocks := make([]overlayBlock, 0)
	for idx := 0; idx < len(d.expressions); idx++ {
		expr := d.expressions[idx]
		if expr.kind != tomlast.ArrayTable || !pathsEqual(expr.path, path) {
			continue
		}

		block := overlayBlock{
			path:     clonePath(path),
			startIdx: idx,
			endIdx:   idx,
			start:    rangeStart(expr.raw),
			end:      rangeEnd(expr.raw),
		}

		for nextIdx := idx + 1; nextIdx < len(d.expressions); nextIdx++ {
			next := d.expressions[nextIdx]
			if next.kind == tomlast.ArrayTable && pathsEqual(next.path, path) {
				break
			}
			if next.kind == tomlast.Table || next.kind == tomlast.ArrayTable {
				if pathHasPrefix(next.path, path) && len(next.path) > len(path) {
					block.endIdx = nextIdx
					block.end = rangeEnd(next.raw)
					continue
				}
				break
			}
			block.endIdx = nextIdx
			block.end = lineEndOffset(d.source, next.raw)
		}

		blocks = append(blocks, block)
		idx = block.endIdx
	}
	return blocks
}

func (d *overlayDocument) blockStringField(block overlayBlock, field string) (string, bool) {
	targetPath := append(clonePath(block.path), field)
	for idx := block.startIdx; idx <= block.endIdx && idx < len(d.expressions); idx++ {
		expr := d.expressions[idx]
		if expr.kind != tomlast.KeyValue || !pathsEqual(expr.path, targetPath) {
			continue
		}
		return decodeStringValue(d.source[rangeStart(expr.value):rangeEnd(expr.value)])
	}
	return "", false
}

func (d *overlayDocument) ensureNoScalarPrefix(path []string) error {
	for _, expr := range d.expressions {
		if expr.kind != tomlast.KeyValue {
			continue
		}
		if pathHasPrefix(path, expr.path) && len(expr.path) <= len(path) {
			return unsupportedTOMLMutation(path, "cannot descend through a scalar value")
		}
	}
	return nil
}

func renderBareValue(value any) (string, error) {
	tree, err := newEmptyTree()
	if err != nil {
		return "", err
	}
	normalized, err := normalizeTOMLValue(value)
	if err != nil {
		return "", err
	}
	tree.SetPath([]string{"value"}, normalized)
	rendered, err := tree.ToTomlString()
	if err != nil {
		return "", fmt.Errorf("config: encode TOML value: %w", err)
	}
	_, after, ok := strings.Cut(rendered, "=")
	if !ok {
		return "", fmt.Errorf("config: encode TOML value: malformed fragment %q", rendered)
	}
	return strings.TrimSpace(after), nil
}

func isUsableValueRange(raw tomlast.Range, value tomlast.Range) bool {
	if value.Length == 0 {
		return false
	}
	rawStart := rangeStart(raw)
	rawEnd := rangeEnd(raw)
	valueStart := rangeStart(value)
	valueEnd := rangeEnd(value)
	return valueStart >= rawStart && valueEnd <= rawEnd && valueStart < valueEnd
}

func lineReplacementOffsets(source []byte, target tomlast.Range) (int, int) {
	start := min(max(rangeStart(target), 0), len(source))

	end := start
	for end < len(source) && source[end] != '\n' {
		end++
	}
	if end < len(source) {
		end++
	}

	return start, end
}

func lineEndOffset(source []byte, target tomlast.Range) int {
	_, end := lineReplacementOffsets(source, target)
	return end
}

func renderKeyValueLine(path []string, value any) (string, error) {
	tree, err := newEmptyTree()
	if err != nil {
		return "", err
	}
	normalized, err := normalizeTOMLValue(value)
	if err != nil {
		return "", err
	}
	tree.SetPath(path, normalized)
	rendered, err := tree.ToTomlString()
	if err != nil {
		return "", fmt.Errorf("config: encode TOML key-value: %w", err)
	}
	return ensureTrailingNewline(strings.TrimRight(rendered, "\n")), nil
}

func renderTableFragment(path []string, values map[string]any) (string, error) {
	cleanPath, err := normalizeMutationPath(path)
	if err != nil {
		return "", err
	}
	body, err := renderTreeFragment(values)
	if err != nil {
		return "", err
	}

	lines := []string{"[" + strings.Join(cleanPath, ".") + "]"}
	for line := range strings.SplitSeq(strings.TrimRight(body, "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "[[") && strings.HasSuffix(trimmed, "]]"):
			nested := strings.TrimSuffix(strings.TrimPrefix(trimmed, "[["), "]]")
			lines = append(lines, "[["+strings.Join(append(clonePath(cleanPath), nested), ".")+"]]")
		case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
			nested := strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
			lines = append(lines, "["+strings.Join(append(clonePath(cleanPath), nested), ".")+"]")
		default:
			lines = append(lines, trimmed)
		}
	}

	return ensureTrailingNewline(strings.Join(lines, "\n")), nil
}

func renderArrayTableFragment(path []string, values map[string]any) (string, error) {
	cleanPath, err := normalizeMutationPath(path)
	if err != nil {
		return "", err
	}
	body, err := renderTreeFragment(values)
	if err != nil {
		return "", err
	}
	header := "[[" + strings.Join(cleanPath, ".") + "]]"

	lines := []string{header}
	for line := range strings.SplitSeq(strings.TrimRight(body, "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "[[") && strings.HasSuffix(trimmed, "]]"):
			nested := strings.TrimSuffix(strings.TrimPrefix(trimmed, "[["), "]]")
			lines = append(lines, "[["+strings.Join(append(clonePath(cleanPath), nested), ".")+"]]")
		case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
			nested := strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
			lines = append(lines, "["+strings.Join(append(clonePath(cleanPath), nested), ".")+"]")
		default:
			lines = append(lines, trimmed)
		}
	}

	return ensureTrailingNewline(strings.Join(lines, "\n")), nil
}

func renderTreeFragment(values map[string]any) (string, error) {
	tree, err := newTreeFromMap(values)
	if err != nil {
		return "", err
	}
	rendered, err := tree.ToTomlString()
	if err != nil {
		return "", fmt.Errorf("config: encode TOML fragment: %w", err)
	}
	return rendered, nil
}

func decodeStringValue(value []byte) (string, bool) {
	tree, err := tomltree.LoadBytes([]byte("value = " + strings.TrimSpace(string(value))))
	if err != nil {
		return "", false
	}
	text, ok := tree.Get("value").(string)
	return text, ok
}

func replaceRange(source []byte, r tomlast.Range, replacement []byte) []byte {
	return replaceOffsets(source, rangeStart(r), rangeEnd(r), replacement)
}

func replaceOffsets(source []byte, start int, end int, replacement []byte) []byte {
	var buffer bytes.Buffer
	buffer.Grow(len(source) - (end - start) + len(replacement))
	buffer.Write(source[:start])
	buffer.Write(replacement)
	buffer.Write(source[end:])
	return buffer.Bytes()
}

func insertAtOffset(source []byte, offset int, fragment string) []byte {
	var buffer bytes.Buffer
	buffer.Grow(len(source) + len(fragment) + 1)
	buffer.Write(source[:offset])
	if offset > 0 && source[offset-1] != '\n' {
		buffer.WriteByte('\n')
	}
	buffer.WriteString(ensureTrailingNewline(fragment))
	buffer.Write(source[offset:])
	return buffer.Bytes()
}

func appendFragment(source []byte, fragment string) []byte {
	if len(source) == 0 {
		return []byte(ensureTrailingNewline(fragment))
	}

	var buffer bytes.Buffer
	buffer.Grow(len(source) + len(fragment) + 2)
	buffer.Write(source)
	if source[len(source)-1] != '\n' {
		buffer.WriteByte('\n')
	}
	if !bytes.HasSuffix(source, []byte("\n\n")) {
		buffer.WriteByte('\n')
	}
	buffer.WriteString(strings.TrimRight(fragment, "\n"))
	buffer.WriteByte('\n')
	return buffer.Bytes()
}

func ensureTrailingNewline(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}

func rangeStart(r tomlast.Range) int {
	return int(r.Offset)
}

func rangeEnd(r tomlast.Range) int {
	return int(r.Offset + r.Length)
}

func clonePath(path []string) []string {
	if len(path) == 0 {
		return nil
	}
	return append([]string(nil), path...)
}

func pathsEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func pathHasPrefix(path []string, prefix []string) bool {
	if len(prefix) > len(path) {
		return false
	}
	for idx := range prefix {
		if path[idx] != prefix[idx] {
			return false
		}
	}
	return true
}

func unsupportedTOMLMutation(path []string, reason string) error {
	joined := strings.Join(path, ".")
	if strings.TrimSpace(joined) == "" {
		joined = "<root>"
	}
	return fmt.Errorf("%w at %q: %s", ErrUnsupportedTOMLMutation, joined, reason)
}

func sortedStringKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringMapToAny(values map[string]string) map[string]any {
	converted := make(map[string]any, len(values))
	for key, value := range values {
		converted[key] = value
	}
	return converted
}

func cloneStringAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	clone := make(map[string]any, len(values))
	maps.Copy(clone, values)
	return clone
}

func readOptionalFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}
	return content, nil
}

func writePersistedFile(path string, contents []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp config file in %q: %w", dir, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp config file %q: %w", tmpPath, err)
	}
	if _, err := tmpFile.Write(contents); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp config file %q: %w", tmpPath, err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp config file %q: %w", tmpPath, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp config file %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace config file %q: %w", path, err)
	}
	if err := syncPersistedDir(dir); err != nil {
		return err
	}
	return nil
}

func samePath(left string, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}

func syncPersistedDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open config directory %q for sync: %w", dir, err)
	}
	defer func() {
		_ = handle.Close()
	}()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("sync config directory %q: %w", dir, err)
	}
	return nil
}
