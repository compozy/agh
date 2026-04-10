package extension

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/testutil"
)

const (
	extensionHelperEnvKey      = "AGH_TEST_EXTENSION_HELPER"
	extensionHelperScenarioKey = "AGH_TEST_EXTENSION_SCENARIO"
	extensionHelperMarkerKey   = "AGH_TEST_EXTENSION_MARKER"
)

func TestExtensionManagerHelperProcess(t *testing.T) {
	if os.Getenv(extensionHelperEnvKey) != "1" {
		return
	}

	server := newExtensionHelperServer(
		os.Getenv(extensionHelperScenarioKey),
		strings.TrimSpace(os.Getenv(extensionHelperMarkerKey)),
	)
	os.Exit(server.run())
}

func TestManagerStartRegistersResourcesAndActivatesExtension(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest("ext-runtime", managerManifestOptions{
		command:      "fake-extension",
		withSkills:   true,
		withAgents:   true,
		withHooks:    true,
		withMCP:      true,
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), map[string]string{
		"skills/review.md": managerSkillFile("ext-review", "External review workflow"),
		"agents/coder.md":  managerAgentFile("ext-agent"),
	})
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	fakeProc := newFakeProcess(101)
	launcher := &fakeLauncher{queue: []*fakeProcess{fakeProc}}
	skillsRegistry := skillspkg.NewRegistry(skillspkg.RegistryConfig{})

	manager := NewManager(
		env.registry,
		WithSkillsRegistry(skillsRegistry),
		WithHostMethodHandler("sessions/list", func(_ context.Context, _ json.RawMessage) (any, error) {
			return []map[string]string{{"id": "sess-1"}}, nil
		}),
		WithHealthCheckTimeout(20*time.Millisecond),
		WithDefaultHookTimeout(25*time.Millisecond),
		withProcessLauncher(launcher.launch),
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

	if got := launcher.launchCount(); got != 1 {
		t.Fatalf("launch count = %d, want 1", got)
	}
	if len(fakeProc.initRequests()) != 1 {
		t.Fatalf("len(initialize requests) = %d, want 1", len(fakeProc.initRequests()))
	}

	request := fakeProc.initRequests()[0]
	if request.ProtocolVersion != defaultProtocolVersion {
		t.Fatalf("initialize protocol version = %q, want %q", request.ProtocolVersion, defaultProtocolVersion)
	}
	if !slices.Equal(request.Capabilities.GrantedActions, []extensionprotocol.HostAPIMethod{
		extensionprotocol.HostAPIMethodSessionsList,
	}) {
		t.Fatalf("initialize granted actions = %#v, want [sessions/list]", request.Capabilities.GrantedActions)
	}
	if !slices.Equal(request.Capabilities.GrantedSecurity, []string{"session.read"}) {
		t.Fatalf("initialize granted security = %#v, want [session.read]", request.Capabilities.GrantedSecurity)
	}
	if !slices.Equal(request.Methods.ExtensionServices, []string{"memory/forget", "memory/recall", "memory/store"}) {
		t.Fatalf("initialize extension services = %#v, want memory backend methods", request.Methods.ExtensionServices)
	}

	decls, err := manager.HookDeclarations(testutil.Context(t))
	if err != nil {
		t.Fatalf("HookDeclarations() error = %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("len(HookDeclarations()) = %d, want 1", len(decls))
	}
	if got, want := decls[0].Name, "ext-runtime-hook"; got != want {
		t.Fatalf("HookDeclarations()[0].Name = %q, want %q", got, want)
	}
	if got, want := decls[0].Metadata["extension"], "ext-runtime"; got != want {
		t.Fatalf("HookDeclarations()[0].Metadata[extension] = %q, want %q", got, want)
	}

	agents := manager.AgentDefinitions()
	if len(agents) != 1 || agents[0].Name != "ext-agent" {
		t.Fatalf("AgentDefinitions() = %#v, want ext-agent", agents)
	}

	servers := manager.MCPServers()
	if len(servers) != 1 || servers[0].Name != "kubectl" {
		t.Fatalf("MCPServers() = %#v, want kubectl server", servers)
	}

	skills := skillsRegistry.List()
	if len(skills) != 1 || skills[0].Meta.Name != "ext-review" {
		t.Fatalf("skills registry List() = %#v, want ext-review", skills)
	}

	loaded, err := manager.Get("ext-runtime")
	if err != nil {
		t.Fatalf("Get(ext-runtime) error = %v", err)
	}
	if !loaded.Status.Active {
		t.Fatalf("Get(ext-runtime).Status.Active = false, want true")
	}
	if got, want := loaded.Status.Phase, ExtensionPhaseActivate; got != want {
		t.Fatalf("Get(ext-runtime).Status.Phase = %q, want %q", got, want)
	}
}

func TestManagerStartSkipsDisabledExtensions(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest("ext-disabled", managerManifestOptions{
		command:      "fake-extension",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, false)

	launcher := &fakeLauncher{}
	manager := NewManager(env.registry, withProcessLauncher(launcher.launch))

	if err := manager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("Stop() cleanup error = %v", err)
		}
	})

	if got := launcher.launchCount(); got != 0 {
		t.Fatalf("launch count = %d, want 0", got)
	}

	statuses := manager.Statuses()
	if len(statuses) != 1 || statuses[0].Enabled {
		t.Fatalf("Statuses() = %#v, want one disabled extension", statuses)
	}
}

