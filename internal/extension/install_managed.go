package extensionpkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	registrypkg "github.com/pedronauck/agh/internal/registry"
)

const managedInstallDirName = "extensions"
const invalidManagedInstallName = "_invalid-extension-name"

type installPackageManifest struct {
	Dependencies         map[string]string `json:"dependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

type managedInstallRegistry interface {
	Get(name string) (*ExtensionInfo, error)
	Install(manifest *Manifest, path string, checksum string, opts ...InstallOption) error
}

// ManagedInstallRoot returns the AGH-managed root directory for installed extensions.
func ManagedInstallRoot(homePaths aghconfig.HomePaths) string {
	return filepath.Join(strings.TrimSpace(homePaths.HomeDir), managedInstallDirName)
}

// ManagedInstallPath returns the AGH-managed directory for one installed extension.
func ManagedInstallPath(homePaths aghconfig.HomePaths, name string) string {
	path, err := ManagedInstallPathChecked(homePaths, name)
	if err != nil {
		return filepath.Join(ManagedInstallRoot(homePaths), invalidManagedInstallName)
	}
	return path
}

// ManagedInstallPathChecked returns the contained managed directory for one installed extension.
func ManagedInstallPathChecked(homePaths aghconfig.HomePaths, name string) (string, error) {
	root := filepath.Clean(ManagedInstallRoot(homePaths))
	if strings.TrimSpace(root) == "" || root == managedInstallDirName {
		return "", errors.New("extension: managed install home path is required")
	}

	safeName, err := validateManagedInstallName(name)
	if err != nil {
		return "", err
	}

	finalDir := filepath.Join(root, safeName)
	rel, err := filepath.Rel(root, finalDir)
	if err != nil {
		return "", fmt.Errorf("extension: resolve managed install path for %q: %w", safeName, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("extension: managed extension name %q escapes install root", name)
	}
	return finalDir, nil
}

// NewManagedInstallStagingDir creates an empty staging directory under the managed extension root.
func NewManagedInstallStagingDir(homePaths aghconfig.HomePaths) (string, error) {
	root := ManagedInstallRoot(homePaths)
	if strings.TrimSpace(root) == "" || root == managedInstallDirName {
		return "", errors.New("extension: managed install home path is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("extension: create managed install root %q: %w", root, err)
	}
	return os.MkdirTemp(root, ".agh-extension-stage-*")
}

// InstallLocalManaged copies one local extension into the managed install root and persists the registry record there.
func InstallLocalManaged(
	homePaths aghconfig.HomePaths,
	registry managedInstallRegistry,
	manifest *Manifest,
	sourceDir string,
	checksum string,
	opts ...InstallOption,
) (err error) {
	normalizedChecksum, err := validateManagedInstallInput(registry, manifest, checksum)
	if err != nil {
		return err
	}
	finalDir, err := ManagedInstallPathChecked(homePaths, manifest.Name)
	if err != nil {
		return err
	}

	actualSourceChecksum, err := ComputeDirectoryChecksum(sourceDir)
	if err != nil {
		return fmt.Errorf("extension: compute source checksum %q: %w", sourceDir, err)
	}
	if actualSourceChecksum != normalizedChecksum {
		return &ExtensionChecksumMismatchError{
			ExpectedChecksum: normalizedChecksum,
			ActualChecksum:   actualSourceChecksum,
		}
	}

	stagingDir, err := NewManagedInstallStagingDir(homePaths)
	if err != nil {
		return err
	}

	cleanupStaging := true
	defer func() {
		if !cleanupStaging {
			return
		}
		if removeErr := os.RemoveAll(stagingDir); removeErr != nil {
			err = errors.Join(
				err,
				fmt.Errorf("extension: remove managed install staging dir %q: %w", stagingDir, removeErr),
			)
		}
	}()

	if err := copyInstallTree(sourceDir, stagingDir); err != nil {
		return err
	}

	if err := registrypkg.MoveInstalledDir(stagingDir, finalDir, false); err != nil {
		return fmt.Errorf("extension: move local extension %q into managed install path: %w", manifest.Name, err)
	}
	cleanupStaging = false

	installedChecksum, err := ComputeDirectoryChecksum(finalDir)
	if err != nil {
		return removeManagedInstallOnError(
			finalDir,
			fmt.Errorf("extension: compute installed checksum %q: %w", finalDir, err),
			"after checksum error",
		)
	}

	if err := registry.Install(manifest, finalDir, installedChecksum, opts...); err != nil {
		return removeManagedInstallOnError(
			finalDir,
			fmt.Errorf("extension: persist managed extension %q: %w", manifest.Name, err),
			"",
		)
	}

	return nil
}

func validateManagedInstallInput(
	registry managedInstallRegistry,
	manifest *Manifest,
	checksum string,
) (string, error) {
	if registry == nil {
		return "", errors.New("extension: registry is required")
	}
	if manifest == nil {
		return "", errors.New("extension: manifest is required")
	}
	if _, err := validateManagedInstallName(manifest.Name); err != nil {
		return "", err
	}

	normalizedChecksum := strings.ToLower(strings.TrimSpace(checksum))
	if normalizedChecksum == "" {
		return "", errors.New("extension: checksum is required")
	}

	if _, err := registry.Get(manifest.Name); err == nil {
		return "", &ExtensionExistsError{Name: manifest.Name}
	} else if !errors.Is(err, ErrExtensionNotFound) {
		return "", err
	}

	return normalizedChecksum, nil
}

func validateManagedInstallName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return "", errors.New("extension: managed extension name is required")
	case trimmed == "." || trimmed == "..":
		return "", fmt.Errorf("extension: managed extension name %q is reserved", name)
	case filepath.IsAbs(trimmed):
		return "", fmt.Errorf("extension: managed extension name %q must be relative", name)
	case strings.Contains(trimmed, "/") || strings.Contains(trimmed, `\`):
		return "", fmt.Errorf("extension: managed extension name %q must be a single path segment", name)
	case filepath.Clean(trimmed) != trimmed:
		return "", fmt.Errorf("extension: managed extension name %q is not normalized", name)
	default:
		return trimmed, nil
	}
}

func removeManagedInstallOnError(finalDir string, baseErr error, contextSuffix string) error {
	removeErr := os.RemoveAll(finalDir)
	if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
		return baseErr
	}

	message := fmt.Sprintf("extension: remove failed local install %q", finalDir)
	if strings.TrimSpace(contextSuffix) != "" {
		message += " " + strings.TrimSpace(contextSuffix)
	}
	return errors.Join(baseErr, fmt.Errorf("%s: %w", message, removeErr))
}

func copyInstallTree(sourceDir string, targetDir string) error {
	sourceRoot := strings.TrimSpace(sourceDir)
	if sourceRoot == "" {
		return errors.New("extension: source directory is required")
	}

	absSourceRoot, err := filepath.Abs(sourceRoot)
	if err != nil {
		return fmt.Errorf("extension: resolve source directory %q: %w", sourceDir, err)
	}
	canonicalSourceRoot, err := canonicalizeInstallPath(absSourceRoot)
	if err != nil {
		return fmt.Errorf("extension: canonicalize source directory %q: %w", absSourceRoot, err)
	}

	info, err := os.Stat(absSourceRoot)
	if err != nil {
		return fmt.Errorf("extension: stat source directory %q: %w", absSourceRoot, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("extension: source directory %q is not a directory", absSourceRoot)
	}

	if strings.TrimSpace(targetDir) == "" {
		return errors.New("extension: target directory is required")
	}
	if err := os.MkdirAll(targetDir, info.Mode().Perm()); err != nil {
		return fmt.Errorf("extension: create target directory %q: %w", targetDir, err)
	}
	if err := os.Chmod(targetDir, info.Mode().Perm()); err != nil {
		return fmt.Errorf("extension: set target directory mode %q: %w", targetDir, err)
	}

	return copyInstallDirectoryContents(canonicalSourceRoot, absSourceRoot, targetDir, map[string]struct{}{
		canonicalSourceRoot: {},
	})
}

func copyInstallDirectoryContents(
	sourceRoot string,
	sourceDir string,
	targetDir string,
	activeDirs map[string]struct{},
) error {
	runtimeDeps, hasPackageManifest, err := loadInstallRuntimeDependencies(sourceDir)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("extension: read source directory %q: %w", sourceDir, err)
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())
		targetPath := filepath.Join(targetDir, entry.Name())
		if hasPackageManifest && entry.Name() == "node_modules" {
			if err := copyInstallNodeModules(sourceRoot, sourcePath, targetPath, activeDirs, runtimeDeps); err != nil {
				return err
			}
			continue
		}
		if err := copyInstallEntry(sourceRoot, sourcePath, targetPath, activeDirs); err != nil {
			return err
		}
	}

	return nil
}

func loadInstallRuntimeDependencies(sourceDir string) (map[string]struct{}, bool, error) {
	manifestPath := filepath.Join(sourceDir, "package.json")
	info, err := os.Stat(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("extension: stat package manifest %q: %w", manifestPath, err)
	}
	if info.IsDir() {
		return nil, false, fmt.Errorf("extension: package manifest %q is a directory", manifestPath)
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, false, fmt.Errorf("extension: read package manifest %q: %w", manifestPath, err)
	}

	var manifest installPackageManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, false, fmt.Errorf("extension: decode package manifest %q: %w", manifestPath, err)
	}

	runtimeDeps := make(map[string]struct{}, len(manifest.Dependencies)+len(manifest.OptionalDependencies))
	for name := range manifest.Dependencies {
		name = strings.TrimSpace(name)
		if name != "" {
			runtimeDeps[name] = struct{}{}
		}
	}
	for name := range manifest.OptionalDependencies {
		name = strings.TrimSpace(name)
		if name != "" {
			runtimeDeps[name] = struct{}{}
		}
	}

	return runtimeDeps, true, nil
}

func copyInstallNodeModules(
	sourceRoot string,
	sourceDir string,
	targetDir string,
	activeDirs map[string]struct{},
	runtimeDeps map[string]struct{},
) error {
	if len(runtimeDeps) == 0 {
		return nil
	}

	names := make([]string, 0, len(runtimeDeps))
	for name := range runtimeDeps {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		sourcePath, err := installNodeModulePath(sourceDir, name)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(sourcePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("extension: runtime dependency %q missing from %q", name, sourceDir)
			}
			return fmt.Errorf("extension: stat runtime dependency %q in %q: %w", name, sourceDir, err)
		}
		targetPath := filepath.Join(targetDir, filepath.FromSlash(name))
		if err := copyInstallRuntimeDependency(sourceRoot, sourcePath, targetPath, activeDirs); err != nil {
			return err
		}
	}

	return nil
}

func installNodeModulePath(nodeModulesDir string, packageName string) (string, error) {
	name := strings.TrimSpace(packageName)
	if name == "" {
		return "", errors.New("extension: runtime dependency name is required")
	}
	if filepath.IsAbs(name) || strings.Contains(name, "\\") {
		return "", fmt.Errorf("extension: invalid runtime dependency name %q", packageName)
	}

	parts := strings.Split(name, "/")
	switch {
	case len(parts) == 1:
		if !validInstallPackageSegment(parts[0], false) {
			return "", fmt.Errorf("extension: invalid runtime dependency name %q", packageName)
		}
	case len(parts) == 2 && strings.HasPrefix(parts[0], "@"):
		if !validInstallPackageSegment(parts[0], true) || !validInstallPackageSegment(parts[1], false) {
			return "", fmt.Errorf("extension: invalid runtime dependency name %q", packageName)
		}
	default:
		return "", fmt.Errorf("extension: invalid runtime dependency name %q", packageName)
	}

	return filepath.Join(nodeModulesDir, filepath.FromSlash(name)), nil
}

func validInstallPackageSegment(segment string, scoped bool) bool {
	if scoped {
		return len(segment) > 1 && segment != "." && segment != ".." && !strings.Contains(segment, "/") &&
			!strings.Contains(segment, "\\")
	}
	return segment != "" && segment != "." && segment != ".." && !strings.Contains(segment, "/") &&
		!strings.Contains(segment, "\\")
}

func copyInstallRuntimeDependency(
	sourceRoot string,
	sourcePath string,
	targetPath string,
	activeDirs map[string]struct{},
) error {
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("extension: stat runtime dependency %q: %w", sourcePath, err)
	}

	switch {
	case info.IsDir():
		return copyInstallPackageRoot(sourcePath, sourcePath, targetPath, activeDirs)
	case info.Mode()&os.ModeSymlink != 0:
		resolvedPath, err := filepath.EvalSymlinks(sourcePath)
		if err != nil {
			return fmt.Errorf("extension: resolve runtime dependency symlink %q: %w", sourcePath, err)
		}
		if err := ensureInstallPathWithinRoot(sourceRoot, resolvedPath); err != nil {
			return fmt.Errorf("extension: reject runtime dependency symlink %q: %w", sourcePath, err)
		}
		resolvedInfo, err := os.Stat(resolvedPath)
		if err != nil {
			return fmt.Errorf("extension: stat runtime dependency target %q: %w", resolvedPath, err)
		}
		switch {
		case resolvedInfo.IsDir():
			return copyInstallPackageRoot(sourcePath, resolvedPath, targetPath, activeDirs)
		case resolvedInfo.Mode().IsRegular():
			return copyInstallFile(resolvedPath, targetPath, resolvedInfo.Mode().Perm())
		default:
			return fmt.Errorf("extension: unsupported runtime dependency target type for %q", sourcePath)
		}
	case info.Mode().IsRegular():
		return copyInstallFile(sourcePath, targetPath, info.Mode().Perm())
	default:
		return fmt.Errorf("extension: unsupported runtime dependency type for %q", sourcePath)
	}
}

func copyInstallPackageRoot(
	sourcePath string,
	sourceDir string,
	targetDir string,
	activeDirs map[string]struct{},
) error {
	absSourceDir, err := filepath.Abs(strings.TrimSpace(sourceDir))
	if err != nil {
		return fmt.Errorf("extension: resolve package root %q: %w", sourceDir, err)
	}
	canonicalSourceRoot, err := canonicalizeInstallPath(absSourceDir)
	if err != nil {
		return fmt.Errorf("extension: canonicalize package root %q: %w", absSourceDir, err)
	}

	info, err := os.Stat(absSourceDir)
	if err != nil {
		return fmt.Errorf("extension: stat package root %q: %w", absSourceDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("extension: package root %q is not a directory", absSourceDir)
	}

	nextActiveDirs, err := pushInstallCopyDir(activeDirs, absSourceDir, sourcePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(targetDir, info.Mode().Perm()); err != nil {
		return fmt.Errorf("extension: create package target directory %q: %w", targetDir, err)
	}
	if err := os.Chmod(targetDir, info.Mode().Perm()); err != nil {
		return fmt.Errorf("extension: set package target directory mode %q: %w", targetDir, err)
	}

	return copyInstallDirectoryContents(canonicalSourceRoot, absSourceDir, targetDir, nextActiveDirs)
}

func copyInstallEntry(sourceRoot string, sourcePath string, targetPath string, activeDirs map[string]struct{}) error {
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("extension: stat source path %q: %w", sourcePath, err)
	}

	switch {
	case info.IsDir():
		nextActiveDirs, err := pushInstallCopyDir(activeDirs, sourcePath, sourcePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("extension: create target directory %q: %w", targetPath, err)
		}
		if err := os.Chmod(targetPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("extension: set target directory mode %q: %w", targetPath, err)
		}
		return copyInstallDirectoryContents(sourceRoot, sourcePath, targetPath, nextActiveDirs)
	case info.Mode().IsRegular():
		return copyInstallFile(sourcePath, targetPath, info.Mode().Perm())
	case info.Mode()&os.ModeSymlink != 0:
		return copyInstallSymlink(sourceRoot, sourcePath, targetPath, activeDirs)
	default:
		return fmt.Errorf("extension: unsupported file type in extension payload %q", sourcePath)
	}
}

func copyInstallFile(sourcePath string, targetPath string, perm os.FileMode) (err error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("extension: create target file parent %q: %w", filepath.Dir(targetPath), err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("extension: open source file %q: %w", sourcePath, err)
	}
	defer func() {
		if closeErr := source.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("extension: close source file %q: %w", sourcePath, closeErr)
		}
	}()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("extension: create target file %q: %w", targetPath, err)
	}
	defer func() {
		if closeErr := target.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("extension: close target file %q: %w", targetPath, closeErr)
		}
	}()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("extension: copy file %q to %q: %w", sourcePath, targetPath, err)
	}
	if err := target.Chmod(perm); err != nil {
		return fmt.Errorf("extension: set target file mode %q: %w", targetPath, err)
	}

	return nil
}

func copyInstallSymlink(sourceRoot string, sourcePath string, targetPath string, activeDirs map[string]struct{}) error {
	resolvedPath, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		return fmt.Errorf("extension: resolve source symlink %q: %w", sourcePath, err)
	}
	if err := ensureInstallPathWithinRoot(sourceRoot, resolvedPath); err != nil {
		return fmt.Errorf("extension: reject source symlink %q: %w", sourcePath, err)
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return fmt.Errorf("extension: stat resolved symlink target %q: %w", resolvedPath, err)
	}

	switch {
	case info.IsDir():
		nextActiveDirs, err := pushInstallCopyDir(activeDirs, resolvedPath, sourcePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf(
				"extension: create target directory %q for symlinked source %q: %w",
				targetPath,
				sourcePath,
				err,
			)
		}
		if err := os.Chmod(targetPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf(
				"extension: set target directory mode %q for symlinked source %q: %w",
				targetPath,
				sourcePath,
				err,
			)
		}
		return copyInstallDirectoryContents(sourceRoot, resolvedPath, targetPath, nextActiveDirs)
	case info.Mode().IsRegular():
		return copyInstallFile(resolvedPath, targetPath, info.Mode().Perm())
	default:
		return fmt.Errorf("extension: unsupported symlink target type for %q", sourcePath)
	}
}

func pushInstallCopyDir(
	activeDirs map[string]struct{},
	resolvedPath string,
	sourcePath string,
) (map[string]struct{}, error) {
	canonicalPath, err := canonicalizeInstallPath(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("extension: resolve directory %q: %w", resolvedPath, err)
	}
	if _, exists := activeDirs[canonicalPath]; exists {
		return nil, fmt.Errorf("extension: symlink directory cycle detected from %q to %q", sourcePath, canonicalPath)
	}

	nextActiveDirs := make(map[string]struct{}, len(activeDirs)+1)
	for path := range activeDirs {
		nextActiveDirs[path] = struct{}{}
	}
	nextActiveDirs[canonicalPath] = struct{}{}
	return nextActiveDirs, nil
}

func ensureInstallPathWithinRoot(sourceRoot string, resolvedPath string) error {
	canonicalRoot, err := canonicalizeInstallPath(sourceRoot)
	if err != nil {
		return fmt.Errorf("resolve source root %q: %w", sourceRoot, err)
	}
	canonicalPath, err := canonicalizeInstallPath(resolvedPath)
	if err != nil {
		return fmt.Errorf("resolve source path %q: %w", resolvedPath, err)
	}

	relToRoot, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil {
		return fmt.Errorf("relate %q to source root %q: %w", canonicalPath, canonicalRoot, err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("symlink target %q escapes source root %q", canonicalPath, canonicalRoot)
	}
	return nil
}

func canonicalizeInstallPath(path string) (string, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absPath)
}
