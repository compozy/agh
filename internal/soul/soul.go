// Package soul resolves optional SOUL.md persona artifacts.
package soul

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/goccy/go-yaml"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/frontmatter"
)

const (
	// FileName is the canonical authored persona filename.
	FileName = "SOUL.md"

	digestPrefix = "agh.soul.v1\n"
)

var (
	// ErrInvalid reports a SOUL.md file that exists but cannot be accepted.
	ErrInvalid = errors.New("soul: invalid SOUL.md")
	// ErrPathEscape reports a SOUL.md path outside its configured workspace root.
	ErrPathEscape = errors.New("soul: path escapes workspace root")
)

// ResolveRequest describes a SOUL.md file located beside an AGENT.md file.
type ResolveRequest struct {
	AgentPath     string
	WorkspaceRoot string
	Config        aghconfig.SoulConfig
}

// ParseRequest describes in-memory SOUL.md content to validate and project.
type ParseRequest struct {
	SourcePath    string
	WorkspaceRoot string
	Content       []byte
	Config        aghconfig.SoulConfig
}

// ResolvedSoul is the normalized result that later runtime surfaces can consume.
type ResolvedSoul struct {
	Enabled     bool
	Present     bool
	Active      bool
	Valid       bool
	SourcePath  string
	Digest      string
	Profile     Profile
	Compact     CompactProjection
	ReadModel   ReadModel
	Diagnostics []Diagnostic
}

// Profile is the normalized authored persona profile.
type Profile struct {
	SourcePath    string
	Digest        string
	Version       string
	Role          string
	Tone          []string
	Principles    []string
	Constraints   []string
	Collaboration []string
	MemoryPolicy  []string
	Tags          []string
	Body          string
	Truncated     bool
}

// CompactProjection is the bounded context-safe soul projection.
type CompactProjection struct {
	Enabled      bool
	Present      bool
	Active       bool
	Digest       string
	SourcePath   string
	Role         string
	Tone         []string
	Principles   []string
	Truncated    bool
	MaxBytes     int64
	MaxBodyBytes int64
}

// ReadModel is the full resolved soul view for dedicated inspect surfaces.
type ReadModel struct {
	Enabled                bool
	Present                bool
	Active                 bool
	Valid                  bool
	SourcePath             string
	Digest                 string
	Frontmatter            Frontmatter
	Body                   string
	Truncated              bool
	MaxBodyBytes           int64
	ContextProjectionBytes int64
	Diagnostics            []Diagnostic
}

// Frontmatter is the allowlisted strict SOUL.md metadata.
type Frontmatter struct {
	Version       string
	Role          string
	Tone          []string
	Principles    []string
	Constraints   []string
	Collaboration []string
	MemoryPolicy  []string
	Tags          []string
}

// Diagnostic describes a closed, redacted SOUL.md validation problem.
type Diagnostic struct {
	Code       string
	Field      string
	Section    string
	Message    string
	SourcePath string
	Line       int
	Column     int
}

// DiagnosticError carries structured diagnostics for invalid authored content.
type DiagnosticError struct {
	Diagnostics []Diagnostic
	cause       error
}

func (e *DiagnosticError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return ErrInvalid.Error()
	}
	first := e.Diagnostics[0]
	return fmt.Sprintf("%s: %s", first.Code, first.Message)
}

// Unwrap exposes the validation sentinel for errors.Is callers.
func (e *DiagnosticError) Unwrap() error {
	if e == nil || e.cause == nil {
		return ErrInvalid
	}
	return e.cause
}

// Resolve reads and resolves SOUL.md beside the provided agent path.
func Resolve(ctx context.Context, req ResolveRequest) (ResolvedSoul, error) {
	if err := req.Config.Validate(); err != nil {
		return ResolvedSoul{}, err
	}
	if err := checkContext(ctx); err != nil {
		return ResolvedSoul{}, err
	}

	soulPath, pathErr := soulPathForAgent(req.AgentPath)
	safePath, diagnostic := safeSourcePath(soulPath, req.WorkspaceRoot)
	result := emptyResult(req.Config, safePath)
	if pathErr != nil {
		diag := diagnosticForError("invalid_source_path", safePath, pathErr, ErrInvalid)
		diag.Message = "SOUL.md source path is required"
		return resultWithDiagnostics(&result, []Diagnostic{diag})
	}
	if diagnostic != nil {
		return resultWithDiagnostics(&result, []Diagnostic{*diagnostic})
	}

	content, err := os.ReadFile(soulPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		diag := diagnosticForError("parser_io", safePath, err, err)
		diag.Message = "SOUL.md could not be read"
		return resultWithDiagnostics(&result, []Diagnostic{diag})
	}
	if err := checkContext(ctx); err != nil {
		return ResolvedSoul{}, err
	}

	return Parse(ctx, ParseRequest{
		SourcePath:    soulPath,
		WorkspaceRoot: req.WorkspaceRoot,
		Content:       content,
		Config:        req.Config,
	})
}

