package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	capabilityCatalogTOMLName = "capabilities.toml"
	capabilityCatalogJSONName = "capabilities.json"
	capabilityCatalogDirName  = "capabilities"
	capabilityFileExtTOML     = ".toml"
	capabilityFileExtJSON     = ".json"
)

// CapabilityDef is one normalized, outcome-oriented capability declaration for an agent.
type CapabilityDef struct {
	ID                string   `json:"id"                     toml:"id"`
	Summary           string   `json:"summary"                toml:"summary"`
	Outcome           string   `json:"outcome"                toml:"outcome"`
	Version           string   `json:"version,omitempty"      toml:"version,omitempty"`
	ContextNeeded     []string `json:"context_needed"         toml:"context_needed"`
	ArtifactsExpected []string `json:"artifacts_expected"     toml:"artifacts_expected"`
	ExecutionOutline  []string `json:"execution_outline"      toml:"execution_outline"`
	Constraints       []string `json:"constraints"            toml:"constraints"`
	Examples          []string `json:"examples"               toml:"examples"`
	Requirements      []string `json:"requirements,omitempty" toml:"requirements,omitempty"`
	Digest            string   `json:"-"                      toml:"-"`
}

// CapabilityCatalog is the normalized local catalog loaded from one agent directory.
type CapabilityCatalog struct {
	Capabilities []CapabilityDef `json:"capabilities" toml:"capabilities"`
}

// CapabilityBrief is the compact discovery projection for one capability.
type CapabilityBrief struct {
	ID      string `json:"id"      toml:"id"`
	Summary string `json:"summary" toml:"summary"`
}

type capabilityCatalogLayoutMode string

const (
	capabilityCatalogLayoutModeNone      capabilityCatalogLayoutMode = ""
	capabilityCatalogLayoutModeFile      capabilityCatalogLayoutMode = "file"
	capabilityCatalogLayoutModeDirectory capabilityCatalogLayoutMode = "directory"
)

type capabilityCatalogLayout struct {
	mode capabilityCatalogLayoutMode
	file string
	dir  string
}

type capabilityCatalogRecord struct {
	source     string
	basename   string
	capability CapabilityDef
}

type canonicalCapabilityDigestPayload struct {
	ID                string   `json:"id"`
	Summary           string   `json:"summary"`
	Outcome           string   `json:"outcome"`
	Version           string   `json:"version,omitempty"`
	ContextNeeded     []string `json:"context_needed,omitempty"`
	ArtifactsExpected []string `json:"artifacts_expected,omitempty"`
	ExecutionOutline  []string `json:"execution_outline,omitempty"`
	Constraints       []string `json:"constraints,omitempty"`
	Examples          []string `json:"examples,omitempty"`
	Requirements      []string `json:"requirements,omitempty"`
}

// Clone returns a deep copy of the catalog.
func (c *CapabilityCatalog) Clone() *CapabilityCatalog {
	if c == nil {
		return nil
	}

	cloned := &CapabilityCatalog{
		Capabilities: make([]CapabilityDef, 0, len(c.Capabilities)),
	}
	for _, capability := range c.Capabilities {
		cloned.Capabilities = append(cloned.Capabilities, cloneCapabilityDef(capability))
	}

	return cloned
}

// LoadAgentCapabilities loads the optional capability catalog for one agent directory.
// When no supported capability catalog exists, it returns nil without error.
func LoadAgentCapabilities(agentDir string) (*CapabilityCatalog, error) {
	trimmedDir := strings.TrimSpace(agentDir)
	if trimmedDir == "" {
		return nil, errors.New("config: agent directory is required")
	}

	layout, err := detectCapabilityCatalogLayout(trimmedDir)
	if err != nil {
		return nil, err
	}

	switch layout.mode {
	case capabilityCatalogLayoutModeNone:
		return nil, nil
	case capabilityCatalogLayoutModeFile:
		return loadCapabilityCatalogFile(layout.file)
	case capabilityCatalogLayoutModeDirectory:
		return loadCapabilityCatalogDirectory(layout.dir)
	default:
		return nil, fmt.Errorf("config: unsupported capability catalog mode %q", layout.mode)
	}
}

