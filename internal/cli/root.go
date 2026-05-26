// Package cli provides the AGH Cobra command tree.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/compozy/agh/internal/agentidentity"
	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	aghdaemon "github.com/compozy/agh/internal/daemon"
	diagnosticspkg "github.com/compozy/agh/internal/diagnostics"
	"github.com/compozy/agh/internal/procutil"
	aghupdate "github.com/compozy/agh/internal/update"
	"github.com/compozy/agh/internal/version"
	"github.com/spf13/cobra"
)

const (
	rootAghKey     = "agh"
	rootVersionKey = "version"
)

const (
	outputFlagName = "output"
	jsonFlagName   = "json"
	yesFlagName    = "yes"

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
	loadConfig                  func() (aghconfig.Config, error)
	loadSkillRegistrySources    skillRegistrySourceLoader
	resolveHome                 func() (aghconfig.HomePaths, error)
	resolveHomeForWorkspace     func(workspaceRoot string) (aghconfig.HomePaths, error)
	ensureHome                  func(aghconfig.HomePaths) error
	runInstallWizard            installWizardRunner
	newClient                   func(socketPath string) (DaemonClient, error)
	newDaemon                   func() (daemonRunner, error)
	runRelaunchHelper           func(context.Context, aghdaemon.RelaunchHelperConfig) error
	readDaemonInfo              func(path string) (aghdaemon.Info, error)
	signalProcess               func(pid int, sig syscall.Signal) error
	processAlive                func(pid int) bool
	processMatchesStartTime     func(pid int, startedAt time.Time) bool
	executable                  func() (string, error)
	getwd                       func() (string, error)
	getenv                      func(string) string
	lookPath                    func(string) (string, error)
	now                         func() time.Time
	pollInterval                time.Duration
	startTimeout                time.Duration
	stopTimeout                 time.Duration
	spawnDetached               func(context.Context, aghconfig.HomePaths) (daemonProcess, error)
	newUpdateManager            func(aghconfig.HomePaths) (updateManager, error)
	newMCPAuthClient            newMCPAuthClientFunc
	runProviderAuthCommand      providerAuthCommandRunner
	runProviderAuthLoginCommand providerAuthCommandRunner
}

// NewRootCommand constructs the AGH v1 CLI command tree.
func NewRootCommand() *cobra.Command {
	return newRootCommand(commandDeps{})
}