// Parse validates and normalizes in-memory SOUL.md content.
func Parse(ctx context.Context, req ParseRequest) (ResolvedSoul, error) {
	if err := req.Config.Validate(); err != nil {
		return ResolvedSoul{}, err
	}
	if err := checkContext(ctx); err != nil {
		return ResolvedSoul{}, err
	}

	safePath, diagnostic := safeSourcePath(req.SourcePath, req.WorkspaceRoot)
	result := emptyResult(req.Config, safePath)
	if diagnostic != nil {
		return resultWithDiagnostics(&result, []Diagnostic{*diagnostic})
	}
	result.Present = true
	result.Active = req.Config.Enabled
	result.Compact.Present = true
	result.Compact.Active = req.Config.Enabled
	result.ReadModel.Present = true
	result.ReadModel.Active = req.Config.Enabled

	front, body, offset, parseDiagnostics := parseDocument(req.Content, req.Config, safePath)
	if len(parseDiagnostics) > 0 {
		return resultWithDiagnostics(&result, parseDiagnostics)
	}

	validationDiagnostics := validateReservedSections(body, safePath, offset)
	if len(validationDiagnostics) > 0 {
		return resultWithDiagnostics(&result, validationDiagnostics)
	}

	digest, err := digestSoul(front, body)
	if err != nil {
		diag := diagnosticForError("digest_failed", safePath, err, ErrInvalid)
		diag.Message = "SOUL.md digest could not be computed"
		return resultWithDiagnostics(&result, []Diagnostic{diag})
	}

	profile := Profile{
		SourcePath:    safePath,
		Digest:        digest,
		Version:       front.Version,
		Role:          front.Role,
		Tone:          cloneStrings(front.Tone),
		Principles:    cloneStrings(front.Principles),
		Constraints:   cloneStrings(front.Constraints),
		Collaboration: cloneStrings(front.Collaboration),
		MemoryPolicy:  cloneStrings(front.MemoryPolicy),
		Tags:          cloneStrings(front.Tags),
		Body:          body,
	}
	compact := compactProjection(req.Config, profile)
	profile.Truncated = compact.Truncated
	readModel := ReadModel{
		Enabled:                req.Config.Enabled,
		Present:                true,
		Active:                 req.Config.Enabled,
		Valid:                  true,
		SourcePath:             safePath,
		Digest:                 digest,
		Frontmatter:            front,
		Body:                   body,
		Truncated:              compact.Truncated,
		MaxBodyBytes:           req.Config.MaxBodyBytes,
		ContextProjectionBytes: req.Config.ContextProjectionBytes,
	}
	return ResolvedSoul{
		Enabled:    req.Config.Enabled,
		Present:    true,
		Active:     req.Config.Enabled,
		Valid:      true,
		SourcePath: safePath,
		Digest:     digest,
		Profile:    profile,
		Compact:    compact,
		ReadModel:  readModel,
	}, nil
}

// Empty returns a valid absent SOUL.md resolution for non-filesystem agent sources.
func Empty(config aghconfig.SoulConfig, sourcePath string) (ResolvedSoul, error) {
	if err := config.Validate(); err != nil {
		return ResolvedSoul{}, err
	}
	safePath, diagnostic := safeSourcePath(sourcePath, "")
	result := emptyResult(config, safePath)
	if diagnostic != nil {
		return resultWithDiagnostics(&result, []Diagnostic{*diagnostic})
	}
	return result, nil
}

