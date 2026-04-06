//go:build integration && !windows

package session

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/kballard/go-shellquote"
	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	testSessionStopHelperEnvKey   = "AGH_TEST_SESSION_STOP_HELPER"
	testSessionStopWrapperEnvKey  = "AGH_TEST_SESSION_STOP_WRAPPER"
	testSessionStopWrapperPIDFile = "AGH_TEST_SESSION_STOP_WRAPPER_PID_FILE"
)

func TestSessionStopACPHelperProcess(t *testing.T) {
	if os.Getenv(testSessionStopHelperEnvKey) != "1" {
		return
	}

	conn := acpsdk.NewAgentSideConnection(sessionStopACPAgent{}, os.Stdout, os.Stdin)
	<-conn.Done()
	os.Exit(0)
}

func TestSessionStopACPWrapperProcess(t *testing.T) {
	if os.Getenv(testSessionStopWrapperEnvKey) != "1" {
		return
	}

	bin, err := os.Executable()
	if err != nil {
		os.Exit(1)
	}

	cmd := exec.Command(bin, "-test.run=TestSessionStopACPHelperProcess")
	cmd.Env = append([]string(nil), os.Environ()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		os.Exit(1)
	}

	if pidFile := strings.TrimSpace(os.Getenv(testSessionStopWrapperPIDFile)); pidFile != "" {
		if writeErr := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); writeErr != nil {
			_ = cmd.Process.Kill()
			os.Exit(1)
		}
	}

	if err := cmd.Wait(); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

func TestManagerIntegrationStopFinalizesWrappedACPProcess(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "wrapped-helper.pid")

	h := newHarness(t)
	driver := acp.New(
		acp.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		acp.WithStopTimeout(100*time.Millisecond),
	)
	h.manager = newManagerWithHarness(t, h,
		WithDriver(NewACPDriverAdapter(driver)),
		WithAgentLoader(staticAgentLoader(aghconfig.AgentDef{
			Provider: "claude",
			Command:  sessionStopWrapperCommand(t, pidFile),
			Prompt:   "You are a coding assistant.",
		})),
	)

	session := createSession(t, h)
	childPID := waitForSessionStopWrapperChildPID(t, pidFile)

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.manager.Stop(stopCtx, session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	waitForSessionStopProcessExit(t, childPID, time.Second)

	meta := readMeta(t, session.MetaPath())
	if meta.State != string(StateStopped) {
		t.Fatalf("meta state = %q, want %q", meta.State, StateStopped)
	}
}

func sessionStopWrapperCommand(t *testing.T, pidFile string) string {
	t.Helper()

	bin, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	return shellquote.Join(
		"env",
		testSessionStopHelperEnvKey+"=1",
		testSessionStopWrapperEnvKey+"=1",
		testSessionStopWrapperPIDFile+"="+pidFile,
		bin,
		"-test.run=TestSessionStopACPWrapperProcess",
	)
}

func waitForSessionStopWrapperChildPID(t *testing.T, path string) int {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			text := strings.TrimSpace(string(data))
			if text == "" {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			pid, convErr := strconv.Atoi(text)
			if convErr != nil {
				t.Fatalf("strconv.Atoi(%q) error = %v", string(data), convErr)
			}
			if pid <= 0 {
				t.Fatalf("wrapper child pid = %d, want > 0", pid)
			}
			return pid
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("os.ReadFile(%q) error = %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for wrapper child pid file %q", path)
	return 0
}

func waitForSessionStopProcessExit(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !sessionStopProcessAlive(pid) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("process %d is still alive after %v", pid, timeout)
}

func sessionStopProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

type sessionStopACPAgent struct{}

func (sessionStopACPAgent) Authenticate(context.Context, acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (sessionStopACPAgent) Initialize(context.Context, acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: true,
		},
		AuthMethods: []acpsdk.AuthMethod{},
	}, nil
}

func (sessionStopACPAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (sessionStopACPAgent) NewSession(context.Context, acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	return acpsdk.NewSessionResponse{
		SessionId: "sess-stop-helper",
	}, nil
}

func (sessionStopACPAgent) LoadSession(context.Context, acpsdk.LoadSessionRequest) (acpsdk.LoadSessionResponse, error) {
	return acpsdk.LoadSessionResponse{}, nil
}

func (sessionStopACPAgent) Prompt(context.Context, acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	return acpsdk.PromptResponse{
		StopReason: acpsdk.StopReasonEndTurn,
	}, nil
}

func (sessionStopACPAgent) SetSessionMode(context.Context, acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}
