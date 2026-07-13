package session

import (
	"path/filepath"
	"strings"

	"github.com/compozy/agh/internal/acp"
	envpkg "github.com/compozy/agh/internal/sandbox"
)

func sandboxStartOpts(
	opts acp.StartOpts,
	prepared envpkg.Prepared,
	state envpkg.SessionState,
) acp.StartOpts {
	next := opts
	if command := strings.TrimSpace(prepared.Launch.Command); command != "" {
		next.Command = command
	}
	if prepared.Launch.Env != nil {
		next.Env = append([]string(nil), prepared.Launch.Env...)
	}
	next.Cwd = sandboxRuntimeCWD(opts.Cwd, prepared, state)
	next.AdditionalDirs = append([]string(nil), prepared.RuntimeAdditionalDirs...)
	if next.AdditionalDirs == nil {
		next.AdditionalDirs = append([]string(nil), state.RuntimeAdditionalDirs...)
	}
	next.Launcher = prepared.Launcher
	next.ToolHost = prepared.ToolHost
	return next
}

func sandboxRuntimeCWD(
	requested string,
	prepared envpkg.Prepared,
	state envpkg.SessionState,
) string {
	runtimeRoot := strings.TrimSpace(prepared.RuntimeRootDir)
	if runtimeRoot == "" {
		runtimeRoot = strings.TrimSpace(state.RuntimeRootDir)
	}
	requested = strings.TrimSpace(requested)
	if state.Backend == envpkg.BackendLocal && requested != "" {
		if runtimeRoot == "" {
			return requested
		}
		relative, err := filepath.Rel(runtimeRoot, requested)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return requested
		}
	}
	return runtimeRoot
}