func parseDocument(
	content []byte,
	cfg aghconfig.SoulConfig,
	sourcePath string,
) (Frontmatter, string, int, []Diagnostic) {
	normalized := normalizeLineEndings(content)
	parts, err := frontmatter.Split(normalized)
	if err != nil {
		if errors.Is(err, frontmatter.ErrMissing) {
			body := normalizeBody(string(normalized))
			if int64(len([]byte(body))) > cfg.MaxBodyBytes {
				return Frontmatter{}, "", 1, []Diagnostic{{
					Code:       "oversized_body",
					Message:    fmt.Sprintf("SOUL.md body exceeds agents.soul.max_body_bytes (%d)", cfg.MaxBodyBytes),
					SourcePath: sourcePath,
					Line:       1,
					Column:     1,
				}}
			}
			return Frontmatter{}, body, 1, nil
		}
		diag := diagnosticForError("malformed_frontmatter", sourcePath, err, ErrInvalid)
		diag.Message = "SOUL.md frontmatter is unterminated or malformed"
		diag.Line = 1
		diag.Column = 1
		return Frontmatter{}, "", 1, []Diagnostic{diag}
	}

	if int64(len(parts.Metadata)) > cfg.MaxBodyBytes {
		return Frontmatter{}, "", 1, []Diagnostic{{
			Code:       "oversized_frontmatter",
			Message:    fmt.Sprintf("SOUL.md frontmatter exceeds agents.soul.max_body_bytes (%d)", cfg.MaxBodyBytes),
			SourcePath: sourcePath,
			Line:       2,
			Column:     1,
		}}
	}
	body := normalizeBody(parts.Body)
	if int64(len([]byte(body))) > cfg.MaxBodyBytes {
		return Frontmatter{}, "", 1, []Diagnostic{{
			Code:       "oversized_body",
			Message:    fmt.Sprintf("SOUL.md body exceeds agents.soul.max_body_bytes (%d)", cfg.MaxBodyBytes),
			SourcePath: sourcePath,
			Line:       bodyStartLine(parts.Metadata),
			Column:     1,
		}}
	}

	front, diagnostics := parseFrontmatter(parts.Metadata, sourcePath)
	if len(diagnostics) > 0 {
		return Frontmatter{}, "", 1, diagnostics
	}
	return front, body, bodyStartLine(parts.Metadata), nil
}

func parseFrontmatter(metadata []byte, sourcePath string) (Frontmatter, []Diagnostic) {
	if strings.TrimSpace(string(metadata)) == "" {
		return Frontmatter{}, nil
	}

	var raw map[string]any
	if err := yaml.UnmarshalWithOptions(metadata, &raw, yaml.Strict()); err != nil {
		diag := diagnosticForError("malformed_frontmatter", sourcePath, err, ErrInvalid)
		diag.Message = "SOUL.md frontmatter is not valid YAML"
		diag.Line = 2
		diag.Column = 1
		return Frontmatter{}, []Diagnostic{diag}
	}

	diagnosticsList := make([]Diagnostic, 0)
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var front Frontmatter
	for _, key := range keys {
		value := raw[key]
		locationLine, locationColumn := frontmatterKeyLocation(metadata, key)
		if !isAllowedField(key) {
			code := "unsupported_field"
			message := fmt.Sprintf("SOUL.md frontmatter field %q is not supported", key)
			if owner := forbiddenOwner(key); owner != "" {
				code = "forbidden_field"
				message = fmt.Sprintf("SOUL.md frontmatter field %q belongs to %s", key, owner)
			}
			diagnosticsList = append(diagnosticsList, Diagnostic{
				Code:       code,
				Field:      key,
				Message:    diagnostics.Redact(message),
				SourcePath: sourcePath,
				Line:       locationLine,
				Column:     locationColumn,
			})
			continue
		}
		if err := assignAllowedField(&front, key, value); err != nil {
			diagnosticsList = append(diagnosticsList, Diagnostic{
				Code:       "invalid_field_type",
				Field:      key,
				Message:    diagnostics.Redact(err.Error()),
				SourcePath: sourcePath,
				Line:       locationLine,
				Column:     locationColumn,
			})
		}
	}
	if len(diagnosticsList) > 0 {
		return Frontmatter{}, diagnosticsList
	}

	return front, nil
}

