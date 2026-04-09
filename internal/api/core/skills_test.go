package core_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// stubSkillsRegistry implements core.SkillsRegistry for testing.
type stubSkillsRegistry struct {
	GetFn          func(name string) (*skills.Skill, bool)
	ListFn         func() []*skills.Skill
	ForWorkspaceFn func(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error)
	SetEnabledFn   func(name string, enabled bool) error
}

func (s *stubSkillsRegistry) Get(name string) (*skills.Skill, bool) {
	if s.GetFn != nil {
		return s.GetFn(name)
	}
	return nil, false
}

func (s *stubSkillsRegistry) List() []*skills.Skill {
	if s.ListFn != nil {
		return s.ListFn()
	}
	return nil
}

func (s *stubSkillsRegistry) ForWorkspace(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error) {
	if s.ForWorkspaceFn != nil {
		return s.ForWorkspaceFn(ctx, resolved)
	}
	return nil, nil
}

func (s *stubSkillsRegistry) SetEnabled(name string, enabled bool) error {
	if s.SetEnabledFn != nil {
		return s.SetEnabledFn(name, enabled)
	}
	return nil
}

var _ core.SkillsRegistry = (*stubSkillsRegistry)(nil)

func newSkillsHandlerFixture(t *testing.T, registry core.SkillsRegistry, workspaces testutil.StubWorkspaceService) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123
	cfg.Daemon.Socket = "/tmp/skills-test.sock"

	handlers := core.NewBaseHandlers(core.BaseHandlerConfig{
		TransportName:  "skills-test",
		Sessions:       testutil.StubSessionManager{},
		Observer:       testutil.StubObserver{},
		Workspaces:     workspaces,
		SkillsRegistry: registry,
		HomePaths:      homePaths,
		Config:         cfg,
		Logger:         testutil.DiscardLogger(),
		StartedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		Now:            func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		PollInterval:   5 * time.Millisecond,
		HTTPPort:       cfg.HTTP.Port,
	})

	engine := gin.New()
	engine.GET("/api/skills", handlers.ListSkills)
	engine.GET("/api/skills/:name", handlers.GetSkill)
	engine.POST("/api/skills/:name/enable", handlers.EnableSkill)
	engine.POST("/api/skills/:name/disable", handlers.DisableSkill)

	return engine
}

func testSkill() *skills.Skill {
	return &skills.Skill{
		Meta: skills.SkillMeta{
			Name:        "test-skill",
			Description: "A test skill",
			Version:     "1.0.0",
			Metadata:    map[string]any{"key": "value"},
		},
		Content: "# Test Skill\nBody content",
		Source:  skills.SourceBundled,
		Dir:     "test-skill",
		Enabled: true,
	}
}