func TestManagerStartContinuesAfterParseFailure(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)

	badFixture := createManagerTestExtension(t, managerTestManifest("ext-bad", managerManifestOptions{
		command:      "fake-extension",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, badFixture, SourceUser, true)
	writeFile(t, filepath.Join(badFixture.dir, manifestTOMLFileName), "not = [valid")

	goodFixture := createManagerTestExtension(t, managerTestManifest("ext-good", managerManifestOptions{
		command:      "fake-extension",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
		withHooks:    true,
	}), nil)
	installManagerFixture(t, env.registry, goodFixture, SourceUser, true)

	launcher := &fakeLauncher{queue: []*fakeProcess{newFakeProcess(202)}}
	manager := NewManager(
		env.registry,
		withProcessLauncher(launcher.launch),
		withHealthPollBounds(time.Millisecond, 2*time.Millisecond),
	)

	err := manager.Start(testutil.Context(t))
	if err == nil {
		t.Fatal("Start() error = nil, want joined parse failure")
	}
	if !strings.Contains(err.Error(), `extension "ext-bad" parse`) {
		t.Fatalf("Start() error = %v, want parse phase detail for ext-bad", err)
	}
	t.Cleanup(func() {
		if stopErr := manager.Stop(testutil.Context(t)); stopErr != nil {
			t.Fatalf("Stop() cleanup error = %v", stopErr)
		}
	})

	if got := launcher.launchCount(); got != 1 {
		t.Fatalf("launch count = %d, want 1 for ext-good only", got)
	}
	good, err := manager.Get("ext-good")
	if err != nil {
		t.Fatalf("Get(ext-good) error = %v", err)
	}
	if !good.Status.Active {
		t.Fatalf("Get(ext-good).Status.Active = false, want true")
	}
}

func TestManagerStartRejectsIncompatibleManifest(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest("ext-incompatible", managerManifestOptions{
		command:      "fake-extension",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)
	writeFile(t, filepath.Join(fixture.dir, manifestTOMLFileName), managerTestManifest("ext-incompatible", managerManifestOptions{
		command:      "fake-extension",
		minVersion:   "9.0.0",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}))

	launcher := &fakeLauncher{}
	manager := NewManager(env.registry, withProcessLauncher(launcher.launch))

	err := manager.Start(testutil.Context(t))
	if err == nil {
		t.Fatal("Start() error = nil, want incompatibility error")
	}
	if !errors.Is(err, ErrManifestIncompatible) {
		t.Fatalf("Start() error = %v, want ErrManifestIncompatible", err)
	}
	if got := launcher.launchCount(); got != 0 {
		t.Fatalf("launch count = %d, want 0", got)
	}
}

func TestManagerCrashTriggersRestartWithBackoff(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest("ext-restart", managerManifestOptions{
		command:      "fake-extension",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	first := newFakeProcess(301)
	second := newFakeProcess(302)
	launcher := &fakeLauncher{queue: []*fakeProcess{first, second}}

	manager := NewManager(
		env.registry,
		withProcessLauncher(launcher.launch),
		withRestartBackoffMax(2*time.Millisecond),
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

	first.crash(errors.New("boom"))

	waitForManagerCondition(t, time.Second, func() bool {
		if launcher.launchCount() != 2 {
			return false
		}
		loaded, err := manager.Get("ext-restart")
		return err == nil && loaded.Status.Active && loaded.Status.PID == 302
	})

	loaded, err := manager.Get("ext-restart")
	if err != nil {
		t.Fatalf("Get(ext-restart) error = %v", err)
	}
	if !loaded.Status.Active || loaded.Status.PID != 302 {
		t.Fatalf("Get(ext-restart).Status = %#v, want restarted process pid 302", loaded.Status)
	}
}

func TestManagerStartDetachesSupervisorFromStartContext(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest("ext-detached", managerManifestOptions{
		command:      "fake-extension",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	first := newFakeProcess(601)
	second := newFakeProcess(602)
	launcher := &fakeLauncher{queue: []*fakeProcess{first, second}}

	manager := NewManager(
		env.registry,
		withProcessLauncher(launcher.launch),
		withRestartBackoffMax(2*time.Millisecond),
		withHealthPollBounds(time.Millisecond, 2*time.Millisecond),
	)

	startCtx, cancelStart := context.WithCancel(testutil.Context(t))
	if err := manager.Start(startCtx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	cancelStart()
	t.Cleanup(func() {
		if err := manager.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("Stop() cleanup error = %v", err)
		}
	})

	first.crash(errors.New("boom"))

	waitForManagerCondition(t, time.Second, func() bool {
		if launcher.launchCount() != 2 {
			return false
		}
		loaded, err := manager.Get("ext-detached")
		return err == nil && loaded.Status.Active && loaded.Status.PID == 602
	})
}

func TestManagerDisablesExtensionAfterConsecutiveFailures(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest("ext-flaky", managerManifestOptions{
		command:      "fake-extension",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
		withHooks:    true,
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	queue := make([]*fakeProcess, 0, 5)
	for pid := 401; pid <= 405; pid++ {
		proc := newFakeProcess(pid)
		proc.initHook = func(p *fakeProcess) func() {
			return func() {
				go p.crash(errors.New("panic"))
			}
		}(proc)
		queue = append(queue, proc)
	}
	launcher := &fakeLauncher{queue: queue}

	manager := NewManager(
		env.registry,
		withProcessLauncher(launcher.launch),
		withRestartBackoffMax(2*time.Millisecond),
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

	waitForManagerCondition(t, time.Second, func() bool {
		info, err := env.registry.Get("ext-flaky")
		return err == nil && !info.Enabled
	})

	if got := launcher.launchCount(); got != 5 {
		t.Fatalf("launch count = %d, want 5 before disable", got)
	}

	decls, err := manager.HookDeclarations(testutil.Context(t))
	if err != nil {
		t.Fatalf("HookDeclarations() error = %v", err)
	}
	if len(decls) != 0 {
		t.Fatalf("HookDeclarations() = %#v, want resources removed after disable", decls)
	}
}

func TestManagerStopUsesRealSubprocessShutdown(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	markerPath := filepath.Join(t.TempDir(), "shutdown.marker")
	fixture := createManagerTestExtension(t, managerTestManifest("ext-stop", managerManifestOptions{
		command:      helperCommand(t),
		args:         helperArgs(),
		withEnv:      helperEnv("default", markerPath),
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	manager := NewManager(
		env.registry,
		WithHealthCheckTimeout(25*time.Millisecond),
		WithDefaultHookTimeout(25*time.Millisecond),
		WithSubprocessSignalGrace(15*time.Millisecond),
	)

	if err := manager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := manager.Stop(testutil.Context(t)); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want shutdown marker", markerPath, err)
	}
}

func TestManagerStopKillsHungSubprocessAfterTimeout(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest("ext-hang", managerManifestOptions{
		command:      helperCommand(t),
		args:         helperArgs(),
		withEnv:      helperEnv("shutdown_hang", ""),
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
		shutdown:     40 * time.Millisecond,
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	manager := NewManager(
		env.registry,
		WithHealthCheckTimeout(20*time.Millisecond),
		WithSubprocessSignalGrace(20*time.Millisecond),
	)

	if err := manager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := manager.Stop(testutil.Context(t)); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	statuses := manager.Statuses()
	if len(statuses) != 1 || statuses[0].Active {
		t.Fatalf("Statuses() after Stop = %#v, want inactive extension", statuses)
	}
}

func TestNewManagerAppliesOptionsAndRestoresDefaults(t *testing.T) {
	t.Parallel()

	manager := NewManager(
		nil,
		WithCapabilityChecker(nil),
		WithLogger(nil),
		WithNow(nil),
		WithGetenv(nil),
		WithHostMethodHandler(" sessions/list ", func(context.Context, json.RawMessage) (any, error) {
			return "ok", nil
		}),
		WithInitializeTimeout(0),
		WithHealthCheckTimeout(0),
		WithDefaultHookTimeout(0),
		WithSubprocessSignalGrace(0),
		withProcessLauncher(nil),
		withRestartBackoffMax(0),
		withRestartFailureThreshold(0),
		withHealthPollBounds(0, 0),
	)

	if manager.capChecker == nil {
		t.Fatal("capChecker = nil, want default checker")
	}
	if manager.logger == nil {
		t.Fatal("logger = nil, want default logger")
	}
	if manager.now == nil {
		t.Fatal("now = nil, want default clock")
	}
	if manager.getenv == nil {
		t.Fatal("getenv = nil, want default env resolver")
	}
	if manager.launch == nil {
		t.Fatal("launch = nil, want default launcher")
	}
	if got, want := manager.initializeTimeout, defaultInitializeTimeout; got != want {
		t.Fatalf("initializeTimeout = %v, want %v", got, want)
	}
	if got, want := manager.healthCheckTimeout, defaultHealthCheckTimeout; got != want {
		t.Fatalf("healthCheckTimeout = %v, want %v", got, want)
	}
	if got, want := manager.defaultHookTimeout, defaultHookTimeout; got != want {
		t.Fatalf("defaultHookTimeout = %v, want %v", got, want)
	}
	if got, want := manager.restartBackoffMax, defaultRestartBackoffMax; got != want {
		t.Fatalf("restartBackoffMax = %v, want %v", got, want)
	}
	if got, want := manager.restartFailureThreshold, defaultRestartFailureThreshold; got != want {
		t.Fatalf("restartFailureThreshold = %d, want %d", got, want)
	}
	if got, want := manager.healthPollFloor, defaultHealthPollFloor; got != want {
		t.Fatalf("healthPollFloor = %v, want %v", got, want)
	}
	if got, want := manager.healthPollCeiling, defaultHealthPollCeiling; got != want {
		t.Fatalf("healthPollCeiling = %v, want %v", got, want)
	}
	if got, want := manager.subprocessSignalGrace, defaultSubprocessSignalGrace; got != want {
		t.Fatalf("subprocessSignalGrace = %v, want %v", got, want)
	}
	if _, ok := manager.hostMethods["sessions/list"]; !ok {
		t.Fatalf("hostMethods = %#v, want trimmed sessions/list key", manager.hostMethods)
	}
}

func TestManagerHelperPathsAndAccessors(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	env := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest("ext-fallback", managerManifestOptions{
		command:      "fake-extension",
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
		security:     []string{"session.read"},
	}), nil)
	installManagerFixture(t, env.registry, fixture, SourceUser, true)

	manager := NewManager(env.registry)

	fallback, err := manager.Get("ext-fallback")
	if err != nil {
		t.Fatalf("Get(ext-fallback) error = %v", err)
	}
	if fallback.Info.Name != "ext-fallback" || !fallback.Status.Enabled {
		t.Fatalf("Get(ext-fallback) = %#v, want registry-backed enabled snapshot", fallback)
	}
	if _, err := manager.Get(" "); err == nil {
		t.Fatal("Get(empty) error = nil, want validation error")
	}

	now := time.Now().UTC()
	proc := newFakeProcess(707)
	proc.health = subprocess.HealthState{
		Healthy:       true,
		Message:       "healthy",
		LastCheckedAt: now,
	}

	manager.extensions = map[string]*managedExtension{
		"zeta": {
			info:                ExtensionInfo{Name: "zeta", Version: "1.0.0", Source: SourceUser, Enabled: true},
			process:             proc,
			active:              true,
			registered:          true,
			phase:               ExtensionPhaseActivate,
			generation:          2,
			awaitingStability:   true,
			consecutiveFailures: 2,
			restartBackoff:      2 * time.Second,
			healthInterval:      40 * time.Millisecond,
			runtime:             subprocess.InitializeRuntime{ShutdownTimeoutMS: 321},
		},
		"alpha": {
			info:       ExtensionInfo{Name: "alpha", Version: "0.1.0", Source: SourceMarketplace, Enabled: false},
			registered: false,
			phase:      ExtensionPhaseDiscover,
		},
	}

	listed := manager.List()
	if got := []string{listed[0].Name, listed[1].Name}; !slices.Equal(got, []string{"alpha", "zeta"}) {
		t.Fatalf("List() names = %#v, want sorted alpha/zeta", got)
	}

	statuses := manager.Statuses()
	if got := []string{statuses[0].Name, statuses[1].Name}; !slices.Equal(got, []string{"alpha", "zeta"}) {
		t.Fatalf("Statuses() names = %#v, want sorted alpha/zeta", got)
	}
	if statuses[1].PID != 707 || !statuses[1].Healthy || statuses[1].HealthMessage != "healthy" {
		t.Fatalf("Statuses()[1] = %#v, want pid/health from process", statuses[1])
	}

	manager.markStable("zeta", 2)
	if manager.extensions["zeta"].awaitingStability {
		t.Fatal("awaitingStability = true, want false after markStable")
	}
	if manager.extensions["zeta"].consecutiveFailures != 0 || manager.extensions["zeta"].restartBackoff != 0 {
		t.Fatalf("post-markStable failures/backoff = %d/%v, want 0/0", manager.extensions["zeta"].consecutiveFailures, manager.extensions["zeta"].restartBackoff)
	}

	current, interval, ok := manager.currentProcess("zeta", 2)
	if !ok || current == nil || current.PID() != 707 || interval != 40*time.Millisecond {
		t.Fatalf("currentProcess() = (%v, %v, %v), want process pid 707 and interval 40ms", current, interval, ok)
	}
	if manager.shouldStopSupervision("zeta", 2, proc) {
		t.Fatal("shouldStopSupervision() = true, want false for current process")
	}
	manager.stopping = true
	if !manager.shouldStopSupervision("zeta", 2, proc) {
		t.Fatal("shouldStopSupervision() = false, want true while stopping")
	}
	manager.stopping = false

	if got, want := manager.shutdownDeadlineForProcess("zeta", 2), 321*time.Millisecond; got != want {
		t.Fatalf("shutdownDeadlineForProcess() = %v, want %v", got, want)
	}
	if got, want := manager.healthPollInterval(0), manager.healthPollCeiling; got != want {
		t.Fatalf("healthPollInterval(0) = %v, want %v", got, want)
	}
	if got, want := manager.healthPollInterval(time.Millisecond), manager.healthPollFloor; got != want {
		t.Fatalf("healthPollInterval(1ms) = %v, want %v", got, want)
	}

	select {
	case <-manager.lifecycleDone():
	default:
		t.Fatal("lifecycleDone() returned open channel, want closed channel when lifecycle is nil")
	}
	if manager.lifecycleContext().Done() != nil {
		t.Fatal("lifecycleContext() returned cancellable context, want background context")
	}

	if got := restartBackoff(0, 5*time.Second); got != 0 {
		t.Fatalf("restartBackoff(0) = %v, want 0", got)
	}
	if got := restartBackoff(10, 5*time.Second); got != 5*time.Second {
		t.Fatalf("restartBackoff(10) = %v, want capped 5s", got)
	}

	if !requiresSubprocess(&Manifest{Actions: ActionsConfig{Requires: []string{"sessions/list"}}}) {
		t.Fatal("requiresSubprocess(actions-only) = false, want true")
	}
	if requiresSubprocess(&Manifest{}) {
		t.Fatal("requiresSubprocess(empty manifest) = true, want false")
	}
	if got, want := skillSourceForExtension(SourceWorkspace), skillspkg.SourceWorkspace; got != want {
		t.Fatalf("skillSourceForExtension(workspace) = %v, want %v", got, want)
	}
	if got, want := skillSourceForExtension(ExtensionSource(99)), skillspkg.SourceUser; got != want {
		t.Fatalf("skillSourceForExtension(default) = %v, want %v", got, want)
	}
	if _, err := loadManifestAtPath(filepath.Join(t.TempDir(), "extension.txt")); err == nil {
		t.Fatal("loadManifestAtPath(.txt) error = nil, want unsupported-extension error")
	}
}

func TestManagerResolveCommandKeepsPathLikeValuesInsideExtensionRoot(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil)
	root := t.TempDir()
	inside := filepath.Join(root, "bin", "tool")

	got, err := manager.resolveCommand(root, "./bin/tool")
	if err != nil {
		t.Fatalf("resolveCommand(relative) error = %v", err)
	}
	if got != inside {
		t.Fatalf("resolveCommand(relative) = %q, want %q", got, inside)
	}

	got, err = manager.resolveCommand(root, inside)
	if err != nil {
		t.Fatalf("resolveCommand(absolute inside) error = %v", err)
	}
	if got != inside {
		t.Fatalf("resolveCommand(absolute inside) = %q, want %q", got, inside)
	}

	outsideCommand := filepath.Join(t.TempDir(), "tool")
	got, err = manager.resolveCommand(root, outsideCommand)
	if err != nil {
		t.Fatalf("resolveCommand(absolute outside) error = %v", err)
	}
	if got != outsideCommand {
		t.Fatalf("resolveCommand(absolute outside) = %q, want %q", got, outsideCommand)
	}

	got, err = manager.resolveCommand(root, "node")
	if err != nil {
		t.Fatalf("resolveCommand(bare) error = %v", err)
	}
	if got != "node" {
		t.Fatalf("resolveCommand(bare) = %q, want %q", got, "node")
	}

	if _, err := manager.resolveCommand(root, "../outside/tool"); err == nil || !strings.Contains(err.Error(), "escapes extension root") {
		t.Fatalf("resolveCommand(escape) error = %v, want extension-root escape failure", err)
	}

	if _, err := resolveResourcePath(root, "../skills"); err == nil || !strings.Contains(err.Error(), "escapes extension root") {
		t.Fatalf("resolveResourcePath(escape) error = %v, want extension-root escape failure", err)
	}

	resourceRoot, err := resolveResourcePath(root, "skills")
	if err != nil {
		t.Fatalf("resolveResourcePath(within root) error = %v", err)
	}
	if resourceRoot != filepath.Join(root, "skills") {
		t.Fatalf("resolveResourcePath(within root) = %q, want %q", resourceRoot, filepath.Join(root, "skills"))
	}
}

func TestManagerResolveEnvMapUsesSafeBaselineOnly(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, WithGetenv(func(key string) string {
		switch key {
		case "PATH":
			return "/usr/bin:/bin"
		case "HOME":
			return "/tmp/home"
		case "LANG":
			return "en_US.UTF-8"
		case "SECRET_TOKEN":
			return "top-secret"
		default:
			return ""
		}
	}))

	env, err := manager.resolveEnvMap(t.TempDir(), map[string]string{
		"APP_MODE": "sandbox",
		"PATH":     "/custom/bin",
	})
	if err != nil {
		t.Fatalf("resolveEnvMap() error = %v", err)
	}

	decoded := envListToMap(t, env)
	if decoded["PATH"] != "/custom/bin" {
		t.Fatalf("PATH = %q, want %q", decoded["PATH"], "/custom/bin")
	}
	if decoded["HOME"] != "/tmp/home" {
		t.Fatalf("HOME = %q, want %q", decoded["HOME"], "/tmp/home")
	}
	if decoded["LANG"] != "en_US.UTF-8" {
		t.Fatalf("LANG = %q, want %q", decoded["LANG"], "en_US.UTF-8")
	}
	if decoded["APP_MODE"] != "sandbox" {
		t.Fatalf("APP_MODE = %q, want %q", decoded["APP_MODE"], "sandbox")
	}
	if _, ok := decoded["SECRET_TOKEN"]; ok {
		t.Fatalf("resolveEnvMap() leaked SECRET_TOKEN in %#v", decoded)
	}
}

func TestManagerCloneExtensionReturnsIsolatedSnapshot(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil)
	ext := &managedExtension{
		info: ExtensionInfo{
			Name:    "snapshot",
			Version: "1.0.0",
			Source:  SourceUser,
			Enabled: true,
			Capabilities: CapabilitiesConfig{
				Provides: []string{"memory.backend"},
			},
			Actions: ActionsConfig{
				Requires: []string{"sessions/list"},
			},
		},
		manifest: &Manifest{
			Name:    "snapshot",
			Version: "1.0.0",
			Resources: ResourcesConfig{
				Skills: []string{"skills/"},
			},
			Capabilities: CapabilitiesConfig{
				Provides: []string{"memory.backend"},
			},
			Actions: ActionsConfig{
				Requires: []string{"sessions/list"},
			},
			Subprocess: SubprocessConfig{
				Command: "snapshot-extension",
				Args:    []string{"--config", "snapshot.toml"},
				Env: map[string]string{
					"TOKEN": "value",
				},
			},
			Security: SecurityConfig{
				Capabilities: []string{"memory.read"},
			},
		},
		skills: []*skillspkg.Skill{{
			Meta: skillspkg.SkillMeta{
				Name:        "snapshot-skill",
				Description: "Snapshot skill",
				Metadata: map[string]any{
					"nested": map[string]any{"value": "original"},
				},
			},
			Hooks: []hookspkg.HookDecl{{
				Name: "snapshot-hook",
				Args: []string{"cleanup"},
				Env:  map[string]string{"PHASE": "stop"},
			}},
			MCPServers: []skillspkg.MCPServerDecl{{
				Name:    "snapshot-server",
				Command: "server",
				Args:    []string{"--once"},
				Env:     map[string]string{"ROOT": "/tmp/original"},
			}},
			Provenance: &skillspkg.Provenance{
				Hash: "hash-original",
			},
		}},
		initialize: &subprocess.InitializeResponse{
			ImplementedMethods:  []string{"shutdown"},
			SupportedHookEvents: []string{"turn.start"},
			AcceptedCapabilities: subprocess.AcceptedCapabilities{
				Provides: []string{"memory.backend"},
				Actions:  []extensionprotocol.HostAPIMethod{"sessions/list"},
				Security: []string{"memory.read"},
			},
		},
	}

	clone := manager.cloneExtension(ext)
	if clone == nil {
		t.Fatal("cloneExtension() = nil, want snapshot")
	}

	clone.Info.Capabilities.Provides[0] = "changed"
	clone.Info.Actions.Requires[0] = "changed"
	clone.Manifest.Resources.Skills[0] = "changed"
	clone.Manifest.Subprocess.Env["TOKEN"] = "changed"
	clone.Skills[0].Meta.Name = "changed"
	clone.Skills[0].Meta.Metadata["nested"].(map[string]any)["value"] = "changed"
	clone.Skills[0].Hooks[0].Args[0] = "changed"
	clone.Skills[0].MCPServers[0].Env["ROOT"] = "/tmp/changed"
	clone.Skills[0].Provenance.Hash = "hash-changed"
	clone.InitializeResult.ImplementedMethods[0] = "changed"
	clone.InitializeResult.AcceptedCapabilities.Provides[0] = "changed"

	if ext.info.Capabilities.Provides[0] != "memory.backend" {
		t.Fatalf("original capabilities mutated to %#v", ext.info.Capabilities.Provides)
	}
	if ext.info.Actions.Requires[0] != "sessions/list" {
		t.Fatalf("original actions mutated to %#v", ext.info.Actions.Requires)
	}
	if ext.manifest.Resources.Skills[0] != "skills/" {
		t.Fatalf("original manifest resources mutated to %#v", ext.manifest.Resources.Skills)
	}
	if ext.manifest.Subprocess.Env["TOKEN"] != "value" {
		t.Fatalf("original manifest env mutated to %#v", ext.manifest.Subprocess.Env)
	}
	if ext.skills[0].Meta.Name != "snapshot-skill" {
		t.Fatalf("original skill name mutated to %q", ext.skills[0].Meta.Name)
	}
	if ext.skills[0].Meta.Metadata["nested"].(map[string]any)["value"] != "original" {
		t.Fatalf("original skill metadata mutated to %#v", ext.skills[0].Meta.Metadata)
	}
	if ext.skills[0].Hooks[0].Args[0] != "cleanup" {
		t.Fatalf("original skill hook args mutated to %#v", ext.skills[0].Hooks[0].Args)
	}
	if ext.skills[0].MCPServers[0].Env["ROOT"] != "/tmp/original" {
		t.Fatalf("original skill MCP env mutated to %#v", ext.skills[0].MCPServers[0].Env)
	}
	if ext.skills[0].Provenance.Hash != "hash-original" {
		t.Fatalf("original skill provenance mutated to %#v", ext.skills[0].Provenance)
	}
	if ext.initialize.ImplementedMethods[0] != "shutdown" {
		t.Fatalf("original initialize methods mutated to %#v", ext.initialize.ImplementedMethods)
	}
	if ext.initialize.AcceptedCapabilities.Provides[0] != "memory.backend" {
		t.Fatalf("original initialize provides mutated to %#v", ext.initialize.AcceptedCapabilities.Provides)
	}
}

func TestManagerDirectPhaseAndMonitorBranches(t *testing.T) {
	t.Parallel()

	withDaemonVersion(t, "0.5.0")
	manager := NewManager(nil, WithGetenv(func(key string) string {
		if key == "EXT_TOKEN" {
			return "resolved-token"
		}
		return ""
	}))

	discover := &managedExtension{info: ExtensionInfo{Name: "ext-discover"}}
	if err := manager.discoverExtension(discover); err == nil {
		t.Fatal("discoverExtension() error = nil, want missing manifest path")
	}
	discover.info.ManifestPath = manifestTOMLFileName
	if err := manager.discoverExtension(discover); err == nil {
		t.Fatal("discoverExtension() error = nil, want invalid relative manifest path")
	}

	resolved, err := manager.resolveString("/tmp/ext", "{{config_dir}}/{{env:EXT_TOKEN}}")
	if err != nil {
		t.Fatalf("resolveString() error = %v", err)
	}
	if got, want := resolved, "/tmp/ext/resolved-token"; got != want {
		t.Fatalf("resolveString() = %q, want %q", got, want)
	}
	if _, err := manager.resolveString("/tmp/ext", "{{env:EXT_TOKEN"); err == nil {
		t.Fatal("resolveString(invalid template) error = nil, want template error")
	}

	validate := &managedExtension{info: ExtensionInfo{Name: "ext-validate", Source: SourceUser}}
	if err := manager.validateExtension(validate); err == nil {
		t.Fatal("validateExtension(nil manifest) error = nil, want manifest-required error")
	}
	validate.manifest = &Manifest{Name: "other", Version: "1.0.0", MinAGHVersion: "0.5.0"}
	if err := manager.validateExtension(validate); err == nil {
		t.Fatal("validateExtension(name mismatch) error = nil, want mismatch error")
	}
	validate.info.Version = "1.0.0"
	validate.manifest = &Manifest{Name: "ext-validate", Version: "2.0.0", MinAGHVersion: "0.5.0"}
	if err := manager.validateExtension(validate); err == nil {
		t.Fatal("validateExtension(version mismatch) error = nil, want mismatch error")
	}
	validate.info.Version = ""
	validate.manifest = &Manifest{
		Name:          "ext-validate",
		Version:       "1.0.0",
		MinAGHVersion: "0.5.0",
		Actions:       ActionsConfig{Requires: []string{"sessions/list"}},
	}
	if err := manager.validateExtension(validate); err == nil {
		t.Fatal("validateExtension(missing subprocess command) error = nil, want subprocess validation error")
	}

	lite := &managedExtension{
		info: ExtensionInfo{Name: "ext-lite", Source: SourceUser, Enabled: true},
		manifest: &Manifest{
			Name:          "ext-lite",
			Version:       "1.0.0",
			MinAGHVersion: "0.5.0",
		},
	}
	if err := manager.validateExtension(lite); err != nil {
		t.Fatalf("validateExtension(lite) error = %v", err)
	}
	if err := manager.initializeExtension(context.Background(), lite); err != nil {
		t.Fatalf("initializeExtension(lite) error = %v", err)
	}
	manager.activateExtension(lite)
	if !lite.active || lite.phase != ExtensionPhaseActivate {
		t.Fatalf("lite extension after activate = %#v, want active activate-phase extension", lite)
	}

	rootDir := t.TempDir()
	skillsDir := filepath.Join(rootDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", skillsDir, err)
	}
	writeFile(t, filepath.Join(skillsDir, "skill.md"), managerSkillFile("missing-registry", "Needs registry"))
	skillExt := &managedExtension{
		info:    ExtensionInfo{Name: "ext-skills", Source: SourceUser},
		rootDir: rootDir,
		manifest: &Manifest{
			Name:          "ext-skills",
			Version:       "1.0.0",
			MinAGHVersion: "0.5.0",
			Resources: ResourcesConfig{
				Skills: []string{"skills"},
			},
		},
	}
	if err := manager.registerExtension(context.Background(), skillExt); err == nil {
		t.Fatal("registerExtension(skills without registry) error = nil, want registry-required error")
	}

	manager.capChecker.Register("ext-host", SourceUser, &Manifest{
		Actions:  ActionsConfig{Requires: []string{"sessions/list"}},
		Security: SecurityConfig{Capabilities: []string{"session.read"}},
	})
	allowed := manager.wrapHostHandler("ext-host", "sessions/list", func(_ context.Context, _ json.RawMessage) (any, error) {
		return "ok", nil
	})
	result, err := allowed(context.Background(), json.RawMessage(`{}`))
	if err != nil || result != "ok" {
		t.Fatalf("wrapHostHandler allowed call = (%v, %v), want (ok, nil)", result, err)
	}
	denied := manager.wrapHostHandler("ext-denied", "sessions/list", func(_ context.Context, _ json.RawMessage) (any, error) {
		return "never", nil
	})
	if _, err := denied(context.Background(), nil); err == nil {
		t.Fatal("wrapHostHandler denied call error = nil, want capability denial")
	}

	monitorManager := NewManager(nil, withHealthPollBounds(time.Millisecond, time.Millisecond))
	monitorCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitorManager.lifecycleCtx = monitorCtx
	unhealthy := newFakeProcess(808)
	unhealthy.health = subprocess.HealthState{
		Healthy:       false,
		Message:       "probe failed",
		LastError:     "rpc timeout",
		LastCheckedAt: time.Now().UTC(),
	}
	monitorManager.extensions = map[string]*managedExtension{
		"ext-health": {
			info:           ExtensionInfo{Name: "ext-health"},
			process:        unhealthy,
			generation:     1,
			healthInterval: 4 * time.Millisecond,
			runtime:        subprocess.InitializeRuntime{ShutdownTimeoutMS: 7},
		},
	}

	reason, shouldRecover := monitorManager.monitorProcess("ext-health", 1, unhealthy, 4*time.Millisecond)
	if !shouldRecover {
		t.Fatal("monitorProcess() shouldRecover = false, want true for unhealthy process")
	}
	if reason == nil || !strings.Contains(reason.Error(), "health check failed") || !strings.Contains(reason.Error(), "rpc timeout") {
		t.Fatalf("monitorProcess() reason = %v, want joined health failure detail", reason)
	}
	if unhealthy.shutdownCnt != 1 {
		t.Fatalf("Shutdown count = %d, want 1 after unhealthy process", unhealthy.shutdownCnt)
	}
}

type managerFixture struct {
	dir      string
	manifest *Manifest
	checksum string
}

type managerManifestOptions struct {
	command      string
	args         []string
	withEnv      map[string]string
	withSkills   bool
	withAgents   bool
	withHooks    bool
	withMCP      bool
	minVersion   string
	capabilities []string
	actions      []string
	security     []string
	shutdown     time.Duration
}

type fakeLauncher struct {
	mu     sync.Mutex
	queue  []*fakeProcess
	config []subprocess.LaunchConfig
}

func (l *fakeLauncher) launch(ctx context.Context, cfg subprocess.LaunchConfig) (processHandle, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config = append(l.config, cfg)
	if len(l.queue) == 0 {
		return nil, errors.New("no fake processes queued")
	}
	process := l.queue[0]
	l.queue = l.queue[1:]
	return process, nil
}

func (l *fakeLauncher) launchCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.config)
}

type fakeProcess struct {
	mu          sync.Mutex
	pid         int
	done        chan struct{}
	waitErr     error
	closed      bool
	initReqs    []subprocess.InitializeRequest
	initResp    subprocess.InitializeResponse
	initErr     error
	initHook    func()
	health      subprocess.HealthState
	handlers    map[string]subprocess.HandlerFunc
	shutdownFn  func(context.Context) error
	shutdownCnt int
}

func newFakeProcess(pid int) *fakeProcess {
	return &fakeProcess{
		pid:      pid,
		done:     make(chan struct{}),
		handlers: make(map[string]subprocess.HandlerFunc),
		health: subprocess.HealthState{
			Healthy: true,
		},
	}
}

func (p *fakeProcess) HandleMethod(method string, handler subprocess.HandlerFunc) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[method] = handler
	return nil
}

func (p *fakeProcess) Initialize(_ context.Context, req subprocess.InitializeRequest) (subprocess.InitializeResponse, error) {
	p.mu.Lock()
	p.initReqs = append(p.initReqs, req)
	resp := p.initResp
	err := p.initErr
	hook := p.initHook
	p.mu.Unlock()

	if err != nil {
		return subprocess.InitializeResponse{}, err
	}
	if resp.ProtocolVersion == "" {
		resp = fakeInitializeResponse(req)
	}
	if hook != nil {
		hook()
	}
	return resp, nil
}

func (p *fakeProcess) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	p.shutdownCnt++
	shutdownFn := p.shutdownFn
	p.mu.Unlock()

	if shutdownFn != nil {
		return shutdownFn(ctx)
	}
	p.close(nil)
	return nil
}

func (p *fakeProcess) Done() <-chan struct{} {
	return p.done
}

func (p *fakeProcess) Wait() error {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waitErr
}

func (p *fakeProcess) HealthState() subprocess.HealthState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.health
}

func (p *fakeProcess) PID() int {
	return p.pid
}

func (p *fakeProcess) initRequests() []subprocess.InitializeRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]subprocess.InitializeRequest(nil), p.initReqs...)
}

func (p *fakeProcess) crash(err error) {
	p.close(err)
}

func (p *fakeProcess) close(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	p.waitErr = err
	close(p.done)
}

func fakeInitializeResponse(req subprocess.InitializeRequest) subprocess.InitializeResponse {
	implemented := append([]string{"health_check", "shutdown"}, req.Methods.ExtensionServices...)
	slices.Sort(implemented)
	return subprocess.InitializeResponse{
		ProtocolVersion: req.ProtocolVersion,
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    req.Extension.Name,
			Version: req.Extension.Version,
		},
		AcceptedCapabilities: subprocess.AcceptedCapabilities{
			Provides: slices.Clone(req.Capabilities.Provides),
			Actions:  slices.Clone(req.Capabilities.GrantedActions),
			Security: slices.Clone(req.Capabilities.GrantedSecurity),
		},
		ImplementedMethods:  implemented,
		SupportedHookEvents: []string{string(hookspkg.HookTurnStart)},
		Supports: subprocess.InitializeSupports{
			HealthCheck: true,
		},
	}
}

type extensionHelperServer struct {
	scenario string
	marker   string

	mu         sync.Mutex
	writer     *bufio.Writer
	pendingReq string
}

func newExtensionHelperServer(scenario string, marker string) *extensionHelperServer {
	return &extensionHelperServer{
		scenario: strings.TrimSpace(scenario),
		marker:   marker,
		writer:   bufio.NewWriter(os.Stdout),
	}
}

func (h *extensionHelperServer) run() int {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024), 10<<20)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var envelope map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			return 1
		}

		if _, ok := envelope["method"]; ok {
			var req helperRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				return 1
			}
			if err := h.handleRequest(req); err != nil {
				return 1
			}
			continue
		}

		var response helperResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			return 1
		}
		h.handleResponse(response)
	}

	if err := scanner.Err(); err != nil {
		return 1
	}
	return 0
}

