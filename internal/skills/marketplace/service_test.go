package marketplace

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	registrypkg "github.com/pedronauck/agh/internal/registry"
	"github.com/pedronauck/agh/internal/skills"
)

func timeNowForTest() time.Time {
	return time.Date(2026, time.May, 17, 12, 0, 0, 0, time.UTC)
}

type fakeInstallRegistry struct {
	archive        []byte
	detail         *registrypkg.Detail
	updateInfo     *registrypkg.UpdateInfo
	checkUpdateErr error
}

func (r fakeInstallRegistry) Download(
	context.Context,
	string,
	registrypkg.DownloadOpts,
) (*registrypkg.DownloadResult, error) {
	return &registrypkg.DownloadResult{
		Reader:      io.NopCloser(bytes.NewReader(r.archive)),
		Slug:        r.detail.Slug,
		Version:     r.detail.Version,
		ContentType: "application/gzip",
	}, nil
}

func (r fakeInstallRegistry) Info(context.Context, string) (*registrypkg.Detail, error) {
	return r.detail, nil
}

func (r fakeInstallRegistry) CheckUpdate(
	context.Context,
	string,
	string,
) (*registrypkg.UpdateInfo, error) {
	if r.checkUpdateErr != nil {
		return nil, r.checkUpdateErr
	}
	return r.updateInfo, nil
}

func TestPathInsideRoot(t *testing.T) {
	t.Parallel()

	t.Run("Should reject targets that escape the root through symlinks", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		outside := t.TempDir()
		linkPath := filepath.Join(root, "escape")
		if err := os.Symlink(outside, linkPath); err != nil {
			t.Fatalf("os.Symlink() error = %v", err)
		}
		outsideSkill := filepath.Join(outside, "SKILL.md")
		if err := os.WriteFile(outsideSkill, []byte("outside"), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		_, err := PathInsideRoot(root, filepath.Join(linkPath, "SKILL.md"))
		if !errors.Is(err, registrypkg.ErrPathOutsideRoot) {
			t.Fatalf("PathInsideRoot() error = %v, want ErrPathOutsideRoot", err)
		}
	})

	t.Run("Should preserve lexical targets that stay within the root", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		targetDir := filepath.Join(root, "review")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		targetPath := filepath.Join(targetDir, "SKILL.md")
		if err := os.WriteFile(targetPath, []byte("inside"), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		resolved, err := PathInsideRoot(root, targetPath)
		if err != nil {
			t.Fatalf("PathInsideRoot() error = %v", err)
		}
		if got := resolved; got != targetPath {
			t.Fatalf("PathInsideRoot() = %q, want %q", got, targetPath)
		}
	})

	t.Run("Should allow missing targets beneath the resolved root", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		targetPath := filepath.Join(root, "review", "SKILL.md")

		resolved, err := PathInsideRoot(root, targetPath)
		if err != nil {
			t.Fatalf("PathInsideRoot() error = %v", err)
		}
		if got := resolved; got != targetPath {
			t.Fatalf("PathInsideRoot() = %q, want %q", got, targetPath)
		}
	})

	t.Run("Should reject blank roots", func(t *testing.T) {
		t.Parallel()

		_, err := PathInsideRoot("   ", "review/SKILL.md")
		if !errors.Is(err, registrypkg.ErrPathRootRequired) {
			t.Fatalf("PathInsideRoot() error = %v, want ErrPathRootRequired", err)
		}
	})

	t.Run("Should reject percent-encoded traversal paths", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()

		_, err := PathInsideRoot(root, filepath.Join(root, "%2e%2e", "escape", "SKILL.md"))
		if !errors.Is(err, registrypkg.ErrPathOutsideRoot) {
			t.Fatalf("PathInsideRoot() error = %v, want ErrPathOutsideRoot", err)
		}
	})
}

