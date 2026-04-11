package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	"github.com/pedronauck/agh/internal/version"
	"github.com/spf13/cobra"
)

const internalChildFlagName = "internal-child"

type daemonProcess interface {
	PID() int
	Wait() error
}

type execDaemonProcess struct {
	cmd       *exec.Cmd
	logPath   string
	logOffset int64
}

func (p *execDaemonProcess) PID() int {
	return p.cmd.Process.Pid
}

func (p *execDaemonProcess) Wait() error {
	err := p.cmd.Wait()
	if err == nil {
		return nil
	}
	return attachCommandLog(err, p.logPath, p.logOffset)
}

func newDaemonCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the AGH daemon",
	}

	cmd.AddCommand(newDaemonStartCommand(deps))
	cmd.AddCommand(newDaemonStopCommand(deps))
	cmd.AddCommand(newDaemonStatusCommand(deps))
	return cmd
}

func newDaemonStartCommand(deps commandDeps) *cobra.Command {
	var (
		foreground    bool
		internalChild bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the AGH daemon",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if foreground || internalChild {
				return runDaemonForeground(cmd.Context(), deps)
			}
			status, err := runDaemonDetached(cmd.Context(), deps)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, daemonStatusBundle(status, deps.now))
		},
	}
	cmd.Flags().BoolVar(&foreground, "foreground", false, "Run the daemon in the foreground")
	cmd.Flags().BoolVar(&internalChild, internalChildFlagName, false, "Internal detached child mode")
	_ = cmd.Flags().MarkHidden(internalChildFlagName)
	return cmd
}

func newDaemonStatusCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			runtime, err := loadRuntimeContext(deps)
			if err != nil {
				return err
			}

			status, err := daemonStatusFromDeps(cmd.Context(), deps, runtime)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, daemonStatusBundle(status, deps.now))
		},
	}
}

func newDaemonStopCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the AGH daemon",
		RunE: func(cmd *cobra.Command, _ []string) error {
			runtime, err := loadRuntimeContext(deps)
			if err != nil {
				return err
			}

			info, running, err := daemonInfo(runtime.HomePaths, deps)
			if err != nil {
				return err
			}
			if !running {
				return errors.New("cli: daemon is not running")
			}

			if err := deps.signalProcess(info.PID, syscall.SIGTERM); err != nil {
				return err
			}

			status, err := waitForDaemonStop(cmd.Context(), deps, runtime, info)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, daemonStatusBundle(status, deps.now))
		},
	}
}

func runDaemonForeground(ctx context.Context, deps commandDeps) error {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return err
	}
	if err := deps.ensureHome(runtime.HomePaths); err != nil {
		return err
	}

	if _, running, err := daemonInfo(runtime.HomePaths, deps); err != nil {
		return err
	} else if running {
		return errors.New("cli: daemon already running")
	}

	runner, err := deps.newDaemon()
	if err != nil {
		return err
	}
	return runner.Run(ctx)
}

func runDaemonDetached(ctx context.Context, deps commandDeps) (DaemonStatus, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return DaemonStatus{}, err
	}
	if err := deps.ensureHome(runtime.HomePaths); err != nil {
		return DaemonStatus{}, err
	}

	if info, running, err := daemonInfo(runtime.HomePaths, deps); err != nil {
		return DaemonStatus{}, err
	} else if running {
		return DaemonStatus{}, fmt.Errorf("cli: daemon already running (pid=%d)", info.PID)
	}

	child, err := deps.spawnDetached(runtime.HomePaths)
	if err != nil {
		return DaemonStatus{}, err
	}
	if child == nil {
		return DaemonStatus{}, errors.New("cli: detached daemon process is required")
	}

	status, err := waitForDaemonStart(ctx, deps, child)
	if err != nil {
		return DaemonStatus{}, err
	}
	return status, nil
}

func waitForDaemonStart(ctx context.Context, deps commandDeps, child daemonProcess) (DaemonStatus, error) {
	waitCtx := ctx
	if waitCtx == nil {
		waitCtx = context.Background()
	}
	if _, hasDeadline := waitCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(waitCtx, deps.startTimeout)
		defer cancel()
	}

	client, _, err := clientFromDeps(deps)
	if err != nil {
		return DaemonStatus{}, err
	}

	childErrCh := make(chan error, 1)
	go func() {
		childErrCh <- child.Wait()
	}()

	ticker := time.NewTicker(deps.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return DaemonStatus{}, errors.New("cli: daemon did not become ready before timeout")
		case err := <-childErrCh:
			if err != nil {
				return DaemonStatus{}, fmt.Errorf("cli: detached daemon exited before readiness: %w", err)
			}
			return DaemonStatus{}, errors.New("cli: detached daemon exited before readiness")
		case <-ticker.C:
			status, statusErr := client.DaemonStatus(waitCtx)
			if statusErr == nil {
				return status, nil
			}
		}
	}
}

