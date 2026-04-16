package acp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/subprocess"
)

// Launcher starts an ACP-capable agent process inside an environment.
type Launcher = environment.Launcher

// Handle represents a running agent process.
type Handle = environment.Handle

// LaunchSpec describes the ACP-capable command to start inside an environment.
type LaunchSpec = environment.LaunchSpec

var (
	_ environment.Launcher = (*localLauncher)(nil)
	_ environment.Handle   = (*localProcessHandle)(nil)
)

type localLauncher struct {
	logger      *slog.Logger
	stopTimeout time.Duration
}

type localProcessHandle struct {
	process *subprocess.Process
	cwd     string
}

// NewLocalLauncher returns the local daemon-host subprocess launcher.
func NewLocalLauncher(logger *slog.Logger, stopTimeout time.Duration) environment.Launcher {
	return newLocalLauncher(logger, stopTimeout)
}

func newLocalLauncher(logger *slog.Logger, stopTimeout time.Duration) *localLauncher {
	if logger == nil {
		logger = slog.Default()
	}
	if stopTimeout <= 0 {
		stopTimeout = defaultStopTimeout
	}
	return &localLauncher{
		logger:      logger,
		stopTimeout: stopTimeout,
	}
}

func (l *localLauncher) Launch(
	_ context.Context,
	spec environment.LaunchSpec,
) (environment.Handle, error) {
	command, args, err := parseCommandString(spec.Command)
	if err != nil {
		return nil, err
	}

	managed, err := subprocess.Launch(context.Background(), subprocess.LaunchConfig{
		Command:          command,
		Args:             args,
		Dir:              spec.Cwd,
		Env:              daemonMatchedEnv(spec.Env),
		Logger:           l.logger,
		DisableTransport: true,
		ShutdownTimeout:  l.stopTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("acp: start subprocess %q: %w", spec.Command, err)
	}

	return &localProcessHandle{
		process: managed,
		cwd:     spec.Cwd,
	}, nil
}

func (h *localProcessHandle) PID() int {
	if h == nil || h.process == nil {
		return 0
	}
	return h.process.PID()
}

func (h *localProcessHandle) Cwd() string {
	if h == nil {
		return ""
	}
	return h.cwd
}

func (h *localProcessHandle) Stdin() io.WriteCloser {
	if h == nil || h.process == nil {
		return nil
	}
	return h.process.Stdin()
}

func (h *localProcessHandle) Stdout() io.ReadCloser {
	if h == nil || h.process == nil {
		return nil
	}
	return h.process.Stdout()
}

func (h *localProcessHandle) Stderr() string {
	if h == nil || h.process == nil {
		return ""
	}
	return h.process.Stderr()
}

func (h *localProcessHandle) Done() <-chan struct{} {
	if h == nil || h.process == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return h.process.Done()
}

func (h *localProcessHandle) Wait() error {
	if h == nil || h.process == nil {
		return nil
	}
	return h.process.Wait()
}

func (h *localProcessHandle) Stop(ctx context.Context) error {
	if h == nil || h.process == nil {
		return nil
	}
	return h.process.Shutdown(ctx)
}
