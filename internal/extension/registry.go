package extension

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	// ErrExtensionNotFound reports that no installed extension matched the lookup.
	ErrExtensionNotFound = errors.New("extension: extension not found")
	// ErrExtensionExists reports that an extension name is already installed.
	ErrExtensionExists = errors.New("extension: extension already exists")
	// ErrExtensionChecksumMismatch reports that the provided checksum does not
	// match the on-disk extension artifact.
	ErrExtensionChecksumMismatch = errors.New("extension: checksum mismatch")
)

// Registry persists installed extension metadata in the global SQLite database.
type Registry struct {
	db  *sql.DB
	now func() time.Time
}

// ExtensionInfo is one persisted extension registry row.
type ExtensionInfo struct {
	Name         string
	Version      string
	Source       ExtensionSource
	Enabled      bool
	ManifestPath string
	InstalledAt  time.Time
	Capabilities CapabilitiesConfig
	Actions      ActionsConfig
	Checksum     string
}

// ExtensionNotFoundError describes a missing extension registry row.
type ExtensionNotFoundError struct {
	Name string
}

// ExtensionExistsError describes a duplicate extension install attempt.
type ExtensionExistsError struct {
	Name string
}

// ExtensionChecksumMismatchError describes a checksum verification failure.
type ExtensionChecksumMismatchError struct {
	ExpectedChecksum string
	ActualChecksum   string
}

