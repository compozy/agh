package procutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DetachedLaunchRequest describes one detached process launch with log capture.
type DetachedLaunchRequest struct {
	Binary      string
	Args        []string
	Environment []string
	LogPath     string
}

// DetachedProcess wraps a detached child process whose stderr/stdout were appended to a log file.
type DetachedProcess struct {
	process   *os.Process
	logPath   string
	logOffset int64
}

// PID reports the launched process id.
func (p *DetachedProcess) PID() int {
	if p == nil || p.process == nil {
		return 0
	}
	return p.process.Pid
}

// Wait blocks until the detached process exits and attaches recent log output to failures.
func (p *DetachedProcess) Wait() error {
	if p == nil || p.process == nil {
		return nil
	}

	state, err := p.process.Wait()
	if err == nil {
		if state == nil || state.Success() {
			return nil
		}
		err = &exec.ExitError{ProcessState: state}
	}
	return attachCommandLog(err, p.logPath, p.logOffset)
}

// SpawnDetachedLoggedProcess launches one detached child process whose stdout/stderr are appended to req.LogPath.
func SpawnDetachedLoggedProcess(
	ctx context.Context,
	req DetachedLaunchRequest,
) (*DetachedProcess, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := req.validate(); err != nil {
		return nil, err
	}

	return spawnDetachedLoggedProcess(ctx, req)
}

func (r DetachedLaunchRequest) validate() error {
	if strings.TrimSpace(r.Binary) == "" {
		return errors.New("procutil: launch binary is required")
	}
	if strings.TrimSpace(r.LogPath) == "" {
		return errors.New("procutil: launch log path is required")
	}
	return nil
}

func resolveLaunchBinary(binary string) (string, error) {
	resolved, err := exec.LookPath(strings.TrimSpace(binary))
	if err != nil {
		return "", fmt.Errorf("procutil: resolve launch binary %q: %w", binary, err)
	}
	return resolved, nil
}

func launchEnvironment(environment []string) []string {
	if len(environment) > 0 {
		return append([]string(nil), environment...)
	}
	return os.Environ()
}

func launchArgv(binary string, args []string) []string {
	argv := make([]string, 1, len(args)+1)
	argv[0] = binary
	return append(argv, args...)
}

func attachCommandLog(err error, logPath string, logOffset int64) error {
	if err == nil {
		return nil
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
		return "", fmt.Errorf("procutil: read log %q: %w", path, err)
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
