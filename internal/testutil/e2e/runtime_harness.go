package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	"golang.org/x/sys/execabs"
)

const (
	defaultStartTimeout = 20 * time.Second
	defaultPollInterval = 100 * time.Millisecond
	maxStartAttempts    = 3
	windowsGOOS         = "windows"
	daemonBinaryEnvVar  = "AGH_TEST_DAEMON_BIN"
	runtimeManifestName = "runtime-manifest.json"
)

var errDaemonExitedBeforeReadiness = errors.New("daemon exited before readiness")

var (
	buildBinaryMu   sync.Mutex
	builtBinaryPath string
)

// RuntimeHarnessOptions configures one isolated daemon runtime.
type RuntimeHarnessOptions struct {
	BinaryPath       string
	HomePaths        aghconfig.HomePaths
	ConfigSeed       ConfigSeedOptions
	MockAgents       []MockAgentSpec
	Workspace        WorkspaceSeedOptions
	Env              map[string]string
	EnableNetwork    bool
	StartTimeout     time.Duration
	PollInterval     time.Duration
	ResolveWorkspace bool
}

type runtimeLayout struct {
	HomePaths     aghconfig.HomePaths
	Config        aghconfig.Config
	WorkspaceRoot string
	Artifacts     *ArtifactCollector
	MockAgents    map[string]acpmock.Registration
	Env           []string
}

// RuntimeHarness exposes the started daemon and its public product surfaces.
type RuntimeHarness struct {
	HomePaths     aghconfig.HomePaths
	Config        aghconfig.Config
	BinaryPath    string
	Artifacts     *ArtifactCollector
	WorkspaceRoot string
	WorkspaceID   string
	MockAgents    map[string]acpmock.Registration

	HTTPBaseURL string
	HTTPClient  *http.Client

	UDSBaseURL string
	UDSClient  *http.Client

	CLI *CLIClient

	process *exec.Cmd
	waitCh  <-chan error

	processLogPath string

	stopOnce sync.Once
	stopErr  error

	processWaitMu sync.Mutex
	processExited bool
	processErr    error
}

// CLIClient shells out to the real `agh` binary against the isolated runtime.
type CLIClient struct {
	binaryPath string
	env        []string
	workdir    string
}

// SSEEvent captures one parsed server-sent event record.
type SSEEvent struct {
	ID    string
	Event string
	Data  []byte
}

// StartRuntimeHarness boots an isolated daemon through the real CLI startup path.
func StartRuntimeHarness(t testing.TB, opts RuntimeHarnessOptions) *RuntimeHarness {
	t.Helper()

	layout := prepareRuntimeLayout(t, opts)
	binaryPath := strings.TrimSpace(opts.BinaryPath)
	if binaryPath == "" {
		binaryPath = buildAGHBinary(t)
	}
	env, err := withRuntimeCLIEnv(layout.HomePaths, layout.Env, binaryPath)
	if err != nil {
		t.Fatalf("prepare runtime CLI env error = %v", err)
	}
	layout.Env = env
	harness := newRuntimeHarness(t, &layout, binaryPath)
	startRuntimeProcess(t, harness, layout.Env, opts)

	workspace, err := harness.ResolveWorkspace(context.Background(), layout.WorkspaceRoot)
	if err != nil {
		if stopErr := harness.Stop(context.Background()); stopErr != nil {
			t.Fatalf("stop runtime harness after workspace resolve failure error = %v", stopErr)
		}
		t.Fatalf("resolve workspace %q error = %v", layout.WorkspaceRoot, err)
	}
	harness.WorkspaceID = workspace.ID
	if _, err := harness.WriteRuntimeManifest(); err != nil {
		if stopErr := harness.Stop(context.Background()); stopErr != nil {
			t.Fatalf("stop runtime harness after runtime manifest failure error = %v", stopErr)
		}
		t.Fatalf("write runtime manifest error = %v", err)
	}

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := harness.Stop(stopCtx); err != nil {
			t.Fatalf("stop runtime harness error = %v", err)
		}
	})

	return harness
}

func startRuntimeProcess(
	t testing.TB,
	harness *RuntimeHarness,
	env []string,
	opts RuntimeHarnessOptions,
) {
	t.Helper()

	startTimeout := defaultDuration(opts.StartTimeout, defaultStartTimeout)
	pollInterval := defaultDuration(opts.PollInterval, defaultPollInterval)

	for attempt := 1; attempt <= maxStartAttempts; attempt++ {
		startDaemonProcess(t, harness, env)

		readyCtx, cancel := context.WithTimeout(context.Background(), startTimeout)
		err := harness.waitForReady(readyCtx, pollInterval)
		cancel()
		if err == nil {
			return
		}
		retryHTTPPort, retrySocketPath := harness.readinessFailureRetryReasons(err)

		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		cleanupErr := harness.cleanupFailedStart(cleanupCtx)
		cleanupCancel()
		if attempt == maxStartAttempts || (!retryHTTPPort && !retrySocketPath) {
			if cleanupErr != nil {
				t.Fatalf("cleanup failed start after readiness error = %v (readiness error = %v)", cleanupErr, err)
			}
			t.Fatalf("wait for daemon readiness error = %v", err)
		}
		if cleanupErr != nil {
			t.Fatalf("cleanup failed start before retry error = %v (readiness error = %v)", cleanupErr, err)
		}
		if retryHTTPPort {
			if err := harness.reseedRuntimeHTTPPort(t); err != nil {
				t.Fatalf("reseed runtime HTTP port error = %v", err)
			}
		}
		if retrySocketPath {
			if err := harness.reseedRuntimeSocketPath(t); err != nil {
				t.Fatalf("reseed runtime UDS socket error = %v", err)
			}
		}
	}
}

func newRuntimeHarness(t testing.TB, layout *runtimeLayout, binaryPath string) *RuntimeHarness {
	t.Helper()

	repoRoot := mustRepoRoot(t)
	processLogPath := filepath.Join(layout.Artifacts.RootDir(), "daemon-process.log")
	return &RuntimeHarness{
		HomePaths:     layout.HomePaths,
		Config:        layout.Config,
		BinaryPath:    binaryPath,
		Artifacts:     layout.Artifacts,
		WorkspaceRoot: layout.WorkspaceRoot,
		MockAgents:    cloneMockAgentRegistrations(layout.MockAgents),
		HTTPBaseURL:   fmt.Sprintf("http://%s:%d", layout.Config.HTTP.Host, layout.Config.HTTP.Port),
		HTTPClient:    newHTTPClient(),
		UDSBaseURL:    "http://unix",
		UDSClient:     newUDSClient(layout.Config.Daemon.Socket),
		CLI: &CLIClient{
			binaryPath: binaryPath,
			env:        layout.Env,
			workdir:    repoRoot,
		},
		processLogPath: processLogPath,
	}
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
}

func newUDSClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

func startDaemonProcess(t testing.TB, harness *RuntimeHarness, env []string) {
	t.Helper()

	processLogPath := harness.processLogPath
	if strings.TrimSpace(processLogPath) == "" {
		processLogPath = filepath.Join(harness.Artifacts.RootDir(), "daemon-process.log")
	}
	processLog, err := os.Create(processLogPath)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", processLogPath, err)
	}

	// #nosec G204 -- test harness intentionally executes the built agh binary against isolated test state.
	cmd := execabs.CommandContext(context.Background(), harness.BinaryPath, "daemon", "start", "--foreground")
	cmd.Env = append([]string(nil), env...)
	cmd.Stdout = processLog
	cmd.Stderr = processLog
	cmd.Dir = mustRepoRoot(t)
	if err := cmd.Start(); err != nil {
		_ = processLog.Close()
		t.Fatalf("start daemon process error = %v", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
		close(waitCh)
		_ = processLog.Close()
	}()

	harness.process = cmd
	harness.waitCh = waitCh
}

