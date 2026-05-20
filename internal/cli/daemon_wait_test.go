package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	"github.com/pedronauck/agh/internal/testutil"
)

type stubDaemonProcess struct {
	pid     int
	done    chan struct{}
	waitErr error
}

func (p *stubDaemonProcess) PID() int {
	if p.pid > 0 {
		return p.pid
	}
	return 42
}

func (p *stubDaemonProcess) Done() <-chan struct{} {
	return p.done
}

func (p *stubDaemonProcess) Wait() error {
	<-p.done
	return p.waitErr
}

func (p *stubDaemonProcess) complete(err error) {
	p.waitErr = err
	close(p.done)
}

func TestWaitForDaemonStartReturnsStatusWhenDaemonBecomesReady(t *testing.T) {
	t.Parallel()

	t.Run("Should return daemon status when daemon becomes ready", func(t *testing.T) {
		t.Parallel()

		child := &stubDaemonProcess{done: make(chan struct{})}
		deps := newTestDeps(t, &stubClient{
			daemonStatusFn: func(context.Context) (DaemonStatus, error) {
				return DaemonStatus{Status: "ready", PID: 42}, nil
			},
		})
		deps.pollInterval = time.Millisecond
		deps.startTimeout = 100 * time.Millisecond

		status, err := waitForDaemonStart(testutil.Context(t), deps, child)
		child.complete(nil)
		if err != nil {
			t.Fatalf("waitForDaemonStart() error = %v", err)
		}
		if status.Status != "ready" || status.PID != 42 {
			t.Fatalf("waitForDaemonStart() status = %#v, want ready pid 42", status)
		}
	})
}

func TestWaitForDaemonStartReturnsDeadlineExceededWhenReadyTimeoutExpires(t *testing.T) {
	t.Parallel()

	t.Run("Should wrap deadline exceeded when daemon readiness times out", func(t *testing.T) {
		t.Parallel()

		child := &stubDaemonProcess{done: make(chan struct{})}
		deps := newTestDeps(t, &stubClient{
			daemonStatusFn: func(context.Context) (DaemonStatus, error) {
				return DaemonStatus{}, errors.New("daemon unavailable")
			},
		})
		deps.pollInterval = time.Millisecond
		deps.startTimeout = 5 * time.Millisecond
		deps.processAlive = func(int) bool { return true }

		_, err := waitForDaemonStart(testutil.Context(t), deps, child)
		child.complete(nil)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("waitForDaemonStart() error = %v, want context.DeadlineExceeded", err)
		}
		if !strings.Contains(err.Error(), "daemon did not become ready before timeout") {
			t.Fatalf("waitForDaemonStart() error = %v, want readiness timeout context", err)
		}
	})
}

func TestWaitForDaemonStopReturnsStoppedStatusWhenProcessExits(t *testing.T) {
	t.Parallel()

	t.Run("Should return stopped status when process exits", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
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
	})
}

func TestWaitForDaemonStopClearsStaleNetworkSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("Should clear stale network snapshot when daemon stops", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
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
				Network: &aghdaemon.NetworkInfo{
					Enabled:      true,
					Status:       "running",
					ListenerHost: "127.0.0.1",
					ListenerPort: 4522,
				},
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
			Network: &aghdaemon.NetworkInfo{
				Enabled:      true,
				Status:       "running",
				ListenerHost: "127.0.0.1",
				ListenerPort: 4522,
			},
		}

		status, err := waitForDaemonStop(testutil.Context(t), deps, runtime, info)
		if err != nil {
			t.Fatalf("waitForDaemonStop() error = %v", err)
		}
		if status.Status != "stopped" || status.PID != 42 {
			t.Fatalf("waitForDaemonStop() status = %#v, want stopped pid 42", status)
		}
		if status.Network != nil {
			t.Fatalf("waitForDaemonStop() network = %#v, want nil after stop", status.Network)
		}
	})
}