func detectCapabilityCatalogLayout(agentDir string) (capabilityCatalogLayout, error) {
	tomlPath := filepath.Join(agentDir, capabilityCatalogTOMLName)
	jsonPath := filepath.Join(agentDir, capabilityCatalogJSONName)
	dirPath := filepath.Join(agentDir, capabilityCatalogDirName)

	tomlExists, err := existingCapabilityCatalogFile(tomlPath)
	if err != nil {
		return capabilityCatalogLayout{}, err
	}
	jsonExists, err := existingCapabilityCatalogFile(jsonPath)
	if err != nil {
		return capabilityCatalogLayout{}, err
	}
	dirExists, err := existingCapabilityCatalogDir(dirPath)
	if err != nil {
		return capabilityCatalogLayout{}, err
	}

	files := make([]string, 0, 2)
	if tomlExists {
		files = append(files, tomlPath)
	}
	if jsonExists {
		files = append(files, jsonPath)
	}

	if dirExists && len(files) > 0 {
		conflicts := append([]string(nil), files...)
		conflicts = append(conflicts, dirPath)
		return capabilityCatalogLayout{}, fmt.Errorf(
			"config: validate capability catalog %q: mixed capability catalog layouts: %s",
			agentDir,
			joinQuotedPaths(conflicts),
		)
	}
	if len(files) > 1 {
		return capabilityCatalogLayout{}, fmt.Errorf(
			"config: validate capability catalog %q: multiple capability catalog files: %s",
			agentDir,
			joinQuotedPaths(files),
		)
	}
	if len(files) == 1 {
		return capabilityCatalogLayout{
			mode: capabilityCatalogLayoutModeFile,
			file: files[0],
		}, nil
	}
	if dirExists {
		return capabilityCatalogLayout{
			mode: capabilityCatalogLayoutModeDirectory,
			dir:  dirPath,
		}, nil
	}

	return capabilityCatalogLayout{}, nil
}

func existingCapabilityCatalogFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("config: stat capability catalog file %q: %w", path, err)
	}
	if info.IsDir() {
		return false, fmt.Errorf("config: capability catalog file %q must be a file", path)
	}
	return true, nil
}

func existingCapabilityCatalogDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("config: stat capability catalog directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("config: capability catalog directory %q must be a directory", path)
	}
	return true, nil
}

func loadCapabilityCatalogFile(path string) (*CapabilityCatalog, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read capability catalog %q: %w", path, err)
	}

	switch filepath.Ext(path) {
	case capabilityFileExtTOML:
		return parseCapabilityCatalogTOML(content, path)
	case capabilityFileExtJSON:
		return parseCapabilityCatalogJSON(content, path)
	default:
		return nil, fmt.Errorf("config: unsupported capability catalog file %q", path)
	}
}

func loadCapabilityCatalogDirectory(dir string) (*CapabilityCatalog, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("config: read capability catalog directory %q: %w", dir, err)
	}

	tomlFiles := make([]string, 0)
	jsonFiles := make([]string, 0)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("config: read capability catalog entry %q: %w", filepath.Join(dir, name), err)
		}
		if !info.Mode().IsRegular() {
			continue
		}

		path := filepath.Join(dir, name)
		switch filepath.Ext(name) {
		case capabilityFileExtTOML:
			tomlFiles = append(tomlFiles, path)
		case capabilityFileExtJSON:
			jsonFiles = append(jsonFiles, path)
		}
	}

	sort.Strings(tomlFiles)
	sort.Strings(jsonFiles)

	if len(tomlFiles) > 0 && len(jsonFiles) > 0 {
		conflicts := append(append([]string(nil), tomlFiles...), jsonFiles...)
		return nil, fmt.Errorf(
			"config: validate capability catalog %q: mixed capability file formats: %s",
			dir,
			joinQuotedPaths(conflicts),
		)
	}

	selected := tomlFiles
	if len(selected) == 0 {
		selected = jsonFiles
	}
	if len(selected) == 0 {
		return &CapabilityCatalog{Capabilities: []CapabilityDef{}}, nil
	}

	records := make([]capabilityCatalogRecord, 0, len(selected))
	for _, path := range selected {
		capability, err := loadCapabilityDefFile(path)
		if err != nil {
			return nil, err
		}
		records = append(records, capabilityCatalogRecord{
			source:     path,
			basename:   strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			capability: capability,
		})
	}

	return normalizeCapabilityCatalogRecords(records, dir)
}

func loadCapabilityDefFile(path string) (CapabilityDef, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return CapabilityDef{}, fmt.Errorf("config: read capability definition %q: %w", path, err)
	}

	switch filepath.Ext(path) {
	case capabilityFileExtTOML:
		return parseCapabilityDefTOML(content, path)
	case capabilityFileExtJSON:
		return parseCapabilityDefJSON(content, path)
	default:
		return CapabilityDef{}, fmt.Errorf("config: unsupported capability definition file %q", path)
	}
}

func parseCapabilityCatalogJSON(content []byte, source string) (*CapabilityCatalog, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()

	var catalog CapabilityCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return nil, fmt.Errorf("config: decode capability JSON %q: %w", source, err)
	}
	if err := ensureJSONDocumentEOF(decoder, source, "capability JSON"); err != nil {
		return nil, err
	}

	return normalizeCapabilityCatalog(&catalog, source)
}