func testSkillWithProvenance() *skills.Skill {
	s := testSkill()
	s.Source = skills.SourceMarketplace
	s.Provenance = &skills.Provenance{
		Slug:        "test-org/test-skill",
		Registry:    "https://skills.example.com",
		Version:     "1.0.0",
		InstalledAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
	return s
}

func TestSkillPayloadFromSkill(t *testing.T) {
	t.Parallel()

	t.Run("converts all fields correctly including provenance", func(t *testing.T) {
		t.Parallel()

		skill := testSkillWithProvenance()
		payload := core.SkillPayloadFromSkill(skill)

		if payload.Name != "test-skill" {
			t.Errorf("Name = %q, want %q", payload.Name, "test-skill")
		}
		if payload.Description != "A test skill" {
			t.Errorf("Description = %q, want %q", payload.Description, "A test skill")
		}
		if payload.Version != "1.0.0" {
			t.Errorf("Version = %q, want %q", payload.Version, "1.0.0")
		}
		if payload.Source != "marketplace" {
			t.Errorf("Source = %q, want %q", payload.Source, "marketplace")
		}
		if !payload.Enabled {
			t.Error("Enabled = false, want true")
		}
		if payload.Dir != "test-skill" {
			t.Errorf("Dir = %q, want %q", payload.Dir, "test-skill")
		}
		if payload.Content != "# Test Skill\nBody content" {
			t.Errorf("Content = %q, want body content", payload.Content)
		}
		if payload.Metadata == nil || payload.Metadata["key"] != "value" {
			t.Errorf("Metadata = %v, want map with key=value", payload.Metadata)
		}
		if payload.Provenance == nil {
			t.Fatal("Provenance = nil, want non-nil")
		}
		if payload.Provenance.Slug != "test-org/test-skill" {
			t.Errorf("Provenance.Slug = %q, want %q", payload.Provenance.Slug, "test-org/test-skill")
		}
		if payload.Provenance.Registry != "https://skills.example.com" {
			t.Errorf("Provenance.Registry = %q", payload.Provenance.Registry)
		}
		if payload.Provenance.Version != "1.0.0" {
			t.Errorf("Provenance.Version = %q", payload.Provenance.Version)
		}
	})

	t.Run("omits empty optional fields", func(t *testing.T) {
		t.Parallel()

		skill := &skills.Skill{
			Meta: skills.SkillMeta{
				Name:        "minimal",
				Description: "Minimal skill",
			},
			Source:  skills.SourceBundled,
			Dir:     "minimal",
			Enabled: true,
		}

		payload := core.SkillPayloadFromSkill(skill)

		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}

		for _, key := range []string{"version", "content", "metadata", "provenance"} {
			if _, exists := m[key]; exists {
				t.Errorf("JSON contains %q but field should be omitted", key)
			}
		}
	})

	t.Run("nil skill returns zero payload", func(t *testing.T) {
		t.Parallel()

		payload := core.SkillPayloadFromSkill(nil)
		if payload.Name != "" {
			t.Errorf("Name = %q, want empty", payload.Name)
		}
	})
}

func TestStatusForSkillError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"nil returns 200", nil, http.StatusOK},
		{"not found returns 404", core.ErrSkillNotFound, http.StatusNotFound},
		{"validation returns 400", core.ErrSkillValidation, http.StatusBadRequest},
		{"unknown error returns 500", http.ErrServerClosed, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := core.StatusForSkillError(tt.err)
			if got != tt.wantStatus {
				t.Errorf("StatusForSkillError(%v) = %d, want %d", tt.err, got, tt.wantStatus)
			}
		})
	}
}