func waitForDaemonStop(ctx context.Context, deps commandDeps, runtime runtimeContext, info aghdaemon.Info) (DaemonStatus, error) {
	waitCtx := ctx
	if waitCtx == nil {
		waitCtx = context.Background()
	}
	if _, hasDeadline := waitCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(waitCtx, deps.stopTimeout)
		defer cancel()
	}

	client, _, clientErr := clientFromDeps(deps)
	ticker := time.NewTicker(deps.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return DaemonStatus{}, errors.New("cli: daemon did not stop before timeout")
		case <-ticker.C:
			if _, running, err := daemonInfo(runtime.HomePaths, deps); err == nil && !running {
				return daemonStatusWithState(runtime, info, "stopped"), nil
			}
			if clientErr == nil {
				if _, err := client.DaemonStatus(waitCtx); err != nil {
					if _, running, infoErr := daemonInfo(runtime.HomePaths, deps); infoErr == nil && !running {
						return daemonStatusWithState(runtime, info, "stopped"), nil
					}
				}
			}
		}
	}
}

func daemonStatusFromDeps(ctx context.Context, deps commandDeps, runtime runtimeContext) (DaemonStatus, error) {
	client, _, err := clientFromDeps(deps)
	if err == nil {
		status, statusErr := client.DaemonStatus(ctx)
		if statusErr == nil {
			return status, nil
		}
	}

	info, running, err := daemonInfo(runtime.HomePaths, deps)
	if err != nil {
		return DaemonStatus{}, err
	}
	if !running {
		return daemonStatusWithState(runtime, info, "stopped"), nil
	}
	return daemonStatusWithState(runtime, info, "starting"), nil
}

func daemonInfo(homePaths aghconfig.HomePaths, deps commandDeps) (aghdaemon.Info, bool, error) {
	info, err := deps.readDaemonInfo(homePaths.DaemonInfo)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return aghdaemon.Info{}, false, nil
	default:
		return aghdaemon.Info{}, false, err
	}

	if !deps.processAlive(info.PID) {
		return info, false, nil
	}
	return info, true, nil
}

func daemonStatusWithState(runtime runtimeContext, info aghdaemon.Info, status string) DaemonStatus {
	networkStatus := daemonNetworkStatusFromInfo(runtime.Config, info.Network)
	if strings.EqualFold(strings.TrimSpace(status), "stopped") {
		networkStatus = nil
	}

	return DaemonStatus{
		Status:         status,
		PID:            info.PID,
		StartedAt:      info.StartedAt,
		Socket:         runtime.Config.Daemon.Socket,
		HTTPHost:       runtime.Config.HTTP.Host,
		HTTPPort:       runtime.Config.HTTP.Port,
		ActiveSessions: 0,
		TotalSessions:  0,
		Version:        version.Current().Version,
		Network:        networkStatus,
	}
}

func daemonStatusBundle(status DaemonStatus, now func() time.Time) outputBundle {
	rows := []keyValue{
		{Label: "Status", Value: stringOrDash(status.Status)},
		{Label: "PID", Value: intOrDash(status.PID)},
		{Label: "Started", Value: stringOrDash(formatTime(status.StartedAt))},
		{Label: "Uptime", Value: stringOrDash(formatAge(now, status.StartedAt))},
		{Label: "Socket", Value: stringOrDash(status.Socket)},
		{Label: "HTTP", Value: stringOrDash(strings.TrimSpace(status.HTTPHost) + ":" + intOrDash(status.HTTPPort))},
		{Label: "Active Sessions", Value: strconv.Itoa(status.ActiveSessions)},
		{Label: "Total Sessions", Value: strconv.Itoa(status.TotalSessions)},
		{Label: "Version", Value: stringOrDash(status.Version)},
	}
	labels := []string{
		"status", "pid", "started_at", "uptime", "socket", "http_host", "http_port", "active_sessions", "total_sessions", "version",
	}
	values := []string{
		status.Status,
		strconv.Itoa(status.PID),
		formatTime(status.StartedAt),
		formatAge(now, status.StartedAt),
		status.Socket,
		status.HTTPHost,
		strconv.Itoa(status.HTTPPort),
		strconv.Itoa(status.ActiveSessions),
		strconv.Itoa(status.TotalSessions),
		status.Version,
	}
	if status.Network != nil {
		rows = append(rows,
			keyValue{Label: "Network", Value: stringOrDash(status.Network.Status)},
			keyValue{Label: "Network Listener", Value: stringOrDash(networkListener(status.Network))},
			keyValue{Label: "Network Local Peers", Value: strconv.Itoa(status.Network.LocalPeers)},
			keyValue{Label: "Network Remote Peers", Value: strconv.Itoa(status.Network.RemotePeers)},
			keyValue{Label: "Network Spaces", Value: strconv.Itoa(status.Network.Spaces)},
			keyValue{Label: "Network Queued Messages", Value: strconv.Itoa(status.Network.QueuedMessages)},
			keyValue{Label: "Network Delivery Workers", Value: strconv.Itoa(status.Network.DeliveryWorkers)},
			keyValue{Label: "Network Messages Sent", Value: strconv.FormatInt(status.Network.MessagesSent, 10)},
			keyValue{Label: "Network Messages Received", Value: strconv.FormatInt(status.Network.MessagesReceived, 10)},
			keyValue{Label: "Network Messages Rejected", Value: strconv.FormatInt(status.Network.MessagesRejected, 10)},
			keyValue{Label: "Network Messages Delivered", Value: strconv.FormatInt(status.Network.MessagesDelivered, 10)},
			keyValue{Label: "Network Workflow Tagged", Value: strconv.FormatInt(status.Network.WorkflowTaggedEvents, 10)},
			keyValue{Label: "Network Handoff Tagged", Value: strconv.FormatInt(status.Network.HandoffTaggedEvents, 10)},
			keyValue{Label: "Network Last Disconnect", Value: stringOrDash(status.Network.LastDisconnect)},
		)
		labels = append(labels,
			"network_status", "network_listener", "network_local_peers", "network_remote_peers", "network_spaces",
			"network_queued_messages", "network_delivery_workers", "network_messages_sent", "network_messages_received",
			"network_messages_rejected", "network_messages_delivered", "network_workflow_tagged_events",
			"network_handoff_tagged_events", "network_last_disconnect",
		)
		values = append(values,
			status.Network.Status,
			networkListener(status.Network),
			strconv.Itoa(status.Network.LocalPeers),
			strconv.Itoa(status.Network.RemotePeers),
			strconv.Itoa(status.Network.Spaces),
			strconv.Itoa(status.Network.QueuedMessages),
			strconv.Itoa(status.Network.DeliveryWorkers),
			strconv.FormatInt(status.Network.MessagesSent, 10),
			strconv.FormatInt(status.Network.MessagesReceived, 10),
			strconv.FormatInt(status.Network.MessagesRejected, 10),
			strconv.FormatInt(status.Network.MessagesDelivered, 10),
			strconv.FormatInt(status.Network.WorkflowTaggedEvents, 10),
			strconv.FormatInt(status.Network.HandoffTaggedEvents, 10),
			status.Network.LastDisconnect,
		)
	}

	return outputBundle{
		jsonValue: status,
		human: func() (string, error) {
			return renderHumanSection("Daemon", rows), nil
		},
		toon: func() (string, error) {
			return renderToonObject("daemon", labels, values), nil
		},
	}
}