func prepareRuntimeLayout(t testing.TB, opts RuntimeHarnessOptions) runtimeLayout {
	t.Helper()

	homePaths := opts.HomePaths
	if strings.TrimSpace(homePaths.HomeDir) == "" {
		homePaths = NewHomePaths(t)
	}
	configSeed := opts.ConfigSeed
	originalMutate := configSeed.Mutate
	configSeed.Mutate = func(cfg *aghconfig.Config) {
		if originalMutate != nil {
			originalMutate(cfg)
		}
		if opts.EnableNetwork {
			cfg.Network.Enabled = true
		}
	}
	config := SeedConfig(t, homePaths, configSeed)
	workspaceRoot := SeedWorkspace(t, opts.Workspace)
	artifacts := NewArtifactCollector(t)
	mockAgents := registerMockAgents(t, homePaths, artifacts, opts.MockAgents)

	return runtimeLayout{
		HomePaths:     homePaths,
		Config:        config,
		WorkspaceRoot: workspaceRoot,
		Artifacts:     artifacts,
		MockAgents:    mockAgents,
		Env:           runtimeEnv(homePaths, opts.Env),
	}
}

// Stop shuts down the started daemon and waits for process exit.
func (h *RuntimeHarness) Stop(ctx context.Context) error {
	if h == nil {
		return nil
	}

	h.stopOnce.Do(func() {
		if err := h.stopWithContext(ctx); err != nil {
			h.stopErr = err
		}
	})
	return h.stopErr
}

func (h *RuntimeHarness) stopWithContext(ctx context.Context) error {
	defer closeIdleConnections(h.HTTPClient)
	defer closeIdleConnections(h.UDSClient)

	if h.process == nil && h.waitCh == nil {
		return nil
	}

	if exited, err := h.pollExit(); exited {
		return err
	}

	signaledProcess := false
	if h.process != nil && h.process.Process != nil {
		if signalErr := h.process.Process.Signal(
			os.Interrupt,
		); signalErr != nil &&
			!errors.Is(signalErr, os.ErrProcessDone) {
			return fmt.Errorf("interrupt daemon process: %w", signalErr)
		}
		signaledProcess = true
	} else if h.CLI != nil {
		if _, _, err := h.CLI.Run(ctx, "daemon", "stop", "-o", "json"); err != nil {
			return fmt.Errorf("stop daemon via CLI: %w", err)
		}
	}

	waitErr := h.waitForExit(ctx)
	if waitErr == nil {
		return nil
	}
	if signaledProcess && !errors.Is(waitErr, context.DeadlineExceeded) {
		return nil
	}

	if h.process != nil && h.process.Process != nil {
		if killErr := h.process.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return fmt.Errorf("kill daemon process: %w", killErr)
		}
	}
	killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if killWaitErr := h.waitForExit(killCtx); killWaitErr != nil && !errors.Is(killWaitErr, context.DeadlineExceeded) {
		return killWaitErr
	}
	return waitErr
}

func (h *RuntimeHarness) cleanupFailedStart(ctx context.Context) error {
	if h == nil {
		return nil
	}

	if h.process != nil || h.waitCh != nil {
		exited, err := h.pollExit()
		if err != nil && !exited {
			return fmt.Errorf("poll failed runtime process exit: %w", err)
		}
		if !exited {
			if err := h.stopWithContext(ctx); err != nil {
				return fmt.Errorf("stop failed runtime start: %w", err)
			}
		}
	}
	h.resetProcessState()

	for _, path := range []string{
		strings.TrimSpace(h.HomePaths.DaemonInfo),
		strings.TrimSpace(h.Config.Daemon.Socket),
	} {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove runtime start artifact %q: %w", path, err)
		}
	}

	return nil
}

func (h *RuntimeHarness) resetProcessState() {
	if h == nil {
		return
	}

	h.processWaitMu.Lock()
	defer h.processWaitMu.Unlock()

	h.process = nil
	h.waitCh = nil
	h.processExited = false
	h.processErr = nil
}

func (h *RuntimeHarness) readinessFailureShouldRetry(err error) bool {
	retryHTTPPort, retrySocketPath := h.readinessFailureRetryReasons(err)
	return retryHTTPPort || retrySocketPath
}

func (h *RuntimeHarness) readinessFailureRetryReasons(err error) (retryHTTPPort bool, retrySocketPath bool) {
	if h == nil || err == nil {
		return false, false
	}
	if !errors.Is(err, errDaemonExitedBeforeReadiness) {
		return false, false
	}

	processLog, readErr := h.readProcessLog()
	if readErr != nil {
		return false, false
	}
	return processLogHasHTTPPortConflict(processLog), processLogHasSocketPathConflict(processLog)
}

func processLogHasHTTPPortConflict(processLog string) bool {
	return strings.Contains(processLog, "listen tcp") && strings.Contains(processLog, "bind: address already in use")
}

func processLogHasSocketPathConflict(processLog string) bool {
	return strings.Contains(processLog, "listen unix") &&
		(strings.Contains(processLog, "bind: file exists") ||
			strings.Contains(processLog, "bind: address already in use"))
}

func (h *RuntimeHarness) readProcessLog() (string, error) {
	if h == nil {
		return "", errors.New("runtime harness is required")
	}
	path := strings.TrimSpace(h.processLogPath)
	if path == "" {
		return "", errors.New("runtime harness process log path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read process log %q: %w", path, err)
	}
	return string(data), nil
}

func (h *RuntimeHarness) reseedRuntimeHTTPPort(t testing.TB) error {
	t.Helper()

	if h == nil {
		return errors.New("runtime harness is required")
	}

	nextPort := testutil.FreeTCPPort(t)
	for nextPort == h.Config.HTTP.Port {
		nextPort = testutil.FreeTCPPort(t)
	}

	h.Config.HTTP.Port = nextPort
	h.HTTPBaseURL = fmt.Sprintf("http://%s:%d", h.Config.HTTP.Host, nextPort)
	return writeSeedConfigFile(h.HomePaths, &h.Config)
}

func (h *RuntimeHarness) reseedRuntimeSocketPath(t testing.TB) error {
	t.Helper()

	if h == nil {
		return errors.New("runtime harness is required")
	}

	previousSocket := h.Config.Daemon.Socket
	nextSocket := shortSocketPath(t)
	for nextSocket == previousSocket {
		nextSocket = shortSocketPath(t)
	}

	h.Config.Daemon.Socket = nextSocket
	h.UDSClient = newUDSClient(nextSocket)
	return writeSeedConfigFile(h.HomePaths, &h.Config)
}

// RuntimeManifestPath returns the stable runtime-manifest path under the harness artifact root.
func (h *RuntimeHarness) RuntimeManifestPath() string {
	if h == nil || h.Artifacts == nil {
		return ""
	}
	return filepath.Join(h.Artifacts.RootDir(), runtimeManifestName)
}