func newRootCommand(deps commandDeps) *cobra.Command {
	deps = deps.withDefaults()

	cmd := &cobra.Command{
		Use:   rootAghKey,
		Short: "AGH — Artificial General Hivemind",
		Example: `  # Start the daemon and create a session in the current workspace
  agh daemon start
  agh session new --agent general

  # Print machine-readable output for automation
  agh session list -o json`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().
		StringP(outputFlagName, "o", string(OutputHuman), "Output format: human, json, jsonl, or toon")
	cmd.PersistentFlags().Bool(jsonFlagName, false, "Emit JSON output")

	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newInstallCommand(deps))
	cmd.AddCommand(newConfigCommand(deps))
	cmd.AddCommand(newSupportCommand(deps))
	cmd.AddCommand(newUpdateCommand(deps))
	cmd.AddCommand(newUninstallCommand(deps))
	cmd.AddCommand(newStatusCommand(deps))
	cmd.AddCommand(newDoctorCommand(deps))
	cmd.AddCommand(newOnboardingCommand(deps))
	cmd.AddCommand(newDaemonCommand(deps))
	cmd.AddCommand(newNetworkCommand(deps))
	cmd.AddCommand(newMeCommand(deps))
	cmd.AddCommand(newSpawnCommand(deps))
	cmd.AddCommand(newChannelCommand(deps))
	cmd.AddCommand(newSessionCommand(deps))
	cmd.AddCommand(newProviderCommand(deps))
	cmd.AddCommand(newBridgeCommand(deps))
	cmd.AddCommand(newNotificationsCommand(deps))
	cmd.AddCommand(newBundleCommand(deps))
	cmd.AddCommand(newWorkspaceCommand(deps))
	cmd.AddCommand(newAgentCommand(deps))
	cmd.AddCommand(newExtensionCommand(deps))
	cmd.AddCommand(newHooksCommand(deps))
	cmd.AddCommand(newAutomationCommand(deps))
	cmd.AddCommand(newSchedulerCommand(deps))
	cmd.AddCommand(newTaskCommand(deps))
	cmd.AddCommand(newSkillCommand(deps))
	cmd.AddCommand(newResourceCommand(deps))
	cmd.AddCommand(newMemoryCommand(deps))
	cmd.AddCommand(newVaultCommand(deps))
	cmd.AddCommand(newToolCommand(deps))
	cmd.AddCommand(newToolsetsCommand(deps))
	cmd.AddCommand(newMCPCommand(deps))
	cmd.AddCommand(newLogsCommand(deps))
	cmd.AddCommand(newWhoamiCommand(deps))
	cmd.AddCommand(newDocCommand())

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   rootVersionKey,
		Short: "Print the AGH version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeCommandOutput(cmd, outputBundle{
				jsonValue: version.Current(),
				human: func() (string, error) {
					return fmt.Sprintf("agh %s", version.Current().Version), nil
				},
				toon: func() (string, error) {
					info := version.Current()
					return renderToonObject(rootVersionKey, []string{rootVersionKey, "commit", "build_date"}, []string{
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
		return writeExecutionError(stderr, args, err)
	}
	return 0
}

func writeExecutionError(stderr io.Writer, args []string, err error) int {
	exitCode := cliExitCodeForError(err)
	if payload, ok := marshalStructuredExecutionError(args, err); ok {
		if _, writeErr := stderr.Write(payload); writeErr == nil {
			if len(payload) == 0 || payload[len(payload)-1] != '\n' {
				_, _ = fmt.Fprintln(stderr)
			}
			return exitCode
		}
	}

	_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
	return exitCode
}

func marshalStructuredExecutionError(args []string, err error) ([]byte, bool) {
	if !isStructuredAgentCommandError(err) {
		return marshalDiagnosticExecutionError(args, err)
	}

	switch requestedOutputFormat(args) {
	case OutputJSON:
		payload, marshalErr := agentidentity.MarshalErrorJSON(err)
		if marshalErr != nil {
			return nil, false
		}
		return payload, true
	case OutputJSONL:
		payload, marshalErr := agentidentity.MarshalErrorJSONL(err)
		if marshalErr != nil {
			return nil, false
		}
		return payload, true
	default:
		return nil, false
	}
}

func marshalDiagnosticExecutionError(args []string, err error) ([]byte, bool) {
	item, ok := diagnosticspkg.ItemFromError(err)
	if !ok {
		return nil, false
	}
	payload := contract.ErrorPayload{Error: diagnosticspkg.Redact(err.Error()), Diagnostic: &item}
	switch requestedOutputFormat(args) {
	case OutputJSON:
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return nil, false
		}
		return encoded, true
	case OutputJSONL:
		encoded, marshalErr := json.Marshal(struct {
			Type  string                `json:"type"`
			Error contract.ErrorPayload `json:"error"`
		}{
			Type:  "error",
			Error: payload,
		})
		if marshalErr != nil {
			return nil, false
		}
		return append(encoded, '\n'), true
	default:
		return nil, false
	}
}

func isStructuredAgentCommandError(err error) bool {
	var identityErr *agentidentity.Error
	return errors.As(err, &identityErr)
}

func requestedOutputFormat(args []string) OutputFormat {
	mode := OutputHuman
	for i := 0; i < len(args); i++ {
		switch arg := strings.TrimSpace(args[i]); {
		case arg == "--json":
			mode = OutputJSON
		case arg == "-o" || arg == "--output":
			if i+1 < len(args) {
				mode = OutputFormat(strings.ToLower(strings.TrimSpace(args[i+1])))
				i++
			}
		case strings.HasPrefix(arg, "--output="):
			mode = OutputFormat(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--output="))))
		case strings.HasPrefix(arg, "-o="):
			mode = OutputFormat(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "-o="))))
		}
	}
	return mode
}

