// Package cli provides the AGH Cobra command tree.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	"github.com/pedronauck/agh/internal/version"
	"github.com/spf13/cobra"
)

const (
	outputFlagName = "output"

	defaultPollInterval = 100 * time.Millisecond
	defaultStartTimeout = 15 * time.Second
	defaultStopTimeout  = 15 * time.Second
)

type daemonRunner interface {
	Run(ctx context.Context) error
}

type runtimeContext struct {
	HomePaths aghconfig.HomePaths
	Config    aghconfig.Config
}

type installWizardRunner func(context.Context, installWizardInput) (installWizardSelection, error)

type commandDeps struct {
	loadConfig       func() (aghconfig.Config, error)
	resolveHome      func() (aghconfig.HomePaths, error)
	ensureHome       func(aghconfig.HomePaths) error
	runInstallWizard installWizardRunner
	newClient        func(socketPath string) (DaemonClient, error)
	newDaemon        func() (daemonRunner, error)
	readDaemonInfo   func(path string) (aghdaemon.Info, error)
	signalProcess    func(pid int, sig syscall.Signal) error
	processAlive     func(pid int) bool
	executable       func() (string, error)
	getwd            func() (string, error)
	getenv           func(string) string
	now              func() time.Time
	pollInterval     time.Duration
	startTimeout     time.Duration
	stopTimeout      time.Duration
	spawnDetached    func(aghconfig.HomePaths) (daemonProcess, error)
}

// NewRootCommand constructs the AGH v1 CLI command tree.
func NewRootCommand() *cobra.Command {
	return newRootCommand(commandDeps{})
}

func newRootCommand(deps commandDeps) *cobra.Command {
	deps = deps.withDefaults()

	cmd := &cobra.Command{
		Use:           "agh",
		Short:         "AGH agent operating system",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringP(outputFlagName, "o", string(OutputHuman), "Output format: human, json, or toon")

	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newInstallCommand(deps))
	cmd.AddCommand(newDaemonCommand(deps))
	cmd.AddCommand(newSessionCommand(deps))
	cmd.AddCommand(newAgentCommand(deps))
	cmd.AddCommand(newSkillCommand(deps))
	cmd.AddCommand(newMemoryCommand(deps))
	cmd.AddCommand(newObserveCommand(deps))
	cmd.AddCommand(newWhoamiCommand(deps))

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the AGH version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeCommandOutput(cmd, outputBundle{
				jsonValue: version.Current(),
				human: func() (string, error) {
					return fmt.Sprintf("agh %s", version.Version), nil
				},
				toon: func() (string, error) {
					info := version.Current()
					return renderToonObject("version", []string{"version", "commit", "build_date"}, []string{
						info.Version,
						info.Commit,
						info.BuildDate,
					}), nil
				},
			})
		},
	}
}

func ExecuteContext(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	cmd := NewRootCommand()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func (d commandDeps) withDefaults() commandDeps {
	if d.loadConfig == nil {
		d.loadConfig = func() (aghconfig.Config, error) {
			return aghconfig.Load()
		}
	}
	if d.resolveHome == nil {
		d.resolveHome = aghconfig.ResolveHomePaths
	}
	if d.ensureHome == nil {
		d.ensureHome = aghconfig.EnsureHomeLayout
	}
	if d.runInstallWizard == nil {
		d.runInstallWizard = runInstallWizard
	}
	if d.newClient == nil {
		d.newClient = NewClient
	}
	if d.newDaemon == nil {
		d.newDaemon = func() (daemonRunner, error) {
			return aghdaemon.New()
		}
	}
	if d.readDaemonInfo == nil {
		d.readDaemonInfo = aghdaemon.ReadInfo
	}
	if d.signalProcess == nil {
		d.signalProcess = signalProcess
	}
	if d.processAlive == nil {
		d.processAlive = processAlive
	}
	if d.executable == nil {
		d.executable = os.Executable
	}
	if d.getwd == nil {
		d.getwd = os.Getwd
	}
	if d.getenv == nil {
		d.getenv = os.Getenv
	}
	if d.now == nil {
		d.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if d.pollInterval <= 0 {
		d.pollInterval = defaultPollInterval
	}
	if d.startTimeout <= 0 {
		d.startTimeout = defaultStartTimeout
	}
	if d.stopTimeout <= 0 {
		d.stopTimeout = defaultStopTimeout
	}
	if d.spawnDetached == nil {
		d.spawnDetached = func(homePaths aghconfig.HomePaths) (daemonProcess, error) {
			return spawnDetachedDaemonProcess(homePaths, d.executable)
		}
	}
	return d
}

func loadRuntimeContext(deps commandDeps) (runtimeContext, error) {
	homePaths, err := deps.resolveHome()
	if err != nil {
		return runtimeContext{}, err
	}
	cfg, err := deps.loadConfig()
	if err != nil {
		return runtimeContext{}, err
	}
	if strings.TrimSpace(cfg.Daemon.Socket) == "" {
		cfg.Daemon.Socket = homePaths.DaemonSocket
	}
	return runtimeContext{
		HomePaths: homePaths,
		Config:    cfg,
	}, nil
}

func clientFromDeps(deps commandDeps) (DaemonClient, runtimeContext, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return nil, runtimeContext{}, err
	}

	socketPath := strings.TrimSpace(runtime.Config.Daemon.Socket)
	if socketPath == "" {
		socketPath = runtime.HomePaths.DaemonSocket
	}
	if socketPath == "" {
		return nil, runtimeContext{}, errors.New("cli: daemon socket path is required")
	}

	client, err := deps.newClient(socketPath)
	if err != nil {
		return nil, runtimeContext{}, err
	}
	return client, runtime, nil
}

func currentWorkingDirectory(deps commandDeps) (string, error) {
	if deps.getwd == nil {
		return "", errors.New("cli: getwd dependency is required")
	}

	wd, err := deps.getwd()
	if err != nil {
		return "", fmt.Errorf("cli: resolve current working directory: %w", err)
	}
	wd = strings.TrimSpace(wd)
	if wd == "" {
		return "", errors.New("cli: current working directory is empty")
	}
	return wd, nil
}

func signalProcess(pid int, sig syscall.Signal) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("cli: find process %d: %w", pid, err)
	}
	if err := process.Signal(sig); err != nil {
		return fmt.Errorf("cli: signal process %d: %w", pid, err)
	}
	return nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return errors.Is(err, syscall.EPERM)
}