// RuntimeManifest returns the current runtime-manifest snapshot without writing it.
func (h *RuntimeHarness) RuntimeManifest() (RuntimeArtifactManifest, error) {
	if h == nil {
		return RuntimeArtifactManifest{}, errors.New("runtime harness is required")
	}
	if h.Artifacts == nil {
		return RuntimeArtifactManifest{}, errors.New("runtime harness artifacts are required")
	}

	runDirectories, err := runtimeRunDirectories(h.HomePaths.SessionsDir)
	if err != nil {
		return RuntimeArtifactManifest{}, err
	}
	cliBinary := ""
	cliWorkdir := ""
	if h.CLI != nil {
		cliBinary = strings.TrimSpace(h.CLI.binaryPath)
		cliWorkdir = strings.TrimSpace(h.CLI.workdir)
	}

	return RuntimeArtifactManifest{
		Version:       1,
		WorkspaceRoot: strings.TrimSpace(h.WorkspaceRoot),
		Home: RuntimeHomeArtifact{
			HomeDir:          strings.TrimSpace(h.HomePaths.HomeDir),
			ConfigFile:       strings.TrimSpace(h.HomePaths.ConfigFile),
			DatabaseFile:     strings.TrimSpace(h.HomePaths.DatabaseFile),
			DaemonSocket:     strings.TrimSpace(h.HomePaths.DaemonSocket),
			DaemonInfo:       strings.TrimSpace(h.HomePaths.DaemonInfo),
			LogsDir:          strings.TrimSpace(h.HomePaths.LogsDir),
			NetworkAuditFile: strings.TrimSpace(h.HomePaths.NetworkAuditFile),
		},
		Logs: RuntimeLogArtifact{
			DaemonLogFile:  strings.TrimSpace(h.HomePaths.LogFile),
			ProcessLogFile: strings.TrimSpace(h.processLogPath),
		},
		Runs: RuntimeRunArtifact{
			RootDir:     strings.TrimSpace(h.HomePaths.SessionsDir),
			Directories: runDirectories,
		},
		Transport: RuntimeTransportArtifact{
			HTTPBaseURL: strings.TrimSpace(h.HTTPBaseURL),
			HTTPHost:    strings.TrimSpace(h.Config.HTTP.Host),
			HTTPPort:    h.Config.HTTP.Port,
			UDSBaseURL:  strings.TrimSpace(h.UDSBaseURL),
			SocketPath:  strings.TrimSpace(h.Config.Daemon.Socket),
			CLIBinary:   cliBinary,
			CLIWorkdir:  cliWorkdir,
		},
		ArtifactRootDir:      strings.TrimSpace(h.Artifacts.RootDir()),
		ArtifactManifestPath: strings.TrimSpace(h.Artifacts.ManifestPath()),
		CapturedArtifacts:    h.Artifacts.Manifest(),
	}, nil
}

// WriteRuntimeManifest persists the current runtime-manifest snapshot.
func (h *RuntimeHarness) WriteRuntimeManifest() (RuntimeArtifactManifest, error) {
	if h == nil || h.Artifacts == nil {
		return RuntimeArtifactManifest{}, errors.New("runtime harness artifacts are required")
	}
	if _, err := h.Artifacts.WriteManifest(); err != nil {
		return RuntimeArtifactManifest{}, err
	}
	manifest, err := h.RuntimeManifest()
	if err != nil {
		return RuntimeArtifactManifest{}, err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return RuntimeArtifactManifest{}, fmt.Errorf("marshal runtime manifest: %w", err)
	}
	data = append(data, '\n')

	path := h.RuntimeManifestPath()
	if strings.TrimSpace(path) == "" {
		return RuntimeArtifactManifest{}, errors.New("runtime harness artifact root is required")
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return RuntimeArtifactManifest{}, fmt.Errorf("write runtime manifest %q: %w", path, err)
	}
	return manifest, nil
}

// HTTPURL returns one absolute HTTP URL under the public daemon surface.
func (h *RuntimeHarness) HTTPURL(path string) string {
	return h.HTTPBaseURL + ensureLeadingSlash(path)
}

// UDSURL returns one absolute UDS-backed URL under the operator daemon surface.
func (h *RuntimeHarness) UDSURL(path string) string {
	return h.UDSBaseURL + ensureLeadingSlash(path)
}

// HTTPJSON performs a JSON request against the daemon HTTP API.
func (h *RuntimeHarness) HTTPJSON(
	ctx context.Context,
	method string,
	path string,
	body any,
	dest any,
) error {
	return doJSONRequest(ctx, h.HTTPClient, h.HTTPURL(path), method, body, dest)
}

// UDSJSON performs a JSON request against the daemon UDS API.
func (h *RuntimeHarness) UDSJSON(
	ctx context.Context,
	method string,
	path string,
	body any,
	dest any,
) error {
	return doJSONRequest(ctx, h.UDSClient, h.UDSURL(path), method, body, dest)
}

// ResolveWorkspace resolves a workspace through the real daemon API.
func (h *RuntimeHarness) ResolveWorkspace(
	ctx context.Context,
	root string,
) (aghcontract.WorkspacePayload, error) {
	var response aghcontract.WorkspaceResponse
	err := h.UDSJSON(
		ctx,
		http.MethodPost,
		"/api/workspaces/resolve",
		aghcontract.ResolveWorkspaceRequest{Path: root},
		&response,
	)
	if err != nil {
		return aghcontract.WorkspacePayload{}, err
	}
	h.WorkspaceID = response.Workspace.ID
	return response.Workspace, nil
}

// GetWorkspace fetches one workspace through the daemon operator surface.
func (h *RuntimeHarness) GetWorkspace(
	ctx context.Context,
	workspaceID string,
) (aghcontract.WorkspacePayload, error) {
	var response aghcontract.WorkspaceResponse
	if err := h.UDSJSON(ctx, http.MethodGet, "/api/workspaces/"+workspaceID, nil, &response); err != nil {
		return aghcontract.WorkspacePayload{}, err
	}
	return response.Workspace, nil
}

func (h *RuntimeHarness) workspaceScopedAPIPath(workspaceID string, suffix string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(h.WorkspaceID)
	}
	if workspaceID == "" {
		return "", errors.New("runtime harness workspace ID is required")
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return "/api/workspaces/" + url.PathEscape(workspaceID) + suffix, nil
}

func (h *RuntimeHarness) sessionScopedAPIPath(sessionID string, suffix string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("session ID is required")
	}
	return h.workspaceScopedAPIPath("", "/sessions/"+url.PathEscape(sessionID)+suffix)
}

func (h *RuntimeHarness) networkScopedAPIPath(workspaceID string, suffix string) (string, error) {
	return h.workspaceScopedAPIPath(workspaceID, "/network"+suffix)
}

// CreateSession creates one session through the operator surface.
func (h *RuntimeHarness) CreateSession(
	ctx context.Context,
	request aghcontract.CreateSessionRequest,
) (aghcontract.SessionPayload, error) {
	var response aghcontract.SessionResponse
	if err := h.UDSJSON(ctx, http.MethodPost, "/api/sessions", request, &response); err != nil {
		return aghcontract.SessionPayload{}, err
	}
	return response.Session, nil
}

// GetSession fetches one session detail through the operator surface.
func (h *RuntimeHarness) GetSession(
	ctx context.Context,
	sessionID string,
) (aghcontract.SessionPayload, error) {
	var response aghcontract.SessionResponse
	path, err := h.sessionScopedAPIPath(sessionID, "")
	if err != nil {
		return aghcontract.SessionPayload{}, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return aghcontract.SessionPayload{}, err
	}
	return response.Session, nil
}

// StopSession stops one session through the operator surface.
func (h *RuntimeHarness) StopSession(ctx context.Context, sessionID string) error {
	path, err := h.sessionScopedAPIPath(sessionID, "/stop")
	if err != nil {
		return err
	}
	response, err := doRequest(
		ctx,
		h.UDSClient,
		h.UDSURL(path),
		http.MethodPost,
		nil,
	)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusNoContent {
		payload, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return fmt.Errorf("read stop session failure response: %w", readErr)
		}
		return fmt.Errorf("stop session status %d: %s", response.StatusCode, bytes.TrimSpace(payload))
	}
	return nil
}

// ResumeSession resumes one stopped session through the operator surface.
func (h *RuntimeHarness) ResumeSession(
	ctx context.Context,
	sessionID string,
) (aghcontract.SessionPayload, error) {
	var response aghcontract.SessionResponse
	path, err := h.sessionScopedAPIPath(sessionID, "/resume")
	if err != nil {
		return aghcontract.SessionPayload{}, err
	}
	if err := h.UDSJSON(
		ctx,
		http.MethodPost,
		path,
		nil,
		&response,
	); err != nil {
		return aghcontract.SessionPayload{}, err
	}
	return response.Session, nil
}