func assignAllowedField(front *Frontmatter, key string, value any) error {
	switch key {
	case "version":
		version, err := scalarString(value)
		if err != nil {
			return fmt.Errorf("SOUL.md frontmatter field %q must be a string or number", key)
		}
		front.Version = version
	case "role":
		role, err := stringOnly(value)
		if err != nil {
			return fmt.Errorf("SOUL.md frontmatter field %q must be a string", key)
		}
		front.Role = role
	case "tone":
		values, err := stringList(value, key)
		if err != nil {
			return err
		}
		front.Tone = values
	case "principles":
		values, err := stringList(value, key)
		if err != nil {
			return err
		}
		front.Principles = values
	case "constraints":
		values, err := stringList(value, key)
		if err != nil {
			return err
		}
		front.Constraints = values
	case "collaboration":
		values, err := stringList(value, key)
		if err != nil {
			return err
		}
		front.Collaboration = values
	case "memory_policy":
		values, err := stringList(value, key)
		if err != nil {
			return err
		}
		front.MemoryPolicy = values
	case "tags":
		values, err := stringList(value, key)
		if err != nil {
			return err
		}
		front.Tags = values
	}
	return nil
}

func validateReservedSections(body string, sourcePath string, bodyLineOffset int) []Diagnostic {
	lines := strings.Split(body, "\n")
	diagnosticsList := make([]Diagnostic, 0)
	for idx, line := range lines {
		section, ok := markdownHeadingKey(line)
		if !ok {
			continue
		}
		if owner := forbiddenOwner(section); owner != "" {
			diagnosticsList = append(diagnosticsList, Diagnostic{
				Code:       "reserved_section",
				Section:    section,
				Message:    diagnostics.Redact(fmt.Sprintf("SOUL.md section %q belongs to %s", section, owner)),
				SourcePath: sourcePath,
				Line:       bodyLineOffset + idx,
				Column:     1,
			})
		}
	}
	return diagnosticsList
}