func (h *extensionHelperServer) handleRequest(req helperRequest) error {
	switch req.Method {
	case "initialize":
		var params subprocess.InitializeRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		response := fakeInitializeResponse(params)
		if err := h.sendResult(req.ID, response); err != nil {
			return err
		}

		switch h.scenario {
		case "host_call":
			h.mu.Lock()
			h.pendingReq = "host-1"
			h.mu.Unlock()
			go func() {
				time.Sleep(15 * time.Millisecond)
				_ = h.sendRequest("host-1", "sessions/list", map[string]string{"workspace": "ext"})
			}()
			return nil
		case "auto_exit":
			if h.marker != "" {
				f, err := os.OpenFile(h.marker, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
				if err == nil {
					_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
					_ = f.Close()
				}
			}
			go func() {
				time.Sleep(15 * time.Millisecond)
				os.Exit(1)
			}()
		}
		return nil
	case "health_check":
		return h.sendResult(req.ID, subprocess.HealthCheckResponse{Healthy: true})
	case "shutdown":
		if h.scenario == "shutdown_hang" {
			select {}
		}
		if h.marker != "" {
			_ = os.WriteFile(h.marker, []byte("shutdown"), 0o600)
		}
		if err := h.sendResult(req.ID, subprocess.ShutdownResponse{Acknowledged: true}); err != nil {
			return err
		}
		return nil
	default:
		return h.sendResult(req.ID, map[string]any{})
	}
}

func (h *extensionHelperServer) handleResponse(resp helperResponse) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pendingReq == "" || fmt.Sprint(resp.ID) != h.pendingReq || h.marker == "" {
		return
	}
	if len(resp.Result) > 0 {
		_ = os.WriteFile(h.marker, resp.Result, 0o600)
	}
	h.pendingReq = ""
}