// NewRegistry constructs a registry over an existing SQLite connection.
func NewRegistry(db *sql.DB) *Registry {
	return &Registry{
		db: db,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// Install verifies the extension artifact checksum and persists the install as
// a user-sourced extension.
func (r *Registry) Install(manifest *Manifest, path string, checksum string) error {
	return r.installWithSource(manifest, path, checksum, SourceUser)
}

// Uninstall removes one extension from the registry.
func (r *Registry) Uninstall(name string) error {
	if err := r.checkReady("uninstall extension"); err != nil {
		return err
	}

	trimmedName, err := normalizeExtensionName(name)
	if err != nil {
		return err
	}

	result, err := r.db.Exec(`DELETE FROM extensions WHERE name = ?`, trimmedName)
	if err != nil {
		return fmt.Errorf("extension: uninstall %q: %w", trimmedName, err)
	}

	return rowsAffectedNotFound(result, trimmedName)
}

// Enable marks one installed extension as enabled.
func (r *Registry) Enable(name string) error {
	return r.updateEnabled(name, true)
}

// Disable marks one installed extension as disabled.
func (r *Registry) Disable(name string) error {
	return r.updateEnabled(name, false)
}

// List returns every installed extension ordered by name.
func (r *Registry) List() ([]ExtensionInfo, error) {
	if err := r.checkReady("list extensions"); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(`
		SELECT name, version, source, enabled, manifest_path, installed_at, capabilities, actions, checksum
		FROM extensions
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("extension: list extensions: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	extensions := make([]ExtensionInfo, 0)
	for rows.Next() {
		info, err := scanExtensionInfo(rows)
		if err != nil {
			return nil, err
		}
		extensions = append(extensions, *info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("extension: iterate extensions: %w", err)
	}

	return extensions, nil
}

// Get returns one installed extension by name.
func (r *Registry) Get(name string) (*ExtensionInfo, error) {
	if err := r.checkReady("get extension"); err != nil {
		return nil, err
	}

	trimmedName, err := normalizeExtensionName(name)
	if err != nil {
		return nil, err
	}

	row := r.db.QueryRow(`
		SELECT name, version, source, enabled, manifest_path, installed_at, capabilities, actions, checksum
		FROM extensions
		WHERE name = ?
	`, trimmedName)

	info, err := scanExtensionInfo(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &ExtensionNotFoundError{Name: trimmedName}
	}
	if err != nil {
		return nil, err
	}

	return info, nil
}

// ComputeDirectoryChecksum returns a deterministic SHA-256 checksum for an
// installed extension directory payload.
func ComputeDirectoryChecksum(path string) (string, error) {
	root := strings.TrimSpace(path)
	if root == "" {
		return "", errors.New("extension: extension directory is required")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("extension: resolve extension directory %q: %w", path, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("extension: stat extension directory %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("extension: extension directory %q is not a directory", absRoot)
	}

	entries := make([]string, 0)
	err = filepath.WalkDir(absRoot, func(entryPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entryPath == absRoot {
			return nil
		}
		if entry.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, entryPath)
		if err != nil {
			return err
		}
		entries = append(entries, relPath)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("extension: walk extension directory %q: %w", absRoot, err)
	}

	slices.Sort(entries)
	hasher := sha256.New()
	for _, relPath := range entries {
		if err := writeChecksumEntry(hasher, absRoot, relPath); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (r *Registry) installWithSource(manifest *Manifest, path string, checksum string, source ExtensionSource) error {
	if err := r.checkReady("install extension"); err != nil {
		return err
	}
	if manifest == nil {
		return errors.New("extension: manifest is required")
	}
	if err := manifest.Validate(); err != nil {
		return err
	}

	trimmedChecksum := strings.ToLower(strings.TrimSpace(checksum))
	if trimmedChecksum == "" {
		return errors.New("extension: checksum is required")
	}

	sourceText := source.String()
	if sourceText == "" {
		return fmt.Errorf("extension: invalid source %d", source)
	}

	artifactRoot, manifestPath, err := resolveInstallArtifact(path)
	if err != nil {
		return err
	}

	actualChecksum, err := ComputeDirectoryChecksum(artifactRoot)
	if err != nil {
		return err
	}
	if actualChecksum != trimmedChecksum {
		return &ExtensionChecksumMismatchError{
			ExpectedChecksum: trimmedChecksum,
			ActualChecksum:   actualChecksum,
		}
	}

	resolvedManifest, err := loadManifestAtPath(manifestPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(manifest.Name) != strings.TrimSpace(resolvedManifest.Name) || strings.TrimSpace(manifest.Version) != strings.TrimSpace(resolvedManifest.Version) {
		return fmt.Errorf(
			"extension: manifest %q does not match provided identity %q@%q",
			manifestPath,
			strings.TrimSpace(manifest.Name),
			strings.TrimSpace(manifest.Version),
		)
	}

	capabilities := normalizeCapabilitiesConfig(resolvedManifest.Capabilities)
	actions := normalizeActionsConfig(resolvedManifest.Actions)
	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("extension: marshal capabilities for %q: %w", resolvedManifest.Name, err)
	}
	actionsJSON, err := json.Marshal(actions)
	if err != nil {
		return fmt.Errorf("extension: marshal actions for %q: %w", resolvedManifest.Name, err)
	}

	info := ExtensionInfo{
		Name:         strings.TrimSpace(resolvedManifest.Name),
		Version:      strings.TrimSpace(resolvedManifest.Version),
		Source:       source,
		Enabled:      true,
		ManifestPath: manifestPath,
		InstalledAt:  r.now().UTC(),
		Capabilities: capabilities,
		Actions:      actions,
		Checksum:     actualChecksum,
	}

	_, err = r.db.Exec(`
		INSERT INTO extensions (
			name, version, source, enabled, manifest_path, installed_at, capabilities, actions, checksum
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		info.Name,
		info.Version,
		sourceText,
		info.Enabled,
		info.ManifestPath,
		store.FormatTimestamp(info.InstalledAt),
		string(capabilitiesJSON),
		string(actionsJSON),
		info.Checksum,
	)
	if err != nil {
		return mapRegistryConstraintError(err, info.Name)
	}

	return nil
}

func (r *Registry) updateEnabled(name string, enabled bool) error {
	if err := r.checkReady("update extension enabled state"); err != nil {
		return err
	}

	trimmedName, err := normalizeExtensionName(name)
	if err != nil {
		return err
	}

	result, err := r.db.Exec(`UPDATE extensions SET enabled = ? WHERE name = ?`, enabled, trimmedName)
	if err != nil {
		return fmt.Errorf("extension: update enabled state for %q: %w", trimmedName, err)
	}

	return rowsAffectedNotFound(result, trimmedName)
}

func (r *Registry) checkReady(action string) error {
	if r == nil {
		return errors.New("extension: registry is required")
	}
	if r.db == nil {
		return fmt.Errorf("extension: %s database is required", action)
	}
	return nil
}

func scanExtensionInfo(scanner interface{ Scan(dest ...any) error }) (*ExtensionInfo, error) {
	var (
		info            ExtensionInfo
		sourceText      string
		installedAtText string
		capabilitiesRaw string
		actionsRaw      string
	)

	if err := scanner.Scan(
		&info.Name,
		&info.Version,
		&sourceText,
		&info.Enabled,
		&info.ManifestPath,
		&installedAtText,
		&capabilitiesRaw,
		&actionsRaw,
		&info.Checksum,
	); err != nil {
		return nil, err
	}

	source, err := parseExtensionSource(sourceText)
	if err != nil {
		return nil, err
	}
	info.Source = source

	info.InstalledAt, err = store.ParseTimestamp(installedAtText)
	if err != nil {
		return nil, fmt.Errorf("extension: parse installed_at for %q: %w", info.Name, err)
	}

	if err := decodeRegistryJSON(capabilitiesRaw, &info.Capabilities); err != nil {
		return nil, fmt.Errorf("extension: decode capabilities for %q: %w", info.Name, err)
	}
	if err := decodeRegistryJSON(actionsRaw, &info.Actions); err != nil {
		return nil, fmt.Errorf("extension: decode actions for %q: %w", info.Name, err)
	}

	info.Capabilities = normalizeCapabilitiesConfig(info.Capabilities)
	info.Actions = normalizeActionsConfig(info.Actions)

	return &info, nil
}

func decodeRegistryJSON(raw string, target any) error {
	payload := strings.TrimSpace(raw)
	if payload == "" {
		payload = "{}"
	}
	return json.Unmarshal([]byte(payload), target)
}

func normalizeExtensionName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New("extension: extension name is required")
	}
	return trimmed, nil
}

func resolveInstallArtifact(path string) (artifactRoot string, manifestPath string, err error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", "", errors.New("extension: install path is required")
	}

	absPath, err := filepath.Abs(trimmedPath)
	if err != nil {
		return "", "", fmt.Errorf("extension: resolve install path %q: %w", path, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", "", fmt.Errorf("extension: stat install path %q: %w", absPath, err)
	}
	if info.IsDir() {
		manifestPath, err := resolveManifestPath(absPath)
		if err != nil {
			return "", "", err
		}
		return absPath, manifestPath, nil
	}

	switch filepath.Base(absPath) {
	case manifestTOMLFileName, manifestJSONFileName:
		return filepath.Dir(absPath), absPath, nil
	default:
		return "", "", fmt.Errorf("extension: install path %q must be an extension directory or manifest file", absPath)
	}
}

func resolveManifestPath(dir string) (string, error) {
	tomlPath := filepath.Join(dir, manifestTOMLFileName)
	if exists, err := fileExists(tomlPath); err != nil {
		return "", fmt.Errorf("extension: stat manifest %q: %w", tomlPath, err)
	} else if exists {
		return tomlPath, nil
	}

	jsonPath := filepath.Join(dir, manifestJSONFileName)
	if exists, err := fileExists(jsonPath); err != nil {
		return "", fmt.Errorf("extension: stat manifest %q: %w", jsonPath, err)
	} else if exists {
		return jsonPath, nil
	}

	return "", &ManifestNotFoundError{
		Dir:   dir,
		Paths: []string{tomlPath, jsonPath},
	}
}

func parseExtensionSource(value string) (ExtensionSource, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case SourceBundled.String():
		return SourceBundled, nil
	case SourceUser.String():
		return SourceUser, nil
	case SourceWorkspace.String():
		return SourceWorkspace, nil
	case SourceMarketplace.String():
		return SourceMarketplace, nil
	default:
		return 0, fmt.Errorf("extension: unknown source %q", value)
	}
}