func digestSoul(front Frontmatter, body string) (string, error) {
	canonical, err := json.Marshal(front)
	if err != nil {
		return "", fmt.Errorf("marshal canonical frontmatter: %w", err)
	}
	sum := sha256.Sum256([]byte(digestPrefix + string(canonical) + "\n" + body))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func compactProjection(cfg aghconfig.SoulConfig, profile Profile) CompactProjection {
	projection := CompactProjection{
		Enabled:      cfg.Enabled,
		Present:      true,
		Active:       cfg.Enabled,
		Digest:       profile.Digest,
		SourcePath:   profile.SourcePath,
		Role:         profile.Role,
		Tone:         cloneStrings(profile.Tone),
		Principles:   cloneStrings(profile.Principles),
		MaxBytes:     cfg.ContextProjectionBytes,
		MaxBodyBytes: cfg.MaxBodyBytes,
	}
	if projectionWithinBudget(projection, cfg.ContextProjectionBytes) {
		return projection
	}
	projection.Truncated = true
	for len(projection.Principles) > 0 && !projectionWithinBudget(projection, cfg.ContextProjectionBytes) {
		projection.Principles = projection.Principles[:len(projection.Principles)-1]
	}
	for len(projection.Tone) > 0 && !projectionWithinBudget(projection, cfg.ContextProjectionBytes) {
		projection.Tone = projection.Tone[:len(projection.Tone)-1]
	}
	if !projectionWithinBudget(projection, cfg.ContextProjectionBytes) {
		for projection.Role != "" && !projectionWithinBudget(projection, cfg.ContextProjectionBytes) {
			runes := []rune(projection.Role)
			projection.Role = string(runes[:len(runes)-1])
		}
	}
	if !projectionWithinBudget(projection, cfg.ContextProjectionBytes) {
		projection.SourcePath = ""
	}
	if !projectionWithinBudget(projection, cfg.ContextProjectionBytes) {
		projection.Digest = ""
	}
	return projection
}

func projectionWithinBudget(projection CompactProjection, maxBytes int64) bool {
	if maxBytes <= 0 {
		return false
	}
	data, err := json.Marshal(projection)
	if err != nil {
		return false
	}
	return int64(len(data)) <= maxBytes
}

func resultWithDiagnostics(result *ResolvedSoul, list []Diagnostic) (ResolvedSoul, error) {
	if result == nil {
		result = &ResolvedSoul{}
	}
	result.Valid = false
	result.Active = false
	result.Diagnostics = sanitizeDiagnostics(list)
	result.ReadModel.Valid = false
	result.ReadModel.Active = false
	result.ReadModel.Diagnostics = cloneDiagnostics(result.Diagnostics)
	result.Compact.Active = false
	return *result, &DiagnosticError{Diagnostics: cloneDiagnostics(result.Diagnostics), cause: ErrInvalid}
}

func emptyResult(cfg aghconfig.SoulConfig, sourcePath string) ResolvedSoul {
	compact := CompactProjection{
		Enabled:      cfg.Enabled,
		Present:      false,
		Active:       false,
		SourcePath:   sourcePath,
		MaxBytes:     cfg.ContextProjectionBytes,
		MaxBodyBytes: cfg.MaxBodyBytes,
	}
	readModel := ReadModel{
		Enabled:                cfg.Enabled,
		Present:                false,
		Active:                 false,
		Valid:                  true,
		SourcePath:             sourcePath,
		MaxBodyBytes:           cfg.MaxBodyBytes,
		ContextProjectionBytes: cfg.ContextProjectionBytes,
	}
	return ResolvedSoul{
		Enabled:    cfg.Enabled,
		Present:    false,
		Active:     false,
		Valid:      true,
		SourcePath: sourcePath,
		Compact:    compact,
		ReadModel:  readModel,
	}
}

func diagnosticForError(code string, sourcePath string, err error, cause error) Diagnostic {
	message := ""
	if err != nil {
		message = diagnostics.RedactAndBound(err.Error(), 300)
	}
	if strings.TrimSpace(message) == "" && cause != nil {
		message = cause.Error()
	}
	return Diagnostic{
		Code:       code,
		Message:    message,
		SourcePath: sourcePath,
	}
}

func safeSourcePath(sourcePath string, workspaceRoot string) (string, *Diagnostic) {
	trimmed := strings.TrimSpace(sourcePath)
	if trimmed == "" {
		return "", nil
	}
	if strings.ContainsRune(trimmed, 0) {
		return FileName, &Diagnostic{
			Code:       "path_escape",
			Message:    "SOUL.md path contains an invalid NUL byte",
			SourcePath: FileName,
		}
	}

	cleanSource := filepath.Clean(trimmed)
	if strings.TrimSpace(workspaceRoot) == "" {
		return safePathWithoutRoot(cleanSource), nil
	}

	absRoot, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return safePathWithoutRoot(cleanSource), &Diagnostic{
			Code:       "path_escape",
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve workspace root: %v", err), 300),
			SourcePath: safePathWithoutRoot(cleanSource),
		}
	}
	sourceForRoot := cleanSource
	if !filepath.IsAbs(sourceForRoot) {
		sourceForRoot = filepath.Join(absRoot, sourceForRoot)
	}
	absSource, err := filepath.Abs(sourceForRoot)
	if err != nil {
		return safePathWithoutRoot(cleanSource), &Diagnostic{
			Code:       "path_escape",
			Message:    diagnostics.RedactAndBound(fmt.Sprintf("resolve SOUL.md path: %v", err), 300),
			SourcePath: safePathWithoutRoot(cleanSource),
		}
	}

	safePath, within := relativePathWithinRoot(absRoot, absSource)
	if !within {
		return safePath, &Diagnostic{
			Code:       "path_escape",
			Message:    "SOUL.md path must stay inside the workspace root",
			SourcePath: safePath,
		}
	}
	if resolvedRoot, rootErr := filepath.EvalSymlinks(absRoot); rootErr == nil {
		if resolvedSource, sourceErr := filepath.EvalSymlinks(absSource); sourceErr == nil {
			safeResolved, resolvedWithin := relativePathWithinRoot(resolvedRoot, resolvedSource)
			if !resolvedWithin {
				return safePath, &Diagnostic{
					Code:       "path_escape",
					Message:    "SOUL.md symlink target must stay inside the workspace root",
					SourcePath: safeResolved,
				}
			}
		}
	}
	return safePath, nil
}

func relativePathWithinRoot(root string, target string) (string, bool) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return safePathWithoutRoot(target), false
	}
	if rel == "." {
		return ".", true
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return safePathWithoutRoot(target), false
	}
	return filepath.ToSlash(rel), true
}

func safePathWithoutRoot(path string) string {
	slashed := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(slashed, "/")
	for idx := 0; idx < len(parts)-2; idx++ {
		if parts[idx] == ".agh" && parts[idx+1] == "agents" {
			return strings.Join(parts[idx:], "/")
		}
	}
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	if slashed == "." {
		return FileName
	}
	return strings.TrimPrefix(slashed, "/")
}