// PromptSession sends one prompt through the operator surface and drains the SSE stream.
func (h *RuntimeHarness) PromptSession(
	ctx context.Context,
	sessionID string,
	message string,
) ([]SSEEvent, error) {
	return h.PromptSessionWithEvents(ctx, sessionID, message, nil)
}

// PromptSessionWithEvents sends one prompt through the operator surface and lets
// callers react to streamed SSE records before the prompt completes.
func (h *RuntimeHarness) PromptSessionWithEvents(
	ctx context.Context,
	sessionID string,
	message string,
	onEvent func(SSEEvent) error,
) ([]SSEEvent, error) {
	body := map[string]string{"message": message}
	path, err := h.sessionScopedAPIPath(sessionID, "/prompt")
	if err != nil {
		return nil, err
	}
	response, err := doRequest(ctx, h.UDSClient, h.UDSURL(path), http.MethodPost, body)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		payload, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read prompt failure response: %w", readErr)
		}
		return nil, fmt.Errorf("prompt session status %d: %s", response.StatusCode, bytes.TrimSpace(payload))
	}

	return readSSERecordsWithCallback(response.Body, 0, onEvent)
}

// PromptSessionUntil sends one prompt through the operator surface and returns
// as soon as the streamed SSE records satisfy predicate.
func (h *RuntimeHarness) PromptSessionUntil(
	ctx context.Context,
	sessionID string,
	message string,
	predicate func(SSEEvent) bool,
) ([]SSEEvent, error) {
	if predicate == nil {
		return nil, errors.New("SSE predicate is required")
	}
	body := map[string]string{"message": message}
	path, err := h.sessionScopedAPIPath(sessionID, "/prompt")
	if err != nil {
		return nil, err
	}
	response, err := doRequest(ctx, h.UDSClient, h.UDSURL(path), http.MethodPost, body)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		payload, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read prompt failure response: %w", readErr)
		}
		return nil, fmt.Errorf("prompt session status %d: %s", response.StatusCode, bytes.TrimSpace(payload))
	}

	return readSSERecordsUntil(response.Body, predicate)
}

// SessionTranscript fetches the persisted transcript for one session.
func (h *RuntimeHarness) SessionTranscript(
	ctx context.Context,
	sessionID string,
) (aghcontract.SessionTranscriptResponse, error) {
	var response aghcontract.SessionTranscriptResponse
	path, err := h.sessionScopedAPIPath(sessionID, "/transcript")
	if err != nil {
		return aghcontract.SessionTranscriptResponse{}, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return aghcontract.SessionTranscriptResponse{}, err
	}
	return response, nil
}

// SessionEvents fetches persisted events for one session.
func (h *RuntimeHarness) SessionEvents(
	ctx context.Context,
	sessionID string,
) (aghcontract.SessionEventsResponse, error) {
	var response aghcontract.SessionEventsResponse
	path, err := h.sessionScopedAPIPath(sessionID, "/events")
	if err != nil {
		return aghcontract.SessionEventsResponse{}, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return aghcontract.SessionEventsResponse{}, err
	}
	return response, nil
}

// CreateNetworkChannel creates one network channel through the public operator surface.
func (h *RuntimeHarness) CreateNetworkChannel(
	ctx context.Context,
	request aghcontract.CreateNetworkChannelRequest,
) (aghcontract.NetworkChannelDetailPayload, error) {
	var response aghcontract.CreateNetworkChannelResponse
	path, err := h.networkScopedAPIPath(request.WorkspaceID, "/channels")
	if err != nil {
		return aghcontract.NetworkChannelDetailPayload{}, err
	}
	if err := h.UDSJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return aghcontract.NetworkChannelDetailPayload{}, err
	}
	return response.Channel, nil
}

// NetworkStatus fetches the current network runtime projection.
func (h *RuntimeHarness) NetworkStatus(
	ctx context.Context,
) (aghcontract.NetworkStatusPayload, error) {
	var response aghcontract.NetworkStatusResponse
	if err := h.UDSJSON(ctx, http.MethodGet, "/api/network/status", nil, &response); err != nil {
		return aghcontract.NetworkStatusPayload{}, err
	}
	return response.Network, nil
}

