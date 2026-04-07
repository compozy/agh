package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	"github.com/pedronauck/agh/internal/testutil"
)

type stubDaemonProcess struct {
	waitCh chan error
}

func (p *stubDaemonProcess) PID() int {
	return 42
}

func (p *stubDaemonProcess) Wait() error {
	return <-p.waitCh
}

func TestWaitForDaemonStartReturnsStatusWhenDaemonBecomesReady(t *testing.T) {
	t.Parallel()

	child := &stubDaemonProcess{waitCh: make(chan error, 1)}
	deps := newTestDeps(t, stubClient{
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{Status: "ready", PID: 42}, nil
		},
	})
	deps.pollInterval = time.Millisecond
	deps.startTimeout = 100 * time.Millisecond

	status, err := waitForDaemonStart(testutil.Context(t), deps, child)
	child.waitCh <- nil
	if err != nil {
		t.Fatalf("waitForDaemonStart() error = %v", err)
	}
	if status.Status != "ready" || status.PID != 42 {
		t.Fatalf("waitForDaemonStart() status = %#v, want ready pid 42", status)
	}
}

func TestWaitForDaemonStopReturnsStoppedStatusWhenProcessExits(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{}, errors.New("daemon unavailable")
		},
	})
	deps.pollInterval = time.Millisecond
	deps.stopTimeout = 100 * time.Millisecond
	deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
		return aghdaemon.Info{
			PID:       42,
			StartedAt: fixedTestNow,
		}, nil
	}

	aliveChecks := 0
	deps.processAlive = func(int) bool {
		aliveChecks++
		return aliveChecks < 2
	}

	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		t.Fatalf("loadRuntimeContext() error = %v", err)
	}
	info := aghdaemon.Info{
		PID:       42,
		StartedAt: fixedTestNow,
	}

	status, err := waitForDaemonStop(testutil.Context(t), deps, runtime, info)
	if err != nil {
		t.Fatalf("waitForDaemonStop() error = %v", err)
	}
	if status.Status != "stopped" || status.PID != 42 {
		t.Fatalf("waitForDaemonStop() status = %#v, want stopped pid 42", status)
	}
}

func TestDaemonStopCommandSignalsAndWaitsForShutdown(t *testing.T) {
	t.Parallel()

	var (
		signalPID  int
		signalSent bool
	)

	deps := newTestDeps(t, stubClient{
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{}, errors.New("daemon unavailable")
		},
	})
	deps.pollInterval = time.Millisecond
	deps.stopTimeout = 100 * time.Millisecond
	deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
		return aghdaemon.Info{
			PID:       42,
			StartedAt: fixedTestNow,
		}, nil
	}
	aliveChecks := 0
	deps.processAlive = func(int) bool {
		aliveChecks++
		return aliveChecks < 2
	}
	deps.signalProcess = func(pid int, _ syscall.Signal) error {
		signalPID = pid
		signalSent = true
		return nil
	}

	stdout, _, err := executeRootCommand(t, deps, "daemon", "stop", "-o", "json")
	if err != nil {
		t.Fatalf("executeRootCommand() error = %v", err)
	}
	if !signalSent || signalPID != 42 {
		t.Fatalf("signalProcess() = (%v, %d), want true pid 42", signalSent, signalPID)
	}

	var decoded DaemonStatus
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Status != "stopped" || decoded.PID != 42 {
		t.Fatalf("decoded = %#v, want stopped pid 42", decoded)
	}
}

func TestDaemonStatusCommandReturnsDaemonStatus(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{
				Status:    "ready",
				PID:       42,
				StartedAt: fixedTestNow,
			}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "daemon", "status", "-o", "json")
	if err != nil {
		t.Fatalf("executeRootCommand() error = %v", err)
	}

	var decoded DaemonStatus
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Status != "ready" || decoded.PID != 42 {
		t.Fatalf("decoded = %#v, want ready pid 42", decoded)
	}
}

func TestRunDaemonForegroundRunsDaemonWhenNotAlreadyRunning(t *testing.T) {
	t.Parallel()

	runner := &stubRunner{}
	deps := newTestDeps(t, stubClient{})
	deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
		return aghdaemon.Info{}, os.ErrNotExist
	}
	deps.newDaemon = func() (daemonRunner, error) {
		return runner, nil
	}

	if err := runDaemonForeground(testutil.Context(t), deps); err != nil {
		t.Fatalf("runDaemonForeground() error = %v", err)
	}
	if !runner.ran {
		t.Fatal("daemon runner did not execute")
	}
}

func TestRunDaemonDetachedReturnsReadyStatus(t *testing.T) {
	t.Parallel()

	child := &stubDaemonProcess{waitCh: make(chan error, 1)}
	deps := newTestDeps(t, stubClient{
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{Status: "ready", PID: 42}, nil
		},
	})
	deps.pollInterval = time.Millisecond
	deps.startTimeout = 100 * time.Millisecond
	deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
		return aghdaemon.Info{}, os.ErrNotExist
	}
	deps.spawnDetached = func(aghconfig.HomePaths) (daemonProcess, error) {
		return child, nil
	}

	status, err := runDaemonDetached(testutil.Context(t), deps)
	child.waitCh <- nil
	if err != nil {
		t.Fatalf("runDaemonDetached() error = %v", err)
	}
	if status.Status != "ready" || status.PID != 42 {
		t.Fatalf("runDaemonDetached() status = %#v, want ready pid 42", status)
	}
}