func parseCapabilityCatalogTOML(content []byte, source string) (*CapabilityCatalog, error) {
	var catalog CapabilityCatalog
	if err := decodeStrictCapabilityTOML(content, &catalog); err != nil {
		return nil, fmt.Errorf("config: decode capability TOML %q: %w", source, err)
	}

	return normalizeCapabilityCatalog(&catalog, source)
}

func parseCapabilityDefJSON(content []byte, source string) (CapabilityDef, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()

	var capability CapabilityDef
	if err := decoder.Decode(&capability); err != nil {
		return CapabilityDef{}, fmt.Errorf("config: decode capability JSON %q: %w", source, err)
	}
	if err := ensureJSONDocumentEOF(decoder, source, "capability JSON"); err != nil {
		return CapabilityDef{}, err
	}

	return capability, nil
}

func parseCapabilityDefTOML(content []byte, source string) (CapabilityDef, error) {
	var capability CapabilityDef
	if err := decodeStrictCapabilityTOML(content, &capability); err != nil {
		return CapabilityDef{}, fmt.Errorf("config: decode capability TOML %q: %w", source, err)
	}

	return capability, nil
}

func decodeStrictCapabilityTOML(content []byte, dest any) error {
	meta, err := toml.Decode(string(content), dest)
	if err != nil {
		return err
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return fmt.Errorf("unknown field %q", undecoded[0].String())
	}
	return nil
}

func normalizeCapabilityCatalog(catalog *CapabilityCatalog, source string) (*CapabilityCatalog, error) {
	if catalog == nil {
		return nil, nil
	}

	normalized := &CapabilityCatalog{
		Capabilities: make([]CapabilityDef, 0, len(catalog.Capabilities)),
	}
	seen := make(map[string]int, len(catalog.Capabilities))

	for idx, capability := range catalog.Capabilities {
		next := normalizeCapabilityDef(capability)
		if err := validateCapabilityDef(capability, next, source, idx); err != nil {
			return nil, err
		}
		next.Requirements = canonicalizeCapabilityRequirements(next.Requirements)
		if priorIdx, ok := seen[next.ID]; ok {
			return nil, fmt.Errorf(
				"config: validate capability catalog %q: duplicate capability id %q after normalization at indexes %d and %d",
				source,
				next.ID,
				priorIdx,
				idx,
			)
		}
		digest, err := computeCapabilityDigest(next)
		if err != nil {
			return nil, fmt.Errorf(
				"config: compute capability digest for %q in %q: %w",
				next.ID,
				source,
				err,
			)
		}
		next.Digest = digest
		seen[next.ID] = idx
		normalized.Capabilities = append(normalized.Capabilities, next)
	}

	return normalized, nil
}

func normalizeCapabilityCatalogRecords(
	records []capabilityCatalogRecord,
	source string,
) (*CapabilityCatalog, error) {
	normalized := &CapabilityCatalog{
		Capabilities: make([]CapabilityDef, 0, len(records)),
	}
	seen := make(map[string]string, len(records))

	for _, record := range records {
		next := normalizeCapabilityDef(record.capability)
		if err := validateDirectoryCapabilityDef(record.capability, next, record); err != nil {
			return nil, err
		}
		next.Requirements = canonicalizeCapabilityRequirements(next.Requirements)
		if priorSource, ok := seen[next.ID]; ok {
			return nil, fmt.Errorf(
				"config: validate capability catalog %q: duplicate capability id %q after normalization in %q and %q",
				source,
				next.ID,
				priorSource,
				record.source,
			)
		}
		digest, err := computeCapabilityDigest(next)
		if err != nil {
			return nil, fmt.Errorf(
				"config: compute capability digest for %q in %q: %w",
				next.ID,
				record.source,
				err,
			)
		}
		next.Digest = digest
		seen[next.ID] = record.source
		normalized.Capabilities = append(normalized.Capabilities, next)
	}

	return normalized, nil
}

func normalizeCapabilityDef(capability CapabilityDef) CapabilityDef {
	return CapabilityDef{
		ID:                strings.TrimSpace(capability.ID),
		Summary:           strings.TrimSpace(capability.Summary),
		Outcome:           strings.TrimSpace(capability.Outcome),
		Version:           strings.TrimSpace(capability.Version),
		ContextNeeded:     normalizeCapabilityStringList(capability.ContextNeeded),
		ArtifactsExpected: normalizeCapabilityStringList(capability.ArtifactsExpected),
		ExecutionOutline:  normalizeCapabilityStringList(capability.ExecutionOutline),
		Constraints:       normalizeCapabilityStringList(capability.Constraints),
		Examples:          normalizeCapabilityStringList(capability.Examples),
		Requirements:      normalizeCapabilityRequirementList(capability.Requirements),
	}
}