// NetworkPeers fetches the current visible peers, optionally filtered by channel.
func (h *RuntimeHarness) NetworkPeers(
	ctx context.Context,
	channel string,
) ([]aghcontract.NetworkPeerPayload, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(channel); trimmed != "" {
		values.Set("channel", trimmed)
	}
	path, err := h.networkScopedAPIPath("", "/peers"+encodeQuery(values))
	if err != nil {
		return nil, err
	}

	var response aghcontract.NetworkPeersResponse
	if err := h.UDSJSON(
		ctx,
		http.MethodGet,
		path,
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Peers, nil
}

// NetworkChannels fetches the current network channel projection.
func (h *RuntimeHarness) NetworkChannels(
	ctx context.Context,
) ([]aghcontract.NetworkChannelPayload, error) {
	var response aghcontract.NetworkChannelsResponse
	path, err := h.networkScopedAPIPath("", "/channels")
	if err != nil {
		return nil, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Channels, nil
}

// NetworkChannel fetches one selected network channel detail payload.
func (h *RuntimeHarness) NetworkChannel(
	ctx context.Context,
	channel string,
) (aghcontract.NetworkChannelDetailPayload, error) {
	var response aghcontract.NetworkChannelResponse
	path, err := h.networkScopedAPIPath("", "/channels/"+url.PathEscape(channel))
	if err != nil {
		return aghcontract.NetworkChannelDetailPayload{}, err
	}
	if err := h.UDSJSON(
		ctx,
		http.MethodGet,
		path,
		nil,
		&response,
	); err != nil {
		return aghcontract.NetworkChannelDetailPayload{}, err
	}
	return response.Channel, nil
}

// NetworkThreads fetches public-thread summaries for one channel.
func (h *RuntimeHarness) NetworkThreads(
	ctx context.Context,
	channel string,
) ([]aghcontract.NetworkThreadSummaryPayload, error) {
	var response aghcontract.NetworkThreadsResponse
	path, err := h.networkScopedAPIPath("", "/channels/"+url.PathEscape(channel)+"/threads")
	if err != nil {
		return nil, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Threads, nil
}

// NetworkThread fetches one public-thread summary.
func (h *RuntimeHarness) NetworkThread(
	ctx context.Context,
	channel string,
	threadID string,
) (aghcontract.NetworkThreadSummaryPayload, error) {
	var response aghcontract.NetworkThreadResponse
	path, err := h.networkScopedAPIPath("", "/channels/"+url.PathEscape(channel)+
		"/threads/"+url.PathEscape(threadID))
	if err != nil {
		return aghcontract.NetworkThreadSummaryPayload{}, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return aghcontract.NetworkThreadSummaryPayload{}, err
	}
	return response.Thread, nil
}

// NetworkThreadMessages fetches messages isolated to one public thread.
func (h *RuntimeHarness) NetworkThreadMessages(
	ctx context.Context,
	channel string,
	threadID string,
) ([]aghcontract.NetworkConversationMessagePayload, error) {
	var response aghcontract.NetworkThreadMessagesResponse
	path, err := h.networkScopedAPIPath("", "/channels/"+url.PathEscape(channel)+
		"/threads/"+url.PathEscape(threadID)+"/messages")
	if err != nil {
		return nil, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

// NetworkDirectRooms fetches direct-room summaries for one channel.
func (h *RuntimeHarness) NetworkDirectRooms(
	ctx context.Context,
	channel string,
) ([]aghcontract.NetworkDirectRoomPayload, error) {
	var response aghcontract.NetworkDirectRoomsResponse
	path, err := h.networkScopedAPIPath("", "/channels/"+url.PathEscape(channel)+"/directs")
	if err != nil {
		return nil, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Directs, nil
}

// NetworkDirectRoom fetches one direct-room summary.
func (h *RuntimeHarness) NetworkDirectRoom(
	ctx context.Context,
	channel string,
	directID string,
) (aghcontract.NetworkDirectRoomPayload, error) {
	var response aghcontract.NetworkDirectRoomResponse
	path, err := h.networkScopedAPIPath("", "/channels/"+url.PathEscape(channel)+
		"/directs/"+url.PathEscape(directID))
	if err != nil {
		return aghcontract.NetworkDirectRoomPayload{}, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return aghcontract.NetworkDirectRoomPayload{}, err
	}
	return response.Direct, nil
}

// NetworkDirectRoomMessages fetches messages isolated to one direct room.
func (h *RuntimeHarness) NetworkDirectRoomMessages(
	ctx context.Context,
	channel string,
	directID string,
) ([]aghcontract.NetworkConversationMessagePayload, error) {
	var response aghcontract.NetworkDirectRoomMessagesResponse
	path, err := h.networkScopedAPIPath("", "/channels/"+url.PathEscape(channel)+
		"/directs/"+url.PathEscape(directID)+"/messages")
	if err != nil {
		return nil, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

// NetworkDirectResolve resolves the deterministic direct room for a local session and peer.
func (h *RuntimeHarness) NetworkDirectResolve(
	ctx context.Context,
	channel string,
	request aghcontract.NetworkDirectResolveRequest,
) (aghcontract.NetworkDirectRoomPayload, error) {
	var response aghcontract.NetworkDirectRoomResponse
	path, err := h.networkScopedAPIPath("", "/channels/"+url.PathEscape(channel)+"/directs/resolve")
	if err != nil {
		return aghcontract.NetworkDirectRoomPayload{}, err
	}
	if err := h.UDSJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return aghcontract.NetworkDirectRoomPayload{}, err
	}
	return response.Direct, nil
}

// NetworkWork fetches one lifecycle-bearing work row.
func (h *RuntimeHarness) NetworkWork(
	ctx context.Context,
	workID string,
) (aghcontract.NetworkWorkPayload, error) {
	var response aghcontract.NetworkWorkResponse
	path, err := h.networkScopedAPIPath("", "/work/"+url.PathEscape(workID))
	if err != nil {
		return aghcontract.NetworkWorkPayload{}, err
	}
	if err := h.UDSJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return aghcontract.NetworkWorkPayload{}, err
	}
	return response.Work, nil
}

// NetworkChannelMessages fetches public-thread messages for one channel.
func (h *RuntimeHarness) NetworkChannelMessages(
	ctx context.Context,
	channel string,
) ([]aghcontract.NetworkConversationMessagePayload, error) {
	threads, err := h.NetworkThreads(ctx, channel)
	if err != nil {
		return nil, err
	}
	messages := make([]aghcontract.NetworkConversationMessagePayload, 0)
	for _, thread := range threads {
		threadMessages, err := h.NetworkThreadMessages(ctx, channel, thread.ThreadID)
		if err != nil {
			return nil, err
		}
		messages = append(messages, threadMessages...)
	}
	return messages, nil
}

// NetworkInbox fetches the queued inbox projection for one local session.
func (h *RuntimeHarness) NetworkInbox(
	ctx context.Context,
	sessionID string,
) ([]aghcontract.NetworkEnvelopePayload, error) {
	values := url.Values{}
	if trimmed := strings.TrimSpace(sessionID); trimmed != "" {
		values.Set("session_id", trimmed)
	}
	path, err := h.networkScopedAPIPath("", "/inbox"+encodeQuery(values))
	if err != nil {
		return nil, err
	}

	var response aghcontract.NetworkInboxResponse
	if err := h.UDSJSON(
		ctx,
		http.MethodGet,
		path,
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

// NetworkSend sends one envelope through the public network operator surface.
func (h *RuntimeHarness) NetworkSend(
	ctx context.Context,
	request aghcontract.NetworkSendRequest,
) (aghcontract.NetworkSendPayload, error) {
	var response aghcontract.NetworkSendResponse
	path, err := h.networkScopedAPIPath(request.WorkspaceID, "/send")
	if err != nil {
		return aghcontract.NetworkSendPayload{}, err
	}
	if strings.TrimSpace(request.WorkspaceID) == "" {
		request.WorkspaceID = strings.TrimSpace(h.WorkspaceID)
	}
	if err := h.UDSJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return aghcontract.NetworkSendPayload{}, err
	}
	return response.Message, nil
}

// NetworkAuditSnapshot decodes the current daemon-owned network audit file into a stable snapshot.
func (h *RuntimeHarness) NetworkAuditSnapshot() ([]store.NetworkAuditEntry, error) {
	if _, err := os.Stat(h.HomePaths.NetworkAuditFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat network audit file %q: %w", h.HomePaths.NetworkAuditFile, err)
	}
	return readNetworkAuditSnapshot(h.HomePaths.NetworkAuditFile)
}

// CaptureSessionTranscript stores the session transcript artifact.
func (h *RuntimeHarness) CaptureSessionTranscript(ctx context.Context, sessionID string) error {
	response, err := h.SessionTranscript(ctx, sessionID)
	if err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindTranscript, response.Messages)
}

// CaptureSessionEvents stores the session-events artifact.
func (h *RuntimeHarness) CaptureSessionEvents(ctx context.Context, sessionID string) error {
	response, err := h.SessionEvents(ctx, sessionID)
	if err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindEvents, response.Events)
}

// CaptureSessionSandbox stores session sandbox metadata.
func (h *RuntimeHarness) CaptureSessionSandbox(ctx context.Context, sessionID string) error {
	artifact, err := h.SessionSandboxArtifact(ctx, sessionID)
	if err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindSessionSandbox, artifact)
}

// SessionSandboxArtifact reads the public session payload and, when present,
// the persisted session metadata for one runtime session.
func (h *RuntimeHarness) SessionSandboxArtifact(
	ctx context.Context,
	sessionID string,
) (SessionSandboxArtifact, error) {
	session, err := h.GetSession(ctx, sessionID)
	if err != nil {
		return SessionSandboxArtifact{}, err
	}
	artifact := SessionSandboxArtifact{
		SessionID:    session.ID,
		SessionState: string(session.State),
		StopReason:   session.StopReason,
		StopDetail:   session.StopDetail,
		API:          session.Sandbox,
	}

	metaPath := store.SessionMetaFile(filepath.Join(h.HomePaths.SessionsDir, strings.TrimSpace(sessionID)))
	meta, err := store.ReadSessionMeta(metaPath)
	switch {
	case err == nil:
		artifact.Persisted = meta.Sandbox
	case errors.Is(err, os.ErrNotExist):
		// Keep the public-surface artifact even when no persisted meta exists yet.
	default:
		return SessionSandboxArtifact{}, fmt.Errorf("read session meta %q: %w", metaPath, err)
	}
	return artifact, nil
}

// CaptureNetworkMessages stores the current network message projection for one channel.
func (h *RuntimeHarness) CaptureNetworkMessages(ctx context.Context, channel string) error {
	messages, err := h.NetworkChannelMessages(ctx, channel)
	if err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindNetworkMessages, messages)
}

// CaptureNetworkThreads stores public-thread summaries for one channel.
func (h *RuntimeHarness) CaptureNetworkThreads(ctx context.Context, channel string) error {
	threads, err := h.NetworkThreads(ctx, channel)
	if err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindNetworkThreads, threads)
}

// CaptureNetworkDirectRooms stores direct-room summaries for one channel.
func (h *RuntimeHarness) CaptureNetworkDirectRooms(ctx context.Context, channel string) error {
	directs, err := h.NetworkDirectRooms(ctx, channel)
	if err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindNetworkDirectRooms, directs)
}

// CaptureNetworkWork stores unique work rows referenced by the channel's thread and direct messages.
func (h *RuntimeHarness) CaptureNetworkWork(ctx context.Context, channel string) error {
	workIDs, err := h.networkWorkIDs(ctx, channel)
	if err != nil {
		return err
	}
	work := make([]aghcontract.NetworkWorkPayload, 0, len(workIDs))
	for _, workID := range workIDs {
		item, err := h.NetworkWork(ctx, workID)
		if err != nil {
			return err
		}
		work = append(work, item)
	}
	return h.Artifacts.CaptureJSON(ArtifactKindNetworkWork, work)
}

func (h *RuntimeHarness) networkWorkIDs(ctx context.Context, channel string) ([]string, error) {
	seen := make(map[string]struct{})
	add := func(messages []aghcontract.NetworkConversationMessagePayload) {
		for _, message := range messages {
			workID := strings.TrimSpace(message.WorkID)
			if workID != "" {
				seen[workID] = struct{}{}
			}
		}
	}

	threadMessages, err := h.NetworkChannelMessages(ctx, channel)
	if err != nil {
		return nil, err
	}
	add(threadMessages)

	directs, err := h.NetworkDirectRooms(ctx, channel)
	if err != nil {
		return nil, err
	}
	for _, direct := range directs {
		messages, err := h.NetworkDirectRoomMessages(ctx, channel, direct.DirectID)
		if err != nil {
			return nil, err
		}
		add(messages)
	}

	workIDs := make([]string, 0, len(seen))
	for workID := range seen {
		workIDs = append(workIDs, workID)
	}
	sort.Strings(workIDs)
	return workIDs, nil
}

// CaptureNetworkAudit stores the raw network audit sink when present.
func (h *RuntimeHarness) CaptureNetworkAudit() error {
	entries, err := h.NetworkAuditSnapshot()
	if err != nil {
		return err
	}
	if entries == nil {
		return nil
	}
	return h.Artifacts.CaptureJSON(ArtifactKindNetworkAudit, entries)
}

// CaptureNetworkArtifacts stores the stable message and audit snapshots for one scenario channel.
func (h *RuntimeHarness) CaptureNetworkArtifacts(ctx context.Context, channel string) error {
	if err := h.CaptureNetworkThreads(ctx, channel); err != nil {
		return err
	}
	if err := h.CaptureNetworkDirectRooms(ctx, channel); err != nil {
		return err
	}
	if err := h.CaptureNetworkMessages(ctx, channel); err != nil {
		return err
	}
	if err := h.CaptureNetworkWork(ctx, channel); err != nil {
		return err
	}
	return h.CaptureNetworkAudit()
}

// CaptureAutomationRuns stores the current automation run projection.
func (h *RuntimeHarness) CaptureAutomationRuns(ctx context.Context, query url.Values) error {
	var response aghcontract.RunsResponse
	if err := h.UDSJSON(ctx, http.MethodGet, "/api/automation/runs"+encodeQuery(query), nil, &response); err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindAutomationRuns, response.Runs)
}

