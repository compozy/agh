package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pedronauck/agh/internal/frontmatter"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultMaxArchiveSize caps the compressed archive stream before extraction.
	DefaultMaxArchiveSize int64 = 50 * 1024 * 1024

	defaultInstallerTempDirPattern = ".agh-install-*"
	defaultInstallerTempDirMaxAge  = time.Hour
)

const (
	installerSkillManifestName     = "SKILL.md"
	installerExtensionManifestName = "extension.toml"
)

var (
	errArchiveTooLargeCompressed = errors.New("registry: archive exceeds max compressed size")
	errInstallMissingManifest    = errors.New("registry: archive missing extension.toml or SKILL.md at root")
	errUnexpectedContentType     = errors.New("registry: unexpected download content type")
	errVerificationBlocked       = errors.New("registry: install blocked by content verification")
)

type installerVerificationRule struct {
	key     string
	regex   *regexp.Regexp
	message string
}

var installerVerificationRules = []installerVerificationRule{
	{
		key:     "ignore-previous-instructions",
		regex:   regexp.MustCompile(`(?i)\bignore\s+(?:\w+\s+)*(?:all|previous|prior|above)\s+(?:\w+\s+)*(?:instructions|rules|guidelines)\b`),
		message: "content attempts to override existing instructions",
	},
	{
		key:     "disregard-existing-rules",
		regex:   regexp.MustCompile(`(?i)\bdisregard\s+(?:\w+\s+)*(?:all|previous|prior|your)\s+(?:\w+\s+)*(?:instructions|rules|guidelines)\b`),
		message: "content attempts to bypass current rules",
	},
	{
		key:     "forget-your-instructions",
		regex:   regexp.MustCompile(`(?i)\bforget\s+(?:\w+\s+)*(?:your|all)\s+(?:\w+\s+)*(?:instructions|rules|guidelines)\b`),
		message: "content attempts to erase active instructions",
	},
	{
		key:     "role-hijack-you-are-now",
		regex:   regexp.MustCompile(`(?i)\byou\s+are\s+now\s+(?:a|an|the|assistant|agent|bot|system)\b`),
		message: "content attempts to redefine the agent role",
	},
	{
		key:     "new-instructions",
		regex:   regexp.MustCompile(`(?i)\bnew\s+instructions\s*:`),
		message: "content introduces overriding instructions",
	},
	{
		key:     "system-prompt-override",
		regex:   regexp.MustCompile(`(?i)\bsystem\s+prompt\s+override\b`),
		message: "content attempts to override the system prompt",
	},
	{
		key:     "delete-all-files",
		regex:   regexp.MustCompile(`(?i)\bdelete\s+all\s+files\b`),
		message: "content instructs destructive file deletion",
	},
	{
		key:     "rm-rf",
		regex:   regexp.MustCompile(`(?i)\brm\s+-rf\b`),
		message: "content includes a destructive shell command",
	},
	{
		key:     "credential-extraction",
		regex:   regexp.MustCompile(`(?i)\b(?:print|show|reveal|display|output)\s+(?:the\s+|your\s+)?(?:api\s+key|access\s+token|credentials?|secret(?:s)?|password(?:s)?)\b`),
		message: "content attempts to extract credentials",
	},
}

// InstallerOption customizes the installer pipeline.
type InstallerOption func(*Installer)

// Installer handles download, extraction, validation, verification, and the
// final atomic move into place.
type Installer struct {
	downloader          Downloader
	maxArchiveSize      int64
	maxDecompressedSize int64
	maxFileCount        int
	now                 func() time.Time
	tempDirMaxAge       time.Duration
}

type countingReader struct {
	reader io.Reader
	total  int64
}

type installedPackageMetadata struct {
	name          string
	version       string
	manifestPath  string
	verifyContent string
}

type skillManifestHeader struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Version     string         `yaml:"version,omitempty"`
	Metadata    map[string]any `yaml:"metadata,omitempty"`
}