func (d commandDeps) withDefaults() commandDeps {
	d = d.withRegistryDefaults()
	d = d.withRuntimeDefaults()
	d = d.withTimingDefaults()
	return d
}

func (d commandDeps) withRegistryDefaults() commandDeps {
	if d.loadConfig == nil {
		d.loadConfig = func() (aghconfig.Config, error) {
			return aghconfig.Load()
		}
	}
	if d.loadSkillRegistrySources == nil {
		d.loadSkillRegistrySources = defaultSkillRegistrySourceLoader
	}
	if d.resolveHome == nil {
		d.resolveHome = aghconfig.ResolveHomePaths
	}
	if d.resolveHomeForWorkspace == nil {
		d.resolveHomeForWorkspace = aghconfig.ResolveHomePathsForWorkspace
	}
	if d.ensureHome == nil {
		d.ensureHome = aghconfig.EnsureHomeLayout
	}
	return d
}

func (d commandDeps) withRuntimeDefaults() commandDeps {
	if d.runInstallWizard == nil {
		d.runInstallWizard = runInstallWizard
	}
	if d.newClient == nil {
		d.newClient = NewClient
	}
	d = d.withMCPAuthDefaults()
	d = d.withProviderAuthDefaults()
	if d.newDaemon == nil {
		d.newDaemon = func() (daemonRunner, error) {
			return aghdaemon.New()
		}
	}
	if d.runRelaunchHelper == nil {
		d.runRelaunchHelper = aghdaemon.RunRelaunchHelper
	}
	if d.readDaemonInfo == nil {
		d.readDaemonInfo = aghdaemon.ReadInfo
	}
	if d.signalProcess == nil {
		d.signalProcess = procutil.Signal
	}
	if d.processAlive == nil {
		d.processAlive = procutil.Alive
	}
	if d.processMatchesStartTime == nil {
		d.processMatchesStartTime = procutil.MatchesStartTime
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
	if d.lookPath == nil {
		d.lookPath = exec.LookPath
	}
	if d.spawnDetached == nil {
		d.spawnDetached = func(ctx context.Context, homePaths aghconfig.HomePaths) (daemonProcess, error) {
			return spawnDetachedDaemonProcess(ctx, homePaths, d.executable)
		}
	}
	if d.newUpdateManager == nil {
		d.newUpdateManager = func(homePaths aghconfig.HomePaths) (updateManager, error) {
			return aghupdate.NewManager(aghupdate.Config{
				HomePaths:      homePaths,
				CurrentVersion: version.Current().Version,
				ExecutablePath: d.executable,
				Getenv:         d.getenv,
			})
		}
	}
	return d
}

func (d commandDeps) withTimingDefaults() commandDeps {
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
	return d
}

func loadRuntimeContext(deps commandDeps) (*runtimeContext, error) {
	homePaths, err := deps.resolveHome()
	if err != nil {
		return nil, err
	}
	cfg, err := deps.loadConfig()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Daemon.Socket) == "" {
		cfg.Daemon.Socket = homePaths.DaemonSocket
	}
	return &runtimeContext{
		HomePaths: homePaths,
		Config:    cfg,
	}, nil
}

func clientFromDeps(deps commandDeps) (DaemonClient, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return nil, err
	}

	socketPath := strings.TrimSpace(runtime.Config.Daemon.Socket)
	if socketPath == "" {
		socketPath = runtime.HomePaths.DaemonSocket
	}
	if socketPath == "" {
		return nil, errors.New("cli: daemon socket path is required")
	}

	client, err := deps.newClient(socketPath)
	if err != nil {
		return nil, err
	}
	return client, nil
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