// CaptureTasks stores the current task projection.
func (h *RuntimeHarness) CaptureTasks(ctx context.Context, query url.Values) error {
	var response aghcontract.TasksResponse
	if err := h.UDSJSON(ctx, http.MethodGet, "/api/tasks"+encodeQuery(query), nil, &response); err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindTasks, response.Tasks)
}

// CaptureTaskRuns stores the task-run projection for one task.
func (h *RuntimeHarness) CaptureTaskRuns(
	ctx context.Context,
	taskID string,
	query url.Values,
) error {
	var response aghcontract.TaskRunsResponse
	if err := h.UDSJSON(
		ctx,
		http.MethodGet,
		"/api/tasks/"+taskID+"/runs"+encodeQuery(query),
		nil,
		&response,
	); err != nil {
		return err
	}
	return h.Artifacts.CaptureJSON(ArtifactKindTaskRuns, response.Runs)
}

// CaptureBridgeHealth stores one bridge health-stream snapshot.
func (h *RuntimeHarness) CaptureBridgeHealth(ctx context.Context) error {
	response, err := doRequest(ctx, h.UDSClient, h.UDSURL("/api/bridges/health/stream"), http.MethodGet, nil)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		payload, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return fmt.Errorf("read bridge health failure response: %w", readErr)
		}
		return fmt.Errorf("bridge health status %d: %s", response.StatusCode, bytes.TrimSpace(payload))
	}

	records, err := readSSERecords(response.Body, 1)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return errors.New("bridge health stream returned no snapshot")
	}

	var snapshot aghcontract.BridgeHealthStreamPayload
	if err := json.Unmarshal(records[0].Data, &snapshot); err != nil {
		return fmt.Errorf("decode bridge health snapshot: %w", err)
	}
	return h.Artifacts.CaptureJSON(ArtifactKindBridgeHealth, snapshot)
}

// CaptureProviderCallsFile stores provider call markers or logs as a raw artifact.
func (h *RuntimeHarness) CaptureProviderCallsFile(path string, mediaType string) error {
	return h.Artifacts.CaptureFile(ArtifactKindProviderCalls, path, mediaType)
}

// CaptureProviderCallsJSON stores provider call diagnostics as JSON.
func (h *RuntimeHarness) CaptureProviderCallsJSON(value any) error {
	return h.Artifacts.CaptureJSON(ArtifactKindProviderCalls, value)
}

// CaptureToolHostDiagnosticsJSON stores tool-host diagnostics separately from
// provider or mock-agent artifacts so combined-flow runs can retain both.
func (h *RuntimeHarness) CaptureToolHostDiagnosticsJSON(value ToolHostDiagnosticsArtifact) error {
	return h.Artifacts.CaptureJSON(ArtifactKindToolHostDiagnostics, value)
}

// CaptureCombinedFlowJSON stores a cross-domain scenario summary alongside the
// domain-specific artifacts captured by the test.
func (h *RuntimeHarness) CaptureCombinedFlowJSON(value CombinedFlowArtifact) error {
	return h.Artifacts.CaptureJSON(ArtifactKindCombinedFlow, value)
}

// CaptureBrowserTraceFile stores the Playwright trace archive for one scenario.
func (h *RuntimeHarness) CaptureBrowserTraceFile(path string) error {
	return h.Artifacts.CaptureFile(ArtifactKindBrowserTrace, path, "application/zip")
}

// CaptureBrowserScreenshots stores one or more screenshot files.
func (h *RuntimeHarness) CaptureBrowserScreenshots(paths []string) error {
	return h.Artifacts.CaptureFiles(ArtifactKindBrowserScreenshots, paths, "image/png")
}

// CaptureBrowserConsoleJSON stores browser console diagnostics.
func (h *RuntimeHarness) CaptureBrowserConsoleJSON(value any) error {
	return h.Artifacts.CaptureJSON(ArtifactKindBrowserConsole, value)
}

// CaptureBrowserNetworkJSON stores browser network diagnostics.
func (h *RuntimeHarness) CaptureBrowserNetworkJSON(value any) error {
	return h.Artifacts.CaptureJSON(ArtifactKindBrowserNetwork, value)
}

// CaptureTransportOutput stores one transport result inside the shared harness artifact root.
func (h *RuntimeHarness) CaptureTransportOutput(
	name string,
	artifact TransportOutputArtifact,
) (string, error) {
	if h == nil {
		return "", errors.New("runtime harness is required")
	}
	if h.Artifacts == nil {
		return "", errors.New("runtime harness artifacts are required")
	}
	artifact.Name = defaultString(artifact.Name, name)
	path, err := h.Artifacts.CaptureNamedJSON(ArtifactKindTransportOutputs, name, artifact)
	if err != nil {
		return "", err
	}
	if _, err := h.WriteRuntimeManifest(); err != nil {
		return "", err
	}
	return path, nil
}

