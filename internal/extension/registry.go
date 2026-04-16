package extensionpkg

import (
	"context"
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
	// ErrExtensionHasActiveBundles reports that the extension lifecycle is
	// blocked by one or more active bundle activations.
	ErrExtensionHasActiveBundles = errors.New("extension: extension has active bundle activations")
)

const (
	registrySelectColumns = `
		SELECT
			name,
			version,
			source,
			enabled,
			manifest_path,
			installed_at,
			capabilities,
			actions,
			checksum,
			registry_slug,
			registry_name,
			remote_version
	`
	registryInsertColumns = `
			name,
			version,
			source,
			enabled,
			manifest_path,
			installed_at,
			capabilities,
			actions,
			checksum,
			registry_slug,
			registry_name,
			remote_version
	`
)

// Registry persists installed extension metadata in the global SQLite database.
type Registry struct {
	db  *sql.DB
	now func() time.Time
}

// ExtensionInfo is one persisted extension registry row.
type ExtensionInfo struct {
	Name          string
	Version       string
	Source        ExtensionSource
	Enabled       bool
	ManifestPath  string
	InstalledAt   time.Time
	Capabilities  CapabilitiesConfig
	Actions       ActionsConfig
	Checksum      string
	RegistrySlug  *string
	RegistryName  *string
	RemoteVersion *string
}

type installConfig struct {
	source          ExtensionSource
	replaceExisting bool
	registrySlug    *string
	registryName    *string
	remoteVersion   *string
}

// InstallOption customizes one extension registry install operation.
type InstallOption func(*installConfig)

// WithInstallSource overrides the persisted source tier for one install.
func WithInstallSource(source ExtensionSource) InstallOption {
	return func(cfg *installConfig) {
		cfg.source = source
	}
}

// WithInstallReplaceExisting allows an install to overwrite an existing row.
func WithInstallReplaceExisting() InstallOption {
	return func(cfg *installConfig) {
		cfg.replaceExisting = true
	}
}