func rowsAffectedNotFound(result sql.Result, name string) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("extension: inspect rows affected for %q: %w", name, err)
	}
	if rowsAffected == 0 {
		return &ExtensionNotFoundError{Name: name}
	}
	return nil
}

func mapRegistryConstraintError(err error, name string) error {
	if err == nil {
		return nil
	}

	if sqliteErr, ok := err.(*sqlite.Error); ok && sqliteErr.Code()&0xff == sqlite3.SQLITE_CONSTRAINT {
		return &ExtensionExistsError{Name: name}
	}
	return fmt.Errorf("extension: persist %q: %w", name, err)
}

func writeChecksumEntry(hasher hash.Hash, root string, relPath string) error {
	normalizedPath := filepath.ToSlash(relPath)
	absPath := filepath.Join(root, relPath)

	info, err := os.Lstat(absPath)
	if err != nil {
		return fmt.Errorf("extension: stat checksum path %q: %w", absPath, err)
	}

	if info.Mode().IsRegular() {
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("extension: read checksum path %q: %w", absPath, err)
		}

		if err := writeChecksumString(hasher, fmt.Sprintf("file:%s\nmode:%#o\n", normalizedPath, info.Mode().Perm())); err != nil {
			return err
		}
		if _, err := hasher.Write(content); err != nil {
			return fmt.Errorf("extension: hash regular file %q: %w", absPath, err)
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return fmt.Errorf("extension: hash separator for %q: %w", absPath, err)
		}
		return nil
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("extension: symlinks are not allowed in extension payload %q", absPath)
	}

	return fmt.Errorf("extension: unsupported file type in extension payload %q", absPath)
}

