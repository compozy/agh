package acpmock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const driverBinaryEnvVar = "AGH_TEST_ACPMOCK_DRIVER_BIN"

var (
	driverBinaryMu   sync.Mutex
	driverBinaryPath string
)

// DefaultDriverPath resolves or builds the test-only ACP mock driver binary.
func DefaultDriverPath() (string, error) {
	driverBinaryMu.Lock()
	cached := driverBinaryPath
	driverBinaryMu.Unlock()
	if strings.TrimSpace(cached) != "" {
		return cached, nil
	}

	repoRoot, err := repoRootFromCaller()
	if err != nil {
		return "", err
	}

	outputDir, err := os.MkdirTemp("", "agh-acpmock-driver-")
	if err != nil {
		return "", fmt.Errorf("acpmock: create driver build directory: %w", err)
	}
	outputPath := filepath.Join(outputDir, driverBinaryName())

	cmd := exec.CommandContext(
		context.Background(),
		"go",
		"build",
		"-o",
		outputPath,
		"./internal/testutil/acpmock/cmd/acpmock-driver",
	)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"acpmock: build driver binary: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	driverBinaryMu.Lock()
	driverBinaryPath = outputPath
	driverBinaryMu.Unlock()
	return outputPath, nil
}

func resolveDriverPath(override string) (string, error) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed, nil
	}
	if trimmed := strings.TrimSpace(os.Getenv(driverBinaryEnvVar)); trimmed != "" {
		return trimmed, nil
	}
	return DefaultDriverPath()
}

func driverBinaryName() string {
	if runtime.GOOS == "windows" {
		return "acpmock-driver.exe"
	}
	return "acpmock-driver"
}

func repoRootFromCaller() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("acpmock: runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..")), nil
}