// WithInstallRegistryMetadata records remote registry provenance for one install.
func WithInstallRegistryMetadata(slug string, registryName string, remoteVersion string) InstallOption {
	return func(cfg *installConfig) {
		cfg.registrySlug = optionalInstallString(slug)
		cfg.registryName = optionalInstallString(registryName)
		cfg.remoteVersion = optionalInstallString(remoteVersion)
	}
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

// DB exposes the backing SQLite handle for composition-root integrations that
// need to build additional stores over the same registry database.
func (r *Registry) DB() *sql.DB {
	if r == nil {
		return nil
	}
	return r.db
}

// Install verifies the extension artifact checksum and persists the install as
// a user-sourced extension.
func (r *Registry) Install(manifest *Manifest, path string, checksum string, opts ...InstallOption) error {
	config := installConfig{
		source: SourceUser,
	}
	applyInstallOptions(&config, opts...)

	return r.installWithConfig(manifest, path, checksum, config)
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
	if err := r.ensureNoActiveBundles(trimmedName); err != nil {
		return err
	}

	result, err := r.db.ExecContext(registryContext(), `DELETE FROM extensions WHERE name = ?`, trimmedName)
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

	rows, err := r.db.QueryContext(registryContext(), `
`+registrySelectColumns+`
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

	row := r.db.QueryRowContext(registryContext(), `
`+registrySelectColumns+`
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

func (r *Registry) installWithConfig(manifest *Manifest, path string, checksum string, config installConfig) error {
	if err := r.checkReady("install extension"); err != nil {
		return err
	}
	sourceText, err := validateInstallConfig(manifest, checksum, &config)
	if err != nil {
		return err
	}

	resolvedManifest, manifestPath, actualChecksum, err := loadVerifiedInstallManifest(manifest, path, checksum)
	if err != nil {
		return err
	}

	info := registryInstallInfo(r, resolvedManifest, manifestPath, actualChecksum, config)
	return r.persistInstalledInfo(info, sourceText, config.replaceExisting)
}

func applyInstallOptions(config *installConfig, opts ...InstallOption) {
	if config == nil {
		return
	}

	for _, opt := range opts {
		if opt != nil {
			opt(config)
		}
	}
}

func validateInstallConfig(manifest *Manifest, checksum string, config *installConfig) (string, error) {
	if manifest == nil {
		return "", errors.New("extension: manifest is required")
	}
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	if strings.TrimSpace(checksum) == "" {
		return "", errors.New("extension: checksum is required")
	}
	if config == nil {
		return "", errors.New("extension: install config is required")
	}

	sourceText := config.source.String()
	if sourceText == "" {
		return "", fmt.Errorf("extension: invalid source %d", config.source)
	}
	if config.source != SourceMarketplace {
		config.registrySlug = nil
		config.registryName = nil
		config.remoteVersion = nil
	}
	return sourceText, nil
}

func loadVerifiedInstallManifest(
	manifest *Manifest,
	path string,
	checksum string,
) (*Manifest, string, string, error) {
	artifactRoot, manifestPath, err := resolveInstallArtifact(path)
	if err != nil {
		return nil, "", "", err
	}

	trimmedChecksum := strings.ToLower(strings.TrimSpace(checksum))
	actualChecksum, err := ComputeDirectoryChecksum(artifactRoot)
	if err != nil {
		return nil, "", "", err
	}
	if actualChecksum != trimmedChecksum {
		return nil, "", "", &ExtensionChecksumMismatchError{
			ExpectedChecksum: trimmedChecksum,
			ActualChecksum:   actualChecksum,
		}
	}

	resolvedManifest, err := loadManifestAtPath(manifestPath)
	if err != nil {
		return nil, "", "", err
	}
	if strings.TrimSpace(manifest.Name) != strings.TrimSpace(resolvedManifest.Name) ||
		strings.TrimSpace(manifest.Version) != strings.TrimSpace(resolvedManifest.Version) {
		return nil, "", "", fmt.Errorf(
			"extension: manifest %q does not match provided identity %q@%q",
			manifestPath,
			strings.TrimSpace(manifest.Name),
			strings.TrimSpace(manifest.Version),
		)
	}

	return resolvedManifest, manifestPath, actualChecksum, nil
}

func registryInstallInfo(
	r *Registry,
	resolvedManifest *Manifest,
	manifestPath string,
	actualChecksum string,
	config installConfig,
) ExtensionInfo {
	return ExtensionInfo{
		Name:          strings.TrimSpace(resolvedManifest.Name),
		Version:       strings.TrimSpace(resolvedManifest.Version),
		Source:        config.source,
		Enabled:       true,
		ManifestPath:  manifestPath,
		InstalledAt:   r.now().UTC(),
		Capabilities:  normalizeCapabilitiesConfig(resolvedManifest.Capabilities),
		Actions:       normalizeActionsConfig(resolvedManifest.Actions),
		Checksum:      actualChecksum,
		RegistrySlug:  config.registrySlug,
		RegistryName:  config.registryName,
		RemoteVersion: config.remoteVersion,
	}
}

func (r *Registry) persistInstalledInfo(info ExtensionInfo, sourceText string, replaceExisting bool) error {
	capabilitiesJSON, err := json.Marshal(info.Capabilities)
	if err != nil {
		return fmt.Errorf("extension: marshal capabilities for %q: %w", info.Name, err)
	}
	actionsJSON, err := json.Marshal(info.Actions)
	if err != nil {
		return fmt.Errorf("extension: marshal actions for %q: %w", info.Name, err)
	}

	query := `
		INSERT INTO extensions (
` + registryInsertColumns + `
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	if replaceExisting {
		query += `
		ON CONFLICT(name) DO UPDATE SET
			version = excluded.version,
			source = excluded.source,
			manifest_path = excluded.manifest_path,
			installed_at = excluded.installed_at,
			capabilities = excluded.capabilities,
			actions = excluded.actions,
			checksum = excluded.checksum,
			registry_slug = excluded.registry_slug,
			registry_name = excluded.registry_name,
			remote_version = excluded.remote_version
		`
	}

	_, err = r.db.ExecContext(
		registryContext(),
		query,
		info.Name,
		info.Version,
		sourceText,
		info.Enabled,
		info.ManifestPath,
		store.FormatTimestamp(info.InstalledAt),
		string(capabilitiesJSON),
		string(actionsJSON),
		info.Checksum,
		nullableStringValue(info.RegistrySlug),
		nullableStringValue(info.RegistryName),
		nullableStringValue(info.RemoteVersion),
	)
	if err != nil {
		if replaceExisting {
			return fmt.Errorf("extension: persist %q: %w", info.Name, err)
		}
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
	if !enabled {
		if err := r.ensureNoActiveBundles(trimmedName); err != nil {
			return err
		}
	}

	result, err := r.db.ExecContext(
		registryContext(),
		`UPDATE extensions SET enabled = ? WHERE name = ?`,
		enabled,
		trimmedName,
	)
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

func (r *Registry) ensureNoActiveBundles(extensionName string) error {
	rows, err := r.db.QueryContext(
		registryContext(),
		`SELECT spec_json FROM resource_records WHERE kind = ?`,
		"bundle.activation",
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return nil
		}
		return fmt.Errorf("extension: count active bundle activations for %q: %w", extensionName, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	count := 0
	trimmedName := strings.TrimSpace(extensionName)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return fmt.Errorf("extension: scan active bundle activation for %q: %w", extensionName, err)
		}
		var spec struct {
			ExtensionName string `json:"extension_name"`
		}
		if err := json.Unmarshal([]byte(raw), &spec); err != nil {
			return fmt.Errorf("extension: decode active bundle activation for %q: %w", extensionName, err)
		}
		if strings.TrimSpace(spec.ExtensionName) == trimmedName {
			count++
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("extension: iterate active bundle activations for %q: %w", extensionName, err)
	}
	if count > 0 {
		return fmt.Errorf("%w: %q has %d active activation(s)", ErrExtensionHasActiveBundles, extensionName, count)
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
		registrySlug    sql.NullString
		registryName    sql.NullString
		remoteVersion   sql.NullString
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
		&registrySlug,
		&registryName,
		&remoteVersion,
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
	info.RegistrySlug = nullableStringPointer(registrySlug)
	info.RegistryName = nullableStringPointer(registryName)
	info.RemoteVersion = nullableStringPointer(remoteVersion)

	return &info, nil
}

func registryContext() context.Context {
	return context.Background()
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return optionalInstallString(value.String)
}

func nullableStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return strings.TrimSpace(*value)
}

func optionalInstallString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
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

		if err := writeChecksumString(
			hasher,
			fmt.Sprintf("file:%s\nmode:%#o\n", normalizedPath, info.Mode().Perm()),
		); err != nil {
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
		target, err := os.Readlink(absPath)
		if err != nil {
			return fmt.Errorf("extension: read checksum symlink %q: %w", absPath, err)
		}
		normalizedTarget := filepath.ToSlash(filepath.Clean(target))
		return writeChecksumString(
			hasher,
			fmt.Sprintf("symlink:%s\nmode:%#o\ntarget:%s\n", normalizedPath, info.Mode().Perm(), normalizedTarget),
		)
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