func TestInstallWithRegistryRejectsUnsafeTargets(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve manual skill directories on marketplace name collision", func(t *testing.T) {
		t.Parallel()

		skillsDir := t.TempDir()
		manualSkill := filepath.Join(skillsDir, "review")
		manualBody := "---\nname: review\ndescription: Manual skill\n---\nmanual body\n"
		if err := os.MkdirAll(manualSkill, 0o755); err != nil {
			t.Fatalf("MkdirAll(manual skill) error = %v", err)
		}
		if err := os.WriteFile(
			filepath.Join(manualSkill, SkillMarkdownFileName),
			[]byte(manualBody),
			0o644,
		); err != nil {
			t.Fatalf("WriteFile(manual skill) error = %v", err)
		}

		registry := fakeInstallRegistry{
			archive: marketplaceSkillArchive(t, "review", "Marketplace skill", "marketplace body"),
			detail: &registrypkg.Detail{
				Listing: registrypkg.Listing{
					Slug:    "@acme/review",
					Name:    "review",
					Version: "2.0.0",
					Source:  "test-registry",
				},
			},
		}

		_, err := InstallWithRegistry(
			context.Background(),
			skillsDir,
			registry,
			"@acme/review",
			"",
			"",
			timeNowForTest,
		)
		if !errors.Is(err, ErrNotMarketplace) {
			t.Fatalf("InstallWithRegistry() error = %v, want ErrNotMarketplace", err)
		}
		body, readErr := os.ReadFile(filepath.Join(manualSkill, SkillMarkdownFileName))
		if readErr != nil {
			t.Fatalf("ReadFile(manual skill) error = %v", readErr)
		}
		if got := string(body); got != manualBody {
			t.Fatalf("manual skill body = %q, want %q", got, manualBody)
		}
	})

	t.Run("Should reject downloaded manifest names that cannot be managed locally", func(t *testing.T) {
		t.Parallel()

		registry := fakeInstallRegistry{
			archive: marketplaceSkillArchive(t, "parent/review", "Bad skill", "bad body"),
			detail: &registrypkg.Detail{
				Listing: registrypkg.Listing{
					Slug:    "@acme/review",
					Name:    "review",
					Version: "1.0.0",
					Source:  "test-registry",
				},
			},
		}

		_, err := InstallWithRegistry(
			context.Background(),
			t.TempDir(),
			registry,
			"@acme/review",
			"",
			"",
			timeNowForTest,
		)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("InstallWithRegistry() error = %v, want ErrValidation", err)
		}
	})
}

func TestUpdateSkillClassifiesRegistryLookupFailures(t *testing.T) {
	t.Parallel()

	t.Run("Should map missing remote package during update to marketplace not found", func(t *testing.T) {
		t.Parallel()

		installed := InstalledSkill{
			Name: "review",
			Dir:  t.TempDir(),
			Provenance: skills.Provenance{
				Registry:    "test-registry",
				Slug:        "@acme/review",
				Version:     "1.0.0",
				InstalledAt: timeNowForTest(),
			},
		}
		registry := fakeInstallRegistry{
			checkUpdateErr: registrypkg.NewPackageNotFoundError("@acme/review"),
		}

		_, err := UpdateSkill(
			context.Background(),
			t.TempDir(),
			registry,
			installed,
			true,
			timeNowForTest,
		)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("UpdateSkill() error = %v, want ErrNotFound", err)
		}
	})
}

func marketplaceSkillArchive(t *testing.T, name string, description string, body string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	content := "---\nname: " + name + "\ndescription: " + description + "\nversion: 1.0.0\n---\n" + body + "\n"
	writeMarketplaceTarEntry(t, tarWriter, "skill/SKILL.md", content)
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tarWriter.Close() error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close() error = %v", err)
	}
	return buffer.Bytes()
}

func writeMarketplaceTarEntry(t *testing.T, writer *tar.Writer, name string, content string) {
	t.Helper()

	header := &tar.Header{
		Name:     name,
		Mode:     0o644,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := writer.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader(%q) error = %v", name, err)
	}
	if _, err := io.WriteString(writer, content); err != nil {
		t.Fatalf("Write(%q) error = %v", name, err)
	}
}
