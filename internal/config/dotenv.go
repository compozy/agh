package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/joho/godotenv"
)

const (
	// DotEnvStatusMissing reports that no .env file exists at the requested path.
	DotEnvStatusMissing = "missing"
	// DotEnvStatusValid reports that the .env file is structured and needs no repair.
	DotEnvStatusValid = "valid"
	// DotEnvStatusRepairable reports that the .env file can be repaired explicitly.
	DotEnvStatusRepairable = "repairable"
	// DotEnvStatusRepaired reports that the .env file was safely rewritten.
	DotEnvStatusRepaired = "repaired"
	// DotEnvStatusUnsupported reports that AGH found content it will not rewrite.
	DotEnvStatusUnsupported = "unsupported"
)

const (
	dotEnvDiagnosticMultiKeyLine        = "multi_key_line"
	dotEnvDiagnosticSanitizedSecret     = "sanitized_secret_value"
	dotEnvDiagnosticUnsupportedLine     = "unsupported_line"
	dotEnvDiagnosticInvalidKey          = "invalid_key"
	dotEnvDiagnosticUnsupportedSymlink  = "unsupported_symlink"
	dotEnvDiagnosticUnsupportedDir      = "unsupported_directory"
	dotEnvDiagnosticUnsupportedFragment = "unsupported_fragment"
)

var (
	// ErrDotEnvUnsupported reports that .env content could not be safely parsed
	// or repaired without risking user-owned intent.
	ErrDotEnvUnsupported = errors.New("config: unsupported .env content")
)