func daemonNetworkStatusFromInfo(cfg aghconfig.Config, info *aghdaemon.NetworkInfo) *contract.NetworkStatusPayload {
	if info != nil {
		return &contract.NetworkStatusPayload{
			Enabled:      info.Enabled,
			Status:       strings.TrimSpace(info.Status),
			ListenerHost: strings.TrimSpace(info.ListenerHost),
			ListenerPort: info.ListenerPort,
		}
	}
	if !cfg.Network.Enabled {
		return &contract.NetworkStatusPayload{
			Enabled: false,
			Status:  "disabled",
		}
	}
	return nil
}

func networkListener(info *contract.NetworkStatusPayload) string {
	if info == nil {
		return ""
	}
	host := strings.TrimSpace(info.ListenerHost)
	switch {
	case host == "" && info.ListenerPort <= 0:
		return ""
	case host == "":
		return intOrDash(info.ListenerPort)
	case info.ListenerPort <= 0:
		return host
	default:
		return host + ":" + strconv.Itoa(info.ListenerPort)
	}
}

func spawnDetachedDaemonProcess(homePaths aghconfig.HomePaths, executable func() (string, error)) (daemonProcess, error) {
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(homePaths.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("cli: open daemon log %q: %w", homePaths.LogFile, err)
	}

	binary, err := executable()
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("cli: resolve executable: %w", err)
	}

	logInfo, err := logFile.Stat()
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("cli: stat daemon log %q: %w", homePaths.LogFile, err)
	}
	child := exec.Command(binary, "daemon", "start", "--foreground", "--"+internalChildFlagName)
	child.Env = os.Environ()
	child.Stdin = nil
	child.Stdout = logFile
	child.Stderr = logFile
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := child.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("cli: spawn detached daemon: %w", err)
	}
	if err := logFile.Close(); err != nil {
		return nil, fmt.Errorf("cli: close daemon log handle: %w", err)
	}

	return &execDaemonProcess{cmd: child, logPath: homePaths.LogFile, logOffset: logInfo.Size()}, nil
}

func attachCommandLog(err error, logPath string, logOffset int64) error {
	if err == nil {
		return err
	}
	text, readErr := readCommandLog(logPath, logOffset)
	if readErr != nil {
		return err
	}
	text = recentCommandError(text)
	if text == "" {
		return err
	}
	return fmt.Errorf("%w: stderr=%s", err, text)
}

func readCommandLog(path string, offset int64) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cli: read daemon log %q: %w", path, err)
	}
	if offset < 0 || offset > int64(len(data)) {
		offset = 0
	}
	return strings.TrimSpace(string(data[offset:])), nil
}

func recentCommandError(logText string) string {
	text := strings.TrimSpace(logText)
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "error:") {
			return line
		}
	}

	return text
}