func TestListSkills(t *testing.T) {
	t.Parallel()

	t.Run("missing workspace returns 400", func(t *testing.T) {
		t.Parallel()

		engine := newSkillsHandlerFixture(t, &stubSkillsRegistry{}, testutil.StubWorkspaceService{})
		rec := testutil.PerformRequest(t, engine, http.MethodGet, "/api/skills", nil)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	t.Run("valid workspace returns skill list", func(t *testing.T) {
		t.Parallel()

		skill := testSkill()
		registry := &stubSkillsRegistry{
			ForWorkspaceFn: func(_ context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error) {
				if resolved.ID != "ws-1" {
					t.Errorf("ForWorkspace got ID %q, want ws-1", resolved.ID)
				}
				return []*skills.Skill{skill}, nil
			},
		}
		workspaces := testutil.StubWorkspaceService{
			ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
				return workspacepkg.ResolvedWorkspace{
					Workspace: workspacepkg.Workspace{
						ID:      "ws-1",
						RootDir: "/workspace",
						Name:    "test",
					},
				}, nil
			},
		}

		engine := newSkillsHandlerFixture(t, registry, workspaces)
		rec := testutil.PerformRequest(t, engine, http.MethodGet, "/api/skills?workspace=ws-1", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp struct {
			Skills []contract.SkillPayload `json:"skills"`
		}
		testutil.DecodeJSONResponse(t, rec, &resp)

		if len(resp.Skills) != 1 {
			t.Fatalf("len(skills) = %d, want 1", len(resp.Skills))
		}
		if resp.Skills[0].Name != "test-skill" {
			t.Errorf("skills[0].Name = %q, want %q", resp.Skills[0].Name, "test-skill")
		}
	})
}

func TestGetSkill(t *testing.T) {
	t.Parallel()

	t.Run("unknown name returns 404", func(t *testing.T) {
		t.Parallel()

		registry := &stubSkillsRegistry{
			GetFn: func(name string) (*skills.Skill, bool) {
				return nil, false
			},
		}
		engine := newSkillsHandlerFixture(t, registry, testutil.StubWorkspaceService{})
		rec := testutil.PerformRequest(t, engine, http.MethodGet, "/api/skills/nonexistent", nil)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}
	})

	t.Run("valid name returns skill detail with content", func(t *testing.T) {
		t.Parallel()

		skill := testSkillWithProvenance()
		registry := &stubSkillsRegistry{
			GetFn: func(name string) (*skills.Skill, bool) {
				if name == "test-skill" {
					return skill, true
				}
				return nil, false
			},
		}
		engine := newSkillsHandlerFixture(t, registry, testutil.StubWorkspaceService{})
		rec := testutil.PerformRequest(t, engine, http.MethodGet, "/api/skills/test-skill", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp struct {
			Skill contract.SkillPayload `json:"skill"`
		}
		testutil.DecodeJSONResponse(t, rec, &resp)

		if resp.Skill.Name != "test-skill" {
			t.Errorf("skill.Name = %q, want %q", resp.Skill.Name, "test-skill")
		}
		if resp.Skill.Content != "# Test Skill\nBody content" {
			t.Errorf("skill.Content = %q, want body content", resp.Skill.Content)
		}
		if resp.Skill.Provenance == nil {
			t.Error("skill.Provenance = nil, want non-nil")
		}
	})

	t.Run("workspace query resolves workspace-only skills", func(t *testing.T) {
		t.Parallel()

		workspaceSkill := testSkill()
		workspaceSkill.Source = skills.SourceWorkspace
		workspaceSkill.Dir = "/workspace/.agh/skills/test-skill"

		registry := &stubSkillsRegistry{
			GetFn: func(name string) (*skills.Skill, bool) {
				return nil, false
			},
			ForWorkspaceFn: func(_ context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error) {
				if resolved.ID != "ws-1" {
					t.Errorf("ForWorkspace got ID %q, want ws-1", resolved.ID)
				}
				return []*skills.Skill{workspaceSkill}, nil
			},
		}
		workspaces := testutil.StubWorkspaceService{
			ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
				if ref != "ws-1" {
					t.Errorf("Resolve got ref %q, want ws-1", ref)
				}
				return workspacepkg.ResolvedWorkspace{
					Workspace: workspacepkg.Workspace{
						ID:      "ws-1",
						RootDir: "/workspace",
						Name:    "test",
					},
				}, nil
			},
		}

		engine := newSkillsHandlerFixture(t, registry, workspaces)
		rec := testutil.PerformRequest(t, engine, http.MethodGet, "/api/skills/test-skill?workspace=ws-1", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp struct {
			Skill contract.SkillPayload `json:"skill"`
		}
		testutil.DecodeJSONResponse(t, rec, &resp)
		if resp.Skill.Source != "workspace" {
			t.Errorf("skill.Source = %q, want %q", resp.Skill.Source, "workspace")
		}
	})
}

func TestEnableSkill(t *testing.T) {
	t.Parallel()

	t.Run("returns ok true on success", func(t *testing.T) {
		t.Parallel()

		skill := testSkill()
		skill.Enabled = false
		registry := &stubSkillsRegistry{
			GetFn: func(name string) (*skills.Skill, bool) {
				if name == "test-skill" {
					return skill, true
				}
				return nil, false
			},
			ForWorkspaceFn: func(_ context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error) {
				if resolved.ID != "ws-1" {
					t.Errorf("ForWorkspace got ID %q, want ws-1", resolved.ID)
				}
				return []*skills.Skill{skill}, nil
			},
			SetEnabledFn: func(name string, enabled bool) error {
				if name != "test-skill" {
					t.Errorf("SetEnabled got name %q, want %q", name, "test-skill")
				}
				skill.Enabled = enabled
				return nil
			},
		}
		workspaces := testutil.StubWorkspaceService{
			ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
				if ref != "ws-1" {
					t.Errorf("Resolve got ref %q, want ws-1", ref)
				}
				return workspacepkg.ResolvedWorkspace{
					Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: "/workspace", Name: "test"},
				}, nil
			},
		}
		engine := newSkillsHandlerFixture(t, registry, workspaces)
		rec := testutil.PerformRequest(t, engine, http.MethodPost, "/api/skills/test-skill/enable?workspace=ws-1", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp contract.SkillActionResponse
		testutil.DecodeJSONResponse(t, rec, &resp)

		if !resp.OK {
			t.Error("ok = false, want true")
		}
		if !skill.Enabled {
			t.Error("skill.Enabled = false after enable, want true")
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		t.Parallel()

		registry := &stubSkillsRegistry{
			GetFn: func(name string) (*skills.Skill, bool) {
				return nil, false
			},
		}
		workspaces := testutil.StubWorkspaceService{
			ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
				return workspacepkg.ResolvedWorkspace{
					Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: "/workspace", Name: "test"},
				}, nil
			},
		}
		engine := newSkillsHandlerFixture(t, registry, workspaces)
		rec := testutil.PerformRequest(t, engine, http.MethodPost, "/api/skills/missing/enable?workspace=ws-1", nil)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}
	})
}

