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
	"strings"
	"testing"
	"time"

	registrypkg "github.com/compozy/agh/internal/registry"
	"github.com/compozy/agh/internal/skills"
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

type fakeSkillResolver struct {
	skills map[string]*skills.Skill
}

func (r fakeSkillResolver) Get(name string) (*skills.Skill, bool) {
	skill, ok := r.skills[name]
	return skill, ok
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

func TestNormalizeSkillSlug(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		input           string
		want            string
		wantErr         error
		wantErrContains string
	}{
		{
			name:  "Should accept scoped marketplace slugs",
			input: "@acme/review",
			want:  "@acme/review",
		},
		{
			name:  "Should accept raw registry skill slugs",
			input: "review",
			want:  "review",
		},
		{
			name:  "Should trim accepted registry slugs",
			input: "  agentvibes-openclaw-skill  ",
			want:  "agentvibes-openclaw-skill",
		},
		{
			name:            "Should reject empty slugs",
			input:           "   ",
			wantErr:         ErrValidation,
			wantErrContains: "skill slug is required",
		},
		{
			name:            "Should reject raw slugs with path separators",
			input:           "acme/review",
			wantErr:         ErrValidation,
			wantErrContains: "skill slug must not include path separators",
		},
		{
			name:            "Should reject raw relative path segments",
			input:           "..",
			wantErr:         ErrValidation,
			wantErrContains: "skill slug must not be a relative path segment",
		},
		{
			name:            "Should reject raw slugs with unsupported characters",
			input:           "bad slug",
			wantErr:         ErrValidation,
			wantErrContains: `skill slug must match "@author/name"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeSkillSlug(tc.input)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("NormalizeSkillSlug(%q) error = %v, want %v", tc.input, err, tc.wantErr)
				}
				if tc.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErrContains)) {
					t.Fatalf(
						"NormalizeSkillSlug(%q) error = %v, want message containing %q",
						tc.input,
						err,
						tc.wantErrContains,
					)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeSkillSlug(%q) error = %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeSkillSlug(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
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

func TestVerifyInstallVisible(t *testing.T) {
	t.Parallel()

	result := InstallResult{
		Name:     "review",
		Slug:     "@agh/review",
		Version:  "1.2.0",
		Registry: "clawhub",
		Path:     "/tmp/agh/skills/review",
		Hash:     "sha256:abc",
		Status:   "installed",
	}
	visibleSkill := func() *skills.Skill {
		return &skills.Skill{
			Meta:   skills.SkillMeta{Name: "review"},
			Source: skills.SourceMarketplace,
			FilePath: filepath.Join(
				"tmp",
				"agh",
				"skills",
				"review",
				SkillMarkdownFileName,
			),
			Enabled: true,
			Provenance: &skills.Provenance{
				Slug:     "@agh/review",
				Registry: "clawhub",
				Version:  "1.2.0",
			},
		}
	}

	t.Run("Should accept visible marketplace skill with matching provenance", func(t *testing.T) {
		t.Parallel()

		err := VerifyInstallVisible(fakeSkillResolver{skills: map[string]*skills.Skill{
			"review": visibleSkill(),
		}}, result)
		if err != nil {
			t.Fatalf("VerifyInstallVisible() error = %v", err)
		}
	})

	cases := []struct {
		name     string
		resolver SkillResolver
		wantText string
	}{
		{
			name:     "Should classify missing discovery as unavailable",
			resolver: fakeSkillResolver{skills: map[string]*skills.Skill{}},
			wantText: "not visible after skill discovery",
		},
		{
			name: "Should classify shadowing source as unavailable",
			resolver: fakeSkillResolver{skills: map[string]*skills.Skill{
				"review": {
					Meta:     skills.SkillMeta{Name: "review"},
					Source:   skills.SourceUser,
					FilePath: "/tmp/agh/skills/review/SKILL.md",
					Enabled:  true,
				},
			}},
			wantText: "resolved as user from /tmp/agh/skills/review/SKILL.md",
		},
		{
			name: "Should classify missing provenance as unavailable",
			resolver: fakeSkillResolver{skills: map[string]*skills.Skill{
				"review": func() *skills.Skill {
					skill := visibleSkill()
					skill.Provenance = nil
					return skill
				}(),
			}},
			wantText: "missing provenance after skill discovery",
		},
		{
			name: "Should classify slug mismatch as unavailable",
			resolver: fakeSkillResolver{skills: map[string]*skills.Skill{
				"review": func() *skills.Skill {
					skill := visibleSkill()
					skill.Provenance.Slug = "@other/review"
					return skill
				}(),
			}},
			wantText: "resolved slug \"@other/review\" after skill discovery, want \"@agh/review\"",
		},
		{
			name: "Should classify disabled skill as unavailable",
			resolver: fakeSkillResolver{skills: map[string]*skills.Skill{
				"review": func() *skills.Skill {
					skill := visibleSkill()
					skill.Enabled = false
					return skill
				}(),
			}},
			wantText: "visible but disabled after skill discovery",
		},
		{
			name:     "Should classify missing resolver as unavailable",
			resolver: nil,
			wantText: "skill discovery is not configured",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := VerifyInstallVisible(tc.resolver, result)
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("VerifyInstallVisible() error = %v, want ErrUnavailable", err)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("VerifyInstallVisible() error = %v, want %q", err, tc.wantText)
			}
		})
	}
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