// CaptureCLIOutput stores one CLI command result in the shared transport-output artifact directory.
func (h *RuntimeHarness) CaptureCLIOutput(
	name string,
	args []string,
	stdout string,
	stderr string,
	commandErr error,
) (string, error) {
	artifact := TransportOutputArtifact{
		Name:      strings.TrimSpace(name),
		Transport: "cli",
		Command:   append([]string(nil), args...),
		Stdout:    stdout,
		Stderr:    stderr,
	}
	if commandErr != nil {
		artifact.Error = commandErr.Error()
	}
	return h.CaptureTransportOutput(name, artifact)
}

// Run executes one CLI command against the isolated daemon runtime.
func (c *CLIClient) Run(ctx context.Context, args ...string) (string, string, error) {
	return c.RunInDir(ctx, "", args...)
}

// RunInDir executes one CLI command against the isolated daemon runtime using the provided working directory.
func (c *CLIClient) RunInDir(ctx context.Context, workdir string, args ...string) (string, string, error) {
	// #nosec G204 -- test helper intentionally shells out to the current agh test binary.
	cmd := execabs.CommandContext(ctx, c.binaryPath, args...)
	cmd.Env = append([]string(nil), c.env...)
	trimmedDir := strings.TrimSpace(workdir)
	switch {
	case trimmedDir == "":
		cmd.Dir = c.workdir
	case filepath.IsAbs(trimmedDir):
		cmd.Dir = trimmedDir
	default:
		cmd.Dir = filepath.Join(c.workdir, trimmedDir)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// RunJSON executes one CLI command and decodes its JSON stdout.
func (c *CLIClient) RunJSON(ctx context.Context, dest any, args ...string) error {
	return c.RunJSONInDir(ctx, "", dest, args...)
}

// RunJSONInDir executes one CLI command in the provided working directory and decodes its JSON stdout.
func (c *CLIClient) RunJSONInDir(ctx context.Context, workdir string, dest any, args ...string) error {
	stdout, stderr, err := c.RunInDir(ctx, workdir, args...)
	if err != nil {
		return fmt.Errorf("run CLI %q: %w; stderr=%s", strings.Join(args, " "), err, strings.TrimSpace(stderr))
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal([]byte(stdout), dest); err != nil {
		return fmt.Errorf("decode CLI JSON %q: %w; stdout=%s", strings.Join(args, " "), err, strings.TrimSpace(stdout))
	}
	return nil
}

func (h *RuntimeHarness) waitForReady(ctx context.Context, pollInterval time.Duration) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if exited, err := h.pollExit(); exited {
			return daemonExitedBeforeReadinessError(err)
		}
		select {
		case <-ctx.Done():
			if exited, err := h.pollExit(); exited {
				return daemonExitedBeforeReadinessError(err)
			}
			return errors.New("daemon did not become ready before timeout")
		case err, ok := <-h.waitCh:
			h.processWaitMu.Lock()
			h.processExited = true
			if ok {
				h.processErr = err
			}
			storedErr := h.processErr
			h.processWaitMu.Unlock()
			return daemonExitedBeforeReadinessError(storedErr)
		case <-ticker.C:
			if err := h.probeReady(ctx); err == nil {
				return nil
			}
		}
	}
}

func daemonExitedBeforeReadinessError(err error) error {
	if err != nil {
		return fmt.Errorf("%w: %w", errDaemonExitedBeforeReadiness, err)
	}
	return errDaemonExitedBeforeReadiness
}

func (h *RuntimeHarness) probeReady(ctx context.Context) error {
	if runtimeHarnessRequiresHTTPReadinessProbe(h.Config.HTTP.Host) {
		var httpStatus aghcontract.DaemonStatusResponse
		if err := h.HTTPJSON(ctx, http.MethodGet, "/api/daemon/status", nil, &httpStatus); err != nil {
			return err
		}
	}

	var udsStatus aghcontract.DaemonStatusResponse
	if err := h.UDSJSON(ctx, http.MethodGet, "/api/daemon/status", nil, &udsStatus); err != nil {
		return err
	}

	var cliStatus aghcontract.DaemonStatusPayload
	if err := h.CLI.RunJSON(ctx, &cliStatus, "daemon", "status", "-o", "json"); err != nil {
		return err
	}

	return nil
}

func runtimeHarnessRequiresHTTPReadinessProbe(host string) bool {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return true
	}
	if strings.EqualFold(trimmed, "localhost") {
		return true
	}
	ip := net.ParseIP(trimmed)
	return ip != nil && ip.IsLoopback()
}

func (h *RuntimeHarness) waitForExit(ctx context.Context) error {
	if exited, err := h.pollExit(); exited {
		return err
	}

	select {
	case err, ok := <-h.waitCh:
		h.processWaitMu.Lock()
		defer h.processWaitMu.Unlock()
		h.processExited = true
		if ok {
			h.processErr = err
		}
		return h.processErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *RuntimeHarness) pollExit() (bool, error) {
	h.processWaitMu.Lock()
	defer h.processWaitMu.Unlock()

	if h.processExited {
		return true, h.processErr
	}

	select {
	case err, ok := <-h.waitCh:
		h.processExited = true
		if ok {
			h.processErr = err
		}
		return true, h.processErr
	default:
		return false, nil
	}
}

func doJSONRequest(
	ctx context.Context,
	client *http.Client,
	targetURL string,
	method string,
	body any,
	dest any,
) error {
	response, err := doRequest(ctx, client, targetURL, method, body)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read %s %s response: %w", method, targetURL, err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf(
			"%s %s status %d: %s",
			method,
			targetURL,
			response.StatusCode,
			strings.TrimSpace(string(payload)),
		)
	}
	if dest == nil || len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, dest); err != nil {
		return fmt.Errorf(
			"decode %s %s response: %w; body=%s",
			method,
			targetURL,
			err,
			strings.TrimSpace(string(payload)),
		)
	}
	return nil
}

func doRequest(
	ctx context.Context,
	client *http.Client,
	targetURL string,
	method string,
	body any,
) (*http.Response, error) {
	if client == nil {
		return nil, errors.New("runtime harness http client is required")
	}
	reader, err := requestBody(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, reader)
	if err != nil {
		return nil, fmt.Errorf("new request %s %s: %w", method, targetURL, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform %s %s: %w", method, targetURL, err)
	}
	return response, nil
}

func requestBody(body any) (io.Reader, error) {
	switch typed := body.(type) {
	case nil:
		return nil, nil
	case []byte:
		return bytes.NewReader(typed), nil
	case string:
		return strings.NewReader(typed), nil
	default:
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		return bytes.NewReader(payload), nil
	}
}

func readSSERecords(reader io.Reader, limit int) ([]SSEEvent, error) {
	return readSSERecordsWithCallback(reader, limit, nil)
}

func readSSERecordsWithCallback(
	reader io.Reader,
	limit int,
	onRecord func(SSEEvent) error,
) ([]SSEEvent, error) {
	return readSSERecordsMatching(reader, limit, onRecord, nil)
}

func readSSERecordsUntil(
	reader io.Reader,
	predicate func(SSEEvent) bool,
) ([]SSEEvent, error) {
	if predicate == nil {
		return nil, errors.New("SSE predicate is required")
	}
	return readSSERecordsMatching(reader, 0, nil, predicate)
}

func readSSERecordsMatching(
	reader io.Reader,
	limit int,
	onRecord func(SSEEvent) error,
	predicate func(SSEEvent) bool,
) ([]SSEEvent, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 16*1024), 1024*1024)

	records := make([]SSEEvent, 0, maxInt(limit, 1))
	current := SSEEvent{}
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			applySSELine(&current, line)
			continue
		}
		var matched bool
		var err error
		records, matched, err = appendSSERecord(records, current, onRecord, predicate)
		if err != nil {
			return nil, err
		}
		current = SSEEvent{}
		if matched || (limit > 0 && len(records) >= limit) {
			return records, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan SSE stream: %w", err)
	}
	var err error
	records, _, err = appendSSERecord(records, current, onRecord, predicate)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func applySSELine(record *SSEEvent, line string) {
	switch {
	case strings.HasPrefix(line, "id: "):
		record.ID = strings.TrimPrefix(line, "id: ")
	case strings.HasPrefix(line, "event: "):
		record.Event = strings.TrimPrefix(line, "event: ")
	case strings.HasPrefix(line, "data: "):
		if len(record.Data) > 0 {
			record.Data = append(record.Data, '\n')
		}
		record.Data = append(record.Data, strings.TrimPrefix(line, "data: ")...)
	}
}