func TestDaemonStopCommandSignalsAndWaitsForShutdown(t *testing.T) {
	t.Parallel()

	t.Run("Should signal daemon and wait for stopped status", func(t *testing.T) {
		t.Parallel()

		var (
			signalPID  int
			signalSent bool
		)

		deps := newTestDeps(t, &stubClient{
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
	})
}

func TestDaemonStopCommandRejectsReusedPIDFromDaemonInfo(t *testing.T) {
	t.Parallel()

	t.Run("Should refuse to signal a reused PID when daemon info start time no longer matches", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{})
		deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
			return aghdaemon.Info{
				PID:       42,
				StartedAt: fixedTestNow,
			}, nil
		}
		deps.processAlive = func(int) bool { return true }
		deps.processMatchesStartTime = func(int, time.Time) bool { return false }

		signalCalled := false
		deps.signalProcess = func(int, syscall.Signal) error {
			signalCalled = true
			return nil
		}

		_, _, err := executeRootCommand(t, deps, "daemon", "stop")
		if err == nil || !strings.Contains(err.Error(), "daemon is not running") {
			t.Fatalf("daemon stop error = %v, want daemon is not running", err)
		}
		if signalCalled {
			t.Fatal("signalProcess() called for reused PID, want no signal")
		}
	})
}

func TestStatusCommandReturnsDaemonStatus(t *testing.T) {
	t.Parallel()

	t.Run("Should return daemon status payload", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
			statusFn: func(context.Context) (StatusRecord, error) {
				return StatusRecord{
					Daemon: DaemonStatus{
						Status:    "ready",
						PID:       42,
						StartedAt: fixedTestNow,
					},
				}, nil
			},
		})

		stdout, _, err := executeRootCommand(t, deps, "status", "-o", "json")
		if err != nil {
			t.Fatalf("executeRootCommand() error = %v", err)
		}

		var decoded StatusRecord
		if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if decoded.Daemon.Status != "ready" || decoded.Daemon.PID != 42 {
			t.Fatalf("decoded = %#v, want ready pid 42", decoded)
		}
	})
}

func TestRunDaemonForegroundRunsDaemonWhenNotAlreadyRunning(t *testing.T) {
	t.Parallel()

	t.Run("Should run daemon when no daemon is already running", func(t *testing.T) {
		t.Parallel()

		runner := &stubRunner{}
		deps := newTestDeps(t, &stubClient{})
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
	})
}

func TestRunDaemonDetachedReturnsReadyStatus(t *testing.T) {
	t.Parallel()

	t.Run("Should return ready status when detached daemon becomes ready", func(t *testing.T) {
		t.Parallel()

		child := &stubDaemonProcess{done: make(chan struct{})}
		deps := newTestDeps(t, &stubClient{
			daemonStatusFn: func(context.Context) (DaemonStatus, error) {
				return DaemonStatus{Status: "ready", PID: 42}, nil
			},
		})
		deps.pollInterval = time.Millisecond
		deps.startTimeout = 100 * time.Millisecond
		deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
			return aghdaemon.Info{}, os.ErrNotExist
		}
		deps.spawnDetached = func(context.Context, aghconfig.HomePaths) (daemonProcess, error) {
			return child, nil
		}

		status, err := runDaemonDetached(testutil.Context(t), deps)
		child.complete(nil)
		if err != nil {
			t.Fatalf("runDaemonDetached() error = %v", err)
		}
		if status.Status != "ready" || status.PID != 42 {
			t.Fatalf("runDaemonDetached() status = %#v, want ready pid 42", status)
		}
	})
}

func TestRunDaemonDetachedIgnoresReusedPIDFromDaemonInfo(t *testing.T) {
	t.Parallel()

	t.Run("Should start a detached daemon when daemon info points to a reused PID", func(t *testing.T) {
		t.Parallel()

		child := &stubDaemonProcess{pid: 84, done: make(chan struct{})}
		deps := newTestDeps(t, &stubClient{
			daemonStatusFn: func(context.Context) (DaemonStatus, error) {
				return DaemonStatus{Status: "ready", PID: 84}, nil
			},
		})
		deps.pollInterval = time.Millisecond
		deps.startTimeout = 100 * time.Millisecond
		deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
			return aghdaemon.Info{
				PID:       42,
				StartedAt: fixedTestNow,
			}, nil
		}
		deps.processAlive = func(int) bool { return true }
		deps.processMatchesStartTime = func(int, time.Time) bool { return false }

		spawned := false
		deps.spawnDetached = func(context.Context, aghconfig.HomePaths) (daemonProcess, error) {
			spawned = true
			return child, nil
		}

		status, err := runDaemonDetached(testutil.Context(t), deps)
		child.complete(nil)
		if err != nil {
			t.Fatalf("runDaemonDetached() error = %v", err)
		}
		if !spawned {
			t.Fatal("spawnDetached() not called, want detached launch for stale daemon info")
		}
		if status.Status != "ready" || status.PID != 84 {
			t.Fatalf("runDaemonDetached() status = %#v, want ready pid 84", status)
		}
	})
}