func (h *extensionHelperServer) sendRequest(id string, method string, params any) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	return h.write(payload)
}

func (h *extensionHelperServer) sendResult(id any, result any) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	return h.write(payload)
}

func (h *extensionHelperServer) write(payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, err := h.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	return h.writer.Flush()
}

type helperRequest struct {
	ID     any             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type helperResponse struct {
	ID     any             `json:"id"`
	Result json.RawMessage `json:"result"`
}

func createManagerTestExtension(t *testing.T, manifestContent string, files map[string]string) managerFixture {
	t.Helper()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, manifestTOMLFileName), manifestContent)
	for relPath, content := range files {
		absPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(absPath), err)
		}
		writeFile(t, absPath, content)
	}

	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest(%q) error = %v", dir, err)
	}
	checksum, err := ComputeDirectoryChecksum(dir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(%q) error = %v", dir, err)
	}

	return managerFixture{
		dir:      dir,
		manifest: manifest,
		checksum: checksum,
	}
}

func installManagerFixture(t *testing.T, registry *Registry, fixture managerFixture, source ExtensionSource, enabled bool) {
	t.Helper()

	if err := registry.installWithSource(fixture.manifest, fixture.dir, fixture.checksum, source); err != nil {
		t.Fatalf("installWithSource(%q) error = %v", fixture.manifest.Name, err)
	}
	if !enabled {
		if err := registry.Disable(fixture.manifest.Name); err != nil {
			t.Fatalf("Disable(%q) error = %v", fixture.manifest.Name, err)
		}
	}
}