func appendSSERecord(
	records []SSEEvent,
	record SSEEvent,
	onRecord func(SSEEvent) error,
	predicate func(SSEEvent) bool,
) ([]SSEEvent, bool, error) {
	if record.ID == "" && record.Event == "" && len(record.Data) == 0 {
		return records, false, nil
	}
	normalizeSSEEvent(&record)
	records = append(records, record)
	if onRecord != nil {
		if err := onRecord(record); err != nil {
			return nil, false, fmt.Errorf("handle SSE record: %w", err)
		}
	}
	return records, predicate != nil && predicate(record), nil
}

func normalizeSSEEvent(record *SSEEvent) {
	if record == nil || strings.TrimSpace(record.Event) != "" {
		return
	}
	record.Event = inferSSEEventName(record.Data)
}

func inferSSEEventName(data []byte) string {
	trimmed := bytes.TrimSpace(data)
	switch {
	case len(trimmed) == 0:
		return ""
	case bytes.Equal(trimmed, []byte("[DONE]")):
		return "done"
	}

	var envelope struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return ""
	}

	switch strings.TrimSpace(envelope.Type) {
	case "data-agh-permission":
		return "permission"
	case "data-agh-event":
		var payload struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return "event"
		}
		if eventType := strings.TrimSpace(payload.Type); eventType != "" {
			return eventType
		}
		return "event"
	case "text-start", "text-delta", "text-end":
		return transportParityEventAgentMessage
	case "reasoning-start", "reasoning-delta", "reasoning-end":
		return "reasoning"
	case "tool-input-start", "tool-input-available":
		return "tool_call"
	case "tool-output-available":
		return "tool_result"
	default:
		return strings.TrimSpace(envelope.Type)
	}
}

func runtimeEnv(homePaths aghconfig.HomePaths, extra map[string]string) []string {
	base := append([]string(nil), os.Environ()...)
	base = setEnvValue(base, "AGH_HOME", homePaths.HomeDir)
	base = setEnvValue(base, "HOME", homePaths.HomeDir)

	keys := make([]string, 0, len(extra))
	for key := range extra {
		keys = append(keys, key)
	}
	sortStrings(keys)
	for _, key := range keys {
		base = append(base, key+"="+extra[key])
	}
	return base
}

func withRuntimeCLIEnv(
	homePaths aghconfig.HomePaths,
	env []string,
	binaryPath string,
) ([]string, error) {
	trimmedBinaryPath := strings.TrimSpace(binaryPath)
	if trimmedBinaryPath == "" {
		return env, nil
	}

	shimPath, err := installRuntimeCLI(homePaths, trimmedBinaryPath)
	if err != nil {
		return nil, err
	}
	withPath := setEnvValue(env, "PATH", prependPath(filepath.Dir(shimPath), lookupEnvValue(env, "PATH")))
	withPath = setEnvValue(withPath, "AGH_E2E_CLI_BIN", shimPath)
	withPath = setEnvValue(withPath, daemonBinaryEnvVar, trimmedBinaryPath)
	return withPath, nil
}

func installRuntimeCLI(homePaths aghconfig.HomePaths, binaryPath string) (string, error) {
	binDir := filepath.Join(homePaths.HomeDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir runtime cli dir %q: %w", binDir, err)
	}

	targetName := "agh"
	if runtime.GOOS == windowsGOOS {
		targetName = "agh.exe"
	}
	targetPath := filepath.Join(binDir, targetName)
	if err := os.Remove(targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("remove existing runtime cli target %q: %w", targetPath, err)
	}
	if runtime.GOOS != windowsGOOS {
		if err := os.Symlink(binaryPath, targetPath); err == nil {
			return targetPath, nil
		}
	}
	if err := os.Link(binaryPath, targetPath); err == nil {
		return targetPath, nil
	}
	if err := copyFile(binaryPath, targetPath); err != nil {
		return "", fmt.Errorf("copy runtime cli binary to %q: %w", targetPath, err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0o755); err != nil {
			return "", fmt.Errorf("chmod runtime cli target %q: %w", targetPath, err)
		}
	}
	return targetPath, nil
}

func setEnvValue(env []string, key string, value string) []string {
	targetKey := strings.TrimSpace(key)
	if targetKey == "" {
		return env
	}
	entry := targetKey + "=" + value
	for idx, current := range env {
		existingKey, _, ok := strings.Cut(current, "=")
		if ok && existingKey == targetKey {
			updated := append([]string(nil), env...)
			updated[idx] = entry
			return updated
		}
	}
	return append(append([]string(nil), env...), entry)
}

func lookupEnvValue(env []string, key string) string {
	targetKey := strings.TrimSpace(key)
	for _, current := range env {
		existingKey, existingValue, ok := strings.Cut(current, "=")
		if ok && existingKey == targetKey {
			return existingValue
		}
	}
	return ""
}

func prependPath(prefix string, current string) string {
	trimmedPrefix := strings.TrimSpace(prefix)
	trimmedCurrent := strings.TrimSpace(current)
	switch {
	case trimmedPrefix == "":
		return trimmedCurrent
	case trimmedCurrent == "":
		return trimmedPrefix
	default:
		return trimmedPrefix + string(os.PathListSeparator) + trimmedCurrent
	}
}

func closeIdleConnections(client *http.Client) {
	if client == nil || client.Transport == nil {
		return
	}
	if closer, ok := client.Transport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func runtimeRunDirectories(root string) ([]string, error) {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(trimmedRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read runtime run root %q: %w", trimmedRoot, err)
	}

	directories := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		directories = append(directories, filepath.Join(trimmedRoot, entry.Name()))
	}
	sortStrings(directories)
	return directories, nil
}

func readNetworkAuditSnapshot(path string) ([]store.NetworkAuditEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open network audit file %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	entries := make([]store.NetworkAuditEntry, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var entry store.NetworkAuditEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("decode network audit line: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan network audit file %q: %w", path, err)
	}
	return entries, nil
}

func buildAGHBinary(t testing.TB) string {
	t.Helper()

	repoRoot := mustRepoRoot(t)
	if override := strings.TrimSpace(os.Getenv(daemonBinaryEnvVar)); override != "" {
		if filepath.IsAbs(override) {
			return override
		}
		return filepath.Clean(filepath.Join(repoRoot, override))
	}

	buildBinaryMu.Lock()
	defer buildBinaryMu.Unlock()

	if builtBinaryPath != "" {
		if _, err := os.Stat(builtBinaryPath); err == nil {
			return builtBinaryPath
		}
	}

	binaryPath := filepath.Join(os.TempDir(), fmt.Sprintf("agh-e2e-%d", os.Getpid()))
	// #nosec G204 -- test harness builds the local agh binary from the checked-out repository.
	cmd := execabs.CommandContext(context.Background(), "go", "build", "-o", binaryPath, "./cmd/agh")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./cmd/agh error = %v\n%s", err, strings.TrimSpace(string(output)))
	}

	builtBinaryPath = binaryPath
	return builtBinaryPath
}

func cloneMockAgentRegistrations(
	in map[string]acpmock.Registration,
) map[string]acpmock.Registration {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]acpmock.Registration, len(in))
	maps.Copy(out, in)
	return out
}

func mustRepoRoot(t testing.TB) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}

	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("failed to locate repository root from runtime_harness.go")
		}
		dir = parent
	}
}

func ensureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func encodeQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}

func defaultDuration(value time.Duration, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
