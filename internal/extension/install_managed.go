package extension

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	registrypkg "github.com/pedronauck/agh/internal/registry"
)

const managedInstallDirName = "extensions"

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
	return filepath.Join(ManagedInstallRoot(homePaths), strings.TrimSpace(name))
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
) error {
	if registry == nil {
		return errors.New("extension: registry is required")
	}
	if manifest == nil {
		return errors.New("extension: manifest is required")
	}
	if strings.TrimSpace(checksum) == "" {
		return errors.New("extension: checksum is required")
	}

	if _, err := registry.Get(manifest.Name); err == nil {
		return &ExtensionExistsError{Name: manifest.Name}
	} else if !errors.Is(err, ErrExtensionNotFound) {
		return err
	}

	actualSourceChecksum, err := ComputeDirectoryChecksum(sourceDir)
	if err != nil {
		return err
	}
	if actualSourceChecksum != checksum {
		return &ExtensionChecksumMismatchError{
			ExpectedChecksum: checksum,
			ActualChecksum:   actualSourceChecksum,
		}
	}

	stagingDir, err := NewManagedInstallStagingDir(homePaths)
	if err != nil {
		return err
	}

	cleanupStaging := true
	defer func() {
		if cleanupStaging {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	if err := copyInstallTree(sourceDir, stagingDir); err != nil {
		return err
	}

	finalDir := ManagedInstallPath(homePaths, manifest.Name)
	if err := registrypkg.MoveInstalledDir(stagingDir, finalDir, false); err != nil {
		return fmt.Errorf("extension: move local extension %q into managed install path: %w", manifest.Name, err)
	}
	cleanupStaging = false

	installedChecksum, err := ComputeDirectoryChecksum(finalDir)
	if err != nil {
		removeErr := os.RemoveAll(finalDir)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("extension: remove failed local install %q after checksum error: %w", finalDir, removeErr))
		}
		return err
	}

	if err := registry.Install(manifest, finalDir, installedChecksum, opts...); err != nil {
		removeErr := os.RemoveAll(finalDir)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("extension: remove failed local install %q: %w", finalDir, removeErr))
		}
		return err
	}

	return nil
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

	return copyInstallDirectoryContents(absSourceRoot, targetDir)
}

func copyInstallDirectoryContents(sourceDir string, targetDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("extension: read source directory %q: %w", sourceDir, err)
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())
		targetPath := filepath.Join(targetDir, entry.Name())
		if err := copyInstallEntry(sourcePath, targetPath); err != nil {
			return err
		}
	}

	return nil
}

func copyInstallEntry(sourcePath string, targetPath string) error {
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("extension: stat source path %q: %w", sourcePath, err)
	}

	switch {
	case info.IsDir():
		if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("extension: create target directory %q: %w", targetPath, err)
		}
		if err := os.Chmod(targetPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("extension: set target directory mode %q: %w", targetPath, err)
		}
		return copyInstallDirectoryContents(sourcePath, targetPath)
	case info.Mode().IsRegular():
		return copyInstallFile(sourcePath, targetPath, info.Mode().Perm())
	case info.Mode()&os.ModeSymlink != 0:
		return copyInstallSymlink(sourcePath, targetPath)
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

func copyInstallSymlink(sourcePath string, targetPath string) error {
	resolvedPath, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		return fmt.Errorf("extension: resolve source symlink %q: %w", sourcePath, err)
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return fmt.Errorf("extension: stat resolved symlink target %q: %w", resolvedPath, err)
	}

	switch {
	case info.IsDir():
		if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("extension: create target directory %q for symlinked source %q: %w", targetPath, sourcePath, err)
		}
		if err := os.Chmod(targetPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("extension: set target directory mode %q for symlinked source %q: %w", targetPath, sourcePath, err)
		}
		return copyInstallDirectoryContents(resolvedPath, targetPath)
	case info.Mode().IsRegular():
		return copyInstallFile(resolvedPath, targetPath, info.Mode().Perm())
	default:
		return fmt.Errorf("extension: unsupported symlink target type for %q", sourcePath)
	}
}