func managerSkillFile(name string, description string) string {
	return fmt.Sprintf(
		`---
name: %s
description: %s
---
Use this skill carefully.
`,
		name,
		description,
	)
}

func managerAgentFile(name string) string {
	return fmt.Sprintf(
		`---
name: %s
provider: codex
---
Prompt body.
`,
		name,
	)
}

func managerTestManifest(name string, opts managerManifestOptions) string {
	minVersion := opts.minVersion
	if minVersion == "" {
		minVersion = "0.5.0"
	}
	command := opts.command
	if command == "" {
		command = "fake-extension"
	}
	capabilities := opts.capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"memory.backend"}
	}
	actions := opts.actions
	if len(actions) == 0 {
		actions = []string{"sessions/list"}
	}
	security := opts.security
	if len(security) == 0 {
		security = []string{"session.read"}
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, `[extension]
name = %q
version = "0.2.1"
description = "Extension manager test fixture"
min_agh_version = %q

[resources]
`, name, minVersion)
	if opts.withSkills {
		builder.WriteString(`skills = ["skills/"]
`)
	}
	if opts.withAgents {
		builder.WriteString(`agents = ["agents/"]
`)
	}
	if opts.withHooks {
		builder.WriteString(`
[[resources.hooks]]
name = "` + name + `-hook"
event = "turn.start"
mode = "sync"
executor.kind = "subprocess"
executor.command = "./bin/hook"
executor.args = ["--hook"]
`)
	}
	if opts.withMCP {
		builder.WriteString(`
[resources.mcp_servers]
[resources.mcp_servers.kubectl]
command = "mcp-kubectl"
args = ["--context", "prod"]
`)
	}

	builder.WriteString(`
[capabilities]
provides = ` + tomlStringArray(capabilities) + `

[actions]
requires = ` + tomlStringArray(actions) + `

[subprocess]
command = ` + fmt.Sprintf("%q", command) + `
`)
	if len(opts.args) > 0 {
		builder.WriteString(`args = ` + tomlStringArray(opts.args) + `
`)
	}
	if opts.shutdown > 0 {
		builder.WriteString(`shutdown_timeout = "` + opts.shutdown.String() + `"
`)
	}
	if len(opts.withEnv) > 0 {
		builder.WriteString(`
[subprocess.env]
`)
		keys := make([]string, 0, len(opts.withEnv))
		for key := range opts.withEnv {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			fmt.Fprintf(&builder, "%s = %q\n", key, opts.withEnv[key])
		}
	}
	builder.WriteString(`
[security]
capabilities = ` + tomlStringArray(security) + `
`)

	return builder.String()
}

func helperCommand(t *testing.T) string {
	t.Helper()
	command, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	return command
}

func helperArgs() []string {
	return []string{
		"-test.run=TestExtensionManagerHelperProcess",
	}
}

func helperEnv(scenario string, markerPath string) map[string]string {
	env := map[string]string{
		extensionHelperEnvKey:      "1",
		extensionHelperScenarioKey: scenario,
	}
	if strings.TrimSpace(markerPath) != "" {
		env[extensionHelperMarkerKey] = markerPath
	}
	return env
}

func envListToMap(t testing.TB, env []string) map[string]string {
	t.Helper()

	decoded := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("env entry %q missing '=' separator", entry)
		}
		decoded[key] = value
	}
	return decoded
}

func waitForManagerCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for manager condition")
}
