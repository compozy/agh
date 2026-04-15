package extension

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

var _ managedInstallRegistry = (*recordingManagedInstallRegistry)(nil)

func TestCopyInstallTreeMaterializesSymlinkTargets(t *testing.T) {
	t.Parallel()

	sourceDir := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(source) error = %v", err)
	}

	internalDir := filepath.Join(sourceDir, "vendor", "extension-sdk")
	if err := os.MkdirAll(filepath.Join(internalDir, "bin"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(internal) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(internalDir, "package.json"), []byte("{\"name\":\"@agh/extension-sdk\"}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(package.json) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(internalDir, "bin", "tsc"), []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatalf("os.WriteFile(tsc) error = %v", err)
	}

	if err := os.MkdirAll(filepath.Join(sourceDir, "node_modules", "@agh"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(node_modules/@agh) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sourceDir, "node_modules", ".bin"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(node_modules/.bin) error = %v", err)
	}
	if err := os.Symlink(filepath.Join(sourceDir, "vendor", "extension-sdk"), filepath.Join(sourceDir, "node_modules", "@agh", "extension-sdk")); err != nil {
		t.Skipf("os.Symlink(directory) unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join(sourceDir, "vendor", "extension-sdk", "bin", "tsc"), filepath.Join(sourceDir, "node_modules", ".bin", "tsc")); err != nil {
		t.Skipf("os.Symlink(file) unavailable: %v", err)
	}

	targetDir := filepath.Join(t.TempDir(), "target")
	if err := copyInstallTree(sourceDir, targetDir); err != nil {
		t.Fatalf("copyInstallTree() error = %v", err)
	}

	copiedDir := filepath.Join(targetDir, "node_modules", "@agh", "extension-sdk")
	info, err := os.Lstat(copiedDir)
	if err != nil {
		t.Fatalf("os.Lstat(%q) error = %v", copiedDir, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("copied sdk dir mode = %v, want materialized directory", info.Mode())
	}
	if !info.IsDir() {
		t.Fatalf("copied sdk dir IsDir() = false, want true")
	}
	if _, err := os.Stat(filepath.Join(copiedDir, "package.json")); err != nil {
		t.Fatalf("os.Stat(copied package.json) error = %v", err)
	}

	copiedFile := filepath.Join(targetDir, "node_modules", ".bin", "tsc")
	fileInfo, err := os.Lstat(copiedFile)
	if err != nil {
		t.Fatalf("os.Lstat(%q) error = %v", copiedFile, err)
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("copied tsc mode = %v, want materialized file", fileInfo.Mode())
	}
	if fileInfo.IsDir() {
		t.Fatalf("copied tsc IsDir() = true, want file")
	}
	content, err := os.ReadFile(copiedFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", copiedFile, err)
	}
	if string(content) != "#!/usr/bin/env node\n" {
		t.Fatalf("copied tsc content = %q, want script payload", string(content))
	}
}

func TestInstallLocalManagedUsesInstalledChecksumForMaterializedSymlinks(t *testing.T) {
	t.Parallel()

	sourceDir := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(filepath.Join(sourceDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(source) error = %v", err)
	}

	internalFile := filepath.Join(sourceDir, "vendor", "external.js")
	if err := os.MkdirAll(filepath.Dir(internalFile), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(vendor) error = %v", err)
	}
	if err := os.WriteFile(internalFile, []byte("export const value = 1;\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(internal) error = %v", err)
	}
	if err := os.Symlink(internalFile, filepath.Join(sourceDir, "node_modules", "external.js")); err != nil {
		t.Skipf("os.Symlink(file) unavailable: %v", err)
	}

	sourceChecksum, err := ComputeDirectoryChecksum(sourceDir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(source) error = %v", err)
	}

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	registry := &recordingManagedInstallRegistry{}
	manifest := &Manifest{Name: "symlink-ext"}

	if err := InstallLocalManaged(homePaths, registry, manifest, sourceDir, sourceChecksum); err != nil {
		t.Fatalf("InstallLocalManaged() error = %v", err)
	}

	finalDir := ManagedInstallPath(homePaths, manifest.Name)
	finalChecksum, err := ComputeDirectoryChecksum(finalDir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(final) error = %v", err)
	}
	if got := registry.installedChecksum; got != finalChecksum {
		t.Fatalf("registry installed checksum = %q, want %q", got, finalChecksum)
	}
	if finalChecksum == sourceChecksum {
		t.Fatalf("final checksum = %q, want checksum different from source symlink tree %q", finalChecksum, sourceChecksum)
	}
}

func TestInstallLocalManagedNormalizesProvidedChecksum(t *testing.T) {
	t.Parallel()

	sourceDir := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "extension.toml"), []byte("name = \"checksum-ext\"\nversion = \"1.0.0\"\nmin_agh_version = \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(extension.toml) error = %v", err)
	}

	sourceChecksum, err := ComputeDirectoryChecksum(sourceDir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(source) error = %v", err)
	}

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	registry := &recordingManagedInstallRegistry{}
	manifest := &Manifest{Name: "checksum-ext"}

	if err := InstallLocalManaged(homePaths, registry, manifest, sourceDir, "  "+strings.ToUpper(sourceChecksum)+"  "); err != nil {
		t.Fatalf("InstallLocalManaged(normalized checksum) error = %v", err)
	}
}

func TestCopyInstallTreeRejectsSymlinkDirectoryCycles(t *testing.T) {
	t.Parallel()

	sourceDir := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(source) error = %v", err)
	}
	if err := os.Symlink(".", filepath.Join(sourceDir, "loop")); err != nil {
		t.Skipf("os.Symlink(directory) unavailable: %v", err)
	}

	targetDir := filepath.Join(t.TempDir(), "target")
	err := copyInstallTree(sourceDir, targetDir)
	if err == nil {
		t.Fatal("copyInstallTree() error = nil, want symlink cycle failure")
	}
	if !strings.Contains(err.Error(), "symlink directory cycle detected") {
		t.Fatalf("copyInstallTree() error = %v, want symlink cycle context", err)
	}
}

func TestCopyInstallTreeRejectsSymlinkTargetsOutsideSourceRoot(t *testing.T) {
	t.Parallel()

	t.Run("ShouldRejectExternalDirectoryTargets", func(t *testing.T) {
		t.Parallel()

		sourceDir := filepath.Join(t.TempDir(), "source")
		if err := os.MkdirAll(filepath.Join(sourceDir, "node_modules"), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(source) error = %v", err)
		}

		externalDir := filepath.Join(t.TempDir(), "external-sdk")
		if err := os.MkdirAll(externalDir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll(external) error = %v", err)
		}
		if err := os.Symlink(externalDir, filepath.Join(sourceDir, "node_modules", "sdk")); err != nil {
			t.Skipf("os.Symlink(directory) unavailable: %v", err)
		}

		err := copyInstallTree(sourceDir, filepath.Join(t.TempDir(), "target"))
		if err == nil {
			t.Fatal("copyInstallTree() error = nil, want symlink escape failure")
		}
		if !strings.Contains(err.Error(), "escapes source root") {
			t.Fatalf("copyInstallTree() error = %v, want escape context", err)
		}
	})

	t.Run("ShouldRejectExternalFileTargets", func(t *testing.T) {
		t.Parallel()

		sourceDir := filepath.Join(t.TempDir(), "source")
		if err := os.MkdirAll(filepath.Join(sourceDir, "node_modules"), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(source) error = %v", err)
		}

		externalFile := filepath.Join(t.TempDir(), "external.js")
		if err := os.WriteFile(externalFile, []byte("export const value = 1;\n"), 0o644); err != nil {
			t.Fatalf("os.WriteFile(external) error = %v", err)
		}
		if err := os.Symlink(externalFile, filepath.Join(sourceDir, "node_modules", "external.js")); err != nil {
			t.Skipf("os.Symlink(file) unavailable: %v", err)
		}

		err := copyInstallTree(sourceDir, filepath.Join(t.TempDir(), "target"))
		if err == nil {
			t.Fatal("copyInstallTree() error = nil, want symlink escape failure")
		}
		if !strings.Contains(err.Error(), "escapes source root") {
			t.Fatalf("copyInstallTree() error = %v, want escape context", err)
		}
	})
}

func TestInstallLocalManagedWrapsPhaseErrors(t *testing.T) {
	t.Parallel()

	t.Run("ShouldWrapSourceChecksumFailures", func(t *testing.T) {
		t.Parallel()

		homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		err = InstallLocalManaged(
			homePaths,
			&recordingManagedInstallRegistry{},
			&Manifest{Name: "missing-ext"},
			filepath.Join(t.TempDir(), "missing"),
			"checksum",
		)
		if err == nil || !strings.Contains(err.Error(), "extension: compute source checksum") {
			t.Fatalf("InstallLocalManaged() error = %v, want wrapped source checksum failure", err)
		}
	})

	t.Run("ShouldWrapRegistryInstallFailures", func(t *testing.T) {
		t.Parallel()

		sourceDir := filepath.Join(t.TempDir(), "source")
		if err := os.MkdirAll(sourceDir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll(source) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, "extension.toml"), []byte("name = \"wrapped-ext\"\nversion = \"1.0.0\"\nmin_agh_version = \"0.1.0\"\n"), 0o644); err != nil {
			t.Fatalf("os.WriteFile(extension.toml) error = %v", err)
		}

		sourceChecksum, err := ComputeDirectoryChecksum(sourceDir)
		if err != nil {
			t.Fatalf("ComputeDirectoryChecksum(source) error = %v", err)
		}
		homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		registry := &recordingManagedInstallRegistry{installErr: errors.New("registry boom")}
		err = InstallLocalManaged(homePaths, registry, &Manifest{Name: "wrapped-ext"}, sourceDir, sourceChecksum)
		if err == nil || !strings.Contains(err.Error(), `extension: persist managed extension "wrapped-ext"`) {
			t.Fatalf("InstallLocalManaged() error = %v, want wrapped registry install failure", err)
		}
	})
}

type recordingManagedInstallRegistry struct {
	installedChecksum string
	installErr        error
}

func (*recordingManagedInstallRegistry) Get(string) (*ExtensionInfo, error) {
	return nil, ErrExtensionNotFound
}

func (r *recordingManagedInstallRegistry) Install(_ *Manifest, _ string, checksum string, _ ...InstallOption) error {
	r.installedChecksum = checksum
	if r.installErr != nil {
		return r.installErr
	}
	return nil
}