func validateCapabilityDef(raw CapabilityDef, capability CapabilityDef, source string, idx int) error {
	fieldPrefix := fmt.Sprintf("config: validate capability catalog %q: capabilities[%d]", source, idx)

	switch {
	case capability.ID == "":
		return fmt.Errorf("%s.id is required", fieldPrefix)
	case capability.Summary == "":
		return fmt.Errorf("%s.summary is required", fieldPrefix)
	case capability.Outcome == "":
		return fmt.Errorf("%s.outcome is required", fieldPrefix)
	case raw.Version != "" && capability.Version == "":
		return fmt.Errorf("%s.version must not be blank when provided", fieldPrefix)
	}

	if err := validateCapabilityRequirements(capability.Requirements, fieldPrefix+".requirements"); err != nil {
		return err
	}
	return nil
}

func validateDirectoryCapabilityDef(raw CapabilityDef, capability CapabilityDef, record capabilityCatalogRecord) error {
	normalizedBasename := strings.TrimSpace(record.basename)

	switch {
	case capability.ID == "":
		return fmt.Errorf("config: validate capability %q: id is required", record.source)
	case capability.Summary == "":
		return fmt.Errorf("config: validate capability %q: summary is required", record.source)
	case capability.Outcome == "":
		return fmt.Errorf("config: validate capability %q: outcome is required", record.source)
	case normalizedBasename != capability.ID:
		return fmt.Errorf(
			"config: validate capability %q: basename %q must match id %q",
			record.source,
			record.basename,
			capability.ID,
		)
	case raw.Version != "" && capability.Version == "":
		return fmt.Errorf("config: validate capability %q: version must not be blank when provided", record.source)
	}

	if err := validateCapabilityRequirements(
		capability.Requirements,
		fmt.Sprintf("config: validate capability %q: requirements", record.source),
	); err != nil {
		return err
	}
	return nil
}

func normalizeCapabilityStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func normalizeCapabilityRequirementList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, len(values))
	for idx, value := range values {
		normalized[idx] = strings.TrimSpace(value)
	}

	return normalized
}

func validateCapabilityRequirements(requirements []string, fieldPrefix string) error {
	if len(requirements) == 0 {
		return nil
	}

	seen := make(map[string]int, len(requirements))
	for idx, requirement := range requirements {
		if requirement == "" {
			return fmt.Errorf("%s[%d] is required", fieldPrefix, idx)
		}
		if priorIdx, ok := seen[requirement]; ok {
			return fmt.Errorf(
				"%s duplicate value %q after normalization at indexes %d and %d",
				fieldPrefix,
				requirement,
				priorIdx,
				idx,
			)
		}
		seen[requirement] = idx
	}

	return nil
}

func canonicalizeCapabilityRequirements(requirements []string) []string {
	if len(requirements) == 0 {
		return nil
	}

	canonical := append([]string(nil), requirements...)
	sort.Strings(canonical)
	return canonical
}

func computeCapabilityDigest(capability CapabilityDef) (string, error) {
	payload, err := json.Marshal(canonicalCapabilityDigestPayload{
		ID:                capability.ID,
		Summary:           capability.Summary,
		Outcome:           capability.Outcome,
		Version:           capability.Version,
		ContextNeeded:     append([]string(nil), capability.ContextNeeded...),
		ArtifactsExpected: append([]string(nil), capability.ArtifactsExpected...),
		ExecutionOutline:  append([]string(nil), capability.ExecutionOutline...),
		Constraints:       append([]string(nil), capability.Constraints...),
		Examples:          append([]string(nil), capability.Examples...),
		Requirements:      canonicalizeCapabilityRequirements(capability.Requirements),
	})
	if err != nil {
		return "", fmt.Errorf("marshal canonical capability: %w", err)
	}

	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func cloneCapabilityDef(capability CapabilityDef) CapabilityDef {
	return CapabilityDef{
		ID:                capability.ID,
		Summary:           capability.Summary,
		Outcome:           capability.Outcome,
		Version:           capability.Version,
		ContextNeeded:     append([]string(nil), capability.ContextNeeded...),
		ArtifactsExpected: append([]string(nil), capability.ArtifactsExpected...),
		ExecutionOutline:  append([]string(nil), capability.ExecutionOutline...),
		Constraints:       append([]string(nil), capability.Constraints...),
		Examples:          append([]string(nil), capability.Examples...),
		Requirements:      append([]string(nil), capability.Requirements...),
		Digest:            capability.Digest,
	}
}

func joinQuotedPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	quoted := make([]string, 0, len(paths))
	for _, path := range paths {
		quoted = append(quoted, fmt.Sprintf("%q", path))
	}
	return strings.Join(quoted, ", ")
}

func ensureJSONDocumentEOF(decoder *json.Decoder, source string, label string) error {
	if decoder == nil {
		return errors.New("config: JSON decoder is required")
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("config: decode %s %q: %w", label, source, err)
	}

	return fmt.Errorf("config: decode %s %q: unexpected trailing JSON value", label, source)
}