func soulPathForAgent(agentPath string) (string, error) {
	trimmed := strings.TrimSpace(agentPath)
	if trimmed == "" {
		return "", errors.New("agent path is required")
	}
	cleaned := filepath.Clean(trimmed)
	if strings.EqualFold(filepath.Base(cleaned), "AGENT.md") {
		return filepath.Join(filepath.Dir(cleaned), FileName), nil
	}
	return filepath.Join(cleaned, FileName), nil
}

func isAllowedField(key string) bool {
	switch key {
	case "version", "role", "tone", "principles", "constraints", "collaboration", "memory_policy", "tags":
		return true
	default:
		return false
	}
}

func forbiddenOwner(key string) string {
	switch key {
	case "name", "provider", "command", "model", "tools", "toolsets", "deny_tools", "permissions",
		"mcp_servers", "hooks", "prompt":
		return "AGENT.md"
	case "capabilities", "capability":
		return "capabilities"
	case "tasks", "task", "task_runs", "run", "ownership", "claim_token", "claim_token_hash":
		return "task runtime"
	case "scheduler", "heartbeat", "lease", "session", "session_liveness", "activity", "wake":
		return "runtime state"
	case "network", "channels", "peers", "presence":
		return "AGH Network presence"
	case "spawn":
		return "session spawn overlays"
	case "env", "config", "defaults", "providers", "sandboxes", "settings":
		return "config"
	case "memory", "memory_store", "memory_scope", "memory_type", "memories":
		return "memory runtime"
	default:
		return ""
	}
}

func markdownHeadingKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level >= len(trimmed) || !unicode.IsSpace(rune(trimmed[level])) {
		return "", false
	}
	title := strings.TrimSpace(trimmed[level:])
	title = strings.Trim(title, "# ")
	if title == "" {
		return "", false
	}
	return normalizeKey(title), true
}

func normalizeKey(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func frontmatterKeyLocation(metadata []byte, key string) (int, int) {
	lines := strings.Split(string(metadata), "\n")
	for idx, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		col := len(line) - len(trimmed) + 1
		before, _, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		candidate := strings.Trim(strings.TrimSpace(before), `"'`)
		if candidate == key {
			return idx + 2, col
		}
	}
	return 2, 1
}

func bodyStartLine(metadata []byte) int {
	if len(metadata) == 0 {
		return 3
	}
	return 3 + strings.Count(string(metadata), "\n")
}

func normalizeLineEndings(content []byte) []byte {
	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return []byte(normalized)
}

func normalizeBody(body string) string {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.TrimRightFunc(normalized, unicode.IsSpace)
}

func stringOnly(value any) (string, error) {
	typed, ok := value.(string)
	if !ok {
		return "", errors.New("value is not a string")
	}
	return strings.TrimSpace(typed), nil
}

func scalarString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), nil
	case int:
		return fmt.Sprintf("%d", typed), nil
	case int64:
		return fmt.Sprintf("%d", typed), nil
	case uint64:
		return fmt.Sprintf("%d", typed), nil
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%.0f", typed), nil
		}
	}
	return "", errors.New("value is not a scalar string")
}

func stringList(value any, key string) ([]string, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil, nil
		}
		return []string{trimmed}, nil
	case []string:
		return normalizeStrings(typed), nil
	case []any:
		values := make([]string, 0, len(typed))
		for idx, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("SOUL.md frontmatter field %q item %d must be a string", key, idx)
			}
			if trimmed := strings.TrimSpace(text); trimmed != "" {
				values = append(values, trimmed)
			}
		}
		return values, nil
	default:
		return nil, fmt.Errorf("SOUL.md frontmatter field %q must be a string list", key)
	}
}

func normalizeStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func sanitizeDiagnostics(list []Diagnostic) []Diagnostic {
	result := make([]Diagnostic, 0, len(list))
	for _, item := range list {
		item.Message = diagnostics.RedactAndBound(item.Message, 300)
		item.SourcePath = sanitizeDiagnosticPath(item.SourcePath)
		result = append(result, item)
	}
	return result
}

func sanitizeDiagnosticPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return safePathWithoutRoot(path)
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func cloneDiagnostics(values []Diagnostic) []Diagnostic {
	return append([]Diagnostic(nil), values...)
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("resolve soul: %w", ctx.Err())
	default:
		return nil
	}
}