// DotEnvDiagnostic describes one .env parse or repair issue without exposing values.
type DotEnvDiagnostic struct {
	Line    int    `json:"line,omitempty"`
	Key     string `json:"key,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DotEnvRepairReport summarizes .env inspection or repair without including values.
type DotEnvRepairReport struct {
	Path        string             `json:"path"`
	Status      string             `json:"status"`
	Repaired    bool               `json:"repaired"`
	Diagnostics []DotEnvDiagnostic `json:"diagnostics,omitempty"`
}

// DotEnvRepairError carries structured diagnostics for unsupported .env content.
type DotEnvRepairError struct {
	Path        string
	Diagnostics []DotEnvDiagnostic
}

type dotEnvParseResult struct {
	values       map[string]string
	lines        []string
	diagnostics  []DotEnvDiagnostic
	needsRepair  bool
	unsupported  bool
	finalNewline bool
}

// WorkspaceDotEnvFile returns the .env file path for a resolved workspace root.
func WorkspaceDotEnvFile(workspaceRoot string) string {
	return filepath.Join(strings.TrimSpace(workspaceRoot), ".env")
}

// InspectDotEnvFile parses one .env file and reports whether explicit repair is possible.
func InspectDotEnvFile(path string) (DotEnvRepairReport, error) {
	normalizedPath, data, exists, err := readDotEnvFile(path)
	if err != nil {
		var repairErr *DotEnvRepairError
		if errors.As(err, &repairErr) {
			return DotEnvRepairReport{
				Path:        normalizedPath,
				Status:      DotEnvStatusUnsupported,
				Diagnostics: append([]DotEnvDiagnostic(nil), repairErr.Diagnostics...),
			}, err
		}
		return DotEnvRepairReport{Path: normalizedPath, Status: DotEnvStatusMissing}, err
	}
	if !exists {
		return DotEnvRepairReport{Path: normalizedPath, Status: DotEnvStatusMissing}, nil
	}

	parsed := parseDotEnvDocument(string(data))
	return dotEnvReport(normalizedPath, parsed, false), nil
}

// RepairDotEnvFile safely rewrites one .env file when every change is bounded and structured.
func RepairDotEnvFile(path string) (DotEnvRepairReport, error) {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return DotEnvRepairReport{Path: normalizedPath, Status: DotEnvStatusMissing},
			errors.New("config: .env path is required")
	}

	info, err := os.Lstat(normalizedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DotEnvRepairReport{Path: normalizedPath, Status: DotEnvStatusMissing}, nil
		}
		return DotEnvRepairReport{Path: normalizedPath, Status: DotEnvStatusMissing}, fmt.Errorf(
			"stat .env file %q: %w",
			normalizedPath,
			err,
		)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		report := DotEnvRepairReport{
			Path:   normalizedPath,
			Status: DotEnvStatusUnsupported,
			Diagnostics: []DotEnvDiagnostic{{
				Code:    dotEnvDiagnosticUnsupportedSymlink,
				Message: ".env repair refuses to rewrite symlinks",
			}},
		}
		return report, dotEnvUnsupportedError(normalizedPath, report.Diagnostics)
	}
	if info.IsDir() {
		report := DotEnvRepairReport{
			Path:   normalizedPath,
			Status: DotEnvStatusUnsupported,
			Diagnostics: []DotEnvDiagnostic{{
				Code:    dotEnvDiagnosticUnsupportedDir,
				Message: ".env path is a directory",
			}},
		}
		return report, dotEnvUnsupportedError(normalizedPath, report.Diagnostics)
	}
	data, err := os.ReadFile(normalizedPath)
	if err != nil {
		return DotEnvRepairReport{Path: normalizedPath, Status: DotEnvStatusMissing}, fmt.Errorf(
			"read .env file %q: %w",
			normalizedPath,
			err,
		)
	}

	parsed := parseDotEnvDocument(string(data))
	report := dotEnvReport(normalizedPath, parsed, false)
	if parsed.unsupported {
		return report, dotEnvUnsupportedError(normalizedPath, parsed.diagnostics)
	}
	if !parsed.needsRepair {
		return report, nil
	}

	repaired := strings.Join(parsed.lines, "\n")
	if parsed.finalNewline {
		repaired += "\n"
	}
	if err := replaceDotEnvFile(normalizedPath, []byte(repaired), info.Mode().Perm()); err != nil {
		return report, err
	}

	report = dotEnvReport(normalizedPath, parsed, true)
	return report, nil
}

func readDotEnvFile(path string) (string, []byte, bool, error) {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return "", nil, false, errors.New("config: .env path is required")
	}

	info, err := os.Lstat(normalizedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return normalizedPath, nil, false, nil
		}
		return normalizedPath, nil, false, fmt.Errorf("stat .env file %q: %w", normalizedPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return normalizedPath, nil, false, dotEnvUnsupportedError(normalizedPath, []DotEnvDiagnostic{{
			Code:    dotEnvDiagnosticUnsupportedSymlink,
			Message: ".env load refuses to read symlinks",
		}})
	}
	if info.IsDir() {
		return normalizedPath, nil, false, dotEnvUnsupportedError(normalizedPath, []DotEnvDiagnostic{{
			Code:    dotEnvDiagnosticUnsupportedDir,
			Message: ".env path is a directory",
		}})
	}
	if !info.Mode().IsRegular() {
		return normalizedPath, nil, false, fmt.Errorf(".env file %q must be a regular file", normalizedPath)
	}

	data, err := os.ReadFile(normalizedPath)
	if err != nil {
		return normalizedPath, nil, false, fmt.Errorf("read .env file %q: %w", normalizedPath, err)
	}
	return normalizedPath, data, true, nil
}

func parseDotEnvDocument(raw string) dotEnvParseResult {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	finalNewline := len(lines) > 0 && lines[len(lines)-1] == ""
	if finalNewline {
		lines = lines[:len(lines)-1]
	}

	result := dotEnvParseResult{
		values:       map[string]string{},
		lines:        make([]string, 0, len(lines)),
		finalNewline: finalNewline,
	}

	for idx, line := range lines {
		parsedLines, values, diagnostics, needsRepair, unsupported := parseDotEnvLine(idx+1, line)
		result.lines = append(result.lines, parsedLines...)
		maps.Copy(result.values, values)
		result.diagnostics = append(result.diagnostics, diagnostics...)
		result.needsRepair = result.needsRepair || needsRepair
		result.unsupported = result.unsupported || unsupported
	}

	return result
}

func parseDotEnvLine(
	lineNumber int,
	line string,
) ([]string, map[string]string, []DotEnvDiagnostic, bool, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return []string{line}, nil, nil, false, false
	}

	lineForSplit := trimmed
	if after, ok := strings.CutPrefix(lineForSplit, "export "); ok {
		lineForSplit = strings.TrimSpace(after)
	}
	segments := splitDotEnvAssignments(lineForSplit)
	if len(segments) == 0 {
		return []string{line}, nil, []DotEnvDiagnostic{{
			Line:    lineNumber,
			Code:    dotEnvDiagnosticUnsupportedLine,
			Message: ".env line is not a KEY=VALUE assignment",
		}}, false, true
	}

	parsedLines := make([]string, 0, len(segments))
	values := make(map[string]string, len(segments))
	diagnostics := make([]DotEnvDiagnostic, 0)
	needsRepair := len(segments) > 1
	unsupported := false
	if len(segments) > 1 {
		diagnostics = append(diagnostics, DotEnvDiagnostic{
			Line:    lineNumber,
			Code:    dotEnvDiagnosticMultiKeyLine,
			Message: ".env line contains multiple assignments and can be split safely",
		})
	}

	for _, segment := range segments {
		parsedLine, key, value, segmentDiagnostics, segmentNeedsRepair, segmentUnsupported := parseDotEnvAssignment(
			lineNumber,
			segment,
			len(segments) > 1,
		)
		diagnostics = append(diagnostics, segmentDiagnostics...)
		needsRepair = needsRepair || segmentNeedsRepair
		unsupported = unsupported || segmentUnsupported
		if segmentUnsupported {
			continue
		}
		parsedLines = append(parsedLines, parsedLine)
		values[key] = value
	}

	if len(parsedLines) == 0 {
		parsedLines = append(parsedLines, line)
	}
	if !needsRepair && !unsupported {
		parsedLines = []string{line}
	}
	return parsedLines, values, diagnostics, needsRepair, unsupported
}

func parseDotEnvAssignment(
	lineNumber int,
	line string,
	forceRewrite bool,
) (string, string, string, []DotEnvDiagnostic, bool, bool) {
	candidate := strings.TrimSpace(line)
	if after, ok := strings.CutPrefix(candidate, "export "); ok {
		candidate = strings.TrimSpace(after)
	}

	key, ok := dotEnvAssignmentKey(candidate)
	if !ok {
		return line, "", "", []DotEnvDiagnostic{{
			Line:    lineNumber,
			Code:    dotEnvDiagnosticInvalidKey,
			Message: ".env assignment key must match [A-Za-z_][A-Za-z0-9_]*",
		}}, false, true
	}

	parsed, err := godotenv.Unmarshal(candidate)
	if err != nil {
		return line, "", "", []DotEnvDiagnostic{{
			Line:    lineNumber,
			Key:     key,
			Code:    dotEnvDiagnosticUnsupportedFragment,
			Message: "assignment value uses unsupported .env syntax",
		}}, false, true
	}
	value, ok := parsed[key]
	if !ok {
		return line, "", "", []DotEnvDiagnostic{{
			Line:    lineNumber,
			Key:     key,
			Code:    dotEnvDiagnosticUnsupportedFragment,
			Message: "assignment could not be parsed as a single .env key",
		}}, false, true
	}

	sanitized := sanitizeDotEnvValue(key, value)
	needsRepair := forceRewrite || sanitized != value
	diagnostics := []DotEnvDiagnostic(nil)
	if sanitized != value {
		diagnostics = append(diagnostics, DotEnvDiagnostic{
			Line:    lineNumber,
			Key:     key,
			Code:    dotEnvDiagnosticSanitizedSecret,
			Message: "secret-like .env value contains non-ASCII characters that will be stripped",
		})
	}
	if needsRepair {
		return formatDotEnvAssignment(key, sanitized), key, sanitized, diagnostics, true, false
	}
	return line, key, sanitized, diagnostics, false, false
}

func splitDotEnvAssignments(line string) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}
	startKey, ok := dotEnvAssignmentKey(trimmed)
	if !ok || startKey == "" {
		return nil
	}

	segments := make([]string, 0, 1)
	start := 0
	inSingle := false
	inDouble := false
	escaped := false
	for idx, r := range trimmed {
		if escaped {
			escaped = false
			continue
		}
		if inDouble && r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		default:
			if inSingle || inDouble || !unicode.IsSpace(r) {
				continue
			}
			next := nextNonSpaceIndex(trimmed, idx)
			if next < 0 {
				continue
			}
			if strings.HasPrefix(trimmed[next:], "export ") {
				exportValueStart := next + len("export ")
				next = nextNonSpaceIndex(trimmed, exportValueStart-1)
				if next < 0 {
					continue
				}
			}
			if _, ok := dotEnvAssignmentKey(trimmed[next:]); ok {
				segments = append(segments, strings.TrimSpace(trimmed[start:idx]))
				start = next
			}
		}
	}
	segments = append(segments, strings.TrimSpace(trimmed[start:]))
	return segments
}

func nextNonSpaceIndex(value string, from int) int {
	for idx := from + 1; idx < len(value); idx++ {
		if !unicode.IsSpace(rune(value[idx])) {
			return idx
		}
	}
	return -1
}

func dotEnvAssignmentKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", false
	}
	for idx, r := range trimmed {
		if r == '=' {
			key := strings.TrimSpace(trimmed[:idx])
			return key, validDotEnvKey(key)
		}
		if unicode.IsSpace(r) {
			return "", false
		}
	}
	return "", false
}

func validDotEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for idx, r := range key {
		if idx == 0 {
			if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}
		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func sanitizeDotEnvValue(key string, value string) string {
	if !secretLikeDotEnvKey(key) {
		return value
	}
	var builder strings.Builder
	for _, r := range value {
		if r <= unicode.MaxASCII {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func secretLikeDotEnvKey(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	return strings.HasSuffix(upper, "_API_KEY") ||
		strings.HasSuffix(upper, "_TOKEN") ||
		strings.HasSuffix(upper, "_SECRET") ||
		strings.HasSuffix(upper, "_KEY")
}

func formatDotEnvAssignment(key string, value string) string {
	if value == "" {
		return key + "="
	}
	if dotEnvValueNeedsQuoting(value) {
		return fmt.Sprintf("%s=%q", key, value)
	}
	return key + "=" + value
}

func dotEnvValueNeedsQuoting(value string) bool {
	for _, r := range value {
		if unicode.IsSpace(r) || r == '#' || r == '"' || r == '\'' || r == '=' {
			return true
		}
	}
	return false
}

func dotEnvReport(path string, parsed dotEnvParseResult, repaired bool) DotEnvRepairReport {
	status := DotEnvStatusValid
	switch {
	case parsed.unsupported:
		status = DotEnvStatusUnsupported
	case repaired:
		status = DotEnvStatusRepaired
	case parsed.needsRepair:
		status = DotEnvStatusRepairable
	}
	return DotEnvRepairReport{
		Path:        path,
		Status:      status,
		Repaired:    repaired,
		Diagnostics: append([]DotEnvDiagnostic(nil), parsed.diagnostics...),
	}
}

func replaceDotEnvFile(path string, contents []byte, mode os.FileMode) (err error) {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".env.repair-*")
	if err != nil {
		return fmt.Errorf("create temporary .env repair file in %q: %w", dir, err)
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			if removeErr := os.Remove(tempPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove temporary .env repair file %q: %w", tempPath, removeErr))
			}
		}
	}()

	if mode == 0 {
		mode = 0o600
	}
	if err := temp.Chmod(mode); err != nil {
		return closeFileAfterError(temp, tempPath, fmt.Errorf("set temporary .env repair mode %q: %w", tempPath, err))
	}
	if _, err := temp.Write(contents); err != nil {
		return closeFileAfterError(temp, tempPath, fmt.Errorf("write temporary .env repair file %q: %w", tempPath, err))
	}
	if err := temp.Sync(); err != nil {
		return closeFileAfterError(temp, tempPath, fmt.Errorf("sync temporary .env repair file %q: %w", tempPath, err))
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary .env repair file %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace .env file %q: %w", path, err)
	}
	cleanup = false
	if err := syncPersistedDir(dir); err != nil {
		return err
	}
	return nil
}

func dotEnvUnsupportedError(path string, diagnostics []DotEnvDiagnostic) error {
	return &DotEnvRepairError{
		Path:        path,
		Diagnostics: append([]DotEnvDiagnostic(nil), diagnostics...),
	}
}

// Error returns a diagnostic summary without including .env values.
func (e *DotEnvRepairError) Error() string {
	if e == nil {
		return ErrDotEnvUnsupported.Error()
	}
	parts := make([]string, 0, len(e.Diagnostics))
	for _, diagnostic := range e.Diagnostics {
		location := ""
		if diagnostic.Line > 0 {
			location = fmt.Sprintf("line %d", diagnostic.Line)
		}
		if diagnostic.Key != "" {
			if location != "" {
				location += " "
			}
			location += "key " + diagnostic.Key
		}
		if location == "" {
			location = "file"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", location, diagnostic.Message))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s in %q", ErrDotEnvUnsupported, e.Path)
	}
	return fmt.Sprintf("%s in %q (%s)", ErrDotEnvUnsupported, e.Path, strings.Join(parts, "; "))
}

// Is matches the unsupported .env sentinel.
func (e *DotEnvRepairError) Is(target error) bool {
	return target == ErrDotEnvUnsupported
}
