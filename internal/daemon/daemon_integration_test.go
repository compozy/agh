//go:build integration

package daemon

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestBootSequenceReady(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.boot(testContext(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testContext(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.sessions == nil || d.observer == nil || d.registry == nil {
		t.Fatalf("boot() did not wire runtime dependencies: sessions=%v observer=%v registry=%v", d.sessions, d.observer, d.registry)
	}
	if _, err := os.Stat(homePaths.DatabaseFile); err != nil {
		t.Fatalf("stat global database error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); err != nil {
		t.Fatalf("stat daemon.json error = %v", err)
	}
	if _, err := AcquireLock(homePaths.DaemonLock, os.Getpid()); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("AcquireLock(second instance) error = %v, want ErrAlreadyRunning", err)
	}
}

func TestRunGracefulShutdownViaContextCancellation(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(runCtx)
	}()

	<-d.readyCh
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("daemon.json after shutdown: stat error = %v, want os.ErrNotExist", err)
	}

	lock, err := AcquireLock(homePaths.DaemonLock, os.Getpid())
	if err != nil {
		t.Fatalf("AcquireLock(after shutdown) error = %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("lock.Release() error = %v", err)
	}
}

func TestRunGracefulShutdownViaSignal(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	signalCh := make(chan os.Signal, 1)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
		WithSignalChannel(signalCh),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(context.Background())
	}()

	<-d.readyCh
	signalCh <- syscall.SIGINT

	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("daemon.json after signal shutdown: stat error = %v, want os.ErrNotExist", err)
	}
}

func integrationHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("AGH_HOME", homeDir)

	homePaths, err := aghconfig.ResolveHomePathsFrom(homeDir)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	homePaths.DaemonSocket = shortSocketPath(t)
	return homePaths
}