func TestDisableSkill(t *testing.T) {
	t.Parallel()

	t.Run("returns ok true on success", func(t *testing.T) {
		t.Parallel()

		skill := testSkill()
		skill.Enabled = true
		registry := &stubSkillsRegistry{
			GetFn: func(name string) (*skills.Skill, bool) {
				if name == "test-skill" {
					return skill, true
				}
				return nil, false
			},
			ForWorkspaceFn: func(_ context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error) {
				if resolved.ID != "ws-1" {
					t.Errorf("ForWorkspace got ID %q, want ws-1", resolved.ID)
				}
				return []*skills.Skill{skill}, nil
			},
			SetEnabledFn: func(name string, enabled bool) error {
				if name != "test-skill" {
					t.Errorf("SetEnabled got name %q, want %q", name, "test-skill")
				}
				skill.Enabled = enabled
				return nil
			},
		}
		workspaces := testutil.StubWorkspaceService{
			ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
				if ref != "ws-1" {
					t.Errorf("Resolve got ref %q, want ws-1", ref)
				}
				return workspacepkg.ResolvedWorkspace{
					Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: "/workspace", Name: "test"},
				}, nil
			},
		}
		engine := newSkillsHandlerFixture(t, registry, workspaces)
		rec := testutil.PerformRequest(t, engine, http.MethodPost, "/api/skills/test-skill/disable?workspace=ws-1", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp contract.SkillActionResponse
		testutil.DecodeJSONResponse(t, rec, &resp)

		if !resp.OK {
			t.Error("ok = false, want true")
		}
		if skill.Enabled {
			t.Error("skill.Enabled = true after disable, want false")
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		t.Parallel()

		registry := &stubSkillsRegistry{
			GetFn: func(name string) (*skills.Skill, bool) {
				return nil, false
			},
		}
		workspaces := testutil.StubWorkspaceService{
			ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
				return workspacepkg.ResolvedWorkspace{
					Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: "/workspace", Name: "test"},
				}, nil
			},
		}
		engine := newSkillsHandlerFixture(t, registry, workspaces)
		rec := testutil.PerformRequest(t, engine, http.MethodPost, "/api/skills/missing/disable?workspace=ws-1", nil)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}
	})
}

func TestSkillsRegistryNotConfigured(t *testing.T) {
	t.Parallel()

	engine := newSkillsHandlerFixture(t, nil, testutil.StubWorkspaceService{})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"ListSkills", http.MethodGet, "/api/skills?workspace=ws-1"},
		{"GetSkill", http.MethodGet, "/api/skills/test"},
		{"EnableSkill", http.MethodPost, "/api/skills/test/enable?workspace=ws-1"},
		{"DisableSkill", http.MethodPost, "/api/skills/test/disable?workspace=ws-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := testutil.PerformRequest(t, engine, tt.method, tt.path, nil)
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("%s status = %d, want %d; body=%s", tt.name, rec.Code, http.StatusServiceUnavailable, rec.Body.String())
			}
		})
	}
}