func writeChecksumString(hasher hash.Hash, value string) error {
	if _, err := hasher.Write([]byte(value)); err != nil {
		return fmt.Errorf("extension: hash payload metadata: %w", err)
	}
	return nil
}

// Error returns the typed missing-extension message.
func (e *ExtensionNotFoundError) Error() string {
	trimmedName := strings.TrimSpace(e.Name)
	if trimmedName == "" {
		return ErrExtensionNotFound.Error()
	}
	return fmt.Sprintf("%s: %s", ErrExtensionNotFound, trimmedName)
}

// Is matches sentinel errors for errors.Is.
func (e *ExtensionNotFoundError) Is(target error) bool {
	return target == ErrExtensionNotFound
}

// Error returns the typed duplicate-extension message.
func (e *ExtensionExistsError) Error() string {
	trimmedName := strings.TrimSpace(e.Name)
	if trimmedName == "" {
		return ErrExtensionExists.Error()
	}
	return fmt.Sprintf("%s: %s", ErrExtensionExists, trimmedName)
}

// Is matches sentinel errors for errors.Is.
func (e *ExtensionExistsError) Is(target error) bool {
	return target == ErrExtensionExists
}

// Error returns the typed checksum mismatch message.
func (e *ExtensionChecksumMismatchError) Error() string {
	if e == nil {
		return ErrExtensionChecksumMismatch.Error()
	}
	return fmt.Sprintf(
		"%s: expected %s, got %s",
		ErrExtensionChecksumMismatch,
		e.ExpectedChecksum,
		e.ActualChecksum,
	)
}

// Is matches sentinel errors for errors.Is.
func (e *ExtensionChecksumMismatchError) Is(target error) bool {
	return target == ErrExtensionChecksumMismatch
}
