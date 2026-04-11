//go:build integration

package extension

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestManagerIntegrationLifecycleAndHostAPICall(t *testing.T) {
	withDaemonVersion(t, "0.5.0")

	env := newRegistryTestEnv(t)
	markerPath := filepath.Join(t.TempDir(), "host-call.json")
	fixture := createManagerTestExtension(t, managerTestManifest("ext-host", managerManifestOptions{
		command:      helperCommand(t),
		args:         helperArgs(),
		withEnv:      helperEnv("host_call", markerPath),
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	manager := NewManager(
		env.registry,
		WithHostMethodHandler("sessions/list", func(_ context.Context, _ json.RawMessage) (any, error) {
			return []map[string]string{{"id": "sess-1"}}, nil
		}),
		WithHealthCheckTimeout(20*time.Millisecond),
		WithSubprocessSignalGrace(15*time.Millisecond),
	)

	if err := manager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("Stop() cleanup error = %v", err)
		}
	})

	waitForManagerCondition(t, time.Second, func() bool {
		_, err := os.Stat(markerPath)
		return err == nil
	})

	payload, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", markerPath, err)
	}
	if !strings.Contains(string(payload), "sess-1") {
		t.Fatalf("host call payload = %s, want sess-1 response", string(payload))
	}
}

func TestManagerIntegrationRestartRecovery(t *testing.T) {
	withDaemonVersion(t, "0.5.0")

	env := newRegistryTestEnv(t)
	markerPath := filepath.Join(t.TempDir(), "starts.log")
	fixture := createManagerTestExtension(t, managerTestManifest("ext-recover", managerManifestOptions{
		command:      helperCommand(t),
		args:         helperArgs(),
		withEnv:      helperEnv("auto_exit", markerPath),
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	manager := NewManager(
		env.registry,
		WithHealthCheckTimeout(20*time.Millisecond),
		WithSubprocessSignalGrace(15*time.Millisecond),
		withRestartBackoffMax(10*time.Millisecond),
		withHealthPollBounds(time.Millisecond, 2*time.Millisecond),
	)

	if err := manager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("Stop() cleanup error = %v", err)
		}
	})

	waitForManagerCondition(t, 2*time.Second, func() bool {
		payload, err := os.ReadFile(markerPath)
		if err != nil {
			return false
		}
		return len(strings.Fields(string(payload))) >= 2
	})
}

func TestManagerIntegrationResourceRegistration(t *testing.T) {
	withDaemonVersion(t, "0.5.0")

	env := newRegistryTestEnv(t)
	skillsRegistry := skillspkg.NewRegistry(skillspkg.RegistryConfig{})
	fixture := createManagerTestExtension(t, managerTestManifest("ext-resources", managerManifestOptions{
		command:      helperCommand(t),
		args:         helperArgs(),
		withEnv:      helperEnv("default", ""),
		withSkills:   true,
		withAgents:   true,
		withHooks:    true,
		withMCP:      true,
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), map[string]string{
		"skills/review.md": managerSkillFile("resource-skill", "Loaded from extension"),
		"agents/agent.md":  managerAgentFile("resource-agent"),
	})
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	manager := NewManager(
		env.registry,
		WithSkillsRegistry(skillsRegistry),
		WithHealthCheckTimeout(20*time.Millisecond),
		WithSubprocessSignalGrace(15*time.Millisecond),
	)

	if err := manager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("Stop() cleanup error = %v", err)
		}
	})

	if skills := skillsRegistry.List(); len(skills) != 1 || skills[0].Meta.Name != "resource-skill" {
		t.Fatalf("skills registry List() = %#v, want resource-skill", skills)
	}
	if agents := manager.AgentDefinitions(); len(agents) != 1 || agents[0].Name != "resource-agent" {
		t.Fatalf("AgentDefinitions() = %#v, want resource-agent", agents)
	}
	if decls, err := manager.HookDeclarations(testutil.Context(t)); err != nil {
		t.Fatalf("HookDeclarations() error = %v", err)
	} else if len(decls) != 1 || decls[0].Name != "ext-resources-hook" {
		t.Fatalf("HookDeclarations() = %#v, want ext-resources-hook", decls)
	}
}