type extensionManifestHeader struct {
	Extension struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"extension"`
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

// NewInstaller constructs a new domain-agnostic install pipeline.
func NewInstaller(dl Downloader, opts ...InstallerOption) *Installer {
	installer := &Installer{
		downloader:          dl,
		maxArchiveSize:      DefaultMaxArchiveSize,
		maxDecompressedSize: DefaultMaxDecompressedSize,
		maxFileCount:        DefaultMaxFileCount,
		now:                 time.Now,
		tempDirMaxAge:       defaultInstallerTempDirMaxAge,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(installer)
		}
	}

	if installer.maxArchiveSize <= 0 {
		installer.maxArchiveSize = DefaultMaxArchiveSize
	}
	if installer.maxDecompressedSize <= 0 {
		installer.maxDecompressedSize = DefaultMaxDecompressedSize
	}
	if installer.maxFileCount <= 0 {
		installer.maxFileCount = DefaultMaxFileCount
	}
	if installer.now == nil {
		installer.now = time.Now
	}
	if installer.tempDirMaxAge <= 0 {
		installer.tempDirMaxAge = defaultInstallerTempDirMaxAge
	}

	return installer
}

// WithInstallerMaxArchiveSize overrides the compressed archive limit.
func WithInstallerMaxArchiveSize(size int64) InstallerOption {
	return func(installer *Installer) {
		installer.maxArchiveSize = size
	}
}

// WithInstallerMaxDecompressedSize overrides the extracted payload limit.
func WithInstallerMaxDecompressedSize(size int64) InstallerOption {
	return func(installer *Installer) {
		installer.maxDecompressedSize = size
	}
}

// WithInstallerMaxFileCount overrides the extracted file-count limit.
func WithInstallerMaxFileCount(count int) InstallerOption {
	return func(installer *Installer) {
		installer.maxFileCount = count
	}
}

// WithInstallerNow overrides the clock used for stale-temp cleanup.
func WithInstallerNow(now func() time.Time) InstallerOption {
	return func(installer *Installer) {
		installer.now = now
	}
}

// WithInstallerTempDirMaxAge overrides the stale-temp cleanup threshold.
func WithInstallerTempDirMaxAge(age time.Duration) InstallerOption {
	return func(installer *Installer) {
		installer.tempDirMaxAge = age
	}
}

// Install downloads, extracts, verifies, and atomically moves a package into
// its final target directory.
func (i *Installer) Install(ctx context.Context, slug string, dlOpts DownloadOpts, targetDir string) (_ *InstallResult, err error) {
	if err := checkMultiRegistryContext(ctx); err != nil {
		return nil, err
	}
	if i == nil {
		return nil, errors.New("registry: installer is required")
	}
	if i.downloader == nil {
		return nil, errors.New("registry: downloader is required")
	}

	trimmedSlug := strings.TrimSpace(slug)
	if trimmedSlug == "" {
		return nil, errors.New("registry: slug is required")
	}

	trimmedTarget := strings.TrimSpace(targetDir)
	if trimmedTarget == "" {
		return nil, errors.New("registry: target directory is required")
	}

	absTarget, err := filepath.Abs(trimmedTarget)
	if err != nil {
		return nil, fmt.Errorf("registry: resolve target directory %q: %w", trimmedTarget, err)
	}
	installParent := filepath.Dir(absTarget)
	if err := os.MkdirAll(installParent, 0o755); err != nil {
		return nil, fmt.Errorf("registry: create install parent %q: %w", installParent, err)
	}
	if err := i.cleanupStaleTempDirs(installParent); err != nil {
		return nil, err
	}

	tempRoot, err := os.MkdirTemp(installParent, defaultInstallerTempDirPattern)
	if err != nil {
		return nil, fmt.Errorf("registry: create temporary install directory: %w", err)
	}
	defer func() {
		removeErr := os.RemoveAll(tempRoot)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = joinInstallerError(err, fmt.Errorf("registry: remove temporary install directory %q: %w", tempRoot, removeErr))
		}
	}()

	download, err := i.downloader.Download(ctx, trimmedSlug, dlOpts)
	if err != nil {
		return nil, err
	}
	if download == nil {
		return nil, fmt.Errorf("registry: download returned no result for %q", trimmedSlug)
	}
	if download.Reader == nil {
		return nil, fmt.Errorf("registry: download returned no archive stream for %q", trimmedSlug)
	}
	defer func() {
		err = joinInstallerError(err, closeDownloadReader(download.Reader, trimmedSlug))
	}()

	if err := validateDownloadContentType(download.ContentType); err != nil {
		return nil, err
	}

	extractRoot := filepath.Join(tempRoot, "extract")
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return nil, fmt.Errorf("registry: create extraction root %q: %w", extractRoot, err)
	}

	compressedReader := &countingReader{
		reader: io.LimitReader(download.Reader, i.maxArchiveSize),
	}
	if err := extractArchive(compressedReader, extractRoot, extractLimits{
		maxDecompressedSize: i.maxDecompressedSize,
		maxFileCount:        i.maxFileCount,
	}); err != nil {
		return nil, normalizeExtractionError(err, compressedReader.total, i.maxArchiveSize)
	}

	packageRoot, metadata, err := loadInstalledPackageMetadata(extractRoot)
	if err != nil {
		return nil, err
	}
	if err := verifyInstallerContent(metadata.verifyContent); err != nil {
		return nil, err
	}

	if err := MoveInstalledDir(packageRoot, absTarget, true); err != nil {
		return nil, err
	}

	checksum, err := computeInstallChecksum(absTarget)
	if err != nil {
		return nil, err
	}

	return &InstallResult{
		Slug:        firstNonEmpty(download.Slug, trimmedSlug),
		Name:        metadata.name,
		Version:     firstNonEmpty(download.Version, metadata.version),
		InstallPath: absTarget,
		Checksum:    checksum,
	}, nil
}

func (i *Installer) cleanupStaleTempDirs(parent string) error {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return fmt.Errorf("registry: read install parent %q: %w", parent, err)
	}

	cutoff := i.now().Add(-i.tempDirMaxAge)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, ".agh-install-") {
			continue
		}

		fullPath := filepath.Join(parent, name)
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("registry: inspect temporary install directory %q: %w", fullPath, err)
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("registry: remove stale temporary install directory %q: %w", fullPath, err)
		}
	}

	return nil
}

func validateDownloadContentType(contentType string) error {
	trimmed := strings.TrimSpace(contentType)
	if trimmed == "" {
		return fmt.Errorf("%w: missing Content-Type", errUnexpectedContentType)
	}

	mediaType, _, err := mime.ParseMediaType(trimmed)
	if err != nil {
		return fmt.Errorf("%w: parse %q: %v", errUnexpectedContentType, trimmed, err)
	}

	switch mediaType {
	case "application/gzip", "application/x-gzip", "application/octet-stream":
		return nil
	default:
		return fmt.Errorf(
			"%w: got %q, want application/gzip, application/x-gzip, or application/octet-stream",
			errUnexpectedContentType,
			trimmed,
		)
	}
}

func normalizeExtractionError(err error, compressedSize int64, maxArchiveSize int64) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errArchiveTooLarge) || errors.Is(err, errArchiveTooManyFiles) {
		return err
	}
	if compressedSize >= maxArchiveSize && (errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)) {
		return fmt.Errorf("%w: limit=%d", errArchiveTooLargeCompressed, maxArchiveSize)
	}
	return err
}

func loadInstalledPackageMetadata(extractRoot string) (string, installedPackageMetadata, error) {
	packageRoot, manifestPath, err := locateInstallManifestRoot(extractRoot)
	if err != nil {
		return "", installedPackageMetadata{}, err
	}

	metadata, err := parseInstalledPackageMetadata(manifestPath)
	if err != nil {
		return "", installedPackageMetadata{}, err
	}
	metadata.manifestPath = manifestPath

	if strings.TrimSpace(metadata.name) == "" {
		return "", installedPackageMetadata{}, fmt.Errorf("registry: manifest %q is missing name", manifestPath)
	}

	return packageRoot, metadata, nil
}

func locateInstallManifestRoot(extractRoot string) (string, string, error) {
	current := extractRoot
	for {
		manifestPath, err := manifestPathAtRoot(current)
		if err != nil {
			return "", "", err
		}
		if manifestPath != "" {
			return current, manifestPath, nil
		}

		entries, err := os.ReadDir(current)
		if err != nil {
			return "", "", fmt.Errorf("registry: read extracted root %q: %w", current, err)
		}

		dirs := make([]string, 0, len(entries))
		files := 0
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, entry.Name())
				continue
			}
			files++
		}

		if len(dirs) == 1 && files == 0 {
			current = filepath.Join(current, dirs[0])
			continue
		}

		return "", "", fmt.Errorf("%w: %q", errInstallMissingManifest, extractRoot)
	}
}

func manifestPathAtRoot(root string) (string, error) {
	extensionManifest := filepath.Join(root, installerExtensionManifestName)
	skillManifest := filepath.Join(root, installerSkillManifestName)

	hasExtensionManifest, err := manifestFileExists(extensionManifest)
	if err != nil {
		return "", fmt.Errorf("registry: inspect manifest %q: %w", extensionManifest, err)
	}
	hasSkillManifest, err := manifestFileExists(skillManifest)
	if err != nil {
		return "", fmt.Errorf("registry: inspect manifest %q: %w", skillManifest, err)
	}

	switch {
	case hasExtensionManifest && hasSkillManifest:
		return "", fmt.Errorf("registry: archive root %q contains both %s and %s", root, installerExtensionManifestName, installerSkillManifestName)
	case hasExtensionManifest:
		return extensionManifest, nil
	case hasSkillManifest:
		return skillManifest, nil
	default:
		return "", nil
	}
}

func parseInstalledPackageMetadata(manifestPath string) (installedPackageMetadata, error) {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return installedPackageMetadata{}, fmt.Errorf("registry: read manifest %q: %w", manifestPath, err)
	}

	switch filepath.Base(manifestPath) {
	case installerSkillManifestName:
		var meta skillManifestHeader
		parts, err := frontmatter.Split(content)
		if err != nil {
			return installedPackageMetadata{}, fmt.Errorf("registry: parse skill manifest %q: %w", manifestPath, err)
		}
		if err := yaml.Unmarshal(parts.Metadata, &meta); err != nil {
			return installedPackageMetadata{}, fmt.Errorf("registry: decode skill manifest %q: %w", manifestPath, err)
		}
		return installedPackageMetadata{
			name:          strings.TrimSpace(meta.Name),
			version:       strings.TrimSpace(meta.Version),
			verifyContent: parts.Body,
		}, nil
	case installerExtensionManifestName:
		var meta extensionManifestHeader
		if _, err := toml.Decode(string(content), &meta); err != nil {
			return installedPackageMetadata{}, fmt.Errorf("registry: decode extension manifest %q: %w", manifestPath, err)
		}
		return installedPackageMetadata{
			name:          firstNonEmpty(meta.Name, meta.Extension.Name),
			version:       firstNonEmpty(meta.Version, meta.Extension.Version),
			verifyContent: string(content),
		}, nil
	default:
		return installedPackageMetadata{}, fmt.Errorf("registry: unsupported manifest %q", manifestPath)
	}
}

func verifyInstallerContent(content string) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}

	messages := make([]string, 0, len(installerVerificationRules))
	seen := make(map[string]struct{}, len(installerVerificationRules))

	for _, rule := range installerVerificationRules {
		if !rule.regex.MatchString(trimmed) {
			continue
		}
		if _, ok := seen[rule.key]; ok {
			continue
		}
		seen[rule.key] = struct{}{}
		messages = append(messages, rule.message)
	}

	if len(messages) == 0 {
		return nil
	}

	slices.Sort(messages)
	return fmt.Errorf("%w: %s", errVerificationBlocked, strings.Join(messages, "; "))
}

func computeInstallChecksum(path string) (string, error) {
	root := strings.TrimSpace(path)
	if root == "" {
		return "", errors.New("registry: install directory is required")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("registry: resolve install directory %q: %w", path, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("registry: stat install directory %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("registry: install directory %q is not a directory", absRoot)
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
		return "", fmt.Errorf("registry: walk install directory %q: %w", absRoot, err)
	}

	slices.Sort(entries)
	hasher := sha256.New()
	for _, relPath := range entries {
		if err := writeInstallChecksumEntry(hasher, absRoot, relPath); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func writeInstallChecksumEntry(hasher hash.Hash, root string, relPath string) error {
	normalizedPath := filepath.ToSlash(relPath)
	absPath := filepath.Join(root, relPath)

	info, err := os.Lstat(absPath)
	if err != nil {
		return fmt.Errorf("registry: stat checksum path %q: %w", absPath, err)
	}

	if info.Mode().IsRegular() {
		file, err := os.Open(absPath)
		if err != nil {
			return fmt.Errorf("registry: open checksum path %q: %w", absPath, err)
		}
		if err := writeInstallChecksumString(hasher, fmt.Sprintf("file:%s\nmode:%#o\n", normalizedPath, info.Mode().Perm())); err != nil {
			closeErr := file.Close()
			if closeErr != nil {
				return errors.Join(err, fmt.Errorf("registry: close checksum path %q: %w", absPath, closeErr))
			}
			return err
		}
		if _, err := io.Copy(hasher, file); err != nil {
			copyErr := fmt.Errorf("registry: hash regular file %q: %w", absPath, err)
			if closeErr := file.Close(); closeErr != nil {
				copyErr = errors.Join(copyErr, fmt.Errorf("registry: close checksum path %q after read failure: %w", absPath, closeErr))
			}
			return copyErr
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("registry: close checksum path %q: %w", absPath, err)
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return fmt.Errorf("registry: hash separator for %q: %w", absPath, err)
		}
		return nil
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absPath)
		if err != nil {
			return fmt.Errorf("registry: read checksum symlink %q: %w", absPath, err)
		}
		normalizedTarget := filepath.ToSlash(filepath.Clean(target))
		return writeInstallChecksumString(
			hasher,
			fmt.Sprintf("symlink:%s\nmode:%#o\ntarget:%s\n", normalizedPath, info.Mode().Perm(), normalizedTarget),
		)
	}

	return fmt.Errorf("registry: unsupported file type in install payload %q", absPath)
}

func writeInstallChecksumString(hasher hash.Hash, value string) error {
	if _, err := hasher.Write([]byte(value)); err != nil {
		return fmt.Errorf("registry: hash payload metadata: %w", err)
	}
	return nil
}

func manifestFileExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	switch {
	case err == nil:
		if info.Mode().IsRegular() {
			return true, nil
		}
		return false, fmt.Errorf("manifest %q must be a regular file", path)
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

func closeDownloadReader(reader io.ReadCloser, slug string) error {
	if reader == nil {
		return nil
	}
	if err := reader.Close(); err != nil {
		return fmt.Errorf("registry: close download stream for %q: %w", slug, err)
	}
	return nil
}

func joinInstallerError(base error, extra error) error {
	if extra == nil {
		return base
	}
	if base == nil {
		return extra
	}
	return errors.Join(base, extra)
}

func (r *countingReader) Read(p []byte) (int, error) {
	if r == nil {
		return 0, errors.New("registry: counting reader is required")
	}
	n, err := r.reader.Read(p)
	r.total += int64(n)
	return n, err
}
