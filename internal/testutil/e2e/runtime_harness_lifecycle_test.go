package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
)

func TestRuntimeHarnessWaitForReadyUsesPublicSurfaces(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := writeJSONResponse(w, aghcontract.DaemonStatusResponse{
			Daemon: aghcontract.DaemonStatusPayload{
				Status:   "running",
				Socket:   "/tmp/agh.sock",
				HTTPHost: "127.0.0.1",
				HTTPPort: 2123,
			},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	harness := &RuntimeHarness{
		HTTPBaseURL: server.URL,
		HTTPClient:  server.Client(),
		UDSBaseURL:  server.URL,
		UDSClient:   server.Client(),
		CLI: &CLIClient{
			binaryPath: writeCLIScript(t, `#!/bin/sh
printf '%s\n' '{"status":"running","socket":"/tmp/agh.sock","http_host":"127.0.0.1","http_port":2123}'
`),
			workdir: t.TempDir(),
		},
		waitCh: make(chan error),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := harness.waitForReady(ctx, time.Millisecond); err != nil {
		t.Fatalf("waitForReady() error = %v", err)
	}
}

func TestRuntimeHarnessWaitForReadyReturnsExitError(t *testing.T) {
	t.Parallel()

	waitCh := make(chan error, 1)
	waitCh <- errors.New("daemon exited")

	harness := &RuntimeHarness{waitCh: waitCh}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := harness.waitForReady(ctx, time.Millisecond); err == nil {
		t.Fatal("waitForReady() error = nil, want non-nil")
	}
}

func TestRuntimeHarnessStopFallsBackToInterruptWhenCLIStopFails(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(
		context.Background(),
		"sh",
		"-c",
		"trap 'exit 0' INT; while :; do sleep 1; done",
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start() error = %v", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
		close(waitCh)
	}()

	harness := &RuntimeHarness{
		process: cmd,
		waitCh:  waitCh,
		CLI: &CLIClient{
			binaryPath: writeCLIScript(t, "#!/bin/sh\nexit 1\n"),
			workdir:    t.TempDir(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := harness.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := harness.Stop(ctx); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
}

func TestRuntimeHelpersCoverRequestAndTimingUtilities(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/bad":
			http.Error(w, "bad request", http.StatusBadRequest)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var payload map[string]string
	if err := doJSONRequest(
		context.Background(),
		server.Client(),
		server.URL+"/ok",
		http.MethodPost,
		map[string]string{"hello": "world"},
		&payload,
	); err != nil {
		t.Fatalf("doJSONRequest(/ok) error = %v", err)
	}
	if got, want := payload["status"], "ok"; got != want {
		t.Fatalf("payload[status] = %q, want %q", got, want)
	}

	if err := doJSONRequest(
		context.Background(),
		server.Client(),
		server.URL+"/bad",
		http.MethodGet,
		nil,
		nil,
	); err == nil {
		t.Fatal("doJSONRequest(/bad) error = nil, want non-nil")
	}
	if _, err := requestBody(make(chan int)); err == nil {
		t.Fatal("requestBody(chan) error = nil, want non-nil")
	}
	if got, want := ensureLeadingSlash("api/demo"), "/api/demo"; got != want {
		t.Fatalf("ensureLeadingSlash() = %q, want %q", got, want)
	}
	if got, want := ensureLeadingSlash("/api/demo"), "/api/demo"; got != want {
		t.Fatalf("ensureLeadingSlash(existing slash) = %q, want %q", got, want)
	}
	if got, want := defaultDuration(0, time.Second), time.Second; got != want {
		t.Fatalf("defaultDuration() = %s, want %s", got, want)
	}
	if got, want := defaultDuration(2*time.Second, time.Second), 2*time.Second; got != want {
		t.Fatalf("defaultDuration(non-zero) = %s, want %s", got, want)
	}
	if got, want := maxInt(1, 2), 2; got != want {
		t.Fatalf("max() = %d, want %d", got, want)
	}
	if got, want := maxInt(3, 2), 3; got != want {
		t.Fatalf("max(non-fallback) = %d, want %d", got, want)
	}
	if got, want := encodeQuery(nil), ""; got != want {
		t.Fatalf("encodeQuery(nil) = %q, want %q", got, want)
	}
}

func writeCLIScript(t testing.TB, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "agh-cli-script.sh")
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	return path
}

func TestHelperUtilitiesCoverSortingAndSanitizing(t *testing.T) {
	t.Parallel()

	values := []string{"zeta", "alpha", "mid"}
	sortStrings(values)
	if got, want := values, []string{"alpha", "mid", "zeta"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sortStrings() = %#v, want %#v", got, want)
	}

	if got, want := sanitizePathComponent("  Hello, World!  "), "hello-world"; got != want {
		t.Fatalf("sanitizePathComponent() = %q, want %q", got, want)
	}
	if got, want := sanitizePathComponent(""), "run"; got != want {
		t.Fatalf("sanitizePathComponent(blank) = %q, want %q", got, want)
	}
}

func TestCaptureNetworkAuditMissingFileIsNoop(t *testing.T) {
	t.Parallel()

	harness := &RuntimeHarness{
		HomePaths: aghconfig.HomePaths{
			NetworkAuditFile: filepath.Join(t.TempDir(), "missing.audit"),
		},
		Artifacts: NewArtifactCollector(t),
	}
	if err := harness.CaptureNetworkAudit(); err != nil {
		t.Fatalf("CaptureNetworkAudit() error = %v", err)
	}
	if got := len(harness.Artifacts.Manifest().Artifacts); got != 0 {
		t.Fatalf("len(artifacts) = %d, want 0", got)
	}
}

func TestRuntimeHelpersCoverCLIEnvAndRepoUtilities(t *testing.T) {
	t.Parallel()

	homePaths := NewHomePaths(t)
	binaryPath := writeCLIScript(t, "#!/bin/sh\nexit 0\n")
	baseEnv := []string{"PATH=/usr/bin", "HOME=/tmp/demo"}

	withCLI, err := withRuntimeCLIEnv(homePaths, baseEnv, binaryPath)
	if err != nil {
		t.Fatalf("withRuntimeCLIEnv() error = %v", err)
	}

	shimPath := lookupEnvValue(withCLI, "AGH_E2E_CLI_BIN")
	if strings.TrimSpace(shimPath) == "" {
		t.Fatal("AGH_E2E_CLI_BIN = empty, want installed runtime shim")
	}
	if _, err := os.Stat(shimPath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", shimPath, err)
	}
	if got, want := filepath.Dir(shimPath), filepath.Join(homePaths.HomeDir, "bin"); got != want {
		t.Fatalf("filepath.Dir(shimPath) = %q, want %q", got, want)
	}
	if got := lookupEnvValue(withCLI, "PATH"); !strings.HasPrefix(got, filepath.Dir(shimPath)) {
		t.Fatalf("PATH = %q, want shim directory prefix", got)
	}
	if got, want := lookupEnvValue(withCLI, "HOME"), "/tmp/demo"; got != want {
		t.Fatalf("lookupEnvValue(HOME) = %q, want %q", got, want)
	}

	unchanged, err := withRuntimeCLIEnv(homePaths, baseEnv, "")
	if err != nil {
		t.Fatalf("withRuntimeCLIEnv(blank) error = %v", err)
	}
	if !reflect.DeepEqual(unchanged, baseEnv) {
		t.Fatalf("withRuntimeCLIEnv(blank) = %#v, want %#v", unchanged, baseEnv)
	}

	if got, want := setEnvValue(
		baseEnv,
		"HOME",
		"/tmp/updated",
	), []string{
		"PATH=/usr/bin",
		"HOME=/tmp/updated",
	}; !reflect.DeepEqual(
		got,
		want,
	) {
		t.Fatalf("setEnvValue(update) = %#v, want %#v", got, want)
	}
	if got, want := setEnvValue(
		baseEnv,
		"NEW_VAR",
		"present",
	), []string{
		"PATH=/usr/bin",
		"HOME=/tmp/demo",
		"NEW_VAR=present",
	}; !reflect.DeepEqual(
		got,
		want,
	) {
		t.Fatalf("setEnvValue(append) = %#v, want %#v", got, want)
	}
	if got, want := setEnvValue(baseEnv, "", "ignored"), baseEnv; !reflect.DeepEqual(got, want) {
		t.Fatalf("setEnvValue(blank key) = %#v, want %#v", got, want)
	}
	if got := lookupEnvValue(baseEnv, "MISSING"); got != "" {
		t.Fatalf("lookupEnvValue(MISSING) = %q, want empty", got)
	}
	if got, want := prependPath("", "/usr/bin"), "/usr/bin"; got != want {
		t.Fatalf("prependPath(blank prefix) = %q, want %q", got, want)
	}
	if got, want := prependPath("/tmp/bin", ""), "/tmp/bin"; got != want {
		t.Fatalf("prependPath(blank current) = %q, want %q", got, want)
	}

	root := mustRepoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("os.Stat(go.mod under %q) error = %v", root, err)
	}

	if got := cloneMockAgentRegistrations(nil); got != nil {
		t.Fatalf("cloneMockAgentRegistrations(nil) = %#v, want nil", got)
	}
	original := map[string]acpmock.Registration{
		"mock-coder": {AgentName: "mock-coder"},
	}
	cloned := cloneMockAgentRegistrations(original)
	original["mock-coder"] = acpmock.Registration{AgentName: "changed"}
	if got, want := cloned["mock-coder"].AgentName, "mock-coder"; got != want {
		t.Fatalf("cloned[mock-coder].AgentName = %q, want %q", got, want)
	}
}

func TestBuildAGHBinaryProducesReusableExecutable(t *testing.T) {
	path := buildAGHBinary(t)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", path, err)
	}

	if got, want := buildAGHBinary(t), path; got != want {
		t.Fatalf("second buildAGHBinary() = %q, want %q", got, want)
	}
}

func TestBuildAGHBinaryHonorsEnvironmentOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "agh-custom")
	if err := os.WriteFile(override, []byte("fake"), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", override, err)
	}

	t.Setenv(daemonBinaryEnvVar, override)
	if got, want := buildAGHBinary(t), override; got != want {
		t.Fatalf("buildAGHBinary() with env override = %q, want %q", got, want)
	}
}
