//go:build !windows

package procutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func spawnDetachedLoggedProcess(
	ctx context.Context,
	req DetachedLaunchRequest,
) (*DetachedProcess, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	binary, err := resolveLaunchBinary(req.Binary)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(req.LogPath), 0o755); err != nil {
		return nil, fmt.Errorf("procutil: create log directory for %q: %w", req.LogPath, err)
	}

	logFile, err := os.OpenFile(req.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("procutil: open log %q: %w", req.LogPath, err)
	}

	logInfo, err := logFile.Stat()
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("procutil: stat log %q: %w", req.LogPath, err)
	}

	stdinFile, err := os.Open(os.DevNull)
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("procutil: open %q: %w", os.DevNull, err)
	}

	if err := ctx.Err(); err != nil {
		_ = stdinFile.Close()
		_ = logFile.Close()
		return nil, err
	}

	process, err := os.StartProcess(binary, launchArgv(binary, req.Args), &os.ProcAttr{
		Env:   launchSandbox(req.Sandbox),
		Files: []*os.File{stdinFile, logFile, logFile},
		Sys:   &syscall.SysProcAttr{Setpgid: true},
	})
	if err != nil {
		_ = stdinFile.Close()
		_ = logFile.Close()
		return nil, fmt.Errorf("procutil: spawn detached process: %w", err)
	}
	if err := stdinFile.Close(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("procutil: close %q handle: %w", os.DevNull, err)
	}
	if err := logFile.Close(); err != nil {
		return nil, fmt.Errorf("procutil: close log handle %q: %w", req.LogPath, err)
	}

	return newDetachedProcess(process, req.LogPath, logInfo.Size()), nil
}
